package contentagent

import (
	"context"
	"errors"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/product/enrichment"
	"task-processor/internal/product/review"
)

type ConfigurationReader interface {
	LoadSnapshot(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (agentconfig.Snapshot, error)
	CheckRegisteredTextExecutionMode(context.Context, agent.ExecutionIdentity, agent.ConfigurationSnapshotRef, agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error)
}

type Validator struct {
	Authorization agent.Authorizer
	Configuration ConfigurationReader
	Identity      func(context.Context) (agent.ExecutionIdentity, error)
	Review        func(context.Context, review.ContentValidationInput) (enrichment.Proposal, error)
	Observed      func(agent.Request, enrichment.Candidate, []agent.Observation) bool
}

func (v Validator) Validate(ctx context.Context, request agent.Request, candidate enrichment.Candidate, history []agent.Observation) (agent.Validation, error) {
	result := agent.Validation{PolicyVersion: enrichment.ContentPolicyVersion}
	if ctx == nil || !request.ContentSelection.Valid() || request.PolicyVersion != enrichment.ContentPolicyVersion || request.PromptVersion != agentconfig.ContentPromptVersion || !request.Binding.Valid() || request.ConfigurationSnapshotRef.Absent() || !request.ConfigurationSnapshotRef.ValidOrAbsent() || !request.ModeSnapshotRef.Valid() || !agent.ValidGoalSummary(request.GoalSummary) {
		return result, agent.ErrInvalid
	}
	if v.Authorization == nil || v.Configuration == nil || v.Identity == nil || v.Review == nil || v.Observed == nil {
		return result, agent.ErrUnavailable
	}
	scope, err := v.Authorization.Authorize(ctx, request.Binding)
	if err != nil {
		return result, err
	}
	identity, err := v.Identity(ctx)
	if err != nil || !identity.Valid() || scope != (agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}) {
		return result, agentconfig.ErrForbidden
	}
	snapshot, err := v.Configuration.LoadSnapshot(ctx, scope, request.ConfigurationSnapshotRef)
	if err != nil {
		return result, err
	}
	frozen := snapshot.Request
	if snapshot.Scope != scope || snapshot.MemberID != identity.MemberID || snapshot.AgentID != agentconfig.ContentAgentID || snapshot.AgentVersion != agentconfig.ContentAgentVersion || frozen.Key != request.Key || frozen.Binding != request.Binding || frozen.PolicyVersion != request.PolicyVersion || frozen.PromptVersion != request.PromptVersion || frozen.ContentSelection != request.ContentSelection || frozen.ModeSnapshotRef != request.ModeSnapshotRef || frozen.GoalSummary != request.GoalSummary || frozen.Limits != request.Limits {
		return result, agent.ErrConflict
	}
	if _, err := v.Configuration.CheckRegisteredTextExecutionMode(ctx, identity, request.ConfigurationSnapshotRef, request.ModeSnapshotRef); err != nil {
		return result, err
	}
	if !v.Observed(request, candidate, history) {
		result.Unresolved = []string{"candidate_evidence_not_observed"}
		return result, nil
	}
	version, err := strconv.ParseUint(request.Binding.CatalogVersion, 10, 63)
	if err != nil || version == 0 || strconv.FormatUint(version, 10) != request.Binding.CatalogVersion {
		return result, agent.ErrInvalid
	}
	proposal, err := v.Review(ctx, review.ContentValidationInput{Selection: request.ContentSelection, Base: review.CreateInput{ProductKey: request.Binding.ProductKey, BaseVersion: version}, PublicationID: request.Binding.PublicationID, Candidate: candidate})
	if err != nil {
		if errors.Is(err, enrichment.ErrEvidenceInsufficient) || errors.Is(err, enrichment.ErrOutputValidation) || errors.Is(err, enrichment.ErrPolicyRejected) || errors.Is(err, review.ErrInvalid) {
			result.Unresolved = []string{"content_or_evidence_invalid"}
			return result, nil
		}
		return result, err
	}
	result.Valid = proposal.Validation.Valid
	return result, nil
}

var _ agent.Validator = Validator{}
