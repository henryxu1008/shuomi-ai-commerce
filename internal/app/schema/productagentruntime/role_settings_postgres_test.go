package productagentruntime

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/integration/openai"
)

func TestPrepareRejectsEffectiveReadOnlyRuntimeLoginWithoutRepair(t *testing.T) {
	for _, role := range []string{"product_agent_runtime", "title_credential_writer", "product_review_runtime"} {
		for _, databaseOnly := range []bool{false, true} {
			scope := "global"
			if databaseOnly {
				scope = "database"
			}
			t.Run(role+"/"+scope, func(t *testing.T) {
				target, prepareTarget := "run", PrepareRun
				if role == "product_review_runtime" {
					target, prepareTarget = "review", PrepareReview
				}
				f := newPrepareFixture(t, target)
				_, err := prepareTarget(context.Background(), f.db)
				require.NoError(t, err)
				prefix := "ALTER ROLE " + quote(role)
				if databaseOnly {
					prefix += " IN DATABASE " + quote(f.name)
				}
				require.NoError(t, f.db.Exec(prefix+" SET default_transaction_read_only=on").Error)
				t.Cleanup(func() { require.NoError(t, f.db.Exec(prefix+" RESET default_transaction_read_only").Error) })
				var readOnly bool
				runtime := f.role(t, role)
				require.NoError(t, runtime.Raw("SELECT current_setting('default_transaction_read_only')::boolean").Scan(&readOnly).Error)
				require.True(t, readOnly, "actual role login differs from SET LOCAL ROLE")
				var writeErr error
				switch role {
				case "product_agent_runtime":
					writeErr = runtime.Exec("INSERT INTO public.product_agent_runs VALUES('run','org','actor','acquisition','context','request',?,1,'running',?)", strings.Repeat("a", 64), []byte(`{}`)).Error
				case "title_credential_writer":
					writeErr = openai.NewGormCredentialResolver(runtime).SaveCredential(context.Background(), openai.AIClientCredential{TenantID: "org", ClientName: "content", APIKey: "synthetic-only", BaseURL: "https://example.test/v1", Model: "synthetic", Enabled: true})
				default:
					writeErr = runtime.Exec("INSERT INTO public.product_title_proposals VALUES('org','proposal','actor','pending',?)", []byte(`{"State":"pending"}`)).Error
				}
				require.ErrorContains(t, writeErr, "read-only")
				tables := []string{"public.preserved_fact", "public.product_agent_runs", "public.product_agent_tool_calls", "public.ai_client_credentials", "public.ai_invocations"}
				if target == "review" {
					tables = []string{"public.preserved_fact", "public.product_title_proposals", "public.product_title_operations", "public.product_snapshot_versions", "public.product_snapshot_heads", "public.product_source_publications", "public.product_source_publication_receipts"}
				}
				before := retainedSnapshot(t, f.db, tables)
				_, err = prepareTarget(context.Background(), f.db)
				require.Error(t, err, "a read-only serving identity cannot be qualified for durable writes")
				require.Equal(t, before, retainedSnapshot(t, f.db, tables))
				var retained bool
				require.NoError(t, f.db.Raw("SELECT option_value::boolean FROM pg_catalog.pg_db_role_setting s CROSS JOIN LATERAL pg_catalog.pg_options_to_table(s.setconfig) o WHERE s.setrole=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname=?) AND s.setdatabase=? AND o.option_name='default_transaction_read_only'", role, func() uint32 {
					if !databaseOnly {
						return 0
					}
					var oid uint32
					require.NoError(t, f.db.Raw("SELECT oid FROM pg_catalog.pg_database WHERE datname=current_database()").Scan(&oid).Error)
					return oid
				}()).Scan(&retained).Error)
				require.True(t, retained, "qualification must not repair role settings")
			})
		}
	}
}

func TestPrepareAcceptsDatabaseReadWriteOverrideForRuntimeRole(t *testing.T) {
	f := newPrepareFixture(t, "run")
	require.NoError(t, f.db.Exec("ALTER ROLE product_agent_runtime SET default_transaction_read_only=on").Error)
	t.Cleanup(func() {
		require.NoError(t, f.db.Exec("ALTER ROLE product_agent_runtime RESET default_transaction_read_only").Error)
	})
	require.NoError(t, f.db.Exec("ALTER ROLE product_agent_runtime IN DATABASE "+quote(f.name)+" SET default_transaction_read_only=off").Error)
	_, err := PrepareRun(context.Background(), f.db)
	require.NoError(t, err, "effective database override is writable and must be preserved")
	var readOnly bool
	runtime := f.role(t, "product_agent_runtime")
	require.NoError(t, runtime.Raw("SELECT current_setting('default_transaction_read_only')::boolean").Scan(&readOnly).Error)
	require.False(t, readOnly)
	require.NoError(t, runtime.Exec("INSERT INTO public.product_agent_runs VALUES('run','org','actor','acquisition','context','request',?,1,'running',?)", strings.Repeat("a", 64), []byte(`{}`)).Error)
}
