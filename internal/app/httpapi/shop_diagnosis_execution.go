package httpapi

import (
	"context"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	c "task-processor/internal/operationscockpit"
	"task-processor/internal/shopdiagnosis"
	"time"
)

type diagnosisInput struct {
	ContextID string `json:"contextId"`
	StoreID   string `json:"storeId"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

func (in diagnosisInput) inputs() (c.Period, error) {
	p := c.Period{Start: in.StartDate, End: in.EndDate}
	if !agentconfig.UUID(in.ContextID) || !c.UUID(in.StoreID) || !p.Valid() {
		return c.Period{}, agent.ErrInvalid
	}
	return p, nil
}

type diagnosisPrepareInput struct {
	diagnosisInput
	RequestKey           string `json:"requestKey"`
	ExpectedModeRevision string `json:"expectedModeRevision"`
	GoalSummary          string `json:"goalSummary"`
}

func (a *shopDiagnosisApplication) inspectInput(ctx context.Context, in diagnosisInput) (agent.Binding, shopdiagnosis.Inspection, error) {
	period, e := in.inputs()
	if e != nil {
		return agent.Binding{}, shopdiagnosis.Inspection{}, e
	}
	id, e := a.freshIdentity(ctx)
	if e != nil {
		return agent.Binding{}, shopdiagnosis.Inspection{}, e
	}
	return a.inspector.Prepare(authidentity.WithAuthenticatedIdentity(ctx, id), in.ContextID, in.StoreID, period)
}
func (a *shopDiagnosisApplication) prepare(ctx context.Context, in diagnosisPrepareInput) (agentconfig.Snapshot, error) {
	if !agentconfig.UUID(in.RequestKey) || !agent.ValidGoalSummary(in.GoalSummary) {
		return agentconfig.Snapshot{}, agent.ErrInvalid
	}
	if _, e := agentconfig.ParseModeRevision(in.ExpectedModeRevision); e != nil {
		return agentconfig.Snapshot{}, e
	}
	binding, _, e := a.inspectInput(ctx, in.diagnosisInput)
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
	request := agent.Request{Key: in.RequestKey, Binding: binding, GoalSummary: in.GoalSummary, PolicyVersion: agentconfig.DiagnosisPolicyVersion, PromptVersion: agentconfig.DiagnosisPromptVersion, Limits: a.parent.config.Limits}
	return a.configuration.Prepare(ctx, agentconfig.StartCommand{Scope: agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, MemberID: id.MemberID, ExpectedModeRevision: in.ExpectedModeRevision, AgentID: agentconfig.DiagnosisAgentID, AgentVersion: agentconfig.DiagnosisAgentVersion, Request: request, ExecutionModelProfile: profile})
}
func (a *shopDiagnosisApplication) readRun(ctx context.Context, contextID, key string) (agent.Record, agentconfig.Snapshot, error) {
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
	run, found, e := a.store.LookupContext(ctx, scope, agent.StoreDiagnosisContextKind, contextID, key)
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
	if frozen.Scope != scope || frozen.MemberID != id.MemberID || frozen.AgentID != agentconfig.DiagnosisAgentID || frozen.AgentVersion != agentconfig.DiagnosisAgentVersion || frozen.Request != expected || expected.Binding.ContextID != contextID || expected.Key != key {
		return run, frozen, agent.ErrConflict
	}
	current, e := a.inspector.AuthorizeHistory(ctx, expected.Binding)
	if e == nil && current != (agent.ExecutionIdentity{OrganizationID: id.OrganizationID, ActorID: id.ActorID, MemberID: id.MemberID}) {
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
func (a *shopDiagnosisApplication) start(ctx context.Context, contextID, key string, ref agent.ConfigurationSnapshotRef) (agent.Record, bool, error) {
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
	if frozen.AgentID != agentconfig.DiagnosisAgentID || frozen.AgentVersion != agentconfig.DiagnosisAgentVersion || frozen.MemberID != id.MemberID || frozen.Request.Key != key || frozen.Request.Binding.ContextID != contextID {
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
	ctx = context.WithValue(ctx, diagnosisExecutionContextKey{}, request)
	if _, e = a.Authorize(ctx, request.Binding); e != nil {
		return agent.Record{}, false, e
	}
	return a.runtime.StartWithClaimAttempt(ctx, request)
}
func (a *shopDiagnosisApplication) resume(ctx context.Context, contextID, key string, ref agent.ConfigurationSnapshotRef, revision uint64, feedback string) (agent.Record, bool, error) {
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
	ctx = context.WithValue(ctx, diagnosisExecutionContextKey{}, run.State.Request)
	if _, e = a.Authorize(ctx, run.State.Request.Binding); e != nil {
		return run, true, e
	}
	return a.runtime.ResumeWithClaimAttempt(ctx, run.State.Request, revision, feedback)
}

type diagnosisRunDTO struct {
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
	Inspection               *shopdiagnosis.Inspection      `json:"inspection,omitempty"`
	Proposal                 *shopdiagnosis.Proposal        `json:"proposal,omitempty"`
	Validated                bool                           `json:"validated"`
	Unresolved               []string                       `json:"unresolved"`
	Tokens                   int64                          `json:"tokens"`
	CostMicros               int64                          `json:"costMicros"`
	Currency                 string                         `json:"currency"`
	UsageStatus              string                         `json:"usageStatus"`
}

func (a *shopDiagnosisApplication) projectRun(ctx context.Context, run agent.Record, frozen agentconfig.Snapshot) (diagnosisRunDTO, error) {
	id, e := a.currentIdentity(ctx)
	if e != nil {
		return diagnosisRunDTO{}, e
	}
	mode, e := a.configuration.LoadModeSnapshot(ctx, id, run.State.Request.ModeSnapshotRef)
	if e != nil {
		return diagnosisRunDTO{}, e
	}
	s := run.State
	if frozen.MemberID != id.MemberID || s.Scope != (agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}) {
		return diagnosisRunDTO{}, agentconfig.ErrForbidden
	}
	out := diagnosisRunDTO{RunID: s.RunID, ContextID: s.Request.Binding.ContextID, RequestKey: s.Request.Key, Phase: s.Phase, Revision: strconv.FormatUint(s.Revision, 10), StopReason: s.StopReason, ConfigurationSnapshotRef: s.Request.ConfigurationSnapshotRef, ModeSnapshotRef: mode.Ref(), ExecutionMode: mode.Mode, ModeRevision: mode.Revision, GoalSummary: s.Request.GoalSummary, Unresolved: append([]string{}, s.Unresolved...), Tokens: s.Usage.Tokens, CostMicros: s.Usage.CostMicros, Currency: s.Request.Limits.Currency, UsageStatus: "observed"}
	if s.Validation != nil {
		out.Unresolved = append(out.Unresolved, s.Validation.Unresolved...)
	}
	if s.PendingInvocationID != "" || s.Phase == agent.Running || s.StopReason == agent.StopUsageUnknown || s.StopReason == agent.StopModelUnknown || s.StopReason == agent.StopExecutionOutcomeUnknown {
		out.UsageStatus = "unknown_reserved"
	}
	if inspection, proposal, valid := shopdiagnosis.ValidatedResult(s); valid {
		out.Inspection, out.Proposal, out.Validated = &inspection, &proposal, true
	} else {
		for _, o := range s.History {
			if o.Tool == agentconfig.DiagnosisInspectToolRef() && o.AuditStatus == commercetool.AuditStatusRecorded && agent.ValidID(o.CallID) && o.InvocationID == "" {
				inspection, e := shopdiagnosis.ParseInspection(o.Output)
				if e == nil && shopdiagnosis.InspectionMatchesBinding(inspection, s.Request.Binding) {
					out.Inspection = &inspection
				}
			}
		}
	}
	return out, nil
}
