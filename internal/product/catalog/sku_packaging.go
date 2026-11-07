package catalog

import "math"

// SKUPackaging records one SKU's measured package facts in explicit metric units.
// A missing value is unknown; it is never a declaration of zero.
type SKUPackaging struct {
	WeightGrams *float64 `json:"weightGrams,omitempty"`
	LengthCm    *float64 `json:"lengthCm,omitempty"`
	WidthCm     *float64 `json:"widthCm,omitempty"`
	HeightCm    *float64 `json:"heightCm,omitempty"`
}

func (p *SKUPackaging) Valid() bool {
	if p == nil {
		return true
	}
	for _, value := range []*float64{p.WeightGrams, p.LengthCm, p.WidthCm, p.HeightCm} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value <= 0 || *value > 9007199254740991) {
			return false
		}
	}
	return true
}

func CloneSKUPackaging(p *SKUPackaging) *SKUPackaging {
	if p == nil {
		return nil
	}
	copyNumber := func(value *float64) *float64 {
		if value == nil {
			return nil
		}
		return new(*value)
	}
	return &SKUPackaging{WeightGrams: copyNumber(p.WeightGrams), LengthCm: copyNumber(p.LengthCm), WidthCm: copyNumber(p.WidthCm), HeightCm: copyNumber(p.HeightCm)}
}
