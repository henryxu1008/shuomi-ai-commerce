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

const Schema = "own-product-intake-candidate-v3"

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
	StockSuggestion *StockSuggestion  `json:"stockSuggestion,omitempty"`
	Image           int               `json:"image,omitempty"`
}

// PriceSuggestion is not a verified product fact. The client must ask the user
// to confirm its meaning and variant before copying both amount and currency.
type PriceSuggestion struct {
	Amount   *float64 `json:"amount"`
	Currency string   `json:"currency,omitempty"`
	Evidence string   `json:"evidence"`
}

// StockSuggestion remains unverified until the user confirms its meaning and
// variant. A missing suggestion is unknown; an explicit quantity may be zero.
type StockSuggestion struct {
	Quantity *int64 `json:"quantity"`
	Evidence string `json:"evidence"`
}

func Decode(raw []byte, in Input) (Candidate, error) {
	var out Candidate
	if len(raw) == 0 || len(raw) > 28<<10 || !utf8.Valid(raw) {
		return Candidate{}, ValidationIssue{"candidate", "invalid_json"}
	}
	issues, err := strict.UnmarshalStrict(raw, &out, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
	if err != nil {
		return Candidate{}, ValidationIssue{"candidate", "invalid_json"}
	}
	if len(issues) != 0 {
		return Candidate{}, ValidationIssue{"candidate", "invalid_schema"}
	}
	if err := out.Validate(in); err != nil {
		return Candidate{}, err
	}
	return out, nil
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
	if stock := r.StockSuggestion; stock != nil {
		if stock.Quantity == nil || *stock.Quantity < 0 || *stock.Quantity > 9007199254740991 {
			return issue("stock", "invalid_value")
		}
		if !text(stock.Evidence, 512, true) || !strings.Contains(in.Notes, stock.Evidence) {
			return issue("stock", "source_evidence")
		}
	}
	if r.Image < 0 || r.Image > in.GalleryCount || !(r.Image == 0 || r.Image == in.SelectedImage || matchesNumber(imageEvidence, r.Evidence, float64(r.Image))) {
		return issue("image", "image_reference")
	}
	return nil
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
