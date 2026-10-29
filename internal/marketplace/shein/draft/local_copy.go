package draft

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

	"task-processor/internal/marketplace/shein/offlinepackage"
	"task-processor/internal/marketplace/shein/publishing"
	"task-processor/internal/product/catalog/canonical"
)

type localCopy struct {
	Title            string
	Description      string
	SKCTitleBase     string
	TitleDiagnostics *offlinepackage.TitleDiagnostics
	DescriptionError error
}

func buildLocalCopy(canonical *canonical.Product, fallbackTitle string) localCopy {
	titleResolution := resolveListingTitle(canonical, fallbackTitle)
	title := titleResolution.title
	description, descriptionErr := resolveListingDescription(canonical)
	copy := localCopy{
		Title:        cleanListingText(title),
		Description:  cleanListingText(description),
		SKCTitleBase: titleResolution.skcBase,
		TitleDiagnostics: &offlinepackage.TitleDiagnostics{
			Source:             titleResolution.source,
			PromptContaminated: titleResolution.contaminate,
			ResolutionNote:     titleResolution.note,
			SKCBaseTitle:       titleResolution.skcBase,
		},
		DescriptionError: descriptionErr,
	}
	sanitizeSheinListingCopy(&copy, publishing.DisplayTextCleanupInput{Words: publishing.DefaultDisplayTextWords()})
	return copy
}

func normalizeLocalCopy(pkg *offlinepackage.Package, canonical *canonical.Product, language string) bool {
	if pkg == nil {
		return false
	}
	copy := buildLocalCopy(canonical, firstNonEmpty(pkg.ProductNameEn, pkg.SpuName))
	changed := false
	if copy.Title != "" && (strings.TrimSpace(pkg.ProductNameEn) == "" || containsCJK(pkg.ProductNameEn)) {
		pkg.ProductNameEn = copy.Title
		pkg.ProductNameMulti = copy.Title
		changed = true
	}
	if copy.Description != "" && (strings.TrimSpace(pkg.Description) == "" || containsCJK(pkg.Description)) {
		pkg.Description = copy.Description
		changed = true
	}
	if pkg.DraftPayload != nil {
		if copy.Title != "" && localizedTextsContainCJK(pkg.DraftPayload.MultiLanguageNameList) {
			pkg.DraftPayload.MultiLanguageNameList = localizedEnglishText(language, copy.Title)
			changed = true
		}
		if copy.Description != "" && localizedTextsContainCJK(pkg.DraftPayload.MultiLanguageDescList) {
			pkg.DraftPayload.MultiLanguageDescList = localizedEnglishText(language, copy.Description)
			changed = true
		}
	}
	return changed
}

func firstEnglishCandidate(values ...string) string {
	for _, value := range values {
		value = cleanListingText(value)
		if value == "" || containsCJK(value) {
			continue
		}
		return value
	}
	return ""
}

func inferEnglishProductType(canonical *canonical.Product, fallback string) string {
	signals := []string{
		lookupCanonicalAttribute(canonical, "product_english_name"),
		lookupCanonicalAttribute(canonical, "english_name"),
		fallback,
	}
	if canonical != nil {
		signals = append(signals, canonical.CategoryPath...)
	}
	for _, signal := range signals {
		value := cleanListingText(signal)
		if value == "" || containsCJK(value) || isPromptLikeTitle(value) {
			continue
		}
		return titleCaseWords(value)
	}
	fallback = cleanListingText(fallback)
	if fallback == "" || containsCJK(fallback) || isPromptLikeTitle(fallback) {
		return ""
	}
	return titleCaseWords(fallback)
}

func lookupCanonicalAttribute(canonical *canonical.Product, key string) string {
	if canonical == nil || len(canonical.Attributes) == 0 {
		return ""
	}
	key = normalizeText(key)
	for name, value := range canonical.Attributes {
		if normalizeText(name) == key {
			return value.Value
		}
	}
	return ""
}

func canonicalDescription(canonical *canonical.Product) string {
	if canonical == nil {
		return ""
	}
	return canonical.Description
}

func cleanListingText(value string) string {
	value = strings.TrimSpace(strings.Join(strings.Fields(value), " "))
	return value
}

func titleCaseWords(value string) string {
	words := strings.Fields(cleanListingText(value))
	for i, word := range words {
		runes := []rune(strings.ToLower(word))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func containsCJK(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func sanitizeListingCopy(value string) string {
	return strings.TrimSpace(listingUnsafeChars.ReplaceAllString(value, " "))
}

func localizedEnglishText(language string, value string) []offlinepackage.LocalizedText {
	value = sanitizeListingCopy(cleanListingText(value))
	language = strings.TrimSpace(language)
	if language == "" {
		language = "en_US"
	}
	return []offlinepackage.LocalizedText{
		{Language: language, Name: value},
		{Language: "en", Name: value},
	}
}

func localizedTextsContainCJK(items []offlinepackage.LocalizedText) bool {
	if len(items) == 0 {
		return true
	}
	for _, item := range items {
		if containsCJK(item.Name) {
			return true
		}
	}
	return false
}

var listingUnsafeChars = regexp.MustCompile(`[^\p{L}\p{N}\s.,'"/+\-xX()]`)

type titleCandidate struct {
	source string
	value  string
}

type titleResolution struct {
	title       string
	source      string
	note        string
	contaminate bool
	skcBase     string
}

var (
	titlePromptCuePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bplease\s+(design|create|generate)\b`),
		regexp.MustCompile(`(?i)\b(the\s+)?image\s+should\b`),
		regexp.MustCompile(`(?i)\b(include|ensure|avoid|need to|must)\b`),
		regexp.MustCompile(`(?i)\b\d{3,5}\s*(pixels?|px)\b`),
		regexp.MustCompile(`(?i)\b\d+\s*[x*]\s*\d+\b`),
		regexp.MustCompile(`帮我设计|请设计|生成|不要侵权|像素`),
	}
	titlePromptSplitPattern = regexp.MustCompile(`(?i)\b(?:please\s+(?:design|create|generate)|the\s+image\s+should|image\s+should|include|ensure|avoid|need to|must)\b`)
	titleCanvasNoisePattern = regexp.MustCompile(`(?i)([,;:，；：]\s*)?(?:\d{3,5}\s*(?:pixels?|px)|\d+\s*[x*]\s*\d+).*$`)
)

func resolveListingTitle(canonical *canonical.Product, fallbackTitle string) titleResolution {
	candidates := []titleCandidate{
		{source: "product_english_name", value: lookupCanonicalAttribute(canonical, "product_english_name")},
		{source: "english_name", value: lookupCanonicalAttribute(canonical, "english_name")},
		{source: "canonical_title", value: fallbackTitle},
	}
	var contaminated bool
	for _, candidate := range candidates {
		value := cleanListingText(candidate.value)
		if value == "" || containsCJK(value) {
			continue
		}
		if !isPromptLikeTitle(value) {
			note := ""
			if contaminated {
				note = "prompt-like higher-priority title candidate replaced by " + candidate.source
			}
			return buildResolvedTitle(value, candidate.source, note, contaminated, canonical, fallbackTitle)
		}
		contaminated = true
		if extracted := extractPromptTitleByRules(value, canonical, fallbackTitle); extracted != "" {
			return buildResolvedTitle(extracted, "prompt_extracted_rule", "prompt-like "+candidate.source+" replaced by rule-extracted title", true, canonical, fallbackTitle)
		}
		return titleResolution{
			title:       "",
			source:      "unresolved_prompt_title",
			note:        "prompt-like title candidates could not be safely resolved; llm title extraction unavailable or unsafe",
			contaminate: true,
			skcBase:     "",
		}
	}
	title := structuredFallbackTitle(canonical, fallbackTitle)
	return buildResolvedTitle(title, "structured_fallback", "", false, canonical, fallbackTitle)
}

func buildResolvedTitle(title, source, note string, contaminated bool, canonical *canonical.Product, fallbackTitle string) titleResolution {
	title = sanitizeResolvedTitle(title)
	if title == "" || containsCJK(title) || isPromptLikeTitle(title) {
		title = structuredFallbackTitle(canonical, fallbackTitle)
		source = "structured_fallback"
		if contaminated && note == "" {
			note = "prompt-like title candidates fell back to structured title"
		}
	}
	return titleResolution{
		title:       title,
		source:      source,
		note:        note,
		contaminate: contaminated,
		skcBase:     buildSKCBaseTitle(title, canonical, fallbackTitle),
	}
}

func sanitizeResolvedTitle(value string) string {
	value = sanitizeListingCopy(cleanListingText(value))
	value = strings.Trim(value, " -_,.;:/")
	return cleanListingText(value)
}

func isPromptLikeTitle(value string) bool {
	value = cleanListingText(value)
	if value == "" {
		return false
	}
	if strings.Count(value, ".") >= 2 {
		return true
	}
	lower := strings.ToLower(value)
	score := 0
	for _, pattern := range titlePromptCuePatterns {
		if pattern.MatchString(lower) {
			score++
		}
	}
	wordCount := len(strings.Fields(lower))
	switch {
	case score >= 2:
		return true
	case score == 1 && wordCount >= 12:
		return true
	case wordCount >= 18 && (strings.Contains(lower, "please") || strings.Contains(lower, "设计")):
		return true
	default:
		return false
	}
}

func extractPromptTitleByRules(promptText string, canonical *canonical.Product, fallbackTitle string) string {
	value := cleanListingText(promptText)
	value = titleCanvasNoisePattern.ReplaceAllString(value, "")
	if strings.Contains(value, " - ") {
		head, tail, ok := strings.Cut(value, " - ")
		if ok && isPromptLikeTitle(tail) {
			value = head
		}
	}
	if loc := titlePromptSplitPattern.FindStringIndex(value); len(loc) == 2 && loc[0] > 0 {
		value = strings.TrimSpace(value[:loc[0]])
	}
	value = sanitizeResolvedTitle(value)
	if value != "" && !isPromptLikeTitle(value) && !containsCJK(value) {
		return trimShortTitle(value, 80, 10)
	}
	return ""
}

func buildSKCBaseTitle(title string, canonical *canonical.Product, fallbackTitle string) string {
	base := sanitizeResolvedTitle(title)
	if base == "" || isPromptLikeTitle(base) {
		base = structuredFallbackTitle(canonical, fallbackTitle)
	}
	if base == "" {
		return ""
	}
	return trimShortTitle(base, 70, 8)
}

func trimShortTitle(value string, maxChars int, maxWords int) string {
	words := strings.Fields(cleanListingText(value))
	if len(words) > maxWords {
		words = words[:maxWords]
	}
	value = strings.Join(words, " ")
	if len(value) <= maxChars {
		return value
	}
	truncated := value[:maxChars]
	if idx := strings.LastIndex(truncated, " "); idx > 0 {
		truncated = truncated[:idx]
	}
	return cleanListingText(truncated)
}

func structuredFallbackTitle(canonical *canonical.Product, fallbackTitle string) string {
	candidates := []string{
		firstEnglishCandidate(
			lookupCanonicalAttribute(canonical, "product_english_name"),
			lookupCanonicalAttribute(canonical, "english_name"),
		),
		sanitizeResolvedTitle(inferEnglishProductType(canonical, fallbackTitle)),
		sanitizeResolvedTitle(fallbackTitle),
	}
	for _, candidate := range candidates {
		candidate = sanitizeResolvedTitle(candidate)
		if candidate == "" || containsCJK(candidate) || isPromptLikeTitle(candidate) {
			continue
		}
		return trimShortTitle(candidate, 80, 10)
	}
	return ""
}

var errEnglishDescriptionUnavailable = errors.New("English product description generation failed")

func resolveListingDescription(product *canonical.Product) (string, error) {
	if existing := firstEnglishCandidate(canonicalDescription(product)); existing != "" {
		return existing, nil
	}
	return "", errEnglishDescriptionUnavailable
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
