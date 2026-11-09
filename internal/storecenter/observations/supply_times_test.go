package observations

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestSupplyTimeContractRejectsMissingOrFabricatedMetadata(t *testing.T) {
	b, err := ParseSupplyBusinessTimes(nil)
	require.NoError(t, err)
	require.True(t, b.Valid())
	require.False(t, (SupplyOrder{ID: "A"}).Valid())
	order := SupplyOrder{ID: "A", BusinessTimes: b, Lines: []SupplyLine{}}
	require.True(t, order.Valid())
	b.Timezone = "UTC+8"
	require.False(t, b.Valid())
	b.Timezone = "unconfirmed"
	b.Values["addTime"] = SupplyTimeValue{State: "local_datetime", RawType: "missing"}
	require.False(t, b.Valid())
	b, _ = ParseSupplyBusinessTimes(nil)
	delete(b.Values, "addTime")
	require.False(t, b.Valid())
}
func TestSupplyTimeUnicodeAndBounds(t *testing.T) {
	for _, raw := range []string{`"\ud800"`, `"\udc00"`, `"\ud800x"`, `"\ud800\u0041"`, `"\u0000"`, `"` + strings.Repeat("采", 86) + `"`, string([]byte{'"', 0xff, '"'})} {
		_, err := ParseSupplyBusinessTimes(map[string]json.RawMessage{"addTime": json.RawMessage(raw)})
		require.Error(t, err, raw)
	}
	for _, raw := range []string{`"\ud83d\ude00"`, `"\\ud800"`, `"�"`, `"` + strings.Repeat("x", 256) + `"`} {
		b, err := ParseSupplyBusinessTimes(map[string]json.RawMessage{"addTime": json.RawMessage(raw)})
		require.NoError(t, err, raw)
		require.True(t, b.Valid())
	}
}
