package intake

import (
	"regexp"
	"slices"
	"sort"
	"strings"

	"golang.org/x/text/currency"
)

var priceClauseSeparator = regexp.MustCompile(`[,，]`)
var unsafeVariantHeader = regexp.MustCompile(`^[0-9.+~～–—-]|^(?i:or|to)\b|或|待确认|待定|未确定|待核实|未知`)

// Price scopes come from complete variant headers in the original notes, never
// from the model's evidence boundaries. Unrecognized/ambiguous syntax retains
// the whole statement so a clipped quote cannot hide a conflict or qualifier.
func (c Candidate) priceStatements(notes string) [][]string {
	result := make([][]string, len(c.Rows))
	headers := make([][]*regexp.Regexp, len(c.Rows))
	var dimensions []string
	for i, row := range c.Rows {
		keys := make([]string, 0, len(row.Attributes))
		for key := range row.Attributes {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if i > 0 && !slices.Equal(keys, dimensions) {
			headers = nil // Arbitrary extra facts cannot create a variant boundary.
			break
		}
		dimensions = keys
		for _, key := range keys {
			value := row.Attributes[key]
			if strings.TrimSpace(value) == "" {
				continue
			}
			headers[i] = append(headers[i], regexp.MustCompile(`(^|\s)(?:`+regexp.QuoteMeta(key)+`\s*[：:=]?\s*)?`+regexp.QuoteMeta(value)+`($|\s)`))
		}
	}
	for _, statement := range strings.FieldsFunc(notes, func(c rune) bool { return strings.ContainsRune("\n\r;；", c) }) {
		scopes := variantPriceScopes(statement, headers)
		if len(scopes) == 0 {
			for i, row := range c.Rows {
				if strings.Contains(statement, row.Evidence) {
					result[i] = append(result[i], statement)
				}
			}
			continue
		}
		for i, scope := range scopes {
			start, end := scope.start, len(statement)
			if i == 0 {
				start = 0 // Preserve any qualification before the first header.
			}
			if i+1 < len(scopes) {
				end = scopes[i+1].start
			}
			result[scope.row] = append(result[scope.row], strings.TrimRight(statement[start:end], " ,，\t"))
		}
	}
	return result
}

type variantPriceScope struct{ start, row int }

func variantPriceScopes(statement string, headers [][]*regexp.Regexp) []variantPriceScope {
	var scopes []variantPriceScope
	seen := map[int]bool{}
	start := 0
	separators := priceClauseSeparator.FindAllStringIndex(statement, -1)
	separators = append(separators, []int{len(statement), len(statement)})
	for _, separator := range separators {
		clause := strings.TrimSpace(statement[start:separator[0]])
		if variantHeaderHasPrice(clause) || unsafeVariantHeader.MatchString(clause) || priceQualified.MatchString(clause) {
			start = separator[1]
			continue
		}
		match := -1
		for row, patterns := range headers {
			if len(patterns) == 0 {
				continue
			}
			rest := clause
			for _, pattern := range patterns {
				if !pattern.MatchString(rest) {
					rest = "!"
					break
				}
				rest = pattern.ReplaceAllString(rest, " ")
			}
			if strings.TrimSpace(rest) == "" {
				if match != -1 {
					return nil // The source does not uniquely identify a row.
				}
				match = row
			}
		}
		if match != -1 {
			scopes = append(scopes, variantPriceScope{start, match})
			seen[match] = true
		}
		start = separator[1]
	}
	if len(seen) < 2 {
		return nil // Repeating one variant cannot reset a conflicting price.
	}
	return scopes
}

func variantHeaderHasPrice(clause string) bool {
	for _, marker := range priceDeclaration.FindAllStringIndex(clause, -1) {
		value := clause[marker[0]:marker[1]]
		if value == "SKU" {
			continue
		}
		if !currencyCode.MatchString(value) || priceAmount.MatchString(clause[marker[1]:]) {
			return true
		}
		// XXX is ISO's non-currency placeholder and also a common size value.
		if unit, err := currency.ParseISO(value); err == nil && unit != currency.XXX {
			return true
		}
	}
	return false
}
