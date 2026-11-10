package listinghttp

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestControlledLocalDraftClosedRoutesAndDisabledCommandReads(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		routes := ControlledLocalDraftRoutes(nil, nil, enabled)
		expected := 3
		if enabled {
			expected = 7
		}
		require.Len(t, routes, expected)
		for _, r := range ListingRoutes(nil, nil) {
			admitted := IsControlledLocalDraftRoute(r.Method, r.Path, enabled)
			if r.Path == ListingBasePath+"/target-commands/:key" {
				require.True(t, admitted)
			}
			for _, forbidden := range []string{"upload-plans", "operations", "images/approve", "shein-publication/prepare", "shein-publication/confirm", "shein-color-splits"} {
				require.False(t, IsControlledLocalDraftRoute("POST", ListingBasePath+"/"+forbidden, enabled))
			}
		}
	}
}
