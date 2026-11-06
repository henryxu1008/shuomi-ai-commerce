package publishing

import "testing"

func TestSKUWeightRequiresKnownUnitAndRespectsSubmitBounds(t *testing.T) {
	for _, test := range []struct {
		value float64
		unit  string
		want  float64
		valid bool
	}{
		{1, "g", 1, true},
		{1, "kilograms", 1000, true},
		{1, " LB ", 453.59237, true},
		{1, "oz", 28.349523125, true},
		{1, "milligrams", 0.001, false},
		{100000, "kg", 100000000, false},
		{0, "g", 0, false},
	} {
		got, known := SKUWeightGrams(test.value, test.unit)
		if !known || got != test.want || ValidSKUWeightGrams(got) != test.valid {
			t.Errorf("%v %q produced %v g, known=%v", test.value, test.unit, got, known)
		}
	}
	for _, unit := range []string{"", "ml", "unknown"} {
		if _, known := SKUWeightGrams(1, unit); known {
			t.Errorf("accepted %q", unit)
		}
	}
}
