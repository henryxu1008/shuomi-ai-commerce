package httpapi

import (
	"context"
	"errors"
	"net/http"
	listingapp "task-processor/internal/app/listing"
	listinghttp "task-processor/internal/app/listing/httpapi"
	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	completionstore "task-processor/internal/integration/persistence/listing/completiondraft"
	officialstore "task-processor/internal/integration/persistence/listing/official"
	prepstore "task-processor/internal/integration/persistence/listing/preparation"
	submissionstore "task-processor/internal/integration/persistence/listing/submission"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	temurules "task-processor/internal/marketplace/temu/rules"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
	"task-processor/internal/storecenter"
	"task-processor/internal/workbenchcontext"
	"time"

	"gorm.io/gorm"
)

type ListingDependencies struct {
	AssetDB   *gorm.DB
	Starter   listingapp.OperationStarter
	NewWorker listingapp.OperationWorkerFactory
	Worker    *listingapp.OperationWorker
}

func WithListing(d ListingDependencies) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.listings++; o.listing = &d }
}

type listingModule struct {
	executionAuthorization listingapp.OrganizationExecutionAuthorizer
	app                    *listingapp.Application
	worker                 listingapp.OperationWorker
	routes                 []httproute.Descriptor
}

func (s listingModule) Name() string                  { return "supply-chain" }
func (s listingModule) Enabled(c *config.Config) bool { return c != nil && c.Workbench.Enabled }
func (s listingModule) Register(r *kernelmodule.Registry) error {
	r.AddRoutes(s.routes...)
	return nil
}

func buildListingModule(ctx context.Context, productDB, storeDB *gorm.DB, d ListingDependencies, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, apps *storeapp.OfficialApplicationRegistry, cfg *config.Config, productAgent *productAgentApplication, capabilities storecenter.RuntimeCapabilities, marketPOD ...bool) (listingModule, error) {
	var empty listingModule
	resolver, ok := deps.organizationResolver.(*workbenchcontext.Resolver)
	if !ok || resolver == nil || permissions == nil || d.AssetDB == nil || d.Starter == nil || d.NewWorker == nil || d.Worker == nil || apps == nil || productDB == nil || storeDB == nil || cfg == nil || cfg.ListingKit.Zitadel.TenantDirectoryToken == "" {
		return empty, preparation.ErrUnavailable
	}
	if err := assetstore.VerifySourceRuntimePermissions(ctx, d.AssetDB); err != nil {
		return empty, err
	}
	if err := storecenter.VerifyCurrentSchemaForCapabilities(ctx, storeDB, capabilities); err != nil {
		return empty, err
	}
	if err := storecenter.VerifyRuntimePermissionsForCapabilities(ctx, storeDB, capabilities); err != nil {
		return empty, err
	}
	core, err := buildNativeDraftReadCore(ctx, productDB, deps, permissions, nil, marketPOD...)
	if err != nil {
		return empty, err
	}
	collections, auth := core.collections, core.app.Authorization
	preparations, sources, effective, records := core.app.Preparations, core.app.Sources, core.app.Products, core.app.Records
	executionAuth := listingapp.OrganizationExecutionAuthorizer{Client: zitadel.NewAuthorizationClient(cfg.ListingKit.Zitadel.AuthorizationAPIURL, &http.Client{Timeout: 5 * time.Second}), ServiceToken: func(context.Context) (string, error) { return cfg.ListingKit.Zitadel.TenantDirectoryToken, nil }, ProjectID: cfg.ListingKit.Zitadel.ProjectID, Permissions: permissions, OrganizationStatus: resolver.BusinessStatusChecker()}
	sources, err = sources.WithExecution(executionAuth, collection.ExecutionOwnerAuthority{Authorization: executionAuth})
	if err != nil {
		return empty, err
	}
	operationsRepo, err := prepstore.NewOperationRepository(ctx, productDB)
	if err != nil {
		return empty, err
	}
	operations, err := preparation.NewOperationService(preparations, collections, operationsRepo)
	if err != nil {
		return empty, err
	}
	storeRepo, err := storecenter.NewMemberScopedStoreRepository(storeDB, currentStoreMemberAuthorizer{authorizer: permissions})
	if err != nil {
		return empty, err
	}
	operations, err = operations.WithExecution(listingapp.OperationExecutionAuthorization{OrganizationExecutionAuthorizer: executionAuth, Stores: storeRepo})
	if err != nil {
		return empty, err
	}
	access, err := storeapp.NewOfficialProductAccess(storeRepo, executionAuth, apps)
	if err != nil {
		return empty, err
	}
	rules := listingapp.RuleReader{Stores: listingapp.OfficialRuleStore{Access: access}}
	temuCatalog, err := temurules.NewOfficialCatalog()
	if err != nil {
		return empty, err
	}
	temuRules := listingapp.TemuRuleReader{Stores: listingapp.OfficialTemuRuleStore{Access: access}, Catalog: temuCatalog}
	tiktokRules := listingapp.TikTokRuleReader{Stores: listingapp.OfficialTikTokRuleStore{Access: access}}
	walmartStores := listingapp.OfficialWalmartRuleStore{Access: access}
	walmartRules := listingapp.WalmartRuleReader{Stores: walmartStores}
	selections := listingapp.SourceImageReader{Sources: sources, Products: effective, Authorization: auth}
	assets, approvals, err := composeSourceApprovals(d.AssetDB, selections)
	if err != nil {
		return empty, err
	}
	imageProbe := listingapp.NewPublicImageProbe()
	var completionVersions record.TargetCompletionVersionReader
	if len(marketPOD) > 4 && marketPOD[4] {
		versions, e := completionstore.NewRepository(ctx, productDB)
		if e != nil {
			return empty, e
		}
		completionVersions = versions
	}
	targets, err := record.NewTargetService(record.TargetDependencies{Sources: sources, Products: effective, Assets: assets, Rules: rules, TemuRules: temuRules, TemuImages: imageProbe, TikTokRules: tiktokRules, TikTokImages: imageProbe, WalmartRules: walmartRules, WalmartImages: imageProbe, CompletionVersions: completionVersions, Records: records, Images: imageProbe, Authorizer: auth, ExecutionSources: sources, ExecutionAuthorization: executionAuth})
	if err != nil {
		return empty, err
	}
	official, err := officialstore.NewOfficialRepository(ctx, productDB)
	if err != nil {
		return empty, err
	}
	effects, err := submissionstore.NewOfficialEffectsRepository(productDB)
	if err != nil {
		return empty, err
	}
	kernel, err := submission.NewExecutionKernel(effects)
	if err != nil {
		return empty, err
	}
	stageProjection := core.app.StageProjection
	uploader, err := listingapp.NewUploadService(listingapp.UploadDependencies{Sources: sources, Products: effective, Assets: assets, Rules: rules, TemuRules: temuRules, TemuImages: imageProbe, TikTokRules: tiktokRules, TikTokImages: imageProbe, WalmartRules: walmartRules, WalmartImages: imageProbe, TikTokStores: listingapp.OfficialTikTokExecutionStore{Access: access}, WalmartStores: listingapp.OfficialWalmartExecutionStore{Access: access}, WalmartReadStores: walmartStores, Records: records, Authorization: executionAuth, Stores: listingapp.OfficialExecutionStore{Access: access}, TemuStores: listingapp.OfficialTemuExecutionStore{Access: access}, Images: imageProbe, Kernel: kernel, Intents: official, Receipts: official, ReviewGate: stageProjection.RequireUploadReady})
	if err != nil {
		return empty, err
	}
	app := &listingapp.Application{Preparations: preparations, Sources: sources, Operations: operations, Execution: listingapp.OperationApplication{Service: operations, Repository: operationsRepo, Starter: d.Starter}, Targets: targets, Records: records, Products: effective, Rules: rules, TemuRules: temuRules, TikTokRules: tiktokRules, TikTokCatalog: tiktokRules, TikTokStatusStores: listingapp.OfficialTikTokRuleStore{Access: access}, WalmartRules: walmartRules, WalmartCatalog: walmartRules, WalmartStatusStores: walmartStores, WalmartResults: official, Assets: assets, Approvals: approvals, Authorization: auth, PublicationReceipts: official, PublicationStores: listingapp.OfficialRuleStore{Access: access}, TargetMerchants: listingapp.OfficialTargetMerchantStore{Access: access}}
	app.WalmartFeedSource = official
	app.WalmartFeedMerchant = func(ctx context.Context, scope collection.Scope, id string) (storecenter.ProductMerchantBinding, error) {
		return access.ReadSavedWalmartFeedBinding(ctx, storecenter.ProductExecutionSubject{
			OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: storecenter.ProductPurposeRules,
		}, id)
	}
	app.SheinPublishSource = official
	app.SheinPublishMerchant = func(ctx context.Context, scope collection.Scope, id string) (storecenter.ProductMerchantBinding, error) {
		handle, err := access.Authorize(ctx, storecenter.ProductExecutionSubject{
			OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: storecenter.ProductPurposeRules,
		}, id, nil)
		if err != nil {
			return storecenter.ProductMerchantBinding{}, err
		}
		return handle.Binding(), nil
	}
	app.TikTokPublishSource = official
	app.TikTokPublishMerchant = func(ctx context.Context, scope collection.Scope, id string) (submission.TikTokGlobalPublishReadScope, error) {
		shop, err := access.ReadSavedTikTokShopIdentity(ctx, storecenter.ProductExecutionSubject{
			OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: storecenter.ProductPurposeRules,
		}, id)
		return submission.TikTokGlobalPublishReadScope{
			StoreID: id, ApplicationID: shop.ApplicationID, MerchantIdentityHash: shop.MerchantIdentityHash, ShopID: shop.ShopID,
		}, err
	}
	var optimizer listingapp.OperationOptimizer
	app.StageProjection = stageProjection
	app.CockpitStoreAccess = func(ctx context.Context, scope collection.Scope, ids []string) error {
		fresh, err := freshWorkbenchIdentity(ctx, resolver, permissions, []string{scope.OrganizationID}, authz.PermissionWorkbenchStoreRead)
		if errors.Is(err, review.ErrForbidden) {
			return record.ErrForbidden
		}
		if err != nil {
			return record.ErrUnavailable
		}
		if fresh.UserID != scope.ActorID || fresh.TenantID != scope.OrganizationID || fresh.EffectiveMemberID != scope.MemberID {
			return record.ErrForbidden
		}
		allowed, err := authz.AuthorizeOrganization(ctx, permissions, scope.ActorID, scope.OrganizationID, fresh.Roles, collection.PermissionRead)
		if err != nil {
			return record.ErrUnavailable
		}
		if !allowed {
			return record.ErrForbidden
		}
		ctx = authidentity.WithAuthenticatedIdentity(ctx, fresh)
		for _, id := range ids {
			store, err := storeRepo.Get(ctx, scope.OrganizationID, id)
			if errors.Is(err, storecenter.ErrNotFound) {
				return record.ErrForbidden
			}
			if err != nil || store == nil || store.ID() != id {
				return record.ErrUnavailable
			}
		}
		return nil
	}
	app.Uploader = uploader
	app.SheinPublication, err = listingapp.NewSheinPublicationService(listingapp.SheinPublicationDependencies{Sources: sources, Records: records, Authorization: auth, ExecutionAuthorization: executionAuth, Receipts: official, Intents: official, Kernel: kernel, Attempts: effects, Stores: listingapp.OfficialSheinPublicationStore{Access: access}})
	if err != nil {
		return empty, err
	}
	if productAgent != nil {
		bridge, e := connectListingProductAgent(ctx, productAgent.config.ReviewDB, app, productAgent, executionAuth)
		if e != nil {
			return empty, e
		}
		if bridge != nil {
			optimizer = bridge
			app.AuthorizeOptimization = bridge.authorizeRequest
			app.OptimizationOptions = bridge.options
		}
	}
	plansRepo, err := prepstore.NewUploadPlanRepository(ctx, productDB)
	if err != nil {
		return empty, err
	}
	app.UploadPlans, err = preparation.NewUploadPlanService(auth, plansRepo, listingapp.UploadPlanInputs{Application: app, Operations: plansRepo, WalmartAttempts: effects})
	if err != nil {
		return empty, err
	}
	currentWorker, err := d.NewWorker(&listingapp.OperationActivities{Operations: operations, Repository: operationsRepo, Sources: sources, Products: effective, TargetMerchants: listingapp.OfficialTargetMerchantStore{Access: access}, Targets: records, Creator: targets, Uploader: uploader, Optimizer: optimizer})
	if err != nil {
		return empty, err
	}
	binder := productReviewCapabilityBinder{now: time.Now}
	return listingModule{app: app, worker: currentWorker, routes: listinghttp.ListingRoutes(app, binder.Bind), executionAuthorization: executionAuth}, nil
}
func validateListingDescriptor(route httproute.Descriptor) error {
	for _, expected := range append(listinghttp.ListingRoutes(nil, nil), listinghttp.SourceMaterialRoutes(nil, nil)...) {
		if route.Method == expected.Method && route.Path == expected.Path {
			if route.Module != expected.Module || route.Permission != expected.Permission || route.AuthPolicy != expected.AuthPolicy || route.OrganizationAccessPolicy != expected.OrganizationAccessPolicy || route.OrganizationTargetResolver != nil || route.RequestTimeout != expected.RequestTimeout || route.RejectUnreadRequestBody != expected.RejectUnreadRequestBody || route.Handler == nil {
				return errors.New("supply route loses live private source boundary")
			}
			return nil
		}
	}
	return errors.New("supply route not admitted")
}
