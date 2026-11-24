package storecenter

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const (
	amazonAwaitingLogin          = "awaiting_login"
	amazonAwaitingRedirect       = "awaiting_redirect"
	amazonExchangeClaimed        = "exchange_claimed"
	amazonAuthorizationBound     = "bound"
	amazonAuthorizationAbandoned = "abandoned"
)

type amazonAuthorizationRow struct {
	SessionDigest                                         string `gorm:"primaryKey"`
	OrganizationID, StoreID, AttemptID, ActorID, MemberID string
	StoreVersion                                          int64
	AppID, AppRevision, RedirectURI                       string
	Beta                                                  bool
	FlowDigest, StateDigest, CodeDigest, SellerID         string
	Generation                                            int64
	Phase                                                 string
	StartedAt, ExpiresAt                                  time.Time
}

func (amazonAuthorizationRow) TableName() string { return "workbench_store_amazon_attempts" }

type amazonAuthorizationGrantRow struct {
	OrganizationID, StoreID, AttemptID string
	AppID, AppRevision, SellerID       string
	Beta                               bool
	Generation                         int64
	KeyID, Ciphertext                  string
	BoundAt                            time.Time
}

func (amazonAuthorizationGrantRow) TableName() string { return "workbench_store_amazon_grants" }

var _ AmazonAuthorizationStore = (*MemberScopedStoreRepository)(nil)

func (r *MemberScopedStoreRepository) amazonAuthorizationTransaction(ctx context.Context, scope AmazonBrowserScope, apply func(*gorm.DB, StoreMemberAccess) error) error {
	if scope.SessionDigest == [32]byte{} {
		return ErrNotFound
	}
	access, err := r.authorize(ctx, scope.OrganizationID, true)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Transaction(func(tx *gorm.DB) error { return apply(tx, access) })
}

func (r *MemberScopedStoreRepository) StartAmazonAuthorization(ctx context.Context, begin AmazonAuthorizationBegin) error {
	if uuid.Validate(begin.StoreID) != nil || uuid.Validate(begin.AttemptID) != nil || begin.ExpectedStoreVersion <= 0 ||
		!amazonAuthorizationDigest(begin.FlowDigest) || !validAmazonAuthorizationApp(begin.Application) ||
		begin.Now.IsZero() || !begin.ExpiresAt.After(begin.Now) || begin.ExpiresAt.Sub(begin.Now) > 9*time.Minute {
		return ErrNotFound
	}
	transaction := func() error {
		return r.amazonAuthorizationTransaction(ctx, begin.Scope, func(tx *gorm.DB, access StoreMemberAccess) error {
			store, err := amazonAuthorizationStore(tx, access, begin.StoreID)
			if err != nil {
				return err
			}
			if store.Version != begin.ExpectedStoreVersion {
				return ErrVersionConflict
			}
			if err := abandonAmazonPendingAuthorization(tx, access.OrganizationID, store.ID); err != nil {
				return err
			}
			app := begin.Application
			row := amazonAuthorizationRow{
				SessionDigest: hex.EncodeToString(begin.Scope.SessionDigest[:]), OrganizationID: access.OrganizationID,
				StoreID: store.ID, AttemptID: begin.AttemptID, ActorID: access.ActorID, MemberID: access.MemberID,
				StoreVersion: store.Version, AppID: app.ID, AppRevision: app.Revision, RedirectURI: app.RedirectURI,
				Beta: app.Beta, FlowDigest: begin.FlowDigest, Phase: amazonAwaitingLogin,
				StartedAt: begin.Now.UTC(), ExpiresAt: begin.ExpiresAt.UTC(),
			}
			// A browser can only carry one Amazon flow, even if the user started
			// it again for a different Store or switched organizations.
			return tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "session_digest"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"organization_id", "store_id", "attempt_id", "actor_id", "member_id", "store_version",
					"app_id", "app_revision", "redirect_uri", "beta", "flow_digest", "state_digest", "code_digest",
					"seller_id", "generation", "phase", "started_at", "expires_at",
				}),
			}).Create(&row).Error
		})
	}
	err := transaction()
	var postgresError interface{ SQLState() string }
	// Crossed retargets can lock each other's old session row. PostgreSQL
	// rolls back the deadlock victim, so retry the local-only Begin once.
	if errors.As(err, &postgresError) && postgresError.SQLState() == "40P01" && ctx.Err() == nil {
		return transaction()
	}
	return err
}

func abandonAmazonPendingAuthorization(tx *gorm.DB, organizationID, storeID string) error {
	return tx.Model(&amazonAuthorizationRow{}).
		Where("organization_id = ? AND store_id = ? AND phase IN ?", organizationID, storeID,
			[]string{amazonAwaitingLogin, amazonAwaitingRedirect, amazonExchangeClaimed}).
		Updates(map[string]any{
			"phase": amazonAuthorizationAbandoned, "seller_id": "", "state_digest": "", "code_digest": "", "generation": 0,
		}).Error
}

func (r *MemberScopedStoreRepository) AdvanceAmazonAuthorization(ctx context.Context, step AmazonAuthorizationStep) error {
	if !validAmazonAuthorizationStep(step) {
		return ErrNotFound
	}
	return r.amazonAuthorizationTransaction(ctx, step.Scope, func(tx *gorm.DB, access StoreMemberAccess) error {
		row, err := readAmazonAuthorizationStep(tx, access, step)
		if err != nil {
			return err
		}
		if row.Phase != amazonAwaitingLogin {
			return ErrNotFound
		}
		return updateAmazonAuthorizationRow(tx, row, map[string]any{
			"phase": amazonAwaitingRedirect, "seller_id": step.SellerID, "state_digest": step.StateDigest,
		})
	})
}

func (r *MemberScopedStoreRepository) ClaimAmazonAuthorization(ctx context.Context, cmd AmazonAuthorizationClaimCommand) (AmazonAuthorizationClaim, error) {
	var claim AmazonAuthorizationClaim
	if !validAmazonAuthorizationStep(cmd.AmazonAuthorizationStep) || !amazonAuthorizationDigest(cmd.CodeDigest) {
		return claim, ErrNotFound
	}
	err := r.amazonAuthorizationTransaction(ctx, cmd.Scope, func(tx *gorm.DB, access StoreMemberAccess) error {
		row, err := readAmazonAuthorizationStep(tx, access, cmd.AmazonAuthorizationStep)
		if err != nil {
			return err
		}
		if row.SellerID != cmd.SellerID || subtle.ConstantTimeCompare([]byte(row.StateDigest), []byte(cmd.StateDigest)) != 1 {
			return ErrNotFound
		}
		if row.CodeDigest != "" && subtle.ConstantTimeCompare([]byte(row.CodeDigest), []byte(cmd.CodeDigest)) != 1 {
			return ErrNotFound
		}
		claim, err = claimAmazonAuthorizationRow(tx, row, cmd)
		return err
	})
	if err != nil {
		return AmazonAuthorizationClaim{}, err
	}
	return claim, nil
}

func claimAmazonAuthorizationRow(tx *gorm.DB, row amazonAuthorizationRow, cmd AmazonAuthorizationClaimCommand) (AmazonAuthorizationClaim, error) {
	claim := AmazonAuthorizationClaim{
		Scope: cmd.Scope, Application: cmd.Application, StoreID: row.StoreID, AttemptID: row.AttemptID,
		ActorID: row.ActorID, MemberID: row.MemberID, SellerID: row.SellerID, Generation: row.Generation,
	}
	grant, err := readAmazonAuthorizationGrant(tx, row)
	if err != nil {
		return AmazonAuthorizationClaim{}, err
	}
	switch row.Phase {
	case amazonAwaitingRedirect:
		if grant.Generation == math.MaxInt64 {
			return AmazonAuthorizationClaim{}, ErrVersionConflict
		}
		claim.Generation = grant.Generation + 1
		err := updateAmazonAuthorizationRow(tx, row, map[string]any{
			"phase": amazonExchangeClaimed, "code_digest": cmd.CodeDigest, "generation": claim.Generation,
		})
		return claim, err
	case amazonExchangeClaimed:
		return AmazonAuthorizationClaim{}, ErrOfficialExchangeUnknown
	case amazonAuthorizationBound:
		if grant.AttemptID != row.AttemptID || grant.Generation != row.Generation ||
			grant.AppID != row.AppID || grant.AppRevision != row.AppRevision {
			return AmazonAuthorizationClaim{}, ErrNotFound
		}
		claim.Replay = true
		return claim, nil
	default:
		return AmazonAuthorizationClaim{}, ErrNotFound
	}
}

func (r *MemberScopedStoreRepository) BindAmazonAuthorization(ctx context.Context, claim AmazonAuthorizationClaim, keyID, ciphertext string, now time.Time) error {
	if claim.Replay || uuid.Validate(claim.AttemptID) != nil || uuid.Validate(claim.StoreID) != nil || claim.Generation <= 0 ||
		keyID == "" || len(keyID) > 128 || ciphertext == "" || len(ciphertext) > 16384 {
		return ErrNotFound
	}
	return r.amazonAuthorizationTransaction(ctx, claim.Scope, func(tx *gorm.DB, access StoreMemberAccess) error {
		filter := amazonAuthorizationFilter(tx, claim.Scope).
			Where("attempt_id = ? AND store_id = ?", claim.AttemptID, claim.StoreID)
		row, err := readLockedAmazonAuthorization(filter, tx, access, claim.Application, now)
		if err != nil {
			return err
		}
		if row.Phase != amazonExchangeClaimed || row.Generation != claim.Generation || row.SellerID != claim.SellerID ||
			row.ActorID != claim.ActorID || row.MemberID != claim.MemberID {
			return ErrNotFound
		}
		return bindAmazonAuthorizationRow(tx, row, keyID, ciphertext, now.UTC())
	})
}

func bindAmazonAuthorizationRow(tx *gorm.DB, row amazonAuthorizationRow, keyID, ciphertext string, now time.Time) error {
	prior, err := readAmazonAuthorizationGrant(tx, row)
	if err != nil {
		return err
	}
	if prior.Generation != row.Generation-1 {
		return ErrVersionConflict
	}
	grant := amazonAuthorizationGrantRow{
		OrganizationID: row.OrganizationID, StoreID: row.StoreID, AttemptID: row.AttemptID,
		AppID: row.AppID, AppRevision: row.AppRevision, SellerID: row.SellerID, Beta: row.Beta,
		Generation: row.Generation, KeyID: keyID, Ciphertext: ciphertext, BoundAt: now,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}, {Name: "store_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"attempt_id", "app_id", "app_revision", "generation", "key_id", "ciphertext", "bound_at"}),
	}).Create(&grant).Error; err != nil {
		if isUniqueConstraint(err) {
			return ErrAlreadyExists
		}
		return err
	}
	return updateAmazonAuthorizationRow(tx, row, map[string]any{"phase": amazonAuthorizationBound})
}

func readAmazonAuthorizationStep(tx *gorm.DB, access StoreMemberAccess, step AmazonAuthorizationStep) (amazonAuthorizationRow, error) {
	query := amazonAuthorizationFilter(tx, step.Scope).Where("flow_digest = ?", step.FlowDigest)
	return readLockedAmazonAuthorization(query, tx, access, step.Application, step.Now)
}

func amazonAuthorizationFilter(tx *gorm.DB, scope AmazonBrowserScope) *gorm.DB {
	return tx.Model(&amazonAuthorizationRow{}).
		Where("organization_id = ? AND session_digest = ?", scope.OrganizationID, hex.EncodeToString(scope.SessionDigest[:]))
}

func readLockedAmazonAuthorization(query, tx *gorm.DB, access StoreMemberAccess, app AmazonAuthorizationApplication, now time.Time) (amazonAuthorizationRow, error) {
	var row amazonAuthorizationRow
	// Lock Store before the attempt, as Begin does. Re-read after acquiring
	// that lock because a different flow may have invalidated this one.
	if err := query.Select("store_id").Take(&row).Error; err != nil {
		return amazonAuthorizationRow{}, amazonAuthorizationNotFound(err)
	}
	store, err := amazonAuthorizationStore(tx, access, row.StoreID)
	if err != nil {
		return amazonAuthorizationRow{}, err
	}
	row = amazonAuthorizationRow{}
	if err := query.Select("*").Clauses(clause.Locking{Strength: "UPDATE"}).Take(&row).Error; err != nil {
		return amazonAuthorizationRow{}, amazonAuthorizationNotFound(err)
	}
	if row.StoreID != store.ID || row.StoreVersion != store.Version || row.ActorID != access.ActorID || row.MemberID != access.MemberID ||
		row.AppID != app.ID || row.AppRevision != app.Revision || row.RedirectURI != app.RedirectURI || row.Beta != app.Beta ||
		now.Before(row.StartedAt) || !now.Before(row.ExpiresAt) {
		return amazonAuthorizationRow{}, ErrNotFound
	}
	return row, nil
}

func amazonAuthorizationStore(tx *gorm.DB, access StoreMemberAccess, id string) (workbenchStoreRecord, error) {
	store, err := connectionStore(tx, access, id)
	if err != nil {
		return store, err
	}
	if store.Platform != "amazon" || store.Region != "US" {
		return workbenchStoreRecord{}, ErrNotFound
	}
	return store, nil
}

func readAmazonAuthorizationGrant(tx *gorm.DB, attempt amazonAuthorizationRow) (amazonAuthorizationGrantRow, error) {
	var row amazonAuthorizationGrantRow
	err := tx.Where("organization_id = ? AND store_id = ?", attempt.OrganizationID, attempt.StoreID).Take(&row).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return amazonAuthorizationGrantRow{}, nil
	case err != nil:
		return amazonAuthorizationGrantRow{}, err
	case row.SellerID != attempt.SellerID || row.Beta != attempt.Beta:
		return amazonAuthorizationGrantRow{}, ErrNotFound
	default:
		return row, nil
	}
}

func updateAmazonAuthorizationRow(tx *gorm.DB, row amazonAuthorizationRow, updates map[string]any) error {
	result := tx.Model(&amazonAuthorizationRow{}).
		Where("session_digest = ? AND attempt_id = ? AND phase = ?", row.SessionDigest, row.AttemptID, row.Phase).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrVersionConflict
	}
	return nil
}

func amazonAuthorizationNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func amazonAuthorizationDigest(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validAmazonAuthorizationStep(step AmazonAuthorizationStep) bool {
	_, err := validateOpaqueIdentity("Amazon seller ID", step.SellerID, 64)
	return err == nil && validAmazonAuthorizationApp(step.Application) && !step.Now.IsZero() &&
		amazonAuthorizationDigest(step.FlowDigest) && amazonAuthorizationDigest(step.StateDigest)
}

func validAmazonAuthorizationApp(app AmazonAuthorizationApplication) bool {
	_, idErr := validateOpaqueIdentity("Amazon app ID", app.ID, 200)
	_, revisionErr := validateOpaqueIdentity("Amazon app revision", app.Revision, 128)
	return idErr == nil && revisionErr == nil && len(app.RedirectURI) <= 1024 && strings.HasPrefix(app.RedirectURI, "https://")
}
