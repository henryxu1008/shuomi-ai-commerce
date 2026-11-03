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
	require.Len(t, spec.Columns, 52)
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
