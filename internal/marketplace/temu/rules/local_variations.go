package rules

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"task-processor/internal/marketplace/temu/model"
)

type LocalVariationIssue struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}
type LocalVariationCheck struct {
	Issues []LocalVariationIssue `json:"issues"`
}

// CheckLocalVariations checks E17/E28/E39 against the current category template.
// It never creates custom specifications, rewrites facts, treats an announcement
// as a different variationType, or proves merchant/publishing readiness.
func CheckLocalVariations(catID int64, template model.LocalVariationTemplate, skus []model.LocalSKU) LocalVariationCheck {
	check := LocalVariationCheck{Issues: []LocalVariationIssue{}}
	issue := func(field, code string) { check.Issues = append(check.Issues, LocalVariationIssue{field, code}) }
	if catID < 1 || template.CatID == nil || *template.CatID != catID {
		issue("variations", "category_mismatch")
		return check
	}
	raw, err := json.Marshal(template)
	if err != nil || len(raw) > 2<<20 || template.Variations == nil {
		issue("variations", "template_unverified")
		return check
	}
	if len(skus) < 1 || len(skus) > 500 {
		issue("skuList", "sku_count_invalid")
		return check
	}
	// Bound all custom text together before building combination keys. A
	// per-value cap would still allow 500 individually large SKU names.
	textBytes := 0
	for _, sku := range skus {
		if len(sku.Specifications) > 2 {
			continue // The bounded SKU loop below reports the invalid count.
		}
		for _, value := range sku.Specifications {
			if len(value.Name) > (2<<20)-textBytes {
				issue("skuList", "variation_input_too_large")
				return check
			}
			textBytes += len(value.Name)
		}
	}
	parents := make(map[int64]model.LocalVariation, len(template.Variations))
	presetParents := make(map[int64][]int64)
	for index, parent := range template.Variations {
		path := "variations." + strconv.Itoa(index)
		if parent.ParentSpecID == nil || *parent.ParentSpecID < 1 {
			issue(path, "parent_unverified")
			return check
		}
		id := *parent.ParentSpecID
		if _, duplicate := parents[id]; duplicate {
			issue(path, "parent_duplicated")
			return check
		}
		parents[id] = parent
		if parent.Required == nil {
			issue(path, "requirement_unverified")
		}
		seen := map[int64]bool{}
		for _, value := range parent.Values {
			if value.SpecID == nil || *value.SpecID < 1 || seen[*value.SpecID] {
				issue(path, "preset_template_unverified")
				return check
			}
			seen[*value.SpecID] = true
			presetParents[*value.SpecID] = append(presetParents[*value.SpecID], id)
		}
	}
	type selected struct {
		Parent int64
		Preset int64
		Custom string
	}
	var firstParents string
	combinations := map[string]bool{}
	for index, sku := range skus {
		path := "skuList." + strconv.Itoa(index) + ".specDetails"
		if len(sku.Specifications) < 1 || len(sku.Specifications) > 2 {
			issue(path, "parent_count_invalid")
			continue
		}
		start := len(check.Issues)
		used := map[int64]bool{}
		values := make([]selected, 0, len(sku.Specifications))
		for specIndex, input := range sku.Specifications {
			field := path + "." + strconv.Itoa(specIndex)
			var parentID int64
			if input.ParentSpecID != nil {
				parentID = *input.ParentSpecID
			} else if input.SpecID != nil && len(presetParents[*input.SpecID]) == 1 {
				// Resolve the observed preset's parent for checking only. The
				// optional parent field in the exact wire input stays absent.
				parentID = presetParents[*input.SpecID][0]
			}
			parent, found := parents[parentID]
			if !found {
				issue(field, "parent_unverified")
				continue
			}
			if used[parentID] {
				issue(field, "parent_duplicated")
				continue
			}
			used[parentID] = true
			if parent.Type == nil || *parent.Type < 0 || *parent.Type > 2 {
				issue(field, "variation_type_unverified")
				continue
			}
			value := selected{Parent: parentID}
			if input.SpecID != nil {
				if *parent.Type == 1 {
					issue(field, "preset_not_allowed")
					continue
				}
				var preset *model.LocalVariationValue
				for i := range parent.Values {
					if *parent.Values[i].SpecID == *input.SpecID {
						preset = &parent.Values[i]
						break
					}
				}
				if preset == nil {
					issue(field, "preset_not_in_template")
					continue
				}
				if input.Name != "" && input.Name != preset.Name {
					issue(field, "preset_name_mismatch")
				}
				value.Preset = *input.SpecID
			} else {
				if *parent.Type == 0 {
					issue(field, "custom_not_allowed")
					continue
				}
				if !utf8.ValidString(input.Name) || strings.TrimSpace(input.Name) == "" || len(input.Name) > 2<<20 {
					issue(field, "custom_value_missing")
					continue
				}
				value.Custom = input.Name
			}
			values = append(values, value)
		}
		for id, parent := range parents {
			if parent.Required != nil && *parent.Required && !used[id] {
				issue(path, "required_parent_missing")
			}
		}
		if len(check.Issues) != start {
			continue
		}
		sort.Slice(values, func(i, j int) bool { return values[i].Parent < values[j].Parent })
		ids := make([]int64, len(values))
		for i, value := range values {
			ids[i] = value.Parent
		}
		parentKey, _ := json.Marshal(ids)
		if firstParents == "" {
			firstParents = string(parentKey)
		} else if firstParents != string(parentKey) {
			issue(path, "sku_parents_inconsistent")
		}
		key, _ := json.Marshal(values)
		if combinations[string(key)] {
			issue(path, "sku_variation_duplicated")
		}
		combinations[string(key)] = true
	}
	return check
}
