package currentapplication

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	coreconfig "task-processor/internal/core/config"
	store "task-processor/internal/integration/persistence/storagecapacity"
	"testing"
)

func storageStartupFixture(t *testing.T) (*Config, *gorm.DB) {
	t.Helper()
	db, e := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	pool, _ := db.DB()
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, store.AutoMigrate(db))
	owner := DatabaseConfig{Host: "commercial", Port: 5432, Database: "commercial"}
	return &Config{StorageCapacity: true, CommercialOwnerDatabase: &owner, MoneyOwnerDatabase: &DatabaseConfig{}}, db
}

func TestStorageStartupRejectsImageWorkerCapacityOwnerMismatch(t *testing.T) {
	for _, field := range []string{"host", "port", "database", "missing"} {
		t.Run(field, func(t *testing.T) {
			c, db := storageStartupFixture(t)
			c.ImageAgent = &ImageAgentConfig{}
			worker := &coreconfig.Config{StorageCapacity: true, CommercialOwnerDatabase: &coreconfig.DatabaseConfig{Host: "commercial", Port: 5432, Database: "commercial"}}
			switch field {
			case "host":
				worker.CommercialOwnerDatabase.Host = "other"
			case "port":
				worker.CommercialOwnerDatabase.Port = 5433
			case "database":
				worker.CommercialOwnerDatabase.Database = "other"
			case "missing":
				worker.CommercialOwnerDatabase = nil
			}
			require.Error(t, prepareStorageCapacity(context.Background(), c, db, nil, nil, worker, nil))
		})
	}
}

func TestStorageStartupAcceptsSameOwnerWithSeparateCredentials(t *testing.T) {
	c, db := storageStartupFixture(t)
	c.ImageAgent = &ImageAgentConfig{}
	worker := &coreconfig.Config{StorageCapacity: true, CommercialOwnerDatabase: &coreconfig.DatabaseConfig{Host: "commercial", Port: 5432, Database: "commercial", User: "worker-role", Password: "different"}}
	require.NoError(t, prepareStorageCapacity(context.Background(), c, db, nil, nil, worker, nil))
}

func TestStorageNamespaceRechecksNewAccountButPreservesCredentialRotation(t *testing.T) {
	for _, different := range []bool{false, true} {
		t.Run(map[bool]string{false: "same_account_key_rotation", true: "account_switch"}[different], func(t *testing.T) {
			var accountB atomic.Bool
			var lists atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				switch {
				case r.URL.Query().Has("acl"):
					owner := "account-a"
					if accountB.Load() {
						owner = "account-b"
					}
					_, _ = w.Write([]byte(`<AccessControlPolicy><Owner><ID>` + owner + `</ID></Owner></AccessControlPolicy>`))
				case r.URL.Query().Has("versioning"):
					_, _ = w.Write([]byte(`<VersioningConfiguration/>`))
				default:
					lists.Add(1)
					if accountB.Load() {
						_, _ = w.Write([]byte(`<ListBucketResult><Contents><Key>product-source/existing</Key></Contents></ListBucketResult>`))
					} else {
						_, _ = w.Write([]byte(`<ListBucketResult><KeyCount>0</KeyCount></ListBucketResult>`))
					}
				}
			}))
			defer server.Close()
			c, db := storageStartupFixture(t)
			c.SourceMedia = &coreconfig.ImageAgentArtifactStoreConfig{S3: coreconfig.ImageAgentArtifactStoreS3Config{Endpoint: server.URL, Region: "us-east-1", Bucket: "same-name", ArtifactMode: "aws", AccessKeyID: "key-a", SecretAccessKey: "synthetic", UsePathStyle: true}}
			require.NoError(t, prepareStorageCapacity(context.Background(), c, db, nil, nil, nil, nil))
			c.SourceMedia.S3.AccessKeyID = "key-b"
			accountB.Store(different)
			e := prepareStorageCapacity(context.Background(), c, db, nil, nil, nil, nil)
			if different {
				require.Error(t, e, "new physical account must not reuse prior empty registration")
				require.Equal(t, int32(2), lists.Load())
			} else {
				require.NoError(t, e)
				require.Equal(t, int32(1), lists.Load(), "key rotation must reuse the same physical account registration")
			}
		})
	}
}
