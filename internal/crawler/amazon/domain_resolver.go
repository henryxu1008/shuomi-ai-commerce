// Package amazon 提供Amazon域名解析功能
package amazon

import (
	"strings"
)

// DomainResolver Amazon域名解析器
type DomainResolver struct{}

// NewDomainResolver 创建域名解析器
func NewDomainResolver() *DomainResolver {
	return &DomainResolver{}
}

// domainMap 地区到域名的映射
var domainMap = map[string]string{
	"us": "amazon.com",
	"uk": "amazon.co.uk",
	"de": "amazon.de",
	"fr": "amazon.fr",
	"it": "amazon.it",
	"es": "amazon.es",
	"ca": "amazon.ca",
	"jp": "amazon.co.jp",
	"au": "amazon.com.au",
	"mx": "amazon.com.mx",
	"br": "amazon.com.br",
	"in": "amazon.in",
	"ae": "amazon.ae",
	"sa": "amazon.sa",
}

// zipcodeMap 地区到默认邮编的映射
var zipcodeMap = map[string]string{
	"us": "94107",     // 旧金山（默认）
	"uk": "SW1A 1AA",  // 伦敦
	"de": "10115",     // 柏林
	"fr": "75001",     // 巴黎
	"jp": "153-0064",  // 东京
	"ca": "M5H 2N2",   // 多伦多
	"it": "00118",     // 罗马
	"es": "28001",     // 马德里
	"in": "110001",    // 新德里
	"mx": "11000",     // 墨西哥城
	"br": "01310-100", // 圣保罗
	"au": "2000",      // 悉尼
	"ae": "00000",     // 迪拜
	"sa": "11564",     // 利雅得
}

// GetZipcodeByRegion 根据地区获取默认邮编
func (r *DomainResolver) GetZipcodeByRegion(region string) string {
	if zipcode, exists := zipcodeMap[strings.ToLower(region)]; exists {
		return zipcode
	}
	return "94107" // 默认使用美国邮编
}

// ExtractRegionFromURL 从URL中提取地区代码
func (r *DomainResolver) ExtractRegionFromURL(url string) string {
	url = strings.ToLower(url)

	// 查找匹配的域名
	for region, domain := range domainMap {
		if strings.Contains(url, domain) {
			return region
		}
	}

	return ""
}
