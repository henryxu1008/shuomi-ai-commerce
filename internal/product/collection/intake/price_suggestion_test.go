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

func TestPriceAmbiguityQuarantinesStockWithoutLosingValidFields(t *testing.T) {
	const notes = "黑 M，SKU BLK-M，CNY 10 或 CNY 20，尚未确认，库存明确为 0"
	raw := []byte(`{"title":"保留标题","rows":[{"attributes":{"颜色":"黑","尺码":"M"},"evidence":"` + notes + `","sku":"BLK-M","stock":0,"priceSuggestion":{"amount":10,"currency":"CNY","evidence":"CNY 10 或 CNY 20，尚未确认"}}]}`)
	out, err := Decode(raw, Input{Notes: notes})
	require.NoError(t, err)
	require.Equal(t, "保留标题", out.Title)
	require.Equal(t, "BLK-M", out.Rows[0].SKU)
	require.Nil(t, out.Rows[0].Stock)
	encoded, err := json.Marshal(out)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"stockReviewRows":[0]`)
	require.Contains(t, string(encoded), `"priceSuggestion"`)
	_, err = Decode([]byte(`{"title":"x","stockReviewRows":[0]}`), Input{Notes: notes})
	require.Equal(t, ValidationIssue{"candidate", "invalid_schema"}, err)
}

func TestStockReviewProjectionPreservesInputAndRejectsInvalidNumbers(t *testing.T) {
	c := Candidate{Rows: []Row{{Evidence: "库存未知", Stock: new(0)}, {Evidence: "库存明确为 0", Stock: new(0)}}}
	first := c.ForReview()
	require.Equal(t, new(0), c.Rows[0].Stock, "raw provider candidate must remain unchanged")
	require.Nil(t, first.Rows[0].Stock)
	require.Equal(t, new(0), first.Rows[1].Stock)
	require.Equal(t, []int{0}, first.StockReviewRows)
	require.Equal(t, first, c.ForReview(), "readback projection must be deterministic")
	for _, stock := range []string{"-1", "9007199254740992", "0.5", "\"unknown\""} {
		_, err := Decode([]byte(`{"rows":[{"attributes":{},"evidence":"库存未知","stock":`+stock+`}]}`), Input{Notes: "库存未知"})
		require.ErrorIs(t, err, ErrInvalid, stock)
	}
	_, err := Decode([]byte(`{"rows":[{"attributes":{},"evidence":"伪造库存未知","stock":0}]}`), Input{Notes: "库存未知"})
	require.Equal(t, ValidationIssue{"rows[0].evidence", "source_evidence"}, err)
}
