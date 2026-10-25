package alibaba1688

import (
	"strings"
	"testing"

	"task-processor/internal/core/config"
	"task-processor/internal/pkg/runtimepath"

	"github.com/mxschmitt/playwright-go"
)

func TestAlibaba1688BrowserProfileNamespaceIsStableAcrossRestarts(t *testing.T) {
	first := runtimepath.NamespaceForDirectory(`C:\checkout\one`)
	restarted := runtimepath.NamespaceForDirectory(`C:\checkout\one`)
	secondInstallation := runtimepath.NamespaceForDirectory(`C:\checkout\two`)

	if first != restarted {
		t.Fatalf("namespace changed across restarts: first=%q restarted=%q", first, restarted)
	}
	if first == secondInstallation {
		t.Fatal("different installations must not share the default browser profile namespace")
	}
	if len(first) == 0 || strings.ContainsAny(first, `/\\:`) {
		t.Fatalf("namespace = %q, want a non-empty path-safe value", first)
	}
}

func TestAlibaba1688PublicBrowserRuntimeConfigPreservesSettingsWithoutMutatingConfig(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.Browser.Headless = false
	cfg.Browser.BrowserPath = "C:/browsers/chrome.exe"
	cfg.Browser.UserDataDir = "C:/global-profile"
	cfg.Browser.ProxyServer = "http://global-proxy:8080"
	cfg.Browser.ViewportWidth = 1600
	cfg.Browser.ViewportHeight = 1000

	runtimeConfig := newAlibaba1688PublicBrowserRuntimeConfig(cfg)

	if runtimeConfig.userDataDir != "" {
		t.Fatalf("public user data dir = %q, want empty non-persistent context", runtimeConfig.userDataDir)
	}
	if runtimeConfig.browser.ProxyServer != cfg.Browser.ProxyServer {
		t.Fatalf("proxy = %q, want configured proxy", runtimeConfig.browser.ProxyServer)
	}
	if runtimeConfig.browser.BrowserPath != cfg.Browser.BrowserPath ||
		runtimeConfig.browser.Headless != cfg.Browser.Headless ||
		runtimeConfig.browser.ViewportWidth != cfg.Browser.ViewportWidth ||
		runtimeConfig.browser.ViewportHeight != cfg.Browser.ViewportHeight {
		t.Fatal("runtime config changed browser settings")
	}
	if cfg.Browser.UserDataDir != "C:/global-profile" || cfg.Browser.ProxyServer != "http://global-proxy:8080" {
		t.Fatal("runtime config mutated the process-wide browser configuration")
	}
}

func TestAlibaba1688PublicBrowserRuntimeConfigUsesConfiguredDirectoryAndProxy(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.Browser.UserDataDir = "C:/global-profile"
	cfg.Browser.ProxyServer = "http://global-proxy:8080"

	runtimeConfig := newAlibaba1688PublicBrowserRuntimeConfig(cfg)

	if runtimeConfig.userDataDir != "" {
		t.Fatalf("public user data dir = %q, want empty non-persistent context", runtimeConfig.userDataDir)
	}
	if runtimeConfig.browser.ProxyServer != cfg.Browser.ProxyServer {
		t.Fatalf("proxy = %q, want configured proxy %q", runtimeConfig.browser.ProxyServer, cfg.Browser.ProxyServer)
	}
}

func TestAlibaba1688PublicBrowserRuntimeConfigDoesNotUsePersistentProfile(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.Browser.UserDataDir = "C:/global-profile"
	cfg.Browser.ProxyServer = "http://global-proxy:8080"

	runtimeConfig := newAlibaba1688PublicBrowserRuntimeConfig(cfg)

	if runtimeConfig.userDataDir != "" {
		t.Fatalf("public user data dir = %q, want empty non-persistent context", runtimeConfig.userDataDir)
	}
	if runtimeConfig.browser.ProxyServer != cfg.Browser.ProxyServer {
		t.Fatalf("proxy = %q, want global proxy %q", runtimeConfig.browser.ProxyServer, cfg.Browser.ProxyServer)
	}
	if cfg.Browser.UserDataDir != "C:/global-profile" || cfg.Browser.ProxyServer != "http://global-proxy:8080" {
		t.Fatal("public browser config mutated configured profile or proxy")
	}
}

func TestCreateBrowserClosesManagerAfterInstallWhenLaunchFails(t *testing.T) {
	manager := &fakeBrowserLifecycleManager{launchErr: assertiveError("launch failed")}

	_, _, _, _, err := (&BrowserManager{
		Manager: manager,
		config:  config.NewDefaultConfig(),
	}).CreateBrowser()

	if err == nil {
		t.Fatal("CreateBrowser() error = nil, want launch failure")
	}
	if manager.closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", manager.closeCalls)
	}
}

func TestCreateBrowserClosesManagerAfterInstallWhenContextMissing(t *testing.T) {
	manager := &fakeBrowserLifecycleManager{}

	_, _, _, _, err := (&BrowserManager{
		Manager: manager,
		config:  config.NewDefaultConfig(),
	}).CreateBrowser()

	if err == nil {
		t.Fatal("CreateBrowser() error = nil, want missing context failure")
	}
	if manager.closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", manager.closeCalls)
	}
}

func TestCreateBrowserClosesManagerAfterInstallWhenNewPageFails(t *testing.T) {
	manager := &fakeBrowserLifecycleManager{
		context:    &stubBrowserContext{},
		newPageErr: assertiveError("page failed"),
	}

	_, _, _, _, err := (&BrowserManager{
		Manager: manager,
		config:  config.NewDefaultConfig(),
	}).CreateBrowser()

	if err == nil {
		t.Fatal("CreateBrowser() error = nil, want new page failure")
	}
	if manager.closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", manager.closeCalls)
	}
}

type fakeBrowserLifecycleManager struct {
	installErr error
	launchErr  error
	context    playwright.BrowserContext
	newPageErr error
	closeCalls int
}

func (m *fakeBrowserLifecycleManager) Install() error { return m.installErr }

func (m *fakeBrowserLifecycleManager) Launch() error { return m.launchErr }

func (m *fakeBrowserLifecycleManager) GetContext() playwright.BrowserContext { return m.context }

func (m *fakeBrowserLifecycleManager) NewPage() (playwright.Page, error) {
	return nil, m.newPageErr
}

func (m *fakeBrowserLifecycleManager) Close() { m.closeCalls++ }

type stubBrowserContext struct{ playwright.BrowserContext }

type assertiveError string

func (e assertiveError) Error() string { return string(e) }
