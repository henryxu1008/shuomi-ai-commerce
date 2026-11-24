package submissionpersistence_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/shopifyguard"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type shopifyRestoreMember struct{}

func (shopifyRestoreMember) AuthorizeStoreMember(_ context.Context, org string) (storecenter.StoreMemberAccess, error) {
	return storecenter.StoreMemberAccess{OrganizationID: org, ActorID: "actor-a", MemberID: "member-a", Administrator: true, CanWrite: true}, nil
}

// The Product half exercises the real guard's command loader, lock and replay
// against its actual Product schema. The Store half uses the real Store schema
// but substitutes test ownership of the exact old generation and apply receipt.
func prepareShopifyGuardRestore(t *testing.T, ctx context.Context, instance *tcpostgres.PostgresContainer, product *gorm.DB, live *shopifyguard.LiveIsolation, binding storecenter.ProductMerchantBinding) func(shopifyguard.Command) {
	t.Helper()
	dsn, err := instance.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	open := func(name string) *gorm.DB {
		endpoint, err := url.Parse(dsn)
		require.NoError(t, err)
		endpoint.Path = "/" + name
		db, err := gorm.Open(postgres.Open(endpoint.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		pool, err := db.DB()
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, pool.Close()) })
		return db.WithContext(ctx)
	}
	for _, name := range []string{"guard_store", "guard_store_restored", "guard_product_restored", "guard_product_mixed"} {
		require.NoError(t, product.Exec("CREATE DATABASE "+name).Error)
	}
	store := open("guard_store")
	installShopifyRestoreStore(t, ctx, store, binding)

	run := func(tool string, args ...string) {
		t.Helper()
		base := []string{"env", "PGPASSWORD=isolated-fixture", tool, "-w", "-h", "127.0.0.1", "-U", "shopify_fixture"}
		exit, output, err := instance.Exec(ctx, append(base, args...))
		require.NoError(t, err)
		log, err := io.ReadAll(output)
		require.NoError(t, err)
		require.Zero(t, exit, "%s: %s", tool, log)
	}
	run("pg_dump", "-Fc", "-f", "/tmp/shopify-guard-store-before.dump", "guard_store")
	run("pg_dump", "-Fc", "-f", "/tmp/shopify-guard-product-before.dump", "shopify_product")

	return func(command shopifyguard.Command) {
		t.Helper()
		// This is an independent synthetic checkpoint, not a claimed production
		// archive or a checkpoint produced by either restored owner.
		expected := collection.Digest(command)
		raw, err := json.Marshal(command)
		require.NoError(t, err)
		archive := filepath.Join(t.TempDir(), "shopify-guard-replay.json")
		require.NoError(t, os.WriteFile(archive, raw, 0600))
		require.NoError(t, applyShopifyRestoreStore(store, command))

		run("pg_dump", "-Fc", "-f", "/tmp/shopify-guard-product-after.dump", "shopify_product")
		run("pg_restore", "--exit-on-error", "-d", "guard_store_restored", "/tmp/shopify-guard-store-before.dump")
		run("pg_restore", "--exit-on-error", "-d", "guard_product_restored", "/tmp/shopify-guard-product-before.dump")
		run("pg_restore", "--exit-on-error", "-d", "guard_product_mixed", "/tmp/shopify-guard-product-after.dump")
		restoreStore := open("guard_store_restored")
		restoreProduct := shopifyguard.RequireIsolated(open("guard_product_restored"))
		mixedProduct := shopifyguard.RequireIsolated(open("guard_product_mixed"))
		verified := func(p *gorm.DB, head string, payload []byte) bool {
			var archived shopifyguard.Command
			if head == "" || json.Unmarshal(payload, &archived) != nil || collection.Digest(archived) != head {
				return false
			}
			var storeDigest string
			if err := restoreStore.Raw("SELECT digest FROM fixture_shopify_recovery_receipts WHERE id=?", archived.ID).Row().Scan(&storeDigest); err != nil || storeDigest != head {
				return false
			}
			var sources []byte
			err := p.Raw(`SELECT source_keys FROM public.listing_shopify_frozen_generations
				WHERE id=? AND organization_id=? AND shop_key=? AND generation=?`, archived.ID, archived.Organization, archived.ShopKey, archived.Generation).Row().Scan(&sources)
			return err == nil && json.Unmarshal(sources, &archived.Sources) == nil && collection.Digest(archived) == head
		}
		raw, err = os.ReadFile(archive)
		require.NoError(t, err)
		var replay shopifyguard.Command
		require.NoError(t, json.Unmarshal(raw, &replay))
		require.Equal(t, expected, collection.Digest(replay))
		require.False(t, verified(restoreProduct, expected, raw))
		require.False(t, verified(mixedProduct, expected, raw))
		require.NoError(t, applyShopifyRestoreStore(restoreStore, replay))
		require.True(t, verified(mixedProduct, expected, raw))
		resealed, err := shopifyguard.SealIsolated(ctx, mixedProduct, command)
		require.NoError(t, err)
		copiedDrain, err := shopifyguard.DrainReadyIsolated(ctx, mixedProduct, live, resealed)
		require.NoError(t, err)
		require.False(t, copiedDrain, "resealing a restored, drained scope cannot create live coverage")
		_, err = shopifyguard.InstallIsolated(ctx, mixedProduct)
		require.ErrorIs(t, err, shopifyguard.ErrConflict, "an existing Shopify attempt prevents retroactive coverage")
		require.False(t, verified(restoreProduct, expected, raw), "Store alone cannot release recovery")
		require.NoError(t, applyShopifyRestoreStore(restoreStore, replay), "original Store receipt can be retried after losing its acknowledgement")
		require.NoError(t, shopifyguard.ReplayIsolated(ctx, restoreProduct, replay))
		require.NoError(t, shopifyguard.ReplayIsolated(ctx, restoreProduct, replay))
		require.True(t, verified(restoreProduct, expected, raw))
		ready, err := shopifyguard.DrainReadyIsolated(ctx, restoreProduct, live, replay)
		require.NoError(t, err)
		require.False(t, ready, "a recovered command does not prove dispatch observation coverage")
		require.False(t, verified(restoreProduct, "", raw))
		require.False(t, verified(restoreProduct, expected, nil))
		changed := replay
		changed.Sources = []string{"conflicting-source"}
		require.ErrorIs(t, shopifyguard.ReplayIsolated(ctx, restoreProduct, changed), shopifyguard.ErrConflict)

		var active int
		require.NoError(t, restoreStore.Raw(`SELECT count(*) FROM public.workbench_store_shopify_authorizations
			WHERE organization_id=? AND shop_key=? AND generation=? AND revoked_at IS NULL`, replay.Organization, replay.ShopKey, replay.Generation+1).Row().Scan(&active))
		require.Equal(t, 1, active)
		require.ErrorIs(t, shopifyguard.Visible(restoreProduct, replay.Organization, replay.ShopKey, replay.Generation), shopifyguard.ErrFrozen)
		require.NoError(t, shopifyguard.Visible(restoreProduct, replay.Organization, replay.ShopKey, replay.Generation+1))
		newShop := uuid.NewString()
		require.ErrorIs(t, restoreProduct.Transaction(func(tx *gorm.DB) error {
			return shopifyguard.Check(tx, replay.Organization, newShop, 1, replay.Sources[0])
		}), shopifyguard.ErrRestricted, "the original attempt was made after the Product backup")
		require.NoError(t, restoreProduct.Transaction(func(tx *gorm.DB) error {
			return shopifyguard.Check(tx, "other-enterprise", newShop, 1, replay.Sources[0])
		}))
	}
}

func installShopifyRestoreStore(t *testing.T, ctx context.Context, db *gorm.DB, binding storecenter.ProductMerchantBinding) {
	t.Helper()
	pool, err := db.DB()
	require.NoError(t, err)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, storecenter.InstallCurrentSchemaTx(ctx, tx))
	require.NoError(t, storecenter.InstallShopifySchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())
	repo, err := storecenter.NewMemberScopedStoreRepository(db, shopifyRestoreMember{})
	require.NoError(t, err)
	store, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: binding.StoreID, OrganizationID: binding.OrganizationID,
		ActorSubject: "actor-a", Name: "Recovery fixture", Platform: "shopify", Region: "US", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
	require.NoError(t, err)
	_, _, err = repo.CreateOrReplay(ctx, binding.OrganizationID, store)
	require.NoError(t, err)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO public.workbench_store_shopify_shops (organization_id,shop_key,store_id,app_id,shop_id,generation)
			VALUES (?,?,?,?,?,?)`, binding.OrganizationID, binding.Shopify.ShopKey, binding.StoreID, binding.ApplicationID, "gid://shopify/Shop/42", binding.ConnectionRevision+1).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO public.workbench_store_shopify_authorizations (organization_id,shop_key,generation,installation_id,domain,currency,approved_at,revoked_at)
			VALUES (?,?,?,'gid://shopify/AppInstallation/1','fixture.myshopify.com','USD',now(),now()),
			(?,?,?,'gid://shopify/AppInstallation/2','fixture.myshopify.com','USD',now(),NULL)`,
			binding.OrganizationID, binding.Shopify.ShopKey, binding.ConnectionRevision,
			binding.OrganizationID, binding.Shopify.ShopKey, binding.ConnectionRevision+1).Error
	}))
	require.NoError(t, db.Exec("CREATE TABLE fixture_shopify_recovery_receipts (id uuid PRIMARY KEY,digest text NOT NULL)").Error)
}

func applyShopifyRestoreStore(db *gorm.DB, command shopifyguard.Command) error {
	return db.Transaction(func(tx *gorm.DB) error {
		digest := collection.Digest(command)
		var prior string
		err := tx.Raw("SELECT digest FROM fixture_shopify_recovery_receipts WHERE id=?", command.ID).Row().Scan(&prior)
		if err == nil {
			if prior != digest {
				return shopifyguard.ErrConflict
			}
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		result := tx.Exec(`DELETE FROM public.workbench_store_shopify_authorizations a
			WHERE a.organization_id=? AND a.shop_key=? AND a.generation=? AND a.revoked_at IS NOT NULL
			AND EXISTS (SELECT 1 FROM public.workbench_store_shopify_shops s
				WHERE s.organization_id=a.organization_id AND s.shop_key=a.shop_key AND s.generation>a.generation)`,
			command.Organization, command.ShopKey, command.Generation)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return shopifyguard.ErrConflict
		}
		return tx.Exec("INSERT INTO fixture_shopify_recovery_receipts VALUES (?,?)", command.ID, digest).Error
	})
}
