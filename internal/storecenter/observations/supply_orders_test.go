package observations

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestSupplyBatchCanonicalIdentityAndBounds(t *testing.T) {
	ids, err := NormalizeSupplyOrderIDs([]string{"B", "A", "A"})
	require.NoError(t, err)
	require.Equal(t, []string{"A", "B"}, ids)
	for _, ids := range [][]string{nil, {"A,B"}, {" A"}, {"A\n"}, {""}} {
		_, err := NormalizeSupplyOrderIDs(ids)
		require.Error(t, err)
	}
}

func TestSupplyIdentityBudgetIncludesWorstCaseMissingIDs(t *testing.T) {
	ids := func(n int, prefix string) []string {
		out := []string{}
		for i := 0; i < n; i++ {
			out = append(out, prefix+fmt.Sprintf("%03d", i))
		}
		return out
	}
	ascii, err := NormalizeSupplyOrderIDs(ids(200, strings.Repeat("A", 125)))
	require.NoError(t, err)
	require.Len(t, ascii, 200)
	_, err = NormalizeSupplyOrderIDs(ids(100, strings.Repeat("采", 125)))
	require.ErrorIs(t, err, ErrInvalid, "a bounded Begin body must not admit an unfinishable missing-ID checkpoint")
	bounded, err := NormalizeSupplyOrderIDs(ids(78, strings.Repeat("采", 125)))
	require.NoError(t, err)
	worst := Checkpoint{Page: 11, Pages: 10, Incomplete: true, SupplyOrders: &SupplyProgress{OrderIDs: bounded, Next: 78, MissingIDs: bounded}, Notes: []string{}}
	for i := 0; i < 20; i++ {
		worst.Notes = append(worst.Notes, strings.Repeat("N", 64))
	}
	raw, err := json.MarshalIndent(worst, "", " ")
	require.NoError(t, err)
	require.LessOrEqual(t, len(raw), 65536)
}

func TestSupplyQuantitiesSeparateMotherAndUnknownAndNeverFillMissingWithZero(t *testing.T) {
	yes, no, five, hundred, zero := true, false, int64(5), int64(100), int64(0)
	orders := []SupplyOrder{
		{ID: "regular", Mother: &no, Lines: []SupplyLine{{SKU: "sku", Ordered: &five, Delivered: &zero}}},
		{ID: "mother", Mother: &yes, Lines: []SupplyLine{{SKU: "sku", Ordered: &hundred}}},
		{ID: "child", Mother: &no, Lines: []SupplyLine{{SKU: "sku", Ordered: &five}}},
		{ID: "unknown", Lines: []SupplyLine{{SKU: "sku", Ordered: &hundred}}},
	}
	result := sumSupplyOrdersForTest(orders)
	require.Equal(t, 2, result.OrderCount)
	require.Equal(t, 1, result.MotherCount)
	require.Equal(t, 1, result.UnknownTypeCount)
	require.Equal(t, "10", *result.Quantities.Ordered.Value)
	require.Nil(t, result.Quantities.Delivered.Value)
	require.Equal(t, 1, result.Quantities.Delivered.MissingLines)
	orders[2].Lines[0].Delivered = &zero
	result = sumSupplyOrdersForTest(orders)
	require.Equal(t, "0", *result.Quantities.Delivered.Value)
	orders[2].Lines = append(orders[2].Lines, orders[2].Lines[0])
	result = sumSupplyOrdersForTest(orders)
	require.Nil(t, result.Quantities.Ordered.Value, "duplicate SKU cannot prove a unique quantity")
}

func sumSupplyOrdersForTest(orders []SupplyOrder) SupplySummary {
	var accumulator supplySummaryAccumulator
	for _, order := range orders {
		accumulator.add(order)
	}
	return accumulator.result()
}
