package shopifyguard

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const dispatchTable = "public.listing_shopify_dispatches"

const (
	MediaSyncPlanTable = "public.listing_shopify_media_sync_plans"
	MediaSyncStepTable = "public.listing_shopify_media_sync_steps"
)

type Dispatch struct {
	Organization, ShopKey, AttemptID string
	Generation, Epoch                int64
}

// AdmitDispatch consumes the isolated dispatch slot in the existing permit
// transaction, under the generation lock and after the original claim is checked.
func AdmitDispatch(tx *gorm.DB, scope Dispatch) error {
	enabled, err := installed(tx)
	if err != nil || !enabled {
		return err
	}
	inserted := tx.Exec("INSERT INTO "+dispatchTable+
		" (organization_id,shop_key,generation,attempt_id,epoch) VALUES (?,?,?,?,?) ON CONFLICT DO NOTHING",
		scope.Organization, scope.ShopKey, scope.Generation, scope.AttemptID, scope.Epoch)
	if inserted.Error != nil {
		return ErrUnavailable
	}
	if inserted.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}

// ObserveDispatchEnd is used only by OfficialCompletion after its original
// intent and synchronous response are verified. The persisted execution claim
// authorizes the narrow update even after its generation or lease has ended.
func ObserveDispatchEnd(tx *gorm.DB, scope Dispatch, claimOwner, claimTokenHash string) error {
	enabled, err := installed(tx)
	if err != nil || !enabled {
		return err
	}
	observed := tx.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Exec(`UPDATE `+dispatchTable+` AS d
		SET response_observed=true FROM public.listing_submission_execution_attempts AS a
		WHERE d.organization_id=? AND d.shop_key=? AND d.generation=? AND d.attempt_id=? AND d.epoch=?
		AND a.organization_id=d.organization_id AND a.attempt_id=d.attempt_id AND a.store_id=d.shop_key::text
		AND a.fence_epoch=d.epoch AND a.claim_owner_id=? AND a.claim_token_hash=?
		AND a.platform='shopify' AND a.action='create_shopify_draft'`,
		scope.Organization, scope.ShopKey, scope.Generation, scope.AttemptID, scope.Epoch, claimOwner, claimTokenHash)
	if observed.Error != nil {
		return ErrUnavailable
	}
	if observed.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}

// LiveIsolation is issued only on successful installation before any Shopify
// attempt in the disposable database. The fixture owns this database throughout
// the run and restores only into other databases and pools.
type LiveIsolation struct{ pool *sql.DB }

// VerifyLiveIsolation restricts supplemental test-only write composition to
// the exact pool that installed the guard before any Shopify execution.
func VerifyLiveIsolation(db *gorm.DB, live *LiveIsolation) error {
	if db == nil || live == nil {
		return ErrUnavailable
	}
	pool, err := db.DB()
	if err != nil || pool != live.pool {
		return ErrUnavailable
	}
	_, err = installed(RequireIsolated(db))
	return err
}

func ObserveMediaDispatchEnd(tx *gorm.DB, scope Dispatch, plan, phase, owner, tokenHash string) error {
	if _, err := installed(RequireIsolated(tx)); err != nil {
		return err
	}
	switch phase {
	case "prepare_shopify_media", "upload_shopify_media", "append_shopify_media", "associate_shopify_media":
	default:
		return ErrConflict
	}
	result := tx.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Exec(`
		UPDATE `+dispatchTable+` d SET response_observed=true
		FROM `+MediaSyncStepTable+` s
		JOIN `+MediaSyncPlanTable+` p USING (organization_id,plan_id)
		JOIN public.listing_submission_execution_attempts a USING (organization_id,attempt_id)
		WHERE d.organization_id=? AND d.shop_key=? AND d.generation=? AND d.attempt_id=? AND d.epoch=?
		AND d.organization_id=s.organization_id AND d.attempt_id=s.attempt_id
		AND d.shop_key=p.shop_key AND d.generation=p.generation AND s.plan_id=? AND s.phase=?
		AND a.fence_epoch=d.epoch AND a.store_id=d.shop_key::text AND a.platform='shopify'
		AND a.action=s.phase AND a.claim_owner_id=? AND a.claim_token_hash=?
		AND NOT d.response_observed`, scope.Organization, scope.ShopKey, scope.Generation, scope.AttemptID, scope.Epoch,
		plan, phase, owner, tokenHash)
	if result.Error != nil {
		return ErrUnavailable
	}
	if result.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}

// DrainReadyIsolated checks the continuously observed test scope.
func DrainReadyIsolated(ctx context.Context, db *gorm.DB, live *LiveIsolation, command Command) (bool, error) {
	if live == nil || db == nil {
		return false, nil
	}
	pool, err := db.DB()
	if err != nil || pool != live.pool {
		return false, nil
	}
	var ready bool
	err = RequireIsolated(db).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := installed(tx); err != nil {
			return err
		}
		if err := lockGeneration(tx, command.Organization, command.ShopKey, command.Generation); err != nil {
			return err
		}
		prior, err := readCommand(tx, command.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return ErrUnavailable
		}
		if prior.Organization != command.Organization || prior.ShopKey != command.ShopKey ||
			prior.Generation != command.Generation || !slices.Equal(prior.Sources, command.Sources) {
			return ErrConflict
		}
		err = tx.Raw(`SELECT c.sealed_locally AND NOT EXISTS
			(SELECT 1 FROM `+dispatchTable+` d WHERE d.organization_id=c.organization_id AND d.shop_key=c.shop_key
			 AND d.generation=c.generation AND NOT d.response_observed)
			FROM `+commandTable+` c WHERE c.id=?`, command.ID).Row().Scan(&ready)
		if err != nil {
			return ErrUnavailable
		}
		return nil
	})
	return ready && err == nil, err
}
