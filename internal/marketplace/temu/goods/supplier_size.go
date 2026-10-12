package goods

import (
	"strconv"
	"task-processor/internal/marketplace/temu/model"
)

func checkSupplierSize(input model.SupplierGoods, snapshot RuleSnapshot) []Issue {
	invalid := func(code string) []Issue {
		return []Issue{{"sizeTemplateIds", code, "选择本类目已有有效尺码表；需要创建时保持待补全"}}
	}
	required := snapshot.SupplierSizeRequirement
	if required == nil || !required.Observed || required.CatID != snapshot.Selection.CatID {
		return invalid("supplier_size_rules_unverified")
	}
	if required.Class == nil {
		if len(input.SizeTemplateIDs) > 0 || len(input.ShowSizeIDs) > 0 {
			return invalid("supplier_size_chart_unverified")
		}
		return nil
	}
	class := required.Class
	if class.CatID == nil || *class.CatID != snapshot.Selection.CatID || class.ClassID == nil || *class.ClassID < 1 || class.ClassType == nil || *class.ClassType < 0 || *class.ClassType > 1 {
		return invalid("supplier_size_rules_unverified")
	}
	if len(input.SizeTemplateIDs) == 0 || *class.ClassType == 1 && len(input.SizeTemplateIDs) < 2 || snapshot.SupplierSizeCharts == nil || snapshot.SupplierAttributes == nil {
		return invalid("supplier_size_chart_missing")
	}
	sizes := map[string]bool{}
	for _, r := range snapshot.SupplierAttributes.Properties {
		if r.IsSale == nil || !*r.IsSale || r.ControlType == nil || *r.ControlType != 10 || r.RefPID == nil {
			continue
		}
		for _, v := range input.SpecProperties {
			if v.RefPID != nil && *v.RefPID == *r.RefPID && v.Value != "" {
				sizes[v.Value] = true
			}
		}
	}
	if len(sizes) == 0 {
		return invalid("supplier_size_specification_unverified")
	}
	selected := map[int64]bool{}
	charts := append([]model.SupplierSizeChart{}, snapshot.SupplierSizeCharts.Charts...)
	for _, page := range snapshot.SupplierSizeChartPages {
		charts = append(charts, page.Charts...)
	}
	for _, id := range input.SizeTemplateIDs {
		if id < 1 || selected[id] {
			return invalid("supplier_size_chart_unverified")
		}
		selected[id] = true
		var chart *model.SupplierSizeChart
		for index := range charts {
			v := &charts[index]
			if v.ID != nil && *v.ID == id {
				if chart != nil {
					return invalid("supplier_size_chart_unverified")
				}
				chart = v
			}
		}
		if chart == nil || chart.ClassID == nil || chart.Reusable == nil || *chart.Reusable || chart.Content == nil || len(chart.Content.Records) != len(sizes) {
			return invalid("supplier_size_chart_unverified")
		}
		allowed := *class.ClassType == 0 && *chart.ClassID == *class.ClassID
		if *class.ClassType == 1 {
			for _, related := range class.RelatedClassIDs {
				if related == *chart.ClassID {
					allowed = true
				}
			}
		}
		if !allowed {
			return invalid("supplier_size_chart_unverified")
		}
		var settings *model.SupplierSizeSettings
		for index := range snapshot.SupplierSizeSettings {
			observed := &snapshot.SupplierSizeSettings[index]
			if observed.ClassID == *chart.ClassID {
				if settings != nil {
					return invalid("supplier_size_settings_unverified")
				}
				settings = &observed.Settings
			}
		}
		if settings == nil || !supplierSizeChartRowsMatch(*chart, *settings, sizes) {
			return invalid("supplier_size_chart_unverified")
		}
	}
	for _, id := range input.ShowSizeIDs {
		if !selected[id] {
			return invalid("supplier_size_chart_unverified")
		}
	}
	return nil
}

// Use current official group IDs and mapping tuples. A measurement whose
// value happens to equal a SKU size label is not a size-group association.
func supplierSizeChartRowsMatch(chart model.SupplierSizeChart, settings model.SupplierSizeSettings, sizes map[string]bool) bool {
	if chart.Content == nil || settings.Mapping == nil || settings.Mapping.Meta == nil || len(settings.Mapping.Meta.Groups) == 0 {
		return false
	}
	groups := map[string]bool{}
	for _, g := range settings.Mapping.Meta.Groups {
		if g.ID == nil || *g.ID < 1 {
			return false
		}
		groups[strconv.FormatInt(*g.ID, 10)] = g.Unnecessary == nil || !*g.Unnecessary
	}
	templates := map[string]map[string]string{}
	for size := range sizes {
		allowed := false
		for _, v := range settings.Sizes {
			if v == size {
				allowed = true
			}
		}
		if !allowed {
			return false
		}
		for _, mapping := range settings.Mapping.Records {
			matches := false
			for key := range groups {
				if mapping.Values[key] == size {
					matches = true
				}
			}
			if !matches {
				continue
			}
			if templates[size] != nil {
				return false
			}
			tuple := map[string]string{}
			for key, required := range groups {
				value, exists := mapping.Values[key]
				if required && (!exists || value == "") {
					return false
				}
				if exists && value != "" {
					tuple[key] = value
				}
			}
			if len(tuple) == 0 {
				return false
			}
			templates[size] = tuple
		}
		if templates[size] == nil {
			return false
		}
	}
	seen := map[string]bool{}
	for _, row := range chart.Content.Records {
		size := ""
		for candidate, tuple := range templates {
			matches := true
			for key, value := range tuple {
				if row.Values[key] != value {
					matches = false
				}
			}
			if matches {
				if size != "" {
					return false
				}
				size = candidate
			}
		}
		if size == "" || seen[size] {
			return false
		}
		seen[size] = true
	}
	return len(seen) == len(sizes)
}
