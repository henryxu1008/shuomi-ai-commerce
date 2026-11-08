package temu

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTemuConsumerOrdersJoinParentAmountWithoutMultiplyingChildren(t *testing.T) {
	list := []byte(`{"success":true,"result":{"totalItemNum":1,"pageItems":[{"parentOrderMap":{"parentOrderSn":"PO-A","parentOrderTime":1794016800,"regionId":211,"parentOrderStatus":2},"orderList":[{"orderSn":"CHILD-A","quantity":7},{"orderSn":"CHILD-B","quantity":9}],"buyerName":"PRIVATE_NAME"}]}}`)
	amount := []byte(`{"success":true,"result":{"parentOrderMap":{"parentOrderSn":"PO-A","retailPriceTotal":{"amount":2525,"currency":"USD"},"basePriceTotal":{"amount":99999,"currency":"USD"}},"orderList":[{"orderSn":"CHILD-A","quantity":7,"unitBasePrice":{"amount":10000,"currency":"USD"}}]}}`)
	got, e := DecodeConsumerOrderPage(list, map[string][]byte{"PO-A": amount})
	require.NoError(t, e)
	require.Len(t, got.Items, 1)
	require.Equal(t, "25.25", got.Items[0].Amount.Value)
	require.Equal(t, "USD", got.Items[0].Amount.Currency)
	require.Equal(t, "PO-A", got.Items[0].ID)
	for _, raw := range []string{
		`{"success":true,"result":{"parentOrderMap":{"parentOrderSn":"OTHER","retailPriceTotal":{"amount":2525,"currency":"USD"}}}}`,
		`{"success":true,"result":{"parentOrderMap":{"parentOrderSn":"PO-A","retailPriceTotal":{"amount":25.25,"currency":"USD"}}}}`,
		`{"success":true,"result":{"parentOrderMap":{"parentOrderSn":"PO-A","retailPriceTotal":{"amount":-1,"currency":"USD"}}}}`,
	} {
		_, e = DecodeConsumerOrderPage(list, map[string][]byte{"PO-A": []byte(raw)})
		require.Error(t, e)
	}
	for _, raw := range []string{"", `{"success":false,"errorCode":140020008}`, `{"success":true,"result":{"parentOrderMap":{"parentOrderSn":"PO-A"}}}`, `{"success":true,"result":{"parentOrderMap":{"parentOrderSn":"PO-A","retailPriceTotal":{"amount":2525,"currency":"USD"}},"warning":["calculating"]}}`} {
		m := map[string][]byte{}
		if raw != "" {
			m["PO-A"] = []byte(raw)
		}
		got, e = DecodeConsumerOrderPage(list, m)
		require.NoError(t, e)
		require.True(t, got.Items[0].Amount == nil || got.Items[0].Amount.Value == "")
	}
}
func TestTemuConsumerAmountsPreserveZeroAndRejectUnsupportedUnits(t *testing.T) {
	list := []byte(`{"success":true,"result":{"totalItemNum":1,"pageItems":[{"parentOrderMap":{"parentOrderSn":"P","regionId":211}}]}}`)
	for _, test := range []struct {
		price string
		valid bool
	}{{`{"amount":0,"currency":"USD"}`, true}, {`{"amount":9223372036854775807,"currency":"USD"}`, true}, {`{"amount":1,"currency":"CNY"}`, false}, {`{"amount":9223372036854775808,"currency":"USD"}`, false}, {`{"amount":"25","currency":"USD"}`, false}} {
		raw := []byte(`{"success":true,"result":{"parentOrderMap":{"parentOrderSn":"P","retailPriceTotal":` + test.price + `}}}`)
		got, e := DecodeConsumerOrderPage(list, map[string][]byte{"P": raw})
		if !test.valid {
			require.Error(t, e)
			continue
		}
		require.NoError(t, e)
		require.Empty(t, got.Items[0].Date)
		if test.price == `{"amount":0,"currency":"USD"}` {
			require.Equal(t, "0.00", got.Items[0].Amount.Value)
		} else {
			require.Equal(t, "92233720368547758.07", got.Items[0].Amount.Value)
		}
	}
}
