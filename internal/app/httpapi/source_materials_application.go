package httpapi

import (
	"context"
	"time"

	"gorm.io/gorm"
	supplyapp "task-processor/internal/app/supplychain"
	supplyhttp "task-processor/internal/app/supplychain/httpapi"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	kernelmodule "task-processor/internal/kernel/module"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
)

func WithSourceMaterials(db *gorm.DB) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.sourceMaterials++; o.sourceMaterialDB = db }
}

type sourceMaterialsModule struct {
	app    *supplyapp.Application
	routes []httproute.Descriptor
}

func (sourceMaterialsModule) Name() string                  { return "source-materials" }
func (sourceMaterialsModule) Enabled(c *config.Config) bool { return c != nil && c.Workbench.Enabled }
func (m sourceMaterialsModule) Register(r *kernelmodule.Registry) error {
	r.AddRoutes(m.routes...)
	return nil
}

// Both compositions construct the existing Asset approval owner identically;
// only the source selection authority is narrowed for source-only serving.
func composeSourceApprovals(db *gorm.DB, selections asset.SourceSelectionReader) (record.ApprovedAssetReader, *asset.SourceApprovalService, error) {
	assets, err := assetstore.NewBoundedApprovedInventoryReader(db, record.MaxPayloadBytes)
	if err != nil {
		return nil, nil, err
	}
	repo, err := assetstore.NewRepository(db)
	if err != nil {
		return nil, nil, err
	}
	commits, err := assetstore.NewBoundedApprovalCommitReader(db, record.MaxPayloadBytes)
	if err != nil {
		return nil, nil, err
	}
	approvals, err := asset.NewSourceApprovalService(selections, repo, commits)
	return assets, approvals, err
}

func buildSourceMaterialsModule(ctx context.Context, productDB, assetDB *gorm.DB, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, collections *collection.Service, full *supplyapp.Application) (sourceMaterialsModule, error) {
	if err := assetstore.VerifySourceRuntimePermissions(ctx, assetDB); err != nil {
		return sourceMaterialsModule{}, err
	}
	app := full
	if app == nil {
		core, err := composePreparationCore(ctx, productDB, deps, permissions, collections)
		if err != nil {
			return sourceMaterialsModule{}, err
		}
		app = core.app
		app.SourceMaterialsOnly = true
		app.Products = supplyapp.EffectiveProductReader{}
		var e error
		app.Assets, app.Approvals, e = composeSourceApprovals(assetDB, supplyapp.SourceImageReader{Sources: app.Sources, Products: app.Products, Authorization: app.Authorization, OriginalOnly: true})
		if e != nil {
			return sourceMaterialsModule{}, e
		}
	}
	bind := productReviewCapabilityBinder{now: time.Now}
	routes := supplyhttp.SourceMaterialRoutes(app, bind.Bind)
	if full == nil {
		routes = append(routes, supplyhttp.PreparationRoutes(app, bind.Bind)...)
	} else {
		// Full Supply already owns both image routes and Preparation routes.
		routes = routes[len(routes)-1:]
	}
	return sourceMaterialsModule{app: app, routes: routes}, nil
}
