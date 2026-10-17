package publishing

import (
	"fmt"
	"strings"
)

// These private snapshots retain validation values, without legacy task or API ownership.
type submissionSpec struct {
	parentSpecID string
	specID       string
	name         string
}

type configuredSubmissionSpec struct {
	parentSpecID string
	specID       string
	vid          int
}

func validateSubmissionSpecs(used []submissionSpec, configured []configuredSubmissionSpec) (map[string]map[string]bool, error) {
	dimensions := make(map[string]map[string]bool)
	for _, spec := range used {
		if dimensions[spec.parentSpecID] == nil {
			dimensions[spec.parentSpecID] = make(map[string]bool)
		}
		dimensions[spec.parentSpecID][spec.specID] = true
	}

	configuredSpecs := make(map[string]map[string]bool)
	incompleteSpecs := make(map[string]map[string]string)
	for _, spec := range configured {
		if configuredSpecs[spec.parentSpecID] == nil {
			configuredSpecs[spec.parentSpecID] = make(map[string]bool)
			incompleteSpecs[spec.parentSpecID] = make(map[string]string)
		}
		configuredSpecs[spec.parentSpecID][spec.specID] = true
		if spec.vid == 0 {
			incompleteSpecs[spec.parentSpecID][spec.specID] = "missing_vid"
		}
	}

	var missing, incomplete []string
	for parentID, specIDs := range dimensions {
		for specID := range specIDs {
			if configuredSpecs[parentID] == nil || !configuredSpecs[parentID][specID] {
				name := submissionSpecName(used, parentID, specID)
				missing = append(missing, fmt.Sprintf("%s(%s)", name, specID))
			} else if incompleteSpecs[parentID][specID] != "" {
				name := submissionSpecName(used, parentID, specID)
				incomplete = append(incomplete, fmt.Sprintf("%s(%s)-%s", name, specID, incompleteSpecs[parentID][specID]))
			}
		}
	}
	if len(missing) > 0 || len(incomplete) > 0 {
		return dimensions, fmt.Errorf("缺失或不完整的规格配置: missing=%v, incomplete=%v", missing, incomplete)
	}
	return dimensions, nil
}

func submissionSpecName(used []submissionSpec, parentID, specID string) string {
	for _, spec := range used {
		if spec.parentSpecID == parentID && spec.specID == specID {
			return spec.name
		}
	}
	return fmt.Sprintf("Unknown_%s", specID)
}

// validateSubmissionPackaging reports the first Multiple Sets property even when valid.
func validateSubmissionPackaging(properties []string, pieces []int) (invalid bool, propertyIndex int) {
	propertyIndex = -1
	for i, value := range properties {
		if strings.Contains(strings.ToLower(value), "multiple sets") {
			propertyIndex = i
			break
		}
	}
	if propertyIndex == -1 {
		return false, -1
	}
	for _, count := range pieces {
		if count <= 1 {
			return true, propertyIndex
		}
	}
	return false, propertyIndex
}
