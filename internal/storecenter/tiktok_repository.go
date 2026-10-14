package storecenter

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	tiktok "task-processor/internal/storecenter/tiktokshop"
)

type tiktokAttemptRow struct {
	OrganizationID, AttemptID, StoreID, ActorID, MemberID string
	AppKey, AppRevision, StateHash                        string
	StoreVersion, ConnectionVersion                       int64
	GrantID                                               *string
	GrantGeneration, GrantRevision                        int64
	SellerIdentityHash, State, KeyID, Ciphertext          string
	ExpiresAt, CreatedAt, UpdatedAt                       time.Time
}

func (tiktokAttemptRow) TableName() string { return "workbench_store_tiktok_oauth_attempts" }

type tiktokGrantRow struct {
	OrganizationID, ID, AppKey, AppRevision, SellerIdentityHash string
	Generation, Version                                         int64
	Status, KeyID, Ciphertext                                   string
	AccessExpiresAt, RefreshExpiresAt                           time.Time
	RefreshOperationID                                          *string
	ClaimExpiresAt                                              *time.Time
	CreatedAt, UpdatedAt                                        time.Time
}

func (tiktokGrantRow) TableName() string { return "workbench_store_tiktok_oauth_grants" }

type tiktokBindingRow struct {
	OrganizationID, StoreID, GrantID, AppKey, ShopID, ShopCode, ShopCipher, Market, SellerType, Status string
	Version                                                                                            int64
	CreatedAt, UpdatedAt                                                                               time.Time
}

func (tiktokBindingRow) TableName() string { return "workbench_store_tiktok_shop_bindings" }

func tiktokAttemptValue(row tiktokAttemptRow) TikTokConnectionAttempt {
	value := TikTokConnectionAttempt{OrganizationID: row.OrganizationID, AttemptID: row.AttemptID, StoreID: row.StoreID, ActorID: row.ActorID, MemberID: row.MemberID, AppKey: row.AppKey, AppRevision: row.AppRevision, StateHash: row.StateHash, StoreVersion: row.StoreVersion, ConnectionVersion: row.ConnectionVersion, GrantGeneration: row.GrantGeneration, GrantRevision: row.GrantRevision, SellerIdentityHash: row.SellerIdentityHash, State: row.State, KeyID: row.KeyID, Ciphertext: row.Ciphertext, ExpiresAt: row.ExpiresAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if row.GrantID != nil {
		value.GrantID = *row.GrantID
	}
	return value
}

func tiktokGrantValue(row tiktokGrantRow) tiktok.GrantSnapshot {
	value := tiktok.GrantSnapshot{OrganizationID: row.OrganizationID, ID: row.ID, AppKey: row.AppKey, AppRevision: row.AppRevision, SellerIdentityHash: row.SellerIdentityHash, Generation: row.Generation, Revision: row.Version, Status: tiktok.GrantStatus(row.Status), KeyID: row.KeyID, Ciphertext: row.Ciphertext, AccessExpiresAt: row.AccessExpiresAt, RefreshExpiresAt: row.RefreshExpiresAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if row.RefreshOperationID != nil {
		value.OperationID = *row.RefreshOperationID
	}
	if row.ClaimExpiresAt != nil {
		value.ClaimExpiresAt = *row.ClaimExpiresAt
	}
	return value
}

func tiktokGrantRowValue(value tiktok.GrantSnapshot) tiktokGrantRow {
	row := tiktokGrantRow{OrganizationID: value.OrganizationID, ID: value.ID, AppKey: value.AppKey, AppRevision: value.AppRevision, SellerIdentityHash: value.SellerIdentityHash, Generation: value.Generation, Version: value.Revision, Status: string(value.Status), KeyID: value.KeyID, Ciphertext: value.Ciphertext, AccessExpiresAt: value.AccessExpiresAt, RefreshExpiresAt: value.RefreshExpiresAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
	if value.OperationID != "" {
		row.RefreshOperationID = &value.OperationID
	}
	if !value.ClaimExpiresAt.IsZero() {
		row.ClaimExpiresAt = &value.ClaimExpiresAt
	}
	return row
}

func (r *MemberScopedStoreRepository) tiktokTransaction(ctx context.Context, org string, write bool, operation func(*gorm.DB, StoreMemberAccess) error) error {
	access, err := r.authorize(ctx, org, write)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Transaction(func(tx *gorm.DB) error { return operation(tx, access) })
}

func (r *MemberScopedStoreRepository) BeginTikTokConnection(ctx context.Context, c OfficialConnectionCommand, app tiktok.Application, stateHash string, now time.Time) (result TikTokConnectionAttempt, err error) {
	hash, e := hex.DecodeString(stateHash)
	if !validConnectionCommand(c) || e != nil || len(hash) != 32 || app.AppKey == "" || len(app.AppKey) > 128 || app.Revision == "" || len(app.Revision) > 128 || now.IsZero() {
		return result, ErrNotFound
	}
	err = r.tiktokTransaction(ctx, c.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		store, e := connectionStore(tx, access, c.StoreID)
		if e != nil {
			return e
		}
		if store.Platform != string(PlatformTikTokShop) || store.Region != "US" || store.ExternalStoreID == "" {
			return ErrNotFound
		}
		if store.Version != c.ExpectedStoreVersion {
			return ErrVersionConflict
		}
		var prior tiktokAttemptRow
		if e = tx.Where("organization_id=? AND attempt_id=?", c.OrganizationID, c.AttemptID).Take(&prior).Error; e == nil {
			return ErrAlreadyExists
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		row := tiktokAttemptRow{OrganizationID: c.OrganizationID, AttemptID: c.AttemptID, StoreID: c.StoreID, ActorID: access.ActorID, MemberID: access.MemberID, AppKey: app.AppKey, AppRevision: app.Revision, StateHash: stateHash, StoreVersion: store.Version + 1, State: "created", ExpiresAt: now.UTC().Add(5 * time.Minute), CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
		var binding tiktokBindingRow
		e = tx.Where("organization_id=? AND store_id=?", c.OrganizationID, c.StoreID).Take(&binding).Error
		if e == nil && binding.AppKey == app.AppKey {
			var grantRow tiktokGrantRow
			if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND id=? AND app_key=?", c.OrganizationID, binding.GrantID, app.AppKey).Take(&grantRow).Error; e != nil {
				return e
			}
			grant, e := tiktok.RestoreGrant(tiktokGrantValue(grantRow))
			if e != nil {
				return ErrOfficialConnectionUnavailable
			}
			if e = grant.Revoke(grantRow.Version, now); e != nil {
				return ErrVersionConflict
			}
			// Reauthorization closes use of the previous generation for every
			// Store referencing this seller, including an in-flight refresh.
			next := tiktokGrantRowValue(grant.Snapshot())
			if e = tx.Model(&tiktokGrantRow{}).Where("organization_id=? AND id=? AND version=?", c.OrganizationID, next.ID, grantRow.Version).Select("version", "status", "refresh_operation_id", "claim_expires_at", "updated_at").Updates(&next).Error; e != nil {
				return e
			}
			row.GrantID = &next.ID
			row.GrantGeneration = next.Generation
			row.GrantRevision = next.Version
			row.SellerIdentityHash = next.SellerIdentityHash
		} else if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		var current officialConnectionRow
		e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", c.OrganizationID, c.StoreID).Take(&current).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		if current.Version == math.MaxInt64 {
			return ErrVersionConflict
		}
		if current.AttemptID != "" {
			if e = tx.Model(&tiktokAttemptRow{}).Where("organization_id=? AND attempt_id=?", c.OrganizationID, current.AttemptID).Updates(map[string]any{"state": "rejected", "key_id": "", "ciphertext": "", "updated_at": now.UTC()}).Error; e != nil {
				return e
			}
		}
		next := officialConnectionRow{OrganizationID: c.OrganizationID, StoreID: c.StoreID, AttemptID: c.AttemptID, Version: current.Version + 1, Status: string(ConnectionStatusDisconnected)}
		if errors.Is(e, gorm.ErrRecordNotFound) {
			e = tx.Create(&next).Error
		} else if current.Version == 0 {
			e = tx.Create(&next).Error
		} else {
			e = tx.Model(&officialConnectionRow{}).Where("organization_id=? AND store_id=?", c.OrganizationID, c.StoreID).Updates(map[string]any{"attempt_id": next.AttemptID, "version": next.Version, "status": next.Status, "observed_at": nil}).Error
		}
		if e != nil {
			return e
		}
		row.ConnectionVersion = next.Version
		if e = updateConnectionRef(tx, store, c.AttemptID, access.ActorID, now); e != nil {
			return e
		}
		if e = tx.Create(&row).Error; e != nil {
			return e
		}
		result = tiktokAttemptValue(row)
		return nil
	})
	return result, err
}

func authorizedTikTokAttempt(tx *gorm.DB, access StoreMemberAccess, storeID, attemptID string) (tiktokAttemptRow, officialConnectionRow, error) {
	store, err := connectionStore(tx, access, storeID)
	if err != nil {
		return tiktokAttemptRow{}, officialConnectionRow{}, err
	}
	if store.Platform != string(PlatformTikTokShop) {
		return tiktokAttemptRow{}, officialConnectionRow{}, ErrNotFound
	}
	var connection officialConnectionRow
	if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", access.OrganizationID, storeID).Take(&connection).Error; err != nil {
		return tiktokAttemptRow{}, connection, ErrNotFound
	}
	var attempt tiktokAttemptRow
	if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=? AND attempt_id=?", access.OrganizationID, storeID, attemptID).Take(&attempt).Error; err != nil {
		return attempt, connection, ErrNotFound
	}
	if attempt.ActorID != access.ActorID || attempt.MemberID != access.MemberID || connection.AttemptID != attemptID || connection.Version != attempt.ConnectionVersion || store.ConnectionRef != attemptID || attempt.State == "rejected" {
		return attempt, connection, ErrNotFound
	}
	if attempt.State != "bound" && store.Version != attempt.StoreVersion {
		return attempt, connection, ErrVersionConflict
	}
	return attempt, connection, nil
}

func (r *MemberScopedStoreRepository) ClaimTikTokExchange(ctx context.Context, c CompleteTikTokConnection, stateHash string, now time.Time) (result TikTokConnectionAttempt, dispatch bool, err error) {
	err = r.tiktokTransaction(ctx, c.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		attempt, _, e := authorizedTikTokAttempt(tx, access, c.StoreID, c.AttemptID)
		if e != nil {
			return e
		}
		if attempt.AppKey != c.AppKey || len(stateHash) != 64 || subtle.ConstantTimeCompare([]byte(attempt.StateHash), []byte(stateHash)) != 1 {
			return ErrNotFound
		}
		if attempt.State == "created" {
			if !now.Before(attempt.ExpiresAt) {
				return ErrOfficialAuthorizationRejected
			}
			update := tx.Model(&tiktokAttemptRow{}).Where("organization_id=? AND attempt_id=? AND state='created'", c.OrganizationID, c.AttemptID).Updates(map[string]any{"state": "exchange_dispatched", "updated_at": now.UTC()})
			if update.Error != nil {
				return update.Error
			}
			if update.RowsAffected != 1 {
				return ErrVersionConflict
			}
			attempt.State = "exchange_dispatched"
			attempt.UpdatedAt = now.UTC()
			dispatch = true
		}
		result = tiktokAttemptValue(attempt)
		return nil
	})
	if err != nil {
		dispatch = false
	}
	return result, dispatch, err
}

func sameTikTokAttempt(row tiktokAttemptRow, a TikTokConnectionAttempt) bool {
	grantID := ""
	if row.GrantID != nil {
		grantID = *row.GrantID
	}
	return row.OrganizationID == a.OrganizationID && row.AttemptID == a.AttemptID && row.StoreID == a.StoreID && row.ActorID == a.ActorID && row.MemberID == a.MemberID && row.AppKey == a.AppKey && row.AppRevision == a.AppRevision && row.StateHash == a.StateHash && row.StoreVersion == a.StoreVersion && row.ConnectionVersion == a.ConnectionVersion && grantID == a.GrantID && row.GrantGeneration == a.GrantGeneration && row.GrantRevision == a.GrantRevision
}

func (r *MemberScopedStoreRepository) ReadTikTokAttempt(ctx context.Context, org, storeID, attemptID string) (result TikTokConnectionAttempt, err error) {
	err = r.tiktokTransaction(ctx, org, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		row, _, e := authorizedTikTokAttempt(tx, access, storeID, attemptID)
		if e != nil {
			return e
		}
		result = tiktokAttemptValue(row)
		return nil
	})
	return result, err
}

func (r *MemberScopedStoreRepository) SaveTikTokCandidate(ctx context.Context, a TikTokConnectionAttempt, keyID, ciphertext, sellerHash string, now time.Time) error {
	hash, e := hex.DecodeString(sellerHash)
	if e != nil || len(hash) != 32 || hex.EncodeToString(hash) != sellerHash || keyID == "" || len(keyID) > 128 || ciphertext == "" || len(ciphertext) > 16384 {
		return ErrOfficialConnectionUnavailable
	}
	return r.tiktokTransaction(ctx, a.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		row, _, e := authorizedTikTokAttempt(tx, access, a.StoreID, a.AttemptID)
		if e != nil {
			return e
		}
		if !sameTikTokAttempt(row, a) || row.State != "exchange_dispatched" {
			return ErrVersionConflict
		}
		if row.SellerIdentityHash != "" && row.SellerIdentityHash != sellerHash {
			return ErrAlreadyExists
		}
		update := tx.Model(&tiktokAttemptRow{}).Where("organization_id=? AND attempt_id=? AND state='exchange_dispatched'", a.OrganizationID, a.AttemptID).Updates(map[string]any{"state": "credential_received", "key_id": keyID, "ciphertext": ciphertext, "seller_identity_hash": sellerHash, "updated_at": now.UTC()})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrVersionConflict
		}
		return nil
	})
}

func (r *MemberScopedStoreRepository) FailTikTokAttempt(ctx context.Context, a TikTokConnectionAttempt, unknown bool, now time.Time) error {
	return r.tiktokTransaction(ctx, a.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		row, _, e := authorizedTikTokAttempt(tx, access, a.StoreID, a.AttemptID)
		if e != nil {
			return e
		}
		if !sameTikTokAttempt(row, a) || row.State == "bound" {
			return ErrVersionConflict
		}
		state := "rejected"
		if unknown {
			state = "unknown"
		}
		return tx.Model(&tiktokAttemptRow{}).Where("organization_id=? AND attempt_id=?", a.OrganizationID, a.AttemptID).Updates(map[string]any{"state": state, "key_id": "", "ciphertext": "", "updated_at": now.UTC()}).Error
	})
}

func (r *MemberScopedStoreRepository) ReadTikTokSellerGrant(ctx context.Context, a TikTokConnectionAttempt, sellerHash string) (result tiktok.GrantSnapshot, found bool, err error) {
	err = r.tiktokTransaction(ctx, a.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		row, _, e := authorizedTikTokAttempt(tx, access, a.StoreID, a.AttemptID)
		if e != nil {
			return e
		}
		if !sameTikTokAttempt(row, a) || row.State != "credential_received" || row.SellerIdentityHash != sellerHash {
			return ErrNotFound
		}
		var grant tiktokGrantRow
		e = tx.Where("app_key=? AND seller_identity_hash=?", row.AppKey, sellerHash).Take(&grant).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return nil
		}
		if e != nil {
			return e
		}
		if grant.OrganizationID != a.OrganizationID {
			return ErrAlreadyExists
		}
		result = tiktokGrantValue(grant)
		found = true
		return nil
	})
	return result, found, err
}
