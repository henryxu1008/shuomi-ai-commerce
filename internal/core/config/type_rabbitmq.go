package config

import "time"

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries    int           `yaml:"maxRetries"`
	InitialDelay  time.Duration `yaml:"initialDelay"`
	MaxDelay      time.Duration `yaml:"maxDelay"`
	BackoffFactor float64       `yaml:"backoffFactor"`
}

// DefaultRetryConfig 返回默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
}
