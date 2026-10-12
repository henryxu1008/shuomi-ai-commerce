package httpapi

import (
	"context"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	einoruntime "task-processor/internal/integration/agent/eino"
	texteino "task-processor/internal/integration/agent/einomodel"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/openai"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/contentagent"
	"task-processor/internal/product/enrichment"
	"task-processor/internal/product/review"
	"task-processor/internal/product/sourcing/tools/sourceevidenceinspect"
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
	definition, definitions := contentagent.Registration()
	a.definition = definition
	principal, ok := deps.PrincipalResolver.(productAgentPrincipal)
	if !ok {
		return nil, agent.ErrUnavailable
	}
	deps.PrincipalResolver = productAgentPrincipal{application: a, request: principal.request}
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
	routes, err := governed.NewOrganizationRouteResolver(openai.NewGormCredentialResolver(a.config.RunDB), policies)
	if err != nil {
		return nil, err
	}
	a.titleRoutes = routes
	a.selectTitleProfile = func(ctx context.Context, org string) (aicapability.ModelProfile, error) {
		route, err := routes.Resolve(ctx, aicapability.TextInputIdentity{OrganizationID: org, Operation: aicapability.OperationProductContentDecision})
		return route.Profile, err
	}
	executor := &governed.Executor{Ledger: a.config.Ledger, Admission: a.textAdmission, Resolve: routes.Resolve, BaseTransport: a.config.TextTransport, Authorize: func(ctx context.Context, input aicapability.TextInputIdentity) error {
		if supplyAgentContext(ctx) {
			return a.authorizeSupplyModel(ctx, input.OrganizationID, input.ActorID, input.MemberID)
		}
		identity, err := a.freshIdentity(ctx)
		if err != nil || identity.TenantID != input.OrganizationID || identity.UserID != input.ActorID || identity.EffectiveMemberID != input.MemberID {
			return review.ErrForbidden
		}
		return nil
	}}
	model, err := texteino.NewContentAgentTextModel(executor, a.configuration, a.selectTitleProfile, func(ctx context.Context, org string) texteino.TextRouteReadiness {
		return texteino.TextRouteReadiness(routes.Readiness(ctx, aicapability.TextInputIdentity{OrganizationID: org, Operation: aicapability.OperationProductContentDecision}))
	}, definition.AllowedTools, a.freshIdentity, contexts...)
	if err != nil {
		return nil, err
	}
	model, err = model.WithExecutionResolver(a.resolveExecutionIdentity)
	if err != nil {
		return nil, err
	}
	a.model = model
	a.policyAdoption, err = review.NewPolicyAdoptionService(reader, source, repository, a.authorizer, a.titlePolicyScope, a.resolveTitlePolicy)
	if err != nil {
		return nil, err
	}
	guard, err := configstore.NewGuard(a.config.RunDB, a.configuration, a.store, func(id, version string) (agent.Limits, error) {
		if id != definition.ID || version != definition.Version {
			return agent.Limits{}, agentconfig.ErrDefinition
		}
		return a.config.Limits, nil
	}, a.currentModeIdentity)
	if err != nil {
		return nil, err
	}
	validator := contentagent.Validator{Authorization: a, Configuration: a.configuration, Identity: a.currentModeIdentity, Observed: contentCandidateEvidenceObserved, Review: func(ctx context.Context, input review.ContentValidationInput) (enrichment.Proposal, error) {
		if supplyAgentContext(ctx) {
			if a.executionReviews == nil {
				return enrichment.Proposal{}, agent.ErrUnavailable
			}
			return a.executionReviews.ValidateContentCandidate(ctx, input)
		}
		identity, err := a.freshIdentity(ctx)
		if err != nil {
			return enrichment.Proposal{}, err
		}
		return a.reviews.ValidateContentCandidate(authidentity.WithAuthenticatedIdentity(ctx, identity), input)
	}}
	a.runtime, err = einoruntime.New(einoruntime.Config{Definition: definition, Tools: definitions, Model: model, Gateway: a, Validator: validator, Authorizer: a, Store: guard})
	return a, err
}
