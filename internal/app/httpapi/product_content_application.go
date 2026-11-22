package httpapi

import (
	"context"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/commercetool"
	einoruntime "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/agent/eino"
	texteino "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/agent/einomodel"
	governed "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/einomodel"
	configstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/agentconfig"
	skillstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/skill"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/skillsandbox"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog/tools/canonicalinspect"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/contentagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/review"
	reviewproposal "github.com/henryxu1008/shuomi-ai-commerce/internal/product/review/proposal"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/sourcing/tools/sourceevidenceinspect"
)

// Only application objects are distinct. Durable runs, configuration, Review,
// Catalog and the invocation/point owners are the current injected instances.
func buildContentProductApplication(ctx context.Context, parent *productAgentApplication, reader catalog.VersionedSnapshotReader, source review.SourcePublicationGateway, repository review.Store, deps commercetool.InvocationDependencies, contexts []texteino.KnowledgeContext) (*productAgentApplication, error) {
	if len(parent.config.ContentTextPolicies) == 0 || configstore.VerifyContentConfigurationSchema(ctx, parent.config.RunDB) != nil {
		return nil, nil
	}
	copy := *parent
	a := &copy
	a.content = nil
	if skillstore.VerifySchema(ctx, parent.config.RunDB) == nil {
		var err error
		a.skills, err = skillstore.New(parent.config.RunDB)
		if err != nil {
			return nil, err
		}
	}
	definition, definitions := contentagent.Registration()
	a.definition = definition
	principal, ok := deps.PrincipalResolver.(productAgentPrincipal)
	if !ok {
		return nil, agent.ErrUnavailable
	}
	deps.PrincipalResolver = productAgentPrincipal{application: a, request: principal.request}
	a.scriptDependencies = deps
	if a.config.ScriptSandbox != nil {
		worker, err := skillsandbox.New(*a.config.ScriptSandbox)
		if err != nil {
			return nil, err
		}
		a.script, a.scriptDescriptor = worker, worker.Descriptor()
	}
	var err error
	a.canonical, err = canonicalinspect.NewInvoker(reader, definition, deps)
	if err != nil {
		return nil, err
	}
	a.sources, err = sourceevidenceinspect.NewInvoker(reader, productAgentSourceGateway{application: a, request: source}, definition, deps)
	if err != nil {
		return nil, err
	}
	a.assets, a.readiness = nil, nil
	policies := make(map[governed.RouteKey]governed.RoutePolicy, len(a.config.ContentTextPolicies))
	for org, policy := range a.config.ContentTextPolicies {
		policies[governed.RouteKey{OrganizationID: org, Operation: aicapability.OperationProductContentDecision}] = policy
	}
	routes, err := newProductTextRouteResolver(a.config, policies)
	if err != nil {
		return nil, err
	}
	a.titleRoutes = routes
	a.selectTitleProfile = func(ctx context.Context, org string) (aicapability.ModelProfile, error) {
		route, err := routes.Resolve(ctx, aicapability.TextInputIdentity{OrganizationID: org, Operation: aicapability.OperationProductContentDecision})
		return route.Profile, err
	}
	executor := &governed.Executor{Ledger: a.config.Ledger, AuthorizeInvocation: a.AuthorizeModelInvocation, Logger: a.config.TextLogger, Admission: a.textAdmission, Resolve: func(ctx context.Context, input aicapability.TextInputIdentity) (governed.QualifiedRoute, error) {
		route, err := routes.Resolve(ctx, input)
		if err == nil && input.Profile.PromptVersion == agentconfig.ScopedContentPromptVersion {
			route.Profile = agentconfig.ScopedContentProfile(route.Profile)
		}
		return route, err
	}, BaseTransport: a.config.TextTransport, Authorize: func(ctx context.Context, input aicapability.TextInputIdentity) error {
		if scheduledExecutionContext(ctx) {
			return a.authorizeScheduledSend(ctx, input)
		}
		if listingAgentContext(ctx) {
			return a.authorizeListingModel(ctx, input.OrganizationID, input.ActorID, input.MemberID)
		}
		identity, err := a.freshIdentity(ctx)
		if err != nil || identity.TenantID != input.OrganizationID || identity.UserID != input.ActorID || identity.EffectiveMemberID != input.MemberID {
			return review.ErrForbidden
		}
		return nil
	}}
	model, err := texteino.NewContentAgentTextModel(executor, a.configuration, a.selectTitleProfile, func(ctx context.Context, org string) texteino.TextRouteReadiness {
		return texteino.TextRouteReadiness(routes.Readiness(ctx, aicapability.TextInputIdentity{OrganizationID: org, Operation: aicapability.OperationProductContentDecision}))
	}, contentagent.ModelTools(), a.freshIdentity, contexts...)
	if err != nil {
		return nil, err
	}
	model, err = model.WithExecutionResolver(a.resolveExecutionIdentity)
	if err != nil {
		return nil, err
	}
	if a.skills != nil {
		model, err = model.WithSkillContext(productSkillContext{application: a})
		if err != nil {
			return nil, err
		}
	}
	model.WithWorkbenchContext(a.readContentWorkbenchContext)
	a.model = model
	a.reviews.WithWorkbenchAccess(a.authorizeWorkbenchReview)
	a.policyAdoption, err = review.NewPolicyAdoptionService(reader, source, repository, a.authorizer, a.titlePolicyScope, a.resolveTitlePolicy)
	if err != nil {
		return nil, err
	}
	a.policyAdoption.WithExecutionGate(a.acquireSkillReview)
	guard, err := configstore.NewGuard(a.config.RunDB, a.configuration, a.store, func(id, version string) (agent.Limits, error) {
		if id != definition.ID || version != definition.Version {
			return agent.Limits{}, agentconfig.ErrDefinition
		}
		return a.config.Limits, nil
	}, a.currentModeIdentity)
	if err != nil {
		return nil, err
	}
	if a.skills != nil {
		guard, err = guard.WithSkillGuard(a.checkSkillClaim)
		if err != nil {
			return nil, err
		}
	}
	validator := contentagent.Validator{Authorization: a, Configuration: a.configuration, Identity: a.currentModeIdentity, Observed: contentCandidateEvidenceObserved, Review: func(ctx context.Context, input review.ContentValidationInput) (reviewproposal.Proposal, error) {
		if scheduledExecutionContext(ctx) {
			if a.scheduledReviews == nil {
				return reviewproposal.Proposal{}, agent.ErrUnavailable
			}
			return a.scheduledReviews.ValidateContentCandidate(ctx, input)
		}
		if listingAgentContext(ctx) {
			if a.executionReviews == nil {
				return reviewproposal.Proposal{}, agent.ErrUnavailable
			}
			return a.executionReviews.ValidateContentCandidate(ctx, input)
		}
		identity, err := a.freshIdentity(ctx)
		if err != nil {
			return reviewproposal.Proposal{}, err
		}
		return a.reviews.ValidateContentCandidate(authidentity.WithAuthenticatedIdentity(ctx, identity), input)
	}}
	// Keep the application port available for exact old receipt reads even if
	// the deployment profile is closed. Prepare refuses every new script intent
	// without an admitted executor, before Quote/Claim/model dispatch.
	a.runtime, err = einoruntime.New(einoruntime.Config{Script: a, Definition: definition, Tools: definitions, Model: model, Gateway: a, Validator: validator, Authorizer: a, Store: guard})
	return a, err
}
