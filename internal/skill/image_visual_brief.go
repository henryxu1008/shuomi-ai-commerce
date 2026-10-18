package skill

import "strings"

const ImageVisualBriefPath = "references/visual-brief.md"
const MaxImageVisualBriefBytes = 4 << 10

// ImageVisualBrief extracts one resource from an already canonical, exact
// package. It never treats the general Skill document as image instructions.
func ImageVisualBrief(p Package) (string, error) {
	if !p.Supports(ImageTarget) {
		return "", ErrInvalid
	}
	var brief string
	for _, file := range p.Files {
		if file.Path != ImageVisualBriefPath {
			continue
		}
		if brief != "" || strings.TrimSpace(file.Text) == "" || len(file.Text) > MaxImageVisualBriefBytes || !textOK(file.Text) {
			return "", ErrInvalid
		}
		brief = file.Text
	}
	if brief == "" {
		return "", ErrInvalid
	}
	return brief, nil
}
