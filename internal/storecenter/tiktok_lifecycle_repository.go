package storecenter

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	tiktok "task-processor/internal/storecenter/tiktokshop"
)

type tiktokAuthorizationEventRow struct {
	AppKey, NotificationID, OrganizationID, StoreID, GrantID, ShopID string
	Topic                                                            int
	BodyHash                                                         string
	OccurredAt, ProcessedAt                                          time.Time
}

func (tiktokAuthorizationEventRow) TableName() string {
	return "workbench_store_tiktok_authorization_events"
}

// The caller must verify the configured application's raw-body signature first.
// This operation can only close an existing exact Shop binding; no actor/org
// from the event can introduce or expand authorization.
func (r *MemberScopedStoreRepository) ApplyTikTokDeauthorization(ctx context.Context, event tiktok.Deauthorization, now time.Time) error {
	hash, e := hex.DecodeString(event.BodyHash)
	if e != nil || len(hash) != 32 || event.AppKey == "" || len(event.AppKey) > 128 || event.NotificationID == "" || len(event.NotificationID) > 128 || event.ShopID == "" || len(event.ShopID) > 128 || event.OccurredAt.IsZero() || now.IsZero() {
		return ErrNotFound
	}
	return r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Transaction(func(tx *gorm.DB) error {
		var located tiktokBindingRow
		e := tx.Where("app_key=? AND shop_id=?", event.AppKey, event.ShopID).Take(&located).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return nil
		}
		if e != nil {
			return e
		}
		if e = lockMemberStore(tx, located.OrganizationID, located.StoreID); e != nil {
			return e
		}
		var grantRow tiktokGrantRow
		if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND id=? AND app_key=?", located.OrganizationID, located.GrantID, event.AppKey).Take(&grantRow).Error; e != nil {
			return ErrNotFound
		}
		var binding tiktokBindingRow
		if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=? AND app_key=? AND shop_id=? AND grant_id=?", located.OrganizationID, located.StoreID, event.AppKey, event.ShopID, grantRow.ID).Take(&binding).Error; e != nil {
			return ErrNotFound
		}
		var prior tiktokAuthorizationEventRow
		e = tx.Where("app_key=? AND notification_id=?", event.AppKey, event.NotificationID).Take(&prior).Error
		if e == nil {
			if prior.BodyHash != event.BodyHash || prior.ShopID != event.ShopID {
				return ErrVersionConflict
			}
			return nil
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		if binding.Version == math.MaxInt64 {
			return ErrVersionConflict
		}
		grant, e := tiktok.RestoreGrant(tiktokGrantValue(grantRow))
		if e != nil {
			return ErrOfficialConnectionUnavailable
		}
		if e = grant.FenceAccess(grantRow.Version, now); e != nil {
			return ErrVersionConflict
		}
		next := tiktokGrantRowValue(grant.Snapshot())
		if e = tx.Model(&tiktokGrantRow{}).Where("organization_id=? AND id=? AND version=?", grantRow.OrganizationID, grantRow.ID, grantRow.Version).Select("version", "status", "updated_at").Updates(&next).Error; e != nil {
			return e
		}
		if e = tx.Model(&tiktokBindingRow{}).Where("organization_id=? AND store_id=? AND version=?", binding.OrganizationID, binding.StoreID, binding.Version).Updates(map[string]any{"status": "revoked", "version": binding.Version + 1, "updated_at": now.UTC()}).Error; e != nil {
			return e
		}
		if e = tx.Model(&officialConnectionRow{}).Where("organization_id=? AND store_id=?", binding.OrganizationID, binding.StoreID).Update("status", string(ConnectionStatusUnavailable)).Error; e != nil {
			return e
		}
		receipt := tiktokAuthorizationEventRow{AppKey: event.AppKey, NotificationID: event.NotificationID, OrganizationID: binding.OrganizationID, StoreID: binding.StoreID, GrantID: grantRow.ID, ShopID: event.ShopID, Topic: 6, BodyHash: event.BodyHash, OccurredAt: event.OccurredAt.UTC(), ProcessedAt: now.UTC()}
		return tx.Create(&receipt).Error
	})
}

func (r *MemberScopedStoreRepository) DisconnectTikTokConnection(ctx context.Context, c OfficialConnectionCommand, now time.Time) (result OfficialConnectionView, err error) {
	if !validConnectionCommand(c) {
		return result, ErrNotFound
	}
	err = r.tiktokTransaction(ctx, c.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		store, e := connectionStore(tx, access, c.StoreID)
		if e != nil {
			return e
		}
		if store.Platform != string(PlatformTikTokShop) {
			return ErrNotFound
		}
		if store.ConnectionRef == "" {
			result = OfficialConnectionView{Platform: PlatformTikTokShop, Status: ConnectionStatusDisconnected, State: "disconnected", TikTok: &TikTokConnectionDetails{Market: "US", SellerType: "CROSS_BORDER"}}
			return nil
		}
		if store.Version != c.ExpectedStoreVersion {
			return ErrVersionConflict
		}
		var binding tiktokBindingRow
		e = tx.Where("organization_id=? AND store_id=?", c.OrganizationID, c.StoreID).Take(&binding).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		if binding.Version == math.MaxInt64 {
			return ErrVersionConflict
		}
		if binding.Version > 0 {
			if e = tx.Model(&tiktokBindingRow{}).Where("organization_id=? AND store_id=? AND version=?", c.OrganizationID, c.StoreID, binding.Version).Updates(map[string]any{"status": "disconnected", "version": binding.Version + 1, "updated_at": now.UTC()}).Error; e != nil {
				return e
			}
		}
		var connection officialConnectionRow
		if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", c.OrganizationID, c.StoreID).Take(&connection).Error; e != nil {
			return e
		}
		if connection.Version == math.MaxInt64 {
			return ErrVersionConflict
		}
		if e = tx.Model(&tiktokAttemptRow{}).Where("organization_id=? AND attempt_id=?", c.OrganizationID, connection.AttemptID).Updates(map[string]any{"state": "rejected", "key_id": "", "ciphertext": "", "updated_at": now.UTC()}).Error; e != nil {
			return e
		}
		if e = tx.Model(&officialConnectionRow{}).Where("organization_id=? AND store_id=? AND version=?", c.OrganizationID, c.StoreID, connection.Version).Updates(map[string]any{"status": string(ConnectionStatusDisconnected), "version": connection.Version + 1, "observed_at": nil}).Error; e != nil {
			return e
		}
		if e = updateConnectionRef(tx, store, "", access.ActorID, now); e != nil {
			return e
		}
		result = OfficialConnectionView{Platform: PlatformTikTokShop, AuthorizationMethod: OfficialAuthorizationMethod("oauth_callback"), AttemptID: connection.AttemptID, State: "disconnected", Status: ConnectionStatusDisconnected, Version: connection.Version + 1, TikTok: &TikTokConnectionDetails{Market: "US", SellerType: "CROSS_BORDER", ShopID: binding.ShopID, BindingRevision: binding.Version + 1}}
		return nil
	})
	return result, err
}

var _ TikTokConnectionStore = (*MemberScopedStoreRepository)(nil)
var _ TikTokLifecycleStore = (*MemberScopedStoreRepository)(nil)
