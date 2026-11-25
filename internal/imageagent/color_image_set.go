package imageagent

import (
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
		if len(plan.Slots) != 3*len(group.Groups) || !slices.Equal(selectedIDs, plan.SourceAssetIDs) {
			return false
		}
	}
	for _, slot := range plan.Slots {
		p := slot.Recipe.Placement
		maxOrder := 2
		if p.Group == "detail" {
			maxOrder = 1
		}
		if p.VariantID != "" || p.Order > maxOrder || len(slot.SourceAssetIDs) != 1 ||
			group.OriginalFor(p.Color) != slot.SourceAssetIDs[0] {
			return false
		}
	}
	return true
}
