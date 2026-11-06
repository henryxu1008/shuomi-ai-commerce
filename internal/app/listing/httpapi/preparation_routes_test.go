package listinghttp

import (
	"context"
	"github.com/stretchr/testify/require"
	listingapp "task-processor/internal/app/listing"
	"testing"
)

func TestPreparationOnlyRoutesExcludeExecutionAndOfficialDependencies(t *testing.T) {
	routes := PreparationRoutes(&listingapp.Application{}, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil })
	require.Len(t, routes, 6)
	for _, r := range routes {
		require.True(t, IsPreparationRoute(r.Method, r.Path))
		require.NotContains(t, r.Path, "operations")
		require.NotContains(t, r.Path, "targets")
		require.NotContains(t, r.Path, "uploads")
		require.NotContains(t, r.Path, "images")
	}
	require.False(t, IsPreparationRoute("POST", ListingBasePath+"/operations"))
	require.False(t, IsPreparationRoute("GET", ListingBasePath+"/:preparation_id/stages"))
}
