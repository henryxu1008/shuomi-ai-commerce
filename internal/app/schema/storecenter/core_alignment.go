package storecenter

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"gorm.io/gorm"
	"task-processor/internal/storecenter"
)

const CoreAlignmentVersion int64 = 2026101601

// AlignCoreContract is the explicit retained-database entry. It never installs
// missing tables, grants privileges, changes business rows or rewrites history.
// Capabilities declare already installed modules; this entry never provisions them.
func AlignCoreContract(ctx context.Context, db *gorm.DB, confirmedDatabase string, capabilities storecenter.RuntimeCapabilities) error {
	if capabilities.TikTokShop || capabilities.WalmartUS || capabilities.SheinLogin {
		return errors.New("core alignment only admits retained observations and operations cockpit")
	}
	if ctx == nil || db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" || confirmedDatabase == "" {
		return errors.New("explicit Store owner database required")
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	preflight := func(ctx context.Context, tx *sql.Tx) (bool, error) {
		if err := storecenter.VerifyCoreAlignmentBoundary(ctx, tx, confirmedDatabase, capabilities); err != nil {
			return false, err
		}
		applied, err := verifyCoreAlignmentHistory(ctx, tx)
		if err != nil {
			return false, err
		}
		current, err := storecenter.VerifyCoreAlignmentState(ctx, tx)
		if err != nil {
			return false, err
		}
		if applied && !current {
			return false, errors.New("Store core version is applied but CHECK contract differs")
		}
		return applied, nil
	}
	// Read-only replay cannot create even Goose's initial version table.
	tx, err := pool.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}
	applied, err := preflight(ctx, tx)
	rollbackErr := tx.Rollback()
	if err != nil {
		return err
	}
	if rollbackErr != nil {
		return rollbackErr
	}
	if applied {
		return nil
	}
	runner, err := newStoreRunner(pool, func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='25s'; LOCK TABLE public.workbench_stores IN ACCESS EXCLUSIVE MODE; LOCK TABLE public.workbench_store_merchant_bindings IN ACCESS EXCLUSIVE MODE`); err != nil {
			return errors.New("Store core table lock unavailable")
		}
		if _, err := preflight(ctx, tx); err != nil {
			return err
		}
		if err := storecenter.AlignCoreSchemaTx(ctx, tx); err != nil {
			return err
		}
		return storecenter.VerifyCoreAlignmentBoundary(ctx, tx, confirmedDatabase, capabilities)
	})
	if err != nil {
		return err
	}
	if _, err := runner.Up(ctx); err != nil {
		return errors.New("Store core Goose transaction failed; re-enter to verify committed state")
	}
	tx, err = pool.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	applied, err = preflight(ctx, tx)
	if err != nil {
		return err
	}
	if !applied {
		return errors.New("Store core version record unavailable")
	}
	return nil
}

func verifyCoreAlignmentHistory(ctx context.Context, tx *sql.Tx) (bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT version_id,is_applied FROM public.goose_store_center_version ORDER BY id`)
	if err != nil {
		return false, errors.New("original Store Goose history unavailable")
	}
	defer rows.Close()
	var versions []int64
	for rows.Next() {
		var version int64
		var applied bool
		if err := rows.Scan(&version, &applied); err != nil {
			return false, err
		}
		if !applied {
			return false, errors.New("original Store Goose history differs")
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	if slices.Equal(versions, []int64{0, 2026092801, 2026101101}) {
		return false, nil
	}
	if slices.Equal(versions, []int64{0, 2026092801, 2026101101, CoreAlignmentVersion}) {
		return true, nil
	}
	return false, errors.New("original Store Goose history differs")
}
