package httpapi

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
)

func TestOwnContentAgentRoutesKeepExplicitFreshBoundaries(t *testing.T) {
	routes := ownContentAgentRoutes(nil)
	require.Len(t, routes, 5)
	for _, r := range routes {
		require.Equal(t, "product-agent", r.Module)
		require.Equal(t, httproute.AuthPolicyVerifiedIdentity, r.AuthPolicy)
		require.Equal(t, httproute.OrganizationAccessPolicyLiveWrite, r.OrganizationAccessPolicy)
		require.NoError(t, validateCollectionDescriptor(r))
		if r.Method == http.MethodGet && r.Path == ownContentAgentSource {
			require.Equal(t, authz.PermissionWorkbenchCollectionRead, r.Permission)
		} else {
			require.Equal(t, authz.PermissionLocalAgentWrite, r.Permission)
			require.Equal(t, 2*time.Minute, r.RequestTimeout)
		}
		bad := r
		bad.Permission = ""
		require.Error(t, validateCollectionDescriptor(bad))
	}
	id, key := uuid.NewString(), uuid.NewString()
	base := "/api/v1/workbench/collections/items/" + id + "/content-agent/runs"
	for _, path := range []string{base, base + "/" + key, base + "/" + key + "/resume", base + "/" + key + "/review"} {
		require.True(t, isProductAgentHTTPPath(path), path)
	}
	for _, path := range []string{base + "/bad", base + "/" + key + "/apply", base + "/" + key + "/review/extra", "/api/v1/workbench/collections/items/bad/content-agent/runs"} {
		require.False(t, isProductAgentHTTPPath(path), path)
	}
}
