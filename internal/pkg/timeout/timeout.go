// Package timeout 提供统一的上下文超时管理工具
package timeout

import (
	"context"
	"time"
)

// 预定义的超时常量
const (
	// AI相关超时
	AIShortTimeout = 30 * time.Second // AI快速调用超时

	// 下载相关超时
	DownloadTimeout     = 10 * time.Minute // 下载标准超时
	DownloadLongTimeout = 30 * time.Minute // 下载长时间超时
)

// WithAIShortTimeout 创建AI快速调用超时上下文 (30秒)
func WithAIShortTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, AIShortTimeout)
}
