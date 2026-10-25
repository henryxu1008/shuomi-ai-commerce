// Package content 提供SHEIN平台的敏感词文本处理功能
package content

import (
	"strings"
)

func (s *SensitiveWordService) removeSensitiveWordsExcept(text string, allowed []string) string {
	if text == "" {
		return text
	}

	// 保留原文大小写与标点，直接在原文上做大小写不敏感替换。
	allWords := s.getAllSensitiveWords()

	for _, word := range allWords {
		if word == "" || containsFoldedText(allowed, word) {
			continue
		}
		text = s.removeWordFromText(text, word)
	}

	text = s.cleanupText(text)
	return text
}

// SanitizeDisplayTextWithContext removes sensitive and brand words while preserving
// the source text's case and punctuation as much as possible for preview/draft copy.
func (s *SensitiveWordService) SanitizeDisplayTextWithContext(ctx *CleanupContext, text string) string {
	if text == "" {
		return text
	}

	allowedBrands := s.authorizedBrandAllowlist(ctx)
	text = s.removeSensitiveWordsExcept(text, allowedBrands)
	text = s.removeAmazonBrandWordsExcept(text, allowedBrands)
	text = s.removeContextBrandWordsExcept(ctx, text, allowedBrands)
	return s.cleanupText(text)
}

func (s *SensitiveWordService) authorizedBrandAllowlist(ctx *CleanupContext) []string {
	if ctx == nil || ctx.AuthorizedBrand == nil || !ctx.AuthorizedBrand.Enabled {
		return nil
	}

	allowed := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	for _, value := range []string{ctx.AuthorizedBrand.Name, ctx.AuthorizedBrand.NameEn} {
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

func (s *SensitiveWordService) removeAmazonBrandWordsExcept(text string, allowed []string) string {
	if text == "" {
		return text
	}

	originalText := text
	cleanedText := text
	amazonBrandWords := s.getAmazonBrandWords()
	removedWords := []string{}

	for _, brandWord := range amazonBrandWords {
		if containsFoldedText(allowed, brandWord) {
			continue
		}
		beforeRemoval := cleanedText
		cleanedText = s.removeWordFromText(cleanedText, brandWord)

		// 记录被移除的品牌词
		if beforeRemoval != cleanedText {
			removedWords = append(removedWords, brandWord)
		}
	}

	// 记录品牌词移除统计
	if len(removedWords) > 0 {
		s.logger.Debugf("🏷️ 移除Amazon品牌词: %v", removedWords)
		s.logger.Debugf("🏷️ 品牌词清理: %s -> %s", originalText, cleanedText)
	}

	return cleanedText
}

func (s *SensitiveWordService) removeContextBrandWordsExcept(ctx *CleanupContext, text string, allowed []string) string {
	if text == "" || ctx == nil {
		return text
	}

	brandWord := strings.TrimSpace(ctx.SourceBrand)
	if brandWord == "" || containsFoldedText(allowed, brandWord) {
		return text
	}

	originalText := text
	cleanedText := s.removeWordFromText(text, brandWord)

	// 记录品牌词移除统计
	if originalText != cleanedText {
		s.logger.Debugf("🏷️ 移除上下文品牌词: %s", brandWord)
		s.logger.Debugf("🏷️ 上下文品牌词清理: %s -> %s", originalText, cleanedText)
	}

	return cleanedText
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
func (s *SensitiveWordService) getAmazonBrandWords() []string {
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
