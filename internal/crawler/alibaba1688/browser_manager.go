// Package alibaba1688 提供1688浏览器管理功能
package alibaba1688

import (
	"github.com/sirupsen/logrus"

	"fmt"
	"task-processor/internal/core/config"

	sharedbrowser "task-processor/internal/crawler/shared/browser"

	"github.com/mxschmitt/playwright-go"
)

type browserLifecycleManager interface {
	Install() error
	Launch() error
	GetContext() playwright.BrowserContext
	NewPage() (playwright.Page, error)
	Close()
}

// BrowserManager 1688专用的浏览器管理器，继承shared的功能
type BrowserManager struct {
	loggerDependency

	Manager browserLifecycleManager
	config  *config.Config
}

type alibaba1688BrowserRuntimeConfig struct {
	browser     *sharedbrowser.BrowserConfig
	userDataDir string
}

// NewPublicBrowserManager creates a clean, non-persistent browser manager for
// public 1688 crawling. It intentionally does not reuse the process-wide
// browser profile, whose cookies could make a public attempt account-backed.
func NewPublicBrowserManager(cfg *config.Config, logs func(string) *logrus.Entry) *BrowserManager {
	return newAlibaba1688BrowserManagerWithRuntimeConfig(cfg, newAlibaba1688PublicBrowserRuntimeConfig(cfg), logs)
}

func newAlibaba1688BrowserManagerWithRuntimeConfig(cfg *config.Config, runtimeConfig alibaba1688BrowserRuntimeConfig, logs func(string) *logrus.Entry) *BrowserManager {
	manager := sharedbrowser.NewManager(runtimeConfig.browser, logs)
	manager.SetUserDataDir(runtimeConfig.userDataDir)

	componentLogger(logs, "crawler/alibaba1688").Info("创建1688浏览器管理器，使用共享持久化浏览器组件")

	return &BrowserManager{
		loggerDependency: loggerDependency{logFor: logs},
		Manager:          manager,
		config:           cfg,
	}
}

func newAlibaba1688PublicBrowserRuntimeConfig(cfg *config.Config) alibaba1688BrowserRuntimeConfig {
	browserConfig := &sharedbrowser.BrowserConfig{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
	if cfg != nil {
		browserConfig.Headless = cfg.Browser.Headless
		browserConfig.BrowserPath = cfg.Browser.BrowserPath
		browserConfig.ProxyServer = cfg.Browser.ProxyServer
		browserConfig.ViewportWidth = cfg.Browser.ViewportWidth
		browserConfig.ViewportHeight = cfg.Browser.ViewportHeight
	}
	return alibaba1688BrowserRuntimeConfig{browser: browserConfig, userDataDir: ""}
}

// CreateBrowser 创建浏览器实例（保持向后兼容的API）
func (bm *BrowserManager) CreateBrowser() (playwright.Browser, playwright.BrowserContext, playwright.Page, func(), error) {
	// 初始化Playwright
	if err := bm.Manager.Install(); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("初始化Playwright失败: %w", err)
	}
	cleanupOnError := true
	defer func() {
		if cleanupOnError {
			bm.Manager.Close()
		}
	}()

	// 启动浏览器
	if err := bm.Manager.Launch(); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("启动浏览器失败: %w", err)
	}

	// 获取上下文
	context := bm.Manager.GetContext()
	if context == nil {
		return nil, nil, nil, nil, fmt.Errorf("浏览器上下文未初始化")
	}

	// 创建页面
	page, err := bm.Manager.NewPage()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("创建页面失败: %w", err)
	}

	// 设置1688特定的超时
	timeout := float64(bm.config.Platforms.Alibaba1688.Timeout * 1000)
	page.SetDefaultTimeout(timeout)
	bm.componentLogger("crawler/alibaba1688").Infof("设置1688页面超时: %.0fms", timeout)

	// 返回清理函数
	cleanup := func() {
		bm.Manager.Close()
	}
	cleanupOnError = false

	// 注意：这里返回nil作为browser，因为使用持久化上下文时没有单独的browser对象
	return nil, context, page, cleanup, nil
}
