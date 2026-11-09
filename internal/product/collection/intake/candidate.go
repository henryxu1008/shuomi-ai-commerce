// Package intake validates suggestions for an unsaved own product.
package intake

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	strict "sigs.k8s.io/json"
)

const Schema = "own-product-intake-candidate-v2"

var ErrInvalid = errors.New("invalid own product intake")

type Input struct {
	Notes         string `json:"notes"`
	GalleryCount  int    `json:"galleryCount"`
	SelectedImage int    `json:"selectedImage,omitempty"`
}

func (in Input) Valid() bool {
	return text(in.Notes, 4096, true) && in.Notes == strings.TrimSpace(in.Notes) && in.GalleryCount >= 0 && in.GalleryCount <= 40 && in.SelectedImage >= 0 && in.SelectedImage <= in.GalleryCount
}

type Fact struct {
	Value    string `json:"value"`
	Evidence string `json:"evidence"`
}

type Candidate struct {
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	Brand       *Fact           `json:"brand,omitempty"`
	Attributes  map[string]Fact `json:"attributes,omitempty"`
	Rows        []Row           `json:"rows,omitempty"`
	Questions   []string        `json:"questions,omitempty"`
}

type Row struct {
	Attributes      map[string]string `json:"attributes"`
	Evidence        string            `json:"evidence"`
	SKU             string            `json:"sku,omitempty"`
	PriceSuggestion *PriceSuggestion  `json:"priceSuggestion,omitempty"`
	Stock           *int              `json:"stock,omitempty"`
	Image           int               `json:"image,omitempty"`
}

// PriceSuggestion is not a verified product fact. The client must ask the user
// to confirm its meaning and variant before copying both amount and currency.
type PriceSuggestion struct {
	Amount   *float64 `json:"amount"`
	Currency string   `json:"currency,omitempty"`
	Evidence string   `json:"evidence"`
}

// ReviewCandidate is the server's projection for human review, not model input.
// The raw provider output and its receipt remain unchanged in the original Run.
type ReviewCandidate struct {
	Candidate
	StockReviewRows []int `json:"stockReviewRows,omitempty"`
}

func (c Candidate) ForReview() ReviewCandidate {
	out := ReviewCandidate{Candidate: c}
	out.Rows = append([]Row(nil), c.Rows...)
	for i, row := range out.Rows {
		// Quarantine uncertainty, not malformed values. Structural/source errors
		// must still fail Validate; never turn an invalid number into an omission.
		if row.Stock != nil && *row.Stock >= 0 && int64(*row.Stock) <= 9007199254740991 && !row.stockIsExplicit() {
			out.Rows[i].Stock = nil
			out.StockReviewRows = append(out.StockReviewRows, i)
		}
	}
	return out
}

func Decode(raw []byte, in Input) (ReviewCandidate, error) {
	var out Candidate
	if len(raw) == 0 || len(raw) > 28<<10 || !utf8.Valid(raw) {
		return ReviewCandidate{}, ValidationIssue{"candidate", "invalid_json"}
	}
	issues, err := strict.UnmarshalStrict(raw, &out, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
	if err != nil {
		return ReviewCandidate{}, ValidationIssue{"candidate", "invalid_json"}
	}
	if len(issues) != 0 {
		return ReviewCandidate{}, ValidationIssue{"candidate", "invalid_schema"}
	}
	review := out.ForReview()
	if err := review.Validate(in); err != nil {
		return ReviewCandidate{}, err
	}
	return review, nil
}

func (c Candidate) Validate(in Input) error {
	if !in.Valid() {
		return ValidationIssue{"candidate", "invalid_value"}
	}
	if !text(c.Title, 2000, false) {
		return ValidationIssue{"title", "invalid_value"}
	}
	if !text(c.Description, 8192, false) {
		return ValidationIssue{"description", "invalid_value"}
	}
	if len(c.Rows) > 256 {
		return ValidationIssue{"rows", "invalid_value"}
	}
	if len(c.Attributes) > 64 {
		return ValidationIssue{"attributes", "invalid_value"}
	}
	if len(c.Questions) > 12 {
		return ValidationIssue{"questions", "invalid_value"}
	}
	if c.Title == "" && c.Description == "" && c.Brand == nil && len(c.Attributes)+len(c.Rows)+len(c.Questions) == 0 {
		return ValidationIssue{"candidate", "empty_candidate"}
	}
	if c.Brand != nil && !c.Brand.valid(in.Notes) {
		return ValidationIssue{"brand", "source_evidence"}
	}
	for k, f := range c.Attributes {
		if !text(k, 200, true) || !f.valid(in.Notes) {
			return ValidationIssue{"attributes", "source_evidence"}
		}
	}
	for _, q := range c.Questions {
		if !text(q, 512, true) {
			return ValidationIssue{"questions", "invalid_value"}
		}
	}
	combinations, skus := map[string]bool{}, map[string]bool{}
	for i, row := range c.Rows {
		if err := row.validate(in, i); err != nil {
			return err
		}
		key, _ := json.Marshal(row.Attributes)
		if len(c.Rows) > 1 && len(row.Attributes) == 0 {
			return ValidationIssue{fmt.Sprintf("rows[%d].attributes", i), "invalid_value"}
		}
		if combinations[string(key)] {
			return ValidationIssue{fmt.Sprintf("rows[%d].attributes", i), "duplicate_variant"}
		}
		if row.SKU != "" && skus[strings.ToLower(row.SKU)] {
			return ValidationIssue{fmt.Sprintf("rows[%d].sku", i), "duplicate_sku"}
		}
		combinations[string(key)] = true
		skus[strings.ToLower(row.SKU)] = true
	}
	return nil
}

func (f Fact) valid(notes string) bool {
	return text(f.Value, 512, true) && text(f.Evidence, 512, true) && strings.Contains(notes, f.Evidence) && strings.Contains(f.Evidence, f.Value)
}

var currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

const stockField = `(?:库存(?:数量)?|\b(?:stock|inventory)\b(?:\s+|[：:=]))`

var stockDeclaration = regexp.MustCompile(`(?i)` + stockField)
var stockQuantity = regexp.MustCompile(`(?i)^\s*(?:明确\s*)?(?:为|是|is|[：:=])?\s*([0-9]+)(?:\s*(?:件|个|units?|pcs))?`)
var stockUncertain = regexp.MustCompile(`(?:^|[，,。;；])\s*(?:尚未确认|不能确认|待确认|未确认|未确定|待核实)(?:库存(?:数量)?|数量)?\s*(?:$|[，,。;；])`)
var stockQualified = regexp.MustCompile(`(?i)(?:并非|不是|不代表|不表示|未确认|没有确认|不能确认|大约|大概|约|预计|估计|可能|至少|至多)\s*库存|\b(?:not|approximately|about)\s+(?:stock|inventory)`)
var followingStockField = regexp.MustCompile(`(?i)^(?:SKU\b|货号|编码|币种|价格|单价|图片|图号|颜色|尺码|品牌|库存|(?:stock|inventory|price|currency|image|color|size|brand)\b)`)

const skuFieldPrefix = `(?i)(?:\bSKU\b|货号|商品编码)\s*[：:=]?\s*`

var skuFieldValue = regexp.MustCompile(skuFieldPrefix + `([\p{L}\p{N}_-]+)`)
var currencyEvidence = regexp.MustCompile(`\b([A-Z]{3})\s*[：:=]?\s*([0-9]+(?:\.[0-9]+)?)(?:[^0-9.]|$)`)
var imageEvidence = regexp.MustCompile(`(?i)(?:图片|image|图)\s*#?\s*([0-9]+)(?:[^0-9]|$)`)

func (r Row) validate(in Input, index int) error {
	issue := func(field, code string) error { return ValidationIssue{fmt.Sprintf("rows[%d].%s", index, field), code} }
	if !text(r.Evidence, 512, true) || strings.ContainsAny(r.Evidence, "\n\r;；") || !strings.Contains(in.Notes, r.Evidence) {
		return issue("evidence", "source_evidence")
	}
	if len(r.Attributes) > 12 {
		return issue("attributes", "invalid_value")
	}
	if !text(r.SKU, 200, false) {
		return issue("sku", "invalid_value")
	}
	for name, value := range r.Attributes {
		if !text(name, 200, true) || !text(value, 512, true) || name != strings.TrimSpace(name) || value != strings.TrimSpace(value) || strings.ContainsAny(value, ",，、\n") || !containsValue(r.Evidence, value) {
			return issue("attributes", "source_evidence")
		}
	}
	if r.SKU != "" && !containsValue(r.Evidence, r.SKU) {
		return issue("sku", "source_evidence")
	}
	if p := r.PriceSuggestion; p != nil {
		if p.Amount == nil || !(*p.Amount >= 0) || math.IsInf(*p.Amount, 0) || p.Currency != "" && !currencyCode.MatchString(p.Currency) {
			return issue("price", "invalid_value")
		}
		if !text(p.Evidence, 512, true) || !strings.Contains(in.Notes, p.Evidence) {
			return issue("price", "source_evidence")
		}
	}
	if r.Stock != nil && (*r.Stock < 0 || int64(*r.Stock) > 9007199254740991 || !r.stockIsExplicit()) {
		return issue("stock", "stock_evidence")
	}
	if r.Image < 0 || r.Image > in.GalleryCount || !(r.Image == 0 || r.Image == in.SelectedImage || matchesNumber(imageEvidence, r.Evidence, float64(r.Image))) {
		return issue("image", "image_reference")
	}
	return nil
}

func (r Row) stockIsExplicit() bool {
	// SKU values are identifiers, including when they contain words such as STOCK.
	// Mask only that field's value; never use its digits as inventory evidence.
	evidence := []byte(r.Evidence)
	identifier := skuFieldValue
	if r.SKU != "" {
		identifier = regexp.MustCompile(skuFieldPrefix + `(` + regexp.QuoteMeta(r.SKU) + `)`)
	}
	for _, m := range identifier.FindAllStringSubmatchIndex(r.Evidence, -1) {
		for i := m[2]; i < m[3]; i++ {
			evidence[i] = '_'
		}
	}
	source := string(evidence)
	if stockUncertain.MatchString(source) || stockQualified.MatchString(source) {
		return false
	}
	matches := stockDeclaration.FindAllStringIndex(source, -1)
	if len(matches) == 0 {
		return false
	}
	for _, m := range matches {
		// Every inventory declaration must state the same exact integer. A later
		// unknown/qualified declaration cannot be silently skipped.
		tail := source[m[1]:]
		value := stockQuantity.FindStringSubmatchIndex(tail)
		if value == nil {
			return false
		}
		quantity, err := strconv.ParseInt(tail[value[2]:value[3]], 10, 64)
		if err != nil || quantity != int64(*r.Stock) {
			return false
		}
		// Reject decimal, range and approximate suffixes rather than accepting a prefix.
		rest := tail[value[1]:]
		after := strings.TrimLeftFunc(rest, unicode.IsSpace)
		if after != "" && !strings.ContainsRune("，,。;；)）]", []rune(after)[0]) {
			currency := currencyEvidence.FindStringSubmatchIndex(after)
			knownCurrency := r.PriceSuggestion != nil && r.PriceSuggestion.Currency != "" && currency != nil && currency[0] == 0 && after[currency[2]:currency[3]] == r.PriceSuggestion.Currency
			if len(rest) == len(after) || !followingStockField.MatchString(after) && !knownCurrency {
				return false
			}
		}
	}
	return true
}

func matchesNumber(pattern *regexp.Regexp, evidence string, value float64) bool {
	for _, m := range pattern.FindAllStringSubmatch(evidence, -1) {
		if n, err := strconv.ParseFloat(m[1], 64); err == nil && n == value {
			return true
		}
	}
	return false
}

func containsValue(source, value string) bool {
	// ASCII SKU/size fragments must be complete tokens. CJK evidence may
	// naturally omit whitespace around a colour or other literal.
	if strings.IndexFunc(value, func(r rune) bool { return r > unicode.MaxASCII }) >= 0 {
		return strings.Contains(source, value)
	}
	boundary := func(b byte) bool {
		return !(b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-' || b == '_')
	}
	for off := 0; off < len(source); {
		i := strings.Index(source[off:], value)
		if i < 0 {
			return false
		}
		i += off
		end := i + len(value)
		if (i == 0 || boundary(source[i-1])) && (end == len(source) || boundary(source[end])) {
			return true
		}
		off = i + 1
	}
	return false
}

func text(s string, limit int, required bool) bool {
	return len(s) <= limit && utf8.ValidString(s) && !strings.ContainsRune(s, 0) && (!required || strings.TrimSpace(s) != "")
}
