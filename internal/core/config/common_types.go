// Package config 提供通用配置类型
package config

import (
	"time"
)

// TimeoutConfig 通用超时配置
type TimeoutConfig struct {
	ConnectTimeout time.Duration `yaml:"connectTimeout" json:"connectTimeout"` // 连接超时
	ReadTimeout    time.Duration `yaml:"readTimeout" json:"readTimeout"`       // 读取超时
	WriteTimeout   time.Duration `yaml:"writeTimeout" json:"writeTimeout"`     // 写入超时
	IdleTimeout    time.Duration `yaml:"idleTimeout" json:"idleTimeout"`       // 空闲超时
}

// DefaultTimeoutConfig 返回默认超时配置
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		ConnectTimeout: 10 * time.Second,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    90 * time.Second,
	}
}

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

// RateLimitConfig 通用限流配置
type RateLimitConfig struct {
	Enabled           bool    `yaml:"enabled" json:"enabled"`                     // 是否启用限流
	RequestsPerSecond float64 `yaml:"requestsPerSecond" json:"requestsPerSecond"` // 每秒请求数
	BurstSize         int     `yaml:"burstSize" json:"burstSize"`                 // 突发大小
}

// DefaultRateLimitConfig 返回默认限流配置
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10.0,
		BurstSize:         20,
	}
}
