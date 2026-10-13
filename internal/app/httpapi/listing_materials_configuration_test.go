package httpapi

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agentconfig"
	"testing"
)

func TestListingMaterialsCatalogKeepsOwnConfigurationAndTruthfulReadiness(t *testing.T) {
	entries, err := (productAgentCatalog{}).ReadCatalog(context.Background())
	require.NoError(t, err)
	var found []agentconfig.CatalogEntry
	for _, entry := range entries {
		if entry.Definition.ID == agentconfig.MaterialsAgentID {
			found = append(found, entry)
		}
	}
	require.Len(t, found, 1, "one registered materials owner, no title alias")
	require.Equal(t, agentconfig.MaterialsParameterSchema, found[0].ParameterSchema)
	require.Equal(t, agentconfig.MaterialsAgentVersion, found[0].Definition.Version)
	require.Equal(t, []string{"listing.materials.inspect"}, []string{found[0].Definition.AllowedTools[0].ID})
	capabilities := listingMaterialsAgentCapabilities(context.Background(), nil)
	require.Equal(t, "NEEDS_CONFIGURATION", capabilities[0].Readiness)
	for _, capability := range capabilities {
		require.NotEqual(t, "AVAILABLE", capability.Readiness)
	}
	require.Equal(t, "NOT_SUPPORTED", capabilities[len(capabilities)-1].Support, "external publication never enabled")
}
