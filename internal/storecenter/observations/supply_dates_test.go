package observations

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func dateOrder(t *testing.T, id string, kind int, mother string, add, allocate json.RawMessage, qty *int64) SupplyOrder {
	t.Helper()
	times, e := ParseSupplyBusinessTimes(map[string]json.RawMessage{"addTime": add, "allocateTime": allocate})
	require.NoError(t, e)
	return SupplyOrder{ID: id, Type: &kind, MotherRaw: mother, Mother: SupplyMother(mother), BusinessTimes: times, Lines: []SupplyLine{{SKU: "sku-" + id, Ordered: qty}}}
}
func TestSavedSupplyDatesUseOriginalDayAndTypeSpecificField(t *testing.T) {
	q := SupplyDateRange{StartDate: "2026-11-01", EndDate: "2026-11-03"}
	five, two := int64(5), int64(2)
	var a supplyDateAccumulator
	for _, order := range []SupplyOrder{
		dateOrder(t, "urgent", 1, "No", json.RawMessage(`"2026-11-01 00:00:00"`), json.RawMessage(`"2026-12-01 00:00:00"`), &five),
		dateOrder(t, "stock", 2, "No", json.RawMessage(`"2026-12-01 00:00:00"`), json.RawMessage(`"2026-11-03T23:30:00-05:00"`), &two),
		dateOrder(t, "mother", 2, "Yes", nil, nil, &five), dateOrder(t, "unknown-mother", 1, "", nil, nil, &five),
		dateOrder(t, "missing", 1, "No", nil, nil, &five), dateOrder(t, "null", 2, "No", nil, json.RawMessage(`null`), &two),
		dateOrder(t, "bad-type", 7, "No", nil, nil, &five), dateOrder(t, "outside", 1, "No", json.RawMessage(`"2026-11-04T00:00:00Z"`), nil, &five),
	} {
		require.True(t, order.Valid())
		a.add(order, q)
	}
	r := a.result(q)
	require.Equal(t, []string{"urgent"}, r.Urgent.OrderIDs)
	require.Equal(t, "5", *r.Urgent.Ordered.Value)
	require.Equal(t, []string{"stock"}, r.Stock.OrderIDs)
	require.Equal(t, "2", *r.Stock.Ordered.Value)
	require.Equal(t, []string{"mother"}, r.MotherIDs)
	require.Equal(t, []string{"unknown-mother"}, r.UnknownMotherIDs)
	require.Equal(t, []string{"bad-type"}, r.UnknownPurchaseTypeIDs)
	require.Equal(t, []string{"outside"}, r.OutsideRangeIDs)
	require.Len(t, r.Undated, 2)
	require.Equal(t, "missing", r.Undated[0].State)
	require.Equal(t, "allocateTime", r.Undated[1].Field)
}

func TestSavedSupplyDateRangeAndUnconfirmedValues(t *testing.T) {
	for _, q := range []SupplyDateRange{{"2024-02-29", "2024-02-29"}, {"1970-01-01", "9999-12-31"}} {
		require.True(t, q.Valid())
	}
	for _, q := range []SupplyDateRange{{"2026-02-29", "2026-03-01"}, {"2026-11-03", "2026-11-01"}, {"", "2026-11-01"}, {"2026-1-01", "2026-11-01"}} {
		require.False(t, q.Valid())
	}
	q := SupplyDateRange{"2026-11-01", "2026-11-03"}
	one := int64(1)
	for _, raw := range []string{`null`, `""`, `" 2026-11-02 00:00:00 "`, `123.00`, `false`, `"2026-02-30 00:00:00"`} {
		var a supplyDateAccumulator
		a.add(dateOrder(t, "A", 1, "No", json.RawMessage(raw), nil, &one), q)
		r := a.result(q)
		require.Empty(t, r.Urgent.OrderIDs)
		require.Len(t, r.Undated, 1)
	}
	var a supplyDateAccumulator
	a.add(dateOrder(t, "missing-qty", 1, "No", json.RawMessage(`"2026-11-02 00:00:00"`), nil, nil), q)
	r := a.result(q)
	require.Nil(t, r.Urgent.Ordered.Value)
	require.Equal(t, 1, r.Urgent.Ordered.MissingLines)
	// A syntactically readable sample is not silently erased or reinterpreted.
	var epoch supplyDateAccumulator
	epoch.add(dateOrder(t, "original-1970", 1, "No", json.RawMessage(`"1970-01-01 08:00:01"`), nil, &one), SupplyDateRange{"1970-01-01", "1970-01-01"})
	require.Equal(t, []string{"original-1970"}, epoch.result(SupplyDateRange{"1970-01-01", "1970-01-01"}).Urgent.OrderIDs)
}

func TestSavedSupplyDateCannotAssignAnUnqueryableCalendarYear(t *testing.T) {
	one := int64(1)
	q := SupplyDateRange{"2026-11-01", "2026-11-03"}
	var a supplyDateAccumulator
	a.add(dateOrder(t, "year-zero", 1, "No", json.RawMessage(`"0000-01-01 00:00:00"`), nil, &one), q)
	r := a.result(q)
	require.Len(t, r.Undated, 1)
	require.Equal(t, "unusable_date", r.Undated[0].State)
	require.Empty(t, r.OutsideRangeIDs)
}
