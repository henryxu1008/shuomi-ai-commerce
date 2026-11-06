package imageagentapp

import (
	"context"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/asset"
)

type AnalysisSnapshotReader interface {
	LoadSnapshot(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (agentconfig.Snapshot, error)
}
type AnalysisRunReader interface {
	Lookup(context.Context, agent.Scope, agent.Binding, string) (agent.Record, bool, error)
}
type AnalysisInvocationReader interface {
	ReadModelInvocation(context.Context, string, string) (aicapability.InvocationRecord, error)
}

// AnalysisResultValidator checks references/shape, never visual truth or
// approval. It reads original frozen configuration, Run and native ledger
// owners and cannot dispatch, settle, adopt or save business data.
type AnalysisResultValidator struct {
	snapshots   AnalysisSnapshotReader
	runs        AnalysisRunReader
	invocations AnalysisInvocationReader
	sources     *AnalysisSources
}

func NewAnalysisResultValidator(snapshots AnalysisSnapshotReader, runs AnalysisRunReader, invocations AnalysisInvocationReader, sources *AnalysisSources) (*AnalysisResultValidator, error) {
	if snapshots == nil || runs == nil || invocations == nil || sources == nil {
		return nil, agent.ErrUnavailable
	}
	return &AnalysisResultValidator{snapshots: snapshots, runs: runs, invocations: invocations, sources: sources}, nil
}

func (v *AnalysisResultValidator) ValidateDomain(ctx context.Context, r agent.Request, p agent.DomainProposal, history []agent.Observation) (agent.Validation, error) {
	invalid := agent.Validation{PolicyVersion: r.PolicyVersion, Unresolved: []string{"Image analysis result or original execution reference could not be verified."}}
	if v == nil || ctx == nil || ctx.Err() != nil || r.PolicyVersion != agentconfig.ImageAnalysisPolicyVersion || r.PromptVersion != agentconfig.ImageAnalysisPromptVersion ||
		!agentconfig.ImageAnalysisBinding(r.Binding) || !r.ConfigurationSnapshotRef.ValidOrAbsent() || r.ConfigurationSnapshotRef.Absent() || p.Kind != asset.ImageVisualAnalysisSchema {
		return invalid, agent.ErrInvalid
	}
	result, err := asset.DecodeImageVisualAnalysisResult(p.Payload)
	if err != nil {
		return invalid, nil
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || i.TenantID != i.EffectiveOrganizationID || !agent.ValidID(i.EffectiveMemberID) {
		return invalid, agent.ErrUnavailable
	}
	scope := agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}
	s, err := v.snapshots.LoadSnapshot(ctx, scope, r.ConfigurationSnapshotRef)
	if err != nil {
		return invalid, err
	}
	if s.Scope != scope || s.MemberID != i.EffectiveMemberID || s.AgentID != agentconfig.ImageAgentID || s.AgentVersion != agentconfig.ImageAgentVersion || s.ImageAnalysis == nil || s.Template != nil || s.KnowledgeBaseID != "" ||
		s.Request.Binding != r.Binding || s.Request.Key != r.Key || s.Request.ModeSnapshotRef != r.ModeSnapshotRef || s.Request.PolicyVersion != r.PolicyVersion || s.Request.PromptVersion != r.PromptVersion ||
		result.ConfigurationDigest != r.ConfigurationSnapshotRef.Digest || result.ModelID != s.ExecutionModelProfile.ModelID {
		return invalid, nil
	}
	configuredRequest := s.Request
	configuredRequest.ConfigurationSnapshotRef = r.ConfigurationSnapshotRef
	if configuredRequest != r {
		return invalid, nil
	}
	a := *s.ImageAnalysis
	if result.SelectionDigest != a.Image.SelectionDigest || result.ContentSHA256 != a.Image.ContentSHA256 || result.QuoteDigest != a.QuoteDigest {
		return invalid, nil
	}
	prereads := 1
	if a.Kind == "approved" {
		prereads = 2
	}
	if len(history) != prereads+1 {
		return invalid, nil
	}
	last := history[prereads]
	if last.InvocationID != result.InvocationID || last.CallID != result.InvocationID || last.Tool.ID != "" || last.Tool.Version != "" || last.ObservedUsage == nil || !last.ObservedUsage.Known {
		return invalid, nil
	}
	if err = v.sources.CheckImageAnalysis(ctx, agent.ModelInput{Binding: r.Binding, History: history[:prereads]}, a); err != nil {
		return invalid, err
	}
	run, found, err := v.runs.Lookup(ctx, scope, r.Binding, r.Key)
	if err != nil {
		return invalid, err
	}
	if !found || run.State.Scope != scope || run.State.Request != r || !agent.ValidID(run.State.RunID) {
		return invalid, nil
	}
	fact, err := v.invocations.ReadModelInvocation(ctx, scope.OrganizationID, result.InvocationID)
	if err != nil {
		return invalid, err
	}
	if fact.InvocationID != result.InvocationID || fact.AgentRunID != run.State.RunID || fact.TenantID != scope.OrganizationID || fact.UserID != scope.ActorID || fact.MemberID != s.MemberID ||
		fact.Operation != aicapability.OperationProductImageAnalysis || fact.ModelID != result.ModelID || fact.ProviderID != s.ExecutionModelProfile.ProviderID || fact.Currency != s.ExecutionModelProfile.Currency ||
		fact.PromptVersion != s.ExecutionModelProfile.PromptVersion || fact.PromptKey != s.ExecutionModelProfile.OutputSchemaVersion ||
		fact.OutputHash != result.ModelOutputSHA256 || fact.Outcome != aicapability.InvocationSucceeded ||
		!fact.UsageKnown || !fact.EstimatedCostKnown || fact.TotalTokens != fact.PromptTokens+fact.CompletionTokens || fact.PromptTokens < 0 || fact.CompletionTokens < 0 ||
		last.ObservedUsage.Tokens != int64(fact.TotalTokens) || last.ObservedUsage.CostMicros != fact.EstimatedCostMicros || last.ObservedUsage.Currency != fact.Currency ||
		fact.OfficialTariff != s.ExecutionModelProfile.OfficialTariff {
		return invalid, nil
	}
	return agent.Validation{Valid: true, PolicyVersion: r.PolicyVersion, Unresolved: []string{}}, nil
}

var _ agent.DomainValidator = (*AnalysisResultValidator)(nil)
