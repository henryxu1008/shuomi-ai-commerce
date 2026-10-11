package storecenter

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"task-processor/internal/authidentity"
)

func (r *MemberScopedStoreRepository) ReadOfficialInstallAttempt(ctx context.Context, org, storeID, attemptID string) (result OfficialConnectionAttempt, err error) {
	access, err := r.authorize(ctx, org, true)
	if err != nil {
		return result, err
	}
	err = r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Transaction(func(tx *gorm.DB) error {
		original, _, err := r.authorizedAttempt(tx, access, attemptID)
		if err != nil {
			return err
		}
		if original.StoreID != storeID || original.Platform != string(PlatformTemu) || original.AuthorizationMethod != string(AuthorizationManualToken) {
			return ErrNotFound
		}
		result = attemptValue(original)
		return nil
	})
	return result, err
}

// A manual install is a local encrypted input, not a remote exchange. Persist
// its exact digest before any provider read, in the existing Store transaction.
func (r *MemberScopedStoreRepository) InstallOfficialPrivateCredential(ctx context.Context, a OfficialConnectionAttempt, digest, keyID, ciphertext string, now time.Time) (result OfficialConnectionAttempt, err error) {
	decoded, e := hex.DecodeString(digest)
	if e != nil || len(decoded) != 32 || len(keyID) < 1 || len(keyID) > 128 || len(ciphertext) < 1 || len(ciphertext) > 16384 {
		return result, ErrNotFound
	}
	access, err := r.authorize(ctx, a.OrganizationID, true)
	if err != nil {
		return result, err
	}
	err = r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Transaction(func(tx *gorm.DB) error {
		original, _, err := r.authorizedAttempt(tx, access, a.AttemptID)
		if err != nil {
			return err
		}
		if !sameOfficialAttemptBinding(original, a) || original.AuthorizationMethod != string(AuthorizationManualToken) || original.Platform != string(PlatformTemu) {
			return ErrNotFound
		}
		if original.PrivateInputDigest != "" {
			if subtle.ConstantTimeCompare([]byte(original.PrivateInputDigest), []byte(digest)) != 1 {
				return ErrVersionConflict
			}
			if original.State != "credential_received" && original.State != "verified" {
				return ErrNotFound
			}
			result = attemptValue(original)
			return nil
		}
		if original.State != "awaiting_consent" || !now.Before(original.ExpiresAt) {
			return ErrOfficialAuthorizationRejected
		}
		update := tx.Model(&officialAttemptRow{}).Where("organization_id = ? AND attempt_id = ? AND state = ?", a.OrganizationID, a.AttemptID, "awaiting_consent").Updates(map[string]any{"state": "credential_received", "private_input_digest": digest, "key_id": keyID, "ciphertext": ciphertext, "updated_at": now.UTC()})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrVersionConflict
		}
		original.State, original.PrivateInputDigest, original.KeyID, original.Ciphertext = "credential_received", digest, keyID, ciphertext
		result = attemptValue(original)
		return nil
	})
	return result, err
}

func bindTemuVerification(tx *gorm.DB, original officialAttemptRow, facts *OfficialVerificationFacts, now time.Time) (string, error) {
	if facts == nil || facts.Platform != PlatformTemu || facts.IdentityPartition != OfficialIdentityPartition(original.IdentityPartition) || facts.VerifiedAt.IsZero() || facts.VerifiedAt.After(now) || !now.Before(facts.ExpiresAt) || !authidentity.IsBoundedIdentifier(facts.ContractRevision) || facts.Scopes == nil || len(facts.Scopes) > 4096 {
		return "", ErrOfficialConnectionUnavailable
	}
	for _, scope := range facts.Scopes {
		if !authidentity.IsBoundedIdentifier(scope) {
			return "", ErrOfficialConnectionUnavailable
		}
	}
	identity, err := facts.MerchantIdentityHash()
	if err != nil {
		return "", err
	}
	// One native Store cannot silently rotate to another official merchant.
	var prior officialMerchantBinding
	err = tx.Where("platform = ? AND organization_id = ? AND store_id = ?", string(PlatformTemu), original.OrganizationID, original.StoreID).Take(&prior).Error
	if err == nil && prior.IdentityHash != identity {
		return "", ErrAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	binding := officialMerchantBinding{IdentityHash: identity, Platform: string(PlatformTemu), AppID: original.AppID, OrganizationID: original.OrganizationID, StoreID: original.StoreID}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&binding).Error; err != nil {
		return "", err
	}
	var claimed officialMerchantBinding
	if err := tx.Where("identity_hash = ?", identity).Take(&claimed).Error; err != nil {
		return "", err
	}
	if claimed.Platform != string(PlatformTemu) || claimed.OrganizationID != original.OrganizationID || claimed.StoreID != original.StoreID {
		return "", ErrAlreadyExists
	}
	data, err := json.Marshal(facts)
	if err != nil || len(data) > 2*1024*1024 {
		return "", ErrOfficialConnectionUnavailable
	}
	return string(data), nil
}
