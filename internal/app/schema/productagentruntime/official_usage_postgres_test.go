package productagentruntime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvocationContractIncludesBillableMetadataWithoutUpgradingOldSchema(t *testing.T) {
	f := newPrepareFixture(t, "run")
	spec, err := invocationContract(f.db)
	require.NoError(t, err)
	require.Len(t, spec.Columns, 54)
	found := map[string]bool{}
	for _, column := range spec.Columns {
		if column.Name == "billable_usage_json" || column.Name == "official_tariff_json" {
			found[column.Name] = true
			require.Equal(t, "text", column.Type)
			require.Equal(t, "'null'::text", column.Default)
			require.True(t, column.NotNull)
		}
	}
	require.True(t, found["billable_usage_json"])
	require.True(t, found["official_tariff_json"])
	require.NoError(t, VerifyInvocationRelation(context.Background(), f.db, "product_agent_owner"))
	require.NoError(t, f.db.Exec("ALTER TABLE public.ai_invocations DROP COLUMN billable_usage_json").Error)
	_, err = PrepareRun(context.Background(), f.db)
	require.Error(t, err, "a relation missing required official usage metadata must be rejected, never repaired")
	require.False(t, f.db.Migrator().HasColumn("ai_invocations", "billable_usage_json"))
}

func TestPrepareInvocationBillingModePreservesOriginalFactsAndAccess(t *testing.T) {
	f := newPrepareFixture(t, "run")
	ctx := context.Background()
	_, err := PrepareRun(ctx, f.db)
	require.NoError(t, err)
	installOriginalInvocationAuditReader(t, f.db)
	require.NoError(t, f.db.Exec("ALTER TABLE public.ai_invocations DROP COLUMN billing_mode").Error)
	require.NoError(t, f.db.Exec("INSERT INTO public.ai_invocations(invocation_id,tenant_id,operation,outcome) VALUES('retained-charge','org','product_agent_decision','dispatched')").Error)

	var original string
	require.NoError(t, f.db.Raw("SELECT to_jsonb(v)::text FROM ai_invocations v WHERE invocation_id='retained-charge'").Scan(&original).Error)
	result, err := PrepareInvocationBillingMode(ctx, f.db)
	require.NoError(t, err)
	require.Equal(t, "CREATED", result.Outcome)
	var after string
	require.NoError(t, f.db.Raw("SELECT (to_jsonb(v)-'billing_mode')::text FROM ai_invocations v WHERE invocation_id='retained-charge' AND billing_mode IS NULL").Scan(&after).Error)
	require.Equal(t, original, after)
	runtime := f.role(t, "product_agent_runtime")
	require.NoError(t, VerifyInvocationRecoveryRuntime(ctx, runtime))
	require.NoError(t, runtime.Exec("INSERT INTO public.ai_invocations(invocation_id,billing_mode) VALUES('owned','usage_only')").Error)
	require.Error(t, runtime.Exec("INSERT INTO public.ai_invocations(invocation_id,billing_mode) VALUES('forged','exempt')").Error)
	require.Error(t, runtime.Exec("UPDATE public.ai_invocations SET billing_mode='platform_points' WHERE invocation_id='owned'").Error)
	var auditReader bool
	require.NoError(t, f.db.Raw("SELECT has_table_privilege('account_audit_product_reader','public.ai_invocations','SELECT')").Scan(&auditReader).Error)
	require.True(t, auditReader)
	result, err = PrepareInvocationBillingMode(ctx, f.db)
	require.NoError(t, err)
	require.Equal(t, "ALREADY_READY", result.Outcome)
}
