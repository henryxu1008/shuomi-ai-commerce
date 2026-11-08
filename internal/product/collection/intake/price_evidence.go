package intake

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Price evidence is deliberately a bounded declaration grammar, not a fallback
// number extractor. Unsupported notation stays unknown instead of guessing.
var priceDeclaration = regexp.MustCompile(`价格|单价|(?i:\bprice\b)|[¥￥$€]|\b[A-Z]{3}`)
var priceAmount = regexp.MustCompile(`^\s*(?:明确\s*)?(?:为|是|(?i:is)|[：:=])?\s*(?:([A-Z]{3}|[¥￥$€])\s*[：:=]?\s*)?([0-9]+(?:\.[0-9]+)?)`)
var priceOtherValue = regexp.MustCompile(`(?:品牌|颜色|尺码|图片|图号|(?i:\b(?:brand|color|size|image)\b))\s*[：:=]?\s*([A-Za-z0-9_-]+)`)
var priceCurrencyField = regexp.MustCompile(`(?:币种|(?i:\bcurrency\b))\s*[：:=]?\s*([A-Z]{3})\b`)
var priceQualified = regexp.MustCompile(`(?i)并非|不是|不为|不代表|不表示|未确认|尚未确认|没有确认|不能确认|待确认|待定|大约|大概|约|预计|估计|可能|至少|至多|不低于|不高于|\b(?:not|pending|unconfirmed|about|approximately|from|between)\b`)
var priceUncertain = regexp.MustCompile(`(?i)(?:^|[，,。(（])\s*(?:尚未确认|不能确认|待确认|未确认|未确定|待核实|待定|未知|not confirmed|pending|unconfirmed)\s*(?:$|[，,。)）])`)
var priceOtherField = regexp.MustCompile(`(?i)(?:尚未确认|不能确认|待确认|未确认|未确定|待核实|待定|未知|并非|不是|大约)?\s*(?:库存|图片|图号|颜色|尺码|品牌|货号|商品编码|\b(?:SKU|stock|inventory|image|color|size|brand)\b)`)
var priceContinuation = regexp.MustCompile(`^[，,。]\s*(?:[0-9]|或|至|到|[-~～–—])`)

func (r Row) priceIsExplicit(notes string) bool {
	if !r.priceStatementIsExplicit(r.Evidence) {
		return false
	}
	// A model may quote a substring of a range or omit a negation/conflict. Check
	// the original containing statement too; never grant authority to that excerpt.
	found := false
	for _, statement := range strings.FieldsFunc(notes, func(c rune) bool { return strings.ContainsRune("\n\r;；", c) }) {
		if strings.Contains(statement, r.Evidence) {
			found = true
			if !r.priceStatementIsExplicit(statement) {
				return false
			}
		}
	}
	return found
}

func (r Row) priceStatementIsExplicit(statement string) bool {
	// Identifiers are not prices, even if a SKU contains PRICE, USD or digits.
	source := []byte(statement)
	currencyFound := r.Currency == ""
	identifier := skuFieldValue
	if r.SKU != "" {
		identifier = regexp.MustCompile(skuFieldPrefix + `(` + regexp.QuoteMeta(r.SKU) + `)`)
	}
	mask := func(start, end int) {
		for i := start; i < end; i++ {
			source[i] = '_'
		}
	}
	for _, m := range identifier.FindAllStringSubmatchIndex(statement, -1) {
		mask(m[2], m[3])
	}
	for _, m := range priceOtherValue.FindAllStringSubmatchIndex(string(source), -1) {
		mask(m[2], m[3])
	}
	// A separate currency field is allowed, but all explicitly declared currencies
	// must agree. Mask it so it cannot masquerade as an amount declaration.
	for _, m := range priceCurrencyField.FindAllStringSubmatchIndex(string(source), -1) {
		if statement[m[2]:m[3]] != r.Currency {
			return false
		}
		mask(m[0], m[1])
		currencyFound = true
	}
	s := string(source)
	count := 0
	for _, m := range priceDeclaration.FindAllStringIndex(s, -1) {
		marker := s[m[0]:m[1]]
		if marker == "SKU" || currencyCode.MatchString(marker) && m[1] < len(s) && (s[m[1]] >= 'A' && s[m[1]] <= 'Z' || s[m[1]] >= 'a' && s[m[1]] <= 'z' || s[m[1]] == '_' || s[m[1]] == '-') {
			continue
		}
		// Qualification before this declaration is significant, unlike an unrelated
		// unknown stock/image field in another clause.
		clauseStart := strings.LastIndexAny(s[:m[0]], "，,。") + 1
		if priceQualified.MatchString(s[clauseStart:m[0]]) {
			return false
		}
		tail := s[m[1]:]
		amount := priceAmount.FindStringSubmatchIndex(tail)
		if amount == nil {
			return false
		}
		explicitCurrency := ""
		if amount[2] >= 0 {
			explicitCurrency = tail[amount[2]:amount[3]]
		}
		if !r.priceCurrencyMatches(marker) || explicitCurrency != "" && !r.priceCurrencyMatches(explicitCurrency) {
			return false
		}
		if currencyCode.MatchString(marker) || currencyCode.MatchString(explicitCurrency) {
			currencyFound = true
		}
		n, err := strconv.ParseFloat(tail[amount[4]:amount[5]], 64)
		if err != nil || n != *r.Price {
			return false
		}
		rest := tail[amount[1]:]
		after := strings.TrimLeftFunc(rest, unicode.IsSpace)
		if priceContinuation.MatchString(after) {
			return false
		}
		if after != "" && !strings.ContainsRune("，,。)）]", []rune(after)[0]) {
			// A space may introduce another typed field, but never a range, decimal,
			// grouping, approximation or arbitrary word after the amount.
			currency := currencyEvidence.FindStringSubmatchIndex(after)
			knownCurrency := r.Currency != "" && currency != nil && currency[0] == 0 && after[currency[2]:currency[3]] == r.Currency
			if len(rest) == len(after) || !followingStockField.MatchString(after) && !knownCurrency {
				return false
			}
		}
		qualification := tail
		if next := priceOtherField.FindStringIndex(qualification); next != nil {
			qualification = qualification[:next[0]]
		}
		if priceUncertain.MatchString(qualification) {
			return false
		}
		count++
	}
	return count != 0 && currencyFound
}

func (r Row) priceCurrencyMatches(marker string) bool {
	if currencyCode.MatchString(marker) {
		return r.Currency == marker
	}
	switch marker {
	case "$":
		return r.Currency == "" || r.Currency == "USD"
	case "€":
		return r.Currency == "" || r.Currency == "EUR"
	case "¥", "￥":
		return r.Currency == "" || r.Currency == "CNY" || r.Currency == "JPY"
	default:
		return true
	}
}
