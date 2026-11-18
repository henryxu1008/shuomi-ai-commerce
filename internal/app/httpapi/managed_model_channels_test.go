package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
)

func TestManagedChannelHTTPRequiresExplicitRoleAndKeyAuthority(t *testing.T) {
	a, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	m := modelRoutePricingModule{authorizer: a}
	r := gin.New()
	id := authidentity.AuthenticatedIdentity{
		UserID: "actor", EffectiveMemberID: "member", EffectiveOrganizationID: "org", TenantID: "org",
		TokenExpiresAt: time.Now().Add(time.Minute), Roles: []string{"listingkit_admin", authz.EnterpriseRoleKey("org", 1)},
	}
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), id))
	})
	for _, route := range m.managedChannelRoutes() {
		r.Handle(route.Method, route.Path, route.Handler)
	}
	request := func(method, suffix string) int {
		body := strings.NewReader(`{"expectedMemberId":"member","input":{"template":"openai-mini-v1","name":"Fixture","apiKey":"synthetic"}}`)
		req := httptest.NewRequest(method, modelRouteBase+"/managed-channels"+suffix, body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "fixture")
		result := httptest.NewRecorder()
		r.ServeHTTP(result, req)
		return result.Code
	}
	for _, modules := range []modelQueryRoleFixture{nil, {"ai-credentials"}, {"ai-channels"}} {
		a.SetRolePolicyReader(modules)
		require.Equal(t, http.StatusForbidden, request(http.MethodPost, "/create"))
	}
	a.SetRolePolicyReader(modelQueryRoleFixture{"ai-channels", "ai-credentials"})
	require.Equal(t, http.StatusServiceUnavailable, request(http.MethodPost, "/create"), "permissions permit reaching the unconfigured owner")
	id.TenantID = "other-org"
	require.Equal(t, http.StatusForbidden, request(http.MethodPost, "/create"))
}
