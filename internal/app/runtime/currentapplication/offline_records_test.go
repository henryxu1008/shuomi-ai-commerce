package currentapplication

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"net/http"
	coreconfig "task-processor/internal/core/config"
	"testing"
)

func TestOfflineRecordsAdmissionWithoutAgentOrWorkflow(t *testing.T) {
	cfg := reportRuntimeConfig()
	product := DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "source_acquisition_runtime", Password: "synthetic", Database: "product", MaxConnections: 2}
	cfg.ProductAcquisitionDatabase = &product
	cfg.ProductCollections = true
	cfg.Identity.TenantDirectoryToken = "synthetic"
	cfg.StoreCenter = &StoreCenterConfig{Enabled: true, Database: DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "store_center_runtime", Password: "synthetic", Database: "stores", MaxConnections: 2}}
	cfg.SheinOfflineRecords = &SheinOfflineRecordsConfig{AssetDatabase: DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "supply_asset_runtime", Password: "synthetic", Database: "assets", MaxConnections: 2}}
	require.NoError(t, cfg.validateOfflineRecords())
	for _, mutate := range []func(){func() { cfg.ProductCollections = false }, func() { cfg.ReportCenter = nil }, func() { cfg.StoreCenter = nil }, func() { cfg.Identity.TenantDirectoryToken = "" }, func() { cfg.LocalTrial = &LocalTrialConfig{} }, func() { cfg.PrivateDraftTrial = &PrivateDraftTrialConfig{} }, func() { cfg.SheinOfflineRecords.AssetDatabase.User = "postgres" }, func() { cfg.SheinOfflineRecords.AssetDatabase.Database = "product" }} {
		before := *cfg
		assets := *cfg.SheinOfflineRecords
		mutate()
		require.Error(t, cfg.validateOfflineRecords())
		*cfg = before
		*cfg.SheinOfflineRecords = assets
	}
	require.Nil(t, cfg.ProductAgent)
	require.Nil(t, cfg.SupplyChain)
}

func TestOfflineRecordsRuntimeUsesOneAssetPoolWithoutAgentOrWorkflow(t *testing.T) {
	cfg := acquisitionRuntimeConfig()
	cfg.ProductCollections = true
	cfg.Identity.TenantDirectoryToken = "synthetic"
	cfg.StoreCenter = storeRuntimeConfig().StoreCenter
	cfg.ReportCenter = reportRuntimeConfig().ReportCenter
	cfg.SheinOfflineRecords = &SheinOfflineRecordsConfig{AssetDatabase: DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "supply_asset_runtime", Password: "synthetic", Database: "assets", MaxConnections: 2}}
	require.NoError(t, cfg.validate())
	source, products, stores, reports, assets, commercial := &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, emptyStorageCommercialOwner(t)
	stop := errors.New("offline assembly stop")
	opened, assembled := 0, false
	closed := map[*gorm.DB]int{}
	err := Run(context.Background(), cfg, logrus.New(), Dependencies{
		IdentityPreflight: func(context.Context, IdentityConfig) error { return nil }, OpenSourceAccount: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },
		OpenCommercialOwner: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return commercial, nil }, OpenProductAcquisition: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return products, nil }, OpenStoreCenter: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return stores, nil }, OpenReportCenter: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return reports, nil }, OpenSupplyAssets: func(_ context.Context, d DatabaseConfig) (*gorm.DB, error) {
			opened++
			require.Equal(t, cfg.SheinOfflineRecords.AssetDatabase, d)
			return assets, nil
		},
		NewApplicationWithFeatures: func(_ context.Context, _ *gorm.DB, f ApplicationFeatures, _ *coreconfig.Config, _ *logrus.Logger) (*http.Server, error) {
			assembled = true
			require.True(t, f.SheinOfflineRecords)
			require.Same(t, assets, f.SupplyAssetDB)
			require.Nil(t, f.SupplyWorkflow)
			require.Nil(t, f.ProductAgent)
			return nil, stop
		},
		CloseDatabase: func(db *gorm.DB) error { closed[db]++; return nil },
	})
	require.ErrorIs(t, err, stop)
	require.True(t, assembled)
	require.Equal(t, 1, opened)
	require.Equal(t, 1, closed[assets])
}
