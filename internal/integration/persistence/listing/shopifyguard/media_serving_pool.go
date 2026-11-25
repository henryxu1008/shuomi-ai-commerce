package shopifyguard

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"time"

	"gorm.io/gorm"
)

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
