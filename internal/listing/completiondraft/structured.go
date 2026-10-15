package completiondraft

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"unicode/utf8"

	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/catalog"
)

const MaxStructuredItems = 256
const maxCandidateNumber = 9007199254740991

type CandidateAttribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type CandidatePrice struct {
	Currency string   `json:"currency"`
	Amount   *float64 `json:"amount"`
}
type CandidateVariant struct {
	SourceIndex *int            `json:"sourceIndex"`
	Price       *CandidatePrice `json:"price"`
	Stock       *int64          `json:"stock"`
}
type StructuredContent struct {
	Attributes []CandidateAttribute `json:"attributes"`
	Variants   []CandidateVariant   `json:"variants"`
}

func validCandidateText(value string, max int) bool {
	return len(value) <= max && utf8.ValidString(value) && !strings.ContainsRune(value, 0)
}
func (c *StructuredContent) validate() error {
	if c == nil {
		return nil
	}
	if c.Attributes == nil || c.Variants == nil || len(c.Attributes) > MaxStructuredItems || len(c.Variants) > MaxStructuredItems {
		return preparation.ErrInvalid
	}
	names := map[string]bool{}
	for _, a := range c.Attributes {
		name := strings.TrimSpace(a.Name)
		if name == "" || names[name] || !validCandidateText(a.Name, 200) || !validCandidateText(a.Value, 8192) {
			return preparation.ErrInvalid
		}
		names[name] = true
	}
	for index, v := range c.Variants {
		if v.SourceIndex == nil || *v.SourceIndex != index || v.Stock != nil && (*v.Stock < 0 || *v.Stock > maxCandidateNumber) {
			return preparation.ErrInvalid
		}
		if p := v.Price; p != nil && (p.Amount == nil || len(p.Currency) != 3 || strings.IndexFunc(p.Currency, func(r rune) bool { return r < 'A' || r > 'Z' }) >= 0 || math.IsNaN(*p.Amount) || math.IsInf(*p.Amount, 0) || *p.Amount < 0 || *p.Amount > maxCandidateNumber) {
			return preparation.ErrInvalid
		}
	}
	return nil
}
func (c Content) matchesSnapshot(product catalog.ProductSnapshot) bool {
	return c.Structured == nil || len(c.Structured.Variants) == len(product.Variants)
}

// EqualContent compares the complete immutable candidate, including pointer and
// array values, rather than their allocation identities.
func EqualContent(a, b Content) bool { return reflect.DeepEqual(a, b) }

func cloneInput(in SaveInput) (SaveInput, error) {
	raw, err := json.Marshal(in)
	if err != nil {
		return SaveInput{}, preparation.ErrInvalid
	}
	var copy SaveInput
	if json.Unmarshal(raw, &copy) != nil {
		return SaveInput{}, preparation.ErrInvalid
	}
	// PostgreSQL JSONB has one numeric zero. Normalize its IEEE sign before
	// hashing; this does not round prices or modify the caller's pointers.
	if copy.Content.Structured != nil {
		for _, v := range copy.Content.Structured.Variants {
			if v.Price != nil && v.Price.Amount != nil && *v.Price.Amount == 0 {
				*v.Price.Amount = 0
			}
		}
	}
	return copy, nil
}
