package storecenter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"strings"
)

const coreRuntimeSubject = "(SELECT current_user::text AS runtime_role) subject"

// VerifyCoreAlignmentBoundary is read-only. Maintenance checks the fixed
// runtime role's catalogs, never its secret or SET ROLE, using serving's query.
func VerifyCoreAlignmentBoundary(ctx context.Context, q coreSchemaQueryer, database string) error {
	var admitted bool
	err := q.QueryRowContext(ctx, `SELECT current_database()=$1 AND current_database() NOT IN ('postgres','template0','template1')
AND current_user='store_center_owner' AND session_user=current_user
AND current_schema()='public' AND current_schemas(false)=ARRAY['public']::name[]
AND NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member IN (SELECT oid FROM pg_roles WHERE rolname IN ('store_center_owner','store_center_runtime')))
AND (SELECT count(*)=12 AND bool_and(c.relowner=(SELECT oid FROM pg_roles WHERE rolname='store_center_owner') AND c.relkind='r' AND NOT c.relrowsecurity AND NOT c.relforcerowsecurity)
 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind IN ('r','p','v','m','f'))
AND NOT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('public','information_schema') AND n.nspname !~ '^pg_' AND c.relkind IN ('r','p','v','m','f'))`, database).Scan(&admitted)
	if err != nil || !admitted {
		return errors.New("retained Store database, original owner or twelve-table boundary mismatch")
	}
	query := strings.Replace(storeRuntimePermissionQuery, coreRuntimeSubject, "(SELECT 'store_center_runtime'::text AS runtime_role) subject", 1)
	var role string
	var required, forbidden bool
	if err := q.QueryRowContext(ctx, query, false, false, false, false).Scan(&role, &required, &forbidden); err != nil {
		return errors.New("retained Store runtime privileges unavailable")
	}
	if role != "store_center_owner" || !required || forbidden {
		return errors.New("retained Store runtime privileges mismatch")
	}
	var grantable bool
	err = q.QueryRowContext(ctx, `WITH grantees(oid) AS (SELECT oid FROM pg_roles WHERE rolname='store_center_runtime' UNION ALL SELECT 0::oid), effective_acl AS (
 SELECT p.* FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) p WHERE d.datname=current_database()
 UNION ALL SELECT p.* FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) p WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema'
 UNION ALL SELECT p.* FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(c.relacl) p WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema'
 UNION ALL SELECT p.* FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(a.attacl) p WHERE a.attnum>0 AND NOT a.attisdropped AND n.nspname !~ '^pg_' AND n.nspname<>'information_schema')
 SELECT EXISTS(SELECT 1 FROM effective_acl WHERE is_grantable AND grantee IN (SELECT oid FROM grantees))`).Scan(&grantable)
	if err != nil || grantable {
		return errors.New("retained Store runtime grant options mismatch")
	}
	return nil
}

// VerifyCoreAlignmentState accepts only the confirmed two-platform baseline A
// or all four current CHECKs B. It also verifies the unchanged core contract.
func VerifyCoreAlignmentState(ctx context.Context, q coreSchemaQueryer) (current bool, err error) {
	var platform, merchant string
	var extras int
	if err = q.QueryRowContext(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='public.workbench_stores'::regclass AND conname='store_platform' AND contype='c' AND convalidated AND NOT connoinherit`).Scan(&platform); err != nil {
		return false, errors.New("Store platform CHECK unavailable")
	}
	if err = q.QueryRowContext(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='public.workbench_store_merchant_bindings'::regclass AND conname='store_merchant_binding_shape' AND contype='c' AND convalidated AND NOT connoinherit`).Scan(&merchant); err != nil {
		return false, errors.New("Store merchant CHECK unavailable")
	}
	if err = q.QueryRowContext(ctx, `SELECT count(*) FROM pg_constraint WHERE conrelid='public.workbench_stores'::regclass AND conname IN ('store_tiktok_identity','store_walmart_market')`).Scan(&extras); err != nil {
		return false, err
	}
	definitions := currentConstraintDefinitions
	switch {
	case platform == currentConstraintDefinitions["store_platform"] && merchant == currentMerchantCheck && extras == 2:
		current = true
	case platform == baselinePlatformCheck && merchant == baselineMerchantCheck && extras == 0:
		definitions = maps.Clone(currentConstraintDefinitions)
		definitions["store_platform"] = baselinePlatformCheck
		delete(definitions, "store_tiktok_identity")
		delete(definitions, "store_walmart_market")
	default:
		return false, errors.New("Store core CHECKs are mixed or differ from the admitted baseline")
	}
	if err = verifyCoreSchema(ctx, q, RuntimeCapabilities{}, definitions); err != nil {
		return false, err
	}
	if err = verifyUnchangedCoreContract(ctx, q); err != nil {
		return false, err
	}
	return current, nil
}

// VerifyAlignedCoreSchemaTx is the new fresh-install version's verification
// callback. It must never silently align a retained baseline.
func VerifyAlignedCoreSchemaTx(ctx context.Context, tx *sql.Tx) error {
	current, err := VerifyCoreAlignmentState(ctx, tx)
	if err != nil {
		return err
	}
	if !current {
		return errors.New("Store core requires explicit --align-core-contract")
	}
	return nil
}

// AlignCoreSchemaTx alters only these four CHECKs. The app owner holds the
// Goose session lock and the two table locks until the version is committed.
func AlignCoreSchemaTx(ctx context.Context, tx *sql.Tx) error {
	current, err := VerifyCoreAlignmentState(ctx, tx)
	if err != nil || current {
		return err
	}
	for _, statement := range []string{
		`ALTER TABLE public.workbench_stores DROP CONSTRAINT store_platform,
 ADD CONSTRAINT store_platform ` + storePlatformCheckSQL + `,
 ADD CONSTRAINT store_tiktok_identity ` + storeTikTokCheckSQL + `,
 ADD CONSTRAINT store_walmart_market ` + storeWalmartCheckSQL,
		`ALTER TABLE public.workbench_store_merchant_bindings DROP CONSTRAINT store_merchant_binding_shape,
 ADD CONSTRAINT store_merchant_binding_shape ` + storeMerchantCheckSQL,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return errors.New("Store core CHECK alignment failed; transaction must roll back")
		}
	}
	return VerifyAlignedCoreSchemaTx(ctx, tx)
}

const storePlatformCheckSQL = `CHECK (platform IN ('shein','temu','tiktokshop','walmart'))`
const storeTikTokCheckSQL = `CHECK (platform<>'tiktokshop' OR (region='US' AND external_store_id<>''))`
const storeWalmartCheckSQL = `CHECK (platform<>'walmart' OR region='US')`
const storeMerchantCheckSQL = `CHECK (octet_length(identity_hash)=64 AND app_id<>'' AND ((platform='shein' AND open_key_id<>'') OR (platform IN ('temu','walmart') AND open_key_id='')))`

const baselinePlatformCheck = `CHECK ((platform = ANY (ARRAY['shein'::text, 'temu'::text])))`
const currentMerchantCheck = `CHECK (((octet_length((identity_hash)::text) = 64) AND ((app_id)::text <> ''::text) AND ((((platform)::text = 'shein'::text) AND ((open_key_id)::text <> ''::text)) OR (((platform)::text = ANY ((ARRAY['temu'::character varying, 'walmart'::character varying])::text[])) AND ((open_key_id)::text = ''::text)))))`
const baselineMerchantCheck = `CHECK (((octet_length((identity_hash)::text) = 64) AND ((app_id)::text <> ''::text) AND ((((platform)::text = 'shein'::text) AND ((open_key_id)::text <> ''::text)) OR (((platform)::text = 'temu'::text) AND ((open_key_id)::text = ''::text)))))`

func verifyUnchangedCoreContract(ctx context.Context, q coreSchemaQueryer) error {
	// Enumerate the fixed core objects as well as checking required definitions;
	// an extra constraint/index is drift rather than permission to repair it.
	coreTable := func(table string) bool {
		return table == "goose_store_center_version" || currentColumnNames[table] != nil && table != sheinLoginTable && !isTikTokSchemaTable(table) && !strings.HasPrefix(table, "workbench_store_walmart_")
	}
	constraints, err := q.QueryContext(ctx, `SELECT c.relname,k.conname FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid WHERE c.relnamespace='public'::regnamespace`)
	if err != nil {
		return err
	}
	for constraints.Next() {
		var table, name string
		if err := constraints.Scan(&table, &name); err != nil {
			constraints.Close()
			return err
		}
		if !coreTable(table) {
			continue
		}
		if table == "workbench_stores" && (name == "store_platform" || name == "store_tiktok_identity" || name == "store_walmart_market") || table == "workbench_store_merchant_bindings" && name == "store_merchant_binding_shape" {
			continue
		}
		if table == "workbench_stores" && currentConstraintDefinitions[name] != "" {
			continue
		}
		if _, ok := unchangedCoreConstraints[table+"."+name]; !ok {
			constraints.Close()
			return errors.New("unexpected Store core constraint")
		}
	}
	err = constraints.Err()
	constraints.Close()
	if err != nil {
		return err
	}
	indexes, err := q.QueryContext(ctx, `SELECT c.relname,i.relname FROM pg_index x JOIN pg_class c ON c.oid=x.indrelid JOIN pg_class i ON i.oid=x.indexrelid WHERE c.relnamespace='public'::regnamespace`)
	if err != nil {
		return err
	}
	for indexes.Next() {
		var table, name string
		if err := indexes.Scan(&table, &name); err != nil {
			indexes.Close()
			return err
		}
		if coreTable(table) {
			if _, ok := unchangedCoreIndexes[name]; !ok {
				indexes.Close()
				return errors.New("unexpected Store core index")
			}
		}
	}
	err = indexes.Err()
	indexes.Close()
	if err != nil {
		return err
	}
	for key, want := range unchangedCoreConstraints {
		table, name, _ := strings.Cut(key, ".")
		var definition string
		err := q.QueryRowContext(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid=('public.'||$1)::regclass AND conname=$2 AND convalidated AND NOT condeferrable AND NOT condeferred AND (contype<>'c' OR NOT connoinherit)`, table, name).Scan(&definition)
		if err != nil || definition != want {
			return fmt.Errorf("unchanged Store constraint mismatch: %s", key)
		}
	}
	for name, want := range unchangedCoreIndexes {
		var definition string
		err := q.QueryRowContext(ctx, `SELECT pg_get_indexdef(indexrelid) FROM pg_index WHERE indexrelid=to_regclass('public.'||$1) AND indisvalid AND indisready AND indislive`, name).Scan(&definition)
		if err != nil || definition != want {
			return fmt.Errorf("unchanged Store index mismatch: %s", name)
		}
	}
	rows, err := q.QueryContext(ctx, `SELECT c.relname,a.attname,coalesce(pg_get_expr(d.adbin,d.adrelid),''),a.attidentity::text,a.attgenerated::text FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum WHERE c.relnamespace='public'::regnamespace AND c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var table, column, expression, identity, generated string
		if err := rows.Scan(&table, &column, &expression, &identity, &generated); err != nil {
			return err
		}
		if !coreTable(table) {
			continue
		}
		if expression != coreColumnDefaults[table+"."+column] {
			return fmt.Errorf("unchanged Store column default mismatch: %s.%s", table, column)
		}
		if generated != "" || identity != "" && !(table == "goose_store_center_version" && column == "id" && identity == "d") {
			return errors.New("unchanged Store generated column mismatch")
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	var valid bool
	if err := q.QueryRowContext(ctx, `SELECT count(*)=4 AND bool_and((a.attname,a.atttypid::regtype::text,a.attnotnull,a.attidentity) IN (('id','integer',true,'d'),('version_id','bigint',true,''),('is_applied','boolean',true,''),('tstamp','timestamp without time zone',true,''))) FROM pg_attribute a WHERE a.attrelid='public.goose_store_center_version'::regclass AND a.attnum>0 AND NOT a.attisdropped`).Scan(&valid); err != nil || !valid {
		return errors.New("original Store Goose columns mismatch")
	}
	return nil
}
