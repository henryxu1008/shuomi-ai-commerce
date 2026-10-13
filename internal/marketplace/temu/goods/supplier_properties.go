package goods

import (
	"math/big"
	"strconv"
	"task-processor/internal/marketplace/temu/model"
)

// E35's current IDs and parent conditions are authoritative. Saving a value
// preserves it; it does not create a missing specification or legal approval.
func CheckSupplierProperties(input model.SupplierGoods, template model.SupplierAttributeTemplate) []Issue {
	out := []Issue{}
	add := func(field, code string) {
		out = append(out, Issue{field, code, "核对当前供应商属性模板和明确资料"})
	}
	selected := map[int64][]model.SupplierBasicProperty{}
	for _, value := range input.Properties {
		if value.RefPID == nil {
			add("attributes", "supplier_attribute_unverified")
			continue
		}
		selected[*value.RefPID] = append(selected[*value.RefPID], value)
	}
	for _, value := range input.SpecProperties {
		if value.RefPID == nil {
			add("specifications", "supplier_attribute_unverified")
			continue
		}
		selected[*value.RefPID] = append(selected[*value.RefPID], value.SupplierBasicProperty)
	}
	known := map[int64]bool{}
	for _, rule := range template.Properties {
		if rule.RefPID == nil || rule.PID == nil || rule.TemplatePID == nil || rule.Required == nil || rule.IsSale == nil {
			add("attributes", "supplier_attribute_rule_unverified")
			continue
		}
		id := *rule.RefPID
		known[id] = true
		values := selected[id]
		field := "attributes." + strconv.FormatInt(id, 10)
		enabled := true
		for _, condition := range rule.ShowConditions {
			matched := false
			if condition.ParentRefPID != nil {
				for _, parent := range selected[*condition.ParentRefPID] {
					for _, vid := range condition.ParentVIDs {
						if parent.VID != nil && *parent.VID == vid {
							matched = true
						}
					}
				}
			}
			enabled = enabled && matched
		}
		if !enabled {
			if len(values) > 0 {
				add(field, "supplier_attribute_parent_inactive")
			}
			continue
		}
		if *rule.Required && len(values) == 0 {
			add(field, "supplier_attribute_missing")
		}
		if len(values) == 0 {
			continue
		}
		if rule.ControlType == nil {
			add(field, "supplier_attribute_rule_unverified")
			continue
		}
		if rule.ChooseMaxNum != nil && *rule.ChooseMaxNum >= 0 && int64(len(values)) > *rule.ChooseMaxNum && *rule.ControlType != 0 {
			add(field, "supplier_attribute_count_invalid")
		}
		seen := map[string]bool{}
		sum := new(big.Rat)
		sumValid := true
		for _, value := range values {
			if value.PID == nil || *value.PID != *rule.PID || value.TemplatePID == nil || *value.TemplatePID != *rule.TemplatePID || value.Name != rule.Name {
				add(field, "supplier_attribute_identity_mismatch")
			}
			identity := value.Value
			preset := false
			if value.VID != nil && *value.VID > 0 {
				identity = "vid:" + strconv.FormatInt(*value.VID, 10)
				for _, official := range rule.Values {
					if official.VID != nil && *official.VID == *value.VID && official.Value == value.Value {
						parentOK := len(official.ParentVIDs) == 0
						for _, parentValues := range selected {
							for _, parent := range parentValues {
								for _, vid := range official.ParentVIDs {
									if parent.VID != nil && *parent.VID == vid {
										parentOK = true
									}
								}
							}
						}
						preset = parentOK
					}
				}
				if !preset {
					add(field, "supplier_attribute_value_unverified")
				}
			}
			if seen[identity] {
				add(field, "supplier_attribute_duplicate")
			}
			seen[identity] = true
			unitOK := len(rule.ValueUnits) == 0 && value.Unit == ""
			for _, unit := range rule.ValueUnits {
				if unit == value.Unit {
					unitOK = true
				}
			}
			if !unitOK {
				add(field, "supplier_attribute_unit_invalid")
			}
			switch *rule.ControlType {
			case 1, 9, 10:
				if !preset {
					add(field, "supplier_attribute_value_unverified")
				}
			case 0, 3:
				if !preset && (rule.InputMaxNum == nil || *rule.InputMaxNum < 1 || !boundedText(value.Value, 16<<10)) {
					add(field, "supplier_attribute_input_unverified")
				}
				if !preset && rule.PropertyValueType == nil {
					add(field, "supplier_attribute_input_unverified")
				}
			case 16:
				if !preset || rule.ValuePrecision == nil {
					add(field, "supplier_attribute_value_unverified")
				}
				r := &model.LocalAttributeValueRules{Minimum: rule.MinValue, Maximum: rule.MaxValue, Precision: rule.ValuePrecision}
				if !checkAttributeValue(value.NumberInputValue, 16, r) {
					add(field, "supplier_attribute_number_invalid")
				}
				n, ok := decimalRat(value.NumberInputValue)
				if !ok {
					sumValid = false
				} else {
					sum.Add(sum, n)
				}
			default:
				add(field, "supplier_attribute_control_unverified")
			}
		}
		if rule.ValueRule != nil && *rule.ValueRule == 1 && (!sumValid || sum.Cmp(big.NewRat(100, 1)) != 0) {
			add(field, "supplier_attribute_sum_invalid")
		}
		if *rule.IsSale {
			for _, property := range input.SpecProperties {
				if property.RefPID == nil || *property.RefPID != id {
					continue
				}
				matched := false
				for _, value := range rule.Values {
					if value.VID != nil && property.VID != nil && *value.VID == *property.VID && value.SpecID != nil && property.SpecID != nil && *value.SpecID == *property.SpecID && rule.ParentSpecID != nil && property.ParentSpecID != nil && *rule.ParentSpecID == *property.ParentSpecID {
						matched = true
					}
				}
				if !matched {
					add(field, "supplier_specification_unverified")
				}
			}
		}
	}
	for id := range selected {
		if !known[id] {
			add("attributes."+strconv.FormatInt(id, 10), "supplier_attribute_unverified")
		}
	}
	return out
}

func checkSupplierPackaging(input model.SupplierGoods) []Issue {
	out := []Issue{}
	for ci, skc := range input.SKCs {
		for si, sku := range skc.SKUs {
			field := "productSkcReqs." + strconv.Itoa(ci) + ".productSkuReqs." + strconv.Itoa(si)
			p := sku.Packaging
			if p == nil || p.Weight == nil || p.Weight.ValueMG == nil || p.Volume == nil || p.Volume.LengthMM == nil || p.Volume.WidthMM == nil || p.Volume.HeightMM == nil || p.Sensitive == nil || p.SensitiveLimits == nil || p.Sensitive.IsSensitive == nil {
				out = append(out, Issue{field, "supplier_packaging_missing", "补充明确的mg重量、mm尺寸和敏感属性"})
				continue
			}
			if *p.Weight.ValueMG < 0 || *p.Volume.LengthMM < 0 || *p.Volume.WidthMM < 0 || *p.Volume.HeightMM < 0 || *p.Sensitive.IsSensitive < 0 || *p.Sensitive.IsSensitive > 1 {
				out = append(out, Issue{field, "supplier_packaging_invalid", "包装数值或敏感属性不符合官方字段合同"})
			}
		}
	}
	return out
}
