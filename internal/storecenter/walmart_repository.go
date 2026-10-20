package storecenter

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	marketplace "task-processor/internal/marketplace/walmart"
	auth "task-processor/internal/storecenter/walmart"
)

func WalmartMerchantIdentity(environment, seller string) string {
	return marketplace.Hash([]string{"walmart", environment, "US", seller})
}
func (r *MemberScopedStoreRepository) walmartTransaction(ctx context.Context, org string, write bool, f func(*gorm.DB, StoreMemberAccess) error) error {
	access, e := r.authorize(ctx, org, write)
	if e != nil {
		return e
	}
	return r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Transaction(func(tx *gorm.DB) error { return f(tx, access) })
}
func walmartStore(tx *gorm.DB, access StoreMemberAccess, id string) (workbenchStoreRecord, error) {
	s, e := connectionStore(tx, access, id)
	if e != nil {
		return s, e
	}
	if s.Platform != "walmart" || s.Region != "US" {
		return s, ErrNotFound
	}
	return s, nil
}
func authorizedWalmartAttempt(tx *gorm.DB, access StoreMemberAccess, storeID, attemptID string) (WalmartConnectionAttempt, officialConnectionRow, error) {
	var a WalmartConnectionAttempt
	var c officialConnectionRow
	s, e := walmartStore(tx, access, storeID)
	if e != nil {
		return a, c, e
	}
	if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", access.OrganizationID, storeID).Take(&c).Error; e != nil {
		return a, c, ErrNotFound
	}
	if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=? AND attempt_id=?", access.OrganizationID, storeID, attemptID).Take(&a).Error; e != nil {
		return a, c, ErrNotFound
	}
	if a.ActorID != access.ActorID || a.MemberID != access.MemberID || c.AttemptID != attemptID || s.ConnectionRef != attemptID || a.ConnectionVersion != c.Version || a.State == "rejected" {
		return a, c, ErrNotFound
	}
	if a.State != "bound" && s.Version != a.StoreVersion {
		return a, c, ErrVersionConflict
	}
	return a, c, nil
}
func (r *MemberScopedStoreRepository) BeginWalmartConnection(ctx context.Context, c OfficialConnectionCommand, app auth.Application, stateHash, nonce string, now time.Time) (out WalmartConnectionAttempt, err error) {
	if !validConnectionCommand(c) || c.ApplicationID != app.ClientID || len(stateHash) != 64 || len(nonce) < 32 || len(nonce) > 128 || app.Revision == "" || app.CallbackURL == "" || app.Environment != "production" && app.Environment != "sandbox" {
		return out, ErrNotFound
	}
	err = r.walmartTransaction(ctx, c.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		s, e := walmartStore(tx, access, c.StoreID)
		if e != nil {
			return e
		}
		if s.Version != c.ExpectedStoreVersion {
			return ErrVersionConflict
		}
		var prior WalmartConnectionAttempt
		e = tx.Where("organization_id=? AND attempt_id=?", c.OrganizationID, c.AttemptID).Take(&prior).Error
		if e == nil {
			return ErrAlreadyExists
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		var conn officialConnectionRow
		e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", c.OrganizationID, c.StoreID).Take(&conn).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		a := WalmartConnectionAttempt{OrganizationID: c.OrganizationID, AttemptID: c.AttemptID, StoreID: c.StoreID, ActorID: access.ActorID, MemberID: access.MemberID, AppID: app.ClientID, AppRevision: app.Revision, Environment: app.Environment, CallbackURL: app.CallbackURL, StateHash: stateHash, Nonce: nonce, StoreVersion: s.Version + 1, ConnectionVersion: conn.Version + 1, State: "created", ExpiresAt: now.Add(5 * time.Minute), CreatedAt: now, UpdatedAt: now}
		var grant WalmartGrant
		e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", c.OrganizationID, c.StoreID).Take(&grant).Error
		if e == nil {
			// Re-consent fences late refresh/callback replies before leaving this txn.
			grant.Version++
			grant.Status = "revoked"
			grant.UpdatedAt = now
			if e = tx.Model(&WalmartGrant{}).Where("organization_id=? AND id=?", grant.OrganizationID, grant.ID).Select("version", "status", "updated_at").Updates(&grant).Error; e != nil {
				return e
			}
			a.GrantID = grant.ID
			a.GrantGeneration = grant.Generation
			a.GrantVersion = grant.Version
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		if conn.AttemptID != "" {
			if e = tx.Model(&WalmartConnectionAttempt{}).Where("organization_id=? AND attempt_id=? AND state<>'bound'", c.OrganizationID, conn.AttemptID).Updates(map[string]any{"state": "rejected", "key_id": "", "ciphertext": "", "updated_at": now}).Error; e != nil {
				return e
			}
		}
		next := officialConnectionRow{OrganizationID: c.OrganizationID, StoreID: c.StoreID, AttemptID: c.AttemptID, Version: a.ConnectionVersion, Status: "disconnected"}
		if conn.Version == 0 {
			e = tx.Create(&next).Error
		} else {
			e = tx.Model(&officialConnectionRow{}).Where("organization_id=? AND store_id=?", c.OrganizationID, c.StoreID).Select("attempt_id", "version", "status", "observed_at").Updates(&next).Error
		}
		if e != nil {
			return e
		}
		if e = updateConnectionRef(tx, s, c.AttemptID, access.ActorID, now); e != nil {
			return e
		}
		if e = tx.Create(&a).Error; e != nil {
			return e
		}
		out = a
		return nil
	})
	return out, err
}
func (r *MemberScopedStoreRepository) ReadWalmartAttempt(ctx context.Context, org, storeID, id string) (out WalmartConnectionAttempt, err error) {
	err = r.walmartTransaction(ctx, org, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		a, _, e := authorizedWalmartAttempt(tx, access, storeID, id)
		out = a
		return e
	})
	return
}
func (r *MemberScopedStoreRepository) ClaimWalmartExchange(ctx context.Context, c CompleteWalmartConnection, hash string, now time.Time) (out WalmartConnectionAttempt, dispatch bool, err error) {
	err = r.walmartTransaction(ctx, c.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		a, _, e := authorizedWalmartAttempt(tx, access, c.StoreID, c.AttemptID)
		if e != nil {
			return e
		}
		if a.AppID != c.AppID || c.CallbackType != "auth" || len(hash) != 64 || subtle.ConstantTimeCompare([]byte(a.StateHash), []byte(hash)) != 1 || c.SellerID == "" || len(c.SellerID) > 200 || a.SellerID != "" && a.SellerID != c.SellerID {
			return ErrNotFound
		}
		if a.State == "created" {
			if !now.Before(a.ExpiresAt) {
				return ErrOfficialAuthorizationRejected
			}
			a.State = "exchange_dispatched"
			a.SellerID = c.SellerID
			a.UpdatedAt = now
			if e = tx.Model(&WalmartConnectionAttempt{}).Where("organization_id=? AND attempt_id=? AND state='created'", a.OrganizationID, a.AttemptID).Select("state", "seller_id", "updated_at").Updates(&a).Error; e != nil {
				return e
			}
			dispatch = true
		}
		out = a
		return nil
	})
	if err != nil {
		dispatch = false
	}
	return
}
func sameWalmartAttempt(a, b WalmartConnectionAttempt) bool {
	return a.OrganizationID == b.OrganizationID && a.AttemptID == b.AttemptID && a.StoreID == b.StoreID && a.ActorID == b.ActorID && a.MemberID == b.MemberID && a.AppID == b.AppID && a.AppRevision == b.AppRevision && a.Environment == b.Environment && a.CallbackURL == b.CallbackURL && a.ConnectionVersion == b.ConnectionVersion && a.StoreVersion == b.StoreVersion && a.SellerID == b.SellerID && a.GrantID == b.GrantID && a.GrantGeneration == b.GrantGeneration && a.GrantVersion == b.GrantVersion
}
func (r *MemberScopedStoreRepository) SaveWalmartCandidate(ctx context.Context, a WalmartConnectionAttempt, key, sealed string, now time.Time) error {
	if key == "" || len(key) > 128 || sealed == "" || len(sealed) > 16384 {
		return ErrOfficialConnectionUnavailable
	}
	return r.walmartTransaction(ctx, a.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		prior, _, e := authorizedWalmartAttempt(tx, access, a.StoreID, a.AttemptID)
		if e != nil {
			return e
		}
		if !sameWalmartAttempt(a, prior) || prior.State != "exchange_dispatched" {
			return ErrVersionConflict
		}
		return tx.Model(&WalmartConnectionAttempt{}).Where("organization_id=? AND attempt_id=?", a.OrganizationID, a.AttemptID).Updates(map[string]any{"state": "credential_received", "key_id": key, "ciphertext": sealed, "updated_at": now}).Error
	})
}
func (r *MemberScopedStoreRepository) FailWalmartExchange(ctx context.Context, a WalmartConnectionAttempt, unknown bool, now time.Time) error {
	return r.walmartTransaction(ctx, a.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		prior, _, e := authorizedWalmartAttempt(tx, access, a.StoreID, a.AttemptID)
		if e != nil {
			return e
		}
		if !sameWalmartAttempt(a, prior) || prior.State != "exchange_dispatched" {
			return ErrVersionConflict
		}
		state := "rejected"
		if unknown {
			state = "unknown"
		}
		return tx.Model(&WalmartConnectionAttempt{}).Where("organization_id=? AND attempt_id=?", a.OrganizationID, a.AttemptID).Updates(map[string]any{"state": state, "updated_at": now}).Error
	})
}
func (r *MemberScopedStoreRepository) BindWalmartConnection(ctx context.Context, intent WalmartBindingIntent, now time.Time) (out OfficialConnectionView, err error) {
	a, g := intent.Attempt, intent.Grant
	if g.OrganizationID != a.OrganizationID || g.StoreID != a.StoreID || g.AppID != a.AppID || g.AppRevision != a.AppRevision || g.Environment != a.Environment || g.SellerID != a.SellerID || g.MerchantIdentityHash != WalmartMerchantIdentity(a.Environment, a.SellerID) || !g.CanRead(now) || g.KeyID == "" || g.Ciphertext == "" || len(g.ScopesHash) != 64 {
		return out, ErrOfficialConnectionUnavailable
	}
	err = r.walmartTransaction(ctx, a.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		prior, conn, e := authorizedWalmartAttempt(tx, access, a.StoreID, a.AttemptID)
		if e != nil {
			return e
		}
		if !sameWalmartAttempt(a, prior) || prior.State != "credential_received" {
			return ErrVersionConflict
		}
		var existing WalmartGrant
		e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", a.OrganizationID, a.StoreID).Take(&existing).Error
		if e == nil {
			if a.GrantID != existing.ID || g.ID != existing.ID || a.GrantVersion != existing.Version || a.GrantGeneration != existing.Generation || g.Generation != existing.Generation+1 || g.Version != existing.Version+1 || g.MerchantIdentityHash != existing.MerchantIdentityHash {
				return ErrVersionConflict
			}
			g.CreatedAt = existing.CreatedAt
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		} else if a.GrantID != "" || g.Generation != 1 || g.Version != 1 || uuid.Validate(g.ID) != nil {
			return ErrVersionConflict
		}
		occupancy := officialMerchantBinding{IdentityHash: g.MerchantIdentityHash, Platform: "walmart", AppID: g.Environment, OpenKeyID: "", OrganizationID: a.OrganizationID, StoreID: a.StoreID}
		if e = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&occupancy).Error; e != nil {
			return e
		}
		var binding officialMerchantBinding
		if e = tx.Where("identity_hash=?", g.MerchantIdentityHash).Take(&binding).Error; e != nil {
			return e
		}
		if binding.Platform != "walmart" || binding.OrganizationID != a.OrganizationID || binding.StoreID != a.StoreID {
			return ErrAlreadyExists
		}
		if existing.ID == "" {
			e = tx.Create(&g).Error
		} else {
			e = tx.Model(&WalmartGrant{}).Where("organization_id=? AND id=? AND version=?", a.OrganizationID, g.ID, existing.Version).Select("*").Updates(&g).Error
		}
		if e != nil {
			return e
		}
		if e = tx.Model(&WalmartConnectionAttempt{}).Where("organization_id=? AND attempt_id=?", a.OrganizationID, a.AttemptID).Updates(map[string]any{"state": "bound", "key_id": "", "ciphertext": "", "grant_id": g.ID, "grant_generation": g.Generation, "grant_version": g.Version, "updated_at": now}).Error; e != nil {
			return e
		}
		if e = tx.Model(&officialConnectionRow{}).Where("organization_id=? AND store_id=? AND version=?", a.OrganizationID, a.StoreID, conn.Version).Updates(map[string]any{"status": "connected", "observed_at": now}).Error; e != nil {
			return e
		}
		a.State = "bound"
		conn.Status = "connected"
		conn.ObservedAt = &now
		out = walmartConnectionView(conn, a, g, now)
		return nil
	})
	return
}
func walmartConnectionView(c officialConnectionRow, a WalmartConnectionAttempt, g WalmartGrant, now time.Time) OfficialConnectionView {
	v := OfficialConnectionView{Platform: PlatformWalmart, AuthorizationMethod: "oauth_callback", AppID: a.AppID, AppRevision: a.AppRevision, AttemptID: c.AttemptID, State: a.State, Status: ConnectionStatus(c.Status), Version: c.Version, ObservedAt: c.ObservedAt, Walmart: &WalmartConnectionDetails{Environment: a.Environment, Market: "US", PartnerID: g.SellerID, MerchantName: g.MerchantName, GrantID: g.ID, Generation: g.Generation, GrantVersion: g.Version}}
	if v.Status == "" {
		v.Status = ConnectionStatusDisconnected
		v.State = "disconnected"
	}
	if g.ID != "" {
		if g.Status == "refresh_dispatched" || g.Status == "refresh_unknown" {
			v.State = g.Status
		}
		v.MerchantIdentityHash = g.MerchantIdentityHash
		v.CredentialExpiresAt = &g.AccessExpiresAt
		if v.Status == ConnectionStatusConnected && !g.CanRead(now) {
			v.Status = ConnectionStatusUnavailable
			if g.Status == "active" && (!now.Before(g.AccessExpiresAt) || !now.Before(g.RefreshExpiresAt)) {
				v.Status = ConnectionStatusExpired
			}
		}
	}
	return v
}
func (r *MemberScopedStoreRepository) ReadWalmartConnection(ctx context.Context, org, storeID string) (out OfficialConnectionView, err error) {
	err = r.walmartTransaction(ctx, org, false, func(tx *gorm.DB, access StoreMemberAccess) error {
		_, e := walmartStore(tx, access, storeID)
		if e != nil {
			return e
		}
		var c officialConnectionRow
		e = tx.Where("organization_id=? AND store_id=?", org, storeID).Take(&c).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		var a WalmartConnectionAttempt
		if c.AttemptID != "" {
			if e = tx.Where("organization_id=? AND store_id=? AND attempt_id=?", org, storeID, c.AttemptID).Take(&a).Error; e != nil {
				return e
			}
		}
		var g WalmartGrant
		e = tx.Where("organization_id=? AND store_id=?", org, storeID).Take(&g).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		out = walmartConnectionView(c, a, g, time.Now().UTC())
		return nil
	})
	return
}
func (r *MemberScopedStoreRepository) ReadWalmartGrant(ctx context.Context, org, storeID string) (out WalmartGrant, err error) {
	err = r.walmartTransaction(ctx, org, false, func(tx *gorm.DB, access StoreMemberAccess) error {
		s, e := walmartStore(tx, access, storeID)
		if e != nil {
			return e
		}
		var c officialConnectionRow
		if e = tx.Where("organization_id=? AND store_id=? AND attempt_id=? AND status='connected'", org, storeID, s.ConnectionRef).Take(&c).Error; e != nil {
			return ErrOfficialConnectionUnavailable
		}
		return tx.Where("organization_id=? AND store_id=?", org, storeID).Take(&out).Error
	})
	return
}
func (r *MemberScopedStoreRepository) DisconnectWalmartConnection(ctx context.Context, c OfficialConnectionCommand, now time.Time) (out OfficialConnectionView, err error) {
	if !validConnectionCommand(c) {
		return out, ErrNotFound
	}
	err = r.walmartTransaction(ctx, c.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		s, e := walmartStore(tx, access, c.StoreID)
		if e != nil {
			return e
		}
		if s.Version != c.ExpectedStoreVersion {
			return ErrVersionConflict
		}
		if e = tx.Model(&WalmartGrant{}).Where("organization_id=? AND store_id=?", c.OrganizationID, c.StoreID).Updates(map[string]any{"status": "revoked", "version": gorm.Expr("version+1"), "updated_at": now}).Error; e != nil {
			return e
		}
		if e = tx.Model(&WalmartConnectionAttempt{}).Where("organization_id=? AND store_id=? AND state<>'bound'", c.OrganizationID, c.StoreID).Updates(map[string]any{"state": "rejected", "key_id": "", "ciphertext": "", "updated_at": now}).Error; e != nil {
			return e
		}
		if e = tx.Model(&officialConnectionRow{}).Where("organization_id=? AND store_id=?", c.OrganizationID, c.StoreID).Updates(map[string]any{"status": "disconnected", "version": gorm.Expr("version+1"), "observed_at": now}).Error; e != nil {
			return e
		}
		return updateConnectionRef(tx, s, "", access.ActorID, now)
	})
	if err == nil {
		return r.ReadWalmartConnection(ctx, c.OrganizationID, c.StoreID)
	}
	return
}
