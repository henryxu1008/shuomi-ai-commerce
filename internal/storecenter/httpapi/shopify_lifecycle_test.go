package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
)

type shopifyNoticeRouteFixture struct {
	ShopifyConnectionService
	enabled bool
	body    string
	calls   int
	failure error
}

func TestIsolatedShopifyPrivacyRoutesAreMountedOnlyWithReceiptReceiver(t *testing.T) {
	base := module.NewRegistry()
	require.NoError(t, NewModule(&Handler{}).Register(base))
	for _, route := range base.Routes() {
		require.NotEqual(t, ShopifyShopRedactPath, route.Path)
	}
	h := &Handler{}
	receive := func(context.Context, string, string, string, []byte) error { return nil }
	require.NoError(t, h.SetIsolatedShopifyPrivacyReceiver(receive))
	registry := module.NewRegistry()
	require.NoError(t, NewModule(h).Register(registry))
	found := false
	for _, route := range registry.Routes() {
		if route.Method == http.MethodPost && route.Path == ShopifyShopRedactPath {
			found = true
			require.NotNil(t, route.Handler)
			require.Empty(t, route.Permission)
			require.Equal(t, httproute.AuthPolicyPublic, route.AuthPolicy)
		}
	}
	require.True(t, found)
}

func (f *shopifyNoticeRouteFixture) LifecycleNotificationsEnabled() bool           { return f.enabled }
func (*shopifyNoticeRouteFixture) ReconcileShopifyLifecycle(context.Context) error { return nil }
func (f *shopifyNoticeRouteFixture) ReceiveShopifyLifecycle(_ context.Context, _, _, _ string, body []byte) error {
	f.calls++
	f.body = string(body)
	return f.failure
}

func TestShopifyLifecycleRouteRequiresExactTopicAndDurableAcceptance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &shopifyNoticeRouteFixture{}
	h := &Handler{shopifyConnections: service}
	require.Empty(t, h.shopifyLifecycleRoutes())
	service.enabled = true
	routes := h.shopifyLifecycleRoutes()
	require.Len(t, routes, 2)
	r := gin.New()
	for _, route := range routes {
		r.POST(route.Path, route.Handler)
	}
	for _, tc := range []struct {
		name, path, topic, body string
		signature               bool
		failure                 error
		status                  int
		accepted                bool
	}{
		{"swapped topic", ShopifyScopesUpdatedPath, "app/uninstalled", "{}", true, nil, 400, false},
		{"missing signature", ShopifyUninstalledPath, "app/uninstalled", "{}", false, nil, 401, false},
		{"over budget", ShopifyUninstalledPath, "app/uninstalled", strings.Repeat("x", 65537), true, nil, 400, false},
		{"invalid signature", ShopifyUninstalledPath, "app/uninstalled", "{}", true, storecenter.ErrOfficialAuthorizationRejected, 401, true},
		{"receipt failure", ShopifyUninstalledPath, "app/uninstalled", "{}", true, storecenter.ErrOfficialConnectionUnavailable, 503, true},
		{"original raw body", ShopifyScopesUpdatedPath, "app/scopes_update", "{ \"id\": 1 }\n", true, nil, 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service.calls, service.body, service.failure = 0, "", tc.failure
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Shopify-Event-Id", "original-event")
			req.Header.Set("X-Shopify-Topic", tc.topic)
			if tc.signature {
				req.Header.Set("X-Shopify-Hmac-Sha256", strings.Repeat("a", 43)+"=")
			}
			res := httptest.NewRecorder()
			r.ServeHTTP(res, req)
			require.Equal(t, tc.status, res.Code)
			require.Empty(t, res.Body.String())
			if tc.accepted {
				require.Equal(t, 1, service.calls)
				require.Equal(t, tc.body, service.body)
			} else {
				require.Zero(t, service.calls)
			}
		})
	}
}
