package httpapi

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"task-processor/internal/product/research/analysis"
	"time"
)

type researchInput struct {
	ProjectID  string           `json:"-"`
	Candidates []researchChoice `json:"candidates"`
}
type researchChoice struct {
	CandidateID      string          `json:"candidateId"`
	ExpectedRevision string          `json:"expectedRevision"`
	LinkID           string          `json:"linkId"`
	EvidenceDigest   string          `json:"evidenceDigest"`
	Selection        selectionChoice `json:"selection"`
}

func (in researchInput) inputs() ([]research.AnalysisChoice, error) {
	if !agentconfig.UUID(in.ProjectID) || len(in.Candidates) < 2 || len(in.Candidates) > 5 {
		return nil, agent.ErrInvalid
	}
	choices := make([]research.AnalysisChoice, 0, len(in.Candidates))
	seen := map[string]bool{}
	for _, c := range in.Candidates {
		revision, e := strconv.ParseInt(c.ExpectedRevision, 10, 64)
		p, pe := c.Selection.input()
		if e != nil || revision < 1 || strconv.FormatInt(revision, 10) != c.ExpectedRevision || pe != nil || !agentconfig.UUID(c.CandidateID) || !agentconfig.UUID(c.LinkID) || len(c.EvidenceDigest) != 64 || seen[c.CandidateID] {
			return nil, agent.ErrInvalid
		}
		seen[c.CandidateID] = true
		choices = append(choices, research.AnalysisChoice{CandidateID: c.CandidateID, ExpectedRevision: revision, LinkID: c.LinkID, EvidenceDigest: c.EvidenceDigest, Selection: research.Selection{ItemID: p.ItemID, Revision: p.ExpectedRevision, PublicationID: p.OriginalPublicationID, Version: p.OriginalVersion}})
	}
	sort.Slice(choices, func(a, b int) bool { return choices[a].Selection.ItemID < choices[b].Selection.ItemID })
	for n, c := range choices {
		if n > 0 && c.Selection.ItemID == choices[n-1].Selection.ItemID {
			return nil, agent.ErrInvalid
		}
	}
	return choices, nil
}

type researchPrepareInput struct {
	researchInput
	RequestKey           string `json:"requestKey"`
	ExpectedModeRevision string `json:"expectedModeRevision"`
	GoalSummary          string `json:"goalSummary"`
}

func (a *marketResearchApplication) inspectInput(ctx context.Context, in researchInput) (agent.Binding, analysis.Inspection, error) {
	choices, e := in.inputs()
	if e != nil {
		return agent.Binding{}, analysis.Inspection{}, e
	}
	id, e := a.readIdentity(ctx)
	if e != nil {
		return agent.Binding{}, analysis.Inspection{}, e
	}
	return a.inspector.Prepare(authidentity.WithAuthenticatedIdentity(ctx, id), in.ProjectID, choices)
}
func researchIntentMatches(s agentconfig.Snapshot, in researchPrepareInput) bool {
	choices, e := in.researchInput.inputs()
	if e != nil || s.Request.Key != in.RequestKey || s.Request.Binding.ContextID != in.ProjectID || s.Request.GoalSummary != in.GoalSummary || s.ExpectedModeRevision != in.ExpectedModeRevision {
		return false
	}
	b := s.Request.Binding
	for n, c := range choices {
		p, r := b.Selection[n], b.ResearchSelection.Candidates[n]
		if p.ItemID != c.Selection.ItemID || p.ItemRevision != strconv.FormatInt(c.Selection.Revision, 10) || p.PublicationID != c.Selection.PublicationID || p.CatalogVersion != strconv.FormatUint(c.Selection.Version, 10) || r.CandidateID != c.CandidateID || r.CandidateRevision != strconv.FormatInt(c.ExpectedRevision, 10) || r.SelectedLinkID != c.LinkID || r.EvidenceDigest != c.EvidenceDigest {
			return false
		}
	}
	return len(choices) == 5 || b.Selection[len(choices)] == (agent.SelectionProductRef{})
}

// Receipt recovery never selects a current model or regenerates an inspection.
func (a *marketResearchApplication) readPreparation(ctx context.Context, project, key string) (agentconfig.Snapshot, error) {
	id, e := a.currentIdentity(ctx)
	if e != nil {
		return agentconfig.Snapshot{}, e
	}
	if a.configuration == nil {
		return agentconfig.Snapshot{}, agent.ErrUnavailable
	}
	s, found, e := a.configuration.LoadResearchPreparationByKey(ctx, agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, id.MemberID, project, key)
	if e != nil {
		return agentconfig.Snapshot{}, e
	}
	if !found {
		return agentconfig.Snapshot{}, agentconfig.ErrNotFound
	}
	if _, e = a.inspector.AuthorizeHistory(ctx, s.Request.Binding); e != nil {
		return agentconfig.Snapshot{}, researchHistoryError(e)
	}
	return s, nil
}
func researchHistoryError(e error) error {
	// A lost source is hidden history, never proof that the operation is absent.
	if errors.Is(e, research.ErrNotFound) || errors.Is(e, collection.ErrNotFound) || errors.Is(e, research.ErrForbidden) {
		return agentconfig.ErrForbidden
	}
	return e
}
func (a *marketResearchApplication) prepare(ctx context.Context, in researchPrepareInput) (agentconfig.Snapshot, error) {
	if !agentconfig.UUID(in.RequestKey) || !agent.ValidGoalSummary(in.GoalSummary) || in.GoalSummary == "" {
		return agentconfig.Snapshot{}, agent.ErrInvalid
	}
	if _, e := in.researchInput.inputs(); e != nil {
		return agentconfig.Snapshot{}, e
	}
	if _, e := agentconfig.ParseModeRevision(in.ExpectedModeRevision); e != nil {
		return agentconfig.Snapshot{}, e
	}
	id, e := a.currentIdentity(ctx)
	if e != nil {
		return agentconfig.Snapshot{}, e
	}
	if a.configuration == nil {
		return agentconfig.Snapshot{}, agent.ErrUnavailable
	}
	prior, found, e := a.configuration.LoadResearchPreparationByKey(ctx, agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, id.MemberID, in.ProjectID, in.RequestKey)
	if e != nil {
		return agentconfig.Snapshot{}, e
	}
	if found {
		if !researchIntentMatches(prior, in) {
			return agentconfig.Snapshot{}, agent.ErrConflict
		}
		if _, e = a.inspector.AuthorizeHistory(ctx, prior.Request.Binding); e != nil {
			return agentconfig.Snapshot{}, researchHistoryError(e)
		}
		return prior, nil
	}
	binding, _, e := a.inspectInput(ctx, in.researchInput)
	if e != nil {
		return agentconfig.Snapshot{}, e
	}
	execution, e := a.currentExecutionIdentity(ctx)
	if e != nil || execution != id {
		return agentconfig.Snapshot{}, agentconfig.ErrForbidden
	}
	if a.runtime == nil || a.model == nil || a.profile == nil || a.parent == nil {
		return agentconfig.Snapshot{}, agent.ErrUnavailable
	}
	profile, e := a.profile(ctx, id.OrganizationID)
	if e != nil {
		return agentconfig.Snapshot{}, agent.ErrUnavailable
	}
	limits := a.parent.config.Limits
	limits.ModelCalls = 2
	request := agent.Request{Key: in.RequestKey, Binding: binding, GoalSummary: in.GoalSummary, PolicyVersion: agentconfig.ResearchPolicyVersion, PromptVersion: agentconfig.ResearchPromptVersion, Limits: limits}
	return a.configuration.Prepare(ctx, agentconfig.StartCommand{Scope: agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, MemberID: id.MemberID, ExpectedModeRevision: in.ExpectedModeRevision, AgentID: agentconfig.ResearchAgentID, AgentVersion: agentconfig.ResearchAgentVersion, Request: request, ExecutionModelProfile: profile})
}
func (a *marketResearchApplication) recent(ctx context.Context, project, cursor string) (any, string, error) {
	id, e := a.currentIdentity(ctx)
	if e != nil {
		return nil, "", e
	}
	if a.configuration == nil || a.store == nil || a.research == nil {
		return nil, "", agent.ErrUnavailable
	}
	scope := collection.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID, MemberID: id.MemberID}
	if e = a.research.AuthorizeAnalysisProject(ctx, scope, project, false); e != nil {
		return nil, "", e
	}
	return a.configuration.Recent(ctx, agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, id.MemberID, agentconfig.ResearchAgentID, cursor, 10, a.store, func(b agent.Binding) error {
		_, e := a.inspector.AuthorizeHistory(ctx, b)
		return recentRunAccessError(researchHistoryError(e))
	}, configstore.RecentContext{Kind: agent.ResearchSelectionContextKind, ID: project})
}

func (a *marketResearchApplication) readRun(ctx context.Context, contextID, key string) (agent.Record, agentconfig.Snapshot, error) {
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
	run, found, e := a.store.LookupContext(ctx, scope, agent.ResearchSelectionContextKind, contextID, key)
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
	if frozen.Scope != scope || frozen.MemberID != id.MemberID || frozen.AgentID != agentconfig.ResearchAgentID || frozen.AgentVersion != agentconfig.ResearchAgentVersion || frozen.Request != expected || expected.Binding.ContextID != contextID || expected.Key != key {
		return run, frozen, agent.ErrConflict
	}
	current, e := a.inspector.AuthorizeHistory(ctx, expected.Binding)
	if e == nil && current != (collection.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID, MemberID: id.MemberID}) {
		e = agentconfig.ErrForbidden
	}
	if e != nil {
		return run, frozen, researchHistoryError(e)
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
func (a *marketResearchApplication) start(ctx context.Context, contextID, key string, ref agent.ConfigurationSnapshotRef) (agent.Record, bool, error) {
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
	if frozen.AgentID != agentconfig.ResearchAgentID || frozen.AgentVersion != agentconfig.ResearchAgentVersion || frozen.MemberID != id.MemberID || frozen.Request.Key != key || frozen.Request.Binding.ContextID != contextID {
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
	ctx = context.WithValue(ctx, researchExecutionContextKey{}, request)
	return a.runtime.StartWithClaimAttempt(ctx, request)
}
func (a *marketResearchApplication) resume(ctx context.Context, contextID, key string, ref agent.ConfigurationSnapshotRef, revision uint64, feedback string) (agent.Record, bool, error) {
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
	ctx = context.WithValue(ctx, researchExecutionContextKey{}, run.State.Request)
	return a.runtime.ResumeWithClaimAttempt(ctx, run.State.Request, revision, feedback)
}

type researchRunDTO struct {
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
	Inspection               *analysis.Inspection           `json:"inspection,omitempty"`
	Proposal                 *analysis.Proposal             `json:"proposal,omitempty"`
	Validated                bool                           `json:"validated"`
	Unresolved               []string                       `json:"unresolved"`
	Tokens                   int64                          `json:"tokens"`
	CostMicros               int64                          `json:"costMicros"`
	Currency                 string                         `json:"currency"`
	UsageStatus              string                         `json:"usageStatus"`
	ReferencePriceVersion    string                         `json:"referencePriceVersion,omitempty"`
}

func (a *marketResearchApplication) projectRun(ctx context.Context, run agent.Record, frozen agentconfig.Snapshot) (researchRunDTO, error) {
	id, e := a.currentIdentity(ctx)
	if e != nil {
		return researchRunDTO{}, e
	}
	mode, e := a.configuration.LoadModeSnapshot(ctx, id, run.State.Request.ModeSnapshotRef)
	if e != nil {
		return researchRunDTO{}, e
	}
	s := run.State
	if frozen.MemberID != id.MemberID || s.Scope != (agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}) {
		return researchRunDTO{}, agentconfig.ErrForbidden
	}
	out := researchRunDTO{RunID: s.RunID, ContextID: s.Request.Binding.ContextID, RequestKey: s.Request.Key, Phase: s.Phase, Revision: strconv.FormatUint(s.Revision, 10), StopReason: s.StopReason, ConfigurationSnapshotRef: s.Request.ConfigurationSnapshotRef, ModeSnapshotRef: mode.Ref(), ExecutionMode: mode.Mode, ModeRevision: mode.Revision, GoalSummary: s.Request.GoalSummary, Unresolved: append([]string{}, s.Unresolved...), Tokens: s.Usage.Tokens, CostMicros: s.Usage.CostMicros, Currency: s.Request.Limits.Currency, UsageStatus: "observed"}
	out.ReferencePriceVersion = frozen.ExecutionModelProfile.OfficialTariff.ReferencePriceVersion()
	if s.Validation != nil {
		out.Unresolved = append(out.Unresolved, s.Validation.Unresolved...)
	}
	if s.PendingInvocationID != "" || s.Phase == agent.Running || s.StopReason == agent.StopUsageUnknown || s.StopReason == agent.StopModelUnknown || s.StopReason == agent.StopExecutionOutcomeUnknown {
		out.UsageStatus = "unknown_reserved"
	}
	if inspection, proposal, valid := analysis.ValidatedResult(s); valid {
		out.Inspection, out.Proposal, out.Validated = &inspection, &proposal, true
	} else {
		for _, o := range s.History {
			if o.Tool == agentconfig.ResearchInspectToolRef() && o.AuditStatus == commercetool.AuditStatusRecorded && agent.ValidID(o.CallID) && o.InvocationID == "" {
				inspection, e := analysis.ParseInspection(o.Output)
				if e == nil && analysis.InspectionMatchesBinding(inspection, s.Request.Binding) {
					out.Inspection = &inspection
				}
			}
		}
	}
	return out, nil
}
