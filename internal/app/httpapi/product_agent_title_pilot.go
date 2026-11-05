package httpapi

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"maps"
	"strconv"

	"github.com/google/uuid"
	"task-processor/internal/agent"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/product/review"
	"task-processor/internal/product/sourcing"
)

const (
	titlePilotContext      = "title_eval"
	titlePilotSampleCount  = 30
	titlePilotCallCost     = 180_000 // CNY micros
	titlePilotAgentCalls   = 6
	titlePilotManifestHash = "8a9d283f9b1fb12a4e2fe9eb9561bce9ec405298cf1999101970c7e2de1376ce"
	titlePilotCasesHash    = "186d392b7cb083f18a2b6a8c5690ef8dd01ca53fe09c1faebbe184e3ded5332e"
)

//go:embed testdata/issue132_title_pilot_v1/manifest.json
var titlePilotManifest string

//go:embed testdata/issue132_title_pilot_v1/cases.json
var titlePilotCases string

var errTitlePilotUnknown = errors.New("title pilot halted: original outcome or settlement is unknown")
var errTitlePilotStopped = errors.New("title pilot halted: slot cannot complete under the frozen admission")

type titlePilotSample struct {
	ID     string                  `json:"case_id"`
	Source sourcing.SourceEnvelope `json:"source"`
}

type titlePilotPlan struct {
	app      *productAgentApplication
	identity agent.ExecutionIdentity
	profile  aicapability.ModelProfile
	slots    []titlePilotSlot
}

type titlePilotSlotKey struct{}

type titlePilotSlot struct {
	plan    *titlePilotPlan
	command sourcing.PublicationCommand
	binding agent.Binding
	key     string
	limits  agent.Limits
	fixed   bool
}

func titlePilotHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func titlePilotID(parts ...string) string {
	raw, _ := json.Marshal(parts)
	return uuid.NewSHA1(uuid.NameSpaceOID, raw).String()
}

func frozenTitlePilotSamples() ([]titlePilotSample, error) {
	var samples []titlePilotSample
	if titlePilotHash(titlePilotManifest) != titlePilotManifestHash || titlePilotHash(titlePilotCases) != titlePilotCasesHash ||
		json.Unmarshal([]byte(titlePilotCases), &samples) != nil || len(samples) != titlePilotSampleCount {
		return nil, agent.ErrInvalid
	}
	return samples, nil
}

func titlePilotPublication(identity agent.ExecutionIdentity, sample titlePilotSample) sourcing.PublicationCommand {
	source := sample.Source
	source.Identity.SourceType = sourcing.SourceTypeManualImport
	source.Identity.SourcePlatform = "local-synthetic"
	source.Identity.SourceURL = ""
	source.RawReference.URL = ""
	source.RawReference.Metadata = maps.Clone(source.RawReference.Metadata)
	source.RawReference.Metadata["issue132_manifest_sha256"] = titlePilotManifestHash
	source.RawReference.Metadata["issue132_case_id"] = sample.ID
	source.AssetCandidates = nil
	base := uint64(0)
	return sourcing.PublicationCommand{
		Producer:            sourcing.ProducerDescriptor{Kind: sourcing.ControlledSnapshotProducerKind, Version: sourcing.ControlledSnapshotProducerVersion},
		ProductKey:          titlePilotID("issue132-product", identity.OrganizationID, identity.ActorID, titlePilotManifestHash, sample.ID),
		PublicationID:       titlePilotID("issue132-publication", identity.OrganizationID, identity.ActorID, titlePilotManifestHash, sample.ID),
		ExpectedBaseVersion: &base, Envelope: source,
	}
}

func titlePilotProfileQualified(p aicapability.ModelProfile) bool {
	maximum, err := p.MaximumCost()
	return err == nil && maximum <= titlePilotCallCost && p.Currency == "CNY" &&
		p.AdapterPolicyVersion == governed.TitlePilotOpenAIAdapterPolicy && p.OfficialTariff.Channel == "openai-direct" &&
		p.UsageMappingVersion == "openai-text-v1" && p.OfficialTariff.UsageVersion == p.UsageMappingVersion && governed.ValidRouteProfile(p, aicapability.OperationProductAgentDecision) &&
		titlePilotQuoteCoversTariff(p, p.MaximumPromptTokens, maximum)
}

func titlePilotQuoteCoversTariff(p aicapability.ModelProfile, prompt, cost int64) bool {
	points, err := p.OfficialTariff.ReserveTextMicropoints(prompt, p.MaximumCompletionTokens)
	// One CNY micro corresponds to 100 micropoints under the existing tariff.
	return err == nil && cost >= 0 && cost <= titlePilotCallCost &&
		(points/100 < cost || points/100 == cost && points%100 == 0)
}

// prepareTitlePilot is internal composition only. It publishes exact frozen
// evidence through SRC, then freezes both groups against the same receipts.
func (a *productAgentApplication) prepareTitlePilot(ctx context.Context, batchID uuid.UUID) (*titlePilotPlan, error) {
	if a.titlePilotPublisher == nil || a.titlePilotFixed == nil || batchID == uuid.Nil {
		return nil, agent.ErrUnavailable
	}
	i, err := a.freshIdentity(ctx)
	if err != nil {
		return nil, err
	}
	identity := agent.ExecutionIdentity{OrganizationID: i.TenantID, ActorID: i.UserID, MemberID: i.EffectiveMemberID}
	profile, err := a.selectTitleProfile(ctx, identity.OrganizationID)
	if err != nil || !titlePilotProfileQualified(profile) {
		return nil, agent.ErrUnavailable
	}
	samples, err := frozenTitlePilotSamples()
	if err != nil {
		return nil, err
	}
	plan := &titlePilotPlan{app: a, identity: identity, profile: profile}
	scope := agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}
	first := titlePilotPublication(identity, samples[0])
	firstKey := titlePilotID("issue132-run", batchID.String(), first.PublicationID, "fixed")
	if _, found, err := a.store.LookupContext(ctx, scope, titlePilotContext, first.PublicationID, firstKey); err != nil || found {
		if err != nil {
			return nil, errors.Join(errTitlePilotUnknown, err)
		}
		return nil, agent.ErrConflict
	}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, i)
	for _, sample := range samples {
		command := titlePilotPublication(identity, sample)
		receipt, err := a.titlePilotPublisher.Publish(ctx, command)
		if err != nil {
			receipt, err = a.titlePilotPublisher.Verify(ctx, command)
			if err != nil {
				return nil, err
			}
		}
		binding := agent.Binding{ContextKind: titlePilotContext, ContextID: command.PublicationID, ProductKey: command.ProductKey,
			CatalogVersion: strconv.FormatUint(receipt.CatalogVersion, 10), PublicationID: receipt.CatalogPublicationID, TargetPlatform: "shein"}
		for _, group := range []string{"fixed", "agent"} {
			limits := a.config.Limits
			fixed := group == "fixed"
			limits.ModelCalls = min(limits.ModelCalls, titlePilotAgentCalls)
			if fixed {
				limits.ModelCalls = 1
			}
			limits.CostMicros = min(limits.CostMicros, int64(limits.ModelCalls)*titlePilotCallCost)
			slot := titlePilotSlot{plan: plan, command: command, binding: binding, fixed: fixed, limits: limits,
				key: titlePilotID("issue132-run", batchID.String(), command.PublicationID, group)}
			if _, err := a.titlePilotPublicationForBinding(slot.context(ctx), binding); err != nil {
				return nil, err
			}
			plan.slots = append(plan.slots, slot)
		}
	}
	return plan, nil
}

func (slot *titlePilotSlot) context(ctx context.Context) context.Context {
	return context.WithValue(ctx, titlePilotSlotKey{}, slot)
}

func (a *productAgentApplication) titlePilotPublicationForBinding(ctx context.Context, binding agent.Binding) (sourcing.PersistedPublication, error) {
	slot, ok := ctx.Value(titlePilotSlotKey{}).(*titlePilotSlot)
	if !ok || slot == nil || slot.plan == nil || slot.plan.app != a || a.titlePilotPublisher == nil || slot.binding != binding ||
		binding.ContextKind != titlePilotContext || !binding.Valid() {
		return sourcing.PersistedPublication{}, review.ErrForbidden
	}
	i, err := a.freshIdentity(ctx)
	identity := slot.plan.identity
	if err != nil || i.TenantID != identity.OrganizationID || i.UserID != identity.ActorID || i.EffectiveMemberID != identity.MemberID {
		return sourcing.PersistedPublication{}, review.ErrForbidden
	}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, i)
	receipt, err := a.titlePilotPublisher.Verify(ctx, slot.command)
	if err != nil {
		return sourcing.PersistedPublication{}, err
	}
	if receipt.OrganizationID != i.TenantID || receipt.ActorID != i.UserID || receipt.PublicationID != binding.ContextID ||
		receipt.ProductKey != binding.ProductKey || receipt.CatalogPublicationID != binding.PublicationID ||
		strconv.FormatUint(receipt.CatalogVersion, 10) != binding.CatalogVersion {
		return sourcing.PersistedPublication{}, agent.ErrConflict
	}
	published, err := a.titlePilotPublisher.Read(ctx, binding.ContextID)
	if err != nil {
		return sourcing.PersistedPublication{}, err
	}
	if published.Receipt.ActorID != receipt.ActorID || published.Receipt.InputHash != receipt.InputHash ||
		published.Receipt.EnvelopeHash != receipt.EnvelopeHash || published.Receipt.SnapshotHash != receipt.SnapshotHash ||
		published.Receipt.CatalogVersion != receipt.CatalogVersion || published.Receipt.CatalogPublicationID != receipt.CatalogPublicationID {
		return sourcing.PersistedPublication{}, agent.ErrConflict
	}
	return published, nil
}

func (a *productAgentApplication) authorizeTitlePilotDispatch(ctx context.Context, input aicapability.TextInputIdentity) error {
	slot, _ := ctx.Value(titlePilotSlotKey{}).(*titlePilotSlot)
	if slot == nil && input.Profile.AdapterPolicyVersion != governed.TitlePilotOpenAIAdapterPolicy {
		return nil
	}
	if slot == nil || slot.plan == nil || slot.plan.app != a || input.Operation != aicapability.OperationProductAgentDecision || input.Profile != slot.plan.profile ||
		input.OrganizationID != slot.plan.identity.OrganizationID || input.ActorID != slot.plan.identity.ActorID ||
		input.MemberID != slot.plan.identity.MemberID {
		return review.ErrForbidden
	}
	scope, err := a.Authorize(ctx, slot.binding)
	if err != nil {
		return err
	}
	run, found, err := a.store.Lookup(ctx, scope, slot.binding, slot.key)
	if err != nil || !found || run.State.RunID != input.AgentRunID || run.State.Phase != agent.Running ||
		run.State.Request.Limits != slot.limits {
		return agent.ErrUnavailable
	}
	quote, err := aicapability.QuoteText(input)
	if err != nil || !titlePilotQuoteCoversTariff(input.Profile, quote.MaximumTokens-input.Profile.MaximumCompletionTokens, quote.MaximumCostMicros) {
		return agent.ErrUnavailable
	}
	return nil
}

// runTitlePilot never advances from a replay, an uncertain commit, or an
// unconfirmed point settlement. The first original run row fences the batch.
func (a *productAgentApplication) runTitlePilot(ctx context.Context, plan *titlePilotPlan) ([]agent.Record, error) {
	if plan == nil || plan.app != a || a.titlePilotFixed == nil || len(plan.slots) != 2*titlePilotSampleCount {
		return nil, agent.ErrInvalid
	}
	if _, found, err := a.readTitlePilotRun(ctx, &plan.slots[0]); err != nil || found {
		if err != nil {
			return nil, errors.Join(errTitlePilotUnknown, err)
		}
		return nil, agent.ErrConflict
	}
	var completed []agent.Record
	for index := range plan.slots {
		slot := &plan.slots[index]
		execution := slot.context(ctx)
		if _, err := a.Authorize(execution, slot.binding); err != nil {
			return completed, err
		}
		request, err := a.startRequestWithProfile(execution, slot.binding, slot.key, "", "", nil, plan.profile, "", "0")
		if err != nil {
			return completed, err
		}
		runtime := a.runtime
		if slot.fixed {
			runtime = a.titlePilotFixed
		}
		record, err := runtime.StartFresh(execution, request)
		if err != nil {
			if errors.Is(err, agent.ErrConflict) {
				return completed, err
			}
			return completed, errors.Join(errTitlePilotUnknown, err)
		}
		if err := a.confirmTitlePilotCosts(execution, record); err != nil {
			return completed, err
		}
		completed = append(completed, record)
		if record.State.Phase != agent.HumanReviewRequired && record.State.Phase != agent.Stopped {
			return completed, errTitlePilotStopped
		}
		switch record.State.StopReason {
		case agent.StopDependency, agent.StopUnauthorized, agent.StopAudit, agent.StopCost, agent.StopTokens:
			return completed, errTitlePilotStopped
		}
	}
	return completed, nil
}

func (a *productAgentApplication) readTitlePilotRun(ctx context.Context, slot *titlePilotSlot) (agent.Record, bool, error) {
	if slot == nil {
		return agent.Record{}, false, agent.ErrInvalid
	}
	ctx = slot.context(ctx)
	scope, err := a.Authorize(ctx, slot.binding)
	if err != nil {
		return agent.Record{}, false, err
	}
	return a.store.Lookup(ctx, scope, slot.binding, slot.key)
}

func (a *productAgentApplication) confirmTitlePilotCosts(ctx context.Context, record agent.Record) error {
	state := record.State
	if state.PendingInvocationID != "" || state.StopReason == agent.StopModelUnknown || state.StopReason == agent.StopUsageUnknown ||
		state.StopReason == agent.StopExecutionOutcomeUnknown || state.Phase == agent.Running {
		return errTitlePilotUnknown
	}
	if _, err := a.Authorize(ctx, state.Request.Binding); err != nil {
		return errors.Join(errTitlePilotUnknown, err)
	}
	calls := 0
	for _, observation := range state.History {
		if observation.InvocationID == "" {
			continue
		}
		calls++
		if err := a.points.ConfirmModelInvocationSettlement(ctx, state.Scope.OrganizationID, observation.InvocationID, state.RunID); err != nil {
			return errors.Join(errTitlePilotUnknown, err)
		}
	}
	if calls != state.Usage.ModelCalls {
		return errTitlePilotUnknown
	}
	return nil
}
