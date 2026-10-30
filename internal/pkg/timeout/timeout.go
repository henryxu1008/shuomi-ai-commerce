// Package timeout 提供统一的上下文超时管理工具
package timeout

import (
	"time"
)

// 预定义的超时常量
const (
	// 下载相关超时
	DownloadTimeout     = 10 * time.Minute // 下载标准超时
	DownloadLongTimeout = 30 * time.Minute // 下载长时间超时
)
