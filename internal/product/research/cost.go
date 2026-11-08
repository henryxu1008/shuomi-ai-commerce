package research

import (
	"encoding/json"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const CostRuleVersion = "manual-unit-economics-v1"

type CostBasis struct {
	Source string `json:"source"`
	AsOf   string `json:"asOf"`
}
type CostValue struct {
	Kind   string     `json:"kind"`
	Value  string     `json:"value,omitempty"`
	Reason string     `json:"reason,omitempty"`
	Basis  *CostBasis `json:"basis,omitempty"`
}

func (v CostValue) MarshalJSON() ([]byte, error) {
	if v.Kind == "value" {
		return json.Marshal(struct {
			Kind  string     `json:"kind"`
			Value string     `json:"value"`
			Basis *CostBasis `json:"basis,omitempty"`
		}{v.Kind, v.Value, v.Basis})
	}
	return json.Marshal(struct {
		Kind   string `json:"kind"`
		Reason string `json:"reason"`
	}{v.Kind, v.Reason})
}

type CostOutcome struct {
	Basis CostBasis            `json:"basis"`
	Costs map[string]CostValue `json:"costs"`
}
type CostInput struct {
	Channel          string      `json:"channel"`
	Fulfillment      string      `json:"fulfillment"`
	Specification    string      `json:"specification"`
	Quantity         string      `json:"quantity"`
	Price            string      `json:"price"`
	RefundRate       string      `json:"refundRate"`
	SalesBasis       CostBasis   `json:"salesBasis"`
	Rule             string      `json:"rule"`
	PurchaseCurrency string      `json:"purchaseCurrency"`
	ExchangeRate     string      `json:"exchangeRate"`
	ExchangeBasis    CostBasis   `json:"exchangeBasis"`
	Completed        CostOutcome `json:"completed"`
	Refunded         CostOutcome `json:"refunded"`
}
type CostResult struct {
	Status                string   `json:"status"`
	Fields                []string `json:"fields,omitempty"`
	Gross                 string   `json:"gross,omitempty"`
	Contribution          string   `json:"contribution,omitempty"`
	ContributionSign      string   `json:"contributionSign,omitempty"`
	BreakEven             *string  `json:"breakEven"`
	CompletedContribution *string  `json:"completedContribution"`
	RefundedContribution  *string  `json:"refundedContribution"`
}

func (r CostResult) MarshalJSON() ([]byte, error) {
	if r.Status == "INVALID_INPUT" || r.Status == "INSUFFICIENT_INPUT" {
		return json.Marshal(struct {
			Status string   `json:"status"`
			Fields []string `json:"fields"`
		}{r.Status, r.Fields})
	}
	if r.Status != "COMPUTABLE" {
		return json.Marshal(struct {
			Status string `json:"status"`
		}{r.Status})
	}
	type plain CostResult
	return json.Marshal(plain(r))
}

var costFields = []string{"procurement", "logistics", "platformFixed", "platformRate", "taxFixed", "taxRate", "advertising", "exchangeCost"}
var costDecimal = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)

func integer(n int64) *big.Int          { return big.NewInt(n) }
func product(a, b *big.Int) *big.Int    { return new(big.Int).Mul(a, b) }
func sum(a, b *big.Int) *big.Int        { return new(big.Int).Add(a, b) }
func difference(a, b *big.Int) *big.Int { return new(big.Int).Sub(a, b) }
func fixedString(n *big.Int, scale int) string {
	v := new(big.Int).Abs(n).String()
	for len(v) <= scale {
		v = "0" + v
	}
	if scale > 0 {
		v = v[:len(v)-scale] + "." + v[len(v)-scale:]
	}
	if n.Sign() < 0 {
		v = "-" + v
	}
	return v
}
func roundedMoney(n, denominator *big.Int) string {
	v := new(big.Int).Abs(n)
	v.Add(v, new(big.Int).Quo(denominator, integer(2)))
	v.Quo(v, denominator)
	if n.Sign() < 0 {
		v.Neg(v)
	}
	return fixedString(v, 2)
}

type costTerms struct {
	slope, fixed, procurement *big.Int
	active                    bool
}

// CalculateCost is the single authority for both preview and immutable saves.
// All decimal inputs are strings; multiplication/rounding never uses floating point.
func CalculateCost(in CostInput) (CostInput, CostResult) {
	missing, invalid := map[string]bool{}, map[string]bool{}
	text := func(path, value string, max int, required bool) string {
		if !utf8.ValidString(value) || len(value) > max || strings.ContainsRune(value, 0) {
			invalid[path] = true
		}
		value = strings.TrimSpace(value)
		if required && value == "" {
			missing[path] = true
		}
		return value
	}
	basis := func(path string, b CostBasis, required bool) CostBasis {
		b.Source = text(path+".source", b.Source, 240, required)
		if b.AsOf == "" {
			if required {
				missing[path+".asOf"] = true
			}
		} else if d, e := time.Parse("2006-01-02", b.AsOf); e != nil || d.Year() == 0 || d.Format("2006-01-02") != b.AsOf {
			invalid[path+".asOf"] = true
		}
		return b
	}
	number := func(path string, value *string, scale int, min, max int64, required bool) *big.Int {
		if strings.TrimSpace(*value) == "" {
			if required {
				missing[path] = true
			}
			*value = ""
			return integer(0)
		}
		parts := strings.Split(*value, ".")
		if len(*value) > 16 || !costDecimal.MatchString(*value) || len(parts) > 1 && len(parts[1]) > scale {
			invalid[path] = true
			return integer(0)
		}
		fraction := ""
		if len(parts) > 1 {
			fraction = parts[1]
		}
		n, ok := new(big.Int).SetString(parts[0]+fraction+strings.Repeat("0", scale-len(fraction)), 10)
		if !ok || n.Cmp(integer(min)) < 0 || n.Cmp(integer(max)) > 0 {
			invalid[path] = true
			return integer(0)
		}
		*value = fixedString(n, scale)
		return n
	}
	in.Channel = text("channel", in.Channel, 120, true)
	in.Fulfillment = text("fulfillment", in.Fulfillment, 120, true)
	in.Specification = text("specification", in.Specification, 120, true)
	number("quantity", &in.Quantity, 0, 1, 1_000_000, true)
	price := number("price", &in.Price, 2, 1, 100_000_000, true)
	refund := number("refundRate", &in.RefundRate, 2, 0, 10_000, true)
	in.SalesBasis = basis("salesBasis", in.SalesBasis, true)
	if in.Rule != "fixed_percent" && in.Rule != "unsupported" {
		invalid["rule"] = true
	}
	fx := integer(1_000_000)
	cny := in.PurchaseCurrency == "CNY"
	if in.PurchaseCurrency != "USD" && !cny {
		invalid["purchaseCurrency"] = true
	}
	parsedFX := number("exchangeRate", &in.ExchangeRate, 6, 1, 1_000_000_000, cny)
	if cny {
		fx = parsedFX
	}
	in.ExchangeBasis = basis("exchangeBasis", in.ExchangeBasis, cny)
	terms := func(name string, out CostOutcome, active bool) (CostOutcome, costTerms) {
		normal := CostOutcome{Basis: basis(name+".basis", out.Basis, false), Costs: map[string]CostValue{}}
		for key := range out.Costs {
			found := false
			for _, k := range costFields {
				found = found || k == key
			}
			if !found {
				invalid[name+".costs."+key] = true
			}
		}
		amounts := map[string]*big.Int{}
		for _, key := range costFields {
			value, exists := out.Costs[key]
			path := name + "." + key
			amounts[key] = integer(0)
			if !exists {
				value = CostValue{Kind: "value"}
			}
			switch value.Kind {
			case "not_applicable":
				if key == "procurement" || value.Value != "" || value.Basis != nil {
					invalid[path] = true
				}
				value.Reason = text(path, value.Reason, 240, active)
			case "value":
				if value.Reason != "" {
					invalid[path] = true
				}
				maximum := int64(100_000_000)
				if key == "platformRate" || key == "taxRate" {
					maximum = 10_000
				}
				amounts[key] = number(path, &value.Value, 2, 0, maximum, active)
				if value.Basis != nil {
					b := basis(path+".basis", *value.Basis, active && value.Value != "")
					value.Basis = &b
				} else if active && value.Value != "" {
					normal.Basis = basis(name+".basis", normal.Basis, true)
				}
			default:
				invalid[path] = true
			}
			normal.Costs[key] = value
		}
		procurement := product(amounts["procurement"], fx)
		fixed := new(big.Int).Set(procurement)
		for _, k := range []string{"logistics", "platformFixed", "taxFixed", "advertising", "exchangeCost"} {
			fixed.Add(fixed, product(amounts[k], integer(1_000_000)))
		}
		revenue := int64(0)
		if name == "completed" {
			revenue = 10_000
		}
		slope := product(difference(difference(integer(revenue), amounts["platformRate"]), amounts["taxRate"]), integer(100))
		return normal, costTerms{slope, fixed, procurement, active}
	}
	completedProbability := difference(integer(10_000), refund)
	var completed, refunded costTerms
	in.Completed, completed = terms("completed", in.Completed, completedProbability.Sign() != 0)
	in.Refunded, refunded = terms("refunded", in.Refunded, refund.Sign() != 0)
	fields := func(values map[string]bool) []string {
		out := []string{}
		for k := range values {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	if len(invalid) > 0 {
		return in, CostResult{Status: "INVALID_INPUT", Fields: fields(invalid)}
	}
	if in.Rule == "unsupported" {
		return in, CostResult{Status: "UNSUPPORTED_COST_RULE"}
	}
	if len(missing) > 0 {
		return in, CostResult{Status: "INSUFFICIENT_INPUT", Fields: fields(missing)}
	}
	slope := sum(product(completedProbability, completed.slope), product(refund, refunded.slope))
	fixed := sum(product(completedProbability, completed.fixed), product(refund, refunded.fixed))
	contribution := difference(product(price, slope), fixed)
	gross := difference(product(completedProbability, difference(product(price, integer(1_000_000)), completed.procurement)), product(refund, refunded.procurement))
	out := CostResult{Status: "COMPUTABLE", Gross: roundedMoney(gross, integer(10_000_000_000)), Contribution: roundedMoney(contribution, integer(10_000_000_000)), ContributionSign: "zero"}
	if contribution.Sign() < 0 {
		out.ContributionSign = "negative"
	} else if contribution.Sign() > 0 {
		out.ContributionSign = "positive"
	}
	threshold := integer(1)
	if slope.Sign() > 0 {
		threshold.Quo(sum(fixed, difference(slope, integer(1))), slope)
		if threshold.Sign() < 1 {
			threshold.SetInt64(1)
		}
	} else if slope.Cmp(fixed) < 0 {
		threshold.SetInt64(100_000_001)
	}
	if threshold.Cmp(integer(100_000_000)) <= 0 {
		v := fixedString(threshold, 2)
		out.BreakEven = &v
	}
	outcome := func(v costTerms) *string {
		if !v.active {
			return nil
		}
		money := roundedMoney(difference(product(price, v.slope), v.fixed), integer(1_000_000))
		return &money
	}
	out.CompletedContribution = outcome(completed)
	out.RefundedContribution = outcome(refunded)
	return in, out
}
