package intake

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func suggestionJSON(t *testing.T, price any) []byte {
	t.Helper()
	row := map[string]any{"attributes": map[string]string{"颜色": "黑", "尺码": "M"}, "evidence": "黑 M", "sku": ""}
	if price != nil {
		row["priceSuggestion"] = price
	}
	raw, err := json.Marshal(map[string]any{"title": "保留有效标题", "rows": []any{row}})
	require.NoError(t, err)
	return raw
}

func TestPriceSuggestionKeepsSemanticsForHumanReview(t *testing.T) {
	for _, notes := range []string{"黑 M，CNY 10", "黑 M，CNY 10-20", "黑 M，USD 1,299.00", "黑 M，不是 CNY 10，价格待定", "黑 M，CNY 10 或 CNY 20，尚未确认", "黑 M，价格10元，运费5元，PVC材料"} {
		t.Run(notes, func(t *testing.T) {
			raw := suggestionJSON(t, map[string]any{"amount": 10, "currency": "CNY", "evidence": notes})
			out, err := Decode(raw, Input{Notes: notes})
			require.NoError(t, err)
			require.Equal(t, "保留有效标题", out.Title)
			encoded, err := json.Marshal(out)
			require.NoError(t, err)
			require.Contains(t, string(encoded), `"priceSuggestion"`)
			require.NotContains(t, string(encoded), `"price":`)
			require.NotContains(t, string(encoded), `"confirmed"`)
		})
	}
}

func TestPriceSuggestionDeterministicConstraints(t *testing.T) {
	for _, price := range []any{
		map[string]any{"amount": -1, "evidence": "黑 M"},
		map[string]any{"amount": 10, "evidence": "伪造来源"},
		map[string]any{"amount": 10, "currency": "cny", "evidence": "黑 M"},
		map[string]any{"amount": 10, "currency": "CN", "evidence": "黑 M"},
		map[string]any{"currency": "CNY", "evidence": "黑 M"},
		map[string]any{"amount": nil, "evidence": "黑 M"},
		map[string]any{"amount": 10, "evidence": "黑 M", "confirmed": true},
	} {
		_, err := Decode(suggestionJSON(t, price), Input{Notes: "黑 M"})
		require.ErrorIs(t, err, ErrInvalid, "%v", price)
	}
	for _, price := range []any{nil, map[string]any{"amount": 0, "evidence": "黑 M"}} {
		out, err := Decode(suggestionJSON(t, price), Input{Notes: "黑 M"})
		require.NoError(t, err)
		encoded, err := json.Marshal(out)
		require.NoError(t, err)
		if price == nil {
			require.NotContains(t, string(encoded), "priceSuggestion")
		} else {
			require.Contains(t, string(encoded), `"amount":0`)
		}
	}
}

func TestPriceSuggestionRejectsOldFactFields(t *testing.T) {
	_, err := Decode([]byte(`{"rows":[{"attributes":{"颜色":"黑"},"evidence":"黑 CNY 0","price":0,"currency":"CNY"}]}`), Input{Notes: "黑 CNY 0"})
	require.Equal(t, ValidationIssue{"candidate", "invalid_schema"}, err)
}

func TestPriceAmbiguityKeepsStockSuggestionAndOtherValidFields(t *testing.T) {
	const notes = "黑 M，SKU BLK-M，CNY 10 或 CNY 20，尚未确认，库存明确为 0"
	raw := []byte(`{"title":"保留标题","rows":[{"attributes":{"颜色":"黑","尺码":"M"},"evidence":"` + notes + `","sku":"BLK-M","stockSuggestion":{"quantity":0,"evidence":"库存明确为 0"},"priceSuggestion":{"amount":10,"currency":"CNY","evidence":"CNY 10 或 CNY 20，尚未确认"}}]}`)
	out, err := Decode(raw, Input{Notes: notes})
	require.NoError(t, err)
	require.Equal(t, "保留标题", out.Title)
	require.Equal(t, "BLK-M", out.Rows[0].SKU)
	require.Equal(t, new(int64(0)), out.Rows[0].StockSuggestion.Quantity)
	encoded, err := json.Marshal(out)
	require.NoError(t, err)
	require.JSONEq(t, string(raw), string(encoded))
}
