package httpapi

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	fx "task-processor/internal/app/runtime/automaticfx"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
)

type fakeFXControl struct{ starts int }

func (f *fakeFXControl) Status(context.Context) (fx.StatusSnapshot, error) {
	return fx.StatusSnapshot{State: "CHECKED"}, nil
}
func (f *fakeFXControl) Start(context.Context) error { f.starts++; return nil }
func (f *fakeFXControl) Stop(context.Context) error  { return fx.ErrControlUnknown }

func TestAutomaticFXControlRequiresPlatformAuthority(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a, err := authz.NewListingKitAuthorizer([]string{"maintainer"}, nil)
	require.NoError(t, err)
	id := authidentity.AuthenticatedIdentity{UserID: "ordinary", Roles: []string{"admin"}, TenantID: "org", EffectiveOrganizationID: "org", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
	f := &fakeFXControl{}
	m := modelRoutePricingModule{services: ModelRoutePricingDependencies{AutomaticFX: f}, authorizer: a}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), id))
	})
	for _, route := range m.routes() {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	request := func(method, path string) int {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(method, modelRouteBase+path, nil))
		return w.Code
	}
	require.Equal(t, 403, request("POST", "/platform-prices/automatic-fx/start"))
	require.Zero(t, f.starts)
	id.UserID = "maintainer"
	id.Roles = []string{"listingkit_admin"}
	require.Equal(t, 200, request("GET", "/platform-prices/automatic-fx"))
	require.Equal(t, 200, request("POST", "/platform-prices/automatic-fx/start"))
	require.Equal(t, 1, f.starts)
	require.Equal(t, 503, request("POST", "/platform-prices/automatic-fx/stop"))
}
