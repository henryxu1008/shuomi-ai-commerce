package httpapi

import (
	"context"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/commercetool"
	einoruntime "task-processor/internal/integration/agent/eino"
	texteino "task-processor/internal/integration/agent/einomodel"
	governed "task-processor/internal/integration/aicapability/einomodel"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	"task-processor/internal/listing/readiness/tools/readinessinspect"
	"task-processor/internal/product/asset/tools/assetinspect"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/review"
	"task-processor/internal/product/sourcing/tools/sourceevidenceinspect"
)

// Only the full title composition consumes Asset; shared owner construction and
// the content-only composition never create these tools or this runtime.
func buildTitleProductRuntime(ctx context.Context, a *productAgentApplication, reader catalog.VersionedSnapshotReader, source review.SourcePublicationGateway, repository review.Store, deps commercetool.InvocationDependencies, contexts []texteino.KnowledgeContext) error {
	assets, err := assetstore.NewBoundedApprovedInventoryReader(a.config.AssetDB, 1<<20)
	if err != nil {
		return err
	}
	a.policyAdoption, err = review.NewPolicyAdoptionService(reader, source, repository, a.authorizer, a.titlePolicyScope, a.resolveTitlePolicy)
	if err != nil {
		return err
	}
	definition, definitions := productTitleRegistration()
	principal, ok := deps.PrincipalResolver.(productAgentPrincipal)
	if !ok {
		return agent.ErrUnavailable
	}
	a.canonical, err = canonicalinspect.NewInvoker(reader, definition, deps)
	if err != nil {
		return err
	}
	a.sources, err = sourceevidenceinspect.NewInvoker(reader, productAgentSourceGateway{application: a, request: source}, definition, deps)
	if err != nil {
		return err
	}
	a.assets, err = assetinspect.NewInvoker(reader, assets, principal, definition, deps)
	if err != nil {
		return err
	}
	a.readiness, err = readinessinspect.NewInvoker(reader, assets, principal, definition, deps)
	if err != nil {
		return err
	}
	routes := make(map[governed.RouteKey]governed.RoutePolicy, len(a.config.TextPolicies))
	for organizationID, policy := range a.config.TextPolicies {
		routes[governed.RouteKey{OrganizationID: organizationID, Operation: aicapability.OperationProductAgentDecision}] = policy
	}
	routeResolver, err := newProductTextRouteResolver(a.config, routes)
	if err != nil {
		return err
	}
	a.titleRoutes = routeResolver
	a.selectTitleProfile = func(ctx context.Context, organizationID string) (aicapability.ModelProfile, error) {
		route, routeErr := routeResolver.Resolve(ctx, aicapability.TextInputIdentity{OrganizationID: organizationID, Operation: aicapability.OperationProductAgentDecision})
		return route.Profile, routeErr
	}
	executor := &governed.Executor{Ledger: a.config.Ledger, Logger: a.config.TextLogger, Admission: a.textAdmission, Resolve: routeResolver.Resolve, BaseTransport: a.config.TextTransport, Authorize: func(ctx context.Context, in aicapability.TextInputIdentity) error {
		if supplyAgentContext(ctx) {
			return a.authorizeSupplyModel(ctx, in.OrganizationID, in.ActorID, in.MemberID)
		}
		i, e := a.freshIdentity(ctx)
		if e != nil || i.TenantID != in.OrganizationID || i.UserID != in.ActorID || i.EffectiveMemberID != in.MemberID {
			return review.ErrForbidden
		}
		return nil
	}}
	model, err := texteino.NewAgentTextModel(executor, a.configuration, a.selectTitleProfile,
		func(ctx context.Context, org string) texteino.TextRouteReadiness {
			return texteino.TextRouteReadiness(routeResolver.Readiness(ctx, aicapability.TextInputIdentity{
				OrganizationID: org, Operation: aicapability.OperationProductAgentDecision}))
		}, definition.AllowedTools, a.freshIdentity, contexts...)
	if err != nil {
		return err
	}
	model, err = model.WithExecutionResolver(a.resolveExecutionIdentity)
	if err != nil {
		return err
	}
	a.model, a.definition = model, definition
	guard, err := configstore.NewGuard(a.config.RunDB, a.configuration, a.store, func(id, version string) (agent.Limits, error) {
		if id != definition.ID || version != definition.Version {
			return agent.Limits{}, agentconfig.ErrDefinition
		}
		return a.config.Limits, nil
	}, a.currentModeIdentity)
	if err != nil {
		return err
	}
	a.runtime, err = einoruntime.New(einoruntime.Config{Definition: definition, Tools: definitions, Model: model, Gateway: a, Validator: a, Authorizer: a, Store: guard})
	if err != nil {
		return err
	}

	return nil
}
