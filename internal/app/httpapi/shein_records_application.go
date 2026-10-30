package httpapi

import (
	"context"
	"errors"
	recordstore "task-processor/internal/app/listingrecordstore"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	"task-processor/internal/listing/record"
	"task-processor/internal/marketplace/shein/draft"
	sheinvalidator "task-processor/internal/marketplace/shein/validator"
	"task-processor/internal/storecenter"
	"time"

	"github.com/sirupsen/logrus"

	"gorm.io/gorm"
)

func buildLocalTrialListingRoutes(currentProductDB *gorm.DB, authorizer *authz.ListingKitAuthorizer, logs func(string) *logrus.Entry) ([]httproute.Descriptor, record.Reader, error) {
	if currentProductDB == nil || authorizer == nil {
		return nil, nil, record.ErrUnavailable
	}
	source, err := catalogstore.NewBoundedSnapshotReader(currentProductDB, record.MaxPayloadBytes)
	if err != nil {
		return nil, nil, err
	}
	assets, err := assetstore.NewBoundedApprovedInventoryReader(currentProductDB, record.MaxPayloadBytes)
	if err != nil {
		return nil, nil, err
	}
	stores, err := storecenter.NewGormStoreRepository(currentProductDB)
	if err != nil {
		return nil, nil, err
	}
	repository, err := recordstore.NewRepository(currentProductDB, authorizer)
	if err != nil {
		return nil, nil, err
	}
	service, err := record.NewService(record.ServiceDependencies{
		Products: source, Assets: assets, Stores: sheinRecordStoreReader{repository: stores}, Records: repository,
		Builder: draft.Builder{LogFor: logs}, Evaluator: sheinvalidator.ExactApprovedAssetValidator{}, Authorizer: authorizer,
		Now: time.Now, RuleRevision: sheinvalidator.DiagnosticRuleVersion, PolicyRevision: sheinvalidator.BindingVersion,
	})
	if err != nil {
		return nil, nil, err
	}
	collection, err := record.NewCollectionService(repository, authorizer)
	if err != nil {
		return nil, nil, err
	}
	diagnostic, err := record.NewDiagnosticService(repository, sheinvalidator.DiagnosticValidator{}, authorizer, sheinvalidator.DiagnosticRuleVersion, sheinvalidator.BindingVersion)
	if err != nil {
		return nil, nil, err
	}
	routes := append(sheinRecordRoutes(service), sheinRecordCollectionRoutes(collection)...)
	routes = append(routes, sheinDiagnosticRoutes(diagnostic)...)
	for i := range routes {
		routes[i].RequestTimeout = record.Timeout + 2*time.Second
	}
	return routes, repository, nil
}

type sheinRecordStoreReader struct{ repository storecenter.Repository }

func (r sheinRecordStoreReader) GetStoreReference(ctx context.Context, organizationID, storeID string) (record.StoreReference, error) {
	stored, err := r.repository.Get(ctx, organizationID, storeID)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return record.StoreReference{}, err
	}
	if errors.Is(err, storecenter.ErrNotFound) {
		return record.StoreReference{}, record.ErrNotFound
	}
	if err != nil {
		return record.StoreReference{}, record.ErrUnavailable
	}
	if stored == nil || stored.OrganizationID() != organizationID || stored.ID() != storeID {
		return record.StoreReference{}, record.ErrNotFound
	}
	return record.StoreReference{OrganizationID: stored.OrganizationID(), StoreID: stored.ID(), Platform: string(stored.Platform())}, nil
}
