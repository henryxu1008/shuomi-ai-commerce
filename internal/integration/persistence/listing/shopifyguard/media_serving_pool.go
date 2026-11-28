package shopifyguard

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"strings"
	"time"

	"gorm.io/gorm"
)

const MediaServingRole = "shopify_media_runtime"

type MediaTrialInstallState int

const (
	MediaTrialInstallFresh MediaTrialInstallState = iota + 1
	MediaTrialInstallExisting
)

// InspectMediaTrialInstall distinguishes a new schema from a complete prior
// install without opening the initializer role or changing persistent state.
func InspectMediaTrialInstall(ctx context.Context, db *gorm.DB) (MediaTrialInstallState, error) {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return 0, ErrUnavailable
	}
	var base, frozen, restricted, dispatches, plans, steps bool
	if err := db.WithContext(ctx).Raw(`SELECT
		to_regclass('public.listing_submission_execution_attempts') IS NOT NULL,
		to_regclass('public.listing_shopify_frozen_generations') IS NOT NULL,
		to_regclass('public.listing_shopify_restricted_sources') IS NOT NULL,
		to_regclass('public.listing_shopify_dispatches') IS NOT NULL,
		to_regclass(?) IS NOT NULL,
		to_regclass(?) IS NOT NULL`, MediaSyncPlanTable, MediaSyncStepTable).Row().Scan(&base, &frozen, &restricted, &dispatches, &plans, &steps); err != nil || !base {
		return 0, ErrUnavailable
	}
	installed := []bool{frozen, restricted, dispatches, plans, steps}
	present := 0
	for _, exists := range installed {
		if exists {
			present++
		}
	}
	if present == 0 {
		return MediaTrialInstallFresh, nil
	}
	if present != len(installed) {
		return 0, ErrUnavailable
	}
	if err := VerifyIsolatedMediaServingPool(ctx, db); err != nil {
		return 0, err
	}
	return MediaTrialInstallExisting, nil
}

// VerifyIsolatedMediaServingPool verifies the exact low-privilege Product
// media role without installing tables, issuing a witness, or granting writes.
func VerifyIsolatedMediaServingPool(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return ErrUnavailable
	}
	pool, err := db.DB()
	if err != nil {
		return ErrUnavailable
	}
	tx, err := pool.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	var ready bool
	if err := tx.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL AND to_regclass($2) IS NOT NULL", MediaSyncPlanTable, MediaSyncStepTable).Scan(&ready); err != nil || !ready || !admittedMediaServingRole(ctx, tx) {
		return ErrUnavailable
	}
	return nil
}

// GrantIsolatedMediaServingPermissions admits the pre-provisioned fixed role
// only to the isolated media ledger and same-generation freeze tables. The
// Product source-serving role remains a separate pool owned by Product.
func GrantIsolatedMediaServingPermissions(ctx context.Context, initializer *gorm.DB) error {
	if ctx == nil || initializer == nil || initializer.Dialector.Name() != "postgres" {
		return ErrUnavailable
	}
	return initializer.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exists bool
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=?) AND
			to_regclass(?) IS NOT NULL AND to_regclass(?) IS NOT NULL AND
			to_regclass('public.listing_shopify_dispatches') IS NOT NULL AND
			to_regclass('public.listing_shopify_frozen_generations') IS NOT NULL AND
			to_regclass('public.listing_shopify_restricted_sources') IS NOT NULL`,
			MediaServingRole, MediaSyncPlanTable, MediaSyncStepTable).Row().Scan(&exists); err != nil || !exists {
			return ErrUnavailable
		}
		statements := []string{
			"REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM " + MediaServingRole,
			"REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM " + MediaServingRole,
			"REVOKE CREATE ON SCHEMA public FROM " + MediaServingRole,
			"GRANT USAGE ON SCHEMA public TO " + MediaServingRole,
			"GRANT SELECT,INSERT,UPDATE ON public.listing_submission_execution_attempts,public.listing_submission_target_fences TO " + MediaServingRole,
			"GRANT SELECT ON public.listing_submission_official_intents TO " + MediaServingRole,
			"GRANT SELECT,INSERT ON public.listing_shopify_frozen_generations,public.listing_shopify_restricted_sources TO " + MediaServingRole,
			"GRANT SELECT,INSERT,UPDATE ON public.listing_shopify_dispatches,public.listing_shopify_media_sync_plans TO " + MediaServingRole,
			"GRANT SELECT,INSERT ON public.listing_shopify_media_sync_steps TO " + MediaServingRole,
			"REVOKE CREATE,TEMPORARY ON DATABASE " + quoteIdentifierFromCurrentDatabase(tx) + " FROM " + MediaServingRole,
		}
		for _, statement := range statements {
			if strings.Contains(statement, "\x00") || tx.Exec(statement).Error != nil {
				return ErrUnavailable
			}
		}
		if err := tx.Exec("DO $$ BEGIN EXECUTE format('REVOKE TEMPORARY ON DATABASE %I FROM PUBLIC', current_database()); END $$").Error; err != nil {
			return ErrUnavailable
		}
		return nil
	})
}

func quoteIdentifierFromCurrentDatabase(tx *gorm.DB) string {
	var database string
	if tx.Raw("SELECT current_database()").Row().Scan(&database) != nil || database == "" {
		return `""`
	}
	return `"` + strings.ReplaceAll(database, `"`, `""`) + `"`
}

// HandoffIsolatedMediaServingPool moves the in-process installation witness
// to the dedicated media role before Shopify work begins. The initializer
// must close its elevated pool before enabling any request handlers.
func HandoffIsolatedMediaServingPool(ctx context.Context, initializer, serving *gorm.DB, live *LiveIsolation) error {
	if ctx == nil || initializer == nil || serving == nil || live == nil || live.state == nil {
		return ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	state := live.state
	state.mu.Lock()
	defer state.mu.Unlock()
	original, err := initializer.DB()
	if err != nil || original != state.installer || !live.applies(original) || state.journal {
		return ErrUnavailable
	}
	next, err := serving.DB()
	if err != nil || next == original {
		return ErrUnavailable
	}
	var secret [8]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return ErrUnavailable
	}
	key := int64(binary.BigEndian.Uint64(secret[:]))

	setup, err := original.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer setup.Rollback()
	if _, err := setup.ExecContext(ctx, "LOCK TABLE public.listing_submission_execution_attempts IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return ErrUnavailable
	}
	var used bool
	if err := setup.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM public.listing_submission_execution_attempts WHERE platform='shopify')").Scan(&used); err != nil || used {
		return ErrUnavailable
	}
	proof, err := original.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return ErrUnavailable
	}
	defer proof.Rollback()
	if _, err := proof.ExecContext(ctx, "SELECT pg_catalog.pg_advisory_xact_lock($1)", key); err != nil {
		return ErrUnavailable
	}

	runtime, err := next.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return ErrUnavailable
	}
	defer runtime.Rollback()
	if !challengeSharedDatabase(ctx, proof, runtime, key) || !admittedMediaServingRole(ctx, runtime) {
		return ErrUnavailable
	}
	if err := runtime.Commit(); err != nil {
		return ErrUnavailable
	}
	state.pool.Store(next)
	if err := setup.Commit(); err != nil {
		state.pool.Store(nil)
		return ErrUnavailable
	}
	return nil
}

func challengeSharedDatabase(ctx context.Context, holder, candidate *sql.Tx, key int64) bool {
	var acquired bool
	const try = "SELECT pg_catalog.pg_try_advisory_xact_lock($1)"
	if err := candidate.QueryRowContext(ctx, try, key).Scan(&acquired); err != nil || acquired {
		return false
	}
	if err := holder.Commit(); err != nil {
		return false
	}
	return candidate.QueryRowContext(ctx, try, key).Scan(&acquired) == nil && acquired
}

func admittedMediaServingRole(ctx context.Context, serving *sql.Tx) bool {
	var admitted bool
	err := serving.QueryRowContext(ctx, mediaServingPrivileges).Scan(&admitted)
	return err == nil && admitted
}

// The normal Product runtime keeps its existing permissions. This role has
// only the storage needed for the media journal and its existing freeze owner.
const mediaServingPrivileges = `WITH allowed(table_name,privilege) AS (VALUES
	('listing_submission_execution_attempts','SELECT'),('listing_submission_execution_attempts','INSERT'),('listing_submission_execution_attempts','UPDATE'),
	('listing_submission_target_fences','SELECT'),('listing_submission_target_fences','INSERT'),('listing_submission_target_fences','UPDATE'),
	('listing_submission_official_intents','SELECT'),
	('listing_shopify_frozen_generations','SELECT'),('listing_shopify_frozen_generations','INSERT'),
	('listing_shopify_restricted_sources','SELECT'),('listing_shopify_restricted_sources','INSERT'),
	('listing_shopify_dispatches','SELECT'),('listing_shopify_dispatches','INSERT'),('listing_shopify_dispatches','UPDATE'),
	('listing_shopify_media_sync_plans','SELECT'),('listing_shopify_media_sync_plans','INSERT'),('listing_shopify_media_sync_plans','UPDATE'),
	('listing_shopify_media_sync_steps','SELECT'),('listing_shopify_media_sync_steps','INSERT')),
	tables AS (SELECT c.*,n.nspname FROM pg_class c JOIN pg_namespace n ON c.relnamespace=n.oid
		WHERE n.nspname NOT LIKE 'pg_%' AND n.nspname<>'information_schema')
	SELECT session_user=current_user AND current_schemas(false)=ARRAY['public']::name[]
	AND pg_my_temp_schema()=0 AND has_schema_privilege('public','USAGE')
	AND NOT has_database_privilege(current_database(),'CREATE,TEMPORARY')
	AND EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolcanlogin AND NOT
		(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
	AND NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE member=(SELECT oid FROM pg_roles WHERE rolname=current_user))
	AND NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname NOT LIKE 'pg_%' AND nspname<>'information_schema'
		AND (has_schema_privilege(oid,'CREATE') OR nspname<>'public' AND has_schema_privilege(oid,'USAGE')))
	AND NOT EXISTS (SELECT 1 FROM allowed a LEFT JOIN tables t ON t.nspname='public' AND t.relname=a.table_name
		WHERE t.oid IS NULL OR t.relkind<>'r' OR t.relrowsecurity OR t.relforcerowsecurity
		OR t.relowner=(SELECT oid FROM pg_roles WHERE rolname=current_user)
		OR NOT has_table_privilege(t.oid,a.privilege))
	AND NOT EXISTS (SELECT 1 FROM tables t CROSS JOIN LATERAL aclexplode(acldefault('r',t.relowner)) p
		WHERE t.relkind IN ('r','p','v','m','f')
		AND CASE WHEN p.privilege_type IN ('SELECT','INSERT','UPDATE','REFERENCES')
			THEN has_any_column_privilege(t.oid,p.privilege_type) ELSE has_table_privilege(t.oid,p.privilege_type) END
		AND (has_table_privilege(t.oid,p.privilege_type||' WITH GRANT OPTION')
			OR NOT EXISTS (SELECT 1 FROM allowed a WHERE t.nspname='public' AND t.relname=a.table_name AND p.privilege_type=a.privilege)))
	AND NOT EXISTS (SELECT 1 FROM tables t WHERE CASE WHEN t.relkind='S' THEN has_sequence_privilege(t.oid,'USAGE,SELECT,UPDATE') ELSE false END)`
