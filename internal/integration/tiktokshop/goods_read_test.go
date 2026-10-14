package tiktokshop

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	marketplace "task-processor/internal/marketplace/tiktokshop"
)

func TestCategoryQueryUsesAdmittedMethodUSVersionAndProtectedShop(t *testing.T) {
	for _, method := range []marketplace.ListingMethod{marketplace.GlobalPublishing, marketplace.LocalReplication} {
		client := newFixtureClient(t, func(request *http.Request) (*http.Response, error) {
			path := "/product/202309/categories"
			if method == marketplace.GlobalPublishing {
				path = "/product/202309/global_categories"
			}
			if request.Method != "GET" || request.URL.Scheme+"://"+request.URL.Host != apiOrigin || request.URL.Path != path || request.URL.Query().Get("category_version") != "v2" || request.URL.Query().Get("locale") != "en-US" || request.Header.Get("x-tts-access-token") != "private-token" || request.URL.Query().Has("access_token") {
				t.Fatal("wrong resource request")
			}
			if method == marketplace.LocalReplication && request.URL.Query().Get("shop_cipher") != "protected-shop" || method == marketplace.GlobalPublishing && request.URL.Query().Has("shop_cipher") {
				t.Fatal("shop context mixed with global query")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"categories":[{"id":"600001","parent_id":"0","local_name":"Category","is_leaf":true,"permission_statuses":["AVAILABLE"]}]}}`))}, nil
		})
		categories, err := client.QueryCategories(context.Background(), "private-token", "protected-shop", method)
		if err != nil || len(categories) != 1 || categories[0].ID != "600001" {
			t.Fatalf("category query: %v", err)
		}
	}
}

func TestProductStatusQueryUsesProtectedShopAndOriginalProduct(t *testing.T) {
	calls := 0
	client := newFixtureClient(t, func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Method != "GET" || request.URL.Path != "/product/202309/products/local-1" || request.URL.Query().Get("shop_cipher") != "protected-shop" || request.URL.Query().Get("locale") != "en-US" || request.Header.Get("x-tts-access-token") != "private-token" || request.URL.Query().Has("return_draft_version") || request.URL.Query().Has("return_under_review_version") {
			t.Fatal("wrong product status request")
		}
		query := request.URL.Query()
		sign := query.Get("sign")
		query.Del("sign")
		want, err := signature(request.URL.Path, query, nil, "application/json", "fixture-secret-never-log")
		if err != nil || sign != want {
			t.Fatal("request signature does not bind product/shop")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"id":"local-1","status":"ACTIVATE","audit":{"status":"APPROVED"},"skus":[{"id":"sku-1"}]}}`))}, nil
	})
	value, err := client.QueryProductStatus(context.Background(), "private-token", "protected-shop", marketplace.ProductStatusExpectation{ProductID: "local-1", LocalSKUIDs: []string{"sku-1"}})
	if err != nil || value.Status != "ACTIVATE" || calls != 1 {
		t.Fatalf("status query: %v", err)
	}
	_, err = client.QueryProductStatus(context.Background(), "private-token", "protected-shop", marketplace.ProductStatusExpectation{ProductID: "../other"})
	if err == nil || calls != 1 {
		t.Fatal("unsafe path dispatched")
	}
}
