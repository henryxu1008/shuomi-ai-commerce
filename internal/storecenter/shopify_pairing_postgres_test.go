package storecenter_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
)

func TestShopifyAmazonAndGoogleStoreSchemasComposeOnEmptyStore(t *testing.T) {
	for _, order := range []string{"Amazon first", "Shopify first"} {
		t.Run(order, func(t *testing.T) {
			ctx := context.Background()
			db := newTikTokStoreDatabase(t)
			pool, err := db.DB()
			require.NoError(t, err)
			tx, err := pool.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer tx.Rollback()
			if order == "Amazon first" {
				require.NoError(t, storecenter.InstallAmazonAuthorizationSchemaTx(ctx, tx))
				require.NoError(t, storecenter.InstallGoogleMerchantSchemaTx(ctx, tx))
				require.NoError(t, storecenter.InstallShopifySchemaTx(ctx, tx))
			} else {
				require.NoError(t, storecenter.InstallShopifySchemaTx(ctx, tx))
				require.NoError(t, storecenter.InstallAmazonAuthorizationSchemaTx(ctx, tx))
				require.NoError(t, storecenter.InstallGoogleMerchantSchemaTx(ctx, tx))
			}
			require.NoError(t, tx.Commit())
			require.NoError(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, db,
				storecenter.RuntimeCapabilities{TikTokShop: true, WalmartUS: true, AmazonAuthorization: true, Shopify: true, GoogleMerchant: true}))
		})
	}
}

func TestShopifyPairingIsConsumedOnceAndShopRemainsExclusive(t *testing.T) {
	ctx := context.Background()
	db := newTikTokStoreDatabase(t)
	shopifyCapability := storecenter.RuntimeCapabilities{Shopify: true}
	showsLifecycle, err := storecenter.ShopifyLifecycleInstalled(ctx, db, false)
	require.NoError(t, err)
	require.False(t, showsLifecycle)
	_, err = storecenter.ShopifyLifecycleInstalled(ctx, db, true)
	require.Error(t, err)

	require.Error(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, db, shopifyCapability))
	pool, err := db.DB()
	require.NoError(t, err)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	require.NoError(t, storecenter.InstallShopifySchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())
	require.NoError(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, db, shopifyCapability))
	authorizer := &storeMemberAuthorizer{member: "member-original", admin: true}
	repo, err := storecenter.NewMemberScopedStoreRepository(db, authorizer)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Microsecond)
	provider := storecenter.ShopifyInstallation{
		AppID: "gid://shopify/App/812", ShopID: "gid://shopify/Shop/19",
		InstallationID: "gid://shopify/AppInstallation/27", Domain: "fixture.myshopify.com", Currency: "USD",
	}
	session := strings.Repeat("a", 64)
	state := strings.Repeat("b", 64)
	start := func(org string) storecenter.ShopifyPairing {
		input := validCreateStoreInput()
		input.ID = uuid.NewString()
		input.Platform = "shopify"
		input.ExternalStoreID = ""
		input.Region = "US"
		input.OrganizationID = org
		input.ActorSubject = "subject-create"
		input.CreateIdempotencyKey = uuid.NewString()
		store, err := storecenter.NewStore(input)
		require.NoError(t, err)
		saved, _, err := repo.CreateOrReplay(ctx, org, store)
		require.NoError(t, err)
		command := storecenter.OfficialConnectionCommand{OrganizationID: org, StoreID: saved.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: saved.Version()}
		pairing, err := repo.BeginShopifyPairing(ctx, command, provider.AppID, session, state, now)
		require.NoError(t, err)
		return pairing
	}

	a := start("org-a")
	_, err = repo.ClaimShopifyPairing(ctx, a.Key(), strings.Repeat("c", 64), state, now)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	claimed, err := repo.ClaimShopifyPairing(ctx, a.Key(), session, state, now)
	require.NoError(t, err)
	_, err = repo.ClaimShopifyPairing(ctx, a.Key(), session, state, now)
	require.ErrorIs(t, err, storecenter.ErrVersionConflict)
	staged, err := repo.StageShopifyPairing(ctx, claimed.Key(), session, provider, 0, "fixture-key", "fixture-ciphertext", now)
	require.NoError(t, err)
	authorizer.member = "replacement-member"
	_, err = repo.ConfirmShopifyPairing(ctx, staged.Key(), session, provider, now, 0)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	authorizer.member = "member-original"
	result, err := repo.ConfirmShopifyPairing(ctx, staged.Key(), session, provider, now, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Generation)
	require.NotEmpty(t, result.ShopKey)
	pairedAgain, err := repo.ConfirmShopifyPairing(ctx, staged.Key(), session, provider, now, 0)
	require.NoError(t, err)
	require.Equal(t, result, pairedAgain)
	var storedCiphertext string
	require.NoError(t, db.Raw("SELECT ciphertext FROM workbench_store_shopify_pairings WHERE organization_id=? AND attempt_id=?", a.OrganizationID, a.AttemptID).Row().Scan(&storedCiphertext))
	require.Empty(t, storedCiphertext)

	b := start("org-b")
	b, err = repo.ClaimShopifyPairing(ctx, b.Key(), session, state, now)
	require.NoError(t, err)
	_, err = repo.StageShopifyPairing(ctx, b.Key(), session, provider, 1, "fixture-key", "fixture-ciphertext", now)
	require.ErrorIs(t, err, storecenter.ErrAlreadyExists)

	addon, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer addon.Rollback()
	require.NoError(t, storecenter.InstallShopifyLifecycleSchemaTx(ctx, addon))
	require.NoError(t, addon.Commit())
	showsLifecycle, err = storecenter.ShopifyLifecycleInstalled(ctx, db, false)
	require.NoError(t, err)
	require.True(t, showsLifecycle)
	shopifyCapability.ShopifyLifecycle = true
	require.NoError(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, db, shopifyCapability))
	readback, err := repo.WithShopifyLifecycle(showsLifecycle).ReadShopifyBinding(ctx, result.OrganizationID, result.StoreID)
	require.NoError(t, err)
	require.Equal(t, result, readback, "provisioning an empty receipt table does not rewrite existing approval")

	destination := start(result.OrganizationID)
	for _, storeID := range []string{result.StoreID, destination.StoreID} {
		require.NoError(t, repo.SetMemberGrant(ctx, storecenter.MemberStoreGrantCommand{
			OrganizationID: result.OrganizationID, StoreID: storeID, MemberID: authorizer.member,
			OperationID: uuid.NewString(), Active: true,
		}))
	}
	authorizer.admin = false
	_, err = repo.ClaimShopifyPairing(ctx, destination.Key(), session, state, now)
	require.NoError(t, err)
	_, err = repo.StageShopifyPairing(ctx, destination.Key(), session, provider, result.Generation, "fixture-key", "fixture-ciphertext", now)
	require.NoError(t, err)
	oldGrantRead, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer oldGrantRead.Rollback()
	var active bool
	require.NoError(t, oldGrantRead.QueryRowContext(ctx, `SELECT active FROM workbench_store_member_grants
		WHERE organization_id=$1 AND store_id=$2 AND member_id=$3 FOR SHARE`, result.OrganizationID, result.StoreID, authorizer.member).Scan(&active))
	require.True(t, active)
	limited, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	moved, err := repo.WithShopifyLifecycle(true).ConfirmShopifyPairing(limited, destination.Key(), session, provider, now, 0)
	require.NoError(t, err, "a concurrent permission check on the old Store must not block the rebind")
	require.Equal(t, destination.StoreID, moved.StoreID)
	require.Equal(t, result.Generation+1, moved.Generation)

}
