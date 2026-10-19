package shein

import (
	"strings"
)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", " ", "-", " ", "/", " ")
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func matchesAnyName(name string, candidates []string) bool {
	name = normalizeText(name)
	for _, candidate := range candidates {
		if normalizeText(candidate) == name {
			return true
		}
	}
	return false
}
