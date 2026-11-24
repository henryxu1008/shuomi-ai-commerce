package googlemerchant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/googlemerchant"
)

func TestListAccountsAndFilterAuthorizedSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing merchant access token")
		}
		switch r.URL.Path {
		case "/accounts/v1/accounts":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"accounts":      []map[string]string{{"name": "accounts/123", "accountId": "123", "accountName": "Sample merchant"}},
				"nextPageToken": "accounts-next",
			})
		case "/datasources/v1/accounts/123/dataSources":
			_, _ = w.Write([]byte(`{
				"dataSources": [
					{"name":"accounts/123/dataSources/456","dataSourceId":"456","input":"API","displayName":"Shuomi products","primaryProductDataSource":{"feedLabel":"US","contentLanguage":"en"}},
					{"name":"accounts/123/dataSources/457","dataSourceId":"457","input":"FILE","primaryProductDataSource":{}},
					{"name":"accounts/123/dataSources/458","dataSourceId":"458","input":"API","supplementalProductDataSource":{}},
					{"name":"accounts/999/dataSources/459","dataSourceId":"459","input":"API","primaryProductDataSource":{}}
				],
				"nextPageToken":"sources-next"
			}`))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := newClient(server.Client(), server.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := client.ListAccounts(context.Background(), "test-token", "")
	if err != nil || accounts.NextPageToken != "accounts-next" || len(accounts.Accounts) != 1 || accounts.Accounts[0].ID != "123" {
		t.Fatalf("accounts = %+v, err = %v", accounts, err)
	}
	sources, err := client.ListSources(context.Background(), "test-token", "123", "")
	if err != nil || sources.NextPageToken != "sources-next" ||
		!reflect.DeepEqual(sources.Sources, []domain.DataSource{{ID: "456", DisplayName: "Shuomi products", ContentLanguage: "en", FeedLabel: "US"}}) {
		t.Fatalf("sources = %+v, err = %v", sources, err)
	}
}

func TestRequireFreeListingsSourceBeforeSending(t *testing.T) {
	for _, tc := range []struct {
		name, primary string
		want          error
	}{
		{"free only", `{"feedLabel":"US","contentLanguage":"en","destinations":[{"destination":"FREE_LISTINGS","state":"ENABLED"},{"destination":"SHOPPING_ADS","state":"DISABLED"}]}`, nil},
		{"also advertising", `{"destinations":[{"destination":"FREE_LISTINGS","state":"ENABLED"},{"destination":"SHOPPING_ADS","state":"ENABLED"}]}`, domain.ErrSourceMismatch},
		{"no free destination", `{"destinations":[{"destination":"FREE_LISTINGS","state":"DISABLED"}]}`, domain.ErrSourceMismatch},
		{"different language", `{"feedLabel":"US","contentLanguage":"fr","destinations":[{"destination":"FREE_LISTINGS","state":"ENABLED"}]}`, domain.ErrSourceMismatch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/datasources/v1/accounts/123/dataSources/456" {
					t.Errorf("unexpected request %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"name":"accounts/123/dataSources/456","dataSourceId":"456","input":"API","primaryProductDataSource":` + tc.primary + `}`))
			}))
			defer server.Close()
			client, err := newClient(server.Client(), server.URL+"/")
			if err != nil {
				t.Fatal(err)
			}
			err = client.VerifySource(context.Background(), "test-token", testTarget())
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}
