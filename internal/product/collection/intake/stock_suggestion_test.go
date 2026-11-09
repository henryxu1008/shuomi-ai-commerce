package intake

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func stockSuggestionJSON(t *testing.T, stock any) []byte {
	t.Helper()
	row := map[string]any{"attributes": map[string]string{}, "evidence": "SKU STOCK-123", "sku": "STOCK-123"}
	if stock != nil {
		row["stockSuggestion"] = stock
	}
	raw, err := json.Marshal(map[string]any{"title": "保留标题", "rows": []any{row}})
	require.NoError(t, err)
	return raw
}

func TestStockSuggestionLeavesMeaningForHumanConfirmation(t *testing.T) {
	for _, source := range []string{"库存 0", "库存未填写", "不是库存 0", "库存 0-5", "库存 1,000", "库存 0 或 5", "库存 0，价格尚未确认", "SKU STOCK-123"} {
		t.Run(source, func(t *testing.T) {
			raw := stockSuggestionJSON(t, map[string]any{"quantity": 0, "evidence": source})
			out, err := Decode(raw, Input{Notes: "SKU STOCK-123，" + source})
			require.NoError(t, err)
			encoded, err := json.Marshal(out)
			require.NoError(t, err)
			require.JSONEq(t, string(raw), string(encoded))
			require.NotContains(t, string(encoded), `"stock":`)
			require.NotContains(t, string(encoded), "stockReviewRows")
		})
	}
}

func TestStockSuggestionRequiresExplicitSafeIntegerAndSource(t *testing.T) {
	for _, stock := range []any{
		map[string]any{"quantity": -1, "evidence": "SKU STOCK-123"},
		map[string]any{"quantity": 0.5, "evidence": "SKU STOCK-123"},
		map[string]any{"quantity": 9007199254740992, "evidence": "SKU STOCK-123"},
		map[string]any{"quantity": "0", "evidence": "SKU STOCK-123"},
		map[string]any{"quantity": nil, "evidence": "SKU STOCK-123"},
		map[string]any{"evidence": "SKU STOCK-123"},
		map[string]any{"quantity": 0, "evidence": "伪造来源"},
		map[string]any{"quantity": 0, "evidence": ""},
		map[string]any{"quantity": 0, "evidence": "SKU STOCK-123", "confirmed": true},
	} {
		_, err := Decode(stockSuggestionJSON(t, stock), Input{Notes: "SKU STOCK-123"})
		require.ErrorIs(t, err, ErrInvalid, "%v", stock)
	}
	for _, stock := range []any{nil, map[string]any{"quantity": 0, "evidence": "SKU STOCK-123"}, map[string]any{"quantity": 9007199254740991, "evidence": "SKU STOCK-123"}} {
		raw := stockSuggestionJSON(t, stock)
		out, err := Decode(raw, Input{Notes: "SKU STOCK-123"})
		require.NoError(t, err)
		encoded, err := json.Marshal(out)
		require.NoError(t, err)
		require.JSONEq(t, string(raw), string(encoded))
	}
}

func TestStockSuggestionRejectsRetiredFields(t *testing.T) {
	for _, raw := range []string{`{"rows":[{"attributes":{},"evidence":"库存 0","stock":0}]}`, `{"title":"x","stockReviewRows":[0]}`} {
		_, err := Decode([]byte(raw), Input{Notes: "库存 0"})
		require.Equal(t, ValidationIssue{"candidate", "invalid_schema"}, err)
	}
}
