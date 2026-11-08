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
		Action   string         `json:"action"`
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
			price := bytes.TrimSpace(v["price"])
			stock := bytes.TrimSpace(v["stock"])
			if len(price) == 0 || bytes.Equal(price, []byte("null")) || bytes.Equal(stock, []byte("null")) || (shape.Action == "import_products" && len(stock) == 0) {
				return false
			}
		}
	}
	return true
}
