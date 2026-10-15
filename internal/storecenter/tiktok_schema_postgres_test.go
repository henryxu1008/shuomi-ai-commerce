package storecenter_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"task-processor/internal/storecenter"
)

func TestTikTokStoreSchemaCanonicalGrantAndBindingConstraints(t *testing.T) {
	db := newTikTokStoreDatabase(t)
	ctx := context.Background()
	repo, err := storecenter.NewGormStoreRepository(db)
	require.NoError(t, err)
	stores := make([]string, 2)
	for i, org := range []string{"org-a", "org-b"} {
		input := validCreateStoreInput()
		input.ID = uuid.NewString()
		input.CreateIdempotencyKey = uuid.NewString()
		input.Platform = "tiktokshop"
		input.Region = "US"
		input.ExternalStoreID = "fixture-code-" + org
		input.OrganizationID = org
		candidate, err := storecenter.NewStore(input)
		require.NoError(t, err)
		stored, _, err := repo.CreateOrReplay(ctx, org, candidate)
		require.NoError(t, err)
		stores[i] = stored.ID()
	}
	grantID := uuid.NewString()
	insertGrant := `INSERT INTO workbench_store_tiktok_oauth_grants (organization_id,id,app_key,app_revision,seller_identity_hash,generation,version,status,key_id,ciphertext,access_expires_at,refresh_expires_at,refresh_operation_id,claim_expires_at,created_at,updated_at) VALUES (?,?,?,'r1',?,1,1,'active','key','sealed',?, ?,NULL,NULL,?,?)`
	now := time.Now().UTC()
	require.NoError(t, db.Exec(insertGrant, "org-a", grantID, "app", strings.Repeat("a", 64), now.Add(time.Hour), now.Add(24*time.Hour), now, now).Error)
	require.Error(t, db.Exec(insertGrant, "org-b", uuid.NewString(), "app", strings.Repeat("a", 64), now.Add(time.Hour), now.Add(24*time.Hour), now, now).Error)
	insertBinding := `INSERT INTO workbench_store_tiktok_shop_bindings (organization_id,store_id,grant_id,app_key,shop_id,shop_code,shop_cipher,market,seller_type,status,version,created_at,updated_at) VALUES (?,?,?,'app',?,?,'protected-provider-cipher','US','CROSS_BORDER','active',1,?,?)`
	require.NoError(t, db.Exec(insertBinding, "org-a", stores[0], grantID, "official-shop-a", "fixture-code-org-a", now, now).Error)
	require.Error(t, db.Exec(insertBinding, "org-b", stores[1], grantID, "official-shop-b", "fixture-code-org-b", now, now).Error)
	require.Error(t, db.Exec(`UPDATE workbench_store_tiktok_shop_bindings SET seller_type='LOCAL' WHERE organization_id='org-a'`).Error)
	require.Error(t, db.Exec(`UPDATE workbench_store_tiktok_oauth_grants SET generation=2,version=1 WHERE organization_id='org-a'`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE workbench_store_tiktok_oauth_grants DROP CONSTRAINT tiktok_grant_bounds`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE workbench_store_tiktok_oauth_grants ADD CONSTRAINT tiktok_grant_bounds CHECK (generation>0)`).Error)
	require.Error(t, storecenter.VerifyCurrentSchema(ctx, db))
	require.Error(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, db, storecenter.RuntimeCapabilities{TikTokShop: true}), "configured TikTok still rejects a weakened constraint")
}

func newTikTokStoreDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("tiktok_schema"), tcpostgres.WithUsername("postgres"), tcpostgres.WithPassword("synthetic-tiktok-test"), tcpostgres.BasicWaitStrategies())
	if container != nil {
		t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	}
	require.NoError(t, err)
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, storecenter.InstallCurrentSchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())
	require.NoError(t, storecenter.VerifyCurrentSchema(ctx, db))
	return db
}
