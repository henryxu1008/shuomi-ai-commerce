// Package extractor 提供Amazon文本验证功能
package extractor

import (
	"regexp"
)

// TextValidator 文本验证器
type TextValidator struct{}

// NewTextValidator 创建文本验证器
func NewTextValidator() *TextValidator {
	return &TextValidator{}
}

// IsValidDescription 验证描述是否有效
func (v *TextValidator) IsValidDescription(text string) bool {
	if text == "" || len(text) < 10 {
		return false
	}

	// 过滤掉无意义的文本
	invalidPatterns := []string{
		"^\\s*$",
		"^[\\s\\p{P}]*$",
		"^(\\s*[•●◦▪▫‣⁃]\\s*)+$",
		"^\\s*see\\s+more\\s*$",
		"^\\s*read\\s+more\\s*$",
		"^\\s*show\\s+more\\s*$",
		"^\\s*click\\s+here\\s*$",
	}

	for _, pattern := range invalidPatterns {
		matched, _ := regexp.MatchString("(?i)"+pattern, text)
		if matched {
			return false
		}
	}

	return true
}

// IsValidProductDescription 验证产品描述是否有效（比IsValidDescription更严格）
func (v *TextValidator) IsValidProductDescription(text string) bool {
	if text == "" || len(text) < 10 {
		return false
	}

	// 首先使用基本验证
	if !v.IsValidDescription(text) {
		return false
	}

	// 过滤掉JavaScript和CSS相关内容
	if v.containsJavaScriptOrCSS(text) {
		return false
	}

	// 过滤掉其他无关内容
	if v.containsOtherInvalidContent(text) {
		return false
	}

	// 检查是否包含过多的特殊字符
	if v.hasTooManySpecialChars(text) {
		return false
	}

	return true
}

// containsJavaScriptOrCSS 检查是否包含JavaScript或CSS内容
func (v *TextValidator) containsJavaScriptOrCSS(text string) bool {
	invalidPatterns := []string{
		"function\\s*\\(",
		"var\\s+\\w+\\s*=",
		"window\\.",
		"document\\.",
		"\\$\\(",
		"jQuery",
		"\\.css\\(",
		"\\.addClass\\(",
		"\\.removeClass\\(",
		"addEventListener",
		"onclick\\s*=",
		"onload\\s*=",
		"setTimeout",
		"setInterval",
		"console\\.",
		"P\\._namespace",
		"guardFatal",
		"logShoppableMetrics",
		"\\{[^}]*color\\s*:",
		"\\{[^}]*background\\s*:",
		"\\{[^}]*font\\s*:",
		"\\{[^}]*margin\\s*:",
		"\\{[^}]*padding\\s*:",
		"@media\\s*\\(",
		"@import\\s+",
		"@keyframes\\s+",
		"\\.\\w+\\s*\\{",
		"#\\w+\\s*\\{",
		"rgb\\s*\\(",
		"rgba\\s*\\(",
		"#[0-9A-Fa-f]{3,6}",
		"\\d+px",
		"\\d+em",
		"\\d+rem",
		"calc\\s*\\(",
		"url\\s*\\(",
		"linear-gradient\\s*\\(",
		"var\\s*\\(",
	}

	for _, pattern := range invalidPatterns {
		matched, _ := regexp.MatchString("(?i)"+pattern, text)
		if matched {
			return true
		}
	}
	return false
}

// containsOtherInvalidContent 检查是否包含其他无效内容
func (v *TextValidator) containsOtherInvalidContent(text string) bool {
	otherInvalidPatterns := []string{
		"logShoppableMetrics",
		"premium-module",
		"comparison-table",
		"aplus-v2",
		"aplus-review",
		"padding-right",
		"Customer Reviews",
		"out of 5 stars",
		"\\$[0-9]+\\.[0-9]+",
		"Add to Cart",
		"P\\._namespace",
		"guardFatal",
		"P\\.when\\(",
		"\\.execute\\(",
		"init\\(\\)",
		"ready\\)",
		"TraeAI-",
		"\\[0:0\\]",
	}

	for _, pattern := range otherInvalidPatterns {
		matched, _ := regexp.MatchString("(?i)"+pattern, text)
		if matched {
			return true
		}
	}
	return false
}

// hasTooManySpecialChars 检查是否包含过多特殊字符
func (v *TextValidator) hasTooManySpecialChars(text string) bool {
	specialCharCount := 0
	for _, char := range text {
		if !v.isNormalChar(char) {
			specialCharCount++
		}
	}

	// 如果特殊字符超过20%，认为不是有效的产品描述
	return float64(specialCharCount)/float64(len(text)) > 0.2
}

// isNormalChar 检查是否为正常字符
func (v *TextValidator) isNormalChar(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') || char == ' ' || char == '.' ||
		char == ',' || char == ';' || char == ':' || char == '!' ||
		char == '?' || char == '-' || char == '_' || char == '(' ||
		char == ')' || char == '[' || char == ']' || char == '\'' ||
		char == '"' || char == '&' || char == '+' || char == '/' ||
		char == '\\' || char == '\n' || char == '\r' || char == '\t'
}
