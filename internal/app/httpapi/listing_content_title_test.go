package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	listingapp "task-processor/internal/app/listing"
	listinghttp "task-processor/internal/app/listing/httpapi"
)

func TestContentTitleOnlyDoesNotConnectAnEmptySupplyTitleOptimizer(t *testing.T) {
	parent := &productAgentApplication{config: ProductAgentDependencies{ContentTitleOnly: true}}
	app := &listingapp.Application{}
	bridge, err := connectListingProductAgent(context.Background(), nil, app, parent, listingapp.OrganizationExecutionAuthorizer{})
	require.NoError(t, err, "no title optimizer must be installed or initialized for the content-only parent")
	require.Nil(t, bridge)
	require.Nil(t, parent.listingExecution)
	require.Nil(t, app.OptimizationOptions)
	require.Nil(t, app.AuthorizeOptimization)
	router := gin.New()
	for _, route := range listinghttp.ListingRoutes(app, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, listinghttp.ListingBasePath+"/optimization-options?limit=100", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), "智能体优化未配置")
}
