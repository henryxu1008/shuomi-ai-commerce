package storecenter

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	tiktok "task-processor/internal/storecenter/tiktokshop"
)

type tiktokRefreshRow struct {
	OrganizationID, GrantID, OperationID, StoreID, ActorID, MemberID string
	ExpectedGeneration, ExpectedGrantRevision                        int64
	InputHash, State                                                 string
	ClaimExpiresAt, CreatedAt, UpdatedAt                             time.Time
}

func (tiktokRefreshRow) TableName() string { return "workbench_store_tiktok_oauth_refresh_operations" }

func (r *MemberScopedStoreRepository) ClaimTikTokRefresh(ctx context.Context, org, storeID, operationID string, expectedGeneration int64, inputHash string, now time.Time) (result TikTokRefreshClaim, err error) {
	hash, e := hex.DecodeString(inputHash)
	if e != nil || len(hash) != 32 || hex.EncodeToString(hash) != inputHash || uuid.Validate(operationID) != nil || expectedGeneration <= 0 {
		return result, ErrNotFound
	}
	err = r.tiktokTransaction(ctx, org, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		store, e := connectionStore(tx, access, storeID)
		if e != nil {
			return e
		}
		if store.Platform != string(PlatformTikTokShop) || store.ConnectionRef == "" {
			return ErrNotFound
		}
		var connection officialConnectionRow
		if e = tx.Where("organization_id=? AND store_id=? AND attempt_id=? AND status=?", org, storeID, store.ConnectionRef, string(ConnectionStatusConnected)).Take(&connection).Error; e != nil {
			return ErrOfficialConnectionUnavailable
		}
		var binding tiktokBindingRow
		if e = tx.Where("organization_id=? AND store_id=? AND status='active'", org, storeID).Take(&binding).Error; e != nil {
			return ErrOfficialConnectionUnavailable
		}
		var row tiktokGrantRow
		if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND id=? AND app_key=?", org, binding.GrantID, binding.AppKey).Take(&row).Error; e != nil {
			return ErrOfficialConnectionUnavailable
		}
		var prior tiktokRefreshRow
		e = tx.Where("organization_id=? AND operation_id=?", org, operationID).Take(&prior).Error
		if e == nil {
			if prior.GrantID != row.ID || prior.StoreID != storeID || prior.ActorID != access.ActorID || prior.MemberID != access.MemberID || prior.ExpectedGeneration != expectedGeneration || prior.InputHash != inputHash {
				return ErrVersionConflict
			}
			result = TikTokRefreshClaim{Grant: tiktokGrantValue(row), Binding: tiktokBindingValue(binding), Fence: tiktok.GrantFence{OrganizationID: org, GrantID: row.ID, OperationID: operationID, Generation: prior.ExpectedGeneration, Revision: prior.ExpectedGrantRevision + 1}}
			return nil
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		if row.Generation != expectedGeneration {
			return ErrVersionConflict
		}
		grant, e := tiktok.RestoreGrant(tiktokGrantValue(row))
		if e != nil {
			return ErrOfficialConnectionUnavailable
		}
		fence, e := grant.ClaimRefresh(row.Version, operationID, now, now.Add(30*time.Second))
		if e != nil {
			return ErrVersionConflict
		}
		operation := tiktokRefreshRow{OrganizationID: org, GrantID: row.ID, OperationID: operationID, StoreID: storeID, ActorID: access.ActorID, MemberID: access.MemberID, ExpectedGeneration: row.Generation, ExpectedGrantRevision: row.Version, InputHash: inputHash, State: "dispatch", ClaimExpiresAt: grant.Snapshot().ClaimExpiresAt, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
		if e = tx.Create(&operation).Error; e != nil {
			return ErrVersionConflict
		}
		next := tiktokGrantRowValue(grant.Snapshot())
		update := tx.Model(&tiktokGrantRow{}).Where("organization_id=? AND id=? AND version=? AND generation=?", org, row.ID, row.Version, row.Generation).Select("version", "status", "refresh_operation_id", "claim_expires_at", "updated_at").Updates(&next)
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrVersionConflict
		}
		result = TikTokRefreshClaim{Grant: grant.Snapshot(), Fence: fence, Binding: tiktokBindingValue(binding), Dispatch: true}
		return nil
	})
	if err != nil {
		result.Dispatch = false
	}
	return result, err
}

func (r *MemberScopedStoreRepository) mutateTikTokRefresh(ctx context.Context, claim TikTokRefreshClaim, now time.Time, mutation func(*tiktok.Grant, tiktokRefreshRow) (string, error)) error {
	return r.tiktokTransaction(ctx, claim.Fence.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		if _, e := connectionStore(tx, access, claim.Binding.StoreID); e != nil {
			return e
		}
		var row tiktokGrantRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND id=?", access.OrganizationID, claim.Fence.GrantID).Take(&row).Error; e != nil {
			return ErrNotFound
		}
		var operation tiktokRefreshRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND grant_id=? AND operation_id=?", access.OrganizationID, row.ID, claim.Fence.OperationID).Take(&operation).Error; e != nil {
			return ErrNotFound
		}
		if operation.StoreID != claim.Binding.StoreID || operation.ActorID != access.ActorID || operation.MemberID != access.MemberID || operation.State != "dispatch" || operation.ExpectedGeneration != claim.Fence.Generation || operation.ExpectedGrantRevision+1 != claim.Fence.Revision {
			return ErrVersionConflict
		}
		grant, e := tiktok.RestoreGrant(tiktokGrantValue(row))
		if e != nil {
			return ErrOfficialConnectionUnavailable
		}
		state, e := mutation(grant, operation)
		if e != nil {
			return ErrVersionConflict
		}
		next := tiktokGrantRowValue(grant.Snapshot())
		update := tx.Model(&tiktokGrantRow{}).Where("organization_id=? AND id=? AND version=? AND generation=?", access.OrganizationID, row.ID, row.Version, row.Generation).Select("generation", "version", "status", "key_id", "ciphertext", "access_expires_at", "refresh_expires_at", "refresh_operation_id", "claim_expires_at", "updated_at").Updates(&next)
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrVersionConflict
		}
		return tx.Model(&tiktokRefreshRow{}).Where("organization_id=? AND grant_id=? AND operation_id=? AND state='dispatch'", access.OrganizationID, row.ID, operation.OperationID).Updates(map[string]any{"state": state, "updated_at": now.UTC()}).Error
	})
}

func (r *MemberScopedStoreRepository) CompleteTikTokRefresh(ctx context.Context, claim TikTokRefreshClaim, result tiktok.GrantSnapshot, now time.Time) error {
	return r.mutateTikTokRefresh(ctx, claim, now, func(grant *tiktok.Grant, _ tiktokRefreshRow) (string, error) {
		original := grant.Snapshot()
		if result.OrganizationID != original.OrganizationID || result.ID != original.ID || result.AppKey != original.AppKey || result.AppRevision != original.AppRevision || result.SellerIdentityHash != original.SellerIdentityHash || result.Generation != original.Generation+1 || result.Revision != original.Revision+1 || result.Status != tiktok.GrantActive {
			return "", ErrVersionConflict
		}
		if e := grant.CompleteRefresh(claim.Fence, result.KeyID, result.Ciphertext, result.AccessExpiresAt, result.RefreshExpiresAt, now); e != nil {
			return "", e
		}
		return "succeeded", nil
	})
}

func (r *MemberScopedStoreRepository) UnknownTikTokRefresh(ctx context.Context, claim TikTokRefreshClaim, now time.Time) error {
	return r.mutateTikTokRefresh(ctx, claim, now, func(grant *tiktok.Grant, _ tiktokRefreshRow) (string, error) {
		return "unknown", grant.MarkRefreshUnknown(claim.Fence, now)
	})
}
