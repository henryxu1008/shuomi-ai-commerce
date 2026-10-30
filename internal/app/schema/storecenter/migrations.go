package storecenter

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"

	platformmigration "task-processor/internal/platform/database/migration"
	"task-processor/internal/storecenter"
)

func Migrate(ctx context.Context, db *gorm.DB) error {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return errors.New("store schema requires PostgreSQL")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	var existing bool
	if err := sqlDB.QueryRowContext(ctx, `SELECT to_regclass('public.workbench_stores') IS NOT NULL`).Scan(&existing); err != nil {
		return err
	}
	if existing {
		if err := storecenter.VerifyCurrentSchema(ctx, db); err != nil {
			// The first Goose transaction creates the core before metadata. Its
			// committed receipt is installation progress, not a retained baseline.
			tx, readErr := sqlDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
			if readErr != nil {
				return readErr
			}
			versions, readErr := appliedStoreVersions(ctx, tx)
			rollbackErr := tx.Rollback()
			if readErr != nil || rollbackErr != nil || !slices.Equal(versions, []int64{0, 2026092801}) {
				return errors.New("retained Store schema requires its explicit admitted maintenance entry")
			}
		}
	}
	runner, err := newStoreRunner(sqlDB, storecenter.VerifyAlignedCoreSchemaTx)
	if err != nil {
		return err
	}
	if _, err = runner.Up(ctx); err != nil {
		return err
	}
	return storecenter.VerifyCurrentSchema(ctx, db)
}

func newStoreRunner(db *sql.DB, align func(context.Context, *sql.Tx) error) (*platformmigration.Runner, error) {
	return platformmigration.NewWithVersionTable(goose.DialectPostgres, db, "public.goose_store_center_version",
		goose.NewGoMigration(2026092801, &goose.GoFunc{RunTx: storecenter.InstallCoreSchemaTx}, nil),
		goose.NewGoMigration(2026101101, &goose.GoFunc{RunTx: func(ctx context.Context, tx *sql.Tx) error {
			if err := storecenter.InstallMetadataSchemaTx(ctx, tx); err != nil {
				return err
			}
			// Keep the resumed step and its original Goose receipt atomic. A
			// retained/drifted core must not commit newly created metadata.
			if err := storecenter.VerifyAlignedCoreSchemaTx(ctx, tx); err != nil {
				return err
			}
			return storecenter.VerifyCurrentSchemaTx(ctx, tx)
		}}, nil),
		goose.NewGoMigration(CoreAlignmentVersion, &goose.GoFunc{RunTx: align}, nil))
}
