// Package content 提供SHEIN平台的内存敏感词配置。
package content

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// SensitiveWordConfig 敏感词配置结构（按语言分类）
type SensitiveWordConfig struct {
	StaticWords  map[string][]string `json:"static_words"`  // 按语言分类的静态敏感词
	DynamicWords map[string][]string `json:"dynamic_words"` // 按语言分类的动态敏感词
}

// SensitiveWordService 仅在内存中维护词表与清理规则。
type SensitiveWordService struct {
	config *SensitiveWordConfig
	mutex  sync.RWMutex
	logger *logrus.Entry
}

// createDefaultConfig 创建默认配置
func (s *SensitiveWordService) createDefaultConfig() *SensitiveWordConfig {
	return &SensitiveWordConfig{
		StaticWords: map[string][]string{
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
		},
		DynamicWords: make(map[string][]string),
	}
}
