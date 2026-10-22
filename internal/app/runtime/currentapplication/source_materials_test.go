package currentapplication

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	coreconfig "task-processor/internal/core/config"
)

func TestSourceMaterialsManifestUsesExistingOwnersWithoutStoreOrAgent(t *testing.T) {
	base := acquisitionRuntimeConfig()
	base.ProductCollections, base.AcquisitionPlans = true, true
	base.Identity.TenantDirectoryToken = "synthetic"
	base.BrowserCollector = &BrowserCollectorConfig{Endpoint: "http://127.0.0.1:19089", Credential: "synthetic-collector-credential"}
	raw, err := json.Marshal(base)
	require.NoError(t, err)
	var manifest map[string]any
	require.NoError(t, json.Unmarshal(raw, &manifest))
	manifest["sourceMaterials"] = map[string]any{"assetDatabase": map[string]any{"host": "127.0.0.1", "port": 5432, "user": "supply_asset_runtime", "password": "synthetic", "database": "assets", "maxConnections": 2}}
	decode := func(t *testing.T) *Config {
		t.Helper()
		raw, err := json.Marshal(manifest)
		require.NoError(t, err)
		var cfg Config
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		require.NoError(t, d.Decode(&cfg))
		return &cfg
	}
	require.NoError(t, decode(t).validate())
	for _, field := range []string{"productCollections", "acquisitionPlans"} {
		t.Run("requires "+field, func(t *testing.T) {
			manifest[field] = false
			require.Error(t, decode(t).validate())
			manifest[field] = true
		})
	}
	assets := manifest["sourceMaterials"].(map[string]any)["assetDatabase"].(map[string]any)
	assets["user"] = "postgres"
	require.Error(t, decode(t).validate())
	assets["user"] = "supply_asset_runtime"
	assets["database"] = base.ProductAcquisitionDatabase.Database
	require.Error(t, decode(t).validate())
}

func TestSourceMaterialsRuntimeOpensOneAssetPoolWithoutWorkflow(t *testing.T) {
	cfg := acquisitionRuntimeConfig()
	cfg.ProductCollections, cfg.AcquisitionPlans = true, true
	cfg.Identity.TenantDirectoryToken = "synthetic"
	cfg.BrowserCollector = &BrowserCollectorConfig{Endpoint: "http://127.0.0.1:19089", Credential: "synthetic"}
	cfg.SourceMaterials = &SourceMaterialsConfig{AssetDatabase: DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "supply_asset_runtime", Password: "synthetic", Database: "assets", MaxConnections: 2}}
	for _, alias := range []bool{false, true} {
		t.Run(map[bool]string{false: "separate owner", true: "aliased Product pool"}[alias], func(t *testing.T) {
			source, product, assets, commercial := &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, emptyStorageCommercialOwner(t)
			if alias {
				assets = product
			}
			stop := errors.New("source materials construction stop")
			assembled, opens := false, 0
			closed := map[*gorm.DB]int{}
			err := Run(context.Background(), cfg, logrus.New(), Dependencies{
				IdentityPreflight:      func(context.Context, IdentityConfig) error { return nil },
				OpenSourceAccount:      func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },
				OpenCommercialOwner:    func(context.Context, DatabaseConfig) (*gorm.DB, error) { return commercial, nil },
				OpenProductAcquisition: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return product, nil },
				OpenSupplyAssets: func(_ context.Context, d DatabaseConfig) (*gorm.DB, error) {
					opens++
					require.Equal(t, cfg.SourceMaterials.AssetDatabase, d)
					return assets, nil
				},
				// No Store, Agent or Temporal dependency is supplied.
				NewApplicationWithFeatures: func(_ context.Context, _ *gorm.DB, f ApplicationFeatures, _ *coreconfig.Config, _ *logrus.Logger) (*http.Server, error) {
					assembled = true
					require.True(t, f.SourceMaterials)
					require.Same(t, assets, f.SupplyAssetDB)
					require.Nil(t, f.SupplyWorkflow)
					return nil, stop
				},
				CloseDatabase: func(db *gorm.DB) error { closed[db]++; return nil },
			})
			require.Equal(t, 1, opens)
			if alias {
				require.ErrorContains(t, err, "independently opened pool")
				require.False(t, assembled)
			} else {
				require.ErrorIs(t, err, stop)
				require.True(t, assembled)
				require.Equal(t, 1, closed[assets])
			}
		})
	}
}
