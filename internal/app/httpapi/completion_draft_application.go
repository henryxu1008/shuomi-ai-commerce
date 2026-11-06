package httpapi

import (
	"context"
	"gorm.io/gorm"
	listingapp "task-processor/internal/app/listing"
	listinghttp "task-processor/internal/app/listing/httpapi"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	store "task-processor/internal/integration/persistence/listing/completiondraft"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/product/collection"
	"time"
)

func WithListingCompletionDrafts() CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.listingCompletionDrafts++ }
}

type completionDraftModule struct {
	app    *listingapp.Application
	routes []httproute.Descriptor
}

func (completionDraftModule) Name() string                  { return "supply-completion-drafts" }
func (completionDraftModule) Enabled(c *config.Config) bool { return c != nil && c.Workbench.Enabled }
func (m completionDraftModule) Register(r *kernelmodule.Registry) error {
	r.AddRoutes(m.routes...)
	return nil
}
func composeCompletionDrafts(ctx context.Context, db *gorm.DB, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, collections *collection.Service, full *listingapp.Application) (completionDraftModule, error) {
	var empty completionDraftModule
	app := full
	if app == nil {
		core, err := composePreparationCore(ctx, db, deps, permissions, collections)
		if err != nil {
			return empty, err
		}
		app = core.app
	}
	repo, err := store.NewRepository(ctx, db)
	if err != nil {
		return empty, err
	}
	service, err := completiondraft.NewService(app.Sources, collections, app.Preparations, app.Authorization, repo)
	if err != nil {
		return empty, err
	}
	bind := productReviewCapabilityBinder{now: time.Now}
	routes := listinghttp.CompletionDraftRoutes(service, bind.Bind)
	if full == nil {
		routes = append(routes, listinghttp.PreparationRoutes(app, bind.Bind)...)
	}
	return completionDraftModule{app, routes}, nil
}
