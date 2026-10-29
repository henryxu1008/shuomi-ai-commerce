package collectionpersistence

import (
	"context"
	"gorm.io/gorm"
	"task-processor/internal/product/collection"
	"time"
)

type CaptureReference struct {
	ID, BatchID, MemberID, ProductKey, PublicationID, SourceKind, SourceOperationID string
	OriginalVersion                                                                 uint64
	ArchivedAt, BatchArchivedAt                                                     *time.Time
}

// ReadCaptureReference includes archived items and their current batch. It is
// internal to capture recovery and does not change normal asset read policy.
func ReadCaptureReference(ctx context.Context, tx *gorm.DB, scope collection.Scope, publication string) (CaptureReference, bool, error) {
	var result CaptureReference
	if tx == nil || scope.Validate() != nil || !collection.ValidID(publication) {
		return result, false, collection.ErrInvalid
	}
	if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return result, false, collection.ErrUnavailable
	}
	id := collection.StableID(scope.OrganizationID, scope.ActorID, publication, "item")
	row := tx.WithContext(ctx).Raw(`SELECT i.id,i.batch_id,i.member_id,i.product_key,i.publication_id,i.original_version,i.source_kind,i.source_operation_id,i.archived_at,b.archived_at AS batch_archived_at
 FROM product_collection_items i LEFT JOIN product_collection_batches b ON b.organization_id=i.organization_id AND b.actor_id=i.actor_id AND b.id=i.batch_id
 WHERE i.organization_id=? AND i.actor_id=? AND i.id=?`, scope.OrganizationID, scope.ActorID, id).Scan(&result)
	if row.Error != nil {
		return result, false, row.Error
	}
	if row.RowsAffected == 0 {
		return result, false, nil
	}
	var batches int64
	if err := tx.Model(&struct{}{}).Table("product_collection_batches").Where("organization_id=? AND actor_id=? AND id=?", scope.OrganizationID, scope.ActorID, result.BatchID).Count(&batches).Error; err != nil {
		return result, false, err
	}
	if batches != 1 {
		return result, false, collection.ErrUnavailable
	}
	return result, true, nil
}
