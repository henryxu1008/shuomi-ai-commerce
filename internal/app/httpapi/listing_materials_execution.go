package httpapi

import (
	"context"
	"errors"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/listing/materials"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
)

var errMaterialsHumanRequired = errors.New("listing materials require explicit human confirmation")

type materialsPrepareInput struct {
	RequestKey           string                   `json:"requestKey"`
	ExpectedModeRevision string                   `json:"expectedModeRevision"`
	Template             *agentconfig.TemplateRef `json:"template,omitempty"`
	GoalSummary          string                   `json:"goalSummary,omitempty"`
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
	if a.runtime == nil || a.model == nil || a.profile == nil {
		return agentconfig.Snapshot{}, agentconfig.ErrUnavailable
	}
	profile, err := a.profile(ctx, id.OrganizationID)
	if err != nil {
		return agentconfig.Snapshot{}, agentconfig.ErrUnavailable
	}
	inspection, err := a.supply.InspectMaterialsRecord(ctx, recordID)
	if err != nil {
		return agentconfig.Snapshot{}, err
	}
	request := agent.Request{Key: in.RequestKey, Binding: inspection.Binding, GoalSummary: in.GoalSummary, PolicyVersion: materials.PolicyVersion, PromptVersion: materials.PromptVersion, Limits: a.parent.config.Limits}
	return a.parent.configuration.Prepare(ctx, agentconfig.StartCommand{Scope: agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, MemberID: id.MemberID, ExpectedModeRevision: in.ExpectedModeRevision, AgentID: materials.AgentID, AgentVersion: materials.AgentVersion, Template: in.Template, Request: request, ExecutionModelProfile: profile})
}

func (a *listingMaterialsApplication) readRun(ctx context.Context, recordID, key string) (agent.Record, agentconfig.Snapshot, error) {
	if !agentconfig.UUID(key) {
		return agent.Record{}, agentconfig.Snapshot{}, agentconfig.ErrInvalid
	}
	id, _, binding, err := a.readBase(ctx, recordID)
	if err != nil {
		return agent.Record{}, agentconfig.Snapshot{}, err
	}
	scope := agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}
	run, found, err := a.parent.store.Lookup(ctx, scope, binding, key)
	if err != nil {
		return run, agentconfig.Snapshot{}, err
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
	if snapshot.AgentID != materials.AgentID || snapshot.AgentVersion != materials.AgentVersion || snapshot.MemberID != id.MemberID || snapshot.Request != request || request.Binding != binding {
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
	id, _, binding, err := a.readBase(ctx, recordID)
	if err != nil {
		return agent.Record{}, false, err
	}
	snapshot, err = a.parent.configuration.LoadSnapshot(ctx, agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, ref)
	if err != nil {
		return agent.Record{}, false, err
	}
	if snapshot.AgentID != materials.AgentID || snapshot.AgentVersion != materials.AgentVersion || snapshot.MemberID != id.MemberID || snapshot.Request.Key != key || snapshot.Request.Binding != binding {
		return agent.Record{}, false, agentconfig.ErrConflict
	}
	if a.runtime == nil {
		return agent.Record{}, false, agentconfig.ErrUnavailable
	}
	request := snapshot.Request
	request.ConfigurationSnapshotRef = ref
	run, attempted, err := a.runtime.StartWithClaimAttempt(ctx, request)
	if err == nil {
		err = a.finishAutomaticAdoption(ctx, recordID, run)
	}
	return run, attempted, err
}

func (a *listingMaterialsApplication) resume(ctx context.Context, recordID, key string, ref agent.ConfigurationSnapshotRef, revision uint64, feedback string) (agent.Record, bool, error) {
	run, _, err := a.readRun(ctx, recordID, key)
	if err != nil {
		return run, true, err
	}
	if run.State.Request.ConfigurationSnapshotRef != ref {
		return run, true, agentconfig.ErrConflict
	}
	if a.runtime == nil {
		return run, true, agentconfig.ErrUnavailable
	}
	next, attempted, err := a.runtime.ResumeWithClaimAttempt(ctx, run.State.Request, revision, feedback)
	if err == nil {
		err = a.finishAutomaticAdoption(ctx, recordID, next)
	}
	return next, attempted, err
}

// Only the admitted Start/resume path may initiate an automatic local save.
// Reads and original run replay have no new side effects; Target remains the
// sole durable adoption receipt and enforces current authority at commit.
func (a *listingMaterialsApplication) finishAutomaticAdoption(ctx context.Context, recordID string, run agent.Record) error {
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
	if !materialsRunReviewable(run.State) {
		return record.TargetReceipt{}, agentconfig.ErrConflict
	}
	return a.supply.Targets.ReadCommand(ctx, materialsAdoptionKey(run.State))
}

func (a *listingMaterialsApplication) adopt(ctx context.Context, recordID, key string, ref agent.ConfigurationSnapshotRef, candidateHash string, human bool) (record.TargetReceipt, error) {
	run, _, err := a.readRun(ctx, recordID, key)
	if err != nil {
		return record.TargetReceipt{}, err
	}
	if !materialsRunReviewable(run.State) || run.State.Request.ConfigurationSnapshotRef != ref || run.State.Validation.CandidateHash != candidateHash {
		return record.TargetReceipt{}, agentconfig.ErrConflict
	}
	commandKey := materialsAdoptionKey(run.State)
	prior, err := a.supply.Targets.ReadCommand(ctx, commandKey)
	if err == nil {
		prior.Replayed = true
		return prior, nil
	}
	if !errors.Is(err, record.ErrNotFound) {
		return record.TargetReceipt{}, err
	}
	inspection, err := a.supply.ReadMaterialsInspection(ctx, run.State.Request.Binding)
	if err != nil {
		return record.TargetReceipt{}, err
	}
	proposal, err := materials.ParseProposal(run.State.DomainProposal.Payload)
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
