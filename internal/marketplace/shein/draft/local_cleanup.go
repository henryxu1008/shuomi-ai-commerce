package draft

import (
	"strings"
	"unicode"

	"task-processor/internal/marketplace/shein/offlinepackage"
	"task-processor/internal/marketplace/shein/publishing"
)

func sanitizeSheinListingCopy(copy *localCopy, input publishing.DisplayTextCleanupInput) bool {
	if copy == nil {
		return false
	}
	changed := false
	changed = sanitizeStringField(input, &copy.Title) || changed
	changed = sanitizeStringField(input, &copy.Description) || changed
	changed = sanitizeStringField(input, &copy.SKCTitleBase) || changed
	if copy.TitleDiagnostics != nil {
		copy.TitleDiagnostics.SKCBaseTitle = copy.SKCTitleBase
	}
	return changed
}

func sanitizeLocalDraft(pkg *offlinepackage.Package, input publishing.DisplayTextCleanupInput) bool {
	if pkg == nil || pkg.DraftPayload == nil {
		return false
	}
	changed := false

	changed = sanitizeCommonAttributes(input, pkg.ProductAttributes) || changed
	changed = sanitizeLocalizedTexts(input, pkg.DraftPayload.MultiLanguageNameList) || changed
	changed = sanitizeLocalizedTexts(input, pkg.DraftPayload.MultiLanguageDescList) || changed
	changed = sanitizeCommonAttributes(input, pkg.DraftPayload.ProductAttributeList) || changed
	changed = sanitizeResolvedAttributes(input, pkg.ResolvedAttributes) || changed

	for i := range pkg.DraftPayload.SKCList {
		skc := &pkg.DraftPayload.SKCList[i]
		changed = sanitizeStringField(input, &skc.SkcName) || changed
		changed = sanitizeLocalizedTexts(input, skc.MultiLanguageNameList) || changed
	}

	return changed
}

func sanitizeResolvedAttributes(input publishing.DisplayTextCleanupInput, attrs []offlinepackage.ResolvedAttribute) bool {
	changed := false
	for i := range attrs {
		if attrs[i].AttributeValueID == nil && shouldSanitizeDraftAttributeValue(attrs[i].Value) {
			changed = sanitizeStringField(input, &attrs[i].Value) || changed
		}
		if shouldSanitizeDraftAttributeValue(attrs[i].AttributeExtraValue) {
			changed = sanitizeStringField(input, &attrs[i].AttributeExtraValue) || changed
		}
	}
	return changed
}

func sanitizeLocalizedTexts(input publishing.DisplayTextCleanupInput, items []offlinepackage.LocalizedText) bool {
	changed := false
	for i := range items {
		changed = sanitizeStringField(input, &items[i].Name) || changed
	}
	return changed
}

func sanitizeCommonAttributes(input publishing.DisplayTextCleanupInput, attrs []offlinepackage.Attribute) bool {
	changed := false
	for i := range attrs {
		if !shouldSanitizeDraftAttributeValue(attrs[i].Value) {
			continue
		}
		changed = sanitizeStringField(input, &attrs[i].Value) || changed
	}
	return changed
}

func sanitizeStringField(input publishing.DisplayTextCleanupInput, value *string) bool {
	if value == nil {
		return false
	}
	original := *value
	cleaned := publishing.SanitizeDisplayText(original, input)
	if cleaned == original {
		return false
	}
	*value = cleaned
	return true
}

func shouldSanitizeDraftAttributeValue(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if len(value) >= 3 && strings.ContainsAny(value, " \t\r\n") {
		return true
	}
	hasLetter := false
	for _, r := range value {
		if unicode.IsLetter(r) {
			hasLetter = true
			continue
		}
		if unicode.IsDigit(r) || unicode.IsSpace(r) {
			continue
		}
		switch r {
		case '.', ',', '-', '_', '/', '\\', '+', '%', 'x', 'X':
			continue
		default:
			return hasLetter
		}
	}
	return false
}
