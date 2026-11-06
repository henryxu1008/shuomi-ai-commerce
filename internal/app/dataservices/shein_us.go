package dataservicesapp

import (
	"context"

	"gorm.io/gorm"

	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	sourcestore "task-processor/internal/integration/persistence/product/sourcing"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/sourcing"
)

// NewSHEINUSPublisher composes the same Sourcing and Catalog transaction
// writers as the Amazon publisher. No runtime execution is registered for it.
func NewSHEINUSPublisher(live dataacquisition.LiveAccess) jobstore.SHEINUSPublisher {
	if live == nil {
		return nil
	}
	return func(ctx context.Context, tx *gorm.DB, scope collection.Scope, funding orgresource.ResourceFunding, command sourcing.PublicationCommand, verify bool) (collection.Source, error) {
		store, err := sourcestore.NewTransactionWriter(tx, newCatalogBridge)
		if err != nil {
			return collection.Source{}, err
		}
		authority := publicationAuthority{live: live, job: dataacquisition.Job{Scope: scope, Funding: funding}}
		producer, err := sourcing.NewInternalProducer(authority, store, dataacquisition.SHEINUSProducer())
		if err != nil {
			return collection.Source{}, err
		}
		var receipt sourcing.PublicationReceipt
		if verify {
			receipt, err = producer.Verify(ctx, command)
		} else {
			receipt, err = producer.Publish(ctx, command)
		}
		if err != nil {
			return collection.Source{}, err
		}
		return collection.Source{ProductKey: receipt.ProductKey, PublicationID: receipt.PublicationID, Version: receipt.CatalogVersion, OperationID: command.PublicationID, Kind: collection.SourceSHEINUSData}, nil
	}
}

// ReadSHEINUSProvenance is internal to consumers that already hold the exact
// Collection selection. It is not a data service route.
func (m *Module) ReadSHEINUSProvenance(ctx context.Context, p dataacquisition.Principal, source collection.Source) (dataacquisition.SHEINUSProvenance, error) {
	return dataacquisition.ReadSHEINUSProvenance(ctx, p, source, m.live, m.repo, m.results.store)
}
