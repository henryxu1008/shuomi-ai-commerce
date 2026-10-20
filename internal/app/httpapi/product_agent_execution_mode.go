package httpapi

import (
	"context"
	"errors"
	"reflect"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/enrichment"
	"task-processor/internal/product/review"
)

func (a *productAgentApplication) modeSubject() agentconfig.ResolvedModeSubject {
	return agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: a.definition.ID}, Definition: commercetool.AgentRef{ID: a.definition.ID, Version: a.definition.Version}}
}

func (a *productAgentApplication) checkManualReviewMode(ctx context.Context, identity agent.ExecutionIdentity, request agent.Request) error {
	if !request.ManualReviewOnly {
		return nil
	}
	if a.configuration == nil {
		return agent.ErrUnavailable
	}
	mode, err := a.configuration.LoadModeSnapshot(ctx, identity, request.ModeSnapshotRef)
	if err != nil {
		return err
	}
	if !request.AllowsManualReviewMode(mode.Mode) {
		return agent.ErrManualReviewRequired
	}
	return nil
}

func agentExecutionProvenance(state agent.State) *review.ExecutionProvenance {
	return &review.ExecutionProvenance{SkillVersionRef: state.Request.SkillVersionRef, ContentSelection: state.Request.ContentSelection, Scope: state.Scope, RunID: state.RunID, RequestKey: state.Request.Key, Binding: state.Request.Binding, Configuration: state.Request.ConfigurationSnapshotRef, Mode: state.Request.ModeSnapshotRef}
}
func (a *productAgentApplication) titlePolicyScope(ctx context.Context) (review.CandidateExecutionScope, error) {
	identity, err := a.currentModeIdentity(ctx)
	return review.CandidateExecutionScope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID, MemberID: identity.MemberID}, err
}
func (a *productAgentApplication) resolveTitlePolicy(ctx context.Context, provenance review.ExecutionProvenance, candidate enrichment.Candidate, human bool) (review.PolicyProvenance, error) {
	var proof review.PolicyProvenance
	if !provenance.Valid() || a.configuration == nil || a.store == nil {
		return proof, review.ErrForbidden
	}
	identity, err := a.currentModeIdentity(ctx)
	if err != nil {
		return proof, err
	}
	if !human {
		var roles []string
		if supplyAgentContext(ctx) {
			current, e := a.supplyExecution.current(ctx)
			if e != nil {
				return proof, e
			}
			roles = current.roles
		} else {
			current, e := a.freshIdentity(ctx)
			if e != nil {
				return proof, e
			}
			roles = current.Roles
		}
		if !a.authorizer.IsTenantAdmin("", roles) || !authz.AllowedOrganization(ctx, a.authorizer, identity.ActorID, identity.OrganizationID, roles, authz.PermissionListingKitAdminWrite) {
			return proof, review.ErrForbidden
		}
	}
	scope := provenance.Scope
	if identity.OrganizationID != scope.OrganizationID || !human && identity.ActorID != scope.ActorID {
		return proof, review.ErrForbidden
	}
	config, err := a.configuration.LoadSnapshot(ctx, scope, provenance.Configuration)
	if err != nil {
		return proof, err
	}
	consumer := a
	actionVersion := review.TitlePolicyAdoptionVersion
	content := config.AgentID == agentconfig.ContentAgentID
	if content {
		if config.AgentVersion != agentconfig.ContentAgentVersion || config.Request.PolicyVersion != enrichment.ContentPolicyVersion || config.Request.ContentSelection != provenance.ContentSelection || !provenance.ContentSelection.Valid() {
			return proof, review.ErrConflict
		}
		actionVersion = review.ContentPolicyAdoptionVersion
		if a.definition.ID != agentconfig.ContentAgentID {
			consumer = a.content
		}
		if consumer == nil {
			return proof, agentconfig.ErrUnavailable
		}
	} else if config.AgentID != "product.title.agent" || !provenance.ContentSelection.Absent() {
		return proof, review.ErrForbidden
	}
	origin := agent.ExecutionIdentity{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: config.MemberID}
	// Human review by another authorized administrator is allowed. The original
	// actor must still have the exact membership frozen by this execution.
	if (!human || identity.ActorID == origin.ActorID) && identity != origin {
		return proof, review.ErrForbidden
	}
	if human && identity.ActorID != origin.ActorID {
		// The reviewer's bearer token proves the reviewer, never the origin.
		// Reuse the server's exact live IAM lookup for the durable origin subject.
		if a.originExecution == nil {
			return proof, review.ErrUnavailable
		}
		if err = a.originExecution(ctx, origin); err != nil {
			return proof, err
		}
	}
	run, found, err := a.store.Lookup(ctx, scope, provenance.Binding, provenance.RequestKey)
	if err != nil {
		return proof, err
	}
	if !found || !agentRunReviewable(run.State) || run.State.RunID != provenance.RunID || *agentExecutionProvenance(run.State) != provenance || !reflect.DeepEqual(candidate.Changes, run.State.Candidate.Changes) {
		return proof, review.ErrConflict
	}
	if !human {
		if _, err = consumer.Authorize(ctx, provenance.Binding); err != nil {
			return proof, err
		}
	}
	mode, err := a.configuration.CheckRegisteredTextExecutionMode(ctx, origin, provenance.Configuration, provenance.Mode)
	if err != nil {
		if human && (errors.Is(err, agentconfig.ErrChanged) || errors.Is(err, agentconfig.ErrNotEnabled) || errors.Is(err, agentconfig.ErrConflict)) {
			return proof, review.ErrConflict
		}
		return proof, err
	}
	proof = review.PolicyProvenance{Execution: provenance, Identity: origin, Mode: mode.Mode, ModeRevision: mode.Revision, PolicyVersion: mode.PolicyVersion, ActionVersion: actionVersion, RequiresHuman: len(run.State.Unresolved) > 0 || content && !provenance.ContentSelection.TitleOnly()}
	return proof, nil
}

func (a *productAgentApplication) finishTitleCandidate(ctx context.Context, run agent.Record) (*review.View, error) {
	if !agentRunReviewable(run.State) {
		return nil, nil
	}
	identity, err := a.currentModeIdentity(ctx)
	if err != nil {
		return nil, err
	}
	mode, err := a.configuration.LoadModeSnapshot(ctx, identity, run.State.Request.ModeSnapshotRef)
	if err != nil {
		return nil, err
	}
	if mode.Mode != agent.ModeAutoApprove && mode.Mode != agent.ModeFullAccess {
		return nil, nil
	}
	if run.State.Request.ManualReviewOnly {
		return nil, agent.ErrManualReviewRequired
	}
	input := agentReviewInput(run.State.Request.Binding, run.State.Request.PolicyVersion, run.State.Candidate)
	input.ContentSelection = run.State.Request.ContentSelection
	input.Execution = agentExecutionProvenance(run.State)
	input.ContextProvenance, err = agentContextProvenance(run.State)
	if err != nil {
		return nil, err
	}
	var view review.View
	policy := a.policyAdoption
	if supplyAgentContext(ctx) {
		view, err = a.executionReviews.CreateFromCandidate(ctx, "agent:"+run.State.RunID, input)
		policy = a.executionPolicyAdoption
	} else {
		view, err = a.reviews.CreateFromCandidate(ctx, "agent:"+run.State.RunID, input)
	}
	if err != nil {
		return nil, err
	}
	if view.State == "applied" {
		return &view, nil
	}
	// Human edits or decisions retain the existing human lifecycle.
	if view.State != "pending" || len(view.History) > 0 || view.Content != nil && len(view.Content.Decisions) > 0 {
		return &view, nil
	}
	if policy == nil {
		return nil, agentconfig.ErrUnavailable
	}
	view, _, err = policy.Adopt(ctx, "policy:"+run.State.RunID, review.PolicyAdoptionCommand{ProposalID: view.ID, ExpectedRevision: view.Revision, Execution: *input.Execution})
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (a *productAgentApplication) currentModeIdentity(ctx context.Context) (agent.ExecutionIdentity, error) {
	if supplyAgentContext(ctx) {
		return a.resolveExecutionIdentity(ctx)
	}
	current, err := a.freshIdentity(ctx)
	if err != nil {
		return agent.ExecutionIdentity{}, err
	}
	identity := agent.ExecutionIdentity{OrganizationID: current.EffectiveOrganizationID, ActorID: current.UserID, MemberID: current.EffectiveMemberID}
	if !identity.Valid() || current.TenantID != identity.OrganizationID {
		return agent.ExecutionIdentity{}, agentconfig.ErrForbidden
	}
	return identity, nil
}
