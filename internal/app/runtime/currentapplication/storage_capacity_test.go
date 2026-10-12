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
	"task-processor/internal/commercial/billing"
	coreconfig "task-processor/internal/core/config"
	commercialstore "task-processor/internal/integration/persistence/commercialbilling"
	store "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/ledger/storagecapacity"
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

func emptyStorageCommercialOwner(t *testing.T) *gorm.DB {
	t.Helper()
	_, db := storageStartupFixture(t)
	require.NoError(t, commercialstore.AutoMigrate(db))
	return db
}

func TestStorageRollbackRejectsDurableCapacityAndOrders(t *testing.T) {
	for _, state := range []string{"namespace", "reserved", "stored", "prepared_lot", "paid_lot", "expired_lot", "pending_order", "cancelled_order"} {
		t.Run(state, func(t *testing.T) {
			c, db := storageStartupFixture(t)
			require.NoError(t, commercialstore.AutoMigrate(db))
			repo, err := store.New(db)
			require.NoError(t, err)
			ctx := context.Background()
			ns := storagecapacity.Digest("original-capacity-bucket")
			switch state {
			case "namespace":
				require.NoError(t, repo.EnsureNamespace(ctx, ns, "product-source/", func(context.Context) error { return nil }))
			case "reserved", "stored":
				meter := &storagecapacity.Meter{Writer: repo, Namespace: ns}
				object := storagecapacity.Object{Key: "original-file", SHA256: storagecapacity.Digest("file"), SizeBytes: 4, Category: "product_media"}
				reservation, err := meter.Admit(ctx, "org", "source_media", "original-write", storagecapacity.Digest("original-write"), object)
				require.NoError(t, err)
				if state == "stored" {
					require.NoError(t, meter.Confirm(ctx, "org", reservation, object))
				}
			case "prepared_lot", "paid_lot", "expired_lot":
				lot := storagecapacity.PurchaseLot{OrganizationID: "org", OrderID: "paid-order", ItemID: "paid-item", TermsFingerprint: storagecapacity.Digest("terms"), Bytes: storagecapacity.PackBytes, DurationSeconds: storagecapacity.PeriodSeconds}
				require.NoError(t, repo.PrepareLot(ctx, lot))
				if state != "prepared_lot" {
					require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return repo.ActivateLotInTransaction(ctx, tx, lot, "original-money-commit") }))
				}
				if state == "expired_lot" {
					require.NoError(t, db.Exec("UPDATE storage_capacity_lots SET expires_at = ?", "2020-01-01T00:00:00Z").Error)
				}
			case "pending_order", "cancelled_order":
				commercial, err := commercialstore.New(db)
				require.NoError(t, err)
				require.NoError(t, commercial.CreateOffer(ctx, billing.InitialStorageOffer()))
				quote, err := commercial.CreateQuote(ctx, billing.QuoteRequest{OrganizationID: "org", OfferID: billing.InitialStorageOffer().OfferID, Quantity: 1})
				require.NoError(t, err)
				order, err := commercial.CreatePendingResourceOrder(ctx, billing.CreateResourceOrderRequest{OrganizationID: "org", ActorID: "admin", QuoteID: quote.QuoteID, IdempotencyKey: "original-purchase"}, quote)
				require.NoError(t, err)
				if state == "cancelled_order" {
					order.Status = billing.OrderCancelled
					order.FailureCode = billing.OrderFailureInsufficientFunds
					require.NoError(t, commercial.UpdateOrder(ctx, order))
				}
			}
			c.StorageCapacity = false
			require.Error(t, prepareStorageCapacity(ctx, c, db, nil, nil, nil, nil), "existing capacity facts must never be bypassed by false configuration")
		})
	}
}

func TestStorageFalseRequiresAProvablyEmptyOwner(t *testing.T) {
	c, db := storageStartupFixture(t)
	require.NoError(t, commercialstore.AutoMigrate(db))
	c.StorageCapacity = false
	require.NoError(t, prepareStorageCapacity(context.Background(), c, db, nil, nil, nil, nil))
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, prepareStorageCapacity(cancelled, c, db, nil, nil, nil, nil))
	require.NoError(t, db.Exec("DROP TABLE storage_capacity_namespaces").Error)
	require.Error(t, prepareStorageCapacity(context.Background(), c, db, nil, nil, nil, nil))
}

func TestStorageFalseWriterCannotDropItsCommercialOwner(t *testing.T) {
	for _, c := range []*Config{{SourceMedia: &coreconfig.ImageAgentArtifactStoreConfig{}}, {Knowledge: &KnowledgeConfig{Enabled: true}}, {Ecoservices: &EcoservicesConfig{Enabled: true}}, {ImageAgent: &ImageAgentConfig{}}} {
		require.Error(t, prepareStorageCapacity(context.Background(), c, nil, nil, nil, nil, nil))
	}
	require.NoError(t, prepareStorageCapacity(context.Background(), &Config{}, nil, nil, nil, nil, nil), "basic application without counted writers does not consume capacity")
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
