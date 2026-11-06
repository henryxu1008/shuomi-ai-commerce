package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/listing/materials"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
)

var errMaterialsHumanRequired = errors.New("listing materials require explicit human confirmation")

type materialsPrepareInput struct {
	RequestKey           string                       `json:"requestKey"`
	ExpectedModeRevision string                       `json:"expectedModeRevision"`
	Template             *agentconfig.TemplateRef     `json:"template,omitempty"`
	GoalSummary          string                       `json:"goalSummary,omitempty"`
	CandidateCategoryID  materialsCandidateCategoryID `json:"candidateCategoryId,omitempty"`
}

type materialsCandidateCategoryID string

func (v *materialsCandidateCategoryID) UnmarshalJSON(raw []byte) error {
	var s string
	if len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, &s) != nil {
		return agentconfig.ErrInvalid
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != s {
		return agentconfig.ErrInvalid
	}
	*v = materialsCandidateCategoryID(s)
	return nil
}

func (a *listingMaterialsApplication) prepare(ctx context.Context, recordID string, in materialsPrepareInput) (agentconfig.Snapshot, error) {
	if !agentconfig.UUID(in.RequestKey) || !agent.ValidGoalSummary(in.GoalSummary) {
		return agentconfig.Snapshot{}, agentconfig.ErrInvalid
	}
	if _, err := agentconfig.ParseModeRevision(in.ExpectedModeRevision); err != nil {
		return agentconfig.Snapshot{}, err
	}
	id, err := a.currentIdentity(ctx)
	if err != nil {
		return agentconfig.Snapshot{}, err
	}
	if a.parent.configuration == nil {
		return agentconfig.Snapshot{}, agentconfig.ErrUnavailable
	}
	prior, found, err := a.parent.configuration.FindMaterialsPreparation(ctx, agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, in.RequestKey)
	if err != nil {
		return agentconfig.Snapshot{}, err
	}
	if found {
		if prior.MemberID != id.MemberID {
			return agentconfig.Snapshot{}, preparation.ErrForbidden
		}
		if prior.Request.Binding.ContextID != recordID || prior.ExpectedModeRevision != in.ExpectedModeRevision || prior.Request.GoalSummary != in.GoalSummary || (prior.Template == nil) != (in.Template == nil) || prior.Template != nil && *prior.Template != *in.Template {
			return agentconfig.Snapshot{}, agentconfig.ErrConflict
		}
		query := prior.Request.Binding.ContextKind == agent.CandidateRulesContextKind
		if query != (in.CandidateCategoryID != "") || query && prior.Request.Binding.CandidateRules.CandidateCategoryID != string(in.CandidateCategoryID) {
			return agentconfig.Snapshot{}, agentconfig.ErrConflict
		}
		_, base, binding, e := a.readBase(ctx, recordID)
		if e != nil {
			return agentconfig.Snapshot{}, e
		}
		if !materialsBindingMatchesReadBase(binding, prior.Request.Binding, base) {
			return agentconfig.Snapshot{}, agentconfig.ErrConflict
		}
		return prior, nil
	}
	if a.runtime == nil || a.model == nil || a.profile == nil {
		return agentconfig.Snapshot{}, agentconfig.ErrUnavailable
	}
	profile, err := a.profile(ctx, id.OrganizationID)
	if err != nil {
		return agentconfig.Snapshot{}, agentconfig.ErrUnavailable
	}
	var binding agent.Binding
	if in.CandidateCategoryID != "" {
		value, e := a.supply.InspectCandidateRulesRecord(ctx, recordID, string(in.CandidateCategoryID))
		if e != nil {
			return agentconfig.Snapshot{}, e
		}
		binding = value.Binding
	} else {
		inspection, e := a.supply.InspectMaterialsRecord(ctx, recordID)
		if e != nil {
			return agentconfig.Snapshot{}, e
		}
		binding = inspection.Binding
	}
	limits := a.parent.config.Limits
	limits.Runtime = min(limits.Runtime, 90*time.Second)
	request := agent.Request{Key: in.RequestKey, Binding: binding, GoalSummary: in.GoalSummary, PolicyVersion: materials.PolicyVersion, PromptVersion: materials.PromptVersion, Limits: limits}
	return a.parent.configuration.Prepare(ctx, agentconfig.StartCommand{Scope: agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, MemberID: id.MemberID, ExpectedModeRevision: in.ExpectedModeRevision, AgentID: materials.AgentID, AgentVersion: materials.AgentVersion, Template: in.Template, Request: request, ExecutionModelProfile: profile})
}

func (a *listingMaterialsApplication) readRun(ctx context.Context, recordID, key string) (agent.Record, agentconfig.Snapshot, error) {
	if !agentconfig.UUID(key) {
		return agent.Record{}, agentconfig.Snapshot{}, agentconfig.ErrInvalid
	}
	id, base, binding, err := a.readBase(ctx, recordID)
	if err != nil {
		return agent.Record{}, agentconfig.Snapshot{}, err
	}
	scope := agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}
	var run agent.Record
	found := false
	for _, kind := range []string{"listing-target-record", agent.CandidateRulesContextKind} {
		value, exists, e := a.parent.store.LookupContext(ctx, scope, kind, recordID, key)
		if e != nil {
			return run, agentconfig.Snapshot{}, e
		}
		if exists {
			if found {
				return run, agentconfig.Snapshot{}, agentconfig.ErrConflict
			}
			run, found = value, true
		}
	}
	if !found {
		return run, agentconfig.Snapshot{}, record.ErrNotFound
	}
	snapshot, err := a.parent.configuration.LoadSnapshot(ctx, scope, run.State.Request.ConfigurationSnapshotRef)
	if err != nil {
		return run, snapshot, err
	}
	request := run.State.Request
	request.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{}
	if snapshot.AgentID != materials.AgentID || !agentconfig.MaterialsStoredVersion(snapshot.AgentVersion, request.PolicyVersion, request.PromptVersion) || snapshot.MemberID != id.MemberID || snapshot.Request != request || !materialsBindingMatchesReadBase(binding, request.Binding, base) {
		return run, snapshot, agentconfig.ErrConflict
	}
	return run, snapshot, nil
}

func (a *listingMaterialsApplication) start(ctx context.Context, recordID, key string, ref agent.ConfigurationSnapshotRef) (agent.Record, bool, error) {
	if !agentconfig.UUID(key) || !ref.ValidOrAbsent() || ref.Absent() {
		return agent.Record{}, false, agentconfig.ErrInvalid
	}
	// Return an exact existing run under current read authority before any
	// new execution gate. Do not require its original Target to stay head.
	prior, snapshot, err := a.readRun(ctx, recordID, key)
	if err == nil {
		if prior.State.Request.ConfigurationSnapshotRef != ref {
			return prior, true, agentconfig.ErrConflict
		}
		return prior, true, nil
	}
	if !errors.Is(err, record.ErrNotFound) {
		return prior, true, err
	}
	id, base, binding, err := a.readBase(ctx, recordID)
	if err != nil {
		return agent.Record{}, false, err
	}
	snapshot, err = a.parent.configuration.LoadSnapshot(ctx, agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, ref)
	if err != nil {
		return agent.Record{}, false, err
	}
	if snapshot.AgentID != materials.AgentID || snapshot.AgentVersion != materials.AgentVersion || snapshot.MemberID != id.MemberID || snapshot.Request.Key != key || !materialsBindingMatchesReadBase(binding, snapshot.Request.Binding, base) {
		return agent.Record{}, false, agentconfig.ErrConflict
	}
	if a.runtime == nil {
		return agent.Record{}, false, agentconfig.ErrUnavailable
	}
	request := snapshot.Request
	request.ConfigurationSnapshotRef = ref
	modelCtx, cancel := materialsModelContext(ctx)
	run, attempted, err := a.runtime.StartWithClaimAttempt(modelCtx, request)
	cancel()
	if err == nil {
		err = a.finishAutomaticAdoption(ctx, recordID, run)
	}
	return run, attempted, err
}

func (a *listingMaterialsApplication) resume(ctx context.Context, recordID, key string, ref agent.ConfigurationSnapshotRef, revision uint64, feedback string) (agent.Record, bool, error) {
	run, snapshot, err := a.readRun(ctx, recordID, key)
	if err != nil {
		return run, true, err
	}
	if snapshot.AgentVersion != materials.AgentVersion || run.State.Request.ConfigurationSnapshotRef != ref {
		return run, true, agentconfig.ErrConflict
	}
	if a.runtime == nil {
		return run, true, agentconfig.ErrUnavailable
	}
	modelCtx, cancel := materialsModelContext(ctx)
	next, attempted, err := a.runtime.ResumeWithClaimAttempt(modelCtx, run.State.Request, revision, feedback)
	cancel()
	if err == nil {
		err = a.finishAutomaticAdoption(ctx, recordID, next)
	}
	return next, attempted, err
}

// The enclosing two-minute route also owns local Target reads and commit.
// Model work cannot consume the time reserved for that existing owner.
func materialsModelContext(ctx context.Context) (context.Context, context.CancelFunc) {
	deadline := time.Now().Add(90 * time.Second)
	if enclosing, ok := ctx.Deadline(); ok && enclosing.Add(-30*time.Second).Before(deadline) {
		deadline = enclosing.Add(-30 * time.Second)
	}
	return context.WithDeadline(ctx, deadline)
}

// Only the admitted Start/resume path may initiate an automatic local save.
// Reads and original run replay have no new side effects; Target remains the
// sole durable adoption receipt and enforces current authority at commit.
func (a *listingMaterialsApplication) finishAutomaticAdoption(ctx context.Context, recordID string, run agent.Record) error {
	if run.State.Request.Binding.ContextKind == agent.CandidateRulesContextKind {
		return nil
	}
	if !materialsRunReviewable(run.State) {
		return nil
	}
	id, err := a.currentIdentity(ctx)
	if err != nil {
		return err
	}
	mode, err := a.parent.configuration.LoadModeSnapshot(ctx, id, run.State.Request.ModeSnapshotRef)
	if err != nil {
		return err
	}
	if mode.Mode != agent.ModeAutoApprove && mode.Mode != agent.ModeFullAccess {
		return nil
	}
	_, err = a.adopt(ctx, recordID, run.State.Request.Key, run.State.Request.ConfigurationSnapshotRef, run.State.Validation.CandidateHash, false)
	if errors.Is(err, errMaterialsHumanRequired) {
		return nil
	}
	return err
}

func (a *listingMaterialsApplication) adoption(ctx context.Context, recordID, key string) (record.TargetReceipt, error) {
	run, _, err := a.readRun(ctx, recordID, key)
	if err != nil {
		return record.TargetReceipt{}, err
	}
	if !materialsHistoricalReviewable(run.State) {
		return record.TargetReceipt{}, agentconfig.ErrConflict
	}
	return a.readAdoptionReceipt(ctx, run.State)
}

// Receipt replay is correlated to the immutable original input and candidate,
// without applying current new-write mode/configuration/head admission.
func verifyMaterialsAdoptionReceipt(base record.TargetRecord, state agent.State, prior record.TargetReceipt) error {
	if !materialsHistoricalReviewable(state) || base.ID != state.Request.Binding.ContextID {
		return agentconfig.ErrConflict
	}
	proposal, err := materials.ParseRequestProposal(state.Request, *state.DomainProposal)
	if err != nil {
		return agentconfig.ErrConflict
	}
	expected, err := materials.ApplyProposal(base, proposal)
	if err != nil || prior.Record.TargetID != base.TargetID || prior.Record.Revision != base.Revision+1 || prior.Record.Source.ID != base.Source.ID || prior.Record.EffectiveVersion != base.EffectiveVersion || prior.Record.ApplyReceiptID != base.ApplyReceiptID || collection.Digest(prior.Record.Input) != collection.Digest(expected) {
		return agentconfig.ErrConflict
	}
	return nil
}

func (a *listingMaterialsApplication) readAdoptionReceipt(ctx context.Context, state agent.State) (record.TargetReceipt, error) {
	prior, err := a.supply.Targets.ReadCommand(ctx, materialsAdoptionKey(state))
	if err != nil {
		return prior, err
	}
	_, base, binding, err := a.readBase(ctx, state.Request.Binding.ContextID)
	if err != nil {
		return record.TargetReceipt{}, err
	}
	if binding != state.Request.Binding || verifyMaterialsAdoptionReceipt(base, state, prior) != nil {
		return record.TargetReceipt{}, agentconfig.ErrConflict
	}
	return prior, nil
}

func (a *listingMaterialsApplication) adopt(ctx context.Context, recordID, key string, ref agent.ConfigurationSnapshotRef, candidateHash string, human bool) (record.TargetReceipt, error) {
	run, _, err := a.readRun(ctx, recordID, key)
	if err != nil {
		return record.TargetReceipt{}, err
	}
	if run.State.Request.Binding.ContextKind == agent.CandidateRulesContextKind || run.State.Request.Binding.CandidateRules != (agent.CandidateRulesRef{}) {
		return record.TargetReceipt{}, agentconfig.ErrConflict
	}
	if !materialsHistoricalReviewable(run.State) || run.State.Request.ConfigurationSnapshotRef != ref || run.State.Validation.CandidateHash != candidateHash {
		return record.TargetReceipt{}, agentconfig.ErrConflict
	}
	commandKey := materialsAdoptionKey(run.State)
	prior, err := a.readAdoptionReceipt(ctx, run.State)
	if err == nil {
		prior.Replayed = true
		return prior, nil
	}
	if !errors.Is(err, record.ErrNotFound) {
		return record.TargetReceipt{}, err
	}
	if !materialsRunReviewable(run.State) {
		return record.TargetReceipt{}, agentconfig.ErrConflict
	}
	inspection, err := a.supply.ReadMaterialsInspection(ctx, run.State.Request.Binding)
	if err != nil {
		return record.TargetReceipt{}, err
	}
	proposal, err := materials.ParseRequestProposal(run.State.Request, *run.State.DomainProposal)
	if err != nil {
		return record.TargetReceipt{}, err
	}
	validation, err := (materials.Validator{Reader: a.supply}).ValidateDomain(ctx, run.State.Request, *run.State.DomainProposal, run.State.History)
	if err != nil {
		return record.TargetReceipt{}, err
	}
	if !validation.Valid {
		return record.TargetReceipt{}, agentconfig.ErrConflict
	}
	preview, err := materials.Preview(inspection, proposal)
	if err != nil {
		return record.TargetReceipt{}, err
	}
	guard := func(ctx context.Context, facts record.TargetCommitFacts) error {
		current, config, e := a.readRun(ctx, recordID, key)
		if e != nil {
			return e
		}
		if !materialsRunReviewable(current.State) || current.State.RunID != run.State.RunID || current.State.Validation.CandidateHash != candidateHash || current.State.Request != run.State.Request {
			return agentconfig.ErrConflict
		}
		identity, e := a.currentIdentity(ctx)
		if e != nil {
			return e
		}
		mode, e := a.parent.configuration.CheckRegisteredTextExecutionMode(ctx, identity, ref, run.State.Request.ModeSnapshotRef)
		if e != nil {
			return e
		}
		if config.MemberID != identity.MemberID {
			return preparation.ErrForbidden
		}
		if e = a.checkTemplate(ctx, config); e != nil {
			return e
		}
		scope, e := a.supply.Authorization.Authorize(ctx, preparation.PermissionManage)
		if e != nil || scope.OrganizationID != identity.OrganizationID || scope.ActorID != identity.ActorID || scope.MemberID != identity.MemberID {
			return preparation.ErrForbidden
		}
		if e = materials.VerifyCommit(inspection, proposal, facts); e != nil {
			return e
		}
		needsHuman := preview.RequiresHuman || len(run.State.Unresolved) != 0 || len(facts.Record.Result.Issues) != 0 || !facts.Record.Result.ReadyForUpload
		automatic := agent.RulePassed
		if needsHuman {
			automatic = agent.RuleNeedsHuman
		}
		decision := materials.EvaluateAdoption(mode.Mode, agent.ExecutionPolicyChecks{OwnerAuthorized: true, ContentValid: true, HumanContentConfirmed: human, Automatic: automatic}, needsHuman)
		if decision.Outcome == agent.PolicyHumanRequired {
			return errMaterialsHumanRequired
		}
		if decision.Outcome != agent.PolicyAllow {
			return preparation.ErrForbidden
		}
		return nil
	}
	return a.supply.Targets.CreateForAgent(ctx, commandKey, preview.Input, guard)
}
