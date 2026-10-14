package collectionpersistence

import (
	"context"
	"gorm.io/gorm"
	"task-processor/internal/product/collection"
)

func (r *Repository) reviseOwn(ctx context.Context, tx *gorm.DB, command collection.Command) (collection.Receipt, error) {
	input, scope := command.Mutation, command.Scope
	if command.Envelope == nil || input.ItemID == "" || input.ExpectedRevision <= 0 || input.ExpectedSourceVersion == 0 || !collection.ValidID(input.ExpectedPublicationID) {
		return collection.Receipt{}, collection.ErrInvalid
	}
	var before itemRow
	found := tx.Raw("SELECT "+itemColumns+" FROM product_collection_items WHERE organization_id=? AND actor_id=? AND id=? AND archived_at IS NULL", scope.OrganizationID, scope.ActorID, input.ItemID).Scan(&before)
	if found.Error != nil {
		return collection.Receipt{}, found.Error
	}
	if found.RowsAffected != 1 {
		return collection.Receipt{}, collection.ErrNotFound
	}
	if _, err := lockBatch(tx, scope, before.BatchID, 0); err != nil {
		return collection.Receipt{}, err
	}
	var item itemRow
	locked := tx.Raw("SELECT "+itemColumns+" FROM product_collection_items WHERE organization_id=? AND actor_id=? AND id=? AND archived_at IS NULL FOR UPDATE", scope.OrganizationID, scope.ActorID, input.ItemID).Scan(&item)
	if locked.Error != nil {
		return collection.Receipt{}, locked.Error
	}
	if locked.RowsAffected != 1 {
		return collection.Receipt{}, collection.ErrNotFound
	}
	if item.SourceKind != "own" {
		return collection.Receipt{}, collection.ErrInvalid
	}
	if item.BatchID != before.BatchID || item.Revision != input.ExpectedRevision || item.OriginalVersion != input.ExpectedSourceVersion || item.PublicationID != input.ExpectedPublicationID {
		return collection.Receipt{}, collection.ErrConflict
	}
	publisher, err := r.own(tx)
	if err != nil {
		return collection.Receipt{}, err
	}
	source, err := publisher.ReviseOwn(ctx, scope, command.OperationID, collection.Source{ProductKey: item.ProductKey, PublicationID: item.PublicationID, Version: item.OriginalVersion, Kind: item.SourceKind}, *command.Envelope)
	if err != nil {
		return collection.Receipt{}, err
	}
	if source.Kind != "own" || source.ProductKey != item.ProductKey || source.PublicationID != command.OperationID || source.Version <= item.OriginalVersion {
		return collection.Receipt{}, collection.ErrUnavailable
	}
	if err := tx.Exec("UPDATE product_collection_items SET publication_id=?,original_version=?,revision=revision+1 WHERE organization_id=? AND actor_id=? AND id=?", source.PublicationID, source.Version, scope.OrganizationID, scope.ActorID, item.ID).Error; err != nil {
		return collection.Receipt{}, err
	}
	if err := tx.Exec("UPDATE product_collection_batches SET revision=revision+1 WHERE organization_id=? AND actor_id=? AND id=?", scope.OrganizationID, scope.ActorID, item.BatchID).Error; err != nil {
		return collection.Receipt{}, err
	}
	return collection.Receipt{OperationID: command.OperationID, BatchID: item.BatchID, ItemID: item.ID, Revision: item.Revision + 1}, nil
}
