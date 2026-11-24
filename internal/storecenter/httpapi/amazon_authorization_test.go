package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
)

const amazonHTTPStoreID = "afafb20d-132d-44c2-bff8-03eb949c7410"
const amazonHTTPCookie = "Gmg87CWS8YpyYsuaAV6L4WHEDgbTSZrbjb4YlL6n03k"

func TestAmazonAuthorizationHTTPRequiresVerifiedScopeAndProtectsFlowCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := &amazonHTTPServiceFixture{}
	handler := &Handler{}
	require.Empty(t, handler.amazonAuthorizationRoutes())
	require.NoError(t, handler.SetAmazonAuthorizations(fixture, func(context.Context) [32]byte { return [32]byte{1} }))
	engine := gin.New()
	for _, route := range handler.amazonAuthorizationRoutes() {
		require.Equal(t, httproute.AuthPolicyCurrentIdentity, route.AuthPolicy)
		require.Equal(t, httproute.OrganizationAccessPolicyLiveWrite, route.OrganizationAccessPolicy)
		require.Equal(t, authz.PermissionWorkbenchStoreUpdate, route.Permission)
		engine.Handle(route.Method, route.Path, route.Handler)
	}
	startPath := "/api/v1/workbench/stores/" + amazonHTTPStoreID + "/amazon-authorization/begin"
	start := httptest.NewRequest(http.MethodPost, startPath, nil)
	start.Header.Set("If-Match", `"3"`)
	start.Header.Set("X-Requested-Organization-ID", "foreign")
	unauthenticated := httptest.NewRecorder()
	engine.ServeHTTP(unauthenticated, start)
	require.Equal(t, http.StatusUnauthorized, unauthenticated.Code)
	require.Zero(t, fixture.begins)

	authenticated := httptest.NewRecorder()
	engine.ServeHTTP(authenticated, withAmazonHTTPIdentity(start))
	require.Equal(t, http.StatusOK, authenticated.Code)
	require.Equal(t, "org-a", fixture.scope.OrganizationID)
	require.Equal(t, [32]byte{1}, fixture.scope.SessionDigest)
	require.Equal(t, "no-store", authenticated.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", authenticated.Header().Get("Referrer-Policy"))
	require.NotContains(t, authenticated.Body.String(), amazonHTTPCookie)
	require.Contains(t, authenticated.Body.String(), "https://sellercentral.amazon.com/apps/authorize/consent")
	cookies := authenticated.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, amazonFlowCookieName, cookies[0].Name)
	require.Equal(t, amazonHTTPCookie, cookies[0].Value)
	require.Equal(t, "/workbench/stores/amazon", cookies[0].Path)
	require.True(t, cookies[0].Secure)
	require.True(t, cookies[0].HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
	require.Positive(t, cookies[0].MaxAge)
}

func TestAmazonAuthorizationHTTPDoesNotAcceptBrowserScopeOrCodeInQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := &amazonHTTPServiceFixture{}
	handler := &Handler{}
	require.NoError(t, handler.SetAmazonAuthorizations(fixture, func(context.Context) [32]byte { return [32]byte{2} }))
	engine := gin.New()
	for _, route := range handler.amazonAuthorizationRoutes() {
		engine.Handle(route.Method, route.Path, route.Handler)
	}
	const callbackPath = "/api/v1/workbench/store-amazon-authorization/redirect"
	const body = `{"state":"good-state","sellerId":"SELLER","code":"private-code"}`
	for _, tc := range []struct {
		path, body string
		cookies    int
		status     int
	}{
		{callbackPath + "?code=private-code", body, 1, 400},
		{callbackPath, `{"state":"s","sellerId":"SELLER","code":"private-code","sessionDigest":"injected"}`, 1, 400},
		{callbackPath, body, 0, 400},
		{callbackPath, body, 2, 400},
		{callbackPath, body, 1, 200},
	} {
		request := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
		request.Header.Set("Content-Type", "application/json")
		for range tc.cookies {
			request.AddCookie(&http.Cookie{Name: amazonFlowCookieName, Value: amazonHTTPCookie})
		}
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, withAmazonHTTPIdentity(request))
		require.Equal(t, tc.status, response.Code, response.Body.String())
		require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		require.NotContains(t, response.Body.String(), "private-code")
	}
	require.Equal(t, 1, fixture.completes)
	require.Equal(t, amazonHTTPCookie, fixture.callback.FlowCookie)
	require.Equal(t, "private-code", fixture.callback.Code)
	require.Equal(t, [32]byte{2}, fixture.scope.SessionDigest)
}

func withAmazonHTTPIdentity(request *http.Request) *http.Request {
	return request.WithContext(authidentity.WithAuthenticatedIdentity(request.Context(), authidentity.AuthenticatedIdentity{
		TenantID: "org-a", EffectiveOrganizationID: "org-a", UserID: "actor", EffectiveMemberID: "membership",
	}))
}

type amazonHTTPServiceFixture struct {
	begins, completes int
	scope             storecenter.AmazonBrowserScope
	callback          storecenter.AmazonRedirectCallback
}

func (f *amazonHTTPServiceFixture) Begin(_ context.Context, scope storecenter.AmazonBrowserScope, _ string, _ int64) (storecenter.AmazonConnectionStart, error) {
	f.begins++
	f.scope = scope
	return storecenter.AmazonConnectionStart{
		AuthorizationURL: "https://sellercentral.amazon.com/apps/authorize/consent?state=fixture",
		FlowCookie:       amazonHTTPCookie, ExpiresAt: time.Now().UTC().Add(time.Minute),
	}, nil
}

func (*amazonHTTPServiceFixture) Continue(context.Context, storecenter.AmazonBrowserScope, storecenter.AmazonLoginCallback) (string, error) {
	return "https://amazon.com/apps/authorize/confirm/fixture", nil
}

func (f *amazonHTTPServiceFixture) Complete(_ context.Context, scope storecenter.AmazonBrowserScope, callback storecenter.AmazonRedirectCallback) (storecenter.AmazonConnectionResult, error) {
	f.completes++
	f.scope, f.callback = scope, callback
	return storecenter.AmazonConnectionResult{StoreID: amazonHTTPStoreID, Generation: 1}, nil
}
