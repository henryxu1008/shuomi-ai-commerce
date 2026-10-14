package httpapi

import (
	"context"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/selection"
	"time"
)

type selectionInput struct {
	ContextID  string            `json:"contextId"`
	Candidates []selectionChoice `json:"candidates"`
}
type selectionChoice struct {
	ItemID                string `json:"itemId"`
	ExpectedRevision      string `json:"expectedRevision"`
	OriginalPublicationID string `json:"originalPublicationId"`
	OriginalVersion       string `json:"originalVersion"`
}

func (in selectionInput) inputs() ([]collection.SelectionInput, error) {
	if !agentconfig.UUID(in.ContextID) || len(in.Candidates) < 2 || len(in.Candidates) > agent.MaxSelectionProducts {
		return nil, agent.ErrInvalid
	}
	choices := make([]collection.SelectionInput, 0, len(in.Candidates))
	for _, c := range in.Candidates {
		revision, e := strconv.ParseInt(c.ExpectedRevision, 10, 64)
		version, ve := strconv.ParseUint(c.OriginalVersion, 10, 63)
		if !agentconfig.UUID(c.ItemID) || e != nil || revision < 1 || strconv.FormatInt(revision, 10) != c.ExpectedRevision || ve != nil || version < 1 || strconv.FormatUint(version, 10) != c.OriginalVersion || !agent.ValidID(c.OriginalPublicationID) {
			return nil, agent.ErrInvalid
		}
		choices = append(choices, collection.SelectionInput{ItemID: c.ItemID, ExpectedRevision: revision, OriginalPublicationID: c.OriginalPublicationID, OriginalVersion: version})
	}
	return choices, nil
}

type selectionPrepareInput struct {
	selectionInput
	RequestKey           string `json:"requestKey"`
	ExpectedModeRevision string `json:"expectedModeRevision"`
	GoalSummary          string `json:"goalSummary"`
}

func (a *productSelectionApplication) inspectInput(ctx context.Context, in selectionInput) (agent.Binding, selection.Inspection, error) {
	choices, e := in.inputs()
	if e != nil {
		return agent.Binding{}, selection.Inspection{}, e
	}
	id, e := a.freshIdentity(ctx)
	if e != nil {
		return agent.Binding{}, selection.Inspection{}, e
	}
	return a.inspector.Prepare(authidentity.WithAuthenticatedIdentity(ctx, id), in.ContextID, choices)
}
func (a *productSelectionApplication) prepare(ctx context.Context, in selectionPrepareInput) (agentconfig.Snapshot, error) {
	if !agentconfig.UUID(in.RequestKey) || !agent.ValidGoalSummary(in.GoalSummary) {
		return agentconfig.Snapshot{}, agent.ErrInvalid
	}
	if _, e := agentconfig.ParseModeRevision(in.ExpectedModeRevision); e != nil {
		return agentconfig.Snapshot{}, e
	}
	binding, _, e := a.inspectInput(ctx, in.selectionInput)
	if e != nil {
		return agentconfig.Snapshot{}, e
	}
	id, e := a.currentExecutionIdentity(ctx)
	if e != nil {
		return agentconfig.Snapshot{}, e
	}
	if a.runtime == nil || a.model == nil || a.profile == nil || a.configuration == nil {
		return agentconfig.Snapshot{}, agent.ErrUnavailable
	}
	profile, e := a.profile(ctx, id.OrganizationID)
	if e != nil {
		return agentconfig.Snapshot{}, agent.ErrUnavailable
	}
	request := agent.Request{Key: in.RequestKey, Binding: binding, GoalSummary: in.GoalSummary, PolicyVersion: agentconfig.SelectionPolicyVersion, PromptVersion: agentconfig.SelectionPromptVersion, Limits: a.parent.config.Limits}
	return a.configuration.Prepare(ctx, agentconfig.StartCommand{Scope: agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, MemberID: id.MemberID, ExpectedModeRevision: in.ExpectedModeRevision, AgentID: agentconfig.SelectionAgentID, AgentVersion: agentconfig.SelectionAgentVersion, Request: request, ExecutionModelProfile: profile})
}
func (a *productSelectionApplication) readRun(ctx context.Context, contextID, key string) (agent.Record, agentconfig.Snapshot, error) {
	if !agentconfig.UUID(contextID) || !agentconfig.UUID(key) {
		return agent.Record{}, agentconfig.Snapshot{}, agent.ErrInvalid
	}
	id, e := a.currentIdentity(ctx)
	if e != nil {
		return agent.Record{}, agentconfig.Snapshot{}, e
	}
	if a.store == nil || a.configuration == nil {
		return agent.Record{}, agentconfig.Snapshot{}, agent.ErrUnavailable
	}
	scope := agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}
	run, found, e := a.store.LookupContext(ctx, scope, agent.SelectionContextKind, contextID, key)
	if e != nil {
		return run, agentconfig.Snapshot{}, e
	}
	if !found {
		return run, agentconfig.Snapshot{}, agentconfig.ErrNotFound
	}
	frozen, e := a.configuration.LoadSnapshot(ctx, scope, run.State.Request.ConfigurationSnapshotRef)
	if e != nil {
		return run, frozen, e
	}
	expected := run.State.Request
	expected.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{}
	if frozen.Scope != scope || frozen.MemberID != id.MemberID || frozen.AgentID != agentconfig.SelectionAgentID || frozen.AgentVersion != agentconfig.SelectionAgentVersion || frozen.Request != expected || expected.Binding.ContextID != contextID || expected.Key != key {
		return run, frozen, agent.ErrConflict
	}
	current, e := a.inspector.AuthorizeHistory(ctx, expected.Binding)
	if e == nil && current != (collection.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID, MemberID: id.MemberID}) {
		e = agentconfig.ErrForbidden
	}
	if e != nil {
		return run, frozen, e
	}
	// Only the existing Agent expiration owner fences an abandoned runtime.
	// This never redispatches a model or resolves uncertain charges by guessing.
	if run.State.Phase == agent.Running && time.Now().After(run.State.Deadline.Add(30*time.Second)) {
		run, e = a.store.FinalizeExpiredRunning(ctx, scope, expected.Binding, key, run.State.Revision, time.Now())
		if e != nil {
			return run, frozen, e
		}
	}
	return run, frozen, nil
}
func (a *productSelectionApplication) start(ctx context.Context, contextID, key string, ref agent.ConfigurationSnapshotRef) (agent.Record, bool, error) {
	if !agentconfig.UUID(contextID) || !agentconfig.UUID(key) || ref.Absent() || !ref.ValidOrAbsent() {
		return agent.Record{}, false, agent.ErrInvalid
	}
	id, e := a.currentIdentity(ctx)
	if e != nil {
		return agent.Record{}, false, e
	}
	if a.configuration == nil || a.store == nil {
		return agent.Record{}, false, agent.ErrUnavailable
	}
	frozen, e := a.configuration.LoadSnapshot(ctx, agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, ref)
	if e != nil {
		return agent.Record{}, false, e
	}
	if frozen.AgentID != agentconfig.SelectionAgentID || frozen.AgentVersion != agentconfig.SelectionAgentVersion || frozen.MemberID != id.MemberID || frozen.Request.Key != key || frozen.Request.Binding.ContextID != contextID {
		return agent.Record{}, false, agent.ErrConflict
	}
	prior, found, e := a.store.Lookup(ctx, frozen.Scope, frozen.Request.Binding, key)
	if e != nil {
		return prior, true, e
	}
	if found {
		if prior.State.Request.ConfigurationSnapshotRef != ref {
			return prior, true, agent.ErrConflict
		}
		prior, _, e = a.readRun(ctx, contextID, key)
		return prior, true, e
	}
	execution, e := a.currentExecutionIdentity(ctx)
	if e != nil {
		return agent.Record{}, false, e
	}
	if execution != id || a.runtime == nil {
		return agent.Record{}, false, agent.ErrUnavailable
	}
	request := frozen.Request
	request.ConfigurationSnapshotRef = ref
	ctx = context.WithValue(ctx, selectionExecutionContextKey{}, request)
	return a.runtime.StartWithClaimAttempt(ctx, request)
}
func (a *productSelectionApplication) resume(ctx context.Context, contextID, key string, ref agent.ConfigurationSnapshotRef, revision uint64, feedback string) (agent.Record, bool, error) {
	run, _, e := a.readRun(ctx, contextID, key)
	if e != nil {
		return run, true, e
	}
	if run.State.Request.ConfigurationSnapshotRef != ref || a.runtime == nil {
		return run, true, agent.ErrConflict
	}
	if _, e = a.currentExecutionIdentity(ctx); e != nil {
		return run, true, e
	}
	ctx = context.WithValue(ctx, selectionExecutionContextKey{}, run.State.Request)
	return a.runtime.ResumeWithClaimAttempt(ctx, run.State.Request, revision, feedback)
}

type selectionRunDTO struct {
	RunID                    string                         `json:"runId"`
	ContextID                string                         `json:"contextId"`
	RequestKey               string                         `json:"requestKey"`
	Phase                    agent.Phase                    `json:"phase"`
	Revision                 string                         `json:"revision"`
	StopReason               agent.StopReason               `json:"stopReason,omitempty"`
	ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
	ModeSnapshotRef          agent.ModeSnapshotRef          `json:"modeSnapshotRef"`
	ExecutionMode            agent.ExecutionMode            `json:"executionMode"`
	ModeRevision             string                         `json:"modeRevision"`
	GoalSummary              string                         `json:"goalSummary"`
	Inspection               *selection.Inspection          `json:"inspection,omitempty"`
	Proposal                 *selection.Proposal            `json:"proposal,omitempty"`
	Validated                bool                           `json:"validated"`
	Unresolved               []string                       `json:"unresolved"`
	Tokens                   int64                          `json:"tokens"`
	CostMicros               int64                          `json:"costMicros"`
	Currency                 string                         `json:"currency"`
	UsageStatus              string                         `json:"usageStatus"`
}

func (a *productSelectionApplication) projectRun(ctx context.Context, run agent.Record, frozen agentconfig.Snapshot) (selectionRunDTO, error) {
	id, e := a.currentIdentity(ctx)
	if e != nil {
		return selectionRunDTO{}, e
	}
	mode, e := a.configuration.LoadModeSnapshot(ctx, id, run.State.Request.ModeSnapshotRef)
	if e != nil {
		return selectionRunDTO{}, e
	}
	s := run.State
	if frozen.MemberID != id.MemberID || s.Scope != (agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}) {
		return selectionRunDTO{}, agentconfig.ErrForbidden
	}
	out := selectionRunDTO{RunID: s.RunID, ContextID: s.Request.Binding.ContextID, RequestKey: s.Request.Key, Phase: s.Phase, Revision: strconv.FormatUint(s.Revision, 10), StopReason: s.StopReason, ConfigurationSnapshotRef: s.Request.ConfigurationSnapshotRef, ModeSnapshotRef: mode.Ref(), ExecutionMode: mode.Mode, ModeRevision: mode.Revision, GoalSummary: s.Request.GoalSummary, Unresolved: append([]string{}, s.Unresolved...), Tokens: s.Usage.Tokens, CostMicros: s.Usage.CostMicros, Currency: s.Request.Limits.Currency, UsageStatus: "observed"}
	if s.Validation != nil {
		out.Unresolved = append(out.Unresolved, s.Validation.Unresolved...)
	}
	if s.PendingInvocationID != "" || s.Phase == agent.Running || s.StopReason == agent.StopUsageUnknown || s.StopReason == agent.StopModelUnknown || s.StopReason == agent.StopExecutionOutcomeUnknown {
		out.UsageStatus = "unknown_reserved"
	}
	if inspection, proposal, valid := selection.ValidatedResult(s); valid {
		out.Inspection, out.Proposal, out.Validated = &inspection, &proposal, true
	} else {
		for _, o := range s.History {
			if o.Tool == agentconfig.SelectionInspectToolRef() && o.AuditStatus == commercetool.AuditStatusRecorded && agent.ValidID(o.CallID) && o.InvocationID == "" {
				inspection, e := selection.ParseInspection(o.Output)
				if e == nil && selection.InspectionMatchesBinding(inspection, s.Request.Binding) {
					out.Inspection = &inspection
				}
			}
		}
	}
	return out, nil
}
