// Package client 提供TEMU平台认证重试处理功能
package client

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// RetryHandler 重试处理器接口
type RetryHandler interface {
	SendRequestWithRetry(client ClientAPI, request map[string]any, result any) error
}

// TemuRetryHandler TEMU平台重试处理器
type TemuRetryHandler struct {
	config        *AuthConfig
	errorDetector ErrorDetector
	logger        *logrus.Entry
}

// NewTemuRetryHandler 创建新的TEMU重试处理器
func NewTemuRetryHandler(
	config *AuthConfig,
	errorDetector ErrorDetector,
	logger *logrus.Entry,
) *TemuRetryHandler {
	return &TemuRetryHandler{
		config:        config,
		errorDetector: errorDetector,
		logger:        logger,
	}
}

// SendRequestWithRetry 发送请求（带重试逻辑）
func (h *TemuRetryHandler) SendRequestWithRetry(client ClientAPI, request map[string]any, result any) error {
	ctx := NewAuthContext(client.GetStoreID())

	for ctx.AttemptCount < h.config.MaxRetries {
		ctx.IncrementAttempt()
		h.logger.Debugf("API调用尝试 %d/%d", ctx.AttemptCount, h.config.MaxRetries)

		err := h.sendRequestOnce(client, request, result)
		if err == nil {
			h.logger.Debugf("API调用成功，尝试次数: %d", ctx.AttemptCount)
			return nil
		}

		h.logger.Warnf("API调用失败 (尝试 %d/%d): %v", ctx.AttemptCount, h.config.MaxRetries, err)

		// 处理认证错误
		if h.errorDetector.IsAuthenticationError(err) {
			if authErr := h.handleAuthError(client, ctx, err); authErr != nil {
				return authErr
			}
		} else {
			ctx.ResetAuthError()
		}

		// 如果不是最后一次尝试，等待后重试
		if ctx.AttemptCount < h.config.MaxRetries {
			h.logger.Debugf("等待 %v 后重试...", h.config.RetryDelay)
			time.Sleep(h.config.RetryDelay)
		}
	}

	return fmt.Errorf("API调用失败，已重试%d次", h.config.MaxRetries)
}

// handleAuthError 处理认证错误
func (h *TemuRetryHandler) handleAuthError(client ClientAPI, ctx *AuthContext, err error) error {
	ctx.IncrementAuthError(err)
	return NewAuthExpiredError("TEMU session expired; explicit authentication required", err)
}

// sendRequestOnce 发送单次请求
func (h *TemuRetryHandler) sendRequestOnce(client ClientAPI, request map[string]any, result any) error {
	requestSender := NewRequestSender(h.logger)
	return requestSender.SendRequest(client, request, result)
}
