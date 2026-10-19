package listingadmin

import (
	"strings"

	"gorm.io/gorm"
)

func applyStoreQuery(db *gorm.DB, query StoreQuery) *gorm.DB {
	db = db.Where("deleted = 0")
	if query.ReadAccess {
		if query.TenantID > 0 {
			db = db.Where("tenant_id = ?", query.TenantID)
		}
		db = applyStoreReadOwnerScope(db, db.Statement.Context)
	} else {
		db = applyOwnedTenantQuery(db, query.TenantID, strings.TrimSpace(query.OwnerUserID))
	}
	if platform := strings.TrimSpace(query.Platform); platform != "" {
		db = db.Where("LOWER(TRIM(platform)) = ?", strings.ToLower(platform))
	}
	return db
}

func applyStoreAccessScope(db *gorm.DB, query StoreQuery) *gorm.DB {
	if query.TenantID > 0 {
		db = db.Where("tenant_id = ?", query.TenantID)
	}
	return applyOwnerScopeForUser(db, db.Statement.Context, query.OwnerUserID, "owner_user_id")
}
