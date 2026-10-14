package selection

import (
	"context"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/collection"
)

type InspectionReader interface {
	Inspect(context.Context, agent.Binding) (Inspection, error)
}
type ValidationAuthority interface {
	AuthorizeHistory(context.Context, agent.Binding) (collection.Scope, error)
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
	invalid := agent.Validation{PolicyVersion: agentconfig.SelectionPolicyVersion, Unresolved: []string{"SELECTION_EVIDENCE_INVALID"}}
	if ctx == nil || v.Reader == nil || v.Configuration == nil || request.PolicyVersion != agentconfig.SelectionPolicyVersion || request.PromptVersion != agentconfig.SelectionPromptVersion || candidate.Kind != agentconfig.SelectionProposalKind || !request.Binding.Valid() || request.Binding.ContextKind != agent.SelectionContextKind {
		return invalid, nil
	}
	proposal, err := ParseProposal(candidate.Payload)
	if err != nil {
		return invalid, nil
	}
	current, err := v.Reader.AuthorizeHistory(ctx, request.Binding)
	if err != nil {
		return agent.Validation{}, err
	}
	identity := agent.ExecutionIdentity{OrganizationID: current.OrganizationID, ActorID: current.ActorID, MemberID: current.MemberID}
	scope := agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}
	frozen, err := v.Configuration.LoadSnapshot(ctx, scope, request.ConfigurationSnapshotRef)
	if err != nil {
		return agent.Validation{}, err
	}
	expected := request
	expected.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{}
	if frozen.Scope != scope || frozen.MemberID != identity.MemberID || frozen.AgentID != agentconfig.SelectionAgentID || frozen.AgentVersion != agentconfig.SelectionAgentVersion || frozen.Request != expected || frozen.Template != nil || frozen.KnowledgeBaseID != "" {
		return agent.Validation{}, agent.ErrConflict
	}
	if _, err = v.Configuration.CheckRegisteredTextExecutionMode(ctx, identity, request.ConfigurationSnapshotRef, request.ModeSnapshotRef); err != nil {
		return agent.Validation{}, err
	}
	disclosed := false
	for _, observation := range history {
		if observation.Tool != agentconfig.SelectionInspectToolRef() || observation.AuditStatus != commercetool.AuditStatusRecorded || !agent.ValidID(observation.CallID) || observation.InvocationID != "" {
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
	return agent.Validation{Valid: true, PolicyVersion: agentconfig.SelectionPolicyVersion}, nil
}

// ValidatedResult distinguishes actual validated suggestions from a repair-
// exhausted diagnostic that shares the original runtime's review phase.
func ValidatedResult(state agent.State) (Inspection, Proposal, bool) {
	if state.Phase != agent.HumanReviewRequired || state.StopReason != "" || state.RejectedPayload != nil || state.Request.Binding.ContextKind != agent.SelectionContextKind || state.Request.PolicyVersion != agentconfig.SelectionPolicyVersion || state.Request.PromptVersion != agentconfig.SelectionPromptVersion || state.Validation == nil || !state.Validation.Valid || state.Validation.PolicyVersion != agentconfig.SelectionPolicyVersion || state.DomainProposal == nil || state.DomainProposal.Kind != agentconfig.SelectionProposalKind || state.Validation.CandidateHash != collection.Digest(state.DomainProposal) {
		return Inspection{}, Proposal{}, false
	}
	proposal, err := ParseProposal(state.DomainProposal.Payload)
	if err != nil {
		return Inspection{}, Proposal{}, false
	}
	for _, observation := range state.History {
		if observation.Tool != agentconfig.SelectionInspectToolRef() || observation.AuditStatus != commercetool.AuditStatusRecorded || !agent.ValidID(observation.CallID) || observation.InvocationID != "" {
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
	if !b.Valid() || b.ContextKind != agent.SelectionContextKind {
		return false
	}
	if len(out.Products) < 2 || len(out.Products) > agent.MaxSelectionProducts {
		return false
	}
	var refs [agent.MaxSelectionProducts]agent.SelectionProductRef
	for n, p := range out.Products {
		refs[n] = p.Ref
	}
	return refs == b.Selection
}

var _ agent.DomainValidator = Validator{}
