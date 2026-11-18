package productagentruntime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	routestore "task-processor/internal/integration/persistence/routeconfig"
)

func TestPrepareManagedChannelsPreservesRouteOwnerBoundary(t *testing.T) {
	ctx := context.Background()
	f := newPrepareFixture(t, "run")
	_, err := PrepareRun(ctx, f.db)
	require.NoError(t, err)
	// The retained route tables belong to the original configuration owner.
	require.NoError(t, routestore.InstallSchema(ctx, f.db))
	require.NoError(t, f.db.Exec(`CREATE ROLE model_route_config_writer NOLOGIN;
 GRANT USAGE ON SCHEMA public TO model_route_config_writer;
 GRANT SELECT ON public.ai_route_config_revisions, public.ai_route_config_current
 TO model_route_config_writer, product_agent_runtime;
 GRANT SELECT ON public.ai_channel_credential_bindings TO model_route_config_writer`).Error)
	tables := []string{"public.ai_route_config_current", "public.ai_route_config_revisions", "public.ai_channel_connections"}
	before := retainedSnapshot(t, f.db, tables)
	var oldRead bool
	require.NoError(t, f.db.Raw("SELECT has_table_privilege('product_agent_owner','public.ai_route_config_current','SELECT')").Scan(&oldRead).Error)
	require.False(t, oldRead)

	result, err := PrepareManagedModelChannels(ctx, f.db)
	require.NoError(t, err)
	require.Equal(t, "CREATED", result.Outcome)
	require.Equal(t, before, retainedSnapshot(t, f.db, tables))
	require.True(t, routestore.ManagedChannelSchemaInstalled(ctx, f.db))
}
