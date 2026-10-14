package httpapi

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOwnHistoryRoutesRejectInvalidQueryAndBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	for _, route := range Routes(&serviceFixture{}, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	base := BasePath + "/items/" + uuid.NewString() + "/history"
	for _, suffix := range []string{"?limit=21", "?limit=01", "?before=0", "?before=01", "?before=9223372036854775808", "?before=1&before=2", "?productKey=foreign", "/01", "/0", "/2?publicationId=foreign"} {
		result := httptest.NewRecorder()
		router.ServeHTTP(result, httptest.NewRequest(http.MethodGet, base+suffix, nil))
		require.Equal(t, http.StatusBadRequest, result.Code, suffix)
	}
	for _, suffix := range []string{"", "/1"} {
		req := httptest.NewRequest(http.MethodGet, base+suffix, strings.NewReader("{}"))
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		require.Equal(t, http.StatusBadRequest, out.Code)
		req = httptest.NewRequest(http.MethodGet, base+suffix, nil)
		req.Header.Set("Idempotency-Key", uuid.NewString())
		out = httptest.NewRecorder()
		router.ServeHTTP(out, req)
		require.Equal(t, http.StatusBadRequest, out.Code)
	}
}
