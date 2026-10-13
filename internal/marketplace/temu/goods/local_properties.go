package goods

import (
	"math/big"
	"strconv"
	"strings"
	"task-processor/internal/marketplace/temu/model"
	"time"
	"unicode/utf8"
)

// E36 controls are checked against the observed template and the value groups
// activated by selected parents. Unknown controls cannot grant readiness.
func CheckLocalProperties(input []model.LocalProperty, template model.LocalAttributeTemplate) []Issue {
	issues := []Issue{}
	add := func(field, code, message string) { issues = append(issues, Issue{field, code, message}) }
	known := map[int64]model.LocalAttribute{}
	children := map[int64]bool{}
	active := map[int64]map[string]bool{}
	selected := map[int64][]model.LocalProperty{}
	for _, v := range input {
		selected[v.RefPID] = append(selected[v.RefPID], v)
	}
	for _, d := range template.Attributes {
		if d.RefPID == nil || *d.RefPID < 1 || d.Required == nil {
			add("attributes", "attribute_rule_unverified", "属性标识或必填规则未核实")
			continue
		}
		id := *d.RefPID
		if _, dup := known[id]; dup {
			add("attributes", "attribute_rule_unverified", "属性模板含重复标识")
		}
		known[id] = d
		for _, g := range d.ValueGroups {
			for _, p := range g.Values {
				for _, c := range p.ChildAttributes {
					if c.RefPID == nil || *c.RefPID < 1 || c.ValueGroupID == "" {
						add("attributes", "attribute_rule_unverified", "子属性值组未核实")
						continue
					}
					children[*c.RefPID] = true
					for _, v := range selected[id] {
						if v.VID != nil && p.VID != nil && *v.VID == *p.VID {
							if active[*c.RefPID] == nil {
								active[*c.RefPID] = map[string]bool{}
							}
							active[*c.RefPID][c.ValueGroupID] = true
						}
					}
				}
			}
		}
	}
	for _, d := range template.Attributes {
		if d.RefPID == nil || d.Required == nil {
			continue
		}
		id := *d.RefPID
		values := selected[id]
		path := "attributes." + strconv.FormatInt(id, 10)
		enabled := !children[id] || len(active[id]) > 0
		if enabled && (d.Rules == nil || d.Rules.Transnational == nil) {
			add(path, "attribute_rule_unverified", "跨境必填属性标志未核实")
		}
		if !enabled && len(values) > 0 {
			add(path, "attribute_child_inactive", "属性尚未由所选父值启用")
		}
		if enabled && (*d.Required || d.Rules != nil && d.Rules.Transnational != nil && *d.Rules.Transnational) && len(values) == 0 {
			add(path, "attribute_missing", "补充当前模板要求的必填属性")
		}
		if len(values) == 0 {
			continue
		}
		if d.Rules == nil || d.Rules.ControlType == nil || d.Rules.ChooseMaxNum == nil || *d.Rules.ChooseMaxNum < 1 {
			add(path, "attribute_rule_unverified", "属性输入及选择规则未核实")
			continue
		}
		if int64(len(values)) > *d.Rules.ChooseMaxNum {
			add(path, "attribute_selection_count_invalid", "属性选择数量不符合模板")
		}
		seen := map[string]bool{}
		sum := new(big.Rat)
		sumValid := true
		for _, v := range values {
			identity := v.Value
			if v.VID != nil {
				identity = "vid:" + strconv.FormatInt(*v.VID, 10)
			}
			if seen[identity] {
				add(path, "attribute_value_duplicate", "同一属性不能重复选择相同值")
			}
			seen[identity] = true
			preset := false
			if v.VID != nil {
				for _, g := range d.ValueGroups {
					if children[id] && !active[id][g.ID] {
						continue
					}
					for _, p := range g.Values {
						if p.VID != nil && *p.VID == *v.VID {
							preset = true
							if v.Value != "" && v.Value != p.Value {
								add(path, "attribute_value_mismatch", "属性名称必须与当前预设值一致")
							}
						}
					}
				}
				if !preset {
					add(path, "attribute_value_not_in_template", "选择当前父值和模板允许的属性值")
				}
			}
			control := *d.Rules.ControlType
			text := v.Value
			var valid bool
			switch control {
			case 0:
				valid = v.VID == nil && boundedText(text, 2<<20) && v.NumberInputValue == ""
			case 1:
				valid = preset && v.NumberInputValue == ""
			case 5, 6, 7, 8:
				valid = v.VID == nil && validAttributeDate(text, control) && v.NumberInputValue == ""
			case 16:
				text = v.NumberInputValue
				valid = preset && text != ""
			case 19:
				valid = (preset || v.VID == nil && boundedText(text, 2<<20)) && v.NumberInputValue == ""
			default:
				add(path, "attribute_control_unverified", "尚未核实此属性输入类型")
				valid = false
			}
			if !valid {
				add(path, "attribute_control_invalid", "按当前属性的输入或选择类型填写")
			}
			if control != 1 && !(control == 19 && preset) && !checkAttributeValue(text, control, d.Rules.Values) {
				add(path, "attribute_value_invalid", "属性值不符合精度、范围或日期要求")
			}
			if len(d.Units) > 0 {
				unit := false
				for _, u := range d.Units {
					if u.ID != nil && v.ValueUnitID != nil && *u.ID == *v.ValueUnitID {
						unit = true
					}
				}
				if !unit {
					add(path, "attribute_unit_unverified", "选择当前属性模板中的单位")
				}
			} else if v.ValueUnitID != nil {
				add(path, "attribute_unit_unverified", "单位不属于当前属性模板")
			}
			if d.Rules.Values != nil && d.Rules.Values.ValueRule != nil {
				switch *d.Rules.Values.ValueRule {
				case 1:
					n, ok := decimalRat(text)
					if !ok {
						sumValid = false
					} else {
						sum.Add(sum, n)
					}
				case 2:
					add(path, "attribute_value_rule_unverified", "该字符规则尚缺明确的允许字符范围")
				default:
					add(path, "attribute_value_rule_unverified", "属性数值规则尚未核实")
				}
			}
		}
		if d.Rules.Values != nil && d.Rules.Values.ValueRule != nil && *d.Rules.Values.ValueRule == 1 && (!sumValid || sum.Cmp(big.NewRat(100, 1)) != 0) {
			add(path, "attribute_sum_invalid", "组成比例之和必须准确等于100")
		}
	}
	for _, v := range input {
		if _, ok := known[v.RefPID]; !ok {
			add("attributes", "attribute_not_in_template", "属性不属于当前模板")
		}
	}
	return issues
}

func decimalRat(value string) (*big.Rat, bool) {
	if value == "" || len(value) > 128 {
		return nil, false
	}
	digits := 0
	dots := 0
	for i, r := range value {
		if r == '.' {
			dots++
			if dots > 1 || i == 0 || i == len(value)-1 {
				return nil, false
			}
		} else if r < '0' || r > '9' {
			return nil, false
		} else {
			digits++
		}
	}
	if digits == 0 {
		return nil, false
	}
	v, ok := new(big.Rat).SetString(value)
	return v, ok
}

func validAttributeDate(value string, control int64) bool {
	format := "2006-01-02"
	if control == 7 || control == 8 {
		format = "2006-01"
	}
	parts := strings.Split(value, ",")
	count := 1
	if control == 6 || control == 8 {
		count = 2
	}
	if len(parts) != count {
		return false
	}
	var previous time.Time
	for _, part := range parts {
		current, err := time.Parse(format, part)
		if err != nil || current.Format(format) != part || !previous.IsZero() && current.Before(previous) {
			return false
		}
		previous = current
	}
	return true
}

func checkAttributeValue(value string, control int64, rule *model.LocalAttributeValueRules) bool {
	if rule == nil {
		return true
	}
	numeric := control == 16 || rule.Precision != nil
	if numeric {
		n, ok := decimalRat(value)
		if !ok {
			return false
		}
		if rule.Precision != nil {
			if *rule.Precision < 0 {
				return false
			}
			parts := strings.Split(value, ".")
			if len(parts) == 2 && int64(len(parts[1])) > *rule.Precision {
				return false
			}
		}
		for i, bound := range []*string{rule.Minimum, rule.Maximum} {
			if bound != nil {
				v, ok := decimalRat(*bound)
				if !ok || i == 0 && n.Cmp(v) < 0 || i == 1 && n.Cmp(v) > 0 {
					return false
				}
			}
		}
		return true
	}
	if control == 5 || control == 6 || control == 7 || control == 8 {
		for _, part := range strings.Split(value, ",") {
			if rule.Minimum != nil && part < *rule.Minimum || rule.Maximum != nil && part > *rule.Maximum {
				return false
			}
		}
		return true
	}
	for i, bound := range []*string{rule.Minimum, rule.Maximum} {
		if bound != nil {
			v, err := strconv.ParseInt(*bound, 10, 64)
			if err != nil || v < 0 || i == 0 && int64(utf8.RuneCountInString(value)) < v || i == 1 && int64(utf8.RuneCountInString(value)) > v {
				return false
			}
		}
	}
	return utf8.ValidString(value)
}
