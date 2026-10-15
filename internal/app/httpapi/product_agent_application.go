package httpapi

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	aistore "task-processor/internal/aicapability/store"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
	"task-processor/internal/httproute"
	einoruntime "task-processor/internal/integration/agent/eino"
	texteino "task-processor/internal/integration/agent/einomodel"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/commercetoolauth"
	"task-processor/internal/integration/knowledgeauth"
	orgresourceadapter "task-processor/internal/integration/orgresource"
	agentstore "task-processor/internal/integration/persistence/agent"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	reviewstore "task-processor/internal/integration/persistence/product/review"
	skillstore "task-processor/internal/integration/persistence/skill"
	"task-processor/internal/knowledge"
	"task-processor/internal/listing/readiness/tools/readinessinspect"
	"task-processor/internal/product/asset/tools/assetinspect"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/enrichment"
	"task-processor/internal/product/review"
	"task-processor/internal/product/sourcing"
	"task-processor/internal/product/sourcing/tools/sourceevidenceinspect"
	"task-processor/internal/workbenchcontext"

	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

// ProductAgentDependencies are supplied by the current application owner. Pools
// and the manager are caller-owned. Construction never installs schema, creates
// entitlements, calls the model or edits a Product.
type ProductAgentDependencies struct {
	Knowledge                *knowledge.Service
	RunDB, AssetDB, ReviewDB *gorm.DB
	PointAccountingDB        *gorm.DB
	Ledger                   ProductAgentInvocationLedger
	TextPolicies             map[string]governed.RoutePolicy
	ContentTextPolicies      map[string]governed.RoutePolicy
	MaterialsTextPolicies    map[string]governed.RoutePolicy
	DiagnosisTextPolicies    map[string]governed.RoutePolicy
	SelectionTextPolicies    map[string]governed.RoutePolicy
	Enabled                  bool
	ContentTitleOnly         bool
	AllowedOrganizationIDs   []string
	Limits                   agent.Limits
	// Code-owned transport injection for isolated protocol fixtures; never
	// populated by the deployment manifest or browser input.
	TextTransport http.RoundTripper `json:"-"`
}

type ProductAgentInvocationLedger interface {
	governed.InvocationLedger
	orgresourceadapter.ModelInvocationFactReader
	SetUsageSettler(aicapability.InvocationUsageSettler)
}

type productAgentApplication struct {
	skills                  *skillstore.Store
	materials               *listingMaterialsApplication
	content                 *productAgentApplication
	configuration           *configstore.Store
	snapshots               catalog.CompleteSnapshotReader
	reviewLookup            review.AppliedPublicationLookup
	model                   *texteino.AgentTextModel
	selectTitleProfile      func(context.Context, string) (aicapability.ModelProfile, error)
	titleRoutes             *governed.OrganizationRouteResolver
	definition              commercetool.AgentDefinition
	context                 *knowledge.ContextService
	runtime                 *einoruntime.Runtime
	store                   *agentstore.Store
	reviews                 *review.Service
	policyAdoption          *review.PolicyAdoptionService
	executionPolicyAdoption *review.PolicyAdoptionService
	receipts                sourcing.PublishedAcquisitionReader
	resolver                organizationIdentityResolver
	originExecution         func(context.Context, agent.ExecutionIdentity) error
	authorizer              *authz.ListingKitAuthorizer
	config                  ProductAgentDependencies
	canonical               *canonicalinspect.Invoker
	sources                 *sourceevidenceinspect.Invoker
	assets                  *assetinspect.Invoker
	readiness               *readinessinspect.Invoker
	points                  *orgresourceadapter.GormModelInvocationRepository
	textAdmission           *governed.BoundedAdmission
	supplyExecution         *supplyProductAgent
	executionSource         *sourcing.ExecutionPublicationGateway
	executionReviews        *review.ExecutionCandidateService
	executionReviewLookup   review.AppliedPublicationLookup
}

func buildProductAgentApplication(ctx context.Context, productDB *gorm.DB, receipts sourcing.PublishedAcquisitionReader, resolver organizationIdentityResolver, auth *authz.ListingKitAuthorizer, cfg ProductAgentDependencies) (*productAgentApplication, error) {
	if !cfg.Enabled || len(cfg.AllowedOrganizationIDs) == 0 || !cfg.Limits.Valid() || cfg.Limits.Runtime > 2*time.Minute || cfg.Limits.Steps > 16 || cfg.Limits.ModelCalls > 8 || productDB == nil || !cfg.ContentTitleOnly && cfg.AssetDB == nil || receipts == nil || resolver == nil || auth == nil {
		return nil, agent.ErrUnavailable
	}
	if cfg.ContentTitleOnly && (cfg.AssetDB != nil || len(cfg.ContentTextPolicies) == 0 || len(cfg.TextPolicies) != 0 || len(cfg.MaterialsTextPolicies) != 0 || len(cfg.SelectionTextPolicies) != 0 || len(cfg.DiagnosisTextPolicies) != 0) {
		return nil, agent.ErrUnavailable
	}
	allowedOrganizations := make(map[string]bool, len(cfg.AllowedOrganizationIDs))
	for _, organizationID := range cfg.AllowedOrganizationIDs {
		if !agent.ValidID(organizationID) || allowedOrganizations[organizationID] {
			return nil, agent.ErrUnavailable
		}
		allowedOrganizations[organizationID] = true
	}
	for organizationID, policy := range cfg.TextPolicies {
		profile := policy.ShapeProfile()
		costMicros, err := profile.MaximumCost()
		if !allowedOrganizations[organizationID] || !governed.ValidRouteProfile(profile, aicapability.OperationProductAgentDecision) || profile.Currency != cfg.Limits.Currency || err != nil || profile.MaximumPromptTokens+profile.MaximumCompletionTokens > cfg.Limits.Tokens || costMicros > cfg.Limits.CostMicros {
			return nil, agent.ErrUnavailable
		}
	}
	for organizationID, policy := range cfg.ContentTextPolicies {
		profile := policy.ShapeProfile()
		cost, err := profile.MaximumCost()
		if !allowedOrganizations[organizationID] || profile.PromptVersion != agentconfig.ContentPromptVersion || profile.OutputSchemaVersion != agentconfig.ContentActionVersion || !governed.ValidRouteProfile(profile, aicapability.OperationProductContentDecision) || profile.Currency != cfg.Limits.Currency || err != nil || profile.MaximumPromptTokens+profile.MaximumCompletionTokens > cfg.Limits.Tokens || cost > cfg.Limits.CostMicros {
			return nil, agent.ErrUnavailable
		}
	}
	for org, policy := range cfg.MaterialsTextPolicies {
		profile := policy.ShapeProfile()
		cost, err := profile.MaximumCost()
		if !allowedOrganizations[org] || profile.PromptVersion != agentconfig.MaterialsPromptVersion || profile.OutputSchemaVersion != agentconfig.MaterialsActionVersion || !governed.ValidRouteProfile(profile, aicapability.OperationListingMaterialsDecision) || profile.Currency != cfg.Limits.Currency || err != nil || profile.MaximumPromptTokens+profile.MaximumCompletionTokens > cfg.Limits.Tokens || cost > cfg.Limits.CostMicros {
			return nil, agent.ErrUnavailable
		}
	}
	for org, policy := range cfg.SelectionTextPolicies {
		if cfg.Limits.Steps < 4 || cfg.Limits.ModelCalls < 2 {
			return nil, agent.ErrUnavailable
		}
		profile := policy.ShapeProfile()
		cost, err := profile.MaximumCost()
		if !allowedOrganizations[org] || profile.PromptVersion != agentconfig.SelectionPromptVersion || profile.OutputSchemaVersion != agentconfig.SelectionActionVersion || !governed.ValidRouteProfile(profile, aicapability.OperationProductSelectionDecision) || profile.Currency != cfg.Limits.Currency || err != nil || profile.MaximumCompletionTokens > cfg.Limits.Tokens/2 || profile.MaximumPromptTokens > cfg.Limits.Tokens/2-profile.MaximumCompletionTokens || cost > cfg.Limits.CostMicros/2 {
			return nil, agent.ErrUnavailable
		}
	}

	for org, policy := range cfg.DiagnosisTextPolicies {
		if cfg.Limits.Steps < 4 || cfg.Limits.ModelCalls < 2 {
			return nil, agent.ErrUnavailable
		}
		profile := policy.ShapeProfile()
		cost, err := profile.MaximumCost()
		if !allowedOrganizations[org] || profile.PromptVersion != agentconfig.DiagnosisPromptVersion || profile.OutputSchemaVersion != agentconfig.DiagnosisActionVersion || !governed.ValidRouteProfile(profile, aicapability.OperationShopDiagnosisDecision) || profile.Currency != cfg.Limits.Currency || err != nil || profile.MaximumCompletionTokens > cfg.Limits.Tokens/2 || profile.MaximumPromptTokens > cfg.Limits.Tokens/2-profile.MaximumCompletionTokens || cost > cfg.Limits.CostMicros/2 {
			return nil, agent.ErrUnavailable
		}
	}
	if cfg.PointAccountingDB == nil || cfg.Ledger == nil {
		return nil, agent.ErrUnavailable
	}
	if err := aistore.VerifyModelPointInvocationSchema(ctx, cfg.RunDB); err != nil {
		return nil, err
	}
	for _, table := range []string{"product_agent_runs", "product_agent_tool_calls"} {
		if cfg.RunDB == nil || !cfg.RunDB.Migrator().HasTable(table) {
			return nil, agent.ErrUnavailable
		}
	}
	if !productReviewSchemaReady(productDB) {
		return nil, agent.ErrUnavailable
	}
	a := &productAgentApplication{receipts: receipts, resolver: resolver, authorizer: auth, config: cfg}
	a.config.AllowedOrganizationIDs = append([]string(nil), cfg.AllowedOrganizationIDs...)
	a.config.TextPolicies = make(map[string]governed.RoutePolicy, len(cfg.TextPolicies))
	for organizationID, policy := range cfg.TextPolicies {
		a.config.TextPolicies[organizationID] = policy
	}
	a.config.ContentTextPolicies = make(map[string]governed.RoutePolicy, len(cfg.ContentTextPolicies))
	for organizationID, policy := range cfg.ContentTextPolicies {
		a.config.ContentTextPolicies[organizationID] = policy
	}
	a.config.MaterialsTextPolicies = make(map[string]governed.RoutePolicy, len(cfg.MaterialsTextPolicies))
	for org, policy := range cfg.MaterialsTextPolicies {
		a.config.MaterialsTextPolicies[org] = policy
	}
	a.config.SelectionTextPolicies = make(map[string]governed.RoutePolicy, len(cfg.SelectionTextPolicies))
	for org, policy := range cfg.SelectionTextPolicies {
		a.config.SelectionTextPolicies[org] = policy
	}

	a.config.DiagnosisTextPolicies = make(map[string]governed.RoutePolicy, len(cfg.DiagnosisTextPolicies))
	for org, policy := range cfg.DiagnosisTextPolicies {
		a.config.DiagnosisTextPolicies[org] = policy
	}
	var err error
	var contexts []texteino.KnowledgeContext
	if cfg.Knowledge != nil {
		knowledgeAuth, authErr := knowledgeauth.NewAuthorizer(resolver, auth)
		if authErr != nil {
			return nil, authErr
		}
		a.context, err = cfg.Knowledge.Context(knowledgeAuth)
		if err != nil {
			return nil, err
		}
		contexts = append(contexts, a.context)
	}
	a.points, err = orgresourceadapter.NewGormModelInvocationRepository(cfg.PointAccountingDB, orgresourceadapter.TransactionConfig{}, cfg.Ledger, a)
	if err != nil {
		return nil, err
	}
	cfg.Ledger.SetUsageSettler(a.points)
	a.store, err = agentstore.New(cfg.RunDB)
	if err != nil {
		return nil, err
	}
	if err := configstore.VerifySchema(ctx, cfg.RunDB); err != nil {
		return nil, err
	}
	a.configuration, err = configstore.New(cfg.RunDB)
	if err != nil {
		return nil, err
	}
	reader, err := catalogstore.NewBoundedSnapshotReader(productDB, 1<<20)
	if err != nil {
		return nil, err
	}
	sources, err := productsourcing.NewInternalProducer(productDB, &productReviewLiveOrganizationAccess{resolver: resolver, now: time.Now}, auth)
	if err != nil {
		return nil, err
	}
	repo, err := reviewstore.NewRepository(productDB, func(tx *gorm.DB) (review.SourcePublicationReader, error) {
		return productsourcing.NewTransactionReader(tx)
	})
	if err != nil {
		return nil, err
	}
	a.reviews, err = review.NewCandidateService(reader, sources, repo, auth)
	if err != nil {
		return nil, err
	}
	a.reviews.WithExecutionPolicy(a.resolveTitlePolicy)
	a.snapshots, a.reviewLookup = reader, repo
	fresh, err := commercetoolauth.NewFreshWorkbenchPrincipalResolver(commercetoolauth.FreshOrganizationResolverFunc(func(ctx context.Context, _ commercetoolauth.OrganizationRequest) (authidentity.AuthenticatedIdentity, error) {
		return a.freshIdentity(ctx)
	}), time.Now)
	if err != nil {
		return nil, err
	}
	permissions, err := commercetoolauth.NewCasbinAuthorizer(auth)
	if err != nil {
		return nil, err
	}
	principal := productAgentPrincipal{application: a, request: fresh}
	deps := commercetool.InvocationDependencies{PrincipalResolver: principal, Authorizer: permissions, Recorder: a.store, Tracer: otel.Tracer("product-agent"), Now: time.Now, AuditTimeout: 2 * time.Second}
	a.textAdmission = governed.NewBoundedAdmission()
	if cfg.ContentTitleOnly {
		a.content, err = buildContentProductApplication(ctx, a, reader, sources, repo, deps, contexts)
		if err != nil {
			return nil, err
		}
		if a.content == nil {
			return nil, agent.ErrUnavailable
		}
		a.reviews.WithExecutionGate(a.acquireSkillReview)
		return a, nil
	}
	if err := buildTitleProductRuntime(ctx, a, reader, sources, repo, deps, contexts); err != nil {
		return nil, err
	}
	a.content, err = buildContentProductApplication(ctx, a, reader, sources, repo, deps, contexts)
	if err != nil {
		return nil, err
	}
	a.reviews.WithExecutionGate(a.acquireSkillReview)
	if a.executionReviews != nil {
		a.executionReviews.WithExecutionGate(a.acquireSkillReview)
	}
	if a.policyAdoption != nil {
		a.policyAdoption.WithExecutionGate(a.acquireSkillReview)
	}
	if a.executionPolicyAdoption != nil {
		a.executionPolicyAdoption.WithExecutionGate(a.acquireSkillReview)
	}
	return a, nil
}

func (a *productAgentApplication) AuthorizeModelInvocation(ctx context.Context, fact aicapability.InvocationRecord) error {
	if supplyAgentContext(ctx) {
		return a.authorizeSupplyModel(ctx, fact.TenantID, fact.UserID, fact.MemberID)
	}
	var i authidentity.AuthenticatedIdentity
	var err error
	if fact.Operation == aicapability.OperationAIWorkbenchChatPlan {
		i, err = a.freshChatIdentity(ctx)
	} else {
		i, err = a.freshIdentity(ctx)
	}
	if err != nil {
		return err
	}
	if i.TenantID != fact.TenantID || i.UserID != fact.UserID || i.EffectiveMemberID != fact.MemberID {
		return review.ErrForbidden
	}
	return nil
}

func (a *productAgentApplication) freshChatIdentity(ctx context.Context) (authidentity.AuthenticatedIdentity, error) {
	return a.freshWorkbenchIdentity(ctx, authz.PermissionWorkbenchChatUse)
}

func (a *productAgentApplication) freshWorkbenchIdentity(ctx context.Context, permission string) (authidentity.AuthenticatedIdentity, error) {
	return freshWorkbenchIdentity(ctx, a.resolver, a.authorizer, a.config.AllowedOrganizationIDs, permission)
}

func (a *aiWorkbenchApplication) freshWorkbenchIdentity(ctx context.Context, permission string) (authidentity.AuthenticatedIdentity, error) {
	if a.agent != nil {
		return a.agent.freshWorkbenchIdentity(ctx, permission)
	}
	return freshWorkbenchIdentity(ctx, a.resolver, a.authorizer, a.allowedOrganizationIDs, permission)
}
func freshWorkbenchIdentity(ctx context.Context, resolver organizationIdentityResolver, authorizer *authz.ListingKitAuthorizer, allowedOrganizations []string, permission string) (authidentity.AuthenticatedIdentity, error) {
	original, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	capability, bound := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability)
	if !ok || !bound || capability.actorID != original.UserID || capability.effectiveOrganizationID != original.EffectiveOrganizationID ||
		original.TenantID != original.EffectiveOrganizationID || !capability.tokenExpiresAt.After(time.Now()) {
		return authidentity.AuthenticatedIdentity{}, review.ErrForbidden
	}
	if resolver == nil {
		return authidentity.AuthenticatedIdentity{}, review.ErrUnavailable
	}
	identity, err := resolver.Resolve(ctx, httproute.OrganizationAccessPolicyLiveWrite,
		workbenchcontext.ResolveInput{Identity: authidentity.AuthenticatedIdentity{UserID: capability.actorID,
			HomeOrganizationID: capability.homeOrganizationID, TokenExpiresAt: capability.tokenExpiresAt},
			BearerToken: capability.bearerToken, RequestedOrganizationID: capability.effectiveOrganizationID})
	if err != nil {
		if workbenchIdentityRejected(err) {
			return authidentity.AuthenticatedIdentity{}, review.ErrForbidden
		}
		return authidentity.AuthenticatedIdentity{}, review.ErrUnavailable
	}
	if identity.UserID != original.UserID || identity.TenantID != original.TenantID ||
		identity.EffectiveOrganizationID != original.TenantID || !agent.ValidID(identity.EffectiveMemberID) {
		return authidentity.AuthenticatedIdentity{}, review.ErrForbidden
	}
	permissionAllowed, err := authz.AuthorizeOrganization(ctx, authorizer, identity.UserID, identity.EffectiveOrganizationID, identity.Roles, permission)
	if err != nil {
		return authidentity.AuthenticatedIdentity{}, review.ErrUnavailable
	}
	if !permissionAllowed {
		return authidentity.AuthenticatedIdentity{}, review.ErrForbidden
	}
	allowed := false
	for _, org := range allowedOrganizations {
		allowed = allowed || org == identity.TenantID
	}
	if !allowed {
		return authidentity.AuthenticatedIdentity{}, agent.ErrUnavailable
	}
	return identity, nil
}

func (a *productAgentApplication) freshIdentity(ctx context.Context) (authidentity.AuthenticatedIdentity, error) {
	original, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	capability, bound := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability)
	if !ok || !bound || capability.actorID != original.UserID || capability.effectiveOrganizationID != original.EffectiveOrganizationID || original.TenantID != original.EffectiveOrganizationID || !capability.tokenExpiresAt.After(time.Now()) {
		return authidentity.AuthenticatedIdentity{}, review.ErrForbidden
	}
	allowed := false
	for _, org := range a.config.AllowedOrganizationIDs {
		allowed = allowed || org == original.EffectiveOrganizationID
	}
	if !a.config.Enabled || !allowed {
		return authidentity.AuthenticatedIdentity{}, agent.ErrUnavailable
	}
	identity, err := a.resolver.Resolve(ctx, httproute.OrganizationAccessPolicyLiveWrite, workbenchcontext.ResolveInput{Identity: authidentity.AuthenticatedIdentity{UserID: capability.actorID, HomeOrganizationID: capability.homeOrganizationID, TokenExpiresAt: capability.tokenExpiresAt}, BearerToken: capability.bearerToken, RequestedOrganizationID: capability.effectiveOrganizationID})
	if err != nil || identity.UserID != original.UserID || identity.TenantID != original.TenantID || identity.EffectiveOrganizationID != original.TenantID || !agent.ValidID(identity.EffectiveMemberID) || !authz.AllowedOrganization(ctx, a.authorizer, "", identity.EffectiveOrganizationID, identity.Roles, authz.PermissionLocalAgentWrite) || !authz.AllowedOrganization(ctx, a.authorizer, identity.UserID, identity.EffectiveOrganizationID, identity.Roles, authz.PermissionWorkbenchAgentUse) {
		return authidentity.AuthenticatedIdentity{}, review.ErrForbidden
	}
	return identity, nil
}

func validAgentTargetPlatform(platform string) bool {
	// These keys select existing exact Asset inventories, not marketplace
	// publishing capability or a promise that marketplace rules were evaluated.
	return platform == "shein" || platform == "temu" || platform == "amazon"
}

func (a *productAgentApplication) validTargetPlatform(platform string) bool {
	return validAgentTargetPlatform(platform) || a.definition.ID == agentconfig.ContentAgentID && platform == "product"
}

func (a *productAgentApplication) binding(ctx context.Context, operationID, targetPlatform string) (agent.Binding, error) {
	if !a.validTargetPlatform(targetPlatform) {
		return agent.Binding{}, agent.ErrInvalid
	}
	i, err := a.freshIdentity(ctx)
	if err != nil {
		return agent.Binding{}, err
	}
	return a.bindingForIdentity(ctx, i, operationID, targetPlatform)
}

func (a *productAgentApplication) bindingForIdentity(ctx context.Context, i authidentity.AuthenticatedIdentity, operationID, targetPlatform string) (agent.Binding, error) {
	if !a.validTargetPlatform(targetPlatform) {
		return agent.Binding{}, agent.ErrInvalid
	}
	p, err := a.acquisitionSnapshot(ctx, i, operationID)
	if err != nil {
		return agent.Binding{}, err
	}
	return acquisitionBinding(operationID, targetPlatform, p), nil
}

func (a *productAgentApplication) acquisitionSnapshot(ctx context.Context, i authidentity.AuthenticatedIdentity, operationID string) (catalog.PublishedSnapshot, error) {
	p, err := a.receipts.ReadPublished(authidentity.WithAuthenticatedIdentity(ctx, i), operationID)
	if err != nil {
		return catalog.PublishedSnapshot{}, err
	}
	op := p.Result.Operation
	if op.ID != operationID || op.State != sourcing.AcquisitionPublished || op.Scope.OrganizationID != i.TenantID || op.Scope.ActorID != i.UserID || p.Result.Publication == nil {
		return catalog.PublishedSnapshot{}, agent.ErrConflict
	}
	r := p.Result.Publication.Receipt
	if r.OrganizationID != i.TenantID || r.ActorID != i.UserID || r.ProductKey != p.Snapshot.Identity.ProductKey || p.Snapshot.Identity.TenantID != i.TenantID || r.CatalogVersion != p.Snapshot.Version || r.CatalogPublicationID != p.Snapshot.PublicationID || !reflect.DeepEqual(p.Result.Publication.Snapshot, p.Snapshot.Snapshot) {
		return catalog.PublishedSnapshot{}, agent.ErrConflict
	}
	return p.Snapshot, nil
}

func (a *productAgentApplication) Authorize(ctx context.Context, binding agent.Binding) (agent.Scope, error) {
	if binding.ContextKind == "collection" {
		return a.authorizeSupplyBinding(ctx, binding)
	}
	if binding.ContextKind != "acquisition" {
		return agent.Scope{}, agent.ErrInvalid
	}
	i, err := a.freshIdentity(ctx)
	if err != nil {
		return agent.Scope{}, err
	}
	if err = a.authorizeAcquisitionBinding(ctx, i, binding); err != nil {
		return agent.Scope{}, err
	}
	return agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, nil
}

func (a *productAgentApplication) Invoke(ctx context.Context, ref commercetool.ToolRef, meta commercetool.CallMetadata, b agent.Binding) (commercetool.Result, error) {
	if _, err := a.Authorize(ctx, b); err != nil {
		return commercetool.Result{}, err
	}
	if !supplyAgentContext(ctx) {
		i, err := a.freshIdentity(ctx)
		if err != nil {
			return commercetool.Result{}, err
		}
		ctx = authidentity.WithAuthenticatedIdentity(ctx, i)
		capability := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability)
		ctx = commercetoolauth.WithOrganizationRequest(ctx, commercetoolauth.OrganizationRequest{Identity: i, BearerToken: capability.bearerToken, RequestedOrganizationID: i.TenantID})
	}
	ctx = context.WithValue(ctx, productAgentSourceBindingKey{}, b)
	switch ref {
	case canonicalinspect.Definition().Ref:
		return a.canonical.Invoke(ctx, meta, canonicalinspect.Input{ProductKey: b.ProductKey, CatalogVersion: b.CatalogVersion})
	case sourceevidenceinspect.Definition().Ref:
		return a.sources.Invoke(ctx, meta, sourceevidenceinspect.Input{ProductKey: b.ProductKey, CatalogVersion: b.CatalogVersion})
	case assetinspect.Definition().Ref:
		return a.assets.Invoke(ctx, meta, assetinspect.Input{ProductKey: b.ProductKey, CatalogVersion: b.CatalogVersion, TargetPlatform: b.TargetPlatform})
	case readinessinspect.Definition().Ref:
		return a.readiness.Invoke(ctx, meta, readinessinspect.Input{ProductKey: b.ProductKey, CatalogVersion: b.CatalogVersion, TargetPlatform: b.TargetPlatform})
	default:
		return commercetool.Result{}, agent.ErrInvalid
	}
}

func agentReviewInput(b agent.Binding, policy string, candidate enrichment.Candidate) review.CandidateInput {
	version, _ := strconv.ParseUint(b.CatalogVersion, 10, 64)
	return review.CandidateInput{Base: review.CreateInput{ProductKey: b.ProductKey, BaseVersion: version}, PublicationID: b.PublicationID, PolicyVersion: policy, Candidate: candidate}
}

func (a *productAgentApplication) Validate(ctx context.Context, request agent.Request, candidate enrichment.Candidate, history []agent.Observation) (agent.Validation, error) {
	// This consumer remains the current title definition. Content is admitted
	// by its own fixed policy/consumer, never through this title validator.
	if !request.ContentSelection.Absent() {
		return agent.Validation{}, agent.ErrInvalid
	}
	b, policy := request.Binding, request.PolicyVersion
	if _, err := a.Authorize(ctx, b); err != nil {
		return agent.Validation{}, err
	}
	if supplyAgentContext(ctx) {
		return a.validateSupplyCandidate(ctx, b, policy, candidate, history)
	}
	i, err := a.freshIdentity(ctx)
	if err != nil {
		return agent.Validation{}, err
	}
	if !agentCandidateEvidenceObserved(b, candidate, history) {
		return agent.Validation{PolicyVersion: policy, Unresolved: []string{"candidate_evidence_not_observed: read canonical source evidence before proposing"}}, nil
	}
	proposal, err := a.reviews.ValidateCandidate(authidentity.WithAuthenticatedIdentity(ctx, i), agentReviewInput(b, policy, candidate))
	if err != nil {
		if errors.Is(err, enrichment.ErrEvidenceInsufficient) || errors.Is(err, enrichment.ErrOutputValidation) || errors.Is(err, enrichment.ErrPolicyRejected) || errors.Is(err, review.ErrInvalid) {
			return agent.Validation{PolicyVersion: policy, Unresolved: []string{"title_or_evidence_invalid"}}, nil
		}
		return agent.Validation{}, err
	}
	return agent.Validation{Valid: proposal.Validation.Valid, PolicyVersion: policy}, nil
}
