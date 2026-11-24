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

func TestShopifyLifecycleFencesFirstConfirmationAndOriginalGeneration(t *testing.T) {
	ctx := context.Background()
	db := newTikTokStoreDatabase(t)
	pool, err := db.DB()
	require.NoError(t, err)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, storecenter.InstallShopifySchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())
	repo, err := storecenter.NewMemberScopedStoreRepository(db, &storeMemberAuthorizer{member: "member-original", admin: true})
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Microsecond)
	installation := storecenter.ShopifyInstallation{
		AppID: "gid://shopify/App/812", ShopID: "gid://shopify/Shop/19", InstallationID: "gid://shopify/AppInstallation/27",
		Domain: "fixture.myshopify.com", Currency: "USD",
	}
	event := storecenter.ShopifyLifecycleNotice{
		AppID: installation.AppID, ShopID: installation.ShopID, Topic: "app/uninstalled", EventID: "event-before-binding", BodyHash: strings.Repeat("a", 64),
	}
	input := validCreateStoreInput()
	input.ID, input.Platform, input.ExternalStoreID, input.Region = uuid.NewString(), "shopify", "", "US"
	s, err := storecenter.NewStore(input)
	require.NoError(t, err)
	saved, _, err := repo.CreateOrReplay(ctx, input.OrganizationID, s)
	require.NoError(t, err)
	begin := func(identity storecenter.ShopifyInstallation, at time.Time) storecenter.ShopifyPairingKey {
		latest, err := repo.Get(ctx, input.OrganizationID, saved.ID())
		require.NoError(t, err)
		cmd := storecenter.OfficialConnectionCommand{OrganizationID: input.OrganizationID, StoreID: saved.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: latest.Version()}
		pair, err := repo.BeginShopifyPairing(ctx, cmd, identity.AppID, strings.Repeat("b", 64), strings.Repeat("c", 64), at)
		require.NoError(t, err)
		_, err = repo.ClaimShopifyPairing(ctx, pair.Key(), strings.Repeat("b", 64), strings.Repeat("c", 64), at)
		require.NoError(t, err)
		gen, err := repo.ShopifyPairingGeneration(ctx, pair.Key(), strings.Repeat("b", 64), identity, at)
		require.NoError(t, err)
		_, err = repo.StageShopifyPairing(ctx, pair.Key(), strings.Repeat("b", 64), identity, gen, "test-key", "test-ciphertext", at)
		require.NoError(t, err)
		return pair.Key()
	}
	key := begin(installation, now)
	before, err := repo.ReadShopifyLifecycleRevision(ctx, installation)
	require.NoError(t, err)
	require.NoError(t, repo.ReceiveShopifyLifecycle(ctx, event, now.Add(time.Second)))
	replay, err := repo.ReadShopifyLifecycleRevision(ctx, installation)
	require.NoError(t, err)
	require.Greater(t, replay, before)
	require.NoError(t, repo.ReceiveShopifyLifecycle(ctx, event, now.Add(2*time.Second)))
	swapped := event
	swapped.BodyHash = strings.Repeat("d", 64)
	require.ErrorIs(t, repo.ReceiveShopifyLifecycle(ctx, swapped, now.Add(2*time.Second)), storecenter.ErrVersionConflict)

	latest, err := repo.ReadShopifyLifecycleRevision(ctx, installation)
	require.NoError(t, err)
	require.Equal(t, replay, latest)
	_, err = repo.ConfirmShopifyPairing(ctx, key, strings.Repeat("b", 64), installation, now.Add(2*time.Second), before)
	require.ErrorIs(t, err, storecenter.ErrVersionConflict)
	authorized, err := repo.ConfirmShopifyPairing(ctx, key, strings.Repeat("b", 64), installation, now.Add(3*time.Second), latest)
	require.NoError(t, err)

	event.EventID = "event-on-binding"
	require.NoError(t, repo.ReceiveShopifyLifecycle(ctx, event, now.Add(4*time.Second)))
	_, err = repo.ReadShopifyBinding(ctx, authorized.OrganizationID, authorized.StoreID)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	claim, err := repo.ClaimShopifyLifecycle(ctx, installation.AppID, now.Add(5*time.Second))
	require.NoError(t, err)
	require.NotNil(t, claim)
	require.Equal(t, authorized.Generation, claim.Generation)
	event.EventID = "second-event-on-binding"
	require.NoError(t, repo.ReceiveShopifyLifecycle(ctx, event, now.Add(6*time.Second)))
	require.NoError(t, repo.ResolveShopifyLifecycle(ctx, *claim, storecenter.ShopifyLifecycleHealthy, now.Add(7*time.Second)))
	_, err = repo.ReadShopifyBinding(ctx, authorized.OrganizationID, authorized.StoreID)
	require.ErrorIs(t, err, storecenter.ErrShopifyLifecyclePending)

	outstanding, err := repo.ClaimShopifyLifecycle(ctx, installation.AppID, now.Add(8*time.Second))
	require.NoError(t, err)
	require.NotNil(t, outstanding)
	restarted, err := storecenter.NewMemberScopedStoreRepository(db, &storeMemberAuthorizer{member: "member-original", admin: true})
	require.NoError(t, err)
	afterLease, err := restarted.ClaimShopifyLifecycle(ctx, installation.AppID, now.Add(70*time.Second))
	require.NoError(t, err)
	require.NotNil(t, afterLease)
	require.Greater(t, afterLease.Attempt, outstanding.Attempt)
	require.NoError(t, repo.ResolveShopifyLifecycle(ctx, *outstanding, storecenter.ShopifyLifecycleHealthy, now.Add(71*time.Second)))
	_, err = repo.ReadShopifyBinding(ctx, authorized.OrganizationID, authorized.StoreID)
	require.ErrorIs(t, err, storecenter.ErrShopifyLifecyclePending, "expired attempt must not release the current hold")
	outstanding = afterLease
	nextInstallation := installation
	nextInstallation.InstallationID = "gid://shopify/AppInstallation/28"
	newKey := begin(nextInstallation, now.Add(72*time.Second))
	revision, err := repo.ReadShopifyLifecycleRevision(ctx, nextInstallation)
	require.NoError(t, err)
	renewed, err := repo.ConfirmShopifyPairing(ctx, newKey, strings.Repeat("b", 64), nextInstallation, now.Add(73*time.Second), revision)
	require.NoError(t, err)
	require.Greater(t, renewed.Generation, authorized.Generation)
	require.NoError(t, repo.ResolveShopifyLifecycle(ctx, *outstanding, storecenter.ShopifyLifecycleRevoked, now.Add(74*time.Second)))
	active, err := repo.ReadShopifyBinding(ctx, authorized.OrganizationID, authorized.StoreID)
	require.NoError(t, err)
	require.Equal(t, renewed, active)
}
