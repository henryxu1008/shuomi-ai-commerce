package shopdiagnosis

import (
	"context"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
)

type InspectionReader interface {
	Inspect(context.Context, agent.Binding) (Inspection, error)
}
type ValidationAuthority interface {
	AuthorizeHistory(context.Context, agent.Binding) (agent.ExecutionIdentity, error)
}
type ConfigurationReader interface {
	LoadSnapshot(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (agentconfig.Snapshot, error)
	CheckRegisteredTextExecutionMode(context.Context, agent.ExecutionIdentity, agent.ConfigurationSnapshotRef, agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error)
}
type Validator struct {
	Reader        ValidationAuthority
	Configuration ConfigurationReader
}

func (v Validator) ValidateDomain(ctx context.Context, request agent.Request, candidate agent.DomainProposal, history []agent.Observation) (agent.Validation, error) {
	invalid := agent.Validation{PolicyVersion: agentconfig.DiagnosisPolicyVersion, Unresolved: []string{"DIAGNOSIS_EVIDENCE_INVALID"}}
	if ctx == nil || v.Reader == nil || v.Configuration == nil || request.PolicyVersion != agentconfig.DiagnosisPolicyVersion || request.PromptVersion != agentconfig.DiagnosisPromptVersion || candidate.Kind != agentconfig.DiagnosisProposalKind || !request.Binding.Valid() || request.Binding.ContextKind != agent.StoreDiagnosisContextKind {
		return invalid, nil
	}
	proposal, err := ParseProposal(candidate.Payload)
	if err != nil {
		return invalid, nil
	}
	identity, err := v.Reader.AuthorizeHistory(ctx, request.Binding)
	if err != nil {
		return agent.Validation{}, err
	}
	scope := agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}
	frozen, err := v.Configuration.LoadSnapshot(ctx, scope, request.ConfigurationSnapshotRef)
	if err != nil {
		return agent.Validation{}, err
	}
	expected := request
	expected.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{}
	if frozen.Scope != scope || frozen.MemberID != identity.MemberID || frozen.AgentID != agentconfig.DiagnosisAgentID || frozen.AgentVersion != agentconfig.DiagnosisAgentVersion || frozen.Request != expected || frozen.Template != nil || frozen.KnowledgeBaseID != "" {
		return agent.Validation{}, agent.ErrConflict
	}
	if _, err = v.Configuration.CheckRegisteredTextExecutionMode(ctx, identity, request.ConfigurationSnapshotRef, request.ModeSnapshotRef); err != nil {
		return agent.Validation{}, err
	}
	disclosed := false
	for _, observation := range history {
		if observation.Tool != agentconfig.DiagnosisInspectToolRef() || observation.AuditStatus != commercetool.AuditStatusRecorded || !agent.ValidID(observation.CallID) || observation.InvocationID != "" {
			continue
		}
		out, err := ParseInspection(observation.Output)
		if err == nil && InspectionMatchesBinding(out, request.Binding) && ValidateProposal(out, proposal) == nil {
			disclosed = true
			break
		}
	}
	if !disclosed {
		return invalid, nil
	}
	return agent.Validation{Valid: true, PolicyVersion: agentconfig.DiagnosisPolicyVersion}, nil
}

// ValidatedResult distinguishes actual validated suggestions from a repair-
// exhausted diagnostic that shares the original runtime's review phase.
func ValidatedResult(state agent.State) (Inspection, Proposal, bool) {
	if state.Phase != agent.HumanReviewRequired || state.StopReason != "" || state.RejectedPayload != nil || state.Request.Binding.ContextKind != agent.StoreDiagnosisContextKind || state.Request.PolicyVersion != agentconfig.DiagnosisPolicyVersion || state.Request.PromptVersion != agentconfig.DiagnosisPromptVersion || state.Validation == nil || !state.Validation.Valid || state.Validation.PolicyVersion != agentconfig.DiagnosisPolicyVersion || state.DomainProposal == nil || state.DomainProposal.Kind != agentconfig.DiagnosisProposalKind || state.Validation.CandidateHash != digest(state.DomainProposal) {
		return Inspection{}, Proposal{}, false
	}
	proposal, err := ParseProposal(state.DomainProposal.Payload)
	if err != nil {
		return Inspection{}, Proposal{}, false
	}
	for _, observation := range state.History {
		if observation.Tool != agentconfig.DiagnosisInspectToolRef() || observation.AuditStatus != commercetool.AuditStatusRecorded || !agent.ValidID(observation.CallID) || observation.InvocationID != "" {
			continue
		}
		out, err := ParseInspection(observation.Output)
		if err == nil && InspectionMatchesBinding(out, state.Request.Binding) && ValidateProposal(out, proposal) == nil {
			return out, proposal, true
		}
	}
	return Inspection{}, Proposal{}, false
}
func InspectionMatchesBinding(out Inspection, b agent.Binding) bool {
	return b.Valid() && b.ContextKind == agent.StoreDiagnosisContextKind && out.Ref == b.StoreDiagnosis
}

var _ agent.DomainValidator = Validator{}
