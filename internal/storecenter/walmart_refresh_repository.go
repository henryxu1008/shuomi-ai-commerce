package storecenter

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

func (r *MemberScopedStoreRepository) ClaimWalmartRefresh(ctx context.Context, org, storeID, op string, generation int64, hash string, now time.Time) (out WalmartRefreshClaim, err error) {
	if uuid.Validate(op) != nil || generation < 1 || len(hash) != 64 {
		return out, ErrNotFound
	}
	err = r.walmartTransaction(ctx, org, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		s, e := walmartStore(tx, access, storeID)
		if e != nil {
			return e
		}
		if s.ConnectionRef == "" {
			return ErrOfficialConnectionUnavailable
		}
		var c officialConnectionRow
		if e = tx.Where("organization_id=? AND store_id=? AND attempt_id=? AND status='connected'", org, storeID, s.ConnectionRef).Take(&c).Error; e != nil {
			return ErrOfficialConnectionUnavailable
		}
		var g WalmartGrant
		if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", org, storeID).Take(&g).Error; e != nil {
			return ErrOfficialConnectionUnavailable
		}
		if g.RefreshOperationID == op {
			if g.RefreshInputHash != hash || g.RefreshActorID != access.ActorID || g.RefreshMemberID != access.MemberID || g.RefreshGeneration != generation {
				return ErrVersionConflict
			}
			out.Grant = g
			return nil
		}
		if g.Status != "active" || g.Generation != generation || !now.Before(g.RefreshExpiresAt) {
			return ErrOfficialConnectionUnavailable
		}
		g.Status = "refresh_dispatched"
		g.Version++
		g.RefreshOperationID = op
		g.RefreshInputHash = hash
		g.RefreshActorID = access.ActorID
		g.RefreshMemberID = access.MemberID
		g.RefreshGeneration = generation
		g.ClaimExpiresAt = now.Add(30 * time.Second)
		g.UpdatedAt = now
		if e = tx.Model(&WalmartGrant{}).Where("organization_id=? AND id=? AND version=?", org, g.ID, g.Version-1).Select("status", "version", "refresh_operation_id", "refresh_input_hash", "refresh_actor_id", "refresh_member_id", "refresh_generation", "claim_expires_at", "updated_at").Updates(&g).Error; e != nil {
			return e
		}
		out = WalmartRefreshClaim{Grant: g, Dispatch: true}
		return nil
	})
	if err != nil {
		out.Dispatch = false
	}
	return
}
func (r *MemberScopedStoreRepository) CompleteWalmartRefresh(ctx context.Context, claim WalmartRefreshClaim, next WalmartGrant, now time.Time) error {
	g := claim.Grant
	if !claim.Dispatch || next.Generation != g.Generation+1 || next.Version != g.Version+1 || next.Status != "active" || next.ID != g.ID || next.OrganizationID != g.OrganizationID || next.StoreID != g.StoreID || next.AppID != g.AppID || next.AppRevision != g.AppRevision || next.Environment != g.Environment || next.MerchantIdentityHash != g.MerchantIdentityHash || next.SellerID != g.SellerID || !next.CanRead(now) || next.KeyID == "" || next.Ciphertext == "" || len(next.ScopesHash) != 64 {
		return ErrVersionConflict
	}
	return r.walmartTransaction(ctx, g.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		if _, e := walmartStore(tx, access, g.StoreID); e != nil {
			return e
		}
		if access.ActorID != g.RefreshActorID || access.MemberID != g.RefreshMemberID || !now.Before(g.ClaimExpiresAt) {
			return ErrVersionConflict
		}
		update := tx.Model(&WalmartGrant{}).Where("organization_id=? AND id=? AND generation=? AND version=? AND status='refresh_dispatched' AND refresh_operation_id=? AND refresh_input_hash=?", g.OrganizationID, g.ID, g.Generation, g.Version, g.RefreshOperationID, g.RefreshInputHash).Select("generation", "version", "status", "key_id", "ciphertext", "scopes_hash", "access_expires_at", "refresh_expires_at", "updated_at").Updates(&next)
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrVersionConflict
		}
		return nil
	})
}
func (r *MemberScopedStoreRepository) UnknownWalmartRefresh(ctx context.Context, claim WalmartRefreshClaim, now time.Time) error {
	g := claim.Grant
	return r.walmartTransaction(ctx, g.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		if _, e := walmartStore(tx, access, g.StoreID); e != nil {
			return e
		}
		if access.ActorID != g.RefreshActorID || access.MemberID != g.RefreshMemberID {
			return ErrNotFound
		}
		return tx.Model(&WalmartGrant{}).Where("organization_id=? AND id=? AND generation=? AND version=? AND status='refresh_dispatched' AND refresh_operation_id=?", g.OrganizationID, g.ID, g.Generation, g.Version, g.RefreshOperationID).Updates(map[string]any{"status": "refresh_unknown", "updated_at": now}).Error
	})
}
