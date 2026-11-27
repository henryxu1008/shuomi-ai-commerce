package httpapi

import (
	"net/http"
	"testing"

	listinghttp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing/httpapi"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	storehttp "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/httpapi"
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

func TestShopifyPairingRoutesExcludeServiceAndListingWrites(t *testing.T) {
	const stores = "/api/v1/workbench/stores"
	staged := []httproute.Descriptor{
		{Method: http.MethodGet, Path: "/api/v1/workbench/context"},
		{Method: http.MethodGet, Path: "/api/v1/workbench/store-catalog"},
		{Method: http.MethodGet, Path: stores},
		{Method: http.MethodGet, Path: stores + "/:store_id"},
		{Method: http.MethodGet, Path: stores + "/:store_id/summary"},
		{Method: http.MethodPost, Path: stores},
		{Method: http.MethodPost, Path: storehttp.ShopifyPairingPath, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite},
	}
	all := append(append([]httproute.Descriptor{}, staged...),
		httproute.Descriptor{Method: http.MethodPost, Path: stores + "/:store_id/activate"},
		httproute.Descriptor{Method: http.MethodPost, Path: "/api/v1/platform/shopify/installation"},
		httproute.Descriptor{Method: http.MethodPost, Path: listinghttp.ListingBasePath + "/shopify-draft-media-confirm"},
		httproute.Descriptor{Method: http.MethodDelete, Path: stores + "/:store_id"},
		httproute.Descriptor{Method: http.MethodGet, Path: listinghttp.ListingBasePath})
	require.Equal(t, staged, shopifyPairingTrialRoutes(all))
}

func TestShopifyMediaTrialRoutesExposeOnlyFixedMediaAndNotices(t *testing.T) {
	const base = listinghttp.ListingBasePath
	allowed := []httproute.Descriptor{
		{Method: http.MethodGet, Path: "/api/v1/workbench/context"},
		{Method: http.MethodGet, Path: "/api/v1/workbench/stores/:store_id"},
		{Method: http.MethodPost, Path: base + "/shopify-draft-media-preview"},
		{Method: http.MethodPost, Path: base + "/shopify-draft-media-confirm"},
		{Method: http.MethodPost, Path: storehttp.ShopifyUninstalledPath},
		{Method: http.MethodPost, Path: storehttp.ShopifyShopRedactPath},
	}
	all := append(append([]httproute.Descriptor{}, allowed...),
		httproute.Descriptor{Method: http.MethodPost, Path: "/api/v1/workbench/stores/:store_id/activate"},
		httproute.Descriptor{Method: http.MethodPost, Path: "/api/v1/platform/shopify/installation"},
		httproute.Descriptor{Method: http.MethodPost, Path: base + "/upload-plans"},
		httproute.Descriptor{Method: http.MethodGet, Path: base + "/sources/:source_id/publications/:store_id"},
		httproute.Descriptor{Method: http.MethodDelete, Path: "/api/v1/workbench/stores/:store_id"})
	require.Equal(t, allowed, shopifyMediaTrialRoutes(all))
}
