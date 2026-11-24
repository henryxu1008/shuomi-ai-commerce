package shopify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

const sourceDefinitionFixture = `{"data":{"metafieldDefinition":{"type":{"name":"single_line_text_field"},"capabilities":{"uniqueValues":{"enabled":true}},"access":{"storefront":"NONE"}}}}`

const fixtureSourceKey = "942d8383cde574aacd35cc667d0da6248521414e9f256a6c4502e72d8f037dc0"

func TestDraftWriteCreatesAndCorrelatesOnlyDraft(t *testing.T) {
	ctx := context.Background()
	var requests int
	client := NewAdminClient(&http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		var document struct {
			Query     string                     `json:"query"`
			Variables map[string]json.RawMessage `json:"variables"`
		}
		if req.Header.Get("X-Shopify-Access-Token") != "fixture-token" || json.NewDecoder(req.Body).Decode(&document) != nil {
			t.Fatal("expected verified token and valid request body")
		}
		if requests == 1 {
			return fixtureResponse(http.StatusOK, installationFixture), nil
		}
		if requests == 2 && strings.Contains(document.Query, "metafieldDefinition") {
			return fixtureResponse(http.StatusOK, sourceDefinitionFixture), nil
		}
		if requests != 3 || strings.Contains(document.Query, "identifier:") || !strings.Contains(document.Query, "synchronous: true") {
			t.Fatalf("expected one create-only mutation, got %s", document.Query)
		}
		var input struct {
			Status     string `json:"status"`
			Metafields []struct {
				Value string `json:"value"`
			} `json:"metafields"`
			Variants []struct {
				Price string `json:"price"`
				SKU   string `json:"sku"`
			} `json:"variants"`
		}
		if err := json.Unmarshal(document.Variables["input"], &input); err != nil || input.Status != "DRAFT" ||
			len(input.Metafields) != 1 || input.Metafields[0].Value != fixtureSourceKey ||
			len(input.Variants) != 1 || input.Variants[0].Price != "15.90" || input.Variants[0].SKU != "SAMPLE-GREEN" {
			t.Fatalf("unexpected draft input: %+v, %v", input, err)
		}
		return fixtureResponse(http.StatusOK, `{"data":{"productSet":{"product":`+productFixture("DRAFT", fixtureSourceKey)+`,"userErrors":[]}}}`), nil
	})})
	writer, err := client.VerifyDraftWrite(ctx, productInstallation(), "fixture-token")
	if err != nil {
		t.Fatal(err)
	}
	got, err := writer.Create(ctx, productDraft())
	if err != nil || got.ID != "gid://shopify/Product/55" || got.Status != "DRAFT" || got.Media[0] != "PROCESSING" {
		t.Fatalf("unexpected create result %+v, %v", got, err)
	}
	if _, err := writer.Create(ctx, productDraft()); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("reusing the same write handle: %v", err)
	}
}

func TestDraftWriteRefusesDifferentInstallation(t *testing.T) {
	calls := 0
	client := makeProductClient(t, &calls, "", http.StatusOK)
	expected := productInstallation()
	expected.ID = "gid://shopify/AppInstallation/22"
	if _, err := client.VerifyDraftWrite(context.Background(), expected, "fixture-token"); !errors.Is(err, ErrInstallationUnavailable) || calls != 1 {
		t.Fatalf("installation mismatch was accepted: %v (requests %d)", err, calls)
	}
}

func TestDraftWriteTreatsAmbiguousRemoteResultsAsUnknown(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		status int
	}{
		{"product keys differ", `{"data":{"productSet":{"product":` + productFixture("DRAFT", strings.Repeat("f", 64)) + `,"userErrors":[]}}}`, 200},
		{"remote status changed", `{"data":{"productSet":{"product":` + productFixture("ACTIVE", fixtureSourceKey) + `,"userErrors":[]}}}`, 200},
		{"partial provider result", `{"data":{"productSet":{"product":` + productFixture("DRAFT", fixtureSourceKey) + `,"userErrors":[{"code":"ERROR","message":"sensitive body"}]}}}`, 200},
		{"unclassified rejection", `{"data":{"productSet":{"product":null,"userErrors":[{"code":"INVALID","message":"sensitive body"}]}}}`, 200},
		{"upstream timeout", "", 504},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			calls := 0
			client := makeProductClient(t, &calls, tc.body, tc.status)
			writer, err := client.VerifyDraftWrite(ctx, productInstallation(), "fixture-token")
			if err != nil {
				t.Fatal(err)
			}
			got, err := writer.Create(ctx, productDraft())
			if got.ID != "" || !errors.Is(err, ErrDraftOutcomeUnknown) || calls != 3 {
				t.Fatalf("got %+v, %v; calls = %d", got, err, calls)
			}
		})
	}
}

func TestDraftReadbackDoesNotTreatAnEmptyLookupAsDefinitive(t *testing.T) {
	ctx := context.Background()
	for _, body := range []string{
		`{"data":{"productByIdentifier":null}}`,
		`{"data":{"productByIdentifier":` + productFixture("ACTIVE", strings.Repeat("f", 64)) + `}}`,
	} {
		calls := 0
		client := makeProductClient(t, &calls, body, 200)
		got, err := client.FindDraftBySource(ctx, productInstallation(), "fixture-token", fixtureSourceKey)
		if got.ID != "" || !errors.Is(err, ErrDraftUnverified) || calls != 2 {
			t.Fatalf("lookup result %+v, %v; calls = %d", got, err, calls)
		}
	}
	calls := 0
	client := makeProductClient(t, &calls, `{"data":{"productByIdentifier":`+productFixture("ACTIVE", fixtureSourceKey)+`}}`, 200)
	got, err := client.FindDraftBySource(ctx, productInstallation(), "fixture-token", fixtureSourceKey)
	if err != nil || got.Status != "ACTIVE" || got.ID != "gid://shopify/Product/55" {
		t.Fatalf("lookup result %+v, %v", got, err)
	}
}

func TestDraftReadbackAcceptsKnownProductAfterMetafieldRemoval(t *testing.T) {
	calls := 0
	client := makeProductClient(t, &calls, `{"data":{"product":{"id":"gid://shopify/Product/55","status":"ACTIVE","metafield":null}}}`, 200)
	expected := productInstallation()
	expected.Currency = "EUR"
	got, err := client.FindProductByID(context.Background(), expected, "fixture-token", "gid://shopify/Product/55")
	if err != nil || got.Status != "ACTIVE" || got.ID != "gid://shopify/Product/55" || calls != 2 {
		t.Fatalf("lookup result %+v, %v; calls = %d", got, err, calls)
	}
}

func makeProductClient(t *testing.T, calls *int, body string, status int) *AdminClient {
	t.Helper()
	return NewAdminClient(&http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		*calls = *calls + 1
		if *calls == 1 {
			return fixtureResponse(http.StatusOK, installationFixture), nil
		}
		var query struct{ Query string }
		if json.NewDecoder(req.Body).Decode(&query) != nil || *calls > 3 {
			t.Fatal("unexpected Shopify request")
		}
		if *calls == 2 && strings.Contains(query.Query, "metafieldDefinition") {
			return fixtureResponse(http.StatusOK, sourceDefinitionFixture), nil
		}
		return fixtureResponse(status, body), nil
	})})
}

func productInstallation() Installation {
	return Installation{ShopID: "gid://shopify/Shop/10", Domain: "demo-shop.myshopify.com", Currency: "USD",
		ID: "gid://shopify/AppInstallation/21", AppID: "gid://shopify/App/32"}
}

func productDraft() Draft {
	return Draft{
		SourceKey: fixtureSourceKey, Title: "Sample", Currency: "USD",
		Options:  []DraftOption{{Name: "Color", Values: []string{"Green"}}},
		Variants: []DraftVariant{{SKU: "SAMPLE-GREEN", Price: "15.90", Values: []string{"Green"}}},
	}
}

func productFixture(status, key string) string {
	return `{"id":"gid://shopify/Product/55","status":"` + status +
		`","metafield":{"value":"` + key + `"},"media":{"nodes":[{"status":"PROCESSING"}]}}`
}

func TestDraftWriteRefusesMissingUniquePrivateSourceDefinition(t *testing.T) {
	for _, body := range []string{
		`{"data":{"metafieldDefinition":null}}`,
		`{"data":{"metafieldDefinition":{"type":{"name":"single_line_text_field"},"capabilities":{"uniqueValues":{"enabled":false}},"access":{"storefront":"NONE"}}}}`,
		`{"data":{"metafieldDefinition":{"type":{"name":"single_line_text_field"},"capabilities":{"uniqueValues":{"enabled":true}},"access":{"storefront":"PUBLIC_READ"}}}}`,
	} {
		var calls int
		client := NewAdminClient(&http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return fixtureResponse(http.StatusOK, installationFixture), nil
			}
			return fixtureResponse(http.StatusOK, body), nil
		})})
		writer, err := client.VerifyDraftWrite(context.Background(), productInstallation(), "fixture-token")
		if writer != nil || !errors.Is(err, ErrInstallationUnavailable) || calls != 2 {
			t.Fatalf("ineligible source definition returned a write handle: %v, requests %d", err, calls)
		}
	}
}

func TestProductAdminURLRequiresCanonicalShopAndProduct(t *testing.T) {
	const shop = "fixture-shop.myshopify.com"
	const product = "gid://shopify/Product/12345"
	if got := ProductAdminURL(shop, product); got != "https://fixture-shop.myshopify.com/admin/products/12345" {
		t.Fatalf("unexpected admin destination %q", got)
	}
	for _, input := range [][2]string{
		{"fixture-shop.myshopify.com.evil.example", product},
		{"fixture-shop.myshopify.com@evil.example", product},
		{shop, "gid://shopify/ProductVariant/12345"},
		{shop, "gid://shopify/Product/0"},
		{shop, "gid://shopify/Product/12345?next=elsewhere"},
		{shop, "gid://shopify/Product/18446744073709551616"},
	} {
		if got := ProductAdminURL(input[0], input[1]); got != "" {
			t.Fatalf("unsafe admin destination %q", got)
		}
	}
}
