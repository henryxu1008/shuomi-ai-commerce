package store

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"task-processor/internal/listingkit/sheinsync"
)

func normalizeSheinSyncPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func sheinSyncQueryPage(query *sheinsync.SheinSyncedProductQuery) (int, int) {
	if query == nil {
		return normalizeSheinSyncPage(0, 0)
	}
	return normalizeSheinSyncPage(query.Page, query.PageSize)
}

func sheinSyncJobQueryPage(query *sheinsync.SheinSyncJobQuery) (int, int) {
	if query == nil {
		return normalizeSheinSyncPage(0, 0)
	}
	return normalizeSheinSyncPage(query.Page, query.PageSize)
}

func sheinActivityCandidateQueryPage(query *sheinsync.SheinActivityCandidateQuery) (int, int) {
	if query == nil {
		return normalizeSheinSyncPage(0, 0)
	}
	if len(query.CandidateIDs) > 0 {
		page := query.Page
		if page <= 0 {
			page = 1
		}
		pageSize := query.PageSize
		if pageSize <= 0 {
			pageSize = len(query.CandidateIDs)
		}
		if pageSize < len(query.CandidateIDs) {
			pageSize = len(query.CandidateIDs)
		}
		return page, pageSize
	}
	return normalizeSheinSyncPage(query.Page, query.PageSize)
}

func sheinEnrollmentRunQueryPage(query *sheinsync.SheinEnrollmentRunQuery) (int, int) {
	if query == nil {
		return normalizeSheinSyncPage(0, 0)
	}
	return normalizeSheinSyncPage(query.Page, query.PageSize)
}

func sheinEnrollmentItemQueryPage(query *sheinsync.SheinEnrollmentItemQuery) (int, int) {
	if query == nil {
		return normalizeSheinSyncPage(0, 0)
	}
	return normalizeSheinSyncPage(query.Page, query.PageSize)
}

func applySheinSyncedProductFilters(db *gorm.DB, query *sheinsync.SheinSyncedProductQuery) *gorm.DB {
	if query == nil {
		return db
	}
	if query.TenantID > 0 {
		db = db.Where("tenant_id = ?", query.TenantID)
	}
	if query.StoreID > 0 {
		db = db.Where("store_id = ?", query.StoreID)
	}
	if query.SKCName != "" {
		db = db.Where("skc_name = ?", query.SKCName)
	}
	if len(query.SKCNames) > 0 {
		db = db.Where("skc_name IN ?", query.SKCNames)
	}
	if query.IsActive != nil {
		db = db.Where("is_active = ?", *query.IsActive)
	}
	return db
}

func applySheinSyncJobFilters(db *gorm.DB, query *sheinsync.SheinSyncJobQuery) *gorm.DB {
	if query == nil {
		return db
	}
	if query.TenantID > 0 {
		db = db.Where("tenant_id = ?", query.TenantID)
	}
	if query.StoreID > 0 {
		db = db.Where("store_id = ?", query.StoreID)
	}
	if query.TriggerMode != nil {
		db = db.Where("trigger_mode = ?", *query.TriggerMode)
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}
	if query.StartedFrom != nil {
		db = db.Where("started_at >= ?", *query.StartedFrom)
	}
	if query.StartedTo != nil {
		db = db.Where("started_at <= ?", *query.StartedTo)
	}
	return db
}

func applySheinActivityCandidateFilters(db *gorm.DB, query *sheinsync.SheinActivityCandidateQuery) *gorm.DB {
	if query == nil {
		return db
	}
	if query.TenantID > 0 {
		db = db.Where("tenant_id = ?", query.TenantID)
	}
	if query.StoreID > 0 {
		db = db.Where("store_id = ?", query.StoreID)
	}
	if query.ActivityType != "" {
		db = db.Where("activity_type = ?", query.ActivityType)
	}
	if query.ActivityKey != "" {
		db = db.Where("activity_key = ?", query.ActivityKey)
	}
	if query.SKCName != "" {
		db = db.Where("skc_name = ?", query.SKCName)
	}
	if query.CandidateVersion != "" {
		db = db.Where("candidate_version = ?", query.CandidateVersion)
	}
	if len(query.CandidateIDs) > 0 {
		db = db.Where("id IN ?", query.CandidateIDs)
	}
	if query.ExecutableOnly {
		db = db.Where("eligibility_status = ?", sheinsync.SheinCandidateEligibilityStatusEligible)
		db = db.Where("review_status IN ?", []sheinsync.SheinCandidateReviewStatus{
			sheinsync.SheinCandidateReviewStatusPendingReview,
			sheinsync.SheinCandidateReviewStatusApproved,
			sheinsync.SheinCandidateReviewStatusAutoQueued,
		})
	}
	return db
}

func applySheinEnrollmentRunFilters(db *gorm.DB, query *sheinsync.SheinEnrollmentRunQuery) *gorm.DB {
	if query == nil {
		return db
	}
	if query.TenantID > 0 {
		db = db.Where("tenant_id = ?", query.TenantID)
	}
	if query.StoreID > 0 {
		db = db.Where("store_id = ?", query.StoreID)
	}
	if query.ActivityType != "" {
		db = db.Where("activity_type = ?", query.ActivityType)
	}
	if query.ActivityKey != "" {
		db = db.Where("activity_key = ?", query.ActivityKey)
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}
	return db
}

func applySheinEnrollmentItemFilters(db *gorm.DB, query *sheinsync.SheinEnrollmentItemQuery) *gorm.DB {
	if query == nil {
		return db
	}
	if query.RunID > 0 {
		db = db.Where("listingkit_shein_activity_enrollment_items.run_id = ?", query.RunID)
	}
	if len(query.CandidateIDs) > 0 {
		db = db.Where("listingkit_shein_activity_enrollment_items.candidate_id IN ?", query.CandidateIDs)
	}
	if query.Status != nil {
		db = db.Where("listingkit_shein_activity_enrollment_items.status = ?", *query.Status)
	}
	if query.TenantID > 0 {
		db = db.Where("runs.tenant_id = ?", query.TenantID)
	}
	if query.StoreID > 0 {
		db = db.Where("runs.store_id = ?", query.StoreID)
	}
	if query.ActivityType != "" {
		db = db.Where("runs.activity_type = ?", query.ActivityType)
	}
	if query.ActivityKey != "" {
		db = db.Where("runs.activity_key = ?", query.ActivityKey)
	}
	if query.RunStatus != nil {
		db = db.Where("runs.status = ?", *query.RunStatus)
	}
	return db
}

func cloneFloat64Ptr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}

func cloneTimePtr(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}

func sheinSyncedProductKey(tenantID, storeID int64, skcName string) string {
	return fmt.Sprintf("%d:%d:%s", tenantID, storeID, skcName)
}

func sheinCandidateKey(record sheinsync.SheinActivityCandidateRecord) string {
	return fmt.Sprintf("%d:%d:%s:%s:%s:%s", record.TenantID, record.StoreID, record.ActivityType, record.ActivityKey, record.SKCName, record.CandidateVersion)
}
