package listingadmin

import (
	"gorm.io/gorm"
)

func applyOwnedTenantQuery(db *gorm.DB, tenantID int64, ownerUserID string) *gorm.DB {
	db = db.Where("deleted = 0")
	if tenantID > 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	return applyOwnerScopeForUser(db, db.Statement.Context, ownerUserID, "owner_user_id")
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func findPagedRows[T any](db *gorm.DB, page, pageSize int, rows *[]T) (total int64, normalizedPage int, normalizedPageSize int, err error) {
	normalizedPage, normalizedPageSize = normalizePage(page, pageSize)
	if err = db.Count(&total).Error; err != nil {
		return 0, normalizedPage, normalizedPageSize, err
	}
	if err = db.Order("id desc").Offset((normalizedPage - 1) * normalizedPageSize).Limit(normalizedPageSize).Find(rows).Error; err != nil {
		return 0, normalizedPage, normalizedPageSize, err
	}
	return total, normalizedPage, normalizedPageSize, nil
}
