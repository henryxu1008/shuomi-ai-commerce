package imageagentworker

import (
	"context"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"task-processor/internal/core/config"
	commercialstore "task-processor/internal/integration/persistence/commercialbilling"
	capacitystore "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

func TestStorageRollbackRejectsOrganizationWorkerBeforeDependencies(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, commercialstore.AutoMigrate(db))
	cfg := &config.Config{}
	require.NoError(t, configureStorageCapacity(cfg, db, nil, nil, nil, nil), "empty owner allows the original disabled mode")
	repo, err := capacitystore.New(db)
	require.NoError(t, err)
	require.NoError(t, repo.EnsureNamespace(context.Background(), storagecapacity.Digest("original-worker-bucket"), "published/", func(context.Context) error { return nil }))
	require.Error(t, configureStorageCapacity(cfg, db, nil, nil, nil, nil), "registered owner must refuse an unmetered worker")
	require.Error(t, configureStorageCapacity(cfg, nil, nil, nil, nil, nil), "missing owner is not a provably empty owner")
}
