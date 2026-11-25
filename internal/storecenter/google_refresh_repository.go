package storecenter

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type googleRefreshRow struct {
	OrganizationID       string `gorm:"primaryKey"`
	StoreID              string `gorm:"primaryKey"`
	Generation, Revision int64
	Phase                GoogleRefreshPhase
	ClaimID              *string
	Deadline             *time.Time
}

func (googleRefreshRow) TableName() string { return googleRefreshTable }

func googleRefreshMetadata(tx *gorm.DB, grant GoogleGrant) (googleRefreshRow, bool, error) {
	row := googleRefreshRow{OrganizationID: grant.OrganizationID, StoreID: grant.StoreID,
		Generation: grant.Generation, Revision: 1, Phase: GoogleRefreshIdle}
	var stored googleRefreshRow
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ? AND store_id = ?", row.OrganizationID, row.StoreID).Take(&stored).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return row, false, nil
	}
	if err != nil {
		return row, false, err
	}
	if stored.Generation != grant.Generation {
		return row, false, ErrVersionConflict
	}
	return stored, true, nil
}

func (r *MemberScopedStoreRepository) googleCredentialTx(ctx context.Context, request GoogleCredentialRequest,
	run func(*gorm.DB, GoogleCredentialState, googleRefreshRow, bool, time.Time, time.Time) error,
) error {
	return r.googleTx(ctx, request.OrganizationID, false, func(tx *gorm.DB, access StoreMemberAccess) error {
		store, err := lockConnectionStore(tx, access.OrganizationID, request.StoreID)
		if err != nil {
			return err
		}
		if store.Platform != string(PlatformGoogleMerchant) || store.RecordStatus != string(RecordStatusActive) ||
			store.ServiceStatus == nil || *store.ServiceStatus != string(ServiceStatusActive) ||
			store.ServiceStartedAt == nil || store.ServiceExpiresAt == nil {
			return ErrOfficialConnectionUnavailable
		}
		var member storeMemberGrantRow
		if !access.Administrator {
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("organization_id = ? AND store_id = ? AND member_id = ? AND active", access.OrganizationID, store.ID, access.MemberID).
				Take(&member).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
		}
		grant, err := readGoogleGrantRow(tx, access.OrganizationID, store.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGoogleReauthorization
		}
		if err != nil {
			return err
		}
		if grant.Target != request.Target || request.Generation != 0 && grant.Generation != request.Generation {
			return ErrVersionConflict
		}
		if grant.Revoked || grant.AttemptID != store.ConnectionRef ||
			grant.AppID != request.Application.ID || grant.AppRevision != request.Application.Revision {
			return ErrGoogleReauthorization
		}
		row, exists, err := googleRefreshMetadata(tx, grant)
		if err != nil {
			return err
		}
		now, err := googleDatabaseTime(tx)
		if err != nil {
			return err
		}
		if now.Before(*store.ServiceStartedAt) || !now.Before(*store.ServiceExpiresAt) {
			return ErrOfficialConnectionUnavailable
		}
		state := GoogleCredentialState{Grant: grant, Phase: row.Phase,
			Usable: row.Phase == GoogleRefreshIdle && now.Add(GoogleAccessHeadroom).Before(grant.AccessExpiresAt),
			Snapshot: GoogleCredentialSnapshot{Binding: grant.CredentialBinding(), Target: grant.Target,
				ActorID: access.ActorID, MemberID: access.MemberID, Administrator: access.Administrator,
				MemberVersion: member.Version, StoreVersion: store.Version, ConnectionRef: store.ConnectionRef, Revision: row.Revision}}
		if row.ClaimID != nil {
			state.ClaimID = *row.ClaimID
		}
		if row.Deadline != nil {
			state.Deadline = *row.Deadline
		}
		return run(tx, state, row, exists, now, *store.ServiceExpiresAt)
	})
}

func (r *MemberScopedStoreRepository) GoogleCredential(ctx context.Context, request GoogleCredentialRequest, claimID string) (GoogleCredentialState, error) {
	var result GoogleCredentialState
	if claimID != "" {
		if id, err := uuid.Parse(claimID); err != nil || id == uuid.Nil || id.String() != claimID {
			return result, ErrVersionConflict
		}
	}
	err := r.googleCredentialTx(ctx, request, func(tx *gorm.DB, current GoogleCredentialState, row googleRefreshRow, exists bool, now, serviceEnd time.Time) error {
		result = current
		if claimID == "" || current.Usable || current.Phase == GoogleRefreshReauthorize || now.Before(current.Deadline) {
			return nil
		}
		if current.Phase == GoogleRefreshPossible {
			result.Phase, result.ClaimID, result.Deadline = GoogleRefreshReauthorize, "", time.Time{}
			return updateGoogleRefresh(tx, row, GoogleRefreshReauthorize, nil, nil, row.Revision)
		}
		deadline := earlierGoogleTime(now.Add(GoogleRefreshClaimWindow), serviceEnd)
		result.Phase, result.ClaimID, result.Deadline = GoogleRefreshClaimed, claimID, deadline
		if !exists {
			row.Phase, row.ClaimID, row.Deadline = result.Phase, &claimID, &deadline
			return tx.Create(&row).Error
		}
		return updateGoogleRefresh(tx, row, result.Phase, &claimID, &deadline, row.Revision)
	})
	return result, err
}

func (r *MemberScopedStoreRepository) AuthorizeGoogleRefresh(ctx context.Context, request GoogleCredentialRequest, own GoogleCredentialState) (time.Time, error) {
	var deadline time.Time
	err := r.googleCredentialTx(ctx, request, func(tx *gorm.DB, current GoogleCredentialState, row googleRefreshRow, _ bool, now, serviceEnd time.Time) error {
		if !matchesGoogleClaim(current, own, now) || current.Phase != GoogleRefreshClaimed {
			return ErrVersionConflict
		}
		deadline = earlierGoogleTime(now.Add(GoogleRefreshPermitWindow), serviceEnd).Truncate(time.Microsecond)
		if !now.Before(deadline.Add(-GoogleRefreshSaveReserve)) {
			return ErrGoogleRefreshUnavailable
		}
		return updateGoogleRefresh(tx, row, GoogleRefreshPossible, row.ClaimID, &deadline, row.Revision)
	})
	return deadline, err
}

func (r *MemberScopedStoreRepository) FinishGoogleRefresh(ctx context.Context, request GoogleCredentialRequest, own GoogleCredentialState, result GoogleRefreshOutcome) error {
	return r.googleCredentialTx(ctx, request, func(tx *gorm.DB, current GoogleCredentialState, row googleRefreshRow, _ bool, now, _ time.Time) error {
		if !matchesGoogleClaim(current, own, now) {
			return ErrVersionConflict
		}
		if result.Reauthorize {
			return updateGoogleRefresh(tx, row, GoogleRefreshReauthorize, nil, nil, row.Revision)
		}
		if result.KeyID == "" && result.Ciphertext == "" {
			cooldown := now.Add(min(max(result.Cooldown, time.Second), 30*time.Second))
			return updateGoogleRefresh(tx, row, GoogleRefreshIdle, nil, &cooldown, row.Revision)
		}
		if current.Phase != GoogleRefreshPossible || row.Revision == math.MaxInt64 ||
			result.KeyID == "" || len(result.KeyID) > 128 || result.Ciphertext == "" || len(result.Ciphertext) > 24000 ||
			!now.Add(GoogleAccessHeadroom).Before(result.AccessExpiresAt) {
			return ErrGoogleRefreshUnavailable
		}
		updated := tx.Model(&GoogleGrant{}).Where("organization_id = ? AND store_id = ?", row.OrganizationID, row.StoreID).
			Updates(map[string]any{"key_id": result.KeyID, "ciphertext": result.Ciphertext,
				"access_expires_at": result.AccessExpiresAt, "updated_at": now})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrVersionConflict
		}
		return updateGoogleRefresh(tx, row, GoogleRefreshIdle, nil, nil, row.Revision+1)
	})
}

func matchesGoogleClaim(current, own GoogleCredentialState, now time.Time) bool {
	return current.Snapshot == own.Snapshot && current.ClaimID != "" && current.ClaimID == own.ClaimID &&
		current.Phase == own.Phase && now.Before(current.Deadline)
}

func updateGoogleRefresh(tx *gorm.DB, row googleRefreshRow, phase GoogleRefreshPhase, claimID *string, deadline *time.Time, revision int64) error {
	updated := tx.Model(&googleRefreshRow{}).Where("organization_id = ? AND store_id = ?", row.OrganizationID, row.StoreID).
		Updates(map[string]any{"phase": phase, "claim_id": claimID, "deadline": deadline, "revision": revision})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return ErrVersionConflict
	}
	return nil
}

func earlierGoogleTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

func googleDatabaseTime(tx *gorm.DB) (time.Time, error) {
	var now time.Time
	err := tx.Raw("SELECT clock_timestamp()").Scan(&now).Error
	return now, err
}
