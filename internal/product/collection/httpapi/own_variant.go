package httpapi

import (
	"bytes"
	"encoding/json"
)

// Called only after the existing strict parser has rejected unknown/duplicate fields.
func ownVariantNumbersDeclared(raw []byte) bool {
	type productShape struct {
		Variants []map[string]json.RawMessage `json:"variants"`
	}
	var shape struct {
		Product  *productShape  `json:"product"`
		Products []productShape `json:"products"`
	}
	if json.Unmarshal(raw, &shape) != nil {
		return false
	}
	products := shape.Products
	if shape.Product != nil {
		products = append(products, *shape.Product)
	}
	for _, p := range products {
		for _, v := range p.Variants {
			for _, field := range []string{"price", "stock"} {
				value := bytes.TrimSpace(v[field])
				if len(value) == 0 || bytes.Equal(value, []byte("null")) {
					return false
				}
			}
		}
	}
	return true
}
