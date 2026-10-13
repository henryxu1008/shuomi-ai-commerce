package httpapi

import (
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	supplyapp "task-processor/internal/app/supplychain"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	einoruntime "task-processor/internal/integration/agent/eino"
	texteino "task-processor/internal/integration/agent/einomodel"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/commercetoolauth"
	"task-processor/internal/integration/openai"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"task-processor/internal/listing/materials"
	"task-processor/internal/listing/materials/tools/materialsinspect"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
)

type listingMaterialsApplication struct {
	parent  *productAgentApplication
	supply  *supplyapp.Application
	model   *texteino.AgentTextModel
	runtime *einoruntime.Runtime
	gateway *materialsinspect.Invoker
	profile func(context.Context, string) (aicapability.ModelProfile, error)
}

// Runs/configuration/ledger are the injected original owners. Missing model or
// installed configuration disables execution while historical reads remain.
func buildListingMaterialsApplication(ctx context.Context, parent *productAgentApplication, supply *supplyapp.Application) (*listingMaterialsApplication, error) {
	if parent == nil || parent.configuration == nil || parent.store == nil || supply == nil || supply.Targets == nil {
		return nil, agent.ErrUnavailable
	}
	a := &listingMaterialsApplication{parent: parent, supply: supply}
	if len(parent.config.MaterialsTextPolicies) == 0 || configstore.VerifyMaterialsConfigurationSchema(ctx, parent.config.RunDB) != nil {
		return a, nil
	}
	policies := map[governed.RouteKey]governed.RoutePolicy{}
	for org, p := range parent.config.MaterialsTextPolicies {
		policies[governed.RouteKey{OrganizationID: org, Operation: aicapability.OperationListingMaterialsDecision}] = p
	}
	routes, err := governed.NewOrganizationRouteResolver(openai.NewGormCredentialResolver(parent.config.RunDB), policies)
	if err != nil {
		return nil, err
	}
	a.profile = func(ctx context.Context, org string) (aicapability.ModelProfile, error) {
		r, e := routes.Resolve(ctx, aicapability.TextInputIdentity{OrganizationID: org, Operation: aicapability.OperationListingMaterialsDecision})
		return r.Profile, e
	}
	executor := &governed.Executor{Ledger: parent.config.Ledger, Admission: parent.textAdmission, Resolve: routes.Resolve, BaseTransport: parent.config.TextTransport, Authorize: func(ctx context.Context, in aicapability.TextInputIdentity) error {
		id, err := a.currentIdentity(ctx)
		if err != nil || id.OrganizationID != in.OrganizationID || id.ActorID != in.ActorID || id.MemberID != in.MemberID {
			return preparation.ErrForbidden
		}
		return nil
	}}
	definition := materialsinspect.AgentDefinition()
	a.model, err = texteino.NewMaterialsAgentTextModel(executor, parent.configuration, a.profile, func(ctx context.Context, org string) texteino.TextRouteReadiness {
		return texteino.TextRouteReadiness(routes.Readiness(ctx, aicapability.TextInputIdentity{OrganizationID: org, Operation: aicapability.OperationListingMaterialsDecision}))
	}, definition.AllowedTools, parent.freshIdentity, materials.ModelAdmission{}, func(ctx context.Context, in agent.ModelInput) error {
		scope, err := a.Authorize(ctx, in.Binding)
		if err != nil {
			return err
		}
		frozen, err := parent.configuration.LoadSnapshot(ctx, scope, in.ConfigurationSnapshotRef)
		if err != nil {
			return err
		}
		if frozen.Request.Binding != in.Binding || frozen.AgentID != materials.AgentID {
			return agentconfig.ErrConflict
		}
		return a.checkTemplate(ctx, frozen)
	})
	if err != nil {
		return nil, err
	}
	fresh, err := commercetoolauth.NewFreshWorkbenchPrincipalResolver(commercetoolauth.FreshOrganizationResolverFunc(func(ctx context.Context, _ commercetoolauth.OrganizationRequest) (authidentity.AuthenticatedIdentity, error) {
		return parent.freshIdentity(ctx)
	}), time.Now)
	if err != nil {
		return nil, err
	}
	permissions, err := commercetoolauth.NewCasbinAuthorizer(parent.authorizer)
	if err != nil {
		return nil, err
	}
	gateway, err := materialsinspect.NewInvoker(supply, fresh, definition, commercetool.InvocationDependencies{Authorizer: permissions, Recorder: parent.store, Tracer: otel.Tracer("listing-materials-agent"), Now: time.Now, AuditTimeout: 2 * time.Second})
	if err != nil {
		return nil, err
	}
	a.gateway = gateway
	guard, err := configstore.NewGuard(parent.config.RunDB, parent.configuration, parent.store, func(id, version string) (agent.Limits, error) {
		if id != materials.AgentID || version != materials.AgentVersion {
			return agent.Limits{}, agentconfig.ErrDefinition
		}
		return parent.config.Limits, nil
	}, a.currentIdentity)
	if err != nil {
		return nil, err
	}
	a.runtime, err = einoruntime.New(einoruntime.Config{Definition: definition, Tools: []commercetool.Definition{materialsinspect.Definition()}, Model: a.model, Gateway: a, DomainProposalKind: materials.ProposalKind, DomainValidator: materials.Validator{Reader: supply}, Authorizer: a, Store: guard})
	return a, err
}

func (a *listingMaterialsApplication) Invoke(ctx context.Context, ref commercetool.ToolRef, meta commercetool.CallMetadata, b agent.Binding) (commercetool.Result, error) {
	if _, err := a.Authorize(ctx, b); err != nil {
		return commercetool.Result{}, err
	}
	id, err := a.parent.freshIdentity(ctx)
	if err != nil {
		return commercetool.Result{}, err
	}
	capability, ok := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability)
	if !ok || a.gateway == nil {
		return commercetool.Result{}, preparation.ErrForbidden
	}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, id)
	ctx = commercetoolauth.WithOrganizationRequest(ctx, commercetoolauth.OrganizationRequest{Identity: id, BearerToken: capability.bearerToken, RequestedOrganizationID: id.TenantID})
	return a.gateway.Invoke(ctx, ref, meta, b)
}

func (a *listingMaterialsApplication) currentIdentity(ctx context.Context) (agent.ExecutionIdentity, error) {
	if a == nil || a.parent == nil {
		return agent.ExecutionIdentity{}, agent.ErrUnavailable
	}
	id, err := a.parent.freshIdentity(ctx)
	if err != nil {
		return agent.ExecutionIdentity{}, err
	}
	return agent.ExecutionIdentity{OrganizationID: id.TenantID, ActorID: id.UserID, MemberID: id.EffectiveMemberID}, nil
}

// The runtime's new execution checks exact current head/owner facts. The
// read path below intentionally does not call this gate after a draft is saved.
func (a *listingMaterialsApplication) Authorize(ctx context.Context, binding agent.Binding) (agent.Scope, error) {
	id, err := a.currentIdentity(ctx)
	if err != nil {
		return agent.Scope{}, err
	}
	i, err := a.supply.ReadMaterialsInspection(ctx, binding)
	if err != nil {
		return agent.Scope{}, err
	}
	if i.Product.Identity.TenantID != id.OrganizationID {
		return agent.Scope{}, preparation.ErrForbidden
	}
	return agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}, nil
}

func (a *listingMaterialsApplication) readBase(ctx context.Context, recordID string) (agent.ExecutionIdentity, record.TargetRecord, agent.Binding, error) {
	id, err := a.currentIdentity(ctx)
	if err != nil {
		return id, record.TargetRecord{}, agent.Binding{}, err
	}
	base, err := a.supply.ReadRecord(ctx, recordID)
	if err != nil {
		return id, base, agent.Binding{}, err
	}
	selected, err := a.supply.Sources.Select(ctx, base.Source.ID)
	if err != nil {
		return id, base, agent.Binding{}, err
	}
	scope, source, _, err := selected.Read(ctx)
	if err != nil || scope != (collection.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID, MemberID: id.MemberID}) || source != base.Source {
		return id, base, agent.Binding{}, preparation.ErrForbidden
	}
	product, err := a.supply.Products.ReadEffectiveTargetProduct(ctx, selected, base.EffectiveVersion, base.ApplyReceiptID)
	if err != nil {
		return id, base, agent.Binding{}, err
	}
	b := agent.Binding{ContextKind: "listing-target-record", ContextID: base.ID, ProductKey: product.Identity.ProductKey, CatalogVersion: strconv.FormatUint(product.Version, 10), PublicationID: product.PublicationID, TargetPlatform: "shein"}
	if !b.Valid() || product.Identity.TenantID != id.OrganizationID || product.Version != base.EffectiveVersion || collection.Digest(product.Snapshot) != base.ProductHash {
		return id, base, b, agent.ErrConflict
	}
	return id, base, b, nil
}

func materialsRunReviewable(s agent.State) bool {
	return s.Phase == agent.HumanReviewRequired && s.StopReason == "" && s.RejectedPayload == nil && s.Request.Binding.ContextKind == "listing-target-record" && s.Request.Binding.TargetPlatform == "shein" && s.Request.PolicyVersion == materials.PolicyVersion && s.Request.PromptVersion == materials.PromptVersion && s.DomainProposal != nil && s.DomainProposal.Kind == materials.ProposalKind && s.Validation != nil && s.Validation.Valid && s.Validation.PolicyVersion == materials.PolicyVersion && s.Validation.CandidateHash == collection.Digest(s.DomainProposal) && len(s.Candidate.Changes) == 0 && len(s.Candidate.Warnings) == 0 && len(s.Candidate.Rejections) == 0
}

func materialsAdoptionKey(s agent.State) string {
	return collection.StableID(s.Scope.OrganizationID, s.Scope.ActorID, "listing-materials-adopt", s.RunID, s.Validation.CandidateHash)
}

var _ agent.Authorizer = (*listingMaterialsApplication)(nil)

func (a *listingMaterialsApplication) checkTemplate(ctx context.Context, config agentconfig.Snapshot) error {
	if config.Template == nil {
		return nil
	}
	version, err := agentconfig.ParseModeRevision(config.Template.Revision)
	if err != nil || version == 0 {
		return agentconfig.ErrConflict
	}
	template, err := a.parent.configuration.ReadTemplate(ctx, config.Scope, materials.AgentID, config.Template.TemplateID, version)
	if err != nil {
		return err
	}
	if template.Lifecycle != "ACTIVE" {
		return agentconfig.ErrArchived
	}
	return nil
}
