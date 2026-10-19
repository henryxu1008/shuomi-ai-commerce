package listingadmin

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

func storeAccessQuery(ctx context.Context, db *gorm.DB, tenantID int64) *gorm.DB {
	if tenantID > 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	return applyOwnerScope(db, ctx, "owner_user_id")
}

func storeReadAccessQuery(ctx context.Context, db *gorm.DB, tenantID int64) *gorm.DB {
	if tenantID > 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	return applyStoreReadOwnerScope(db, ctx)
}

func applyStoreReadOwnerScope(db *gorm.DB, ctx context.Context) *gorm.DB {
	if db == nil || !ownerScopeEnabled() {
		return db
	}
	if requestHasTenantAdminAccess(ctx) {
		return db
	}
	ownerUserID := requestUserIDFromContext(ctx)
	if ownerUserID == "" {
		return db
	}
	return db.Where("(owner_user_id = ? OR owner_user_id IS NULL OR owner_user_id = '')", ownerUserID)
}

func takeStoreAccessRow(ctx context.Context, db *gorm.DB, tenantID, id int64, row *listingStore) error {
	err := storeReadAccessQuery(ctx, db, tenantID).Where("id = ? AND deleted = 0", id).Take(row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrStoreNotFound
	}
	return err
}

func updateStoreAccessRow(ctx context.Context, db *gorm.DB, tenantID, id int64, updates map[string]any) error {
	res := storeAccessQuery(ctx, db, tenantID).Where("id = ? AND deleted = 0", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrStoreNotFound
	}
	return nil
}
