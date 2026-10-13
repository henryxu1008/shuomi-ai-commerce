package store

import (
	"context"

	"task-processor/internal/listingkit/sheinsync"
)

func (r *GormSheinSyncRepository) CountSheinEnrollmentSummary(ctx context.Context, tenantID, storeID int64, activityType string) (syncedProductCount, missingCostCount, pendingReviewCount, readyToEnrollCount int, err error) {
	var productCounts struct {
		SyncedProductCount int
		MissingCostCount   int
	}
	if err := r.db.WithContext(ctx).
		Model(&sheinsync.SheinSyncedProductRecord{}).
		Select(`
			COUNT(*) AS synced_product_count,
			COALESCE(SUM(CASE WHEN effective_cost_price IS NULL THEN 1 ELSE 0 END), 0) AS missing_cost_count
		`).
		Where("tenant_id = ? AND store_id = ? AND is_active = ?", tenantID, storeID, true).
		Scan(&productCounts).Error; err != nil {
		return 0, 0, 0, 0, err
	}

	latestCandidates := r.db.WithContext(ctx).
		Model(&sheinsync.SheinActivityCandidateRecord{}).
		Select(`
			skc_name,
			eligibility_status,
			review_status,
			ROW_NUMBER() OVER (
				PARTITION BY skc_name
				ORDER BY updated_at DESC, created_at DESC, id DESC
			) AS rn
		`).
		Where("tenant_id = ? AND store_id = ? AND activity_type = ?", tenantID, storeID, activityType)

	var candidateCounts struct {
		PendingReviewCount int
		ReadyToEnrollCount int
	}
	if err := r.db.WithContext(ctx).
		Table("(?) AS latest_candidates", latestCandidates).
		Select(`
			COALESCE(SUM(CASE WHEN rn = 1 AND review_status = ? THEN 1 ELSE 0 END), 0) AS pending_review_count,
			COALESCE(SUM(CASE WHEN rn = 1 AND eligibility_status = ? AND review_status IN ? THEN 1 ELSE 0 END), 0) AS ready_to_enroll_count
		`,
			sheinsync.SheinCandidateReviewStatusPendingReview,
			sheinsync.SheinCandidateEligibilityStatusEligible,
			[]sheinsync.SheinCandidateReviewStatus{
				sheinsync.SheinCandidateReviewStatusApproved,
				sheinsync.SheinCandidateReviewStatusAutoQueued,
			},
		).
		Scan(&candidateCounts).Error; err != nil {
		return 0, 0, 0, 0, err
	}

	return productCounts.SyncedProductCount,
		productCounts.MissingCostCount,
		candidateCounts.PendingReviewCount,
		candidateCounts.ReadyToEnrollCount,
		nil
}

func (r *MemSheinSyncRepository) CountSheinEnrollmentSummary(_ context.Context, tenantID, storeID int64, activityType string) (syncedProductCount, missingCostCount, pendingReviewCount, readyToEnrollCount int, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	active := true
	productQuery := &sheinsync.SheinSyncedProductQuery{
		TenantID: tenantID,
		StoreID:  storeID,
		IsActive: &active,
	}
	for _, product := range r.products {
		if !matchesSheinSyncedProductQuery(product, productQuery) {
			continue
		}
		syncedProductCount++
		if product.EffectiveCostPrice == nil {
			missingCostCount++
		}
	}

	candidateQuery := &sheinsync.SheinActivityCandidateQuery{
		TenantID:     tenantID,
		StoreID:      storeID,
		ActivityType: activityType,
	}
	latestBySKC := make(map[string]sheinsync.SheinActivityCandidateRecord)
	for _, candidate := range r.candidates {
		if !matchesSheinActivityCandidateQuery(candidate, candidateQuery) {
			continue
		}
		current, ok := latestBySKC[candidate.SKCName]
		if !ok || isNewerSheinSummaryCandidate(candidate, current) {
			latestBySKC[candidate.SKCName] = candidate
		}
	}
	for _, candidate := range latestBySKC {
		if candidate.ReviewStatus == sheinsync.SheinCandidateReviewStatusPendingReview {
			pendingReviewCount++
		}
		if candidate.EligibilityStatus == sheinsync.SheinCandidateEligibilityStatusEligible &&
			(candidate.ReviewStatus == sheinsync.SheinCandidateReviewStatusApproved || candidate.ReviewStatus == sheinsync.SheinCandidateReviewStatusAutoQueued) {
			readyToEnrollCount++
		}
	}
	return syncedProductCount, missingCostCount, pendingReviewCount, readyToEnrollCount, nil
}

func isNewerSheinSummaryCandidate(left, right sheinsync.SheinActivityCandidateRecord) bool {
	switch {
	case left.UpdatedAt.After(right.UpdatedAt):
		return true
	case left.UpdatedAt.Before(right.UpdatedAt):
		return false
	case left.CreatedAt.After(right.CreatedAt):
		return true
	case left.CreatedAt.Before(right.CreatedAt):
		return false
	default:
		return left.ID > right.ID
	}
}
