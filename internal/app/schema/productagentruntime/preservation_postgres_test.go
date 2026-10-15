package productagentruntime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/aicapability"
	aistore "task-processor/internal/aicapability/store"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	workstore "task-processor/internal/integration/persistence/aiworkbench"
	skillstore "task-processor/internal/integration/persistence/skill"
)

// Snapshot original facts and physical definitions/ACL, rather than accepting a
// successful installer result as evidence that unrelated retained owners survived.
func retainedSnapshot(t *testing.T, db *gorm.DB, tables []string) []string {
	t.Helper()
	var result []string
	for _, table := range tables {
		var facts, metadata string
		require.NoError(t, db.Raw("SELECT COALESCE(jsonb_agg(to_jsonb(r) ORDER BY to_jsonb(r)::text),'[]'::jsonb)::text FROM "+table+" r").Scan(&facts).Error)
		require.NoError(t, db.Raw(`SELECT jsonb_build_object('owner',c.relowner,'acl',c.relacl,'columns',(SELECT jsonb_agg(to_jsonb(a) ORDER BY a.attnum) FROM pg_catalog.pg_attribute a WHERE a.attrelid=c.oid AND a.attnum>0),'constraints',(SELECT jsonb_agg(pg_catalog.pg_get_constraintdef(x.oid) ORDER BY x.conname) FROM pg_catalog.pg_constraint x WHERE x.conrelid=c.oid),'indexes',(SELECT jsonb_agg(pg_catalog.pg_get_indexdef(x.indexrelid) ORDER BY x.indexrelid) FROM pg_catalog.pg_index x WHERE x.indrelid=c.oid))::text FROM pg_catalog.pg_class c WHERE c.oid=pg_catalog.to_regclass(?)`, table).Scan(&metadata).Error)
		result = append(result, facts, metadata)
	}
	return result
}

func TestPrepareRunPreservesChatSkillAndSharedLedger(t *testing.T) {
	f := newPrepareFixture(t, "run")
	require.NoError(t, configstore.InstallSchema(f.db))
	require.NoError(t, skillstore.InstallSchema(context.Background(), f.db))
	require.NoError(t, skillstore.GrantRuntime(context.Background(), f.db, "product_agent_runtime"))
	require.NoError(t, workstore.InstallSchema(f.db))
	require.NoError(t, f.db.Exec(`DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='ai_workbench_runtime') THEN CREATE ROLE ai_workbench_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; END IF; END $$`).Error)
	require.NoError(t, workstore.GrantRuntime(f.db, "ai_workbench_runtime"))
	require.NoError(t, f.db.Exec(`INSERT INTO ai_workbench.conversations VALUES('00000000-0000-0000-0000-000000000688','org','actor','retained conversation',false,'ACTIVE',1,1,now(),now()); INSERT INTO skill_capability.skills VALUES('00000000-0000-0000-0000-000000000688','USER','actor','actor',1,1,now(),now()); INSERT INTO skill_capability.versions VALUES('00000000-0000-0000-0000-000000000688',1,'{}',repeat('a',64),'DRAFT','PRIVATE',false,false,NULL,now()); GRANT SELECT ON public.ai_invocations TO ai_workbench_runtime`).Error)
	ledger := aistore.NewGormInvocationRecorder(f.db)
	_, err := ledger.ClaimInvocation(context.Background(), aicapability.InvocationRecord{InvocationID: "retained", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "old-run", InputHash: "input", Operation: aicapability.OperationProductContentDecision, StartedAt: time.Now().UTC(), Outcome: aicapability.InvocationDispatched})
	require.NoError(t, err)
	tables := []string{"ai_workbench.conversations", "ai_workbench.commands", "ai_workbench.business_tasks", "skill_capability.skills", "skill_capability.versions", "skill_capability.commands", "agent_configuration.skill_admissions"}
	before := retainedSnapshot(t, f.db, tables)
	var ledgerBefore, aclBefore string
	require.NoError(t, f.db.Raw("SELECT to_jsonb(r)::text FROM public.ai_invocations r WHERE invocation_id='retained'").Scan(&ledgerBefore).Error)
	sharedACL := `SELECT string_agg(a.privilege_type||':'||a.is_grantable::text,',' ORDER BY a.privilege_type) FROM pg_catalog.pg_class c CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) a WHERE c.oid='public.ai_invocations'::regclass AND a.grantee=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ai_workbench_runtime')`
	require.NoError(t, f.db.Raw(sharedACL).Scan(&aclBefore).Error)
	for i := 0; i < 2; i++ {
		result, err := PrepareRun(context.Background(), f.db)
		require.NoError(t, err)
		if i == 0 {
			require.Equal(t, "CREATED", result.Outcome)
		} else {
			require.Equal(t, "ALREADY_READY", result.Outcome)
		}
		require.Equal(t, before, retainedSnapshot(t, f.db, tables))
		var facts, acl string
		require.NoError(t, f.db.Raw("SELECT to_jsonb(r)::text FROM public.ai_invocations r WHERE invocation_id='retained'").Scan(&facts).Error)
		require.NoError(t, f.db.Raw(sharedACL).Scan(&acl).Error)
		require.Equal(t, ledgerBefore, facts)
		require.Equal(t, aclBefore, acl)
	}
	// A complete repeat must not rewrite even target table/column ACL.
	before = retainedSnapshot(t, f.db, []string{"public.product_agent_runs", "public.product_agent_tool_calls", "public.ai_client_credentials", "public.ai_invocations"})
	_, err = PrepareRun(context.Background(), f.db)
	require.NoError(t, err)
	require.Equal(t, before, retainedSnapshot(t, f.db, []string{"public.product_agent_runs", "public.product_agent_tool_calls", "public.ai_client_credentials", "public.ai_invocations"}))
}

func TestPrepareRunRefusesWrongOwnerWeakenedConstraintAndPermissions(t *testing.T) {
	for _, test := range []string{"owner", "check", "table-update", "column-update", "grant-option", "sequence", "missing-grant", "shared-drift", "default-acl", "index-include", "connection-option", "schema-option"} {
		t.Run(test, func(t *testing.T) {
			f := newPrepareFixture(t, "run")
			if test == "shared-drift" {
				require.NoError(t, f.db.Exec("ALTER TABLE public.ai_invocations ALTER COLUMN invocation_id TYPE varchar(255)").Error)
			} else if test == "default-acl" {
				require.NoError(t, f.db.Exec("ALTER DEFAULT PRIVILEGES FOR ROLE product_agent_owner IN SCHEMA public GRANT SELECT ON TABLES TO PUBLIC").Error)
			} else {
				_, err := PrepareRun(context.Background(), f.db)
				require.NoError(t, err)
				sql := map[string]string{
					"owner":             "ALTER TABLE public.product_agent_runs OWNER TO acquisition_owner",
					"check":             "ALTER TABLE public.product_agent_runs DROP CONSTRAINT product_agent_runs_payload_check; ALTER TABLE public.product_agent_runs ADD CONSTRAINT product_agent_runs_payload_check CHECK(octet_length(payload)<=4194304)",
					"table-update":      "GRANT UPDATE ON public.product_agent_runs TO product_agent_runtime",
					"column-update":     "GRANT UPDATE(actor) ON public.product_agent_runs TO product_agent_runtime",
					"grant-option":      "GRANT SELECT ON public.product_agent_runs TO product_agent_runtime WITH GRANT OPTION",
					"sequence":          "GRANT SELECT ON SEQUENCE public.ai_client_credentials_id_seq TO title_credential_writer",
					"missing-grant":     "REVOKE UPDATE(phase) ON public.product_agent_runs FROM product_agent_runtime",
					"index-include":     "DROP INDEX public.idx_ai_client_credential_scope; CREATE UNIQUE INDEX idx_ai_client_credential_scope ON public.ai_client_credentials(tenant_id,user_id,client_name) INCLUDE(api_key)",
					"connection-option": "GRANT CONNECT ON DATABASE " + quote(f.name) + " TO product_agent_runtime WITH GRANT OPTION",
					"schema-option":     "GRANT USAGE ON SCHEMA public TO title_credential_writer WITH GRANT OPTION",
				}[test]
				require.NoError(t, f.db.Exec(sql).Error)
			}
			var before string
			require.NoError(t, f.db.Raw("SELECT value FROM public.preserved_fact").Scan(&before).Error)
			_, err := PrepareRun(context.Background(), f.db)
			require.Error(t, err)
			var after string
			require.NoError(t, f.db.Raw("SELECT value FROM public.preserved_fact").Scan(&after).Error)
			require.Equal(t, before, after)
			if test == "shared-drift" || test == "default-acl" {
				require.False(t, f.db.Migrator().HasTable("product_agent_runs"))
			}
		})
	}
}

func TestPrepareReviewRefusesPartialGrantsAndRetainsExistingFacts(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh", true: "partial"}[partial], func(t *testing.T) {
			f := newPrepareFixture(t, "review")
			require.NoError(t, f.db.Exec("INSERT INTO public.product_title_proposals VALUES('old-org','old-proposal','old-actor','pending',?)", []byte(`{"State":"pending"}`)).Error)
			if partial {
				require.NoError(t, f.db.Exec("GRANT SELECT ON public.product_title_proposals TO product_review_runtime").Error)
			}
			before := retainedSnapshot(t, f.db, []string{"public.product_title_proposals"})
			result, err := PrepareReview(context.Background(), f.db)
			if partial {
				require.Error(t, err)
				require.Equal(t, before, retainedSnapshot(t, f.db, []string{"public.product_title_proposals"}))
				return
			}
			require.NoError(t, err)
			require.Equal(t, "GRANTED", result.Outcome)
			var state string
			require.NoError(t, f.db.Raw("SELECT state FROM public.product_title_proposals WHERE id='old-proposal'").Scan(&state).Error)
			require.Equal(t, "pending", state)
			after := retainedSnapshot(t, f.db, result.Tables)
			_, err = PrepareReview(context.Background(), f.db)
			require.NoError(t, err)
			require.Equal(t, after, retainedSnapshot(t, f.db, result.Tables))
		})
	}
}
