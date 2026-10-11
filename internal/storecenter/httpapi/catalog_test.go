package httpapi

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestMetadataRejectsMissingExplicitIntentFields(t *testing.T) {
	for _, body := range []string{
		`{"action":"create_group","name":"A"}`,
		`{"action":"set_profile","name":"","groupId":null,"groupVersion":0,"stores":[{"storeId":"11111111-1111-4111-8111-111111111111"}],"responsibleMemberId":null,"lowStockThreshold":null}`,
		`{"action":"set_profile","name":"","groupId":null,"groupVersion":0,"stores":[],"responsibleMemberId":null,"lowStockThreshold":null,"lowStockThreshold":5}`,
	} {
		t.Run(body, func(t *testing.T) {
			h := mustHandler(t, newStoreServiceStub(t))
			router := gin.New()
			router.Use(gin.Recovery())
			router.POST("/metadata", h.MetadataCommand)
			result := serve(t, router, validIdentity(), http.MethodPost, "/metadata", body, map[string][]string{"Idempotency-Key": {testCreateKey}})
			require.Equal(t, http.StatusBadRequest, result.Code, result.Body.String())
		})
	}
}

func TestStoreCenterReadsRejectBodiesBeforeQueryingDependencies(t *testing.T) {
	h := mustHandler(t, newStoreServiceStub(t))
	for _, route := range []struct {
		path    string
		handler gin.HandlerFunc
	}{
		{"/store-catalog", h.catalogList},
		{"/groups", h.Groups},
		{"/stores/:store_id/profile", h.Profile},
		{"/metadata/:operation_id", h.MetadataReceipt},
		{"/stores/:store_id/summary", h.StoreSummary},
	} {
		t.Run(route.path, func(t *testing.T) {
			router := gin.New()
			router.Use(gin.Recovery())
			router.GET(route.path, route.handler)
			result := serve(t, router, validIdentity(), http.MethodGet, route.path, `{"unread":true}`, nil)
			require.Equal(t, http.StatusBadRequest, result.Code, result.Body.String())
		})
	}
}
