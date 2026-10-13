package supplychainhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	supplyapp "task-processor/internal/app/supplychain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTemuRuleRouteRejectsClientMallCredentialAndDuplicateSelectors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	for _, route := range SupplyRoutes(&supplyapp.Application{}, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	for _, selector := range []string{`{"parentCatId":12,"catId":34}`, `{"catId":34,"mallId":1}`, `{"catId":34,"siteId":142}`, `{"catId":34,"access_token":"isolated-secret-marker"}`, `{"catId":34,"catId":35}`} {
		request := httptest.NewRequest(http.MethodPost, SupplyBasePath+"/target-rules", strings.NewReader(`{"sourceId":"11111111-1111-4111-8111-111111111111","storeId":"22222222-2222-4222-8222-222222222222","effectiveVersion":"1","temuRules":`+selector+`}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if selector == `{"parentCatId":12,"catId":34}` {
			require.Equal(t, http.StatusServiceUnavailable, response.Code, "named metadata selectors are decoded but no capability is fabricated for an unconfigured application")
		} else {
			require.Equal(t, http.StatusBadRequest, response.Code)
		}
		require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		require.NotContains(t, response.Body.String(), "isolated-secret-marker")
	}
}
