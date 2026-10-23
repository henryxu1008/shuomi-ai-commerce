package catalogpersistence

import (
	"context"
	"gorm.io/gorm"
	"task-processor/internal/product/catalog"
)

// PublicationPresence checks every product stream for an orphan publication.
// This narrow caller-transaction port never turns a current-head miss into absence.
func PublicationPresence(ctx context.Context, tx *gorm.DB, org, id string) (int64, error) {
	if tx == nil {
		return 0, catalog.ErrRepositoryUnavailable
	}
	if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return 0, catalog.ErrRepositoryUnavailable
	}
	var count int64
	err := tx.WithContext(ctx).Model(&SnapshotVersionRecord{}).Where("tenant_id=? AND publication_id=?", org, id).Count(&count).Error
	return count, err
}
