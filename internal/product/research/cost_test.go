package research

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func costFixture() CostInput {
	basis := CostBasis{Source: "synthetic cost sheet", AsOf: "2026-11-05"}
	amount := func(v string) CostValue { return CostValue{Kind: "value", Value: v} }
	na := CostValue{Kind: "not_applicable", Reason: "not incurred in this synthetic scenario"}
	outcome := func(logistics, rate string) CostOutcome {
		return CostOutcome{Basis: basis, Costs: map[string]CostValue{
			"procurement": amount("8"), "logistics": amount(logistics), "platformFixed": amount("0"), "platformRate": amount(rate),
			"taxFixed": na, "taxRate": na, "advertising": amount("3"), "exchangeCost": na,
		}}
	}
	return CostInput{Channel: "US", Fulfillment: "merchant shipped", Specification: "blue single item", Quantity: "20", Price: "30", RefundRate: "5",
		SalesBasis: basis, Rule: "fixed_percent", PurchaseCurrency: "USD", Completed: outcome("6", "10"), Refunded: outcome("8", "0")}
}

func TestCostOriginalVectorsAndNormalization(t *testing.T) {
	in := costFixture()
	normal, out := CalculateCost(in)
	require.Equal(t, "COMPUTABLE", out.Status)
	require.Equal(t, "8.55", out.Contribution)
	require.Equal(t, "20.50", out.Gross)
	require.Equal(t, "positive", out.ContributionSign)
	require.Equal(t, "20.00", *out.BreakEven)
	require.Equal(t, "10.00", *out.CompletedContribution)
	require.Equal(t, "-19.00", *out.RefundedContribution)
	require.Equal(t, "30.00", normal.Price)
	require.Equal(t, "8.00", normal.Completed.Costs["procurement"].Value)
	require.Equal(t, "8", in.Completed.Costs["procurement"].Value, "normalization never mutates caller input")
	in.Price = "19.99"
	_, out = CalculateCost(in)
	require.Equal(t, "-0.01", out.Contribution)
	require.Equal(t, "negative", out.ContributionSign)
	in.PurchaseCurrency, in.ExchangeRate, in.ExchangeBasis = "CNY", "0.200000", in.SalesBasis
	in.Completed.Costs["procurement"], in.Refunded.Costs["procurement"] = CostValue{Kind: "value", Value: "40"}, CostValue{Kind: "value", Value: "40"}
	in.Price = "30"
	_, out = CalculateCost(in)
	require.Equal(t, "8.55", out.Contribution)
}

func TestCostMissingZeroInvalidAndUnsupported(t *testing.T) {
	in := costFixture()
	in.Completed.Costs["procurement"] = CostValue{Kind: "value"}
	_, out := CalculateCost(in)
	require.Equal(t, "INSUFFICIENT_INPUT", out.Status)
	require.Contains(t, out.Fields, "completed.procurement")
	in.Completed.Costs["procurement"] = CostValue{Kind: "value", Value: "0"}
	_, out = CalculateCost(in)
	require.Equal(t, "COMPUTABLE", out.Status)
	for _, change := range []func(*CostInput){
		func(c *CostInput) { c.Price = "3e1" }, func(c *CostInput) { c.Quantity = "1.5" },
		func(c *CostInput) { c.SalesBasis.AsOf = "2026-02-29" }, func(c *CostInput) { c.RefundRate = "100.01" },
		func(c *CostInput) { c.PurchaseCurrency = "EUR" }, func(c *CostInput) {
			c.Refunded.Costs["procurement"] = CostValue{Kind: "value", Value: "-1"}
			c.RefundRate = "0"
		},
	} {
		c := costFixture()
		change(&c)
		_, out := CalculateCost(c)
		require.Equal(t, "INVALID_INPUT", out.Status)
	}
	in = costFixture()
	in.Rule = "unsupported"
	_, out = CalculateCost(in)
	require.Equal(t, "UNSUPPORTED_COST_RULE", out.Status)
	in.Price = "1e10"
	_, out = CalculateCost(in)
	require.Equal(t, "INVALID_INPUT", out.Status, "unsupported cannot conceal an explicit illegal input")
}

func TestCostExactSignPriceGridAndHalfCents(t *testing.T) {
	in := costFixture()
	in.RefundRate, in.Price = "0", "0.01"
	for key := range in.Completed.Costs {
		in.Completed.Costs[key] = CostValue{Kind: "not_applicable", Reason: "not incurred"}
	}
	in.Completed.Costs["procurement"] = CostValue{Kind: "value", Value: "0.01"}
	in.Completed.Costs["platformRate"] = CostValue{Kind: "value", Value: "0.01"}
	_, out := CalculateCost(in)
	require.Equal(t, "0.00", out.Contribution)
	require.Equal(t, "negative", out.ContributionSign)
	require.Equal(t, "0.02", *out.BreakEven)
	require.Nil(t, out.RefundedContribution)
	in.Completed.Costs["platformRate"] = CostValue{Kind: "value", Value: "100"}
	_, out = CalculateCost(in)
	require.Nil(t, out.BreakEven)
	in.Completed.Costs["procurement"] = CostValue{Kind: "value", Value: "0"}
	_, out = CalculateCost(in)
	require.Equal(t, "0.01", *out.BreakEven)
	in.Completed.Costs["taxRate"] = CostValue{Kind: "value", Value: "1"}
	_, out = CalculateCost(in)
	require.Nil(t, out.BreakEven)
	in.Completed.Costs["taxRate"] = CostValue{Kind: "not_applicable", Reason: "not incurred"}
	in.Completed.Costs["platformRate"] = CostValue{Kind: "value", Value: "50"}
	_, out = CalculateCost(in)
	require.Equal(t, "0.01", out.Contribution)
	in.Completed.Costs["procurement"] = CostValue{Kind: "value", Value: "0.01"}
	_, out = CalculateCost(in)
	require.Equal(t, "-0.01", out.Contribution)
}
