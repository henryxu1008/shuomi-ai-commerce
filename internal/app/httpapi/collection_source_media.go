package httpapi

import (
	"context"
	"gorm.io/gorm"
	imageapp "task-processor/internal/app/imageagent"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/ledger/storagecapacity"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	collectionhttp "task-processor/internal/product/collection/httpapi"
	"time"
)

func buildProductCollectionModuleWithSourceMedia(ctx context.Context, db *gorm.DB, dependencies routeAuthDependencies, permissions *authz.ListingKitAuthorizer, cfg *config.Config, supply bool, media *collectionSourceMediaDependencies, marketPODImages ...bool) (kernelmodule.Module, error) {
	if err := configureCollectionSourceMediaTrial(media, cfg); err != nil {
		return nil, err
	}
	service, err := buildProductCollectionService(ctx, db, dependencies, permissions, append([]bool{supply}, marketPODImages...)...)
	if err != nil {
		return nil, err
	}
	if cfg != nil && cfg.ProductCollectionSourceMedia.Enabled {
		if media == nil || media.Storage == nil {
			return nil, collection.ErrUnavailable
		}
		authority, err := collection.NewContextAuthorizer(&productReviewLiveOrganizationAccess{resolver: dependencies.organizationResolver, now: time.Now}, permissions)
		if err != nil {
			return nil, err
		}
		if cfg.StorageCapacity && media.Capacity == nil {
			return nil, collection.ErrUnavailable
		}
		service.WithMedia(productsourcing.SourceMedia{Storage: media.Storage, Authorization: authority, Capacity: media.Capacity, TrialURLs: media.TrialURLs})
	}
	binder := productReviewCapabilityBinder{now: time.Now}
	return productCollectionModule{routes: collectionhttp.Routes(service, binder.Bind), service: service}, nil
}

// Runtime constructs the current storage adapter; HTTP only consumes its port.
type collectionSourceMediaDependencies struct {
	Capacity  *storagecapacity.Meter
	Storage   productsourcing.SourceMediaStorage
	TrialURLs *productsourcing.IsolatedTrialSourceMediaURLPolicy
}

func configureCollectionSourceMediaTrial(media *collectionSourceMediaDependencies, cfg *config.Config) error {
	if cfg == nil || !cfg.ProductCollectionSourceMediaIsolatedTrial {
		if media != nil {
			media.TrialURLs = nil
		}
		return nil
	}
	if !cfg.StorageCapacity || !cfg.ProductCollectionSourceMedia.Enabled || media == nil || media.Storage == nil || media.Capacity == nil {
		return collection.ErrUnavailable
	}
	storage := cfg.ProductCollectionSourceMedia
	if storage.S3.ArtifactMode != "aws" || !storage.S3.UsePathStyle || storage.IsolatedTrialGeneratedURLs {
		return collection.ErrUnavailable
	}
	policy, err := productsourcing.NewIsolatedTrialSourceMediaURLPolicy(storage.PublicBase, storage.S3.Bucket, storage.S3.Endpoint)
	if err != nil {
		return err
	}
	media.TrialURLs = policy
	return nil
}

func WithCollectionSourceMedia(storage productsourcing.SourceMediaStorage) CurrentApplicationOption {
	return func(o *currentApplicationOptions) {
		o.collectionSourceMedias++
		o.collectionSourceMedia = &collectionSourceMediaDependencies{Storage: storage}
	}
}

func newImageSetManualMedia(media *collectionSourceMediaDependencies, authority imageMediaScopeAuthority) asset.ManualImageReader {
	return imageapp.ImageSetManualMedia{Media: productsourcing.SourceMedia{Storage: media.Storage, Authorization: authority, Capacity: media.Capacity, TrialURLs: media.TrialURLs}}
}
