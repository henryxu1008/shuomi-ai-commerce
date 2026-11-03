package httpapi

import (
	"context"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
	einoruntime "task-processor/internal/integration/agent/eino"
	texteino "task-processor/internal/integration/agent/einomodel"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/commercetoolauth"
	runstore "task-processor/internal/integration/persistence/agent"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"task-processor/internal/product/research/analysis"
	"task-processor/internal/product/research/analysis/tools/researchinspect"
	"task-processor/internal/product/selection"
	"time"

	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type researchExecutionContextKey struct{}
type marketResearchApplication struct {
	parent        *productAgentApplication
	inspector     analysis.Inspector
	research      *research.Service
	resolver      organizationIdentityResolver
	authorizer    *authz.ListingKitAuthorizer
	configuration *configstore.Store
	store         *runstore.Store
	model         *texteino.AgentTextModel
	runtime       *einoruntime.Runtime
	gateway       *researchinspect.Invoker
	profile       func(context.Context, string) (aicapability.ModelProfile, error)
}

// Facts stay available without a configured model. The original optional
// run/configuration pools supply history; construction performs no DDL/DML.
func buildMarketResearchApplication(ctx context.Context, parent *productAgentApplication, collections *collection.Service, researchService *research.Service, db *gorm.DB, resolver organizationIdentityResolver, auth *authz.ListingKitAuthorizer) (*marketResearchApplication, error) {
	if researchService == nil || collections == nil || resolver == nil || auth == nil {
		return nil, agent.ErrUnavailable
	}
	a := &marketResearchApplication{parent: parent, research: researchService, inspector: analysis.Inspector{Research: researchService, Products: selection.Inspector{Collections: collections}}, resolver: resolver, authorizer: auth}
	if parent != nil {
		if db != nil && parent.config.RunDB != db {
			return nil, agent.ErrUnavailable
		}
		a.configuration, a.store = parent.configuration, parent.store
	} else if db != nil {
		var err error
		a.configuration, err = configstore.New(db)
		if err != nil {
			return nil, err
		}
		if db.Migrator().HasTable("product_agent_runs") {
			a.store, err = runstore.New(db)
			if err != nil {
				return nil, err
			}
		}
	}
	if parent == nil || len(parent.config.ResearchTextPolicies) == 0 {
		return a, nil
	}
	policies := map[governed.RouteKey]governed.RoutePolicy{}
	for org, p := range parent.config.ResearchTextPolicies {
		policies[governed.RouteKey{OrganizationID: org, Operation: aicapability.OperationProductResearchDecision}] = p
	}
	routes, err := newProductTextRouteResolver(parent.config, policies)
	if err != nil {
		return nil, err
	}
	a.profile = func(ctx context.Context, org string) (aicapability.ModelProfile, error) {
		r, e := routes.Resolve(ctx, aicapability.TextInputIdentity{OrganizationID: org, Operation: aicapability.OperationProductResearchDecision})
		return r.Profile, e
	}
	executor := &governed.Executor{Ledger: parent.config.Ledger, Logger: parent.config.TextLogger, Admission: parent.textAdmission, Resolve: routes.Resolve, BaseTransport: parent.config.TextTransport, Authorize: func(ctx context.Context, in aicapability.TextInputIdentity) error {
		identity, e := a.executionIdentity(ctx)
		if e != nil || identity.TenantID != in.OrganizationID || identity.UserID != in.ActorID || identity.EffectiveMemberID != in.MemberID {
			return agentconfig.ErrForbidden
		}
		return nil
	}}
	a.model, err = texteino.NewResearchAgentTextModel(executor, a.configuration, a.profile, func(ctx context.Context, org string) texteino.TextRouteReadiness {
		return texteino.TextRouteReadiness(routes.Readiness(ctx, aicapability.TextInputIdentity{OrganizationID: org, Operation: aicapability.OperationProductResearchDecision}))
	}, []commercetool.ToolRef{agentconfig.ResearchInspectToolRef()}, a.executionIdentity, analysis.ModelAdmission{}, func(ctx context.Context, in agent.ModelInput) error {
		ctx = context.WithValue(ctx, researchExecutionContextKey{}, agent.Request{Binding: in.Binding, ConfigurationSnapshotRef: in.ConfigurationSnapshotRef, ModeSnapshotRef: in.ModeSnapshotRef})
		_, e := a.Authorize(ctx, in.Binding)
		if e == nil {
			_, e = a.inspector.Inspect(ctx, in.Binding)
		}
		return e
	})
	if err != nil {
		return nil, err
	}
	fresh, err := commercetoolauth.NewFreshWorkbenchPrincipalResolver(commercetoolauth.FreshOrganizationResolverFunc(func(ctx context.Context, _ commercetoolauth.OrganizationRequest) (authidentity.AuthenticatedIdentity, error) {
		return a.executionIdentity(ctx)
	}), time.Now)
	if err != nil {
		return nil, err
	}
	permissions, err := commercetoolauth.NewCasbinAuthorizer(auth)
	if err != nil {
		return nil, err
	}
	a.gateway, err = researchinspect.NewInvoker(a.inspector, fresh, commercetool.InvocationDependencies{Authorizer: permissions, Recorder: a.store, Tracer: otel.Tracer("market-research-agent"), Now: time.Now, AuditTimeout: 2 * time.Second})
	if err != nil {
		return nil, err
	}
	guard, err := configstore.NewGuard(parent.config.RunDB, a.configuration, a.store, func(id, version string) (agent.Limits, error) {
		if id != agentconfig.ResearchAgentID || version != agentconfig.ResearchAgentVersion {
			return agent.Limits{}, agentconfig.ErrDefinition
		}
		return parent.config.Limits, nil
	}, a.currentExecutionIdentity)
	if err != nil {
		return nil, err
	}
	a.runtime, err = einoruntime.New(einoruntime.Config{Definition: researchinspect.AgentDefinition(), Tools: []commercetool.Definition{researchinspect.Definition()}, Model: a.model, Gateway: a, DomainProposalKind: agentconfig.ResearchProposalKind, DomainValidator: analysis.Validator{Reader: a.inspector, Configuration: a.configuration}, Authorizer: a, Store: guard})
	return a, err
}
func (a *marketResearchApplication) freshIdentity(ctx context.Context) (authidentity.AuthenticatedIdentity, error) {
	original, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok {
		return original, agentconfig.ErrForbidden
	}
	id, err := freshWorkbenchIdentity(ctx, a.resolver, a.authorizer, []string{original.EffectiveOrganizationID}, collection.PermissionRead)
	if err == nil && !authz.AllowedOrganization(ctx, a.authorizer, id.UserID, id.EffectiveOrganizationID, id.Roles, authz.PermissionWorkbenchAgentUse) {
		err = agentconfig.ErrForbidden
	}
	return id, err
}
func (a *marketResearchApplication) currentIdentity(ctx context.Context) (agent.ExecutionIdentity, error) {
	id, e := a.freshIdentity(ctx)
	return agent.ExecutionIdentity{OrganizationID: id.TenantID, ActorID: id.UserID, MemberID: id.EffectiveMemberID}, e
}
func (a *marketResearchApplication) executionIdentity(ctx context.Context) (authidentity.AuthenticatedIdentity, error) {
	if a.parent == nil {
		return authidentity.AuthenticatedIdentity{}, agent.ErrUnavailable
	}
	return a.parent.freshIdentity(ctx)
}
func (a *marketResearchApplication) currentExecutionIdentity(ctx context.Context) (agent.ExecutionIdentity, error) {
	id, e := a.executionIdentity(ctx)
	return agent.ExecutionIdentity{OrganizationID: id.TenantID, ActorID: id.UserID, MemberID: id.EffectiveMemberID}, e
}
func (a *marketResearchApplication) Authorize(ctx context.Context, binding agent.Binding) (agent.Scope, error) {
	id, e := a.currentIdentity(ctx)
	if e != nil {
		return agent.Scope{}, e
	}
	if request, ok := ctx.Value(researchExecutionContextKey{}).(agent.Request); ok {
		if request.Binding != binding || a.configuration == nil {
			return agent.Scope{}, agent.ErrConflict
		}
		if _, e = a.configuration.CheckRegisteredTextExecutionMode(ctx, id, request.ConfigurationSnapshotRef, request.ModeSnapshotRef); e != nil {
			return agent.Scope{}, e
		}
	}
	current, e := a.inspector.AuthorizeHistory(ctx, binding)
	if e != nil {
		return agent.Scope{}, e
	}
	if current != (collection.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID, MemberID: id.MemberID}) {
		return agent.Scope{}, agentconfig.ErrForbidden
	}
	return agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, nil
}
func (a *marketResearchApplication) Invoke(ctx context.Context, ref commercetool.ToolRef, meta commercetool.CallMetadata, binding agent.Binding) (commercetool.Result, error) {
	if _, e := a.Authorize(ctx, binding); e != nil {
		return commercetool.Result{}, e
	}
	id, e := a.executionIdentity(ctx)
	if e != nil {
		return commercetool.Result{}, e
	}
	capability, ok := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability)
	if !ok || a.gateway == nil {
		return commercetool.Result{}, agentconfig.ErrForbidden
	}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, id)
	ctx = commercetoolauth.WithOrganizationRequest(ctx, commercetoolauth.OrganizationRequest{Identity: id, BearerToken: capability.bearerToken, RequestedOrganizationID: id.TenantID})
	return a.gateway.Invoke(ctx, ref, meta, binding)
}

func (a *marketResearchApplication) readIdentity(ctx context.Context) (authidentity.AuthenticatedIdentity, error) {
	original, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok {
		return original, agentconfig.ErrForbidden
	}
	return freshWorkbenchIdentity(ctx, a.resolver, a.authorizer, []string{original.EffectiveOrganizationID}, collection.PermissionRead)
}
