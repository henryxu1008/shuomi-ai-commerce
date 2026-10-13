package tiktokshop

import (
	"strconv"
	"strings"
)

func usdCents(value string) (int64, bool) {
	if len(value) == 0 || len(value) > 14 {
		return 0, false
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, false
	}
	for _, part := range parts {
		for _, r := range part {
			if r < '0' || r > '9' {
				return 0, false
			}
		}
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) < 1 || len(parts[1]) > 2 {
			return 0, false
		}
		fraction = parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole > 99999999 {
		return 0, false
	}
	minor, _ := strconv.ParseInt(fraction, 10, 64)
	return whole*100 + minor, true
}

// US prices are exact decimals, with limits from official Product Pricing.
// A SKU routed through multiple warehouses must satisfy every selected type.
func ValidateUSSalePrice(value string, warehouses []Warehouse) error {
	price, valid := usdCents(value)
	if !valid || price < 1 || len(warehouses) == 0 || len(warehouses) > 50 {
		return ErrPublicationInvalid
	}
	seen := map[string]bool{}
	for _, warehouse := range warehouses {
		if !identifier(warehouse.ID) || seen[warehouse.ID] || warehouse.EffectStatus != "ENABLED" || warehouse.Type != "SALES_WAREHOUSE" {
			return ErrPublicationInvalid
		}
		seen[warehouse.ID] = true
		maximum := int64(760000)
		switch warehouse.SubType {
		case "CB_DIRECT_SHIPPING_WAREHOUSE":
			maximum = 80000
		case "CB_OVERSEA_WAREHOUSE":
		default:
			return ErrListingRules
		}
		if price > maximum {
			return ErrPublicationInvalid
		}
	}
	return nil
}

func ValidateGlobalBasePrice(value string) error {
	price, valid := usdCents(value)
	if !valid || price < 1 || price > 9999999999 {
		return ErrPublicationInvalid
	}
	return nil
}
