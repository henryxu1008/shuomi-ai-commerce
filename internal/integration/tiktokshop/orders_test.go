package tiktokshop

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestDecodeConsumerOrdersKeepsOriginalSubtotalAndSeconds(t *testing.T) {
	raw := []byte(`{"code":0,"data":{"total_count":1,"next_page_token":"next","orders":[{"id":"order-1","create_time":1794023999,"status":"UNPAID","payment":{"currency":"USD","sub_total":"20.01","total_amount":"999","seller_discount":"8"},"recipient_address":{"name":"PRIVATE_NAME"},"line_items":[{"id":"one","sale_price":"999","quantity":3}],"packages":[{"id":"a"},{"id":"b"}]}]}}`)
	p, e := DecodeOrderPage(raw)
	require.NoError(t, e)
	require.Equal(t, 1, p.Total)
	require.Equal(t, "next", p.NextToken)
	require.Len(t, p.Items, 1)
	o := p.Items[0]
	require.Equal(t, "20.01", o.Amount.Value)
	require.Equal(t, "USD", o.Amount.Currency)
	require.Equal(t, "2026-11-07T03:59:59Z", o.Date)
	b, e := json.Marshal(o)
	require.NoError(t, e)
	require.NotContains(t, string(b), "PRIVATE_NAME")
	require.NotContains(t, string(b), "quantity")
	require.NotContains(t, string(b), "999")
	for _, bad := range []string{`{"code":1,"data":{"total_count":0,"orders":[]}}`, `{"code":0,"data":{"total_count":0}}`, strings.Replace(string(raw), `"next"`, `null`, 1), strings.Replace(string(raw), `"20.01"`, `"1e3"`, 1), strings.Replace(string(raw), `1794023999`, `1794023999000`, 1)} {
		_, e = DecodeOrderPage([]byte(bad))
		require.Error(t, e)
	}
	p, e = DecodeOrderPage([]byte(`{"code":0,"data":{"total_count":1,"orders":[{"id":"x","payment":{"sub_total":"0"}}]}}`))
	require.NoError(t, e)
	require.Empty(t, p.Items[0].Date)
	require.Equal(t, "0", p.Items[0].Amount.Value)
	require.Empty(t, p.Items[0].Amount.Currency)
}
