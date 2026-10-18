// Package image 提供TEMU平台图片处理相关功能
package image

import "strings"

// imageConfig TEMU图片配置
type imageConfig struct {
	MaxImageCount     int
	MinWidth          int
	MinHeight         int
	MaxSize           int64
	MinAspectRatio    float64
	SupportedFormats  []string
	RecommendedWidth  int
	RecommendedHeight int
	RecommendedSize   int64
	CDNDomains        []string
}

var defaultImageConfig = &imageConfig{
	MaxImageCount:     49,
	MinWidth:          480,
	MinHeight:         480,
	MaxSize:           3 * 1024 * 1024,
	MinAspectRatio:    1.0 / 3.0,
	SupportedFormats:  []string{"JPEG", "JPG", "PNG"},
	RecommendedWidth:  1500,
	RecommendedHeight: 1500,
	RecommendedSize:   1 * 1024 * 1024,
	CDNDomains: []string{
		"img.kwcdn.com",
		"local-goods-image",
		"temu.com",
		"pdd.com",
	},
}

// isCDNImage 检查图片是否已经在TEMU CDN上
func isCDNImage(imageURL string) bool {
	for _, domain := range defaultImageConfig.CDNDomains {
		if strings.Contains(imageURL, domain) {
			return true
		}
	}
	return false
}

// needsUpload 判断图片是否需要上传
func needsUpload(imageURL string) bool {
	return !isCDNImage(imageURL)
}
