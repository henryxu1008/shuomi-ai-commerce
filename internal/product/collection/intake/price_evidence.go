package intake

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/currency"
)

// Price evidence is deliberately a bounded declaration grammar, not a fallback
// number extractor. Unsupported notation stays unknown instead of guessing.
var priceDeclaration = regexp.MustCompile(`价格|单价|(?i:\bprice\b)|[¥￥$€]|\b[A-Z]{3}`)
var priceAmount = regexp.MustCompile(`^\s*(?:明确\s*)?(?:为|是|(?i:is)|[：:=])?\s*(?:([A-Z]{3}|[¥￥$€])\s*[：:=]?\s*)?([0-9]+(?:\.[0-9]+)?)`)
var priceMonetaryUnit = regexp.MustCompile(`^(人民币|美元|美金|欧元|英镑|日元|日圆|円|元|[¥￥$€]|[A-Z]{3}\b)(?:\s*/(?:件|个|套))?`)
var priceOtherValue = regexp.MustCompile(`(?:品牌|颜色|尺码|图片|图号|(?i:\b(?:brand|color|size|image)\b))\s*[：:=]?\s*([A-Za-z0-9_-]+)`)
var priceCurrencyField = regexp.MustCompile(`(?:币种|(?i:\bcurrency\b))\s*[：:=]?\s*([A-Z]{3})\b`)
var priceShippingFee = regexp.MustCompile(`(?i)(?:^|[，,。])\s*(?:运费|邮费|配送费|shipping(?:\s+(?:fee|cost))?|postage|freight)\s*(?:为|是|is|[：:=])?\s*((?:[A-Z]{3}|[¥￥$€])\s*[：:=]?\s*[0-9]+(?:\.[0-9]+)?)`)
var priceQualified = regexp.MustCompile(`(?i)并非|不是|不为|不代表|不表示|未确认|尚未确认|没有确认|不能确认|待确认|待定|大约|大概|约|预计|估计|可能|至少|至多|不低于|不高于|\b(?:not|pending|unconfirmed|about|approximately|from|between)\b`)
var priceUncertain = regexp.MustCompile(`(?i)(?:^|[，,。(（])\s*(?:尚未确认|不能确认|待确认|未确认|未确定|待核实|待定|未知|not confirmed|pending|unconfirmed)(?:\s|$|[，,。)）])`)
var priceOtherField = regexp.MustCompile(`(?i)(?:尚未确认|不能确认|待确认|未确认|未确定|待核实|待定|未知|并非|不是|大约)?\s*(?:库存|图片|图号|颜色|尺码|品牌|货号|商品编码|\b(?:SKU|stock|inventory|image|color|size|brand)\b)`)
var priceContinuation = regexp.MustCompile(`^[，,。]\s*(?:[0-9]|或|至|到|[-~～–—])`)

func (r Row) priceIsExplicit(statements []string) bool {
	currencyMatches := r.priceCurrencyValidator()
	if !r.priceStatementIsExplicit(r.Evidence, currencyMatches) {
		return false
	}
	// A model may quote a substring of a range or omit a negation/conflict. Check
	// the original containing statement too; never grant authority to that excerpt.
	found := false
	for _, statement := range statements {
		if strings.Contains(statement, r.Evidence) {
			found = true
		}
		if !r.priceStatementIsExplicit(statement, currencyMatches) {
			return false
		}
	}
	return found
}

func (r Row) priceStatementIsExplicit(statement string, currencyMatches func(string) bool) bool {
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
	// A separately labelled shipping charge is not the variant's unit price.
	// Mask only its direct currency/amount, never the surrounding clause where a
	// product-price conflict or qualifier may follow.
	for _, m := range priceShippingFee.FindAllStringSubmatchIndex(string(source), -1) {
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
	count, consumedUntil := 0, 0
	for _, m := range priceDeclaration.FindAllStringIndex(s, -1) {
		if m[0] < consumedUntil {
			continue // A currency inside an already checked amount is not a new declaration.
		}
		marker := s[m[0]:m[1]]
		if marker == "SKU" || currencyCode.MatchString(marker) && m[1] < len(s) && (s[m[1]] >= 'A' && s[m[1]] <= 'Z' || s[m[1]] >= 'a' && s[m[1]] <= 'z' || s[m[1]] == '_' || s[m[1]] == '-') {
			continue
		}
		// A product abbreviation is not a currency declaration. Keep actual ISO
		// currencies (including conflicts), the row's currency, and explicitly
		// labelled price/currency fields subject to the strict checks below.
		if currencyCode.MatchString(marker) && marker != r.Currency {
			if _, err := currency.ParseISO(marker); err != nil {
				continue
			}
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
		if !currencyMatches(marker) || explicitCurrency != "" && !currencyMatches(explicitCurrency) {
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
		consumedUntil = m[1] + amount[1]
		unit := priceMonetaryUnit.FindStringSubmatchIndex(after)
		if unit != nil && currencyCode.MatchString(after[unit[2]:unit[3]]) && after[unit[2]:unit[3]] != r.Currency {
			if _, err := currency.ParseISO(after[unit[2]:unit[3]]); err != nil {
				unit = nil // SKU and other following fields are not currency units.
			}
		}
		if unit != nil {
			if !currencyMatches(after[unit[2]:unit[3]]) {
				return false
			}
			currencyFound = true
			consumedUntil += len(rest) - len(after) + unit[1]
			rest = after[unit[1]:]
			after = strings.TrimLeftFunc(rest, unicode.IsSpace)
		}
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

// Keep the intersection across the quote and every original scope. Omitting
// Currency cannot turn conflicting dollar/yuan declarations into one price.
func (r Row) priceCurrencyValidator() func(string) bool {
	allowed := []string{"CNY", "JPY", "USD", "EUR", "GBP"}
	if r.Currency != "" {
		allowed = []string{r.Currency}
	}
	return func(marker string) bool {
		if currencyCode.MatchString(marker) {
			return r.Currency == marker
		}
		options := priceCurrencyOptions(marker)
		if len(options) == 0 {
			return true
		}
		allowed = slices.DeleteFunc(allowed, func(code string) bool { return !slices.Contains(options, code) })
		return len(allowed) != 0
	}
}

func priceCurrencyOptions(marker string) []string {
	switch marker {
	case "$", "美元", "美金":
		return []string{"USD"}
	case "€", "欧元":
		return []string{"EUR"}
	case "人民币", "元":
		return []string{"CNY"}
	case "英镑":
		return []string{"GBP"}
	case "日元", "日圆", "円":
		return []string{"JPY"}
	case "¥", "￥":
		return []string{"CNY", "JPY"}
	default:
		return nil
	}
}
