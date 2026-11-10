package storecenter

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
	observations "task-processor/internal/integration/persistence/storeobservations"
	"task-processor/internal/storecenter"
)

// AlignEmptyObservationsContract is explicit owner maintenance. Serving never
// calls it. The original identity, privilege and core guards stay in force.
func AlignEmptyObservationsContract(ctx context.Context, db *gorm.DB, database string) error {
	if ctx == nil || db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" || database == "" {
		return errors.New("explicit Store owner database confirmation required")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := db.WithContext(ctx).Transaction(func(g *gorm.DB) error {
		tx, ok := g.Statement.ConnPool.(*sql.Tx)
		if !ok {
			return errors.New("pinned Store transaction required")
		}
		if _, err := tx.ExecContext(ctx, `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='25s'; SELECT pg_advisory_xact_lock(hashtextextended(current_database()||':store-observations-install',0)); LOCK TABLE store_observations.commands,store_observations.heads,store_observations.records,store_observations.syncs IN ACCESS EXCLUSIVE MODE`); err != nil {
			return err
		}
		if err := storecenter.VerifyEmptyObservationsMaintenanceBoundary(ctx, tx, database); err != nil {
			return err
		}
		before, err := observationPreservation(ctx, tx)
		if err != nil {
			return err
		}
		if err = observations.AlignEmptyContractTx(ctx, tx); err != nil {
			return err
		}
		if err = storecenter.VerifyEmptyObservationsMaintenanceBoundary(ctx, tx, database); err != nil {
			return err
		}
		after, err := observationPreservation(ctx, tx)
		if err != nil {
			return err
		}
		if before != after {
			return errors.New("observation preservation postcondition differs")
		}
		return nil

	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(g *gorm.DB) error {
		tx, ok := g.Statement.ConnPool.(*sql.Tx)
		if !ok {
			return errors.New("pinned read-only transaction required")
		}
		if err := storecenter.VerifyEmptyObservationsMaintenanceBoundary(ctx, tx, database); err != nil {
			return err
		}
		current, err := observations.InspectEmptyContractTx(ctx, tx)
		if err != nil {
			return err
		}
		if !current {
			return errors.New("committed observation contract unavailable; inspect original operation")
		}
		return nil
	}, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
}

// Catalog preservation excludes only the four replaced CHECK objects. No rows
// or permissions are written by this operation, including the original history.
func observationPreservation(ctx context.Context, tx *sql.Tx) (string, error) {
	var fingerprint string
	err := tx.QueryRowContext(ctx, `SELECT jsonb_build_object(
 'database',(SELECT to_jsonb(d)-'datfrozenxid'-'datminmxid' FROM pg_database d WHERE datname=current_database()),
 'schemas',(SELECT jsonb_agg(to_jsonb(n) ORDER BY oid) FROM pg_namespace n WHERE nspname IN('public','store_observations','operations_cockpit')),
 'tables',(SELECT jsonb_agg(jsonb_build_array(oid,relname,relowner,relacl) ORDER BY oid) FROM pg_class WHERE relnamespace IN(SELECT oid FROM pg_namespace WHERE nspname IN('public','store_observations','operations_cockpit'))),
 'columnACL',(SELECT jsonb_agg(jsonb_build_array(attrelid,attnum,attacl) ORDER BY attrelid,attnum) FROM pg_attribute WHERE attrelid IN(SELECT oid FROM pg_class WHERE relnamespace IN(SELECT oid FROM pg_namespace WHERE nspname IN('public','store_observations','operations_cockpit'))) AND attnum>0 AND NOT attisdropped),
 'constraints',(SELECT jsonb_agg(to_jsonb(k) ORDER BY oid) FROM pg_constraint k WHERE connamespace IN(SELECT oid FROM pg_namespace WHERE nspname IN('public','store_observations','operations_cockpit')) AND NOT(connamespace='store_observations'::regnamespace AND conname IN('heads_kind_check','records_kind_check','syncs_kind_check','store_observation_record_variant'))),
 'defaults',(SELECT jsonb_agg(to_jsonb(d) ORDER BY oid) FROM pg_default_acl d),
 'roles',(SELECT jsonb_agg(to_jsonb(r) ORDER BY rolname) FROM pg_roles r WHERE rolname IN('store_center_owner','store_center_runtime')),
 'members',(SELECT jsonb_agg(to_jsonb(m) ORDER BY oid) FROM pg_auth_members m WHERE member IN(SELECT oid FROM pg_roles WHERE rolname IN('store_center_owner','store_center_runtime'))),
 'history',(SELECT jsonb_agg(to_jsonb(v) ORDER BY id) FROM public.goose_store_center_version v)
)::text`).Scan(&fingerprint)
	return fingerprint, err
}
