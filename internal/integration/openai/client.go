// Package openai 提供OpenAI API客户端功能
package openai

import (
	"context"
	"fmt"
	"task-processor/internal/ai"
	"task-processor/internal/shared/ptr"
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

// GetStats 获取客户端统计信息
func (c *Client) GetStats() map[string]any {
	if c.pool == nil {
		return map[string]any{"error": "请求池未初始化"}
	}
	return c.pool.GetStats()
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.pool == nil {
		return nil
	}
	return c.pool.Close()
}

// Generate 简单的文本生成（将 prompt 转换为 ChatCompletionRequest）
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	req := &ai.ChatCompletionRequest{
		Messages: []ai.ChatCompletionMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   ptr.IntPtr(1000),
		Temperature: ptr.Float32Ptr(0.7),
	}

	resp, err := c.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// AnalyzeImage 使用 vision 模型分析图片。
// imageURL 必须是可公开访问的 HTTPS URL。
func (c *Client) AnalyzeImage(ctx context.Context, imageURL string, prompt string) (string, error) {
	req := &ai.ChatCompletionRequest{
		Messages: []ai.ChatCompletionMessage{
			{
				Role: "user",
				MultiContent: []ai.ChatCompletionContentPart{
					{
						Type: "image_url",
						ImageURL: &ai.ChatCompletionContentPartImage{
							URL:    imageURL,
							Detail: "auto",
						},
					},
					{
						Type: "text",
						Text: prompt,
					},
				},
			},
		},
		MaxTokens:   ptr.IntPtr(1000),
		Temperature: ptr.Float32Ptr(0.2),
	}

	resp, err := c.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("vision request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI vision")
	}

	return resp.Choices[0].Message.Content, nil
}
