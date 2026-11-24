package storecenter_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	market "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	google "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
	"github.com/stretchr/testify/require"
)

func TestGoogleMerchantConnectionClaimsCallbackAndPreservesExclusiveAccount(t *testing.T) {
	ctx := context.Background()
	db := newTikTokStoreDatabase(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	tx, err := sqlDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, storecenter.InstallGoogleMerchantSchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())

	repo, err := storecenter.NewMemberScopedStoreRepository(db, &storeMemberAuthorizer{member: "original-member", admin: true})
	require.NoError(t, err)
	newStore := func(org string) *storecenter.Store {
		input := validCreateStoreInput()
		input.ID, input.CreateIdempotencyKey = uuid.NewString(), uuid.NewString()
		input.OrganizationID, input.Platform, input.ExternalStoreID = org, "googlemerchant", ""
		candidate, err := storecenter.NewStore(input)
		require.NoError(t, err)
		store, _, err := repo.CreateOrReplay(ctx, org, candidate)
		require.NoError(t, err)
		return store
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	app := google.Application{ID: "google-client", Revision: "r1", CallbackURL: "https://example.org/google"}
	start := func(store *storecenter.Store) storecenter.GoogleAttempt {
		key := uuid.NewString()
		digest := sha256.Sum256([]byte(key))
		hash := hex.EncodeToString(digest[:])
		command := storecenter.OfficialConnectionCommand{
			OrganizationID: store.OrganizationID(), StoreID: store.ID(), AttemptID: key,
			ApplicationID: app.ID, ExpectedStoreVersion: store.Version(),
		}
		_, err := repo.BeginGoogleConnection(ctx, command, app, hash, now)
		require.NoError(t, err)
		attempt, err := repo.ClaimGoogleExchange(ctx, store.OrganizationID(), store.ID(), key, hash, app, now)
		require.NoError(t, err)
		_, err = repo.ClaimGoogleExchange(ctx, store.OrganizationID(), store.ID(), key, hash, app, now)
		require.Error(t, err)
		require.NoError(t, repo.SaveGoogleCandidate(ctx, attempt, "key", "ciphertext", now))
		attempt, err = repo.ReadGoogleCandidate(ctx, store.OrganizationID(), store.ID(), key, now)
		require.NoError(t, err)
		return attempt
	}
	grant := func(attempt storecenter.GoogleAttempt, prefix string) storecenter.GoogleGrant {
		return storecenter.GoogleGrant{
			OrganizationID: attempt.OrganizationID, StoreID: attempt.StoreID, ID: attempt.GrantID,
			AppID: attempt.AppID, AppRevision: attempt.AppRevision, Generation: attempt.Generation,
			Target: market.Target{AccountID: "123", DataSourceID: "456", ContentLanguage: "en", FeedLabel: "US", OfferPrefix: prefix},
			KeyID:  "key", Ciphertext: "sealed", AccessExpiresAt: now.Add(time.Hour),
		}
	}
	first := newStore("org-one")
	firstAttempt := start(first)
	connected, err := repo.BindGoogleConnection(ctx, firstAttempt, grant(firstAttempt, "sm-0123456789abcdef0123-"), now)
	require.NoError(t, err)
	require.Equal(t, "123", connected.Target.AccountID)
	_, err = repo.ReadGoogleCandidate(ctx, first.OrganizationID(), first.ID(), firstAttempt.AttemptID, now)
	require.Error(t, err)
	secondAttempt := start(newStore("org-two"))
	_, err = repo.BindGoogleConnection(ctx, secondAttempt, grant(secondAttempt, "sm-0123456789abcdef0124-"), now)
	require.ErrorIs(t, err, storecenter.ErrAlreadyExists)
}
