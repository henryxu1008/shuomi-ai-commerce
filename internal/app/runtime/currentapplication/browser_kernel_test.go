package currentapplication

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"net"
	"net/http"
	core "task-processor/internal/core/config"
	"testing"
)

func kernelConfig() *BrowserKernelConfig {
	return &BrowserKernelConfig{Database: DatabaseConfig{Host: "127.0.0.1", Port: 15432, User: "browser_kernel_runtime", Password: "fixture", Database: "browser_kernel", MaxConnections: 4}, ArtifactOrigins: []string{"https://assets.example.test"}}
}
func TestBrowserKernelRequiresIndependentBoundedOwner(t *testing.T) {
	cfg := runtimeTestConfig()
	require.NoError(t, cfg.validate())
	cfg.BrowserKernel = kernelConfig()
	require.NoError(t, cfg.validate())
	for _, change := range []func(*Config){func(c *Config) { c.BrowserKernel.Database.User = "source_account_runtime" }, func(c *Config) { c.BrowserKernel.Database.Database = "task_processor" }, func(c *Config) { c.BrowserKernel.Database.MaxConnections = 5 }, func(c *Config) { c.BrowserKernel.ArtifactOrigins = []string{"https://assets.example.test/"} }, func(c *Config) {
		d := c.BrowserKernel.Database
		d.User = "notification_center_runtime"
		c.NotificationCenterDatabase = &d
	}} {
		c := runtimeTestConfig()
		c.BrowserKernel = kernelConfig()
		change(c)
		require.Error(t, c.validate())
	}
}
func TestBrowserKernelRuntimeOptInLifecycle(t *testing.T) {
	for _, mode := range []string{"disabled", "enabled", "open-error", "aliased", "construction-error"} {
		t.Run(mode, func(t *testing.T) {
			cfg := runtimeTestConfig()
			if mode != "disabled" {
				cfg.BrowserKernel = kernelConfig()
			}
			source, pool := &gorm.DB{}, &gorm.DB{}
			opened, assembled := 0, 0
			var closed []*gorm.DB
			stop := errors.New("fixture stop")
			deps := Dependencies{
				IdentityPreflight: func(context.Context, IdentityConfig) error { return nil }, OpenSourceAccount: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },
				OpenBrowserKernel: func(_ context.Context, d DatabaseConfig) (*gorm.DB, error) {
					opened++
					require.Equal(t, "browser_kernel", d.Database)
					if mode == "open-error" {
						return nil, stop
					}
					if mode == "aliased" {
						return source, nil
					}
					return pool, nil
				},
				NewApplicationWithFeatures: func(_ context.Context, _ *gorm.DB, f ApplicationFeatures, _ *core.Config, _ *logrus.Logger) (*http.Server, error) {
					assembled++
					if mode == "disabled" {
						require.Nil(t, f.BrowserKernelDB)
						require.Nil(t, f.BrowserKernel)
					} else {
						require.Same(t, pool, f.BrowserKernelDB)
						require.Equal(t, cfg.BrowserKernel, f.BrowserKernel)
					}
					if mode == "construction-error" {
						return nil, stop
					}
					return &http.Server{}, nil
				},
				Listen: func(string, string) (net.Listener, error) { return nil, stop }, CloseDatabase: func(db *gorm.DB) error { closed = append(closed, db); return nil },
			}
			require.Error(t, Run(context.Background(), cfg, logrus.New(), deps))
			if mode == "disabled" {
				require.Zero(t, opened)
			} else {
				require.Equal(t, 1, opened)
			}
			if mode == "open-error" || mode == "aliased" {
				require.Zero(t, assembled)
			} else {
				require.Equal(t, 1, assembled)
			}
			if mode == "enabled" || mode == "construction-error" {
				require.Equal(t, []*gorm.DB{pool, source}, closed)
			} else {
				require.Equal(t, []*gorm.DB{source}, closed)
			}
		})
	}
}
