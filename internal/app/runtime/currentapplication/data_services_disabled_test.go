package currentapplication

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	core "task-processor/internal/core/config"
)

func TestDataServicesDisabledStartupKeepsReadPoolWithoutExecutionDependencies(t *testing.T) {
	for _, declared := range []bool{true, false} {
		t.Run(map[bool]string{true: "declared runtime", false: "no execution metadata"}[declared], func(t *testing.T) {
			cfg := dataServicesRuntimeConfig(t)
			require.NoError(t, json.Unmarshal([]byte(`{"executionEnabled":false}`), cfg.DataServices))
			if !declared {
				cfg.DataServices.BrowserExecutable, cfg.DataServices.DriverDirectory = "", ""
				cfg.DataServices.TemporalAddress, cfg.DataServices.TemporalNamespace = "", ""
				cfg.DataServices.EnabledSites = nil
			}
			require.NoError(t, cfg.validate())
			source, product, resource, data := &gorm.DB{}, &gorm.DB{}, emptyStorageCommercialOwner(t), &gorm.DB{}
			var closed []*gorm.DB
			assembled := false
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			deps := Dependencies{
				IdentityPreflight:      func(context.Context, IdentityConfig) error { return nil },
				OpenSourceAccount:      func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },
				OpenCommercialOwner:    func(context.Context, DatabaseConfig) (*gorm.DB, error) { return resource, nil },
				OpenProductAcquisition: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return product, nil },
				OpenDataServices:       func(context.Context, DatabaseConfig) (*gorm.DB, error) { return data, nil },
				CloseDatabase:          func(db *gorm.DB) error { closed = append(closed, db); return nil },
				NewApplicationWithFeatures: func(_ context.Context, _ *gorm.DB, f ApplicationFeatures, _ *core.Config, _ *logrus.Logger) (*http.Server, error) {
					assembled = true
					require.Same(t, data, f.DataServicesDB)
					require.Same(t, cfg.DataServices, f.DataServices)
					require.Nil(t, f.DataServicesProvider)
					require.Nil(t, f.DataServicesWorkflow)
					require.Nil(t, f.DataServicesWorker)
					return &http.Server{Handler: http.NotFoundHandler()}, nil
				},
				Listen: func(network, address string) (net.Listener, error) {
					listener, err := net.Listen(network, "127.0.0.1:0")
					cancel()
					return listener, err
				},
			}
			require.NoError(t, Run(ctx, cfg, logrus.New(), deps))
			require.True(t, assembled)
			require.Equal(t, []*gorm.DB{data, product, resource, source}, closed)
		})
	}
}
