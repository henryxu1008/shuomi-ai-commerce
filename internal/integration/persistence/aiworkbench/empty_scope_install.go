package aiworkbenchpersistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"task-processor/internal/aiworkbench"
)

// EmptyScopeTarget binds an explicit operator-approved database to a TCP server.
type EmptyScopeTarget struct {
	Database string
	Address  string
	Port     int
}

const emptyScopeOldFingerprint = "b87717472e0bb44a1bf5c094eae7da5a4f75308a8ef83f00580b1b14062127f8"
const emptyScopeNewFingerprint = "d2d929fec0980711b7bfc2625b5906f656428209644c465f8703ece830723676"

const emptyScopeFacts = `WITH facts AS (
 SELECT 'namespace|' || jsonb_build_array(n.nspname,pg_get_userbyid(n.nspowner))::text AS fact FROM pg_namespace n WHERE n.nspname='ai_workbench'
 UNION ALL SELECT 'relation|' || jsonb_build_array(c.relname,c.relkind,pg_get_userbyid(c.relowner),c.relpersistence,c.relrowsecurity,c.relforcerowsecurity,coalesce(c.reloptions::text,''))::text FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='ai_workbench' AND c.relkind IN ('r','p','v','m','S','f')
 UNION ALL SELECT 'column|' || jsonb_build_array(c.relname,a.attname,format_type(a.atttypid,a.atttypmod),a.attnotnull,a.attidentity,a.attgenerated,coalesce(pg_get_expr(d.adbin,d.adrelid),''))::text FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace LEFT JOIN pg_attrdef d ON d.adrelid=c.oid AND d.adnum=a.attnum WHERE n.nspname='ai_workbench' AND c.relkind IN ('r','p') AND a.attnum>0 AND NOT a.attisdropped
 UNION ALL SELECT 'constraint|' || jsonb_build_array(c.relname,k.conname,k.contype,k.convalidated,k.condeferrable,k.condeferred,pg_get_constraintdef(k.oid))::text FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='ai_workbench'
 UNION ALL SELECT 'index|' || jsonb_build_array(tablename,indexname,indexdef)::text FROM pg_indexes WHERE schemaname='ai_workbench'
 UNION ALL SELECT 'trigger|' || jsonb_build_array(c.relname,t.tgname,pg_get_triggerdef(t.oid))::text FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='ai_workbench' AND NOT t.tgisinternal
 UNION ALL SELECT 'function|' || jsonb_build_array(p.proname,pg_get_function_identity_arguments(p.oid),pg_get_functiondef(p.oid))::text FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='ai_workbench'
) SELECT fact FROM facts ORDER BY fact COLLATE "C"`
const emptyScopeDDL = `ALTER TABLE ai_workbench.conversations ADD COLUMN business_scope jsonb CONSTRAINT ai_workbench_fixed_scope CHECK (business_scope IS NULL OR (jsonb_typeof(business_scope) = 'object' AND octet_length(business_scope::text) <= 1024 AND business_scope->>'version' IS NOT DISTINCT FROM 'workbench-scope-v1' AND ((business_scope->>'kind' = 'general' AND NOT (business_scope ?| array['projectId','storeId'])) OR (business_scope->>'kind' = 'project' AND coalesce(business_scope->>'projectId','') <> '') OR (business_scope->>'kind' = 'store' AND NOT (business_scope ? 'projectId') AND coalesce(business_scope->>'storeId','') <> '')) IS TRUE));
ALTER TABLE ai_workbench.commands ADD COLUMN frozen_scope jsonb, ADD COLUMN scope_digest char(64), ADD CONSTRAINT ai_workbench_frozen_scope CHECK ((frozen_scope IS NULL AND scope_digest IS NULL) OR (operation = 'chat_message_plan' AND frozen_scope IS NOT NULL AND jsonb_typeof(frozen_scope) = 'object' AND octet_length(frozen_scope::text) <= 16384 AND scope_digest IS NOT NULL AND scope_digest ~ '^[a-f0-9]{64}$'));
ALTER TABLE ai_workbench.execution_proposals ADD COLUMN workbench_context_ref jsonb CONSTRAINT ai_workbench_proposal_context CHECK (workbench_context_ref IS NULL OR (jsonb_typeof(workbench_context_ref) = 'object' AND octet_length(workbench_context_ref::text) <= 1024));`

func emptyScopeIdentity(db *gorm.DB, expected EmptyScopeTarget, role string) error {
	var actual struct {
		Database string
		Address  string
		Port     int
		Role     string
		Version  int
	}
	if expected.Database == "" || expected.Address == "" || expected.Port <= 0 {
		return aiworkbench.ErrInvalid
	}
	err := db.Raw(`SELECT current_database() AS database, coalesce(inet_server_addr()::text,'') AS address, coalesce(inet_server_port(),0) AS port, current_user AS role, current_setting('server_version_num')::int AS version`).Scan(&actual).Error
	if err != nil || actual.Database != expected.Database || actual.Address != expected.Address || actual.Port != expected.Port || actual.Role != role || actual.Version/10000 != 17 {
		return aiworkbench.ErrUnavailable
	}
	return nil
}

func emptyScopeFingerprint(db *gorm.DB) (string, error) {
	var rows []string
	if err := db.Raw(emptyScopeFacts).Scan(&rows).Error; err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(strings.Join(rows, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

// Verify existing privileges without granting, revoking, or changing memberships.
func emptyScopePrivileges(db *gorm.DB) error {
	for _, role := range []string{"product_agent_owner", "ai_workbench_runtime"} {
		member, err := hasRoleMembership(db, role)
		if err != nil || member {
			return aiworkbench.ErrUnavailable
		}
		var unsafe bool
		if err = db.Raw(`SELECT coalesce((SELECT rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls OR NOT rolcanlogin FROM pg_roles WHERE rolname=?),true)`, role).Scan(&unsafe).Error; err != nil || unsafe {
			return aiworkbench.ErrUnavailable
		}
	}
	if err := validateRuntimeRoleIdentity(db, "ai_workbench_runtime"); err != nil {
		return err
	}
	cross, err := hasCrossOwnerRelationPrivilege(db, "ai_workbench_runtime")
	if err != nil || cross {
		return aiworkbench.ErrUnavailable
	}
	if err = validateBoundedWorkbenchPrivileges(db, "ai_workbench_runtime"); err != nil {
		return err
	}
	var unsafe bool
	// No PUBLIC, unrelated principals or delegated grants on the schema, tables, or columns.
	query := `SELECT EXISTS (
 SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.nspname='ai_workbench' AND (a.grantee NOT IN (n.nspowner,(SELECT oid FROM pg_roles WHERE rolname='ai_workbench_runtime')) OR (a.grantee<>n.nspowner AND a.is_grantable))
 UNION ALL SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE n.nspname='ai_workbench' AND c.relkind IN ('r','p') AND (a.grantee NOT IN (c.relowner,(SELECT oid FROM pg_roles WHERE rolname='ai_workbench_runtime')) OR (a.grantee<>c.relowner AND a.is_grantable))
 UNION ALL SELECT 1 FROM pg_attribute col JOIN pg_class c ON c.oid=col.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(col.attacl) a WHERE n.nspname='ai_workbench' AND (a.grantee NOT IN (c.relowner,(SELECT oid FROM pg_roles WHERE rolname='ai_workbench_runtime')) OR (a.grantee<>c.relowner AND a.is_grantable))
 )`
	if err := db.Raw(query).Scan(&unsafe).Error; err != nil || unsafe {
		return aiworkbench.ErrUnavailable
	}
	// Cross-schema references and standalone types make this an unrecognised installation.
	query = `SELECT EXISTS (
 SELECT 1 FROM pg_constraint k JOIN pg_class r ON r.oid=k.conrelid JOIN pg_namespace n ON n.oid=r.relnamespace JOIN pg_class target ON target.oid=k.confrelid JOIN pg_namespace tn ON tn.oid=target.relnamespace WHERE tn.nspname='ai_workbench' AND n.nspname<>'ai_workbench'
 UNION ALL SELECT 1 FROM pg_depend d JOIN pg_class target ON target.oid=d.refobjid JOIN pg_namespace tn ON tn.oid=target.relnamespace JOIN pg_rewrite rw ON rw.oid=d.objid JOIN pg_class v ON v.oid=rw.ev_class JOIN pg_namespace vn ON vn.oid=v.relnamespace WHERE d.refclassid='pg_class'::regclass AND d.classid='pg_rewrite'::regclass AND tn.nspname='ai_workbench' AND vn.nspname<>'ai_workbench'
 UNION ALL SELECT 1 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname='ai_workbench' AND t.typrelid=0 AND t.typelem=0
 UNION ALL SELECT 1 FROM pg_rewrite rw JOIN pg_class r ON r.oid=rw.ev_class JOIN pg_namespace n ON n.oid=r.relnamespace WHERE n.nspname='ai_workbench'
 UNION ALL SELECT 1 FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace JOIN pg_type t ON t.oid=a.atttypid WHERE n.nspname='ai_workbench' AND a.attnum>0 AND NOT a.attisdropped AND a.attcollation<>t.typcollation
 UNION ALL SELECT 1 FROM pg_trigger tr JOIN pg_class c ON c.oid=tr.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='ai_workbench' AND tr.tgisinternal AND tr.tgenabled <> 'O'
 UNION ALL SELECT 1 FROM pg_depend d JOIN pg_type target ON target.oid=d.refobjid JOIN pg_class tab ON tab.reltype=target.oid OR tab.reltype=target.typelem JOIN pg_namespace tn ON tn.oid=tab.relnamespace JOIN pg_proc p ON d.classid='pg_proc'::regclass AND p.oid=d.objid JOIN pg_namespace pn ON pn.oid=p.pronamespace WHERE d.refclassid='pg_type'::regclass AND tn.nspname='ai_workbench' AND tab.relname IN ('conversations','commands','execution_proposals') AND pn.nspname<>'ai_workbench'
 UNION ALL SELECT 1 FROM pg_depend d JOIN pg_type target ON target.oid=d.refobjid JOIN pg_class tab ON tab.reltype=target.oid OR tab.reltype=target.typelem JOIN pg_namespace tn ON tn.oid=tab.relnamespace JOIN pg_class dep ON d.classid='pg_class'::regclass AND dep.oid=d.objid JOIN pg_namespace dn ON dn.oid=dep.relnamespace WHERE d.refclassid='pg_type'::regclass AND d.objsubid>0 AND tn.nspname='ai_workbench' AND tab.relname IN ('conversations','commands','execution_proposals') AND dn.nspname<>'ai_workbench'
 )`
	if err := db.Raw(query).Scan(&unsafe).Error; err != nil || unsafe {
		return aiworkbench.ErrUnavailable
	}
	return nil
}

// InstallEmptyScope only installs the exact, empty, owner-held baseline. A
// separate runtime connection verifies the committed schema before success.
func InstallEmptyScope(ctx context.Context, owner, runtime *gorm.DB, target EmptyScopeTarget) error {
	if ctx == nil || owner == nil || runtime == nil || owner.Dialector.Name() != "postgres" || runtime.Dialector.Name() != "postgres" {
		return aiworkbench.ErrInvalid
	}
	// Keep each identity check, transaction and runtime verification on its
	// own dedicated physical connection, not an arbitrary pool member.
	return owner.WithContext(ctx).Connection(func(ownerConn *gorm.DB) error {
		return runtime.WithContext(ctx).Connection(func(runtimeConn *gorm.DB) error {
			return installEmptyScopeOnConnections(ctx, ownerConn, runtimeConn, target)
		})
	})
}

func installEmptyScopeOnConnections(ctx context.Context, owner, runtime *gorm.DB, target EmptyScopeTarget) error {
	if err := emptyScopeIdentity(owner, target, "product_agent_owner"); err != nil {
		return err
	}
	if err := emptyScopeIdentity(runtime, target, "ai_workbench_runtime"); err != nil {
		return err
	}
	err := owner.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SET LOCAL lock_timeout = '2s'; SET LOCAL statement_timeout = '10s'`).Error; err != nil {
			return err
		}
		if err := emptyScopeIdentity(tx, target, "product_agent_owner"); err != nil {
			return err
		}
		if err := tx.Exec(`LOCK TABLE ai_workbench.conversations, ai_workbench.metadata_audit, ai_workbench.messages, ai_workbench.commands, ai_workbench.execution_proposals, ai_workbench.business_tasks, ai_workbench.task_action_receipts IN ACCESS EXCLUSIVE MODE`).Error; err != nil {
			return err
		}
		if err := emptyScopePrivileges(tx); err != nil {
			return err
		}
		fingerprint, err := emptyScopeFingerprint(tx)
		if err != nil {
			return err
		}
		if fingerprint != emptyScopeOldFingerprint && fingerprint != emptyScopeNewFingerprint {
			return aiworkbench.ErrInvalid
		}
		var nonempty bool
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM ai_workbench.conversations LIMIT 1) OR EXISTS(SELECT 1 FROM ai_workbench.metadata_audit LIMIT 1) OR EXISTS(SELECT 1 FROM ai_workbench.messages LIMIT 1) OR EXISTS(SELECT 1 FROM ai_workbench.commands LIMIT 1) OR EXISTS(SELECT 1 FROM ai_workbench.execution_proposals LIMIT 1) OR EXISTS(SELECT 1 FROM ai_workbench.business_tasks LIMIT 1) OR EXISTS(SELECT 1 FROM ai_workbench.task_action_receipts LIMIT 1)`).Scan(&nonempty).Error; err != nil || nonempty {
			return aiworkbench.ErrInvalid
		}
		if fingerprint == emptyScopeNewFingerprint {
			return nil
		}
		if err := tx.Exec(emptyScopeDDL).Error; err != nil {
			return err
		}
		fingerprint, err = emptyScopeFingerprint(tx)
		if err != nil || fingerprint != emptyScopeNewFingerprint {
			return aiworkbench.ErrUnavailable
		}
		return emptyScopePrivileges(tx)
	})
	if err != nil {
		return err
	}
	if err := emptyScopeIdentity(runtime, target, "ai_workbench_runtime"); err != nil {
		return fmt.Errorf("committed or unchanged; runtime identity unavailable: %w", err)
	}
	if err := VerifySchema(ctx, runtime); err != nil {
		return fmt.Errorf("committed or unchanged; runtime verification unavailable: %w", err)
	}
	return nil
}
