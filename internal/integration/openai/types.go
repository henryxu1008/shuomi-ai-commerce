// Package openai 提供OpenAI API客户端功能
package openai

import (
	"net/http"
	"time"
)

// ClientConfig OpenAI客户端配置
type ClientConfig struct {
	Logger     Logger        `json:"-"`
	APIKey     string        `json:"api_key"`
	Model      string        `json:"model"`
	BaseURL    string        `json:"base_url"`
	APIStyle   string        `json:"api_style,omitempty"`
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`
	// ImageReferenceHTTPClient is an explicit trusted transport override for
	// tests and controlled in-process callers. Production defaults to the
	// SSRF-safe transport in images.go.
	ImageReferenceHTTPClient *http.Client `json:"-"`
	// MaxReferenceMaterializedBytes bounds reference image bytes retained until
	// the multipart request completes.
	MaxReferenceMaterializedBytes int64 `json:"max_reference_materialized_bytes,omitempty"`
	// MaxReferenceMaterializationConcurrency bounds concurrent reference downloads.
	MaxReferenceMaterializationConcurrency int `json:"max_reference_materialization_concurrency,omitempty"`
}

// NewClientConfig 创建新的OpenAI客户端配置
func NewClientConfig(apiKey, model, baseURL string, timeout int) *ClientConfig {
	return &ClientConfig{
		APIKey:     apiKey,
		Model:      model,
		BaseURL:    baseURL,
		Timeout:    time.Duration(timeout) * time.Second,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
	}
}

// PoolConfig 请求池配置
type PoolConfig struct {
	Logger        Logger          `json:"-"`
	RateLimit     float64         `json:"rate_limit"`
	BurstLimit    float64         `json:"burst_limit"`
	ClientConfigs []*ClientConfig `json:"client_configs"`
	MaxConcurrent int             `json:"max_concurrent"`
}
