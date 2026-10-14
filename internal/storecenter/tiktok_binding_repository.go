package storecenter

import (
	"context"
	"errors"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	tiktok "task-processor/internal/storecenter/tiktokshop"
)

func (r *MemberScopedStoreRepository) BindTikTokConnection(ctx context.Context, intent TikTokBindingIntent, now time.Time) (result OfficialConnectionView, err error) {
	proposed, e := tiktok.RestoreGrant(intent.Grant)
	if e != nil || !proposed.CanSend(now) || intent.Shop.Region != "US" || intent.Shop.SellerType != "CROSS_BORDER" || intent.Shop.ID == "" || len(intent.Shop.ID) > 128 || intent.Shop.Code == "" || len(intent.Shop.Code) > 128 || intent.Shop.Cipher == "" || len(intent.Shop.Cipher) > 8192 {
		return result, ErrOfficialConnectionUnavailable
	}
	a := intent.Attempt
	err = r.tiktokTransaction(ctx, a.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		attempt, connection, e := authorizedTikTokAttempt(tx, access, a.StoreID, a.AttemptID)
		if e != nil {
			return e
		}
		if !sameTikTokAttempt(attempt, a) || attempt.State != "credential_received" || intent.Grant.OrganizationID != a.OrganizationID || intent.Grant.AppKey != a.AppKey || intent.Grant.AppRevision != a.AppRevision || intent.Grant.SellerIdentityHash != attempt.SellerIdentityHash {
			return ErrVersionConflict
		}
		var store workbenchStoreRecord
		if e = tx.Where("organization_id=? AND id=?", a.OrganizationID, a.StoreID).Take(&store).Error; e != nil {
			return e
		}
		if store.ExternalStoreID != intent.Shop.Code {
			return ErrAlreadyExists
		}
		var original tiktokGrantRow
		e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("app_key=? AND seller_identity_hash=?", a.AppKey, attempt.SellerIdentityHash).Take(&original).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			if intent.ExpectedGrantRevision != 0 || !intent.ReplaceCredential || attempt.GrantID != nil || intent.Grant.Generation != 1 || intent.Grant.Revision != 1 {
				return ErrVersionConflict
			}
			if e = tx.Create(valuePointer(tiktokGrantRowValue(intent.Grant))).Error; e != nil {
				return ErrAlreadyExists
			}
		} else if e != nil {
			return e
		} else {
			if original.OrganizationID != a.OrganizationID {
				return ErrAlreadyExists
			}
			if original.ID != intent.Grant.ID || original.Version != intent.ExpectedGrantRevision {
				return ErrVersionConflict
			}
			if intent.ReplaceCredential {
				if attempt.GrantID == nil || *attempt.GrantID != original.ID || attempt.GrantGeneration != original.Generation || attempt.GrantRevision != original.Version || original.Generation == math.MaxInt64 || original.Version == math.MaxInt64 || intent.Grant.Generation != original.Generation+1 || intent.Grant.Revision != original.Version+1 || intent.Grant.Status != tiktok.GrantActive {
					return ErrVersionConflict
				}
				next := tiktokGrantRowValue(intent.Grant)
				update := tx.Model(&tiktokGrantRow{}).Where("organization_id=? AND id=? AND version=? AND generation=?", a.OrganizationID, original.ID, original.Version, original.Generation).Select("app_revision", "generation", "version", "status", "key_id", "ciphertext", "access_expires_at", "refresh_expires_at", "refresh_operation_id", "claim_expires_at", "updated_at").Updates(&next)
				if update.Error != nil {
					return update.Error
				}
				if update.RowsAffected != 1 {
					return ErrVersionConflict
				}
			} else if intent.Grant != tiktokGrantValue(original) || !proposed.CanSend(now) {
				return ErrVersionConflict
			}
		}
		var prior tiktokBindingRow
		e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", a.OrganizationID, a.StoreID).Take(&prior).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		if prior.ShopID != "" && (prior.ShopID != intent.Shop.ID || prior.ShopCode != intent.Shop.Code) {
			return ErrAlreadyExists
		}
		if prior.Version == math.MaxInt64 {
			return ErrVersionConflict
		}
		binding := tiktokBindingRow{OrganizationID: a.OrganizationID, StoreID: a.StoreID, GrantID: intent.Grant.ID, AppKey: a.AppKey, ShopID: intent.Shop.ID, ShopCode: intent.Shop.Code, ShopCipher: intent.Shop.Cipher, Market: "US", SellerType: "CROSS_BORDER", Status: "active", Version: prior.Version + 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
		if prior.Version == 0 {
			if e = tx.Create(&binding).Error; e != nil {
				return ErrAlreadyExists
			}
		} else {
			binding.CreatedAt = prior.CreatedAt
			if e = tx.Model(&tiktokBindingRow{}).Where("organization_id=? AND store_id=? AND version=?", a.OrganizationID, a.StoreID, prior.Version).Select("grant_id", "app_key", "shop_cipher", "status", "version", "updated_at").Updates(&binding).Error; e != nil {
				return e
			}
		}
		// Canonical credential, binding and staging deletion are one Store UoW.
		if e = tx.Model(&tiktokAttemptRow{}).Where("organization_id=? AND attempt_id=? AND state='credential_received'", a.OrganizationID, a.AttemptID).Updates(map[string]any{"state": "bound", "grant_id": intent.Grant.ID, "grant_generation": intent.Grant.Generation, "grant_revision": intent.Grant.Revision, "key_id": "", "ciphertext": "", "updated_at": now.UTC()}).Error; e != nil {
			return e
		}
		if e = tx.Model(&officialConnectionRow{}).Where("organization_id=? AND store_id=? AND attempt_id=? AND version=?", a.OrganizationID, a.StoreID, a.AttemptID, connection.Version).Updates(map[string]any{"status": string(ConnectionStatusConnected), "observed_at": now.UTC()}).Error; e != nil {
			return e
		}
		connection.Status = string(ConnectionStatusConnected)
		connection.ObservedAt = valuePointer(now.UTC())
		attempt.State = "bound"
		result = tiktokConnectionView(connection, attempt, intent.Grant, binding, now)
		return nil
	})
	return result, err
}

func valuePointer[T any](value T) *T { return &value }

func tiktokBindingValue(row tiktokBindingRow) TikTokShopBinding {
	return TikTokShopBinding{OrganizationID: row.OrganizationID, StoreID: row.StoreID, GrantID: row.GrantID, AppKey: row.AppKey, Shop: tiktok.Shop{ID: row.ShopID, Code: row.ShopCode, Cipher: row.ShopCipher, Region: row.Market, SellerType: row.SellerType}, Status: row.Status, Version: row.Version, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func tiktokConnectionView(connection officialConnectionRow, attempt tiktokAttemptRow, grant tiktok.GrantSnapshot, binding tiktokBindingRow, now time.Time) OfficialConnectionView {
	view := OfficialConnectionView{Platform: PlatformTikTokShop, AuthorizationMethod: OfficialAuthorizationMethod("oauth_callback"), AppID: attempt.AppKey, AppRevision: attempt.AppRevision, AttemptID: connection.AttemptID, State: attempt.State, Status: ConnectionStatus(connection.Status), Version: connection.Version, ObservedAt: connection.ObservedAt, TikTok: &TikTokConnectionDetails{Market: "US", SellerType: "CROSS_BORDER", ShopID: binding.ShopID, GrantGeneration: grant.Generation, GrantRevision: grant.Revision, BindingRevision: binding.Version}}
	if grant.ID != "" {
		view.CredentialExpiresAt = valuePointer(grant.AccessExpiresAt)
		valid, e := tiktok.RestoreGrant(grant)
		if e != nil || !valid.CanSend(now) || binding.Status != "active" {
			view.Status = ConnectionStatusUnavailable
		}
		if !now.Before(grant.AccessExpiresAt) || !now.Before(grant.RefreshExpiresAt) {
			view.Status = ConnectionStatusExpired
		}
	}
	return view
}

func (r *MemberScopedStoreRepository) ReadTikTokGrant(ctx context.Context, org, storeID string) (result tiktok.GrantSnapshot, binding TikTokShopBinding, err error) {
	err = r.tiktokTransaction(ctx, org, false, func(tx *gorm.DB, access StoreMemberAccess) error {
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
		var row tiktokBindingRow
		if e = tx.Where("organization_id=? AND store_id=?", org, storeID).Take(&row).Error; e != nil {
			return ErrOfficialConnectionUnavailable
		}
		var grant tiktokGrantRow
		if e = tx.Where("organization_id=? AND id=? AND app_key=?", org, row.GrantID, row.AppKey).Take(&grant).Error; e != nil {
			return ErrOfficialConnectionUnavailable
		}
		result = tiktokGrantValue(grant)
		binding = tiktokBindingValue(row)
		return nil
	})
	return result, binding, err
}

func (r *MemberScopedStoreRepository) ReadTikTokConnection(ctx context.Context, org, storeID string) (result OfficialConnectionView, err error) {
	err = r.tiktokTransaction(ctx, org, false, func(tx *gorm.DB, access StoreMemberAccess) error {
		store, e := connectionStore(tx, access, storeID)
		if e != nil {
			return e
		}
		if store.Platform != string(PlatformTikTokShop) {
			return ErrNotFound
		}
		var connection officialConnectionRow
		e = tx.Where("organization_id=? AND store_id=?", org, storeID).Take(&connection).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			result = OfficialConnectionView{Platform: PlatformTikTokShop, Status: ConnectionStatusDisconnected, State: "disconnected", TikTok: &TikTokConnectionDetails{Market: "US", SellerType: "CROSS_BORDER"}}
			return nil
		}
		if e != nil {
			return e
		}
		var attempt tiktokAttemptRow
		if e = tx.Where("organization_id=? AND store_id=? AND attempt_id=?", org, storeID, connection.AttemptID).Take(&attempt).Error; e != nil {
			return ErrOfficialConnectionUnavailable
		}
		var binding tiktokBindingRow
		e = tx.Where("organization_id=? AND store_id=?", org, storeID).Take(&binding).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		var grant tiktokGrantRow
		if binding.GrantID != "" {
			if e = tx.Where("organization_id=? AND id=? AND app_key=?", org, binding.GrantID, binding.AppKey).Take(&grant).Error; e != nil {
				return ErrOfficialConnectionUnavailable
			}
		}
		result = tiktokConnectionView(connection, attempt, tiktokGrantValue(grant), binding, time.Now().UTC())
		return nil
	})
	return result, err
}
