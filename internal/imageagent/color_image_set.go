package imageagent

import (
	"maps"
	"slices"
	"strings"
)

type ImageColorOption struct {
	Name   string            `json:"name"`
	Groups []ImageColorGroup `json:"groups"`
}

type ImageColorGroup struct {
	Value       string   `json:"value"`
	OriginalIDs []string `json:"originalIds"`
}

type ColorImageSelection struct {
	Option string               `json:"option"`
	Groups []ColorImageOriginal `json:"groups"`
}

type ColorImageOriginal struct {
	Color      string `json:"color"`
	OriginalID string `json:"originalId"`
}

type colorImagePosition struct {
	group string
	order int
}

type colorImageTask struct {
	purpose, brief, background, language, evidenceDigest string
	role                                                 SlotRole
}

func (s ColorImageSelection) Valid() bool {
	if !canonicalImageValue(s.Option) || len(s.Groups) < 2 || len(s.Groups) > 3 {
		return false
	}
	colors, originals := make(map[string]bool), make(map[string]bool)
	for _, group := range s.Groups {
		if !canonicalImageValue(group.Color) || strings.Contains(group.Color, "://") ||
			!canonicalImageValue(group.OriginalID) || colors[group.Color] || originals[group.OriginalID] {
			return false
		}
		colors[group.Color], originals[group.OriginalID] = true, true
	}
	return true
}

func (s ColorImageSelection) Matches(options []ImageColorOption) bool {
	if !s.Valid() {
		return false
	}
	for _, option := range options {
		if s.Option != option.Name || len(s.Groups) != len(option.Groups) {
			continue
		}
		for i, group := range s.Groups {
			if group.Color != option.Groups[i].Value || !slices.Contains(option.Groups[i].OriginalIDs, group.OriginalID) {
				return false
			}
		}
		return true
	}
	return false
}

func (s ColorImageSelection) SourceIDs() []string {
	ids := make([]string, 0, len(s.Groups))
	for _, group := range s.Groups {
		ids = append(ids, group.OriginalID)
	}
	return ids
}

func (s ColorImageSelection) OriginalFor(color string) string {
	for _, group := range s.Groups {
		if group.Color == color {
			return group.OriginalID
		}
	}
	return ""
}

func validateColorImageSlots(plan Plan, singleSlotExecution bool) bool {
	group := plan.Set.ColorImages
	if group == nil {
		return true
	}
	if !group.Valid() || !plan.Set.Source.OwnProduct || plan.Set.Target.Platform != "product" ||
		plan.Set.Regeneration != nil || plan.Set.Text.Mode != "none" {
		return false
	}
	if singleSlotExecution {
		if len(plan.Slots) != 1 || len(plan.SourceAssetIDs) != 1 {
			return false
		}
	} else {
		selectedIDs := group.SourceIDs()
		slices.Sort(selectedIDs)
		if !slices.Equal(selectedIDs, plan.SourceAssetIDs) {
			return false
		}
	}
	layouts := make(map[string]map[colorImagePosition]colorImageTask, len(group.Groups))
	for _, slot := range plan.Slots {
		r := slot.Recipe
		p := r.Placement
		if p.VariantID != "" || len(slot.SourceAssetIDs) != 1 ||
			group.OriginalFor(p.Color) != slot.SourceAssetIDs[0] {
			return false
		}
		if !singleSlotExecution {
			if layouts[p.Color] == nil {
				layouts[p.Color] = make(map[colorImagePosition]colorImageTask)
			}
			layouts[p.Color][colorImagePosition{p.Group, p.Order}] = colorImageTask{
				purpose: r.Purpose, brief: slot.Brief, background: r.Background, language: r.Language,
				evidenceDigest: r.EvidenceDigest, role: slot.Role,
			}
		}
	}
	if singleSlotExecution {
		return true
	}
	// The plan contains the full ordered selection for each color. The original
	// and color-specific prompt may differ, while the template tasks must match.
	first := layouts[group.Groups[0].Color]
	if len(first) == 0 {
		return false
	}
	for position := range first {
		if position.order > 1 {
			prior := colorImagePosition{position.group, position.order - 1}
			if _, ok := first[prior]; !ok {
				return false
			}
		}
	}
	for _, color := range group.Groups[1:] {
		if !maps.Equal(layouts[color.Color], first) {
			return false
		}
	}
	return true
}
