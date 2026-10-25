package review

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	"task-processor/internal/product/catalog"
	reviewproposal "task-processor/internal/product/review/proposal"
)

// applyContentPatch computes one complete patch on an isolated Catalog clone.
// Evidence, base/version and authority are checked by Review intake and again
// by its ProductDB transaction. This function never publishes a Product.
func applyContentPatch(base catalog.ProductSnapshot, selection reviewproposal.ContentSelection, changes []reviewproposal.FieldChange) (catalog.ProductSnapshot, error) {
	if !selection.Valid() || len(changes) != len(selection.Fields()) {
		return catalog.ProductSnapshot{}, ErrInvalid
	}
	allowed := make(map[string]bool, 4)
	for _, field := range selection.Fields() {
		allowed[field] = true
	}
	seen := make(map[string]bool, 4)
	snapshot, err := catalog.CloneProductSnapshot(base)
	if err != nil {
		return catalog.ProductSnapshot{}, err
	}
	for _, change := range changes {
		if !allowed[change.Field] || seen[change.Field] {
			return catalog.ProductSnapshot{}, ErrInvalid
		}
		seen[change.Field] = true
		if len(change.Value) > MaxRecordBytes {
			return catalog.ProductSnapshot{}, ErrTooLarge
		}
		switch change.Field {
		case "title":
			if err := ValidateTitle(change.Value); err != nil {
				return catalog.ProductSnapshot{}, err
			}
			snapshot.Title = change.Value
		case "description":
			if change.Value == "" || strings.TrimSpace(change.Value) != change.Value || !utf8.ValidString(change.Value) || strings.IndexFunc(change.Value, func(r rune) bool { return unicode.IsControl(r) && r != '\n' }) >= 0 {
				return catalog.ProductSnapshot{}, ErrInvalid
			}
			snapshot.Description = change.Value
			snapshot.DescriptionFormat = ""
		case "selling_points", "seo_keywords":
			values, err := contentArray(change.Value)
			if err != nil {
				return catalog.ProductSnapshot{}, err
			}
			if change.Field == "selling_points" {
				snapshot.SellingPoints = values
			} else {
				snapshot.SEOKeywords = values
			}
		}
	}
	return snapshot, nil
}

func contentArray(value string) ([]string, error) {
	if !utf8.ValidString(value) {
		return nil, ErrInvalid
	}
	var values []string
	if json.Unmarshal([]byte(value), &values) != nil || len(values) == 0 {
		return nil, ErrInvalid
	}
	for _, item := range values {
		if err := ValidateTitle(item); err != nil {
			return nil, err
		}
	}
	// Use the same compact string-array representation as the browser, while
	// retaining ordinary &, < and > text as data rather than HTML.
	var canonical bytes.Buffer
	encoder := json.NewEncoder(&canonical)
	encoder.SetEscapeHTML(false)
	if encoder.Encode(values) != nil || strings.TrimSuffix(canonical.String(), "\n") != value {
		return nil, ErrInvalid
	}
	return values, nil
}
