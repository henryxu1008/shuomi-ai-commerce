package proposal

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/language"
)

const ContentPolicyVersion = "product-content-review-v1"

// ContentSelection is comparable so it can be frozen in the original Request.
// The field set is code-owned; there are no paths or model-defined field names.
type ContentSelection struct {
	TargetLanguage string `json:"targetLanguage"`
	Title          bool   `json:"title,omitempty"`
	Description    bool   `json:"description,omitempty"`
	SellingPoints  bool   `json:"sellingPoints,omitempty"`
	SEOKeywords    bool   `json:"seoKeywords,omitempty"`
}

func (s ContentSelection) Absent() bool { return s == (ContentSelection{}) }

func (s ContentSelection) Valid() bool {
	if s.TargetLanguage == "" || len(s.TargetLanguage) > 128 || !utf8.ValidString(s.TargetLanguage) || strings.TrimSpace(s.TargetLanguage) != s.TargetLanguage || len(s.Fields()) == 0 {
		return false
	}
	tag, err := language.Parse(s.TargetLanguage)
	if err != nil || tag.String() != s.TargetLanguage {
		return false
	}
	// Raw does not infer a likely language for an unspecified `und` tag.
	base, _, _ := tag.Raw()
	return base.String() != "und"
}

func NewContentSelection(locale string, fields []string) (ContentSelection, error) {
	s := ContentSelection{TargetLanguage: locale}
	for _, field := range fields {
		var selected *bool
		switch field {
		case "title":
			selected = &s.Title
		case "description":
			selected = &s.Description
		case "selling_points":
			selected = &s.SellingPoints
		case "seo_keywords":
			selected = &s.SEOKeywords
		default:
			return ContentSelection{}, ErrInputInvalid
		}
		if *selected {
			return ContentSelection{}, ErrInputInvalid
		}
		*selected = true
	}
	if !s.Valid() {
		return ContentSelection{}, ErrInputInvalid
	}
	return s, nil
}

func (s ContentSelection) Fields() []string {
	var fields []string
	for _, item := range []struct {
		field string
		value bool
	}{{"title", s.Title}, {"description", s.Description}, {"selling_points", s.SellingPoints}, {"seo_keywords", s.SEOKeywords}} {
		if item.value {
			fields = append(fields, item.field)
		}
	}
	return fields
}

func (s ContentSelection) TitleOnly() bool {
	return s.Valid() && s.Title && !s.Description && !s.SellingPoints && !s.SEOKeywords
}
