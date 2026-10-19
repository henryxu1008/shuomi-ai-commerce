package shein

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	compositionPercentPattern        = regexp.MustCompile(`(?i)([^,;+\n]+?)\s*[:：]?\s*(\d+(?:\.\d+)?)\s*%`)
	compositionLeadingPercentPattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*%\s*([^,;+\n]+)`)
)

type compositionItem struct {
	Name    string
	Percent float64
}

func parseCompositionItems(value string) []compositionItem {
	items := collectCompositionItems(value, compositionPercentPattern, false)
	if len(items) > 0 {
		return items
	}
	return collectCompositionItems(value, compositionLeadingPercentPattern, true)
}

func collectCompositionItems(value string, pattern *regexp.Regexp, percentFirst bool) []compositionItem {
	matches := pattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return nil
	}
	items := make([]compositionItem, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		var (
			nameRaw    string
			percentRaw string
		)
		if percentFirst {
			percentRaw = match[1]
			nameRaw = match[2]
		} else {
			nameRaw = match[1]
			percentRaw = match[2]
		}
		name := strings.TrimSpace(strings.Trim(nameRaw, ",;+/ "))
		percent, err := strconv.ParseFloat(strings.TrimSpace(percentRaw), 64)
		if err != nil || name == "" {
			continue
		}
		items = append(items, compositionItem{Name: name, Percent: percent})
	}
	return items
}
