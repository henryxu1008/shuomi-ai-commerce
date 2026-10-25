package browser

import (
	"github.com/sirupsen/logrus"

	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mxschmitt/playwright-go"
)

// Manager 通用浏览器管理器
type Manager struct {
	loggerDependency

	pw          *playwright.Playwright
	browser     playwright.Browser
	context     playwright.BrowserContext
	config      *BrowserConfig
	fingerprint *FingerprintConfig
	userAgents  []string
	userDataDir string // 用户数据目录（用于持久化上下文）
	closeErr    error
	installer   *PlaywrightInstaller
}

// NewManager 创建浏览器管理器
func NewManager(cfg *BrowserConfig, logs func(string) *logrus.Entry) *Manager {
	return &Manager{
		loggerDependency: loggerDependency{logFor: logs},
		config:           cfg,
		userAgents:       GetUserAgentPool(),
		installer:        NewPlaywrightInstaller(logs),
	}
}

// SetFingerprint 设置浏览器指纹
func (m *Manager) SetFingerprint(fingerprint *FingerprintConfig) {
	m.fingerprint = fingerprint
}

// SetUserDataDir 设置用户数据目录（用于持久化上下文，避免隐身模式检测）
func (m *Manager) SetUserDataDir(dir string) {
	m.userDataDir = dir
}

// Install 初始化Playwright（自动安装驱动）
func (m *Manager) Install() error {
	pw, err := m.installer.InstallIfNeeded()
	if err != nil {
		return err
	}
	m.pw = pw
	return nil
}

// Launch 启动浏览器（使用持久化上下文，与Python版本一致）
func (m *Manager) Launch() error {
	if m.pw == nil {
		return fmt.Errorf("playwright未初始化")
	}

	// 动态生成用户代理
	userAgent := GenerateUserAgent(m.config.UserAgent, m.userAgents)

	// 创建上下文启动器
	launcher := NewContextLauncher(m.pw, m.config, m.fingerprint, m.userDataDir, m.logFor)

	// 启动浏览器上下文
	browser, context, err := launcher.Launch(userAgent)
	m.browser = browser
	m.context = context
	if err != nil {
		if browser == nil && context == nil {
			m.closeErr = errors.Join(m.closeErr, ErrBrowserClosureUnknown)
		}
		return err
	}

	return nil
}

// NewPage 创建新页面
func (m *Manager) NewPage() (playwright.Page, error) {
	if m.context == nil {
		return nil, fmt.Errorf("浏览器上下文未初始化")
	}

	page, err := m.context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("创建页面失败: %w", err)
	}

	return page, nil
}

// NavigateTo 导航到URL（通用版本，子类可以重写）
func (m *Manager) NavigateTo(page playwright.Page, url string) error {
	_, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(30000),
	})

	if err != nil {
		// 重试一次
		_, err = page.Goto(url, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
			Timeout:   playwright.Float(30000),
		})
	}

	return err
}

// Close 关闭浏览器，并将所有字段置 nil，
// 使后续 NewPage() 等调用能立即检测到已关闭状态而不是 hang 住。
var ErrBrowserClosureUnknown = errors.New("browser closure is unconfirmed")

func (m *Manager) Close() { _ = m.CloseWithResult() }

// CloseWithResult preserves native errors across repeated Close calls. A nil
// field after a failed close is not proof that the owned browser stopped.
func (m *Manager) CloseWithResult() error {
	if m.context != nil {
		m.closeErr = errors.Join(m.closeErr, m.context.Close())
		m.context = nil
	}
	if m.browser != nil {
		m.closeErr = errors.Join(m.closeErr, m.browser.Close())
		m.browser = nil
	}
	if m.pw != nil {
		m.closeErr = errors.Join(m.closeErr, (*m.pw).Stop())
		m.pw = nil
	}
	return m.closeErr
}

// RunDriver initializes an explicitly provisioned driver without shared-cache
// fallback, auto-installation or downloads. Other callers retain Install().
func (m *Manager) RunDriver(options *playwright.RunOptions) error {
	if options == nil || !filepath.IsAbs(options.DriverDirectory) {
		return fmt.Errorf("explicit driver directory required")
	}
	pw, err := playwright.Run(options)
	if err != nil {
		return err
	}
	m.pw = pw
	return nil
}

// GetContext 获取浏览器上下文（供子类使用）
func (m *Manager) GetContext() playwright.BrowserContext {
	return m.context
}

// GetConfig 获取浏览器配置（供子类使用）
func (m *Manager) GetConfig() *BrowserConfig {
	return m.config
}

// GetDefaultUserDataDir 获取默认用户数据目录
func GetDefaultUserDataDir(identifier string) string {
	// 获取当前可执行文件目录
	execPath, err := os.Executable()
	if err != nil {
		return filepath.Join(".", "chromeData", identifier)
	}
	execDir := filepath.Dir(execPath)
	return filepath.Join(execDir, "chromeData", identifier)
}
