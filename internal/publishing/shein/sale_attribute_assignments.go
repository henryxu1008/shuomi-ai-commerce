package shein

import "strings"

func lookupAttributeValue(attributes map[string]string, candidateName string) string {
	if len(attributes) == 0 || strings.TrimSpace(candidateName) == "" {
		return ""
	}
	for key, value := range attributes {
		if matchesAnyName(key, []string{candidateName}) && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
