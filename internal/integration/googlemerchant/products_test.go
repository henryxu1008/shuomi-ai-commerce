package googlemerchant

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/googlemerchant"
)

const testPrefix = "sm-0123456789abcdef0123-"
const testOfferID = testPrefix + "0123456789abcdef01234567"

func testTarget() domain.Target {
	return domain.Target{AccountID: "123", DataSourceID: "456", ContentLanguage: "en", FeedLabel: "US", OfferPrefix: testPrefix}
}

func testOffer() domain.Offer {
	return domain.Offer{
		ID: testOfferID, Title: "Desk organizer", Description: "Bamboo desk organizer",
		Link: "https://example.com/organizers/oak", ImageLink: "https://example.com/images/oak.jpg",
		Currency: "USD", AmountMicros: 23990000, Availability: "IN_STOCK", Condition: "NEW",
	}
}

func testProductResource(collection string) string {
	return "accounts/123/" + collection + "/en~US~" + testOfferID
}

func TestInsertReadsExistingIdentityAndSubmitsOnlyFreeListings(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("wrong authorization header %q", r.Header.Get("Authorization"))
		}
		if testWriteSource(w, r, true) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			want := "/products/v1/accounts/123/products/" + base64.RawURLEncoding.EncodeToString([]byte("en~US~"+testOfferID))
			if r.URL.Path != want {
				t.Errorf("product path = %q, want %q", r.URL.Path, want)
			}
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPost:
			if r.URL.Path != "/products/v1/accounts/123/productInputs:insert" || r.URL.Query().Get("dataSource") != "accounts/123/dataSources/456" {
				t.Errorf("wrong insert destination: %s", r.URL)
			}
			var body struct {
				OfferID, ContentLanguage, FeedLabel string
				ProductAttributes                   struct {
					IncludedDestinations, ExcludedDestinations []string
					Price                                      struct {
						AmountMicros string
						CurrencyCode string
					}
				}
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode request: %v", err)
			}
			if body.OfferID != testOfferID || body.ContentLanguage != "en" || body.FeedLabel != "US" {
				t.Errorf("wrong product identity: %+v", body)
			}
			attributes := body.ProductAttributes
			if !reflect.DeepEqual(attributes.IncludedDestinations, []string{"FREE_LISTINGS"}) {
				t.Errorf("unexpected inclusions: %v", attributes.IncludedDestinations)
			}
			for _, excluded := range []string{"SHOPPING_ADS", "DISPLAY_ADS", "LOCAL_INVENTORY_ADS", "VEHICLE_ADS", "RENTAL_ADS"} {
				if !strings.Contains(strings.Join(attributes.ExcludedDestinations, ","), excluded) {
					t.Errorf("missing excluded destination %s", excluded)
				}
			}
			if attributes.Price.AmountMicros != "23990000" || attributes.Price.CurrencyCode != "USD" {
				t.Errorf("wrong price: %+v", attributes.Price)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"name": testProductResource("productInputs"), "product": testProductResource("products"),
				"offerId": testOfferID, "feedLabel": "US", "contentLanguage": "en",
			})
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	client, err := newClient(server.Client(), server.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := client.InsertNew(context.Background(), "test-token", testTarget(), testOffer())
	if err != nil || receipt.Product != testProductResource("products") {
		t.Fatalf("receipt = %+v, error = %v", receipt, err)
	}
	if !reflect.DeepEqual(methods, []string{"GET /datasources/v1/accounts/123/dataSources/456", "GET /products/v1/accounts/123/products/" + base64.RawURLEncoding.EncodeToString([]byte("en~US~"+testOfferID)), "POST /products/v1/accounts/123/productInputs:insert"}) {
		t.Fatalf("methods = %v", methods)
	}
}

func TestInsertDoesNotWriteWhenOfferExistsOrProbeIsUnqualified(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"other data source", 200, `{"name":"` + testProductResource("products") + `","dataSource":"accounts/123/dataSources/987"}`, domain.ErrOfferExists},
		{"same data source", 200, `{"name":"` + testProductResource("products") + `","dataSource":"accounts/123/dataSources/456"}`, domain.ErrOfferExists},
		{"unqualified response", 200, `{"name":"accounts/999/products/en~US~elsewhere"}`, domain.ErrUnavailable},
		{"revoked", 403, `{"error":{"message":"sensitive provider diagnostic"}}`, domain.ErrUnauthorized},
		{"upstream unavailable", 503, `{}`, domain.ErrUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if testWriteSource(w, r, true) {
					return
				}
				if r.Method != http.MethodGet {
					t.Errorf("unexpected write to %s", r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client, err := newClient(server.Client(), server.URL+"/")
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.InsertNew(context.Background(), "test-token", testTarget(), testOffer())
			if !errors.Is(err, tc.want) || strings.Contains(err.Error(), "sensitive") {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestInsertLostResponseIsUnknownWithoutResend(t *testing.T) {
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if testWriteSource(w, r, true) {
			return
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writes++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"product":`))
	}))
	defer server.Close()
	client, err := newClient(server.Client(), server.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.InsertNew(context.Background(), "test-token", testTarget(), testOffer())
	if !errors.Is(err, domain.ErrOutcomeUnknown) || writes != 1 {
		t.Fatalf("error = %v, writes = %d", err, writes)
	}
}

func TestGetProductReportsProcessingAndDetectsUnintendedDestinations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": testProductResource("products"), "dataSource": "accounts/123/dataSources/456",
			"productStatus": map[string]any{"destinationStatuses": []map[string]any{
				{"reportingContext": "FREE_LISTINGS", "pendingCountries": []string{"US"}, "approvedCountries": []string{"CA"}},
				{"reportingContext": "SHOPPING_ADS", "approvedCountries": []string{"US"}},
			}},
		})
	}))
	defer server.Close()
	client, err := newClient(server.Client(), server.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	view, err := client.ReadProduct(context.Background(), "test-token", testTarget(), testOfferID)
	if err != nil || !view.UnexpectedDestination || !reflect.DeepEqual(view.FreeListings.Pending, []string{"US"}) || !reflect.DeepEqual(view.FreeListings.Approved, []string{"CA"}) {
		t.Fatalf("view = %+v, error = %v", view, err)
	}
}

func TestInsertStopsBeforeProductRequestsWhenSourceAllowsAdvertising(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !testWriteSource(w, r, false) {
			t.Errorf("unexpected product request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := newClient(server.Client(), server.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.InsertNew(context.Background(), "test-token", testTarget(), testOffer())
	if !errors.Is(err, domain.ErrSourceMismatch) {
		t.Fatalf("error = %v, want source mismatch", err)
	}
}

func testWriteSource(w http.ResponseWriter, r *http.Request, freeOnly bool) bool {
	if r.URL.Path != "/datasources/v1/accounts/123/dataSources/456" {
		return false
	}
	adState := "DISABLED"
	if !freeOnly {
		adState = "ENABLED"
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name": "accounts/123/dataSources/456", "dataSourceId": "456", "input": "API",
		"primaryProductDataSource": map[string]any{"destinations": []map[string]string{
			{"destination": "FREE_LISTINGS", "state": "ENABLED"}, {"destination": "SHOPPING_ADS", "state": adState},
		}},
	})
	return true
}
