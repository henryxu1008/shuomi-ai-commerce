package storecenter

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
)

// InstallBrowserSchemaTx is invoked only by the existing Store schema owner.
// It contains no data adoption, historical rewriting, or runtime migration.
func InstallBrowserSchemaTx(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`CREATE TABLE public.workbench_store_browser_operations (
 organization_id VARCHAR(200) NOT NULL,operation_id CHAR(36) NOT NULL,store_id CHAR(36) NOT NULL,
 issuer TEXT NOT NULL,actor_id VARCHAR(200) NOT NULL,member_id VARCHAR(200) NOT NULL,
 action VARCHAR(32) NOT NULL,binding_id CHAR(36) NOT NULL,profile_id CHAR(36) NOT NULL,binding_version VARCHAR(19) NOT NULL,
 fingerprint VARCHAR(64) NOT NULL,store_version BIGINT NOT NULL,admitted_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY(organization_id,operation_id),
 CONSTRAINT store_browser_operation_store FOREIGN KEY(organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
 CONSTRAINT store_browser_operation_action CHECK(action IN('configure_environment','open_environment')),
 CONSTRAINT store_browser_operation_shape CHECK(issuer<>'' AND actor_id<>'' AND member_id<>'' AND store_version>0 AND octet_length(fingerprint)=64 AND binding_version ~ '^[1-9][0-9]{0,18}$'))`,
		`CREATE INDEX idx_store_browser_operation_actor ON public.workbench_store_browser_operations(store_id,operation_id,issuer,actor_id)`,
		`CREATE TABLE public.workbench_store_browser_reports (
 organization_id VARCHAR(200) NOT NULL,operation_id CHAR(36) NOT NULL,kind VARCHAR(32) NOT NULL,
 outcome VARCHAR(32) NOT NULL,reason_code VARCHAR(40) NOT NULL,observed_at TIMESTAMPTZ NOT NULL,received_at TIMESTAMPTZ NOT NULL,fingerprint VARCHAR(64) NOT NULL,
 PRIMARY KEY(organization_id,operation_id,kind),
 CONSTRAINT store_browser_report_operation FOREIGN KEY(organization_id,operation_id) REFERENCES public.workbench_store_browser_operations(organization_id,operation_id),
 CONSTRAINT store_browser_report_kind CHECK(kind IN('configure_result','open_result','close_result')),
 CONSTRAINT store_browser_report_outcome CHECK(outcome IN('succeeded','failed','unknown')),
 CONSTRAINT store_browser_report_fingerprint CHECK(octet_length(fingerprint)=64))`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

// Missing optional tables keep the current Store usable with browser endpoints
// unavailable. Partial/corrupt installations fail closed. Serving never installs.
func VerifyBrowserSchemaIfPresent(ctx context.Context, q coreSchemaQueryer) (bool, error) {
	var count int
	var shape bool
	if err := q.QueryRowContext(ctx, `SELECT count(*),coalesce(bool_and(c.relkind='r' AND NOT c.relrowsecurity AND NOT c.relforcerowsecurity),false) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname IN ('workbench_store_browser_operations','workbench_store_browser_reports')`).Scan(&count, &shape); err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}
	if count != 2 || !shape {
		return false, errors.New("Store browser schema incomplete")
	}
	for table, want := range map[string]string{
		"workbench_store_browser_operations": "organization_id operation_id store_id issuer actor_id member_id action binding_id profile_id binding_version fingerprint store_version admitted_at",
		"workbench_store_browser_reports":    "organization_id operation_id kind outcome reason_code observed_at received_at fingerprint",
	} {
		rows, err := q.QueryContext(ctx, `SELECT a.attname,format_type(a.atttypid,a.atttypmod),a.attnotnull FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_class c ON c.oid=a.attrelid JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname=$1 AND a.attnum>0 AND NOT a.attisdropped ORDER BY a.attnum`, table)
		if err != nil {
			return false, err
		}
		var names []string
		for rows.Next() {
			var name, kind string
			var required bool
			if err = rows.Scan(&name, &kind, &required); err != nil {
				break
			}
			names = append(names, name)
			expected := "character varying(200)"
			switch name {
			case "operation_id", "store_id", "binding_id", "profile_id":
				expected = "character(36)"
			case "issuer":
				expected = "text"
			case "action", "kind", "outcome":
				expected = "character varying(32)"
			case "binding_version":
				expected = "character varying(19)"
			case "reason_code":
				expected = "character varying(40)"
			case "fingerprint":
				expected = "character varying(64)"
			case "store_version":
				expected = "bigint"
			case "admitted_at", "observed_at", "received_at":
				expected = "timestamp with time zone"
			}
			if !required || kind != expected {
				err = errors.New("Store browser schema column mismatch")
				break
			}
		}
		rowErr := rows.Err()
		rows.Close()
		if err != nil {
			return false, err
		}
		if rowErr != nil {
			return false, rowErr
		}
		if !slices.Equal(names, strings.Fields(want)) {
			return false, errors.New("Store browser schema shape mismatch")
		}
	}
	for _, b := range []struct{ table, name, definition string }{
		{"workbench_store_browser_operations", "workbench_store_browser_operations_pkey", "PRIMARY KEY (organization_id, operation_id)"},
		{"workbench_store_browser_reports", "workbench_store_browser_reports_pkey", "PRIMARY KEY (organization_id, operation_id, kind)"},
		{"workbench_store_browser_operations", "store_browser_operation_store", "FOREIGN KEY (organization_id, store_id) REFERENCES workbench_stores(organization_id, id)"},
		{"workbench_store_browser_reports", "store_browser_report_operation", "FOREIGN KEY (organization_id, operation_id) REFERENCES workbench_store_browser_operations(organization_id, operation_id)"},
	} {
		var definition string
		if err := q.QueryRowContext(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_catalog.pg_constraint WHERE conrelid=('public.'||$1)::regclass AND conname=$2 AND convalidated`, b.table, b.name).Scan(&definition); err != nil || strings.Join(strings.Fields(definition), " ") != b.definition {
			return false, errors.New("Store browser immutable ownership constraint mismatch")
		}
	}
	return true, nil
}

// Owner maintenance verifies the serving role, including inherited column
// privileges, before reporting an immutable installation as complete.
func VerifyBrowserRuntimeGrants(ctx context.Context, q coreSchemaQueryer) error {
	var valid bool
	err := q.QueryRowContext(ctx, `SELECT bool_and(coalesce(has_table_privilege('store_center_runtime',to_regclass('public.'||name),'SELECT') AND has_table_privilege('store_center_runtime',to_regclass('public.'||name),'INSERT') AND NOT has_any_column_privilege('store_center_runtime',to_regclass('public.'||name),'UPDATE') AND NOT has_any_column_privilege('store_center_runtime',to_regclass('public.'||name),'REFERENCES') AND NOT has_table_privilege('store_center_runtime',to_regclass('public.'||name),'DELETE,TRUNCATE,TRIGGER'),false)) FROM (VALUES ('workbench_store_browser_operations'),('workbench_store_browser_reports')) t(name)`).Scan(&valid)
	if err != nil || !valid {
		return errors.New("Store browser runtime immutable grants mismatch")
	}
	return nil
}
