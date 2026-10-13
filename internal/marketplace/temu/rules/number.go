package rules

import (
	"math/big"
	"strings"
)

// NumberRule retains official bounds and lexical precision. Values are
// compared with the standard library's exact rational arithmetic, never a
// float, default, clamp, unit conversion or exchange rate.
type NumberRule struct {
	Precision        int    `json:"precision"`
	Minimum          string `json:"minimum"`
	Maximum          string `json:"maximum"`
	MinimumExclusive bool   `json:"minimumExclusive"`
	MaximumExclusive bool   `json:"maximumExclusive"`
	RawRange         string `json:"rawRange"`
}

func (r NumberRule) Allows(value string) bool {
	if r.Precision < 0 || r.Precision > 2 || r.RawRange == "" {
		return false
	}
	v, decimals, ok := exactDecimal(value)
	minimum, _, minimumOK := exactDecimal(r.Minimum)
	maximum, _, maximumOK := exactDecimal(r.Maximum)
	if !ok || !minimumOK || !maximumOK || decimals > r.Precision || minimum.Cmp(maximum) > 0 {
		return false
	}
	lower, upper := v.Cmp(minimum), v.Cmp(maximum)
	return (lower > 0 || lower == 0 && !r.MinimumExclusive) && (upper < 0 || upper == 0 && !r.MaximumExclusive)
}

func exactDecimal(value string) (*big.Rat, int, bool) {
	if len(value) < 1 || len(value) > 128 {
		return nil, 0, false
	}
	whole, fraction, dot := strings.Cut(value, ".")
	if whole == "" || dot && fraction == "" {
		return nil, 0, false
	}
	for _, part := range []string{whole, fraction} {
		for _, char := range part {
			if char < '0' || char > '9' {
				return nil, 0, false
			}
		}
	}
	number, ok := new(big.Rat).SetString(value)
	return number, len(fraction), ok
}

func parseNumberRule(precision, rawRange string) (NumberRule, error) {
	rule := NumberRule{RawRange: rawRange}
	switch precision {
	case "Integer":
		rule.Precision = 0
	case "1 decimal point":
		rule.Precision = 1
	case "2 decimal points":
		rule.Precision = 2
	default:
		return NumberRule{}, ErrOfficialSnapshotInvalid
	}
	if len(rawRange) < 5 || rawRange[0] != '[' && rawRange[0] != '(' || rawRange[len(rawRange)-1] != ']' && rawRange[len(rawRange)-1] != ')' {
		return NumberRule{}, ErrOfficialSnapshotInvalid
	}
	minimum, maximum, comma := strings.Cut(rawRange[1:len(rawRange)-1], ",")
	rule.Minimum, rule.Maximum = strings.TrimSpace(minimum), strings.TrimSpace(maximum)
	rule.MinimumExclusive = rawRange[0] == '('
	rule.MaximumExclusive = rawRange[len(rawRange)-1] == ')'
	min, _, minOK := exactDecimal(rule.Minimum)
	max, _, maxOK := exactDecimal(rule.Maximum)
	if !comma || !minOK || !maxOK || min.Cmp(max) > 0 {
		return NumberRule{}, ErrOfficialSnapshotInvalid
	}
	return rule, nil
}
