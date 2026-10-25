package browser

import (
	"github.com/sirupsen/logrus"

	"encoding/json"
	"fmt"
	"strings"

	"task-processor/internal/pkg/httpclient"
	"time"
)

// IPLocationInfo IP地理位置信息
type IPLocationInfo struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Timezone    string  `json:"timezone"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
}

// GetLocationInfoByIP 根据IP获取地理位置信息
func GetLocationInfoByIP(ip string) (*IPLocationInfo, error) {
	client := httpclient.NewSimpleWithTimeout(10 * time.Second)
	resp, err := client.Get(fmt.Sprintf("http://ip-api.com/json/%s", ip))
	if err != nil {
		return nil, fmt.Errorf("请求IP地理位置失败: %w", err)
	}
	defer resp.Body.Close()

	var info IPLocationInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("解析IP地理位置响应失败: %w", err)
	}

	if info.Status != "success" {
		return nil, fmt.Errorf("IP地理位置查询失败: %s", info.Status)
	}

	return &info, nil
}

// GetTimezoneForRegion 根据代理IP地理位置返回对应时区
func GetTimezoneForRegion(proxyServer string, logs func(string) *logrus.Entry) *string {
	ip := extractIPFromProxy(proxyServer)
	if ip == "" {
		timezone := "America/Los_Angeles"
		return &timezone
	}

	info, err := GetLocationInfoByIP(ip)
	if err != nil {
		componentLogger(logs, "crawler/shared").Warnf("获取IP地理位置失败: %v，使用默认时区", err)
		timezone := "America/Los_Angeles"
		return &timezone
	}

	componentLogger(logs, "crawler/shared").Infof("IP %s 对应时区: %s, 国家: %s", ip, info.Timezone, info.Country)
	return &info.Timezone
}

// extractIPFromProxy 从代理服务器地址中提取IP
func extractIPFromProxy(proxyServer string) string {
	if proxyServer == "" {
		return ""
	}
	proxy := strings.TrimPrefix(proxyServer, "https://")
	proxy = strings.TrimPrefix(proxy, "http://")
	if idx := strings.Index(proxy, ":"); idx != -1 {
		proxy = proxy[:idx]
	}
	return proxy
}
