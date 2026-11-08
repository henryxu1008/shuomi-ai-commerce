package recordpersistence

import (
	"context"
	"time"

	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
)

// ListNewTargetIDs reads the first saved revision, independent of later edits.
func (r *Repository) ListNewTargetIDs(ctx context.Context, scope collection.Scope, stores []string, start, end time.Time, limit int) ([]string, error) {
	if scope.Validate() != nil || len(stores) == 0 || !start.Before(end) || limit < 1 {
		return nil, record.ErrInvalid
	}
	var ids []string
	if err := r.db.WithContext(ctx).Model(&recordRow{}).
		Where("organization_id = ? AND actor_id = ? AND member_id = ? AND store_id IN ? AND revision = 1 AND created_at >= ? AND created_at < ?",
			scope.OrganizationID, scope.ActorID, scope.MemberID, stores, start, end).
		Distinct("target_id").Limit(limit).Pluck("target_id", &ids).Error; err != nil {
		return nil, record.ErrUnavailable
	}
	return ids, nil
}
