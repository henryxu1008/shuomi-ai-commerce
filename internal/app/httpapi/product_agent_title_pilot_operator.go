package httpapi

import (
	"context"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"task-processor/internal/agent"
	"task-processor/internal/aicapability"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	resourceadapter "task-processor/internal/integration/orgresource"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/review"
	"task-processor/internal/product/sourcing"
)

const TitlePilotRequestLimit = 65 * time.Minute

var errTitlePilotExpired = errors.New("title pilot authorization is absent or expired")
var errTitlePilotFunds = errors.New("title pilot funds or member limit are insufficient")

// TitlePilotOperator is injected only by the verified private runtime manifest.
// The optional window authorizes this exact immutable approval, never HTTP input.
type TitlePilotOperator struct {
	Host, OrganizationID, ActorID, MemberID                        string
	SamplesDigest, ProfileDigest, CredentialRevision, PriceVersion string
	FXVersion                                                      string
	ExecutionApprovalID                                            string
	BatchID                                                        uuid.UUID
	NotBefore, NotAfter                                            time.Time
	Register                                                       func(http.Handler)
	VerifyIsolation                                                func(context.Context) error
	now                                                            func() time.Time
}

func (op TitlePilotOperator) approval() titlePilotApproval {
	return titlePilotApproval{batchID: op.BatchID, identity: agent.ExecutionIdentity{
		OrganizationID: op.OrganizationID, ActorID: op.ActorID, MemberID: op.MemberID,
	}, profileDigest: op.ProfileDigest}
}

func (op TitlePilotOperator) valid() bool {
	host, port, err := net.SplitHostPort(op.Host)
	_, digestErr := hex.DecodeString(op.ProfileDigest)
	return err == nil && host == "127.0.0.1" && port != "" && op.approval().valid() && digestErr == nil &&
		op.SamplesDigest == titlePilotManifestHash && op.Register != nil && op.VerifyIsolation != nil &&
		op.CredentialRevision != "" && op.PriceVersion != "" && op.FXVersion != "" &&
		(op.ExecutionApprovalID == "" && op.NotBefore.IsZero() && op.NotAfter.IsZero() ||
			agent.ValidID(op.ExecutionApprovalID) && !op.NotBefore.IsZero() && !op.NotAfter.IsZero() && op.NotAfter.After(op.NotBefore))
}

func (op TitlePilotOperator) matchesProfile(profile aicapability.ModelProfile) bool {
	return profile.CredentialVersion == op.CredentialRevision && profile.OfficialTariff.PriceVersion == op.PriceVersion &&
		profile.OfficialTariff.FXVersion == op.FXVersion
}

func (op TitlePilotOperator) authorizeTime(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now
	if op.now != nil {
		now = op.now
	}
	at := now()
	capability, ok := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability)
	if !ok || op.ExecutionApprovalID == "" || op.NotBefore.IsZero() || at.Before(op.NotBefore) || !at.Before(op.NotAfter) || !at.Before(capability.tokenExpiresAt) {
		return errTitlePilotExpired
	}
	return nil
}

func (a *productAgentApplication) authorizeTitlePilotEffect(ctx context.Context, plan *titlePilotPlan) error {
	op := a.config.TitlePilotOperator
	if op == nil {
		return nil
	}
	if plan == nil || plan.app != a || plan.approval != a.config.titlePilotApproval || plan.approval != op.approval() ||
		plan.identity != plan.approval.identity || !op.matchesProfile(plan.profile) {
		return review.ErrForbidden
	}
	return op.authorizeTime(ctx)
}

func (a *productAgentApplication) titlePilotBudget(profile aicapability.ModelProfile) (int64, error) {
	maxCost, err := profile.MaximumCost()
	if err != nil || maxCost <= 0 || !titlePilotQuoteCoversTariff(profile, profile.MaximumPromptTokens, maxCost) {
		return 0, errTitlePilotStopped
	}
	reserve, err := profile.OfficialTariff.ReserveTextMicropoints(profile.MaximumPromptTokens, profile.MaximumCompletionTokens)
	if err != nil || reserve <= 0 {
		return 0, errTitlePilotStopped
	}
	var perPairCosts, perPairCalls int64
	for _, fixed := range []bool{true, false} {
		limits := a.titlePilotLimits(fixed)
		if !limits.Valid() || limits.Currency != profile.Currency || limits.CostMicros < maxCost ||
			limits.Tokens < profile.MaximumPromptTokens+profile.MaximumCompletionTokens ||
			limits.CostMicros > titlePilotBatchCost/titlePilotSampleCount {
			return 0, errTitlePilotStopped
		}
		perPairCosts += limits.CostMicros
		perPairCalls += int64(limits.ModelCalls)
	}
	if perPairCosts > titlePilotBatchCost/titlePilotSampleCount {
		return 0, errTitlePilotStopped
	}
	return reserve * perPairCalls * titlePilotSampleCount, nil
}

type titlePilotPreflight struct {
	BatchID             string `json:"batchId"`
	ProfileDigest       string `json:"profileDigest"`
	PriceVersion        string `json:"priceVersion"`
	FXVersion           string `json:"fxVersion"`
	ExistingSources     int    `json:"existingSources"`
	FirstPublications   int    `json:"firstPublications"`
	ExpectedRunSlots    int    `json:"expectedRunSlots"`
	RequiredMicropoints int64  `json:"requiredMicropoints"`
}

func (a *productAgentApplication) preflightTitlePilot(ctx context.Context) (titlePilotPreflight, error) {
	approval := a.config.titlePilotApproval
	plan, samples, err := a.inspectTitlePilot(ctx, approval.batchID)
	if err != nil {
		return titlePilotPreflight{}, err
	}
	required, err := a.titlePilotBudget(plan.profile)
	if err != nil {
		return titlePilotPreflight{}, err
	}
	out := titlePilotPreflight{BatchID: approval.batchID.String(), ProfileDigest: approval.profileDigest,
		PriceVersion: plan.profile.OfficialTariff.PriceVersion, FXVersion: plan.profile.OfficialTariff.FXVersion,
		ExpectedRunSlots: 2 * len(samples), RequiredMicropoints: required}
	for _, sample := range samples {
		command := titlePilotPublication(plan.identity, sample)
		receipt, err := a.titlePilotPublisher.Verify(ctx, command)
		if err == sourcing.ErrSourcePublicationNotFound {
			out.FirstPublications++
			continue
		}
		if err != nil {
			return titlePilotPreflight{}, err
		}
		slot := a.titlePilotSlotFor(plan, command, receipt, "fixed")
		if _, err := a.titlePilotPublicationForBinding(slot.context(ctx), slot.binding); err != nil {
			return titlePilotPreflight{}, err
		}
		out.ExistingSources++
	}
	if err := a.checkTitlePilotFunds(ctx, plan.identity, required); err != nil {
		return titlePilotPreflight{}, err
	}
	return out, nil
}

func (a *productAgentApplication) checkTitlePilotFunds(ctx context.Context, identity agent.ExecutionIdentity, required int64) error {
	repo, err := resourceadapter.NewGormRepository(a.config.PointAccountingDB, resourceadapter.TransactionConfig{})
	if err != nil {
		return agent.ErrUnavailable
	}
	balances, err := repo.ReadBalances(ctx, identity.OrganizationID)
	if err != nil || balances.OrganizationID != identity.OrganizationID {
		return agent.ErrUnavailable
	}
	funded := false
	for _, balance := range balances.Resources {
		if balance.ResourceType != orgresource.ResourceAIPoint {
			continue
		}
		if balance.State != "recorded" || balance.UnitVersion != orgresource.AIPointQuantityUnitV1 || balance.Available == nil {
			return agent.ErrUnavailable
		}
		amount, err := orgresource.ParseAIPointQuantity(*balance.Available)
		if err != nil {
			return agent.ErrUnavailable
		}
		funded = amount.Micropoints >= required
	}
	if !funded {
		return errTitlePilotFunds
	}
	limits, err := resourceadapter.NewGormMemberLimitRepository(a.config.PointAccountingDB, resourceadapter.TransactionConfig{})
	if err != nil {
		return agent.ErrUnavailable
	}
	limit, err := limits.ReadMonthlyLimit(ctx, identity.OrganizationID, identity.MemberID)
	if err != nil || limit.UnitVersion != orgresource.AIPointQuantityUnitV1 || limit.OrganizationID != identity.OrganizationID ||
		limit.MemberID != identity.MemberID || limit.MonthlyLimit < 0 || limit.Consumed < 0 || limit.Reserved < 0 {
		return agent.ErrUnavailable
	}
	if limit.Consumed > limit.MonthlyLimit || limit.Reserved > limit.MonthlyLimit-limit.Consumed ||
		required > limit.MonthlyLimit-limit.Consumed-limit.Reserved {
		return errTitlePilotFunds
	}
	return nil
}

func titlePilotLoopback(host string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		peer, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || peer != "127.0.0.1" || r.Host != host || r.URL.IsAbs() || r.URL.RawQuery != "" || r.URL.ForceQuery ||
			r.ContentLength != 0 || len(r.TransferEncoding) != 0 || r.Header.Get("Origin") != "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for name := range r.Header {
			name = strings.ToLower(name)
			if name == "forwarded" || strings.HasPrefix(name, "x-forwarded-") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *productAgentApplication) titlePilotOperatorHandler(ctx context.Context, deps routeAuthDependencies) (http.Handler, error) {
	op := a.config.TitlePilotOperator
	if op == nil || deps.workbenchVerifier == nil || deps.organizationResolver == nil || a.titlePilotFixed == nil || a.selectTitleProfile == nil {
		return nil, agent.ErrUnavailable
	}
	profile, err := a.readTitlePilotProfile(ctx, op.approval())
	if err != nil {
		return nil, err
	}
	if _, err := a.titlePilotBudget(profile); err != nil {
		return nil, err
	}
	var routes []httproute.Descriptor
	for _, start := range []bool{false, true} {
		method, path, timeout := http.MethodGet, "/title-pilot/preflight", 2*time.Minute
		if start {
			method, path, timeout = http.MethodPost, "/title-pilot/start", TitlePilotRequestLimit
		}
		routes = append(routes, httproute.Descriptor{Method: method, Path: path, Module: "title-pilot-operator",
			AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite,
			Permission: authz.PermissionLocalAgentWrite, RequestTimeout: timeout, Handler: func(c *gin.Context) {
				ctx, err := (productReviewCapabilityBinder{now: op.now}).Bind(c.Request.Context(), c.GetHeader("Authorization"))
				if err != nil {
					writeTitlePilotOperatorError(c, err)
					return
				}
				capability := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability)
				deadline := capability.tokenExpiresAt
				if start {
					if err := op.authorizeTime(ctx); err != nil {
						writeTitlePilotOperatorError(c, err)
						return
					}
					if op.NotAfter.Before(deadline) {
						deadline = op.NotAfter
					}
				}
				ctx, cancel := context.WithDeadline(ctx, deadline)
				defer cancel()
				if start {
					records, err := a.startApprovedTitlePilot(ctx)
					if err != nil {
						writeTitlePilotOperatorError(c, err)
						return
					}
					c.JSON(http.StatusOK, gin.H{"batchId": op.BatchID.String(), "completedRunSlots": len(records)})
					return
				}
				result, err := a.preflightTitlePilot(ctx)
				if err != nil {
					writeTitlePilotOperatorError(c, err)
					return
				}
				c.JSON(http.StatusOK, result)
			}})
	}
	router := buildHTTPServerFromRoutesAtWithAuthDependencies("", 0, routes, deps).Handler
	return titlePilotLoopback(op.Host, router), nil
}

func writeTitlePilotOperatorError(c *gin.Context, err error) {
	status, code := http.StatusServiceUnavailable, "UNAVAILABLE"
	switch {
	case errors.Is(err, errTitlePilotUnknown):
		status, code = http.StatusConflict, "UNKNOWN"
	case errors.Is(err, errTitlePilotExpired), errors.Is(err, review.ErrForbidden), errors.Is(err, sourcing.ErrPublicationForbidden):
		status, code = http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, agent.ErrConflict), errors.Is(err, sourcing.ErrSourcePublicationConflict):
		status, code = http.StatusConflict, "CONFLICT"
	case errors.Is(err, errTitlePilotFunds):
		status, code = http.StatusConflict, "INSUFFICIENT_POINTS"
	case errors.Is(err, errTitlePilotStopped):
		status, code = http.StatusConflict, "STOPPED"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		status, code = http.StatusRequestTimeout, "INTERRUPTED"
	}
	c.JSON(status, gin.H{"status": code})
}
