package storecenter

import (
	"context"
	"errors"
)

// VerifyCurrentObservationsInstallBoundary is used only by the explicit owner
// installer on its pinned transaction. Optional platform capabilities stay off.
func VerifyCurrentObservationsInstallBoundary(ctx context.Context, q coreSchemaQueryer, database string, installed bool) error {
	if err := verifyCurrentObservationsObjectBoundary(ctx, q, database, installed); err != nil {
		return err
	}
	return verifyStoreMaintenancePermissions(ctx, q, RuntimeCapabilities{Observations: installed})
}

func verifyCurrentObservationsObjectBoundary(ctx context.Context, q coreSchemaQueryer, database string, installed bool) error {
	if ctx == nil || ctx.Err() != nil || q == nil || database == "" {
		return errors.New("explicit current Store database required")
	}
	var admitted bool
	err := q.QueryRowContext(ctx, `SELECT current_database()=$1 AND current_database() NOT IN ('postgres','template0','template1')
AND current_user='store_center_owner' AND session_user=current_user
AND current_schema()='public' AND current_schemas(false)=ARRAY['public']::name[]
AND (SELECT count(*)=2 AND bool_and(NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls)) FROM pg_roles WHERE rolname IN ('store_center_owner','store_center_runtime'))
AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member IN (SELECT oid FROM pg_roles WHERE rolname IN ('store_center_owner','store_center_runtime')))
AND NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname NOT IN ('public','information_schema') AND nspname !~ '^pg_' AND NOT($2 AND nspname='store_observations'))
AND NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='store_observations' AND nspowner<>(SELECT oid FROM pg_roles WHERE rolname=current_user))`, database, installed).Scan(&admitted)
	if err != nil || !admitted {
		return errors.New("current Store database, owner or namespace mismatch")
	}
	if err := verifyCurrentSchema(ctx, q, RuntimeCapabilities{}); err != nil {
		return err
	}
	if err := verifyUnchangedCoreContract(ctx, q); err != nil {
		return err
	}
	var versions string
	var applied bool
	if err := q.QueryRowContext(ctx, `SELECT string_agg(version_id::text,',' ORDER BY id),bool_and(is_applied) FROM public.goose_store_center_version`).Scan(&versions, &applied); err != nil || !applied || versions != "0,2026092801,2026101101,2026101601" && versions != "0,2026092801,2026101101,2026101601,2026102501" {
		return errors.New("current Store version history mismatch; no base migration admitted")
	}
	rows, err := q.QueryContext(ctx, `SELECT n.nspname,c.relname,c.relkind::text,c.relowner=(SELECT oid FROM pg_roles WHERE rolname=current_user),c.relrowsecurity OR c.relforcerowsecurity FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('public','store_observations') AND c.relkind IN ('r','p','v','m','f')`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var schema, name, kind string
		var owned, rls bool
		if err := rows.Scan(&schema, &name, &kind, &owned, &rls); err != nil {
			rows.Close()
			return err
		}
		allowed := schema == "public" && (currentColumnNames[name] != nil || name == "goose_store_center_version" || name == "workbench_store_browser_operations" || name == "workbench_store_browser_reports")
		allowed = allowed || installed && schema == "store_observations" && (name == "commands" || name == "syncs" || name == "records" || name == "heads")
		if !allowed || !owned || kind != "r" || rls {
			rows.Close()
			return errors.New("current Store object boundary mismatch")
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	return nil
}

// VerifyEmptyObservationsMaintenanceBoundary reuses the original owner/core and
// effective privilege checks; the observation owner must separately verify its
// complete A/B structure on the same pinned transaction before any DDL.
func VerifyEmptyObservationsMaintenanceBoundary(ctx context.Context, q coreSchemaQueryer, database string) error {
	if err := verifyCurrentObservationsObjectBoundary(ctx, q, database, true); err != nil {
		return err
	}
	var versions string
	if err := q.QueryRowContext(ctx, `SELECT string_agg(version_id::text,',' ORDER BY id) FROM public.goose_store_center_version`).Scan(&versions); err != nil || versions != "0,2026092801,2026101101,2026101601" {
		return errors.New("empty observation maintenance requires exact original Store history")
	}
	if err := verifyStoreMaintenancePrivilegeCatalog(ctx, q, RuntimeCapabilities{Observations: true}); err != nil {
		return err
	}
	return verifyNativeCapabilityGrants(ctx, q, "store_observations")
}

// ReadObservationsAlignmentCatalog exposes the existing immutable owner projection,
// not a caller-selected schema or a relaxed verification mode.
func ReadObservationsAlignmentCatalog(ctx context.Context, q coreSchemaQueryer) ([]byte, []byte, error) {
	var raw string
	if err := q.QueryRowContext(ctx, retainedCapabilityCatalogQuery, "store_observations").Scan(&raw); err != nil {
		return nil, nil, err
	}
	return []byte(raw), append([]byte(nil), retainedObservationsContract...), nil
}
