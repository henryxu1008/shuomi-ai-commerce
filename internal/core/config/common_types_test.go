package config

import (
	"testing"
	"time"
)

// TestDefaultRetryConfig 测试默认重试配置
func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config == nil {
		t.Fatal("DefaultRetryConfig 返回 nil")
	}

	// 验证默认值
	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, 期望 3", config.MaxRetries)
	}

	if config.InitialDelay != 1*time.Second {
		t.Errorf("InitialDelay = %v, 期望 1s", config.InitialDelay)
	}

	if config.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, 期望 30s", config.MaxDelay)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("BackoffFactor = %v, 期望 2.0", config.BackoffFactor)
	}
}

// TestRetryConfigCustom 测试自定义重试配置
func TestRetryConfigCustom(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:    5,
		InitialDelay:  2 * time.Second,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 1.5,
	}

	if config.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, 期望 5", config.MaxRetries)
	}

	if config.InitialDelay != 2*time.Second {
		t.Errorf("InitialDelay = %v, 期望 2s", config.InitialDelay)
	}

	if config.MaxDelay != 60*time.Second {
		t.Errorf("MaxDelay = %v, 期望 60s", config.MaxDelay)
	}

	if config.BackoffFactor != 1.5 {
		t.Errorf("BackoffFactor = %v, 期望 1.5", config.BackoffFactor)
	}
}

// TestDefaultHTTPClientConfig 测试默认HTTP客户端配置
func TestDefaultHTTPClientConfig(t *testing.T) {
	config := DefaultHTTPClientConfig()

	if config == nil {
		t.Fatal("DefaultHTTPClientConfig 返回 nil")
	}

	// 验证默认值
	if config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, 期望 30s", config.Timeout)
	}

	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, 期望 3", config.MaxRetries)
	}

	if config.RetryDelay != 1*time.Second {
		t.Errorf("RetryDelay = %v, 期望 1s", config.RetryDelay)
	}

	if config.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, 期望 100", config.MaxIdleConns)
	}

	if config.MaxConnsPerHost != 10 {
		t.Errorf("MaxConnsPerHost = %d, 期望 10", config.MaxConnsPerHost)
	}
}

// TestHTTPClientConfigCustom 测试自定义HTTP客户端配置
func TestHTTPClientConfigCustom(t *testing.T) {
	config := &HTTPClientConfig{
		BaseURL:         "https://api.example.com",
		Timeout:         60 * time.Second,
		MaxRetries:      5,
		RetryDelay:      2 * time.Second,
		MaxIdleConns:    200,
		MaxConnsPerHost: 20,
		Headers:         map[string]string{"User-Agent": "test"},
	}

	if config.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q, 期望 'https://api.example.com'", config.BaseURL)
	}

	if config.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, 期望 60s", config.Timeout)
	}

	if config.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, 期望 5", config.MaxRetries)
	}

	if config.RetryDelay != 2*time.Second {
		t.Errorf("RetryDelay = %v, 期望 2s", config.RetryDelay)
	}

	if config.MaxIdleConns != 200 {
		t.Errorf("MaxIdleConns = %d, 期望 200", config.MaxIdleConns)
	}

	if config.MaxConnsPerHost != 20 {
		t.Errorf("MaxConnsPerHost = %d, 期望 20", config.MaxConnsPerHost)
	}

	if config.Headers["User-Agent"] != "test" {
		t.Errorf("Headers[User-Agent] = %q, 期望 'test'", config.Headers["User-Agent"])
	}
}

// TestRetryConfigZeroValues 测试零值配置
func TestRetryConfigZeroValues(t *testing.T) {
	config := &RetryConfig{}

	// 零值应该是有效的
	if config.MaxRetries != 0 {
		t.Errorf("MaxRetries = %d, 期望 0", config.MaxRetries)
	}

	if config.InitialDelay != 0 {
		t.Errorf("InitialDelay = %v, 期望 0", config.InitialDelay)
	}

	if config.MaxDelay != 0 {
		t.Errorf("MaxDelay = %v, 期望 0", config.MaxDelay)
	}

	if config.BackoffFactor != 0 {
		t.Errorf("BackoffFactor = %v, 期望 0", config.BackoffFactor)
	}
}
