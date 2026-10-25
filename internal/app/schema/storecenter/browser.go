package storecenter

import (
	"context"
	"database/sql"
	"errors"
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
	platformmigration "task-processor/internal/platform/database/migration"
	"task-processor/internal/storecenter"
)

// InstallBrowserWorkbench is a separately selected mode of the existing Store
// schema owner. It never changes core alignment, facts or existing role grants.
func InstallBrowserWorkbench(ctx context.Context, db *gorm.DB, database string) error {
	if db == nil || db.Dialector.Name() != "postgres" || database == "" {
		return errors.New("explicit Store owner database required")
	}
	if err := storecenter.VerifyCurrentSchema(ctx, db); err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	var owner bool
	if err = pool.QueryRowContext(ctx, `SELECT current_database()=$1 AND current_user='store_center_owner' AND session_user=current_user AND current_schema()='public' AND current_schemas(false)=ARRAY['public']::name[] AND NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))`, database).Scan(&owner); err != nil || !owner {
		return errors.New("Store browser owner confirmation mismatch")
	}
	// Pin the original applied history. Missing base steps must never be run by
	// this mode; only the new two-table version may execute.
	var versions string
	if err = pool.QueryRowContext(ctx, `SELECT string_agg(version_id::text,',' ORDER BY id) FROM public.goose_store_center_version WHERE is_applied`).Scan(&versions); err != nil {
		return err
	}
	if versions != "0,2026092801,2026101101,2026101601" && versions != "0,2026092801,2026101101,2026101601,2026102501" {
		return errors.New("Store browser base version mismatch")
	}
	noBase := func(context.Context, *sql.Tx) error {
		return errors.New("Store browser cannot install historical base schema")
	}
	install := func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='25s'`); err != nil {
			return err
		}
		if err := storecenter.InstallBrowserSchemaTx(ctx, tx); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `GRANT SELECT,INSERT ON public.workbench_store_browser_operations,public.workbench_store_browser_reports TO store_center_runtime`); err != nil {
			return err
		}
		_, err := storecenter.VerifyBrowserSchemaIfPresent(ctx, tx)
		return err
	}
	runner, err := platformmigration.NewWithVersionTable(goose.DialectPostgres, pool, "public.goose_store_center_version",
		goose.NewGoMigration(2026092801, &goose.GoFunc{RunTx: noBase}, nil), goose.NewGoMigration(2026101101, &goose.GoFunc{RunTx: noBase}, nil), goose.NewGoMigration(CoreAlignmentVersion, &goose.GoFunc{RunTx: noBase}, nil), goose.NewGoMigration(2026102501, &goose.GoFunc{RunTx: install}, nil))
	if err != nil {
		return err
	}
	if _, err = runner.Up(ctx); err != nil {
		return err
	}
	installed, err := storecenter.VerifyBrowserSchemaIfPresent(ctx, pool)
	if err != nil {
		return err
	}
	if !installed {
		return errors.New("Store browser schema version applied without tables")
	}
	return storecenter.VerifyBrowserRuntimeGrants(ctx, pool)
}
