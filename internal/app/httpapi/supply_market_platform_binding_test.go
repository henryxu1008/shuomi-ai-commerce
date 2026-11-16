package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/product/supplymarket"
	markethttp "task-processor/internal/product/supplymarket/httpapi"
)

type platformMarketBindingProbe struct {
	markethttp.Service
	authority   *supplymarket.ContextAuthorizer
	calls       int
	actor       string
	memberCalls int
}

func (s *platformMarketBindingProbe) ListApplications(ctx context.Context, _ supplymarket.Query, platform bool) (supplymarket.Page[supplymarket.Record], error) {
	if !platform {
		s.memberCalls++
		return supplymarket.Page[supplymarket.Record]{}, supplymarket.ErrForbidden
	}
	actor, err := s.authority.AuthorizePlatform(ctx)
	if err == nil {
		s.calls++
		s.actor = actor
	}
	return supplymarket.Page[supplymarket.Record]{Items: []supplymarket.Record{}}, err
}

func (s *platformMarketBindingProbe) ExecutePlatform(ctx context.Context, _ string, _ supplymarket.Mutation) (supplymarket.Receipt, error) {
	actor, err := s.authority.AuthorizePlatform(ctx)
	if err == nil {
		s.calls++
		s.actor = actor
	}
	return supplymarket.Receipt{OperationID: "11111111-1111-4111-8111-111111111111", Revision: 1}, err
}

func TestSupplyMarketPlatformBindingUsesExistingGlobalAuthorization(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		for _, tc := range []struct {
			name, user string
			expired    bool
			status     int
		}{
			{"configured platform user", "configured-user", false, http.StatusOK},
			{"ordinary enterprise administrator", "ordinary-user", false, http.StatusForbidden},
			{"expired configured user", "configured-user", true, http.StatusForbidden},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				permissions, err := authz.NewListingKitAuthorizer([]string{"configured-user"}, nil)
				require.NoError(t, err)
				authority, err := supplymarket.NewContextAuthorizer(&productReviewLiveOrganizationAccess{}, permissions)
				require.NoError(t, err)
				service := &platformMarketBindingProbe{authority: authority}
				identity := authidentity.AuthenticatedIdentity{UserID: tc.user, HomeOrganizationID: "home", TenantID: "home", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Minute)}
				if tc.expired {
					identity.TokenExpiresAt = time.Now().Add(-time.Minute)
				}
				routes := markethttp.Routes(service, nil, productReviewCapabilityBinder{now: time.Now}.Bind)
				server := buildCurrentApplicationHTTPServer(routes, routeAuthDependencies{workbenchVerifier: mountedVerifierStub{identity: identity}, authorizer: permissions})
				path, body := markethttp.AdminPath+"/records?kind=official&limit=20", ""
				if method == http.MethodPost {
					path, body = markethttp.AdminPath+"/commands", `{"action":"publish","id":"22222222-2222-4222-8222-222222222222","expectedRevision":1}`
				}
				request := httptest.NewRequest(method, path, strings.NewReader(body))
				if method == http.MethodGet {
					request = httptest.NewRequest(method, path, nil)
				}
				request.Header.Set("Authorization", "Bearer verified-token")
				if method == http.MethodPost {
					request.Header.Set("Content-Type", "application/json")
					request.Header.Set("Idempotency-Key", "33333333-3333-4333-8333-333333333333")
					request.Header.Set("If-Match", `"1"`)
				}
				response := httptest.NewRecorder()
				server.Handler.ServeHTTP(response, request)
				require.Equal(t, tc.status, response.Code, response.Body.String())
				if tc.status == http.StatusOK {
					require.Equal(t, 1, service.calls)
					require.Equal(t, tc.user, service.actor)
				} else {
					require.Zero(t, service.calls)
				}
			})
		}
	}
}

func TestSupplyMarketMemberBindingStillRejectsGlobalOnlyContext(t *testing.T) {
	// A global platform identity must never become a customer member capability.
	binder := productReviewCapabilityBinder{now: time.Now}
	identity := authidentity.AuthenticatedIdentity{UserID: "configured-user", TokenExpiresAt: time.Now().Add(time.Minute)}
	service := &platformMarketBindingProbe{}
	bindCalls := 0
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), identity))
		c.Next()
	})
	for _, route := range markethttp.Routes(service, nil, func(ctx context.Context, authorization string) (context.Context, error) {
		bindCalls++
		return binder.Bind(ctx, authorization)
	}) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	request := httptest.NewRequest(http.MethodGet, markethttp.BasePath+"/records", nil)
	request.Header.Set("Authorization", "Bearer verified-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusForbidden, response.Code)
	require.Equal(t, 1, bindCalls)
	require.Zero(t, service.memberCalls)
}
