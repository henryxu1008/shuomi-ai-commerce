package publishing

import "strings"

// MatchPlatformAttributeValue preserves the platform lookup order without owning text sanitization.
func MatchPlatformAttributeValue(value, sanitizedValue string, candidates map[string]int) int {
	if value == "" {
		return 0
	}
	if id, ok := candidates[value]; ok {
		return id
	}
	if id, ok := candidates[strings.ToLower(strings.TrimSpace(value))]; ok {
		return id
	}
	if sanitizedValue != value {
		if id, ok := candidates[sanitizedValue]; ok {
			return id
		}
		if id, ok := candidates[strings.ToLower(strings.TrimSpace(sanitizedValue))]; ok {
			return id
		}
	}
	normalize := func(s string) string {
		return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
	}
	normalized := normalize(value)
	for candidate, id := range candidates {
		if normalize(candidate) == normalized {
			return id
		}
	}
	return 0
}
