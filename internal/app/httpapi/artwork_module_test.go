package httpapi

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authz"
	"task-processor/internal/imageagent"
	"testing"
)

func TestPODArtworkRegistrationHasExactCreativeSchema(t *testing.T) {
	entries, err := (productAgentCatalog{}).ReadCatalog(context.Background())
	require.NoError(t, err)
	found := false
	for _, e := range entries {
		if e.Definition.ID == agentconfig.ArtworkAgentID {
			found = true
			require.Equal(t, agentconfig.ArtworkAgentVersion, e.Definition.Version)
			require.Equal(t, agentconfig.ArtworkParameterSchema, e.ParameterSchema)
		}
	}
	require.True(t, found, "POD must be independently configurable, not a product-image alias")
}

func TestPODArtworkRoutesAndCommandsKeepExactBoundaries(t *testing.T) {
	for _, route := range (fullImageModule{}).artworkRoutes() {
		want := authz.PermissionImageAgentWrite
		if route.Method == http.MethodGet || route.Path == artworkAgentBase+"/prepare" {
			want = authz.PermissionImageAgentRead
		}
		require.Equal(t, want, route.Permission)
	}
	require.Error(t, validateArtworkCommandBody("confirm", artworkCommandBody{PlanRevision: 1}))
	require.NoError(t, validateArtworkCommandBody("confirm", artworkCommandBody{PlanRevision: 1, PlanDigest: strings.Repeat("a", 64), QuoteDigest: strings.Repeat("b", 64)}))
	require.Error(t, validateArtworkCommandBody("cancel", artworkCommandBody{PlanRevision: 1, ResultDigest: strings.Repeat("c", 64)}))
	require.Error(t, validateArtworkHTTPRun("org", "actor", "member", "run", imageagent.RunProjection{}))
}
