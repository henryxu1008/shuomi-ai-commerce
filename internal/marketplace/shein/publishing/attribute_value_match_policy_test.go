package publishing

import "testing"

func TestMatchPlatformAttributeValuePreservesLookupOrder(t *testing.T) {
	cases := []struct {
		name, value, sanitized string
		candidates             map[string]int
		want                   int
	}{
		{"exact", "Black", "Black", map[string]int{"Black": 101, "black": 102}, 101},
		{"case lower", "black", "black", map[string]int{"Black": 101}, 101},
		{"case upper", "BLACK", "BLACK", map[string]int{"Black": 101}, 101},
		{"lower key", "red", "red", map[string]int{"red": 103}, 103},
		{"upper value", "RED", "RED", map[string]int{"red": 103}, 103},
		{"upper key", "yellow", "yellow", map[string]int{"YELLOW": 104}, 104},
		{"missing", "Green", "Green", map[string]int{"Black": 101}, 0},
		{"empty value", "", "", map[string]int{"": 999}, 0},
		{"empty map", "Black", "Black", nil, 0},
		{"raw before sanitized", "Red,Blue", "Red Blue", map[string]int{"Red,Blue": 1, "Red Blue": 2}, 1},
		{"sanitized exact", "Red,Blue", "Red Blue", map[string]int{"Red Blue": 2}, 2},
		{"sanitized lower", "Red,Blue", "Red Blue", map[string]int{"red blue": 3}, 3},
		{"fields normalization", "  Dark   BLUE ", "Dark BLUE", map[string]int{"dark blue": 4}, 4},
		{"no color substitution", "Navy Blue", "Navy Blue", map[string]int{"Blue": 5}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MatchPlatformAttributeValue(tc.value, tc.sanitized, tc.candidates); got != tc.want {
				t.Fatalf("match=%d, want %d", got, tc.want)
			}
		})
	}
}
