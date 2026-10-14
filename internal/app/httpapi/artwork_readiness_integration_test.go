package httpapi

import (
	"context"
	"github.com/stretchr/testify/require"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"testing"
)

func TestArtworkReadinessRequiresAdmittedCollectionsAndItsSchema(t *testing.T) {
	f := newAcquisitionHTTPFixture(t)
	ctx := context.Background()
	require.NoError(t, configstore.InstallSchema(f.owner))
	require.NoError(t, configstore.VerifyArtworkConfigurationSchema(ctx, f.owner))
	require.False(t, artworkConfigurationReady(ctx, f.owner, f.owner, false), "installed artwork configuration cannot enable an unadmitted Collection consumer")
	require.False(t, artworkConfigurationReady(ctx, f.owner, f.owner, true), "the save owner must be usable before admitting paid generation")
	require.NoError(t, collectionstore.InstallSchema(f.owner))
	require.True(t, artworkConfigurationReady(ctx, f.owner, f.owner, true))
	require.False(t, artworkConfigurationReady(ctx, f.owner, f.owner, false), "even an existing schema cannot opt in the capability")
}
