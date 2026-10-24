package einomodel

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"task-processor/internal/aicapability"
	"task-processor/internal/integration/googleinteractions"
)

// QualifiedRoute is returned only by the current organization credential and
// deployment policy owner. The API key must never be stored in a Workbench or
// invocation receipt.
type QualifiedRoute struct {
	Profile  aicapability.ModelProfile
	Endpoint string
	APIKey   string
}

type InvocationLedger interface {
	aicapability.InvocationDispatchClaimer
	aicapability.InvocationRecorder
	aicapability.InvocationUsageReservation
}

// Executor uses the same native invocation and ResourceAIPoint owners for
// planning and Product Agent title text. Resolve and Authorize must read the
// current route and permission on every call, including the final handoff.
type Executor struct {
	Ledger        InvocationLedger
	Admission     *BoundedAdmission
	Resolve       func(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error)
	Authorize     func(context.Context, aicapability.TextInputIdentity) error
	BaseTransport http.RoundTripper
	Logger        *logrus.Entry `json:"-"`
}

func endpointDigest(endpoint string) string {
	sum := sha256.Sum256([]byte(endpoint))
	return hex.EncodeToString(sum[:])
}

func sameRoute(a, b QualifiedRoute) bool {
	ad, ae := a.Profile.Digest()
	bd, be := b.Profile.Digest()
	if ae != nil || be != nil || ad != bd || a.Endpoint != b.Endpoint ||
		endpointDigest(a.Endpoint) != a.Profile.EndpointIdentityDigest ||
		a.APIKey == "" || b.APIKey == "" {
		return false
	}
	ah, bh := sha256.Sum256([]byte(a.APIKey)), sha256.Sum256([]byte(b.APIKey))
	return subtle.ConstantTimeCompare(ah[:], bh[:]) == 1
}

// The cancellation callback may still be pending after the absolute deadline.
// Dispatch must check the frozen bound as well as explicit cancellation.
func dispatchContextActive(ctx context.Context) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	deadline, bounded := ctx.Deadline()
	return !bounded || time.Now().Before(deadline)
}

func (e *Executor) admitted(ctx context.Context, input aicapability.TextInputIdentity) (QualifiedRoute, error) {
	if e == nil || e.Ledger == nil || e.Admission == nil || e.Resolve == nil || e.Authorize == nil || !dispatchContextActive(ctx) {
		return QualifiedRoute{}, ErrNotDispatched
	}
	if err := e.Authorize(ctx, input); err != nil {
		return QualifiedRoute{}, errors.Join(ErrNotDispatched, err)
	}
	if !ValidRouteProfile(input.Profile, input.Operation) {
		return QualifiedRoute{}, ErrNotDispatched
	}
	route, err := e.Resolve(ctx, input)
	if err != nil || !sameRoute(route, QualifiedRoute{Profile: input.Profile, Endpoint: route.Endpoint, APIKey: route.APIKey}) {
		return QualifiedRoute{}, ErrNotDispatched
	}
	return route, nil
}

func capabilityFor(operation aicapability.Operation) aicapability.Capability {
	if operation == aicapability.OperationAIWorkbenchChatPlan {
		return aicapability.CapabilityAIWorkbenchChatPlanning
	}
	return aicapability.CapabilityProductEnrichText
}

func textRecord(input aicapability.TextInputIdentity, quote aicapability.TextQuote, now time.Time) aicapability.InvocationRecord {
	p := input.Profile
	prompt := sha256.Sum256([]byte(input.System + "\x00" + input.Prompt))
	return aicapability.InvocationRecord{
		InvocationID: input.InvocationID, AgentRunID: input.AgentRunID, BusinessTaskID: input.BusinessTaskID,
		TenantID: input.OrganizationID, UserID: input.ActorID, MemberID: input.MemberID,
		Capability: capabilityFor(input.Operation), Operation: input.Operation,
		ProviderID: p.ProviderID, ModelID: p.ModelID, CredentialReference: p.ClientName,
		ConfigurationVersion: p.CredentialVersion, PolicyVersion: p.CostPricingVersion,
		PromptKey: p.OutputSchemaVersion, PromptVersion: p.PromptVersion, PromptHash: hex.EncodeToString(prompt[:]),
		InputHash: quote.InputHash, PointTariff: p.PointTariff,
		MaximumPromptTokens: quote.MaximumTokens - p.MaximumCompletionTokens, MaximumCompletionTokens: p.MaximumCompletionTokens,
		StartedAt: now, Attempt: 1, Outcome: aicapability.InvocationDispatched, Currency: p.Currency,
	}
}

func (e *Executor) terminal(ctx context.Context, record aicapability.InvocationRecord) error {
	// A canceled HTTP request cannot erase an already observed provider outcome.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	err := e.Ledger.RecordInvocation(writeCtx, record)
	if diagnostic, ok := ctx.Value(textDiagnosticKey{}).(*textDiagnostics); ok {
		category := "recorded"
		if err != nil {
			category = "record_failed"
		}
		diagnostic.emit(writeCtx, "terminal", category)
	}
	return err
}

// Generate requires a quote prepared with the exact frozen input. A receipt
// replay cannot acquire a second provider send: the ledger claim owns that
// boundary. The output validator runs only after provider usage is observed.
func (e *Executor) Generate(ctx context.Context, input aicapability.TextInputIdentity, expected aicapability.TextQuote, validate func(string) error) (TextOutput, error) {
	return e.GenerateWithGate(ctx, input, expected, validate, nil)
}

// GenerateWithGate lets a consumer acquire an existing-owner permit exactly
// at the final transport handoff. cleanup runs after every model outcome.
func (e *Executor) GenerateWithGate(ctx context.Context, input aicapability.TextInputIdentity, expected aicapability.TextQuote,
	validate func(string) error, beforeSend func(context.Context) (func(), error)) (TextOutput, error) {
	quote, err := aicapability.QuoteText(input)
	if err != nil || quote != expected || input.InvocationID == "" || ctx == nil {
		return TextOutput{}, ErrNotDispatched
	}
	// Admission and the final transport handoff share the frozen attempt bound.
	// A shorter caller deadline (such as a Workbench command) wins.
	boundedCtx, cancel := context.WithTimeout(ctx, input.Profile.DeadlineBound)
	defer cancel()
	ctx = boundedCtx
	var logger *logrus.Entry
	if e != nil {
		logger = e.Logger
	}
	diagnostic := newTextDiagnostics(ctx, input.InvocationID, logger)
	if input.AgentRunID != "" {
		diagnostic.set("agent_run_id_hash", diagnosticHash(input.AgentRunID))
	}
	diagnostic.set("profile_digest", quote.ProfileDigest)
	ctx = context.WithValue(ctx, textDiagnosticKey{}, diagnostic)
	defer func() {
		if recovered := recover(); recovered != nil {
			diagnostic.emit(ctx, "executor", "panic")
			panic(recovered) // The existing Eino recovery owner remains authoritative.
		}
	}()
	route, err := e.admitted(ctx, input)
	if err != nil {
		diagnostic.emit(ctx, "admission", "rejected_before_claim")
		return TextOutput{}, err
	}
	p := input.Profile
	var cleanup func()
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()
	guardCfg := GuardConfig{Logger: logger, Adapter: AdapterKind(p.AdapterKind), Endpoint: route.Endpoint, ModelID: p.ModelID,
		ProviderID: p.ProviderID, AdapterPolicyVersion: p.AdapterPolicyVersion,
		MaximumOutputTokens: int(p.MaximumCompletionTokens), MaximumRequestBytes: int64(p.MaximumInputBytes),
		MaximumResponseBytes: int64(p.MaximumOutputBytes), Timeout: p.DeadlineBound}
	guard, err := NewGuardedClient(guardCfg, e.BaseTransport, func(gateCtx context.Context) error {
		// The SDK may construct its own request context. The frozen caller bound
		// remains authoritative at the actual guarded network handoff.
		if !dispatchContextActive(ctx) || !dispatchContextActive(gateCtx) {
			return ErrNotDispatched
		}
		current, checkErr := e.admitted(ctx, input)
		if checkErr != nil || !sameRoute(route, current) {
			return ErrNotDispatched
		}
		if beforeSend != nil {
			var permitErr error
			cleanup, permitErr = beforeSend(ctx)
			if permitErr != nil {
				return permitErr
			}
		}
		if !dispatchContextActive(ctx) || !dispatchContextActive(gateCtx) {
			return ErrNotDispatched
		}
		return nil
	})
	if err != nil {
		diagnostic.emit(ctx, "component", "rejected_before_claim")
		return TextOutput{}, ErrNotDispatched
	}
	component, err := NewComponent(ctx, ComponentConfig{Adapter: guardCfg.Adapter, Endpoint: route.Endpoint,
		ProviderID: p.ProviderID, AdapterPolicyVersion: p.AdapterPolicyVersion,
		APIKey: route.APIKey, ModelID: p.ModelID, MaximumOutputTokens: guardCfg.MaximumOutputTokens}, guard)
	if err != nil {
		diagnostic.emit(ctx, "component", "rejected_before_claim")
		return TextOutput{}, ErrNotDispatched
	}
	now := time.Now().UTC()
	record := textRecord(input, quote, now)
	acquired, err := e.Ledger.ClaimInvocation(ctx, record)
	if err != nil || !acquired {
		category := "existing"
		if err != nil {
			category = "claim_unconfirmed"
		}
		diagnostic.emit(ctx, "claim", category)
		// The claim may have committed even when its response was lost. A
		// duplicate also has an existing durable fact. Neither proves a
		// terminal no-send outcome or permits another provider attempt.
		return TextOutput{}, ErrOutcomeUnknown
	}
	if err := e.Ledger.ReserveAIInvocationUsage(ctx, record.TenantID, record.MemberID, record.InvocationID, quote.MaximumTokens, now); err != nil {
		diagnostic.emit(ctx, "reservation", "reserve_failed")
		record.Outcome = aicapability.InvocationFailed
		record.FinishedAt = time.Now().UTC()
		record.ErrorCode = "reservation_failed_before_dispatch"
		record.UsageKnown, record.EstimatedCostKnown = true, true
		if e.terminal(ctx, record) != nil {
			return TextOutput{}, ErrOutcomeUnknown
		}
		return TextOutput{}, errors.Join(ErrNotDispatched, ErrNoDispatchReleased)
	}
	var output TextOutput
	var generationErr error
	release, admissionErr := e.Admission.Acquire(ctx, input)
	if admissionErr == nil {
		func() {
			defer release()
			output, generationErr = GenerateText(ctx, component, input.System, input.Prompt, guard)
		}()
	}
	if admissionErr != nil || guard.NetworkSends() == 0 {
		diagnostic.emit(ctx, "handoff", "not_dispatched")
		record.Outcome = aicapability.InvocationFailed
		record.FinishedAt = time.Now().UTC()
		record.ErrorCode = "rejected_before_dispatch"
		record.UsageKnown, record.EstimatedCostKnown = true, true
		if e.terminal(ctx, record) != nil {
			return TextOutput{}, ErrOutcomeUnknown
		}
		return TextOutput{}, errors.Join(ErrNotDispatched, ErrNoDispatchReleased)
	}
	if !output.Usage.Known || output.Usage.PromptTokens > int(record.MaximumPromptTokens) ||
		output.Usage.CompletionTokens > int(p.MaximumCompletionTokens) ||
		errors.Is(generationErr, ErrOutcomeUnknown) || errors.Is(generationErr, ErrUsageUnknown) {
		category := "outcome_or_usage_unknown"
		if output.Usage.Known && (output.Usage.PromptTokens > int(record.MaximumPromptTokens) || output.Usage.CompletionTokens > int(p.MaximumCompletionTokens)) {
			category = "usage_bound_exceeded"
		}
		diagnostic.emit(ctx, "usage", category)
		return TextOutput{}, ErrOutcomeUnknown
	}
	record.FinishedAt = time.Now().UTC()
	record.ProviderRequestID = googleinteractions.SafeRequestID(output.ProviderRequestID)
	record.LatencyMilliseconds = record.FinishedAt.Sub(now).Milliseconds()
	record.PromptTokens, record.CompletionTokens, record.TotalTokens = output.Usage.PromptTokens, output.Usage.CompletionTokens, output.Usage.TotalTokens
	record.UsageKnown, record.EstimatedCostKnown = true, true
	record.EstimatedCostMicros, err = p.CostFor(int64(record.PromptTokens), int64(record.CompletionTokens))
	if err != nil {
		diagnostic.emit(ctx, "cost_mapping", "rejected")
		return TextOutput{}, ErrOutcomeUnknown
	}
	record.Outcome = aicapability.InvocationUsageObservedFailed
	record.ErrorCategory = aicapability.ErrorStructuredOutputInvalid
	validErr := generationErr
	if validErr == nil && validate != nil {
		validErr = validate(output.Content)
	}
	if validErr != nil {
		diagnostic.emit(ctx, "output_validation", "schema_rejected")
	}
	if validErr == nil {
		record.Outcome = aicapability.InvocationSucceeded
		record.ErrorCategory = ""
	}
	sum := sha256.Sum256([]byte(output.Content))
	record.OutputHash = hex.EncodeToString(sum[:])
	if e.terminal(ctx, record) != nil {
		return TextOutput{}, ErrOutcomeUnknown
	}
	if validErr != nil {
		// The terminal invocation and its observed usage are already durable.
		// Consumers need that usage to settle their own run budget even though
		// the content cannot be admitted.
		return output, errors.Join(ErrInvalid, ErrObservedInvalidSettled)
	}
	return output, nil
}
