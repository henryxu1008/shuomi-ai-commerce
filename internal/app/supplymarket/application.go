package supplymarketapp

import (
	"context"
	catalogstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/catalog"
	collectionstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/collection"
	reviewstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/review"
	marketstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/supplymarket"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/supplymarket"

	"gorm.io/gorm"
)

type Dependencies struct {
	ProductDB             *gorm.DB
	Authorization         *supplymarket.ContextAuthorizer
	OriginalAuthorization collection.ExecutionAuthorizer
	Collections           *collection.Service
	PrivateStorage        supplymarket.PrivateFileStorage
	Receiver              func(*gorm.DB) (marketstore.Receiver, error)
	Merchants             supplymarket.VerifiedMerchantReader
}
type Application struct {
	Market *supplymarket.Service
	Files  *supplymarket.FileService
}

// NewApplication composes current owners against one Product database. The
// currentapplication/runtime owner installs fresh schemas and grants separately;
// constructing this feature never migrates, seeds or manufactures authority.
func NewApplication(ctx context.Context, d Dependencies) (*Application, error) {
	if ctx == nil || d.ProductDB == nil || d.Authorization == nil || d.OriginalAuthorization == nil || d.Collections == nil || d.PrivateStorage == nil || d.Receiver == nil {
		return nil, supplymarket.ErrUnavailable
	}
	if collectionstore.VerifyReceiverSchema(ctx, d.ProductDB) != nil {
		return nil, supplymarket.ErrUnavailable
	}
	repo, err := marketstore.NewRepository(ctx, d.ProductDB, marketstore.Dependencies{
		Disclosure: func(tx *gorm.DB) (marketstore.DisclosureVerifier, error) {
			return NewDisclosureVerifier(tx, d.OriginalAuthorization)
		},
		Files: func(tx *gorm.DB) (marketstore.FileVerifier, error) {
			return marketstore.NewFileVerifier(tx, d.PrivateStorage)
		},
		Receiver: d.Receiver,
	})
	if err != nil {
		return nil, err
	}
	snapshots, err := catalogstore.NewBoundedSnapshotReader(d.ProductDB, 2<<20)
	if err != nil {
		return nil, err
	}
	applied, err := reviewstore.NewAppliedPublicationReader(d.ProductDB)
	if err != nil {
		return nil, err
	}
	service, err := supplymarket.NewService(d.Authorization, d.Collections, EffectiveProductReader{snapshots, applied}, repo, d.Merchants)
	if err != nil {
		return nil, err
	}
	files, err := supplymarket.NewFileService(d.Authorization, d.PrivateStorage, repo)
	if err != nil {
		return nil, err
	}
	return &Application{service, files}, nil
}
