package sourceevidenceinspect

import "golang.org/x/text/language"

// ContentSelection is read-only lineage metadata on this tool's wire. Review
// validates the actual patch; the application maps its result into this DTO.
// Sourcing does not depend on a downstream candidate or adoption policy owner.
type ContentSelection struct {
	TargetLanguage string `json:"targetLanguage"`
	Title          bool   `json:"title,omitempty"`
	Description    bool   `json:"description,omitempty"`
	SellingPoints  bool   `json:"sellingPoints,omitempty"`
	SEOKeywords    bool   `json:"seoKeywords,omitempty"`
}

func (s ContentSelection) Absent() bool { return s == (ContentSelection{}) }

func (s ContentSelection) Valid() bool {
	if len(s.TargetLanguage) == 0 || len(s.TargetLanguage) > 128 || !(s.Title || s.Description || s.SellingPoints || s.SEOKeywords) {
		return false
	}
	tag, err := language.Parse(s.TargetLanguage)
	if err != nil || tag.String() != s.TargetLanguage {
		return false
	}
	base, _, _ := tag.Raw()
	return base.String() != "und"
}
