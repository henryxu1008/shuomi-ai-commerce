// Package client 提供TEMU平台认证管理功能
package client

import (
	"github.com/sirupsen/logrus"
)

// AuthManager 认证管理器
type AuthManager struct {
	config        *AuthConfig
	retryHandler  RetryHandler
	errorDetector ErrorDetector
	logger        *logrus.Entry
}

// NewAuthManager 创建新的认证管理器
func NewAuthManager(logger *logrus.Entry) *AuthManager {
	config := DefaultAuthConfig()
	errorDetector := NewTemuErrorDetector(config, logger)
	retryHandler := NewTemuRetryHandler(config, errorDetector, logger)

	return &AuthManager{
		config:        config,
		retryHandler:  retryHandler,
		errorDetector: errorDetector,
		logger:        logger,
	}
}

// NewAuthManagerWithDependencies 使用自定义依赖创建认证管理器（用于测试）
func NewAuthManagerWithDependencies(
	config *AuthConfig,
	retryHandler RetryHandler,
	errorDetector ErrorDetector,
	logger *logrus.Entry,
) *AuthManager {
	return &AuthManager{
		config:        config,
		retryHandler:  retryHandler,
		errorDetector: errorDetector,
		logger:        logger,
	}
}

// SendRequestWithAuth 发送带认证的请求
func (a *AuthManager) SendRequestWithAuth(client ClientAPI, request map[string]any, result any) error {
	// 检查Cookie
	if err := a.validateCookies(client); err != nil {
		return err
	}

	// 使用重试逻辑发送请求
	return a.retryHandler.SendRequestWithRetry(client, request, result)
}

// validateCookies 验证Cookie状态
func (a *AuthManager) validateCookies(client ClientAPI) error {
	if !client.HasCookies() {
		return NewAuthExpiredError("explicit TEMU session required", nil)
	}
	return nil
}
