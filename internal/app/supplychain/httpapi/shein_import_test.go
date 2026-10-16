package supplychainhttp

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	supplyapp "task-processor/internal/app/supplychain"
	"task-processor/internal/listing/preparation"
	"testing"
)

func TestSheinImportPreviewHasManageBoundaryAndStrictReadOnlyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	found := false
	for _, r := range SupplyRoutes(&supplyapp.Application{}, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(r.Method, r.Path, r.Handler)
		if r.Path == SupplyBasePath+"/shein-completion-preview" {
			found = true
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, preparation.PermissionManage, r.Permission)
		}
	}
	require.True(t, found)
	for _, body := range []string{`{"sourceId":"bad"}`, `{"unknown":true}`, `{"selection":{"title":true,"title":false}}`} {
		request := httptest.NewRequest(http.MethodPost, SupplyBasePath+"/shein-completion-preview", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	}
}
