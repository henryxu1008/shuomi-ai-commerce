package researchpersistence

import (
	"context"
	"gorm.io/gorm"
	"strings"
	"task-processor/internal/product/research"
)

const RuntimeRole = "product_research_runtime"

var updates = map[string]string{"research_opportunities": "selected_link_id,state,reason,revision", "research_evidence_links": "removed"}

func GrantRuntime(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return research.ErrUnavailable
	}
	var unsafe bool
	if db.Raw(`SELECT rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r.oid) FROM pg_roles r WHERE rolname=?`, RuntimeRole).Scan(&unsafe).Error != nil || unsafe {
		return research.ErrUnavailable
	}
	for _, table := range Tables {
		if e := db.Exec("REVOKE ALL ON TABLE public." + table + " FROM " + RuntimeRole).Error; e != nil {
			return e
		}
		if e := db.Exec("GRANT SELECT,INSERT ON TABLE public." + table + " TO " + RuntimeRole).Error; e != nil {
			return e
		}
		if columns := updates[table]; columns != "" {
			if e := db.Exec("GRANT UPDATE (" + columns + ") ON TABLE public." + table + " TO " + RuntimeRole).Error; e != nil {
				return e
			}
		}
	}
	return db.Exec("GRANT USAGE ON SCHEMA public TO " + RuntimeRole).Error
}
func VerifyRuntimePermissions(ctx context.Context, db *gorm.DB) error {
	var safe bool
	if db.WithContext(ctx).Raw(`SELECT current_user=? AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r.oid) AND NOT has_schema_privilege(current_user,'public','CREATE') FROM pg_roles r WHERE rolname=current_user`, RuntimeRole).Scan(&safe).Error != nil || !safe {
		return research.ErrUnavailable
	}
	for _, table := range Tables {
		if db.WithContext(ctx).Raw(`SELECT has_table_privilege(current_user,?,'SELECT') AND has_table_privilege(current_user,?,'INSERT') AND NOT has_table_privilege(current_user,?,'DELETE,TRUNCATE,REFERENCES,TRIGGER')`, "public."+table, "public."+table, "public."+table).Scan(&safe).Error != nil || !safe {
			return research.ErrUnavailable
		}
		excluded := ""
		if columns := updates[table]; columns != "" {
			excluded = " AND attname NOT IN ('" + strings.ReplaceAll(columns, ",", "','") + "')"
		}
		var unsafe bool
		if db.WithContext(ctx).Raw("SELECT EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid=?::regclass AND attnum>0 AND NOT attisdropped"+excluded+" AND has_column_privilege(current_user,attrelid,attname,'UPDATE'))", "public."+table).Scan(&unsafe).Error != nil || unsafe {
			return research.ErrUnavailable
		}
		for _, column := range strings.Split(updates[table], ",") {
			if column != "" {
				if db.WithContext(ctx).Raw("SELECT has_column_privilege(current_user,?,?, 'UPDATE')", "public."+table, column).Scan(&safe).Error != nil || !safe {
					return research.ErrUnavailable
				}
			}
		}
	}
	var supplierPresent bool
	if db.WithContext(ctx).Raw("SELECT to_regclass(?) IS NOT NULL", supplierTable).Scan(&supplierPresent).Error != nil {
		return research.ErrUnavailable
	}
	if supplierPresent {
		if VerifySupplierReviewSchema(ctx, db) != nil {
			return research.ErrUnavailable
		}
		if db.WithContext(ctx).Raw(`SELECT has_table_privilege(current_user,?,'SELECT')
 AND has_table_privilege(current_user,?,'INSERT')
 AND NOT has_table_privilege(current_user,?,'UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
 AND NOT has_any_column_privilege(current_user,?,'UPDATE,REFERENCES,SELECT WITH GRANT OPTION,INSERT WITH GRANT OPTION')
 AND NOT EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(c.relacl) acl
    WHERE c.oid=to_regclass(?) AND acl.grantee NOT IN (c.relowner,(SELECT oid FROM pg_roles WHERE rolname=current_user)))
 AND NOT EXISTS(SELECT 1 FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid
    CROSS JOIN LATERAL aclexplode(a.attacl) acl
    WHERE a.attrelid=to_regclass(?) AND acl.grantee<>c.relowner)`, supplierTable, supplierTable, supplierTable, supplierTable, supplierTable, supplierTable).Scan(&safe).Error != nil || !safe {
			return research.ErrUnavailable
		}
	}
	var unsafe bool
	if db.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT LIKE 'pg_%' AND n.nspname<>'information_schema' AND c.relname NOT IN ('research_opportunities','research_evidence_links','research_commands') AND NOT (? AND c.oid=to_regclass(?)) AND c.relkind IN ('r','p','v','m','f') AND (has_table_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR has_any_column_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))`, supplierPresent, supplierTable).Scan(&unsafe).Error != nil || unsafe {
		return research.ErrUnavailable
	}
	return nil
}
