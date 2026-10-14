package httpapi

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"task-processor/internal/agentconfig"
	podapp "task-processor/internal/app/pod"
	podhttp "task-processor/internal/app/pod/httpapi"
	"task-processor/internal/app/productsourcing"
	supplyapp "task-processor/internal/app/supplychain"
	marketapp "task-processor/internal/app/supplymarket"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	"task-processor/internal/imageagent"
	imagestore "task-processor/internal/imageagent/store"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	marketstore "task-processor/internal/integration/persistence/product/supplymarket"
	"task-processor/internal/integration/sds"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/supplymarket"
	markethttp "task-processor/internal/product/supplymarket/httpapi"
	"task-processor/internal/workbenchcontext"
	"time"
)

type SupplyMarketDependencies struct {
	Storage supplymarket.PrivateFileStorage
}
type PODDependencies struct {
	ArtworkOriginGuard  podapp.ArtworkOriginAuthorizer
	SourceApprovalGuard asset.SourceApprovalCommitGuard
	AssetDB             *gorm.DB
	Credentials         sds.CredentialSource
	HTTP                *http.Client
	OSSHosts            []string
	Starter             podapp.Starter
	NewWorker           podapp.WorkerFactory
	Worker              *podapp.Worker
}

func WithSupplyMarket(d SupplyMarketDependencies) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.supplyMarkets++; o.supplyMarket = &d }
}
func WithPOD(d PODDependencies) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.pods++; o.pod = &d }
}

type supplyMarketModule struct {
	routes []httproute.Descriptor
	worker podapp.Worker
}

func (s supplyMarketModule) Name() string                  { return "supply-market" }
func (s supplyMarketModule) Enabled(c *config.Config) bool { return c != nil && c.Workbench.Enabled }
func (s supplyMarketModule) Register(r *kernelmodule.Registry) error {
	r.AddRoutes(s.routes...)
	return nil
}

func buildSupplyMarketModule(ctx context.Context, db *gorm.DB, d SupplyMarketDependencies, p *PODDependencies, deps routeAuthDependencies, a *authz.ListingKitAuthorizer, cfg *config.Config, supplyChain ...bool) (supplyMarketModule, error) {
	var empty supplyMarketModule
	resolver, ok := deps.organizationResolver.(*workbenchcontext.Resolver)
	if !ok || resolver == nil || db == nil || d.Storage == nil || a == nil || cfg == nil || cfg.ListingKit.Zitadel.TenantDirectoryToken == "" {
		return empty, supplymarket.ErrUnavailable
	}
	if len(supplyChain) > 4 {
		return empty, supplymarket.ErrUnavailable
	}
	collections, e := buildProductCollectionService(ctx, db, deps, a, len(supplyChain) >= 1 && supplyChain[0], true, p != nil, len(supplyChain) >= 2 && supplyChain[1], len(supplyChain) >= 3 && supplyChain[2], len(supplyChain) >= 4 && supplyChain[3])
	if e != nil {
		return empty, e
	}
	auth, e := supplymarket.NewContextAuthorizer(&productReviewLiveOrganizationAccess{resolver: resolver, now: time.Now}, a)
	if e != nil {
		return empty, e
	}
	original := supplyapp.OrganizationExecutionAuthorizer{Client: zitadel.NewAuthorizationClient(cfg.ListingKit.Zitadel.AuthorizationAPIURL, &http.Client{Timeout: 5 * time.Second}), ServiceToken: func(context.Context) (string, error) { return cfg.ListingKit.Zitadel.TenantDirectoryToken, nil }, ProjectID: cfg.ListingKit.Zitadel.ProjectID, Permissions: a, OrganizationStatus: resolver.BusinessStatusChecker()}
	app, e := marketapp.NewApplication(ctx, marketapp.Dependencies{ProductDB: db, Authorization: auth, OriginalAuthorization: original, Collections: collections, PrivateStorage: d.Storage, Receiver: func(tx *gorm.DB) (marketstore.Receiver, error) { return productsourcing.NewMarketReceiver(tx, auth) }})
	if e != nil {
		return empty, e
	}
	binder := productReviewCapabilityBinder{now: time.Now}
	result := supplyMarketModule{routes: markethttp.Routes(app.Market, app.Files, binder.Bind)}
	if p != nil {
		if e = assetstore.VerifySourceRuntimePermissions(ctx, p.AssetDB); e != nil {
			return empty, e
		}
		pod, e := podapp.NewApplication(ctx, podapp.Dependencies{ArtworkOriginGuard: p.ArtworkOriginGuard, SourceApprovalGuard: p.SourceApprovalGuard, ProductDB: db, AssetDB: p.AssetDB, Authorization: auth, OriginalAuthorization: original, DesignAuthorization: original, Collections: collections, Credentials: p.Credentials, HTTP: p.HTTP, OSSHosts: p.OSSHosts, Starter: p.Starter, ReceiveProduct: productsourcing.ReceivePOD})
		if e != nil {
			return empty, e
		}
		result.worker, e = p.NewWorker(pod.Processor)
		if e != nil || result.worker == nil {
			return empty, supplymarket.ErrUnavailable
		}
		result.routes = append(result.routes, podhttp.Routes(pod.Service, binder.Bind)...)
	}
	return result, nil
}

func marketPODRoute(path string) bool {
	return path == markethttp.BasePath || strings.HasPrefix(path, markethttp.BasePath+"/") || path == markethttp.AdminPath || strings.HasPrefix(path, markethttp.AdminPath+"/") || path == podhttp.BasePath || strings.HasPrefix(path, podhttp.BasePath+"/")
}
func validateMarketPODDescriptor(r httproute.Descriptor) error {
	expected := append(markethttp.Routes(nil, nil, nil), podhttp.Routes(nil, nil)...)
	for _, e := range expected {
		if r.Method == e.Method && r.Path == e.Path {
			if r.Module != e.Module || r.Permission != e.Permission || r.AuthPolicy != e.AuthPolicy || r.OrganizationAccessPolicy != e.OrganizationAccessPolicy || r.OrganizationTargetResolver != nil || r.RequestTimeout != e.RequestTimeout || r.RejectUnreadRequestBody != e.RejectUnreadRequestBody || r.Handler == nil {
				return errors.New("market/POD route loses admitted authorization boundary")
			}
			return nil
		}
	}
	return errors.New("market/POD route not admitted")
}

// Generated artwork enters Source only through the admitted Supply Market/POD composition.
func buildArtworkOriginGuard(productDB, imageDB, configurationDB *gorm.DB) (*productsourcing.ArtworkOriginGuard, error) {
	snapshots, err := catalogstore.NewBoundedSnapshotReader(productDB, 2<<20)
	if err != nil {
		return nil, err
	}
	configuration, err := configstore.New(configurationDB)
	if err != nil {
		return nil, err
	}
	images := imagestore.NewOrganizationRepository(imageDB)
	facts, ok := images.(imageagent.GenerationFactRepository)
	if !ok {
		return nil, imageagent.ErrCommandBlocked
	}
	return &productsourcing.ArtworkOriginGuard{Snapshots: snapshots, Images: images, Facts: facts, Configuration: configuration}, nil
}

func buildPODArtworkPublisher(ctx context.Context, productDB *gorm.DB, images imageagent.Repository, urls imageagent.DurableAssetPublicURLResolver, configuration agentconfig.ArtworkConfigurationRepository, authorization collection.Authorizer) (imageagent.ArtworkPublisher, error) {
	repository, err := collectionstore.NewRepository(ctx, productDB, func(tx *gorm.DB) (collectionstore.OwnPublisher, error) {
		return productsourcing.NewOwnProductWriter(tx, authorization)
	})
	if err != nil {
		return nil, err
	}
	facts, ok := images.(imageagent.GenerationFactRepository)
	if !ok {
		return nil, imageagent.ErrCommandBlocked
	}
	artifacts, ok := images.(imageagent.SlotExternalEffectV3Repository)
	if !ok {
		return nil, imageagent.ErrCommandBlocked
	}
	return &productsourcing.ArtworkReceiver{Images: images, Facts: facts, Artifacts: artifacts, URLs: urls, Configuration: configuration, Collections: repository, Authorize: authorization}, nil
}
