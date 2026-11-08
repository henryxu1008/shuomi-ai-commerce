package walmart

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrderProductChargesAreExactLineFeesNotQuantityMultiplication(t *testing.T) {
	o := Order{ID: "po-1", Site: "walmart-us", Date: "2026-11-07T15:59:59.999Z", Lines: []OrderLine{
		{Number: "1", Quantity: "3", Charges: []OrderCharge{{Currency: "USD", Amount: "0.10"}}},
		{Number: "2", Quantity: "2", Charges: []OrderCharge{{Currency: "USD", Amount: "0.20"}}},
	}}
	amount, currency, missingAmount, missingCurrency := o.ProductAmount()
	require.Equal(t, "0.30", amount)
	require.Equal(t, "USD", currency)
	require.False(t, missingAmount)
	require.False(t, missingCurrency)
	o.Lines[1].Ambiguous = true
	_, _, missingAmount, _ = o.ProductAmount()
	require.True(t, missingAmount)
	o.Lines[1].Ambiguous = false
	o.Lines[1].Charges[0].Currency = "CNY"
	_, _, _, missingCurrency = o.ProductAmount()
	require.True(t, missingCurrency)
}
func TestOrderProjectionKeepsMissingDataAndRejectsInvalidMembers(t *testing.T) {
	o := Order{ID: "po", Site: "walmart-us", Lines: []OrderLine{{Number: "1"}}}
	require.True(t, o.Valid())
	_, _, missing, currency := o.ProductAmount()
	require.True(t, missing)
	require.True(t, currency)
	o.Lines = append(o.Lines, o.Lines[0])
	require.False(t, o.Valid())
	o.Lines = []OrderLine{{Number: "1", Charges: []OrderCharge{{Currency: "USD", Amount: "1e3"}}}}
	require.False(t, o.Valid())
}
