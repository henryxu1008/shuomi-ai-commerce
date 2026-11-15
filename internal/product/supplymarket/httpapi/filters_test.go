package httpapi

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"task-processor/internal/product/supplymarket"
	"testing"
)

type filterService struct {
	fixtureService
	query supplymarket.Query
}

func (s *filterService) ListMarket(_ context.Context, q supplymarket.Query) (supplymarket.Page[supplymarket.Release], error) {
	s.calls++
	s.query = q
	return supplymarket.Page[supplymarket.Release]{Items: []supplymarket.Release{}}, nil
}
func TestMarketFilterQueryIsValidatedAndMarketOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &filterService{}
	router := gin.New()
	for _, route := range Routes(service, nil, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	values := url.Values{"kind": {"official"}, "province": {"浙江省"}, "city": {"杭州市"}, "minPrice": {"0"}, "maxPrice": {"50.25"}, "priceCurrency": {"CNY"}, "maxMOQ": {"2"}, "maxLeadDays": {"0"}}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+"/releases?"+values.Encode(), nil))
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Equal(t, 1, service.calls)
	require.Equal(t, supplymarket.MarketFilters{Province: "浙江省", City: "杭州市", PriceCurrency: "CNY", MinPrice: new(0.0), MaxPrice: new(50.25), MaxMOQ: new(2), MaxLeadDays: new(0)}, service.query.Filters)
	for _, query := range []string{"minPrice=-1&priceCurrency=CNY", "minPrice=10&maxPrice=2&priceCurrency=CNY", "minPrice=NaN&priceCurrency=CNY", "minPrice=Inf&priceCurrency=CNY", "minPrice=1e2&priceCurrency=CNY", "minPrice=0", "priceCurrency=CNY", "maxMOQ=0", "maxMOQ=1.5", "maxLeadDays=3651", "maxLeadDays=1&maxLeadDays=2", "province=a&province=b", "minPrice=999999999999999999999999999999999&priceCurrency=CNY"} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+"/releases?"+query, nil))
		require.Equal(t, 400, response.Code, query)
	}
	for _, path := range []string{"/records?province=x", "/records/11111111-1111-4111-8111-111111111111/events?maxMOQ=2", "/records/11111111-1111-4111-8111-111111111111/releases?maxLeadDays=2"} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+path, nil))
		require.Equal(t, 400, response.Code, path)
	}
	for _, query := range []string{"city=" + url.QueryEscape("杭州市"), "province=x&city=a&city=b", "province=x&city=" + strings.Repeat("a", 101)} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+"/releases?"+query, nil))
		require.Equal(t, 400, response.Code, query)
	}
	require.Equal(t, 1, service.calls)
}
