package httpapi

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	commercialstore "task-processor/internal/integration/persistence/commercialbilling"
	capacitystore "task-processor/internal/integration/persistence/storagecapacity"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

func emptyStorageCommercialOwner(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, commercialstore.AutoMigrate(db))
	return db
}

func TestStorageRollbackRejectsActualHTTPAssemblyBeforeBuilders(t *testing.T) {
	db := emptyStorageCommercialOwner(t)
	repo, err := capacitystore.New(db)
	require.NoError(t, err)
	require.NoError(t, repo.EnsureNamespace(context.Background(), storagecapacity.Digest("original-http-bucket"), "product-source/", func(context.Context) error { return nil }))
	called := false
	factories := currentApplicationFactories{
		buildWorkbench: func(*config.Config, *logrus.Logger) (workbenchContextBuildResult, error) {
			called = true
			return workbenchContextBuildResult{}, nil
		},
		buildSourceAccount: func(*gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) { return nil, nil },
		buildCommercial:    func(*gorm.DB, *gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) { return nil, nil },
	}
	server, err := buildCurrentApplication(context.Background(), &gorm.DB{}, currentApplicationTestConfig(), logrus.New(), factories, WithCommercialOwnerDatabase(db))
	require.Error(t, err)
	require.Nil(t, server)
	require.False(t, called, "rollback must fail before assembling handlers or recovery")
}
