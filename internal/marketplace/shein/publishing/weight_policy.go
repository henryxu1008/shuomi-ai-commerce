package publishing

import "strings"

// SKUWeightGrams converts a measurement with a recognized original unit.
func SKUWeightGrams(value float64, unit string) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "g", "gram", "grams":
		return value, true
	case "kg", "kilogram", "kilograms":
		return value * 1000, true
	case "lb", "lbs", "pound", "pounds":
		return value * 453.59237, true
	case "oz", "ounce", "ounces":
		return value * 28.349523125, true
	case "mg", "milligram", "milligrams":
		return value / 1000, true
	}
	return 0, false
}

// ValidSKUWeightGrams checks the existing SHEIN submit bounds.
func ValidSKUWeightGrams(grams float64) bool {
	return grams >= 0.01 && grams <= 50000000
}
