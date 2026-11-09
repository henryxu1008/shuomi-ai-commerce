package shein

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	o "task-processor/internal/storecenter/observations"
	"testing"
)

func TestSupplyOrdersUseExplicitBatchAndRetainUnknownWithoutPrivateOrMoneyFields(t *testing.T) {
	client := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "/open-api/order/purchase-order-infos", r.URL.Path)
		require.Equal(t, "A,B", r.URL.Query().Get("orderNos"))
		require.Equal(t, "1", r.URL.Query().Get("pageNumber"))
		require.Empty(t, r.URL.Query().Get("pageNo"))
		require.Equal(t, "1", r.URL.Query().Get("selectJitMother"))
		return observationResponse(`{"code":"0","info":{"count":3,"list":[{"orderNo":"A","type":2,"status":2,"sceneType":1,"isJitMotherName":"No","currency":"CNY","orderExtends":[{"skuCode":"sku","price":99,"orderQuantity":5,"deliveryQuantity":0}],"privateName":"secret"},{"orderNo":"A","type":2,"status":2,"sceneType":1,"isJitMotherName":"No","orderExtends":[{"skuCode":"sku","orderQuantity":5,"deliveryQuantity":0}]},{"orderNo":"B","type":2,"status":2,"sceneType":null,"isJitMotherName":"是","orderExtends":[]}]}}`), nil
	}))
	page, err := client.QueryObservedSupplyOrders(context.Background(), goodsCredential(), []string{"B", "A"})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.False(t, *page.Items[0].Mother)
	require.True(t, *page.Items[1].Mother)
	require.Equal(t, int64(0), *page.Items[0].Lines[0].Delivered)
	require.Nil(t, page.Items[0].Lines[0].Received)
	raw, err := json.Marshal(page)
	require.NoError(t, err)
	for _, value := range []string{"secret", "privateName", "price", "currency"} {
		require.NotContains(t, string(raw), value)
	}
}

func TestSupplyReceiptAmbiguityNeverBecomesConfirmedCoverage(t *testing.T) {
	for _, test := range []struct {
		name, raw string
		err       error
		count     int
		note      string
	}{
		{"foreign identity", `{"code":"0","info":{"count":1,"list":[{"orderNo":"FOREIGN"}]}}`, o.ErrConflict, 0, ""},
		{"conflicting identity", `{"code":"0","info":{"count":2,"list":[{"orderNo":"A","isJitMotherName":"No","orderExtends":[{"skuCode":"x","orderQuantity":1}]},{"orderNo":"A","isJitMotherName":"No","orderExtends":[{"skuCode":"x","orderQuantity":2}]}]}}`, nil, 0, "supply_identity_conflict"},
		{"count absent", `{"code":"0","info":{"list":[{"orderNo":"A"}]}}`, nil, 1, "supply_count_unverified"},
		{"negative quantity", `{"code":"0","info":{"count":1,"list":[{"orderNo":"A","orderExtends":[{"skuCode":"x","orderQuantity":-1}]}]}}`, o.ErrUnavailable, 0, ""},
		{"fractional quantity", `{"code":"0","info":{"count":1,"list":[{"orderNo":"A","orderExtends":[{"skuCode":"x","orderQuantity":1.1}]}]}}`, o.ErrUnavailable, 0, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) { return observationResponse(test.raw), nil }))
			page, err := client.QueryObservedSupplyOrders(context.Background(), goodsCredential(), []string{"A"})
			require.ErrorIs(t, err, test.err)
			require.Len(t, page.Items, test.count)
			if test.note != "" {
				require.Contains(t, page.Notes, test.note)
			}
		})
	}
}
