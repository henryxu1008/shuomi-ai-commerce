package productsourcing

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	sourcingpersistence "task-processor/internal/integration/persistence/product/sourcing"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

type collectionSourceQuotes struct{ db *gorm.DB }

func NewCollectionSourceQuoteReader(db *gorm.DB) (collection.SourceQuoteReader, error) {
	if db == nil {
		return nil, collection.ErrUnavailable
	}
	return &collectionSourceQuotes{db: db}, nil
}

func (r *collectionSourceQuotes) ReadSourceQuotes(ctx context.Context, scope collection.Scope, source collection.Source) (collection.SourceQuotes, error) {
	if scope.Validate() != nil || source.PublicationID == "" || source.ProductKey == "" || source.Version == 0 {
		return collection.SourceQuotes{}, collection.ErrForbidden
	}
	var result collection.SourceQuotes
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		reader, err := sourcingpersistence.NewTransactionReader(tx, newCatalogBridge)
		if err != nil {
			return err
		}
		publication, err := reader.Read(ctx, scope.OrganizationID, source.PublicationID)
		if err != nil {
			return err
		}
		result, err = projectCollectionQuotes(scope, source, publication)
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return result, err
}

func projectCollectionQuotes(scope collection.Scope, source collection.Source, publication sourcing.PersistedPublication) (collection.SourceQuotes, error) {
	receipt := publication.Receipt
	if receipt.OrganizationID != scope.OrganizationID || receipt.ActorID != scope.ActorID || receipt.ProductKey != source.ProductKey || receipt.PublicationID != source.PublicationID || receipt.CatalogPublicationID != source.PublicationID || receipt.CatalogVersion != source.Version {
		return collection.SourceQuotes{}, collection.ErrUnavailable
	}
	producer := receipt.Producer
	if !(producer.Kind == sourcing.BrowserAcquisitionProducerKind && (producer.Version == "v1" || producer.Version == "v2") || producer.Kind == sourcing.AmazonBrowserProducerKind && producer.Version == "v1") {
		return collection.SourceQuotes{State: "not_applicable"}, nil
	}
	prices, err := sourcing.ProjectSourceQuotes(publication.Envelope.SupplierOrCostFacts.Facts)
	if err != nil {
		return collection.SourceQuotes{}, err
	}
	return collection.SourceQuotes{State: "available", PublicationID: source.PublicationID, Version: source.Version, Prices: prices}, nil
}
