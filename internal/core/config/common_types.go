// Package config 提供通用配置类型
package config

import (
	"time"
)

// HTTPClientConfig 通用HTTP客户端配置
type HTTPClientConfig struct {
	BaseURL            string            `yaml:"baseURL" json:"baseURL"`                 // 基础URL
	Timeout            time.Duration     `yaml:"timeout" json:"timeout"`                 // 请求超时
	MaxRetries         int               `yaml:"maxRetries" json:"maxRetries"`           // 最大重试次数
	RetryDelay         time.Duration     `yaml:"retryDelay" json:"retryDelay"`           // 重试延迟
	MaxIdleConns       int               `yaml:"maxIdleConns" json:"maxIdleConns"`       // 最大空闲连接数
	MaxConnsPerHost    int               `yaml:"maxConnsPerHost" json:"maxConnsPerHost"` // 每个主机最大连接数
	InsecureSkipVerify bool              `yaml:"insecureSkipVerify" json:"insecureSkipVerify"`
	Headers            map[string]string `yaml:"headers" json:"headers"` // 自定义请求头
}

// DefaultHTTPClientConfig 返回默认HTTP客户端配置
func DefaultHTTPClientConfig() *HTTPClientConfig {
	return &HTTPClientConfig{
		Timeout:            30 * time.Second,
		MaxRetries:         3,
		RetryDelay:         1 * time.Second,
		MaxIdleConns:       100,
		MaxConnsPerHost:    10,
		InsecureSkipVerify: false,
		Headers:            make(map[string]string),
	}
}
