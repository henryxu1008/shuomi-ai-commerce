package openai

import (
	"context"
	"errors"
	"time"
)

type ClientConfig struct {
	APIKey   string
	Model    string
	BaseURL  string
	APIStyle string
	Timeout  time.Duration
}
type ResolvedClientConfig struct {
	CacheKey string
	Config   *ClientConfig
}
type ClientConfigResolver interface {
	ResolveClientConfig(context.Context, string) (*ResolvedClientConfig, error)
}

var (
	ErrClientConfigurationUnavailable = errors.New("client configuration is unavailable")
	ErrClientConfigurationChanged     = errors.New("client configuration changed")
	ErrClientConfigurationUnsupported = errors.New("client configuration provider is unsupported")
)
