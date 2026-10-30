// Package openai 提供OpenAI API客户端功能
package openai

import (
	"context"
	"fmt"
	"task-processor/internal/ai"
)

// Client OpenAI API客户端 - 简化版本，专注于基本功能
type Client struct {
	pool   *RequestPool
	config *ClientConfig
	logger Logger
}

// NewClient 创建新的OpenAI客户端
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		return nil
	}
	clientConfig := cloneClientConfig(config)
	clientLogger := loggerOrNoop(clientConfig.Logger)
	// 创建请求池配置
	poolConfig := &PoolConfig{
		Logger:        clientLogger,
		MaxConcurrent: 10,
		RateLimit:     5.0,
		BurstLimit:    15.0,
		ClientConfigs: []*ClientConfig{clientConfig},
	}

	// 创建请求池
	pool, err := NewRequestPool(poolConfig)
	if err != nil {
		clientLogger.Error("创建OpenAI请求池失败", map[string]any{"error": err.Error()})
		return nil
	}

	return &Client{
		pool:   pool,
		config: clientConfig,
		logger: clientLogger,
	}
}

// CreateChatCompletion 创建聊天完成
func (c *Client) CreateChatCompletion(ctx context.Context, req *ai.ChatCompletionRequest) (*ai.ChatCompletionResponse, error) {
	if c.pool == nil {
		return nil, fmt.Errorf("请求池未初始化")
	}

	// 设置默认模型
	if req.Model == "" {
		req.Model = c.config.Model
	}

	return c.pool.CreateChatCompletion(ctx, req)
}

// GetDefaultModel 获取默认模型
func (c *Client) GetDefaultModel() string {
	return c.config.Model
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.pool == nil {
		return nil
	}
	return c.pool.Close()
}
