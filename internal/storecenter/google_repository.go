package storecenter

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	auth "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func (r *MemberScopedStoreRepository) googleTx(ctx context.Context, org string, write bool, run func(*gorm.DB, StoreMemberAccess) error) error {
	access, err := r.authorize(ctx, org, write)
	if err != nil {
		return err
	}
	if write && !access.Administrator {
		return ErrNotFound
	}
	return r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Transaction(func(tx *gorm.DB) error { return run(tx, access) })
}

func googleStore(tx *gorm.DB, access StoreMemberAccess, id string) (workbenchStoreRecord, error) {
	store, err := connectionStore(tx, access, id)
	if err == nil && store.Platform != string(PlatformGoogleMerchant) {
		err = ErrNotFound
	}
	return store, err
}

func readGoogleGrantRow(tx *gorm.DB, org, store string) (GoogleGrant, error) {
	var grant GoogleGrant
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ? AND store_id = ?", org, store).Take(&grant).Error
	return grant, err
}

func (r *MemberScopedStoreRepository) BeginGoogleConnection(ctx context.Context, command OfficialConnectionCommand, app auth.Application, stateHash string, now time.Time) (GoogleAttempt, error) {
	var result GoogleAttempt
	decoded, err := hex.DecodeString(stateHash)
	if !validConnectionCommand(command) || command.ApplicationID != app.ID || app.Revision == "" || err != nil || len(decoded) != 32 {
		return result, ErrNotFound
	}
	err = r.googleTx(ctx, command.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		store, err := googleStore(tx, access, command.StoreID)
		if err != nil {
			return err
		}
		if store.Version != command.ExpectedStoreVersion || store.RecordStatus != string(RecordStatusActive) {
			return ErrVersionConflict
		}
		grant, err := readGoogleGrantRow(tx, store.OrganizationID, store.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if grant.ID == "" {
			grant.ID = uuid.NewString()
		}
		result = GoogleAttempt{
			OrganizationID: access.OrganizationID, StoreID: store.ID, AttemptID: command.AttemptID,
			ActorID: access.ActorID, MemberID: access.MemberID, AppID: app.ID, AppRevision: app.Revision,
			State: "created", StateHash: stateHash, StoreVersion: store.Version + 1, GrantID: grant.ID, Generation: grant.Generation + 1,
			CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(10 * time.Minute),
		}
		if err := tx.Create(&result).Error; err != nil {
			return googleConflict(err)
		}
		if err := clearGoogleCandidate(tx, store.OrganizationID, store.ConnectionRef, now); err != nil {
			return err
		}
		return updateConnectionRef(tx, store, result.AttemptID, access.ActorID, now)
	})
	return result, err
}

func authorizedGoogleAttempt(tx *gorm.DB, access StoreMemberAccess, storeID, attemptID string, now time.Time) (GoogleAttempt, error) {
	var attempt GoogleAttempt
	store, err := googleStore(tx, access, storeID)
	if err != nil {
		return attempt, err
	}
	if store.ConnectionRef != attemptID || store.RecordStatus != string(RecordStatusActive) {
		return attempt, ErrNotFound
	}
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ? AND store_id = ? AND attempt_id = ?", access.OrganizationID, storeID, attemptID).Take(&attempt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return attempt, ErrNotFound
	}
	if err != nil {
		return attempt, err
	}
	if attempt.ActorID != access.ActorID || attempt.MemberID != access.MemberID || attempt.StoreVersion != store.Version || !now.Before(attempt.ExpiresAt) {
		return GoogleAttempt{}, ErrNotFound
	}
	return attempt, nil
}

func (r *MemberScopedStoreRepository) ClaimGoogleExchange(ctx context.Context, org, storeID, attemptID, stateHash string, app auth.Application, now time.Time) (GoogleAttempt, error) {
	var attempt GoogleAttempt
	err := r.googleTx(ctx, org, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		var err error
		attempt, err = authorizedGoogleAttempt(tx, access, storeID, attemptID, now)
		if err != nil {
			return err
		}
		if attempt.State != "created" || attempt.AppID != app.ID || attempt.AppRevision != app.Revision ||
			subtle.ConstantTimeCompare([]byte(attempt.StateHash), []byte(stateHash)) != 1 {
			return ErrNotFound
		}
		return updateGoogleAttempt(tx, attempt, "exchanging", "", "", now)
	})
	return attempt, err
}

func (r *MemberScopedStoreRepository) SaveGoogleCandidate(ctx context.Context, attempt GoogleAttempt, key, ciphertext string, now time.Time) error {
	if key == "" || len(key) > 128 || ciphertext == "" || len(ciphertext) > 24000 {
		return ErrNotFound
	}
	return r.googleTx(ctx, attempt.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		current, err := authorizedGoogleAttempt(tx, access, attempt.StoreID, attempt.AttemptID, now)
		if err != nil {
			return err
		}
		if current.State != "exchanging" || current.CredentialBinding() != attempt.CredentialBinding() {
			return ErrVersionConflict
		}
		return updateGoogleAttempt(tx, current, "candidate", key, ciphertext, now)
	})
}

func (r *MemberScopedStoreRepository) ReadGoogleCandidate(ctx context.Context, org, storeID, attemptID string, now time.Time) (GoogleAttempt, error) {
	var attempt GoogleAttempt
	err := r.googleTx(ctx, org, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		var err error
		attempt, err = authorizedGoogleAttempt(tx, access, storeID, attemptID, now)
		if err == nil && attempt.State != "candidate" {
			err = ErrNotFound
		}
		return err
	})
	return attempt, err
}

func (r *MemberScopedStoreRepository) BindGoogleConnection(ctx context.Context, attempt GoogleAttempt, grant GoogleGrant, now time.Time) (GoogleGrant, error) {
	if !grant.Target.Valid() || grant.KeyID == "" || len(grant.KeyID) > 128 || grant.Ciphertext == "" || len(grant.Ciphertext) > 24000 ||
		!now.Before(grant.AccessExpiresAt) || grant.Revoked {
		return GoogleGrant{}, ErrNotFound
	}
	err := r.googleTx(ctx, attempt.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		current, err := authorizedGoogleAttempt(tx, access, attempt.StoreID, attempt.AttemptID, now)
		if err != nil {
			return err
		}
		if current.State != "candidate" || current.CredentialBinding() != attempt.CredentialBinding() ||
			grant.OrganizationID != current.OrganizationID || grant.StoreID != current.StoreID ||
			grant.ID != current.GrantID || grant.Generation != current.Generation ||
			grant.AppID != current.AppID || grant.AppRevision != current.AppRevision {
			return ErrVersionConflict
		}
		grant.AttemptID, grant.CreatedAt, grant.UpdatedAt = current.AttemptID, now, now
		prior, err := readGoogleGrantRow(tx, grant.OrganizationID, grant.StoreID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&grant).Error; err != nil {
				return googleConflict(err)
			}
		} else {
			if err != nil {
				return err
			}
			if prior.ID != grant.ID || prior.Generation+1 != grant.Generation || prior.Target != grant.Target {
				return ErrVersionConflict
			}
			grant.CreatedAt = prior.CreatedAt
			update := tx.Model(&GoogleGrant{}).Where("organization_id = ? AND store_id = ? AND generation = ?", grant.OrganizationID, grant.StoreID, prior.Generation).
				Select("attempt_id", "app_id", "app_revision", "generation", "key_id", "ciphertext", "revoked", "access_expires_at", "updated_at").Updates(&grant)
			if update.Error != nil {
				return update.Error
			}
			if update.RowsAffected != 1 {
				return ErrVersionConflict
			}
			if err := tx.Model(&googleRefreshRow{}).Where("organization_id = ? AND store_id = ?", grant.OrganizationID, grant.StoreID).
				Updates(map[string]any{"generation": grant.Generation, "revision": 1, "phase": GoogleRefreshIdle,
					"claim_id": nil, "deadline": nil}).Error; err != nil {
				return err
			}
		}
		return updateGoogleAttempt(tx, current, "bound", "", "", now)
	})
	return grant, err
}

func (r *MemberScopedStoreRepository) ReadGoogleGrant(ctx context.Context, org, storeID string, publish bool) (GoogleGrant, error) {
	var grant GoogleGrant
	err := r.googleTx(ctx, org, false, func(tx *gorm.DB, access StoreMemberAccess) error {
		store, err := googleStore(tx, access, storeID)
		if err != nil {
			return err
		}
		grant, err = readGoogleGrantRow(tx, org, storeID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if grant.AttemptID != store.ConnectionRef {
			grant.Revoked = true
		}
		if grant.Revoked {
			if publish {
				return ErrGoogleReauthorization
			}
			return nil
		}
		refresh, _, err := googleRefreshMetadata(tx, grant)
		if err != nil {
			return err
		}
		grant.RefreshUnavailable = refresh.Phase == GoogleRefreshReauthorize
		if publish || refresh.Phase == GoogleRefreshPossible {
			now, err := googleDatabaseTime(tx)
			if err != nil {
				return err
			}
			if publish && (store.RecordStatus != string(RecordStatusActive) ||
				store.ServiceStatus == nil || *store.ServiceStatus != string(ServiceStatusActive) ||
				store.ServiceStartedAt == nil || store.ServiceExpiresAt == nil || now.Before(*store.ServiceStartedAt) || !now.Before(*store.ServiceExpiresAt)) {
				return ErrOfficialConnectionUnavailable
			}
			if refresh.Phase == GoogleRefreshPossible {
				grant.RefreshUnavailable = !now.Before(*refresh.Deadline)
			}
		}
		return nil
	})
	return grant, err
}

func (r *MemberScopedStoreRepository) DisconnectGoogleConnection(ctx context.Context, command OfficialConnectionCommand, now time.Time) error {
	return r.googleTx(ctx, command.OrganizationID, true, func(tx *gorm.DB, access StoreMemberAccess) error {
		store, err := googleStore(tx, access, command.StoreID)
		if err != nil {
			return err
		}
		if store.Version != command.ExpectedStoreVersion {
			return ErrVersionConflict
		}
		if err := tx.Model(&GoogleGrant{}).Where("organization_id = ? AND store_id = ?", store.OrganizationID, store.ID).
			Updates(map[string]any{"revoked": true, "key_id": "", "ciphertext": "", "updated_at": now}).Error; err != nil {
			return err
		}
		if err := clearGoogleCandidate(tx, store.OrganizationID, store.ConnectionRef, now); err != nil {
			return err
		}
		return updateConnectionRef(tx, store, "", access.ActorID, now)
	})
}

func clearGoogleCandidate(tx *gorm.DB, org, attemptID string, now time.Time) error {
	if attemptID == "" {
		return nil
	}
	return tx.Model(&GoogleAttempt{}).Where("organization_id = ? AND attempt_id = ? AND state = 'candidate'", org, attemptID).
		Updates(map[string]any{"state": "superseded", "key_id": "", "ciphertext": "", "updated_at": now}).Error
}

func updateGoogleAttempt(tx *gorm.DB, attempt GoogleAttempt, state, key, ciphertext string, now time.Time) error {
	update := tx.Model(&GoogleAttempt{}).Where("organization_id = ? AND attempt_id = ? AND state = ?", attempt.OrganizationID, attempt.AttemptID, attempt.State).
		Updates(map[string]any{"state": state, "key_id": key, "ciphertext": ciphertext, "updated_at": now})
	if update.Error != nil {
		return update.Error
	}
	if update.RowsAffected != 1 {
		return ErrVersionConflict
	}
	return nil
}

func googleConflict(err error) error {
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) && postgres.Code == "23505" {
		return ErrAlreadyExists
	}
	return err
}
