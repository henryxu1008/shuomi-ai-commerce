package collectionpersistence

import (
	"context"
	"errors"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"gorm.io/gorm"
	"sort"
)

// VisibleMarketItems consumes caller-authorized, recipient-private receipts.
// Current batch membership, not the receipt's historical batch, owns visibility.
func VisibleMarketItems(ctx context.Context, tx *gorm.DB, scope collection.Scope, refs []collection.Receipt, lock bool) (map[string]collection.Receipt, error) {
	result := map[string]collection.Receipt{}
	if ctx == nil || tx == nil || scope.Validate() != nil || len(refs) > 1000 {
		return nil, collection.ErrInvalid
	}
	if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return nil, collection.ErrUnavailable
	}
	expected := map[string]collection.Receipt{}
	ids := []string{}
	for _, ref := range refs {
		if !collection.ValidID(ref.ItemID) || !collection.ValidID(ref.OperationID) {
			return nil, collection.ErrInvalid
		}
		if previous, ok := expected[ref.ItemID]; ok && previous.OperationID != ref.OperationID {
			return nil, collection.ErrUnavailable
		}
		expected[ref.ItemID] = ref
		ids = append(ids, ref.ItemID)
	}
	if len(ids) == 0 {
		return result, nil
	}
	type row struct {
		ID, BatchID, SourceOperationID string
		Revision                       int64
	}
	rows := []row{}
	query := tx.WithContext(ctx).Raw(`SELECT i.id,i.batch_id,i.source_operation_id,b.revision FROM product_collection_items i JOIN product_collection_batches b ON b.organization_id=i.organization_id AND b.actor_id=i.actor_id AND b.id=i.batch_id AND b.member_id=i.member_id WHERE i.organization_id=? AND i.actor_id=? AND i.member_id=? AND i.source_kind='market' AND i.archived_at IS NULL AND b.archived_at IS NULL AND i.id IN ?`, scope.OrganizationID, scope.ActorID, scope.MemberID, ids)
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].BatchID == rows[j].BatchID {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].BatchID < rows[j].BatchID
	})
	for _, item := range rows {
		ref := expected[item.ID]
		if ref.OperationID != item.SourceOperationID {
			continue
		}
		if lock {
			batch, e := lockBatch(tx.WithContext(ctx), scope, item.BatchID, 0)
			if errors.Is(e, collection.ErrNotFound) {
				// This batch was visible in the initial join. Its disappearance
				// may follow a move, so absence cannot authorize another receive.
				return nil, collection.ErrConflict
			}
			if e != nil {
				return nil, e
			}
			var current row
			read := tx.WithContext(ctx).Raw(`SELECT id,batch_id,source_operation_id FROM product_collection_items WHERE organization_id=? AND actor_id=? AND member_id=? AND id=? AND source_kind='market' AND archived_at IS NULL FOR UPDATE`, scope.OrganizationID, scope.ActorID, scope.MemberID, item.ID).Scan(&current)
			if read.Error != nil {
				return nil, read.Error
			}
			if read.RowsAffected != 1 {
				continue
			}
			if current.BatchID != item.BatchID {
				return nil, collection.ErrConflict
			}
			if current.SourceOperationID != ref.OperationID {
				return nil, collection.ErrUnavailable
			}
			item.Revision = batch.Revision
		}
		result[item.ID] = collection.Receipt{OperationID: ref.OperationID, ItemID: item.ID, BatchID: item.BatchID, Revision: item.Revision}
	}
	return result, nil
}
