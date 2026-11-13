package aiworkbenchpersistence

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// This entry is deliberately limited to the reviewed 35444 baseline and its
// isolated copies. It must never be called by a serving constructor.
type RetainedTarget struct {
	Database          string    `json:"database"`
	Address           string    `json:"address"`
	SessionUser       string    `json:"session_user"`
	SystemIdentifier  string    `json:"system_identifier"`
	DatabaseOID       string    `json:"database_oid" gorm:"column:database_oid"`
	Port              int       `json:"port"`
	MaintenanceWindow string    `json:"maintenance_window"`
	BackupEvidence    string    `json:"backup_evidence"`
	FactsDigest       string    `json:"facts_digest"`
	PrivilegesDigest  string    `json:"privileges_digest"`
	NotAfter          time.Time `json:"not_after"`
}

type RetainedState string

const (
	RetainedOld   RetainedState = "OLD"
	RetainedReady RetainedState = "READY"
)

const retainedOldFingerprint = "8ae6ff3130daa07483cc6e16ae8dad233adc82a0ab5f73086453e7e442c002c8"
const retainedNewFingerprint = "55110ab58b16be50277d212dd85c3360757ba9432ef55cab7637191ea259fbe8"

//go:embed retained_wiring.sql
var retainedWiringDDL string

func retainedIdentity(db *gorm.DB, want RetainedTarget) error {
	if want.Database == "" || want.SessionUser == "" || want.SystemIdentifier == "" || want.DatabaseOID == "" || (want.Address == "" && want.Port != 0) || (want.Address != "" && want.Port <= 0) {
		return errors.New("incomplete target identity")
	}
	var got RetainedTarget
	err := db.Raw(`SELECT current_database() AS database, coalesce(inet_server_addr()::text,'') AS address, coalesce(inet_server_port(),0) AS port, session_user AS session_user, (SELECT system_identifier::text FROM pg_control_system()) AS system_identifier, (SELECT oid::text FROM pg_database WHERE datname=current_database()) AS database_oid`).Scan(&got).Error
	if err != nil || got.Database != want.Database || got.Address != want.Address || got.Port != want.Port || got.SessionUser != want.SessionUser || got.SystemIdentifier != want.SystemIdentifier || got.DatabaseOID != want.DatabaseOID {
		return errors.New("target identity mismatch or unavailable")
	}
	return nil
}

func retainedPrivileges(db *gorm.DB, state RetainedState) error {
	if err := validateRetainedDatabasePrivileges(db); err != nil {
		return err
	}
	for _, role := range []string{"product_agent_owner", "ai_workbench_runtime"} {
		var unsafe bool
		err := db.Raw(`SELECT coalesce((SELECT rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls OR NOT rolcanlogin FROM pg_roles WHERE rolname=?),true)`, role).Scan(&unsafe).Error
		member, memberErr := hasRoleMembership(db, role)
		if err != nil || memberErr != nil || unsafe || member {
			return errors.New("owner or runtime role drift")
		}
	}
	if err := validateRuntimeRoleIdentity(db, "ai_workbench_runtime"); err != nil {
		return err
	}
	if unsafe, err := hasCrossOwnerRelationPrivilege(db, "ai_workbench_runtime"); err != nil || unsafe {
		return errors.New("cross-owner privilege drift")
	}
	mutable := map[string][]string{
		"conversations":         {"title", "favorite", "lifecycle", "metadata_revision", "next_sequence", "updated_at"},
		"commands":              {"state", "assistant_message_id", "proposal_id", "terminal_digest", "mode", "committed_at"},
		"task_action_receipts":  {"state", "error_code", "finished_at"},
		"scheduled_plans":       {"state", "revision", "consumed_occurrences", "active_occurrence_id", "updated_at", "registration_state", "rule_start_at", "frequency", "time_zone", "once_at_utc", "local_hour", "local_minute", "local_weekday", "until_utc"},
		"scheduled_occurrences": {"state", "business_task_id", "updated_at", "reason", "reconciliation_digest"},
	}
	if state == RetainedReady {
		mutable["commands"] = append(mutable["commands"], "route_result")
	}
	if err := validateWorkbenchPrivilegeBounds(db, "ai_workbench_runtime", []string{"conversations", "metadata_audit", "messages", "commands", "execution_proposals", "business_tasks", "task_action_receipts", "scheduled_plans", "scheduled_occurrences", "scheduled_plan_commands"}, mutable); err != nil {
		return err
	}
	var extra bool
	err := db.Raw(`SELECT EXISTS (
 SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.nspname='ai_workbench' AND (a.grantee NOT IN (n.nspowner,(SELECT oid FROM pg_roles WHERE rolname='ai_workbench_runtime')) OR (a.grantee<>n.nspowner AND a.is_grantable))
 UNION ALL SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE n.nspname='ai_workbench' AND c.relkind IN ('r','p') AND (a.grantee NOT IN (c.relowner,(SELECT oid FROM pg_roles WHERE rolname='ai_workbench_runtime')) OR (a.grantee<>c.relowner AND a.is_grantable))
 UNION ALL SELECT 1 FROM pg_attribute col JOIN pg_class c ON c.oid=col.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(col.attacl) a WHERE n.nspname='ai_workbench' AND (a.grantee NOT IN (c.relowner,(SELECT oid FROM pg_roles WHERE rolname='ai_workbench_runtime')) OR (a.grantee<>c.relowner AND a.is_grantable))
 UNION ALL SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='ai_workbench' AND c.relkind IN ('r','p') AND has_any_column_privilege('ai_workbench_runtime',c.oid,'REFERENCES')
 UNION ALL SELECT 1 FROM pg_trigger tr JOIN pg_class c ON c.oid=tr.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='ai_workbench' AND tr.tgisinternal AND tr.tgenabled <> 'O'
 )`).Scan(&extra).Error
	if err != nil || extra {
		return errors.New("unexpected grants or disabled internal trigger")
	}
	return nil
}

// The retained database has a bounded, independently observed set of owners
// and consumers. Database ACLs are separate from schema/table privileges.
func validateRetainedDatabasePrivileges(db *gorm.DB) error {
	var ok bool
	err := db.Raw(`SELECT pg_get_userbyid(d.datdba) = 'product_agent_owner'
 AND has_database_privilege('ai_workbench_runtime', d.oid, 'CONNECT')
 AND NOT has_database_privilege('ai_workbench_runtime', d.oid, 'CREATE')
 AND NOT has_database_privilege('ai_workbench_runtime', d.oid, 'TEMP')
 AND NOT has_database_privilege('ai_workbench_runtime', d.oid, 'CONNECT WITH GRANT OPTION')
 AND NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
   WHERE a.grantee <> d.datdba AND (a.grantee = 0 OR pg_get_userbyid(a.grantee) NOT IN ('product_agent_runtime','title_credential_writer','ai_channel_secret_reader','account_audit_product_reader','ai_workbench_runtime','model_route_config_writer') OR a.privilege_type <> 'CONNECT' OR a.is_grantable))
 AND NOT coalesce((SELECT o.option_value::boolean FROM pg_db_role_setting s
 JOIN pg_roles r ON r.rolname='ai_workbench_runtime'
 CROSS JOIN LATERAL pg_options_to_table(s.setconfig) o
 WHERE s.setrole IN (0,r.oid) AND s.setdatabase IN (0,d.oid) AND o.option_name='default_transaction_read_only'
 ORDER BY (s.setrole=r.oid) DESC,(s.setdatabase=d.oid) DESC LIMIT 1),false)
 FROM pg_database d WHERE d.datname=current_database()`).Scan(&ok).Error
	if err != nil || !ok {
		return errors.New("retained database privileges or runtime settings drift")
	}
	return nil
}

func retainedState(db *gorm.DB, target RetainedTarget) (RetainedState, error) {
	if err := retainedIdentity(db, target); err != nil {
		return "", err
	}
	fingerprint, err := emptyScopeFingerprint(db)
	if err != nil {
		return "", err
	}
	var state RetainedState
	switch fingerprint {
	case retainedOldFingerprint:
		state = RetainedOld
	case retainedNewFingerprint:
		state = RetainedReady
	default:
		return "", errors.New("unrecognized or mixed workbench schema")
	}
	if err := retainedPrivileges(db, state); err != nil {
		return "", fmt.Errorf("privilege verification: %w", err)
	}
	facts, privileges, err := retainedDigests(db)
	if err != nil || facts != target.FactsDigest || privileges != target.PrivilegesDigest {
		return "", errors.New("preserved facts or privilege snapshot drift")
	}
	if state == RetainedReady {
		var changed bool
		if err := db.Raw(`SELECT EXISTS(SELECT 1 FROM ai_workbench.messages WHERE message_kind <> 'TEXT' OR store_query_status IS NOT NULL) OR EXISTS(SELECT 1 FROM ai_workbench.commands WHERE route_commit_deadline IS NOT NULL OR route_result IS NOT NULL)`).Scan(&changed).Error; err != nil || changed {
			return "", errors.New("new fields on preserved rows changed after preparation")
		}
	}
	return state, nil
}

// Digests never expose row contents. They bind a maintenance window to its
// approved snapshot and deliberately ignore only the four new columns and the
// one approved new grant. A changed fact must be investigated, not overwritten.
func retainedDigests(db *gorm.DB) (string, string, error) {
	facts := sha256.New()
	for _, table := range []string{"conversations", "metadata_audit", "messages", "commands", "execution_proposals", "business_tasks", "task_action_receipts", "scheduled_plans", "scheduled_occurrences", "scheduled_plan_commands"} {
		fmt.Fprintf(facts, "%d:%s\n", len(table), table)
		q := `SELECT (to_jsonb(r)-'message_kind'-'store_query_status'-'route_commit_deadline'-'route_result')::text FROM ai_workbench.` + table + ` r ORDER BY (to_jsonb(r)-'message_kind'-'store_query_status'-'route_commit_deadline'-'route_result')::text COLLATE "C"`
		rows, err := db.Raw(q).Rows()
		if err != nil {
			return "", "", err
		}
		for rows.Next() {
			var row string
			if err := rows.Scan(&row); err != nil {
				rows.Close()
				return "", "", err
			}
			fmt.Fprintf(facts, "%d:%s\n", len(row), row)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return "", "", err
		}
		rows.Close()
	}
	var acl []string
	err := db.Raw(`WITH p AS (
 SELECT 'schema|' || jsonb_build_array(n.nspname,pg_get_userbyid(a.grantor),pg_get_userbyid(a.grantee),a.privilege_type,a.is_grantable)::text fact FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.nspname='ai_workbench'
 UNION ALL SELECT 'table|' || jsonb_build_array(c.relname,pg_get_userbyid(a.grantor),pg_get_userbyid(a.grantee),a.privilege_type,a.is_grantable)::text FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE n.nspname='ai_workbench' AND c.relkind IN ('r','p')
 UNION ALL SELECT 'column|' || jsonb_build_array(c.relname,col.attname,pg_get_userbyid(a.grantor),pg_get_userbyid(a.grantee),a.privilege_type,a.is_grantable)::text FROM pg_attribute col JOIN pg_class c ON c.oid=col.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(col.attacl) a WHERE n.nspname='ai_workbench' AND NOT(c.relname='commands' AND col.attname='route_result' AND pg_get_userbyid(a.grantee)='ai_workbench_runtime' AND a.privilege_type='UPDATE' AND NOT a.is_grantable)
 UNION ALL SELECT 'role|' || jsonb_build_array(rolname,rolsuper,rolinherit,rolcreaterole,rolcreatedb,rolcanlogin,rolreplication,rolbypassrls,rolconnlimit,rolvaliduntil)::text FROM pg_roles WHERE rolname IN ('ai_workbench_runtime','product_agent_owner')
 UNION ALL SELECT 'database|' || jsonb_build_array(pg_get_userbyid(d.datdba),pg_get_userbyid(a.grantor),pg_get_userbyid(a.grantee),a.privilege_type,a.is_grantable)::text FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a WHERE d.datname=current_database()
 UNION ALL SELECT 'db-setting|' || jsonb_build_array(s.setrole::text,s.setdatabase::text,s.setconfig)::text FROM pg_db_role_setting s WHERE s.setrole IN (0,(SELECT oid FROM pg_roles WHERE rolname='ai_workbench_runtime'),(SELECT oid FROM pg_roles WHERE rolname='product_agent_owner')) AND s.setdatabase IN (0,(SELECT oid FROM pg_database WHERE datname=current_database()))
 UNION ALL SELECT 'member|' || jsonb_build_array(pg_get_userbyid(roleid),pg_get_userbyid(member),admin_option)::text FROM pg_auth_members WHERE roleid IN (SELECT oid FROM pg_roles WHERE rolname IN ('ai_workbench_runtime','product_agent_owner')) OR member IN (SELECT oid FROM pg_roles WHERE rolname IN ('ai_workbench_runtime','product_agent_owner'))
) SELECT fact FROM p ORDER BY fact COLLATE "C"`).Scan(&acl).Error
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(strings.Join(acl, "\n")))
	return hex.EncodeToString(facts.Sum(nil)), hex.EncodeToString(sum[:]), nil
}

// CaptureRetainedBaseline is read-only and only accepts the exact old schema
// and privilege bounds. The operator must capture it within the owner freeze;
// this function cannot establish or maintain a cross-owner freeze.
func CaptureRetainedBaseline(ctx context.Context, db *gorm.DB, target RetainedTarget) (string, string, error) {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return "", "", errors.New("invalid connection")
	}
	var facts, acl string
	err := db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
		var err error
		facts, acl, err = captureRetainedOnConnection(conn, target)
		return err
	})
	return facts, acl, err
}

func captureRetainedOnConnection(conn *gorm.DB, target RetainedTarget) (string, string, error) {
	var facts, acl string
	err := conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET TRANSACTION READ ONLY, ISOLATION LEVEL REPEATABLE READ").Error; err != nil {
			return err
		}
		if err := retainedIdentity(tx, target); err != nil {
			return err
		}
		fingerprint, err := emptyScopeFingerprint(tx)
		if err != nil || fingerprint != retainedOldFingerprint {
			return errors.New("not the approved old baseline")
		}
		if err := retainedPrivileges(tx, RetainedOld); err != nil {
			return err
		}
		facts, acl, err = retainedDigests(tx)
		return err
	})
	return facts, acl, err
}

// InspectRetainedWorkbench does not write. UNKNOWN is returned as an error;
// callers must not assume an error means a preceding commit rolled back.
func InspectRetainedWorkbench(ctx context.Context, db *gorm.DB, target RetainedTarget) (RetainedState, error) {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return "", errors.New("invalid connection")
	}
	var state RetainedState
	err := db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
		return conn.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SET TRANSACTION READ ONLY, ISOLATION LEVEL REPEATABLE READ").Error; err != nil {
				return err
			}
			var err error
			state, err = retainedState(tx, target)
			return err
		})
	})
	return state, err
}

func prepareRetainedWorkbench(ctx context.Context, db *gorm.DB, target RetainedTarget) (RetainedState, error) {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" || strings.TrimSpace(target.MaintenanceWindow) == "" || strings.TrimSpace(target.BackupEvidence) == "" || target.NotAfter.IsZero() || !time.Now().Before(target.NotAfter) {
		return "", errors.New("target, maintenance window and backup evidence required")
	}
	windowCtx, cancel := context.WithDeadline(ctx, target.NotAfter)
	defer cancel()
	var result RetainedState
	err := db.WithContext(windowCtx).Connection(func(conn *gorm.DB) error {
		var err error
		result, err = prepareRetainedOnConnection(conn, target)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("preparation not confirmed; inspect the same target read-only before any retry: %w", err)
	}
	return result, nil
}

func prepareRetainedOnConnection(conn *gorm.DB, target RetainedTarget) (RetainedState, error) {
	var result RetainedState
	err := conn.Transaction(func(tx *gorm.DB) error {
		if target.NotAfter.IsZero() || !time.Now().Before(target.NotAfter) {
			return errors.New("maintenance window expired")
		}
		if err := tx.Exec("SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s'").Error; err != nil {
			return err
		}
		if err := retainedIdentity(tx, target); err != nil {
			return err
		}
		if err := tx.Exec(`LOCK TABLE ai_workbench.conversations,ai_workbench.metadata_audit,ai_workbench.messages,ai_workbench.commands,ai_workbench.execution_proposals,ai_workbench.business_tasks,ai_workbench.task_action_receipts,ai_workbench.scheduled_plans,ai_workbench.scheduled_occurrences,ai_workbench.scheduled_plan_commands IN ACCESS EXCLUSIVE MODE`).Error; err != nil {
			return err
		}
		if !time.Now().Before(target.NotAfter) {
			return errors.New("maintenance window expired while waiting for lock")
		}
		state, err := retainedState(tx, target)
		if err != nil {
			return err
		}
		var pending bool
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM ai_workbench.commands WHERE state NOT IN ('COMPLETE','FAILED_BEFORE_DISPATCH','PLANNER_INVALID_OUTPUT')) OR EXISTS(SELECT 1 FROM ai_workbench.task_action_receipts WHERE state NOT IN ('COMPLETE','FAILED')) OR EXISTS(SELECT 1 FROM ai_workbench.scheduled_plans WHERE state <> 'CLOSED') OR EXISTS(SELECT 1 FROM ai_workbench.scheduled_occurrences WHERE state NOT IN ('RECONCILED','SKIPPED','BLOCKED'))`).Scan(&pending).Error; err != nil || pending {
			return errors.New("in-flight or unknown work must be resolved by its owner")
		}
		if state == RetainedReady {
			if !time.Now().Before(target.NotAfter) {
				return errors.New("maintenance window expired before commit")
			}
			result = state
			return nil
		}
		if err := tx.Exec("SET LOCAL ROLE product_agent_owner").Error; err != nil {
			return err
		}
		if err := tx.Exec(retainedWiringDDL).Error; err != nil {
			return err
		}
		if err := tx.Exec("RESET ROLE").Error; err != nil {
			return err
		}
		if !time.Now().Before(target.NotAfter) {
			return errors.New("maintenance window expired before commit")
		}
		state, err = retainedState(tx, target)
		if err != nil || state != RetainedReady {
			return errors.New("post-change verification failed")
		}
		if !time.Now().Before(target.NotAfter) {
			return errors.New("maintenance window expired before commit")
		}
		result = state
		return nil
	})
	return result, err
}
