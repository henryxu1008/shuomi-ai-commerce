package httpapi

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"testing"
)

type quoteDetailFixture struct {
	serviceFixture
	detail collection.ItemDetail
}

func (f *quoteDetailFixture) ReadItem(context.Context, string) (collection.ItemDetail, error) {
	return f.detail, nil
}

func TestQuoteDetailResponseHasBoundedRoomWithoutTruncatingKnownProduct(t *testing.T) {
	f := &quoteDetailFixture{detail: collection.ItemDetail{Product: catalog.ProductSnapshot{Description: strings.Repeat("x", collection.MaxPayloadBytes-64)}, SourceQuotes: &collection.SourceQuotes{State: "available", PublicationID: "fixed", Version: 1, Prices: []collection.SourcePrice{{Amount: "27.00"}}}}}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	for _, route := range Routes(f, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+"/items/"+uuid.NewString(), nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Greater(t, response.Body.Len(), collection.MaxPayloadBytes)
	var actual collection.ItemDetail
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &actual))
	require.Equal(t, f.detail.Product.Description, actual.Product.Description)
	require.Equal(t, "27.00", actual.SourceQuotes.Prices[0].Amount)
	f.detail.Product.Description = strings.Repeat("x", collection.MaxDetailResponseBytes)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+"/items/"+uuid.NewString(), nil))
	require.NotEqual(t, http.StatusOK, response.Code)
}
