package goods

import (
	"sort"
	"strconv"
	"strings"
	"task-processor/internal/marketplace/temu/model"
)

// E27 requires complete combinations. Counts alone cannot detect a repeated
// SKU replacing a missing combination or a spec moved to a different parent.
func checkSupplierSpecifications(input model.SupplierGoods, template model.SupplierAttributeTemplate) []Issue {
	out := []Issue{}
	add := func(code string) {
		out = append(out, Issue{"specifications", code, "选择当前官方规格，并补齐完整SKU组合"})
	}
	definitions := map[int64]model.SupplierProperty{}
	for _, r := range template.Properties {
		if r.IsSale != nil && *r.IsSale && r.ParentSpecID != nil && *r.ParentSpecID > 0 {
			definitions[*r.ParentSpecID] = r
		}
	}
	axes := map[int64]map[int64]model.SupplierSpecification{}
	primary := map[int64]bool{}
	for _, p := range input.SpecProperties {
		if p.ParentSpecID == nil || p.SpecID == nil || *p.ParentSpecID < 1 || *p.SpecID < 1 {
			add("supplier_specification_unverified")
			continue
		}
		r, ok := definitions[*p.ParentSpecID]
		matched := false
		if ok && r.RefPID != nil && p.RefPID != nil && *r.RefPID == *p.RefPID && r.MainSale != nil {
			for _, v := range r.Values {
				if v.SpecID != nil && v.VID != nil && p.VID != nil && *v.SpecID == *p.SpecID && *v.VID == *p.VID && v.Value == p.Value {
					matched = true
				}
			}
		}
		if !matched {
			add("supplier_specification_unverified")
			continue
		}
		if axes[*p.ParentSpecID] == nil {
			axes[*p.ParentSpecID] = map[int64]model.SupplierSpecification{}
		}
		if _, exists := axes[*p.ParentSpecID][*p.SpecID]; exists {
			add("supplier_specification_duplicate")
		}
		axes[*p.ParentSpecID][*p.SpecID] = model.SupplierSpecification{ID: p.SpecID, ParentID: p.ParentSpecID, ParentName: r.Name, Name: p.Value}
		if *r.MainSale {
			primary[*p.ParentSpecID] = true
		}
	}
	axisIDs := make([]int64, 0, len(axes))
	expected := 1
	for id, values := range axes {
		axisIDs = append(axisIDs, id)
		if len(values) == 0 || expected > 10000/len(values) {
			add("supplier_specification_budget")
			return out
		}
		expected *= len(values)
	}
	sort.Slice(axisIDs, func(i, j int) bool { return axisIDs[i] < axisIDs[j] })
	combinations, external, groups := map[string]bool{}, map[string]bool{}, map[string]bool{}
	count := 0
	for _, skc := range input.SKCs {
		main := map[int64]int64{}
		if len(primary) == 0 {
			if len(input.SKCs) != 1 || len(skc.MainSpecs) != 1 || skc.MainSpecs[0].ID == nil || *skc.MainSpecs[0].ID != 0 || skc.MainSpecs[0].ParentID == nil || *skc.MainSpecs[0].ParentID != 0 || skc.MainSpecs[0].Name != "" || skc.MainSpecs[0].ParentName != "" {
				add("supplier_main_specification_invalid")
			}
		} else {
			for _, v := range skc.MainSpecs {
				if v.ID == nil || v.ParentID == nil || !primary[*v.ParentID] {
					add("supplier_main_specification_invalid")
					continue
				}
				expectedSpec, ok := axes[*v.ParentID][*v.ID]
				if !ok || expectedSpec.Name != v.Name || expectedSpec.ParentName != v.ParentName || main[*v.ParentID] != 0 {
					add("supplier_main_specification_invalid")
				}
				main[*v.ParentID] = *v.ID
			}
			if len(main) != len(primary) {
				add("supplier_main_specification_invalid")
			}
		}
		groupKey := []string{}
		for _, id := range axisIDs {
			if primary[id] {
				groupKey = append(groupKey, strconv.FormatInt(main[id], 10))
			}
		}
		group := strings.Join(groupKey, ":")
		if groups[group] {
			add("supplier_main_specification_duplicate")
		}
		groups[group] = true
		if len(skc.SKUs) == 0 {
			add("supplier_sku_missing")
		}
		for _, sku := range skc.SKUs {
			count++
			if !boundedText(sku.ExternalCode, 256) || external[sku.ExternalCode] {
				add("supplier_sku_code_invalid")
			}
			external[sku.ExternalCode] = true
			selected := map[int64]int64{}
			for _, v := range sku.Specs {
				if v.ID == nil || v.ParentID == nil {
					add("supplier_specification_unverified")
					continue
				}
				expectedSpec, ok := axes[*v.ParentID][*v.ID]
				if !ok || expectedSpec.Name != v.Name || expectedSpec.ParentName != v.ParentName || selected[*v.ParentID] != 0 {
					add("supplier_specification_unverified")
				}
				selected[*v.ParentID] = *v.ID
				if primary[*v.ParentID] && main[*v.ParentID] != *v.ID {
					add("supplier_main_specification_mismatch")
				}
			}
			if len(selected) != len(axes) {
				add("supplier_specification_missing")
			}
			keyParts := []string{}
			for _, id := range axisIDs {
				keyParts = append(keyParts, strconv.FormatInt(selected[id], 10))
			}
			key := strings.Join(keyParts, ":")
			if combinations[key] {
				add("supplier_sku_combination_duplicate")
			}
			combinations[key] = true
		}
	}
	if count != expected || len(combinations) != expected {
		add("supplier_sku_combinations_incomplete")
	}
	return out
}
