package metacatalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/metacatalog"
	"github.com/stretchr/testify/require"
)

func TestMetaSimulationSendsCreateOnlyAndChecksWhatsAppCatalog(t *testing.T) {
	var writes int
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.Equal(t, "Bearer simulated-customer", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/v27.0/123/products":
			writes++
			require.Equal(t, http.MethodPost, r.Method)
			var body struct {
				AllowUpsert *bool  `json:"allow_upsert"`
				Price       int64  `json:"price"`
				RetailerID  string `json:"retailer_id"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.NotNil(t, body.AllowUpsert)
			require.False(t, *body.AllowUpsert)
			require.EqualValues(t, 1995, body.Price)
			require.Equal(t, "sm-13aff7", body.RetailerID)
			_, _ = w.Write([]byte(`{"id":"321"}`))
		case "/v27.0/456/product_catalogs":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"data":[{"id":"123"}]}`))
		case "/v27.0/321":
			_, _ = w.Write([]byte(`{"id":"321","retailer_id":"sm-13aff7","product_catalog":{"id":"123"}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer host.Close()
	ctx := context.Background()
	client, err := NewSimulation(host.URL)
	require.NoError(t, err)
	product := domain.Product{RetailerID: "sm-13aff7", Title: "Lamp", Description: "Desk lamp", ProductURL: "https://shop.example/lamp", ImageURL: "https://shop.example/lamp.jpg", PriceCents: 1995}
	remoteID, err := client.Create(ctx, "simulated-customer", "123", product)
	require.NoError(t, err)
	require.Equal(t, "321", remoteID)
	require.Equal(t, 1, writes)
	observed, err := client.ReadProduct(ctx, "simulated-customer", "123", "321", product.RetailerID)
	require.NoError(t, err)
	require.True(t, observed)
	linked, err := client.WhatsAppLinked(ctx, "simulated-customer", "456", "123")
	require.NoError(t, err)
	require.True(t, linked)
	unrelated, err := client.WhatsAppLinked(ctx, "simulated-customer", "456", "124")
	require.NoError(t, err)
	require.False(t, unrelated)
	_, err = NewSimulation("https://graph.facebook.com")
	require.Error(t, err)
}
