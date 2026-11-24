package shopify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestInstallationUsesPinnedAPIAndReturnsVerifiedShop(t *testing.T) {
	client := NewAdminClient(&http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.String() != "https://demo-shop.myshopify.com/admin/api/2026-10/graphql.json" {
			t.Fatalf("unexpected request %s %s", req.Method, req.URL)
		}
		if req.Header.Get("X-Shopify-Access-Token") != "fixture-token" {
			t.Fatal("missing Shopify token")
		}
		var request struct{ Query string }
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil || !strings.Contains(request.Query, "currentAppInstallation") || !strings.Contains(request.Query, "accessScopes") {
			t.Fatalf("invalid installation query: %v", err)
		}
		return fixtureResponse(http.StatusOK, installationFixture), nil
	})})

	got, err := client.Installation(context.Background(), "demo-shop.myshopify.com", "fixture-token", "gid://shopify/App/32")
	want := Installation{ShopID: "gid://shopify/Shop/10", Domain: "demo-shop.myshopify.com", Currency: "USD", ID: "gid://shopify/AppInstallation/21", AppID: "gid://shopify/App/32"}
	if err != nil || got != want {
		t.Fatalf("installation = %+v, %v; want %+v", got, err, want)
	}
}

func TestInstallationRejectsUnverifiedEvidence(t *testing.T) {
	tests := []struct {
		name   string
		reply  string
		status int
	}{
		{"different app", strings.Replace(installationFixture, "App/32", "App/33", 1), 200},
		{"different domain", strings.Replace(installationFixture, "demo-shop.myshopify.com", "other-shop.myshopify.com", 1), 200},
		{"read-only scope", strings.Replace(installationFixture, "write_products", "read_products", 1), 200},
		{"null installation", `{"data":{"currentAppInstallation":null}}`, 200},
		{"response with errors", strings.Replace(installationFixture, `{"data"`, `{"errors":[{"message":"fixture-token"}],"data"`, 1), 200},
		{"duplicate authority field", strings.Replace(installationFixture, `"currencyCode":"USD"`, `"currencyCode":"USD","currencyCode":"EUR"`, 1), 200},
		{"redirect", installationFixture, 302},
		{"denied", installationFixture, 403},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := NewAdminClient(&http.Client{Transport: transportFunc(func(_ *http.Request) (*http.Response, error) {
				calls++
				if calls > 1 {
					t.Fatal("retried or followed redirect")
				}
				return fixtureResponse(test.status, test.reply), nil
			})})
			got, err := client.Installation(context.Background(), "demo-shop.myshopify.com", "fixture-token", "gid://shopify/App/32")
			if !errors.Is(err, ErrInstallationUnavailable) || got != (Installation{}) || calls != 1 {
				t.Fatalf("installation = %+v, %v; calls = %d", got, err, calls)
			}
		})
	}
}

func TestInstallationRejectsInvalidShopBeforeRequest(t *testing.T) {
	client := NewAdminClient(&http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("attempted network request for invalid shop")
		return nil, nil
	})})
	for _, shop := range []string{"myshopify.com", "demo.myshopify.com.invalid", "https://demo.myshopify.com", "demo.myshopify.com:443", "other.demo.myshopify.com", "demo@merchant.myshopify.com"} {
		if _, err := client.Installation(context.Background(), shop, "fixture-token", "gid://shopify/App/32"); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("shop %q: want ErrInvalidRequest, got %v", shop, err)
		}
	}
}

func fixtureResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Location": []string{"https://other-shop.myshopify.com/"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

const installationFixture = `{"data":{"shop":{"id":"gid://shopify/Shop/10","myshopifyDomain":"demo-shop.myshopify.com","currencyCode":"USD"},"currentAppInstallation":{"id":"gid://shopify/AppInstallation/21","app":{"id":"gid://shopify/App/32"},"accessScopes":[{"handle":"read_products"},{"handle":"write_products"}]}},"extensions":{"cost":{"requestedQueryCost":4}}}`
