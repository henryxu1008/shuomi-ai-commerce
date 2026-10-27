package publishing

import (
	"regexp"
	"strings"
)

// DisplayTextCleanupInput contains explicit text values, not an authorization fact.
type DisplayTextCleanupInput struct {
	Words        []string
	SourceBrand  string
	BrandEnabled bool
	BrandName    string
	BrandNameEn  string
}

// DefaultDisplayTextWords returns independent normalized copies of the default words.
func DefaultDisplayTextWords() []string {
	byLanguage := map[string][]string{
		"en": {
			"amazon", "ebay", "alibaba", "aliexpress", "walmart", "target",
			"brand", "trademark", "copyright", "patent", "licensed",
		},
		"zh": {
			"亚马逊", "淘宝", "天猫", "京东", "拼多多",
			"品牌", "商标", "版权", "专利", "授权",
		},
		"ja": {
			"アマゾン", "楽天", "ヤフー",
			"ブランド", "商標", "著作権", "特許",
		},
		"ko": {
			"아마존", "쿠팡", "네이버",
			"브랜드", "상표", "저작권", "특허",
		},
	}
	var words []string
	for _, languageWords := range byLanguage {
		words = append(words, languageWords...)
	}
	return deduplicateWords(words)
}

// SanitizeDisplayText preserves the original word, brand and whitespace rules.
// An empty Words slice does not implicitly load defaults.
func SanitizeDisplayText(text string, input DisplayTextCleanupInput) string {
	if text == "" {
		return text
	}
	allowed := authorizedBrandAllowlist(input)
	text = removeSensitiveWordsExcept(text, input.Words, allowed)
	text = removeAmazonBrandWordsExcept(text, allowed)
	text = removeSourceBrandWordsExcept(input, text, allowed)
	return cleanupText(text)
}

func authorizedBrandAllowlist(input DisplayTextCleanupInput) []string {
	if !input.BrandEnabled {
		return nil
	}
	allowed := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	for _, value := range []string{input.BrandName, input.BrandNameEn} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		allowed = append(allowed, value)
	}
	return allowed
}

func removeSensitiveWordsExcept(text string, words, allowed []string) string {
	if text == "" {
		return text
	}
	for _, word := range deduplicateWords(words) {
		if word == "" || containsFoldedText(allowed, word) {
			continue
		}
		text = removeWordFromText(text, word)
	}
	return cleanupText(text)
}

func removeAmazonBrandWordsExcept(text string, allowed []string) string {
	if text == "" {
		return text
	}
	for _, brandWord := range getAmazonBrandWords() {
		if containsFoldedText(allowed, brandWord) {
			continue
		}
		text = removeWordFromText(text, brandWord)
	}
	return text
}

func removeSourceBrandWordsExcept(input DisplayTextCleanupInput, text string, allowed []string) string {
	if text == "" {
		return text
	}
	word := strings.TrimSpace(input.SourceBrand)
	if word == "" || containsFoldedText(allowed, word) {
		return text
	}
	return removeWordFromText(text, word)
}

// containsJapanese 检测是否包含日文字符
func containsJapanese(text string) bool {
	japanesePattern := regexp.MustCompile("[\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF\u3000-\u303F]")
	return japanesePattern.MatchString(text)
}

// removeWordFromText 从文本中移除指定单词（大小写不敏感）
// 同时处理含连字符的词（如 Non-toxic），兼容文本中连字符被空格替代的情况
func removeWordFromText(text, word string) string {
	if text == "" || word == "" {
		return text
	}

	// 日文直接字符串替换（无单词边界概念）
	if containsJapanese(word) {
		return strings.ReplaceAll(text, word, "")
	}

	// 构建正则：将词中的连字符匹配为"连字符或空格"，支持 Non-toxic / Non toxic 两种形式
	quotedWord := regexp.QuoteMeta(word)
	// 将转义后的 \- 替换为 [-\s]，使连字符和空格都能匹配
	flexibleWord := strings.ReplaceAll(quotedWord, `\-`, `[-\s]`)

	pattern := `(?i)\b` + flexibleWord + `\b`
	re, err := regexp.Compile(pattern)
	if err != nil {
		// 正则编译失败时降级为大小写不敏感的简单替换
		re2, err2 := regexp.Compile(`(?i)` + regexp.QuoteMeta(word))
		if err2 != nil {
			return strings.ReplaceAll(text, word, "")
		}
		return re2.ReplaceAllString(text, "")
	}
	return re.ReplaceAllString(text, "")
}

// cleanupText 清理文本中的多余空格
func cleanupText(text string) string {
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// deduplicateWords 去重单词列表
func deduplicateWords(words []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, word := range words {
		word = strings.TrimSpace(strings.ToLower(word))
		if word != "" && !seen[word] {
			seen[word] = true
			result = append(result, word)
		}
	}

	return result
}

func containsFoldedText(values []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	targetKey := normalizeBrandToken(target)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if strings.EqualFold(value, target) {
			return true
		}
		valueKey := normalizeBrandToken(value)
		if valueKey == "" || targetKey == "" {
			continue
		}
		if strings.Contains(valueKey, targetKey) || strings.Contains(targetKey, valueKey) {
			return true
		}
	}
	return false
}

func normalizeBrandToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", "&", "")
	return replacer.Replace(value)
}

// getAmazonBrandWords 获取 Amazon 自有品牌词列表（仅保留真正的品牌名，避免误删正常产品描述）
func getAmazonBrandWords() []string {
	return []string{
		// Amazon 自有品牌
		"Amazon Basics",
		"Amazon Essentials",
		"Amazon",
		"AmazonBasics",
		"AmazonEssentials",
		"Solimo",
		"Goodthreads",
		"Daily Ritual",
		"Core 10",
		"Lark & Ro",
		"28 Palms",
		"Buttoned Down",
		// Amazon 平台专属标识（不会出现在正常产品描述中）
		"Fulfillment by Amazon",
		"Ships from Amazon",
		"Sold by Amazon",
		"Amazon Warehouse",
		"Amazon's Choice",
		"#1 Best Seller",
	}
}
