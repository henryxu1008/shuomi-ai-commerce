package listingadmin

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

type GormStoreRepository struct {
	db *gorm.DB
}

func NewGormStoreRepository(db *gorm.DB) *GormStoreRepository {
	return &GormStoreRepository{db: db}
}

func AutoMigrateStoreRepository(db *gorm.DB) error {
	if db == nil {
		return errors.New("database is not configured")
	}
	return ensureOwnerAuditColumns(db, (listingStore{}).TableName())
}

func (r *GormStoreRepository) ListStores(ctx context.Context, query StoreQuery) (*StorePage, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("store repository database is not configured")
	}
	db := applyStoreQuery(r.db.WithContext(ctx).Table("listing_store"), query)
	var rows []listingStore
	total, page, pageSize, err := findPagedRows(db, query.Page, query.PageSize, &rows)
	if err != nil {
		return nil, err
	}
	items := make([]Store, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toStore())
	}
	return &StorePage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *GormStoreRepository) GetStore(ctx context.Context, tenantID, id int64) (*Store, error) {
	var row listingStore
	err := takeStoreAccessRow(ctx, r.db.WithContext(ctx).Table("listing_store"), tenantID, id, &row)
	if err != nil {
		return nil, err
	}
	store := row.toStore()
	return &store, nil
}

func (r *GormStoreRepository) UpdateStoreID(ctx context.Context, id int64, storeID string) (*Store, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("store repository database is not configured")
	}
	store, err := r.FindStoreByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := updateStoreAccessRow(ctx, r.db.WithContext(ctx).Table("listing_store"), store.TenantID, id, map[string]any{
		"store_id": storeID,
	}); err != nil {
		return nil, err
	}
	return r.FindStoreByID(ctx, id)
}

func (r *GormStoreRepository) UpdateStoreStatus(ctx context.Context, tenantID, id int64, status int16, remark string) (*Store, error) {
	updates := map[string]any{
		"status": status,
	}
	if updatedBy := requestUserIDFromContext(ctx); updatedBy != "" {
		updates["updater"] = updatedBy
		updates["updated_by"] = updatedBy
	}
	if strings.TrimSpace(remark) != "" {
		updates["remark"] = strings.TrimSpace(remark)
	}
	if err := updateStoreAccessRow(ctx, r.db.WithContext(ctx).Table("listing_store"), tenantID, id, updates); err != nil {
		return nil, err
	}
	return r.GetStore(ctx, tenantID, id)
}

func (r *GormStoreRepository) FindStoreByID(ctx context.Context, id int64) (*Store, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("store repository database is not configured")
	}
	var row listingStore
	err := applyStoreAccessScope(r.db.WithContext(ctx).Table("listing_store"), StoreQuery{}).
		Where("id = ? AND deleted = 0", id).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStoreNotFound
	}
	if err != nil {
		return nil, err
	}
	store := row.toStore()
	return &store, nil
}
