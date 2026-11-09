package observations

import (
	"github.com/stretchr/testify/require"
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

func TestSupplyQuantitiesSeparateMotherAndUnknownAndNeverFillMissingWithZero(t *testing.T) {
	yes, no, five, hundred, zero := true, false, int64(5), int64(100), int64(0)
	orders := []SupplyOrder{
		{ID: "regular", Mother: &no, Lines: []SupplyLine{{SKU: "sku", Ordered: &five, Delivered: &zero}}},
		{ID: "mother", Mother: &yes, Lines: []SupplyLine{{SKU: "sku", Ordered: &hundred}}},
		{ID: "child", Mother: &no, Lines: []SupplyLine{{SKU: "sku", Ordered: &five}}},
		{ID: "unknown", Lines: []SupplyLine{{SKU: "sku", Ordered: &hundred}}},
	}
	result := SummarizeSupplyOrders(orders)
	require.Equal(t, 2, result.OrderCount)
	require.Equal(t, 1, result.MotherCount)
	require.Equal(t, 1, result.UnknownTypeCount)
	require.Equal(t, "10", *result.Quantities.Ordered.Value)
	require.Nil(t, result.Quantities.Delivered.Value)
	require.Equal(t, 1, result.Quantities.Delivered.MissingLines)
	orders[2].Lines[0].Delivered = &zero
	result = SummarizeSupplyOrders(orders)
	require.Equal(t, "0", *result.Quantities.Delivered.Value)
	orders[2].Lines = append(orders[2].Lines, orders[2].Lines[0])
	result = SummarizeSupplyOrders(orders)
	require.Nil(t, result.Quantities.Ordered.Value, "duplicate SKU cannot prove a unique quantity")
}
