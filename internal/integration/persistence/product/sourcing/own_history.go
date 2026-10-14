package sourcingpersistence

import (
	"context"
	"gorm.io/gorm"
	"math"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/sourcing"
)

type ownHistoryReader struct{ repository *repository }

func NewOwnHistoryReader(db *gorm.DB, catalog CatalogBridgeFactory) (sourcing.OwnHistoryReadStore, error) {
	store, err := NewRepository(db, catalog)
	if err != nil {
		return nil, err
	}
	return &ownHistoryReader{store.(*repository)}, nil
}
func (r *ownHistoryReader) Read(ctx context.Context, org, id string) (sourcing.PersistedPublication, error) {
	return r.repository.Read(ctx, org, id)
}
func (r *ownHistoryReader) ListOwnVersions(ctx context.Context, scope sourcing.PublicationScope, key string, maxVersion uint64, query sourcing.OwnHistoryQuery) ([]sourcing.OwnHistoryReference, error) {
	if !authidentity.IsBoundedIdentifier(scope.OrganizationID) || !authidentity.IsBoundedIdentifier(scope.ActorID) || !authidentity.IsBoundedIdentifier(key) || maxVersion == 0 || maxVersion > math.MaxInt64 || query.Before > math.MaxInt64 || query.Version > math.MaxInt64 || query.Limit < 1 || query.Limit > 20 || query.Before != 0 && query.Version != 0 {
		return nil, sourcing.ErrInvalidSourcePublication
	}
	statement := r.repository.db.WithContext(ctx).Table("product_source_publications").Select("publication_id,catalog_version AS version").Where("organization_id=? AND actor_id=? AND product_key=? AND producer_kind='own_product' AND producer_version='v1' AND catalog_version<=?", scope.OrganizationID, scope.ActorID, key, maxVersion)
	if query.Before != 0 {
		statement = statement.Where("catalog_version<?", query.Before)
	}
	limit := query.Limit + 1
	if query.Version != 0 {
		statement = statement.Where("catalog_version=?", query.Version)
		limit = 1
	}
	refs := make([]sourcing.OwnHistoryReference, 0)
	err := statement.Order("catalog_version DESC").Limit(limit).Scan(&refs).Error
	return refs, err
}
