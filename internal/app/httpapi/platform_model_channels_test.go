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

func TestPlatformChannelPriceWritesRequireTheSameNativeAuthorization(t *testing.T) {
	a, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	m := modelRoutePricingModule{authorizer: a, services: ModelRoutePricingDependencies{PlatformChannels: &PlatformModelChannels{
		MaintainerOrganization: "platform-org", ProjectID: "project",
	}}}
	request := func(org, project string) int {
		id := authidentity.AuthenticatedIdentity{UserID: "actor", TenantID: org, HomeOrganizationID: org, EffectiveOrganizationID: org,
			EffectiveMemberID: "member", Roles: []string{"platform_admin"}, TokenExpiresAt: time.Now().Add(time.Minute),
			OrganizationGrants: []authidentity.OrganizationGrant{{AuthorizationID: "member", OrganizationID: org, ProjectID: project, Roles: []string{"platform_admin"}}}}
		rr := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rr)
		req := httptest.NewRequest(http.MethodPost, modelRouteBase+"/platform-prices/publish", strings.NewReader("{}"))
		ctx.Request = req.WithContext(authidentity.WithAuthenticatedIdentity(req.Context(), id))
		m.serve(ctx, "publish", true)
		return rr.Code
	}
	require.Equal(t, http.StatusForbidden, request("other-org", "project"))
	require.Equal(t, http.StatusForbidden, request("platform-org", "other-project"))
	require.Equal(t, http.StatusServiceUnavailable, request("platform-org", "project"), "native maintainer reaches the pricing owner")
}
