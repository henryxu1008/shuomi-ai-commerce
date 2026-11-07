// Package intake validates suggestions for an unsaved own product.
package intake

import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	strict "sigs.k8s.io/json"
)

const Schema = "own-product-intake-candidate-v1"

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
	Attributes map[string]string `json:"attributes"`
	Evidence   string            `json:"evidence"`
	SKU        string            `json:"sku,omitempty"`
	Currency   string            `json:"currency,omitempty"`
	Price      *float64          `json:"price,omitempty"`
	Stock      *int              `json:"stock,omitempty"`
	Image      int               `json:"image,omitempty"`
}

func Decode(raw []byte, in Input) (Candidate, error) {
	var out Candidate
	if len(raw) == 0 || len(raw) > 28<<10 || !utf8.Valid(raw) {
		return out, ErrInvalid
	}
	issues, err := strict.UnmarshalStrict(raw, &out, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
	if err != nil || len(issues) != 0 || out.Validate(in) != nil {
		return Candidate{}, ErrInvalid
	}
	return out, nil
}

func (c Candidate) Validate(in Input) error {
	if !in.Valid() || !text(c.Title, 2000, false) || !text(c.Description, 8192, false) || len(c.Rows) > 256 || len(c.Attributes) > 64 || len(c.Questions) > 12 {
		return ErrInvalid
	}
	if c.Title == "" && c.Description == "" && c.Brand == nil && len(c.Attributes)+len(c.Rows)+len(c.Questions) == 0 {
		return ErrInvalid
	}
	if c.Brand != nil && !c.Brand.valid(in.Notes) {
		return ErrInvalid
	}
	for k, f := range c.Attributes {
		if !text(k, 200, true) || !f.valid(in.Notes) {
			return ErrInvalid
		}
	}
	for _, q := range c.Questions {
		if !text(q, 512, true) {
			return ErrInvalid
		}
	}
	combinations, skus := map[string]bool{}, map[string]bool{}
	for _, row := range c.Rows {
		key, _ := json.Marshal(row.Attributes)
		if !row.valid(in) || len(c.Rows) > 1 && len(row.Attributes) == 0 || combinations[string(key)] || row.SKU != "" && skus[strings.ToLower(row.SKU)] {
			return ErrInvalid
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
var stockEvidence = regexp.MustCompile(`(?i)(?:库存|stock|inventory)\s*[：:=]?\s*([0-9]+)(?:[^0-9.]|$)`)
var priceEvidence = regexp.MustCompile(`(?i)(?:价格|单价|price|[¥￥$€])\s*[：:=]?\s*([0-9]+(?:\.[0-9]+)?)(?:[^0-9.]|$)`)
var currencyEvidence = regexp.MustCompile(`\b([A-Z]{3})\s*[：:=]?\s*([0-9]+(?:\.[0-9]+)?)(?:[^0-9.]|$)`)
var imageEvidence = regexp.MustCompile(`(?i)(?:图片|image|图)\s*#?\s*([0-9]+)(?:[^0-9]|$)`)

func (r Row) valid(in Input) bool {
	if !text(r.Evidence, 512, true) || strings.ContainsAny(r.Evidence, "\n\r;；") || !strings.Contains(in.Notes, r.Evidence) || len(r.Attributes) > 12 || !text(r.SKU, 200, false) {
		return false
	}
	for name, value := range r.Attributes {
		if !text(name, 200, true) || !text(value, 512, true) || name != strings.TrimSpace(name) || value != strings.TrimSpace(value) || strings.ContainsAny(value, ",，、\n") || !containsValue(r.Evidence, value) {
			return false
		}
	}
	return (r.SKU == "" || containsValue(r.Evidence, r.SKU)) &&
		(r.Currency == "" || currencyCode.MatchString(r.Currency) && containsValue(r.Evidence, r.Currency)) &&
		(r.Price == nil || *r.Price >= 0 && !math.IsInf(*r.Price, 0) && r.priceIsExplicit()) &&
		(r.Stock == nil || *r.Stock >= 0 && int64(*r.Stock) <= 9007199254740991 && matchesNumber(stockEvidence, r.Evidence, float64(*r.Stock))) &&
		r.Image >= 0 && r.Image <= in.GalleryCount && (r.Image == 0 || r.Image == in.SelectedImage || matchesNumber(imageEvidence, r.Evidence, float64(r.Image)))
}

func (r Row) priceIsExplicit() bool {
	if matchesNumber(priceEvidence, r.Evidence, *r.Price) {
		return true
	}
	for _, m := range currencyEvidence.FindAllStringSubmatch(r.Evidence, -1) {
		if n, err := strconv.ParseFloat(m[2], 64); err == nil && m[1] == r.Currency && n == *r.Price {
			return true
		}
	}
	return false
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
