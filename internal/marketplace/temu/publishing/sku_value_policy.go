package publishing

import "strings"

func deduplicateSKUSpecs(specs []submissionSpec) []submissionSpec {
	if len(specs) <= 1 {
		return specs
	}
	seen := make(map[string]bool)
	var result []submissionSpec
	for _, spec := range specs {
		if !seen[spec.parentSpecID] {
			seen[spec.parentSpecID] = true
			result = append(result, spec)
		}
	}
	return result
}

func hasUnresolvedSKUSpecs(specs []submissionSpec) bool {
	for _, spec := range specs {
		if strings.HasPrefix(spec.specID, "TEMP_") {
			return true
		}
	}
	return false
}

func normalizeSingleItemPackaging(classification, pieces, individuallyPacked int) (int, int) {
	if classification == 1 {
		return 1, 1
	}
	return pieces, individuallyPacked
}
