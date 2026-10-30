package httpapi

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strings"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	kh "task-processor/internal/browserkernel/httpapi"
	"task-processor/internal/httproute"
	"testing"
	"time"
)

func TestKernelCatalogExactRouteAdmission(t *testing.T) {
	base := []httproute.Descriptor{}
	for _, r := range currentWorkbenchApplicationRoutes {
		base = append(base, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	all := append(append([]httproute.Descriptor{}, base...), kh.Routes(nil)...)
	validate := func(r []httproute.Descriptor, enabled bool) error {
		return validateCurrentApplicationRoutesInternal(r, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{BrowserKernel: enabled})
	}
	require.NoError(t, validate(base, false))
	require.Error(t, validate(all, false))
	require.Error(t, validate(base, true))
	require.NoError(t, validate(all, true))
	for i := len(base); i < len(all); i++ {
		for _, change := range []func(*httproute.Descriptor){func(r *httproute.Descriptor) {
			r.Permission = ""
			if r.Method == "GET" {
				r.Permission = authz.PermissionListingKitPlatformAdm
			}
		}, func(r *httproute.Descriptor) { r.AuthPolicy = httproute.AuthPolicyPublic }, func(r *httproute.Descriptor) {
			r.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyLiveWrite
		}, func(r *httproute.Descriptor) { r.RequestTimeout = 0 }, func(r *httproute.Descriptor) { r.RejectUnreadRequestBody = !r.RejectUnreadRequestBody }, func(r *httproute.Descriptor) { r.Module = "store-center" }} {
			copy := append([]httproute.Descriptor{}, all...)
			change(&copy[i])
			require.Error(t, validate(copy, true))
		}
	}
}
func TestKernelCatalogGlobalAuthorityMiddleware(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		roles, users, globalRoles []string
		authenticated, write      bool
	}{
		{name: "anonymous"}, {name: "ordinary", authenticated: true}, {name: "tenant-admin", roles: []string{"admin", "listingkit_admin"}, authenticated: true}, {name: "verified-global-role", roles: []string{"platform_admin"}, authenticated: true, write: true}, {name: "configured-global-user", users: []string{"verified-user"}, authenticated: true, write: true}, {name: "configured-global-role", roles: []string{"support-platform"}, globalRoles: []string{"support-platform"}, authenticated: true, write: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			authorizer, e := authz.NewListingKitAuthorizer(tc.users, tc.globalRoles)
			require.NoError(t, e)
			for _, route := range kh.Routes(nil) {
				calls := 0
				read := route.Method == "GET"
				route.Handler = func(c *gin.Context) {
					calls++
					identity, ok := authidentity.AuthenticatedIdentityFromContext(c.Request.Context())
					require.True(t, ok)
					require.Empty(t, identity.TenantID)
					require.Empty(t, identity.OrganizationGrants)
					c.Status(204)
				}
				router := gin.New()
				mountRoutesWithAuthDependencies(router, []httproute.Descriptor{route}, routeAuthDependencies{workbenchVerifier: mountedVerifierStub{identity: authidentity.AuthenticatedIdentity{UserID: "verified-user", TenantID: "tenant", Roles: tc.roles, TokenExpiresAt: time.Now().Add(time.Minute), OrganizationGrants: []authidentity.OrganizationGrant{{OrganizationID: "other-tenant", Roles: []string{"platform_admin", "admin"}}}}}, authorizer: authorizer})
				path := strings.ReplaceAll(route.Path, ":releaseId", "00000000-0000-4000-8000-000000000001")
				req := httptest.NewRequest(route.Method, path, nil)
				if tc.authenticated {
					req.Header.Set("Authorization", "Bearer fixture")
				}
				req.Header.Set("X-Organization-ID", "other-tenant")
				req.Header.Set("X-Roles", "platform_admin")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				if tc.authenticated && (read || tc.write) {
					require.Equal(t, 204, w.Code, w.Body.String())
					require.Equal(t, 1, calls)
				} else {
					require.Contains(t, []int{401, 403}, w.Code, w.Body.String())
					require.Zero(t, calls)
				}
			}
		})
	}
}
