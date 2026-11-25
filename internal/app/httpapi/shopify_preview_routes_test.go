package httpapi

import (
	"net/http"
	"testing"

	listinghttp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing/httpapi"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/stretchr/testify/require"
)

func TestShopifyPreviewRoutesExcludeExecutionAndReconciliation(t *testing.T) {
	const base = listinghttp.ListingBasePath
	all := append(listinghttp.ListingRoutes(nil, nil),
		httproute.Descriptor{Method: http.MethodGet, Path: "/api/v1/workbench/context"},
		httproute.Descriptor{Method: http.MethodGet, Path: "/api/v1/workbench/stores"},
		httproute.Descriptor{Method: http.MethodPost, Path: "/api/v1/workbench/stores"})
	var mounted []string
	for _, route := range shopifyDraftPreviewRoutes(all) {
		mounted = append(mounted, route.Method+" "+route.Path)
		if route.Method == http.MethodPost {
			require.Equal(t, httproute.AuthPolicyVerifiedIdentity, route.AuthPolicy)
			require.Equal(t, httproute.OrganizationAccessPolicyLiveWrite, route.OrganizationAccessPolicy)
		}
	}
	require.ElementsMatch(t, []string{
		"GET /api/v1/workbench/context",
		"GET /api/v1/workbench/stores",
		"GET " + base,
		"GET " + base + "/:preparation_id",
		"GET " + base + "/:preparation_id/sources",
		"GET " + base + "/sources/:source_id",
		"GET " + base + "/:preparation_id/stages",
		"GET " + base + "/:preparation_id/operations",
		"GET " + base + "/sources/:source_id/targets/:store_id",
		"GET " + base + "/sources/:source_id/publications/:store_id",
		"POST " + base + "/shopify-draft-media-preview",
	}, mounted)
}
