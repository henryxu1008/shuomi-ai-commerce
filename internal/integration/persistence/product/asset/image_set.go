package assetpersistence

import (
	"context"
	"database/sql"
	"errors"
	productasset "github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *repository) ColorReworkSaveNotCommitted(ctx context.Context, scope productasset.InventoryScope, source productasset.SourceSelectionRequest, actionID string, expected productasset.ImageInventoryHead) (bool, error) {
	if productasset.ValidateInventoryScope(scope) != nil || scope.TargetPlatform != "product" ||
		scope.SourceSnapshotVersion == 0 || source.EffectiveCatalogVersion != scope.SourceSnapshotVersion ||
		source.TargetPlatform != scope.TargetPlatform || !expected.Valid() ||
		expected.ActionID == "" || expected.ActionID == actionID {
		return false, productasset.ErrInvalidApproval
	}
	notCommitted := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		reader := *r
		reader.db = tx
		_, err := reader.ReadApprovalCommit(ctx, scope.TenantID, actionID)
		if err == nil {
			return nil
		}
		if !errors.Is(err, productasset.ErrApprovedAssetsNotReady) {
			return err
		}
		baseline, err := reader.ReadApprovalCommit(ctx, scope.TenantID, expected.ActionID)
		if err != nil {
			return err
		}
		hash, err := approvalPayloadHash(baseline)
		if err != nil {
			return err
		}
		if hash != expected.PayloadHash || baseline.ProductKey != scope.ProductKey || baseline.TargetPlatform != scope.TargetPlatform ||
			baseline.SourceSnapshotVersion != scope.SourceSnapshotVersion || baseline.ImageSet == nil || baseline.ImageSet.Source != source {
			return productasset.ErrApprovalConflict
		}
		// Head and both receipts are read from one snapshot. The original
		// transaction publishes its head and receipt atomically. Replaying an
		// immutable receipt never makes its action the head again.
		current, err := reader.ReadImageSetInventory(ctx, productasset.InventoryScope{
			TenantID: scope.TenantID, ProductKey: scope.ProductKey, TargetPlatform: scope.TargetPlatform,
		})
		if err != nil {
			return err
		}
		notCommitted = current.Head.Valid() && current.Head.ActionID != "" &&
			current.Head.ActionID != expected.ActionID && current.Head.ActionID != actionID
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return notCommitted && err == nil, err
}

// The successful action receipt is resolved before this function. Its head CAS
// and complete asset set are committed in the same existing Asset transaction.
func compareImageInventoryHead(tx *gorm.DB, commit productasset.ApprovalCommit) error {
	expected := commit.ImageSet.ExpectedHead
	query := tx.Where("tenant_id = ? AND product_key = ? AND target_platform = ?", commit.TenantID, commit.ProductKey, commit.TargetPlatform)
	if expected.ActionID == "" {
		candidate := ApprovedInventoryHeadRecord{TenantID: commit.TenantID, ProductKey: commit.ProductKey, TargetPlatform: commit.TargetPlatform, ActionID: commit.ActionID}
		created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&candidate)
		if created.Error != nil {
			return mapRepositoryError("create selected inventory head", created.Error)
		}
		if created.RowsAffected != 1 {
			return productasset.ErrApprovalConflict
		}
		return nil
	}
	var head ApprovedInventoryHeadRecord
	if err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Take(&head).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return productasset.ErrApprovalConflict
		}
		return mapRepositoryError("lock selected inventory head", err)
	}
	if head.ActionID != expected.ActionID {
		return productasset.ErrApprovalConflict
	}
	var original ApprovalReceiptRecord
	if err := tx.Where("tenant_id = ? AND action_id = ?", commit.TenantID, head.ActionID).Take(&original).Error; err != nil {
		return mapRepositoryError("read selected inventory head receipt", err)
	}
	if original.PayloadHash != expected.PayloadHash {
		return productasset.ErrApprovalConflict
	}
	return nil
}

func (r *repository) ReadImageSetInventory(ctx context.Context, scope productasset.InventoryScope) (productasset.ImageSetInventory, error) {
	if err := productasset.ValidateInventoryScope(scope); err != nil {
		return productasset.ImageSetInventory{}, err
	}
	var head ApprovedInventoryHeadRecord
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND product_key = ? AND target_platform = ?", scope.TenantID, scope.ProductKey, scope.TargetPlatform).Take(&head).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return productasset.ImageSetInventory{Scope: scope, Assets: []productasset.ApprovedAsset{}}, nil
	}
	if err != nil {
		return productasset.ImageSetInventory{}, mapRepositoryError("read selected inventory head", err)
	}
	commit, err := r.ReadApprovalCommit(ctx, scope.TenantID, head.ActionID)
	if err != nil {
		return productasset.ImageSetInventory{}, err
	}
	if commit.ProductKey != scope.ProductKey || commit.TargetPlatform != scope.TargetPlatform {
		return productasset.ImageSetInventory{}, productasset.ErrRepositoryStateInvalid
	}
	hash, err := approvalPayloadHash(commit)
	if err != nil {
		return productasset.ImageSetInventory{}, productasset.ErrRepositoryStateInvalid
	}
	result := productasset.ImageSetInventory{Scope: scope, Assets: []productasset.ApprovedAsset{}, Head: productasset.ImageInventoryHead{ActionID: commit.ActionID, PayloadHash: hash}}
	// The global head protects selection CAS; assets follow the requested
	// source version, using the same exact inventory as downstream consumers.
	if scope.SourceSnapshotVersion == 0 {
		result.ApprovalActionID = commit.ActionID
		result.Assets = commit.Assets
		return result, nil
	}
	actionID, err := r.approvedInventoryAction(ctx, scope)
	if errors.Is(err, productasset.ErrApprovedAssetsNotReady) {
		return result, nil
	}
	if err != nil {
		return productasset.ImageSetInventory{}, err
	}
	inventory, err := r.readApprovedInventory(ctx, scope, actionID)
	if err != nil {
		return productasset.ImageSetInventory{}, err
	}
	result.ApprovalActionID = actionID
	result.Assets = inventory.Assets
	return result, nil
}
