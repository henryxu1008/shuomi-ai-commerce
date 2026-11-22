package dataservicesapp

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	sourcestore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/sourcing"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/dataacquisition"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/sourcing"
)

type capturedResultReader struct{ store sourcing.PublicationReadStore }

func (m *Module) Provenance(ctx context.Context, p dataacquisition.Principal, source collection.Source) (dataacquisition.Provenance, error) {
	return m.reader.Provenance(ctx, p, source, m.results)
}

// UsageProduct derives display text for one original saved console-visible
// Amazon result. It keeps the result's requested-field projection.
func (m *Module) UsageProduct(ctx context.Context, organization, itemID string) (string, error) {
	scope, err := m.access.Resolve(ctx, collection.PermissionRead)
	if err != nil || scope.OrganizationID != organization || !collection.ValidID(itemID) {
		return "", dataacquisition.ErrNotFound
	}
	job, item, err := m.repo.ReadCaptured(ctx, scope, itemID)
	if err != nil {
		return "", err
	}
	if job.Scope != scope || !collection.ValidID(job.ID) || item.ID != itemID || item.State != "SAVED" ||
		item.Source == nil || item.Source.OperationID != itemID || item.Source.PublicationID != itemID ||
		item.Evidence == nil || item.ASIN != item.Evidence.ASIN {
		return "", dataacquisition.ErrNotFound
	}
	if err = m.results.Verify(ctx, scope, *item.Source, *item.Evidence); err != nil {
		return "", err
	}
	if err = m.live.CheckRead(ctx, dataacquisition.Principal{Scope: scope}); err != nil {
		return "", err
	}
	for _, field := range job.Query.Fields {
		if field == "title" {
			return item.Evidence.Title, nil
		}
	}
	for _, field := range job.Query.Fields {
		if field == "asin" {
			return item.Evidence.ASIN, nil
		}
	}
	return "", dataacquisition.ErrNotFound
}

func transactionResultReader(tx *gorm.DB) (dataacquisition.CapturedResultReader, error) {
	reader, err := sourcestore.NewTransactionReader(tx, newCatalogBridge)
	if err != nil {
		return nil, err
	}
	return capturedResultReader{reader}, nil
}

func (r capturedResultReader) Verify(ctx context.Context, scope collection.Scope, source collection.Source, evidence dataacquisition.Evidence) error {
	if scope.Validate() != nil || source.Kind != "amazon_data" || source.PublicationID != source.OperationID || !collection.ValidID(source.PublicationID) || evidence.Validate() != nil {
		return dataacquisition.ErrInvalid
	}
	p, err := r.store.Read(ctx, scope.OrganizationID, source.PublicationID)
	if err != nil {
		return dataacquisition.ErrUnavailable
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		return dataacquisition.ErrUnavailable
	}
	if p.Receipt.ActorID != scope.ActorID || p.Receipt.OrganizationID != scope.OrganizationID || p.Receipt.ProductKey != source.ProductKey || p.Receipt.PublicationID != source.PublicationID || p.Receipt.CatalogVersion != source.Version || p.Receipt.Producer.Kind != "amazon_data" || p.Envelope.RawReference.Checksum != sourcing.RawSnapshotChecksum(string(raw)) {
		return dataacquisition.ErrUnavailable
	}
	return nil
}
