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

func TestShopifyStartedInstallationConsumesOnlyTheConfirmedStore(t *testing.T) {
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
	identity := storecenter.ShopifyInstallation{
		AppID: "gid://shopify/App/812", ShopID: "gid://shopify/Shop/19",
		InstallationID: "gid://shopify/AppInstallation/27", Domain: "fixture.myshopify.com", Currency: "USD",
	}
	session, state, foreignSession, codeDigest := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64), strings.Repeat("d", 64)
	newStore := func(org string) storecenter.OfficialConnectionCommand {
		input := validCreateStoreInput()
		input.ID, input.Platform, input.ExternalStoreID = uuid.NewString(), "shopify", ""
		input.Region, input.OrganizationID = "US", org
		input.ActorSubject, input.CreateIdempotencyKey = "subject-create", uuid.NewString()
		store, err := storecenter.NewStore(input)
		require.NoError(t, err)
		saved, _, err := repo.CreateOrReplay(ctx, org, store)
		require.NoError(t, err)
		return storecenter.OfficialConnectionCommand{OrganizationID: org, StoreID: saved.ID(), ExpectedStoreVersion: saved.Version()}
	}
	first, other := newStore("org-a"), newStore("org-b")

	stage := func() storecenter.ShopifyInstallProof {
		attempt := uuid.NewString()
		require.NoError(t, repo.BeginShopifyInstall(ctx, attempt, identity.AppID, identity.Domain, session, state, now))
		_, err := repo.ClaimShopifyInstall(ctx, attempt, identity.AppID, identity.Domain, foreignSession, state, codeDigest, now)
		require.ErrorIs(t, err, storecenter.ErrNotFound)
		proof, err := repo.ClaimShopifyInstall(ctx, attempt, identity.AppID, identity.Domain, session, state, codeDigest, now)
		require.NoError(t, err)
		_, err = repo.ClaimShopifyInstall(ctx, attempt, identity.AppID, identity.Domain, session, state, codeDigest, now)
		require.ErrorIs(t, err, storecenter.ErrShopifyInstallChecking)
		generation, err := repo.ShopifyInstallGeneration(ctx, identity)
		require.NoError(t, err)
		proof, err = repo.StageShopifyInstall(ctx, proof, identity, generation, "fixture-key", "fixture-ciphertext", now)
		require.NoError(t, err)
		replayed, err := repo.ClaimShopifyInstall(ctx, attempt, identity.AppID, identity.Domain, session, state, codeDigest, now)
		require.NoError(t, err)
		require.Equal(t, identity.Domain, replayed.Domain)
		require.Empty(t, replayed.Ciphertext)
		_, err = repo.ClaimShopifyInstall(ctx, attempt, identity.AppID, identity.Domain, session, state, strings.Repeat("e", 64), now)
		require.ErrorIs(t, err, storecenter.ErrVersionConflict)
		return proof
	}
	proof := stage()
	first.AttemptID, other.AttemptID = proof.AttemptID, proof.AttemptID
	_, err = repo.ReadShopifyBinding(ctx, first.OrganizationID, first.StoreID)
	require.ErrorIs(t, err, storecenter.ErrNotFound, "Shopify OAuth alone must not bind a Workbench store")
	stale := first
	stale.ExpectedStoreVersion++
	_, err = repo.SelectShopifyInstall(ctx, stale, session, now)
	require.ErrorIs(t, err, storecenter.ErrVersionConflict)
	_, err = repo.SelectShopifyInstall(ctx, first, foreignSession, now)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	_, err = repo.SelectShopifyInstall(ctx, first, session, now.Add(10*time.Minute))
	require.ErrorIs(t, err, storecenter.ErrVersionConflict)
	_, err = repo.SelectShopifyInstall(ctx, first, session, now)
	require.NoError(t, err)
	_, err = repo.SelectShopifyInstall(ctx, other, session, now)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	notice := storecenter.ShopifyLifecycleNotice{AppID: identity.AppID, ShopID: identity.ShopID,
		Topic: "app/uninstalled", EventID: "before-started-install-confirm", BodyHash: strings.Repeat("f", 64)}
	require.NoError(t, repo.ReceiveShopifyLifecycle(ctx, notice, now))
	_, err = repo.ConfirmShopifyInstall(ctx, first, session, identity, now, 0)
	require.ErrorIs(t, err, storecenter.ErrVersionConflict, "first binding must recheck after a new notice")
	revision, err := repo.ReadShopifyLifecycleRevision(ctx, identity)
	require.NoError(t, err)
	require.EqualValues(t, 1, revision)
	type confirmation struct {
		binding storecenter.ShopifyBinding
		err     error
	}
	ready := make(chan struct{})
	result := make(chan confirmation, 2)
	for range 2 {
		go func() {
			<-ready
			binding, err := repo.ConfirmShopifyInstall(ctx, first, session, identity, now, revision)
			result <- confirmation{binding, err}
		}()
	}
	close(ready)
	one, two := <-result, <-result
	require.NoError(t, one.err)
	require.NoError(t, two.err)
	require.Equal(t, one.binding, two.binding)
	bound := one.binding
	require.Equal(t, first.StoreID, bound.StoreID)
	require.EqualValues(t, 1, bound.Generation)
	selection, err := repo.InspectShopifyInstall(ctx, storecenter.ShopifyPairingKey{OrganizationID: first.OrganizationID, StoreID: first.StoreID, AttemptID: first.AttemptID}, identity.AppID, session, now)
	require.NoError(t, err)
	require.True(t, selection.Confirmed)
	require.Equal(t, first.ExpectedStoreVersion, selection.StoreVersion)
	replayed, err := repo.ConfirmShopifyInstall(ctx, first, session, identity, now, 0)
	require.NoError(t, err)
	require.Equal(t, bound, replayed)
	_, err = repo.ConfirmShopifyInstall(ctx, other, session, identity, now, 0)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	_, err = repo.ReadShopifyBinding(ctx, other.OrganizationID, other.StoreID)
	require.ErrorIs(t, err, storecenter.ErrNotFound)

	var ciphertext string
	require.NoError(t, db.Raw("SELECT ciphertext FROM workbench_store_shopify_installs WHERE attempt_id=?", proof.AttemptID).Row().Scan(&ciphertext))
	require.Empty(t, ciphertext)
	other.AttemptID = stage().AttemptID
	_, err = repo.SelectShopifyInstall(ctx, other, session, now)
	require.ErrorIs(t, err, storecenter.ErrAlreadyExists)
	require.NoError(t, repo.ExpireShopifyInstallProofs(ctx, now.Add(10*time.Minute)))
	_, err = repo.SelectShopifyInstall(ctx, first, session, now)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
}
