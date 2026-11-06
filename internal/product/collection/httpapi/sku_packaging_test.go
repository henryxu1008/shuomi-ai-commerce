package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOwnSKUPackagingStrictHTTPIngress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &serviceFixture{}
	router := gin.New()
	for _, route := range Routes(service, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) { router.Handle(route.Method, route.Path, route.Handler) }
	for _, test := range []struct { value string; want int }{
		{`{"weightGrams":12,"lengthCm":3}`, http.StatusOK},
		{`{"widthCm":null}`, http.StatusOK},
		{`null`, http.StatusOK},
		{`{"weightGrams":"12"}`, http.StatusBadRequest},
		{`{"weightGrams":12,"unit":"kg"}`, http.StatusBadRequest},
		{`{"weightGrams":12,"weightGrams":13}`, http.StatusBadRequest},
		{`[]`, http.StatusBadRequest},
	} {
		request := httptest.NewRequest(http.MethodPost, BasePath+"/commands", strings.NewReader(`{"action":"create_product","product":{"title":"Measured","description":"","images":[],"variants":[{"sourceId":"red","sku":"RED","title":"Red","attributes":{},"currency":"CNY","price":0,"stock":0,"packaging":`+test.value+`}]}}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", uuid.NewString())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, test.want, response.Code, "%s: %s", test.value, response.Body.String())
	}
}
