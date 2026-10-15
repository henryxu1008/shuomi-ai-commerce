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
		{"product_agent_runtime", "ai_client_credentials", true, false, nil},
		{"title_credential_writer", "ai_client_credentials", true, true, []string{"api_key", "base_url", "model", "api_style", "timeout_second", "enabled", "updated_at"}},
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
	err := db.Raw(`SELECT rolcanlogin AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid OR m.roleid=r.oid) AND NOT pg_catalog.has_database_privilege(r.oid,current_database(),'CREATE') AND NOT pg_catalog.has_database_privilege(r.oid,current_database(),'TEMP') AND NOT pg_catalog.has_schema_privilege(r.oid,'public','CREATE') AND NOT pg_catalog.has_database_privilege(r.oid,current_database(),'CONNECT WITH GRANT OPTION') AND NOT pg_catalog.has_schema_privilege(r.oid,'public','USAGE WITH GRANT OPTION') FROM pg_catalog.pg_roles r WHERE rolname=?`, role).Scan(&allowed).Error
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
	err := db.Raw(`SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_class c CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(c.relacl,pg_catalog.acldefault('r',c.relowner))) a WHERE c.oid=pg_catalog.to_regclass(?) AND a.grantee<>c.relowner AND a.grantee NOT IN (SELECT oid FROM pg_catalog.pg_roles WHERE rolname IN ('product_agent_runtime','title_credential_writer'))) OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute c CROSS JOIN LATERAL pg_catalog.aclexplode(c.attacl) a WHERE c.attrelid=pg_catalog.to_regclass(?) AND a.grantee NOT IN (SELECT oid FROM pg_catalog.pg_roles WHERE rolname IN ('product_agent_owner','product_agent_runtime','title_credential_writer')))`, "public."+table, "public."+table).Scan(&unexpected).Error
	if err != nil {
		return err
	}
	if unexpected {
		return fmt.Errorf("public.%s has unapproved ACL entries", table)
	}
	return nil
}

func verifyCredentialSequence(db *gorm.DB) error {
	var ok bool
	err := db.Raw(`SELECT c.relkind='S' AND c.relpersistence='p' AND pg_catalog.pg_get_userbyid(c.relowner)='product_agent_owner' AND pg_catalog.pg_get_serial_sequence('public.ai_client_credentials','id')='public.ai_client_credentials_id_seq' AND EXISTS(SELECT 1 FROM pg_catalog.pg_depend d JOIN pg_catalog.pg_attribute a ON a.attrelid=d.refobjid AND a.attnum=d.refobjsubid WHERE d.classid='pg_class'::regclass AND d.objid=c.oid AND d.refobjid='public.ai_client_credentials'::regclass AND a.attname='id' AND d.deptype='a') AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(c.relacl,pg_catalog.acldefault('S',c.relowner))) a WHERE a.grantee<>c.relowner AND a.grantee<>(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='title_credential_writer')) FROM pg_catalog.pg_class c WHERE c.oid=pg_catalog.to_regclass('public.ai_client_credentials_id_seq')`).Scan(&ok).Error
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("credential id sequence owner/dependency/ACL differs")
	}
	return verifyCredentialSequenceState(db)
}

// The owner model uses a default bigint sequence. Read its prospective value
// without reserving/advancing it; gaps from conflict/rollback are valid.
func verifyCredentialSequenceState(db *gorm.DB) error {
	var originalParameters bool
	if err := db.Raw(`SELECT seqtypid='pg_catalog.int8'::regtype AND seqstart=1 AND seqincrement=1 AND seqmin=1 AND seqmax=9223372036854775807 AND seqcache=1 AND NOT seqcycle FROM pg_catalog.pg_sequence WHERE seqrelid='public.ai_client_credentials_id_seq'::regclass`).Scan(&originalParameters).Error; err != nil {
		return err
	}
	if !originalParameters {
		return fmt.Errorf("credential id sequence parameters differ from the original model")
	}
	var state struct {
		LastValue int64
		IsCalled  bool
	}
	if err := db.Raw("SELECT last_value,is_called FROM public.ai_client_credentials_id_seq").Scan(&state).Error; err != nil {
		return err
	}
	next := state.LastValue
	if state.IsCalled {
		if next == 1<<63-1 {
			return fmt.Errorf("credential id sequence is exhausted")
		}
		next++
	}
	var maximumID int64
	if err := db.Raw("SELECT COALESCE(pg_catalog.max(id),0) FROM public.ai_client_credentials").Scan(&maximumID).Error; err != nil {
		return err
	}
	if next < 1 || next <= maximumID {
		return fmt.Errorf("credential id sequence does not follow existing IDs")
	}
	return nil
}

func verifyGrants(db *gorm.DB, grants []tableGrant, sequence bool) (bool, error) {
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
	if sequence {
		for _, role := range []string{"product_agent_runtime", "title_credential_writer"} {
			for _, p := range []string{"USAGE", "SELECT", "UPDATE"} {
				var actual, option bool
				if err := db.Raw("SELECT pg_catalog.has_sequence_privilege(?, 'public.ai_client_credentials_id_seq', ?),pg_catalog.has_sequence_privilege(?, 'public.ai_client_credentials_id_seq', ?)", role, p, role, p+" WITH GRANT OPTION").Row().Scan(&actual, &option); err != nil {
					return false, err
				}
				expected := role == "title_credential_writer" && p == "USAGE"
				if option || actual && !expected {
					return false, fmt.Errorf("credential sequence privileges differ for %s", role)
				}
				if expected && !actual {
					ready = false
				}
			}
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
