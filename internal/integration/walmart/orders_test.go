package walmart

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestDecodeOrderPagePreservesDecimalAndDropsPrivateFields(t *testing.T) {
	raw := []byte(`{"list":{"meta":{"totalCount":1,"nextCursor":"?poIndex=1"},"elements":{"order":[{"purchaseOrderId":"po-1","customerOrderId":"co-1","orderDate":1794067199999,"customerEmailId":"private@example.test","orderLines":{"orderLine":[{"lineNumber":"1","orderLineQuantity":{"amount":"3"},"charges":{"charge":[{"chargeType":"PRODUCT","chargeAmount":{"currency":"USD","amount":99999999999999999999.00000001},"tax":{"taxAmount":{"amount":10}}},{"chargeType":"SHIPPING","chargeAmount":{"currency":"USD","amount":10}}]}}]}}]}}}`)
	p, err := DecodeOrderPage(raw)
	require.NoError(t, err)
	require.Len(t, p.Items, 1)
	require.Equal(t, "99999999999999999999.00000001", p.Items[0].Lines[0].Charges[0].Amount)
	require.Len(t, p.Items[0].Lines[0].Charges, 1)
	require.Equal(t, "3", p.Items[0].Lines[0].Quantity)
	saved, _ := json.Marshal(p.Items)
	require.False(t, strings.Contains(string(saved), "private"))
	require.NotContains(t, string(saved), "tax")
	p, err = DecodeOrderPage([]byte(strings.Replace(string(raw), `"chargeType":"PRODUCT"`, `"isDiscount":true,"chargeType":"PRODUCT"`, 1)))
	require.NoError(t, err)
	require.True(t, p.Items[0].Lines[0].Ambiguous)
	_, err = DecodeOrderPage([]byte(`{"list":{"meta":{"totalCount":0},"elements":{}}}`))
	require.Error(t, err)
}
