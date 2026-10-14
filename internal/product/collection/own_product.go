package collection

import (
	"math"
	"net/url"
	"strings"
	"task-processor/internal/product/sourcing"
	"unicode/utf8"
)

func validateOwnProduct(input OwnProduct) error {
	validString := func(value string, max int) bool {
		return len(value) <= max && utf8.ValidString(value) && !strings.ContainsRune(value, 0)
	}
	validAttributes := func(values map[string]string) bool {
		if len(values) > 256 {
			return false
		}
		for k, v := range values {
			if !validString(k, 200) || !validString(v, 8192) {
				return false
			}
		}
		return true
	}
	validImages := func(values []string) bool {
		if len(values) > 40 {
			return false
		}
		for _, v := range values {
			u, err := url.Parse(v)
			if err != nil || !validString(v, 2048) || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
				return false
			}
		}
		return true
	}
	if strings.TrimSpace(input.Title) == "" || !validString(input.Title, 2000) || !validString(input.Description, sourcing.MaxSourceEnvelopeStringBytes) || !validString(input.Brand, 2000) || !validAttributes(input.Attributes) || !validImages(input.Images) || len(input.Variants) > sourcing.MaxSourceEnvelopeCollectionItems {
		return ErrInvalid
	}
	items := len(input.Attributes) + len(input.Images) + len(input.Variants)
	ids, skus := map[string]bool{}, map[string]bool{}
	for _, v := range input.Variants {
		sku := strings.TrimSpace(v.SKU)
		if strings.TrimSpace(v.SourceID) == "" || !validString(v.SourceID, 128) || ids[v.SourceID] || !validString(v.Title, 2000) || !validString(v.SKU, 200) || sku != "" && skus[sku] || !validAttributes(v.Attributes) || !validImages(v.Images) || math.IsNaN(v.Price) || math.IsInf(v.Price, 0) || v.Price < 0 || v.Stock < 0 || int64(v.Stock) > 9007199254740991 || len(v.Currency) != 3 || strings.IndexFunc(v.Currency, func(r rune) bool { return r < 'A' || r > 'Z' }) >= 0 {
			return ErrInvalid
		}
		ids[v.SourceID] = true
		if sku != "" {
			skus[sku] = true
		}
		items += len(v.Attributes) + len(v.Images)
		if items > sourcing.MaxSourceEnvelopeAggregateItems {
			return ErrInvalid
		}
	}
	if items > sourcing.MaxSourceEnvelopeAggregateItems {
		return ErrInvalid
	}
	return nil
}
