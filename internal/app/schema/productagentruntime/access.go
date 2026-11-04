package productagentruntime

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"
)

func quote(s string) string { return pgx.Identifier{s}.Sanitize() }

type tableGrant struct {
	Role, Table    string
	Select, Insert bool
	Update         []string
}

func runGrants(ledger relationContract) []tableGrant {
	var cols []string
	for _, c := range ledger.Columns {
		cols = append(cols, c.Name)
	}
	return []tableGrant{
		{"product_agent_runtime", "ai_invocations", true, true, cols},
		{"product_agent_runtime", "product_agent_runs", true, true, []string{"revision", "phase", "payload"}},
		{"product_agent_runtime", "product_agent_tool_calls", true, true, nil},
		{"product_agent_runtime", "ai_channel_credential_bindings", true, false, nil},
		{"title_credential_writer", "ai_channel_connections", true, true, []string{"revision", "enabled", "protection_key_id", "secret_ciphertext"}},
		{"title_credential_writer", "ai_channel_connection_receipts", true, true, nil},
		{"ai_channel_secret_reader", "ai_channel_connections", true, false, nil},
		{"title_credential_writer", "product_agent_runs", false, false, nil},
		{"title_credential_writer", "product_agent_tool_calls", false, false, nil},
		{"title_credential_writer", "ai_invocations", false, false, nil},
	}
}
func reviewGrants() []tableGrant {
	return []tableGrant{
		{"product_review_runtime", "product_title_proposals", true, true, []string{"state", "payload"}},
		{"product_review_runtime", "product_title_operations", true, true, nil},
		{"product_review_runtime", "product_snapshot_versions", true, true, nil},
		{"product_review_runtime", "product_snapshot_heads", true, true, []string{"current_version", "publication_id"}},
		{"product_review_runtime", "product_source_publications", true, false, nil},
		{"product_review_runtime", "product_source_publication_receipts", true, false, nil},
	}
}

func verifyRole(db *gorm.DB, role string) error {
	var allowed bool
	err := db.Raw(`SELECT rolcanlogin AND rolconnlimit<>0 AND (rolvaliduntil IS NULL OR rolvaliduntil>pg_catalog.clock_timestamp()) AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid OR m.roleid=r.oid) AND NOT pg_catalog.has_database_privilege(r.oid,current_database(),'CREATE') AND NOT pg_catalog.has_database_privilege(r.oid,current_database(),'TEMP') AND NOT pg_catalog.has_schema_privilege(r.oid,'public','CREATE') AND NOT pg_catalog.has_database_privilege(r.oid,current_database(),'CONNECT WITH GRANT OPTION') AND NOT pg_catalog.has_schema_privilege(r.oid,'public','USAGE WITH GRANT OPTION') FROM pg_catalog.pg_roles r WHERE rolname=?`, role).Scan(&allowed).Error
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("role %s is missing or has unapproved authority", role)
	}
	// SET LOCAL ROLE does not apply login defaults. Resolve only the current
	// database's effective write mode, preserving more specific overrides.
	var readOnly bool
	if err := db.Raw(`SELECT COALESCE((SELECT o.option_value::boolean FROM pg_catalog.pg_db_role_setting s JOIN pg_catalog.pg_roles r ON r.rolname=? JOIN pg_catalog.pg_database d ON d.datname=current_database() CROSS JOIN LATERAL pg_catalog.pg_options_to_table(s.setconfig) o WHERE s.setrole IN (0,r.oid) AND s.setdatabase IN (0,d.oid) AND o.option_name='default_transaction_read_only' ORDER BY (s.setrole=r.oid) DESC,(s.setdatabase=d.oid) DESC LIMIT 1),false)`, role).Scan(&readOnly).Error; err != nil {
		return err
	}
	if readOnly {
		return fmt.Errorf("role %s starts transactions read-only", role)
	}
	return verifyObjectBounds(db, role)
}

// Only the three private targets have a closed ACL. Existing shared ledger,
// Skill, Chat and Catalog roles are preserved, never rewritten or revoked.
func verifyPrivateACL(db *gorm.DB, table string) error {
	var unexpected bool
	err := db.Raw(`SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_class c CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(c.relacl,pg_catalog.acldefault('r',c.relowner))) a WHERE c.oid=pg_catalog.to_regclass(?) AND a.grantee<>c.relowner AND a.grantee NOT IN (SELECT oid FROM pg_catalog.pg_roles WHERE rolname IN ('product_agent_runtime','title_credential_writer','ai_channel_secret_reader'))) OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute c CROSS JOIN LATERAL pg_catalog.aclexplode(c.attacl) a WHERE c.attrelid=pg_catalog.to_regclass(?) AND a.grantee NOT IN (SELECT oid FROM pg_catalog.pg_roles WHERE rolname IN ('product_agent_owner','product_agent_runtime','title_credential_writer','ai_channel_secret_reader')))`, "public."+table, "public."+table).Scan(&unexpected).Error
	if err != nil {
		return err
	}
	if unexpected {
		return fmt.Errorf("public.%s has unapproved ACL entries", table)
	}
	return nil
}

func verifyGrants(db *gorm.DB, grants []tableGrant) (bool, error) {
	ready := true
	roles := map[string]bool{}
	for _, g := range grants {
		roles[g.Role] = true
		for _, p := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER"} {
			expected := (p == "SELECT" && g.Select) || (p == "INSERT" && g.Insert)
			var actual, option bool
			if err := db.Raw("SELECT pg_catalog.has_table_privilege(?, ?, ?),pg_catalog.has_table_privilege(?, ?, ?)", g.Role, "public."+g.Table, p, g.Role, "public."+g.Table, p+" WITH GRANT OPTION").Row().Scan(&actual, &option); err != nil {
				return false, err
			}
			if option || actual && !expected {
				return false, fmt.Errorf("role %s public.%s has excess %s authority", g.Role, g.Table, p)
			}
			if expected && !actual {
				ready = false
			}
		}
		allowed := map[string]bool{}
		for _, c := range g.Update {
			allowed[c] = true
		}
		var columns []string
		if err := db.Raw("SELECT attname FROM pg_catalog.pg_attribute WHERE attrelid=pg_catalog.to_regclass(?) AND attnum>0 AND NOT attisdropped", "public."+g.Table).Scan(&columns).Error; err != nil {
			return false, err
		}
		for _, c := range columns {
			for _, p := range []string{"SELECT", "INSERT", "UPDATE", "REFERENCES"} {
				expected := (p == "SELECT" && g.Select) || (p == "INSERT" && g.Insert) || (p == "UPDATE" && allowed[c])
				var actual, option bool
				if err := db.Raw("SELECT pg_catalog.has_column_privilege(?, ?, ?, ?),pg_catalog.has_column_privilege(?, ?, ?, ?)", g.Role, "public."+g.Table, c, p, g.Role, "public."+g.Table, c, p+" WITH GRANT OPTION").Row().Scan(&actual, &option); err != nil {
					return false, err
				}
				if option || actual && !expected {
					return false, fmt.Errorf("role %s public.%s.%s has excess %s authority", g.Role, g.Table, c, p)
				}
				if expected && !actual {
					ready = false
				}
			}
		}
	}
	for role := range roles {
		var complete bool
		if err := db.Raw("SELECT pg_catalog.has_database_privilege(?,current_database(),'CONNECT') AND pg_catalog.has_schema_privilege(?,'public','USAGE')", role, role).Scan(&complete).Error; err != nil {
			return false, err
		}
		if !complete {
			ready = false
		}
	}
	return ready, nil
}

func applyGrants(db *gorm.DB, grants []tableGrant) error {
	roles := map[string]bool{}
	for _, g := range grants {
		roles[g.Role] = true
		var permissions []string
		if g.Select {
			permissions = append(permissions, "SELECT")
		}
		if g.Insert {
			permissions = append(permissions, "INSERT")
		}
		if len(g.Update) > 0 {
			var cols []string
			for _, c := range g.Update {
				cols = append(cols, quote(c))
			}
			permissions = append(permissions, "UPDATE ("+strings.Join(cols, ",")+")")
		}
		if len(permissions) > 0 {
			if err := db.Exec("GRANT " + strings.Join(permissions, ",") + " ON TABLE public." + quote(g.Table) + " TO " + quote(g.Role)).Error; err != nil {
				return err
			}
		}
	}
	for role := range roles {
		if err := db.Exec("GRANT CONNECT ON DATABASE " + quote(db.Migrator().CurrentDatabase()) + " TO " + quote(role)).Error; err != nil {
			return err
		}
		if err := db.Exec("GRANT USAGE ON SCHEMA public TO " + quote(role)).Error; err != nil {
			return err
		}
	}
	return nil
}
