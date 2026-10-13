package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"

	"github.com/gin-gonic/gin"
)

func TestRoutePolicyPreservesVerifiedIdentityAndExplicitPublicAuthority(t *testing.T) {
	for _, route := range []httproute.Descriptor{
		{Method: http.MethodPost, Path: "/api/v1/image-agent/runs", Module: "image-agent"},
		{Method: http.MethodGet, Path: "/api/v1/current", AuthPolicy: httproute.AuthPolicyVerifiedIdentity},
	} {
		if !RouteRequiresZitadelAuth(route) {
			t.Errorf("verified route lost its identity requirement: %+v", route)
		}
	}
	if RouteRequiresZitadelAuth(httproute.Descriptor{Method: http.MethodPost, Path: "/api/v1/identity/notifications/sms", AuthPolicy: httproute.AuthPolicyPublic}) {
		t.Fatal("explicit current provider authentication must remain independent of user bearer authentication")
	}
}

func TestRoleMiddlewareUsesVerifiedContextAndOriginalPermissionPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authorizer, err := authz.NewListingKitAuthorizer(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		identity *authidentity.AuthenticatedIdentity
		want     int
	}{
		{name: "forged headers have no authority", want: http.StatusForbidden},
		{name: "verified viewer cannot write", identity: &authidentity.AuthenticatedIdentity{UserID: "viewer", Roles: []string{"listingkit_viewer"}}, want: http.StatusForbidden},
		{name: "verified admin uses existing policy", identity: &authidentity.AuthenticatedIdentity{UserID: "admin", Roles: []string{"listingkit_admin"}}, want: http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			route := httproute.Descriptor{Method: http.MethodPost, Path: "/current", Permission: authz.PermissionListingKitAdminWrite}
			router.POST(route.Path, NewRouteRoleMiddlewareWithAuthorizer(route, authorizer), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			request := httptest.NewRequest(route.Method, route.Path, nil)
			request.Header.Set("X-User-ID", "admin")
			request.Header.Set("X-User-Roles", "listingkit_admin,platform_admin")
			if test.identity != nil {
				request = request.WithContext(authidentity.WithAuthenticatedIdentity(context.Background(), *test.identity))
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.want, response.Body.String())
			}
		})
	}
}
