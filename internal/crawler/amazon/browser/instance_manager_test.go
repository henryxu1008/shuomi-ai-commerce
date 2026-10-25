package browser

import (
	"testing"

	"task-processor/internal/core/config"
	sharedbrowser "task-processor/internal/crawler/shared/browser"
)

func TestBrowserInstanceWithoutContextCannotCreatePage(t *testing.T) {
	instance := &BrowserInstance{
		ID: 1,
		Manager: &BrowserManager{
			Manager: sharedbrowser.NewManager(&sharedbrowser.BrowserConfig{}, nil),
		},
	}

	page, err := instance.Manager.NewPage()
	if page != nil {
		t.Fatal("没有活动 context 的实例不应创建页面")
	}
	if err == nil || err.Error() != "浏览器上下文未初始化" {
		t.Fatalf("期望没有活动 context 时拒绝创建页面，实际: %v", err)
	}
}

func TestBrowserManagerDisabledRandomConfigRetainsWindowsStrategy(t *testing.T) {
	cfg := &config.Config{}
	cfg.Browser.Headless = true
	cfg.Browser.BrowserPath = "fixture-browser"
	cfg.Browser.ProxyServer = "http://fixture-proxy.invalid:8080"
	cfg.Browser.ViewportWidth = 1111
	cfg.Browser.ViewportHeight = 777
	cfg.Browser.RandomConfig.Enabled = false
	cfg.Browser.RandomConfig.Strategy = "preset"
	cfg.Browser.RandomConfig.PresetName = "mac_high_end"

	got := NewBrowserManagerWithConfig(cfg, "windows", "windows_high_end", 0, nil).GetConfig()
	if got.FingerprintPlatform != "windows" {
		t.Fatalf("disabled override must retain the Windows strategy, got %q", got.FingerprintPlatform)
	}
	if !got.Headless || got.BrowserPath != cfg.Browser.BrowserPath || got.ProxyServer != cfg.Browser.ProxyServer || got.ViewportWidth != 1111 || got.ViewportHeight != 777 {
		t.Fatalf("base browser configuration was not preserved: %+v", got)
	}
}

func TestBrowserManagerEnabledRandomConfigOverridesRequestedPreset(t *testing.T) {
	cfg := &config.Config{}
	cfg.Browser.ViewportWidth = 1234
	cfg.Browser.ViewportHeight = 789
	cfg.Browser.RandomConfig.Enabled = true
	cfg.Browser.RandomConfig.Strategy = "preset"
	cfg.Browser.RandomConfig.PresetName = "mac_high_end"

	got := NewBrowserManagerWithConfig(cfg, "windows", "windows_mid_range", 19, nil).GetConfig()
	if got.FingerprintPlatform != "macos" {
		t.Fatalf("enabled configured preset must override the requested Windows strategy, got %q", got.FingerprintPlatform)
	}
	if got.ViewportWidth != 1234 || got.ViewportHeight != 789 {
		t.Fatalf("selected preset must retain the configured viewport, got %dx%d", got.ViewportWidth, got.ViewportHeight)
	}
}
