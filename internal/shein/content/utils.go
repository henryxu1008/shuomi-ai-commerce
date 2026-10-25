// Package content 提供SHEIN平台的敏感词工具函数
package content

import (
	"regexp"
	"strings"
)

// containsJapanese 检测是否包含日文字符
func (s *SensitiveWordService) containsJapanese(text string) bool {
	japanesePattern := regexp.MustCompile("[\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF\u3000-\u303F]")
	return japanesePattern.MatchString(text)
}

// removeWordFromText 从文本中移除指定单词（大小写不敏感）
// 同时处理含连字符的词（如 Non-toxic），兼容文本中连字符被空格替代的情况
func (s *SensitiveWordService) removeWordFromText(text, word string) string {
	if text == "" || word == "" {
		return text
	}

	// 日文直接字符串替换（无单词边界概念）
	if s.containsJapanese(word) {
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
func (s *SensitiveWordService) cleanupText(text string) string {
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// deduplicateWords 去重单词列表
func (s *SensitiveWordService) deduplicateWords(words []string) []string {
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
