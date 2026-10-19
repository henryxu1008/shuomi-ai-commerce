package shein

import (
	"fmt"

	sheinattribute "task-processor/internal/shein/api/attribute"
)

func dedupeCustomAttributeRelations(relations []sheinattribute.CustomAttributeRelation) []sheinattribute.CustomAttributeRelation {
	if len(relations) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(relations))
	result := make([]sheinattribute.CustomAttributeRelation, 0, len(relations))
	for _, relation := range relations {
		key := fmt.Sprintf("%d:%d", relation.PreAttributeValueID, relation.AttributeValueID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, relation)
	}
	return result
}
