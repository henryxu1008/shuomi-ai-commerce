package listinghttp

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestControlledLocalDraftClosedRoutesAndDisabledCommandReads(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		routes := ControlledLocalDraftRoutes(nil, nil, enabled)
		expected := 4
		if enabled {
			expected = 8
		}
		require.Len(t, routes, expected)
		for _, r := range routes {
			require.True(t, strings.HasPrefix(r.Path, "/api/v1/workbench/local-platform-drafts/"))
			require.Equal(t, "controlled-local-drafts", r.Module)
			require.True(t, IsControlledLocalDraftRoute(r.Method, r.Path, enabled))
			require.False(t, IsControlledLocalDraftRoute(r.Method, strings.Replace(r.Path, "/local-platform-drafts/", "/supply-preparations/", 1), enabled))
			for _, forbidden := range []string{"upload-plans", "operations", "images/approve", "shein-publication/prepare", "shein-publication/confirm", "shein-color-splits"} {
				require.False(t, IsControlledLocalDraftRoute("POST", ListingBasePath+"/"+forbidden, enabled))
			}
		}
	}
}
