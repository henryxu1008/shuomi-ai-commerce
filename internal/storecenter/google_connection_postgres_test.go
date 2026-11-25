package storecenter_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
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
			Target: google.Target{AccountID: "123", DataSourceID: "456", ContentLanguage: "en", FeedLabel: "US", OfferPrefix: prefix},
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

func TestGoogleStoreRuntimeRequiresNarrowGrants(t *testing.T) {
	ctx := context.Background()
	db := newTikTokStoreDatabase(t)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, storecenter.InstallGoogleMerchantSchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())

	for _, statement := range []string{
		`CREATE ROLE store_center_runtime`,
		`REVOKE ALL ON DATABASE tiktok_schema FROM PUBLIC`,
		`REVOKE CREATE ON SCHEMA public FROM PUBLIC`,
		`GRANT CONNECT ON DATABASE tiktok_schema TO store_center_runtime`,
		`GRANT USAGE ON SCHEMA public TO store_center_runtime`,
		`GRANT SELECT,INSERT,UPDATE ON workbench_stores,workbench_store_member_grants,workbench_store_service_operations,workbench_store_connections,workbench_store_connection_attempts,workbench_store_groups,workbench_store_profiles TO store_center_runtime`,
		`GRANT SELECT,INSERT ON workbench_store_audit_logs,workbench_store_member_grant_operations,workbench_store_merchant_bindings,workbench_store_metadata_operations TO store_center_runtime`,
	} {
		require.NoError(t, db.Exec(statement).Error)
	}

	googleEnabled := storecenter.RuntimeCapabilities{GoogleMerchant: true}
	require.NoError(t, db.Exec("SET ROLE store_center_runtime").Error)
	require.NoError(t, storecenter.VerifyRuntimePermissions(ctx, db))
	require.ErrorContains(t, storecenter.VerifyRuntimePermissionsForCapabilities(ctx, db, googleEnabled), "Google Merchant required Store grants unavailable")

	require.NoError(t, db.Exec("RESET ROLE").Error)
	require.NoError(t, db.Exec("GRANT SELECT, INSERT, UPDATE ON workbench_store_google_oauth_attempts, workbench_store_google_oauth_grants, workbench_store_google_oauth_refresh TO store_center_runtime").Error)
	require.NoError(t, db.Exec("SET ROLE store_center_runtime").Error)
	require.NoError(t, storecenter.VerifyRuntimePermissionsForCapabilities(ctx, db, googleEnabled))
	require.NoError(t, storecenter.VerifyRuntimePermissions(ctx, db))

	require.NoError(t, db.Exec("RESET ROLE").Error)
	require.NoError(t, db.Exec("GRANT DELETE ON workbench_store_google_oauth_grants TO store_center_runtime").Error)
	require.NoError(t, db.Exec("SET ROLE store_center_runtime").Error)
	require.ErrorContains(t, storecenter.VerifyRuntimePermissionsForCapabilities(ctx, db, googleEnabled), "store runtime permissions do not match the admitted boundary")
}

func TestGoogleStoreSchemaPreservesIsolatedAmazonAuthorization(t *testing.T) {
	ctx := context.Background()
	db := newTikTokStoreDatabase(t)
	pool, err := db.DB()
	require.NoError(t, err)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, storecenter.InstallAmazonAuthorizationSchemaTx(ctx, tx))
	require.NoError(t, storecenter.InstallGoogleMerchantSchemaTx(ctx, tx))
	require.NoError(t, storecenter.InstallGoogleMerchantSchemaTx(ctx, tx), "owner replay preserves both installed platform contracts")
	require.NoError(t, tx.Commit())

	require.NoError(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, db, storecenter.RuntimeCapabilities{
		AmazonAuthorization: true, GoogleMerchant: true,
	}))
	require.NoError(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, db, storecenter.RuntimeCapabilities{
		AmazonAuthorization: true,
	}))
	require.Error(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, db, storecenter.RuntimeCapabilities{
		GoogleMerchant: true,
	}), "an isolated Amazon schema still requires explicit admission")
}

func TestGoogleStoreSchemaInstallsRenewalOverFirstRelease(t *testing.T) {
	ctx := context.Background()
	db := newTikTokStoreDatabase(t)
	pool, err := db.DB()
	require.NoError(t, err)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, storecenter.InstallGoogleMerchantSchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())
	// Simulate a correctly installed Google schema before renewal was admitted.
	require.NoError(t, db.Exec("DROP TABLE workbench_store_google_oauth_refresh").Error)
	enabled := storecenter.RuntimeCapabilities{GoogleMerchant: true, TikTokShop: true, WalmartUS: true}
	require.ErrorContains(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, db, enabled), "renewal schema unavailable")
	tx, err = pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, storecenter.InstallGoogleMerchantSchemaTx(ctx, tx))
	require.NoError(t, storecenter.InstallGoogleMerchantSchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())
	require.NoError(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, db, enabled))
	require.NoError(t, db.Exec("ALTER TABLE workbench_store_google_oauth_refresh DROP CONSTRAINT google_refresh_phase").Error)
	require.ErrorContains(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, db, enabled), "renewal schema unavailable")
}
