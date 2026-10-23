// Package appenv provides current runtime logging composition and version output.
package appenv

import (
	loggerPkg "task-processor/internal/platform/logging"

	"github.com/sirupsen/logrus"
)

// =============================================================================
// 日志
// =============================================================================

// ComponentLoggers supplies the current Platform getter to runtime composition.
// Consumers receive this dependency; they never import appenv to look it up.
func ComponentLoggers() func(string) *logrus.Entry {
	return loggerPkg.GetGlobalLogger
}

// SetupLoggerWithLevel 以指定级别初始化全局日志管理器，返回底层 *logrus.Logger。
func SetupLoggerWithLevel(level string) *logrus.Logger {
	loggerPkg.InitGlobalLogger(&loggerPkg.LogConfig{
		Level:   level,
		Format:  "text",
		Console: true,
	})
	return loggerPkg.GetGlobalLogManager().GetRawLogger()
}

// =============================================================================
// 版本
// =============================================================================

// VersionInfo 版本信息
type VersionInfo struct {
	Version   string
	BuildTime string
}

// PrintVersionInfo 打印版本信息
func PrintVersionInfo(logger *logrus.Logger, v VersionInfo) {
	logger.Infof("========================================")
	logger.Infof("Task Processor 启动")
	logger.Infof("版本: %s", v.Version)
	logger.Infof("构建时间: %s", v.BuildTime)
	logger.Infof("========================================")
}
