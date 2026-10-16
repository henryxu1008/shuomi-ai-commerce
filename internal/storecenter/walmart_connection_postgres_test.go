package storecenter_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sync"
	storeapp "task-processor/internal/app/storecenter"
	integration "task-processor/internal/integration/walmart"
	marketplace "task-processor/internal/marketplace/walmart"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
	auth "task-processor/internal/storecenter/walmart"
)

func TestWalmartDurableCallbackClaimAndRefreshFence(t *testing.T) {
	db := newTikTokStoreDatabase(t)
	a := &storeMemberAuthorizer{member: "original-member"}
	repo, e := storecenter.NewMemberScopedStoreRepository(db, a)
	require.NoError(t, e)
	in := validCreateStoreInput()
	in.Platform = "walmart"
	in.Region = "US"
	in.OrganizationID = "org-a"
	in.ActorSubject = "subject-create"
	candidate, e := storecenter.NewStore(in)
	require.NoError(t, e)
	s, _, e := repo.CreateOrReplay(context.Background(), "org-a", candidate)
	require.NoError(t, e)
	now := time.Now().UTC()
	hash := sha256.Sum256([]byte("state"))
	app := auth.Application{ClientID: "app", Revision: "r1", Environment: "production", CallbackURL: "https://example.org/callback"}
	cmd := storecenter.OfficialConnectionCommand{OrganizationID: "org-a", StoreID: s.ID(), ApplicationID: "app", AttemptID: uuid.NewString(), ExpectedStoreVersion: s.Version()}
	attempt, e := repo.BeginWalmartConnection(context.Background(), cmd, app, hex.EncodeToString(hash[:]), "01234567890123456789012345678901", now)
	require.NoError(t, e)
	callback := storecenter.CompleteWalmartConnection{OrganizationID: "org-a", StoreID: s.ID(), AttemptID: attempt.AttemptID, AppID: "app", State: "state", SellerID: "verified-partner", CallbackType: "auth"}
	var wg sync.WaitGroup
	results := make(chan bool, 8)
	errs := make(chan error, 8)
	for range 8 {
		wg.Go(func() {
			_, sent, e := repo.ClaimWalmartExchange(context.Background(), callback, hex.EncodeToString(hash[:]), now)
			results <- sent
			errs <- e
		})
	}
	wg.Wait()
	close(results)
	close(errs)
	count := 0
	for sent := range results {
		if sent {
			count++
		}
	}
	for e := range errs {
		require.NoError(t, e)
	}
	require.Equal(t, 1, count)
	attempt, e = repo.ReadWalmartAttempt(context.Background(), "org-a", s.ID(), attempt.AttemptID)
	require.NoError(t, e)
	require.NoError(t, repo.SaveWalmartCandidate(context.Background(), attempt, "key", "sealed-candidate", now))
	g := storecenter.WalmartGrant{OrganizationID: "org-a", ID: uuid.NewString(), StoreID: s.ID(), AppID: "app", AppRevision: "r1", Environment: "production", MerchantIdentityHash: storecenter.WalmartMerchantIdentity("production", "verified-partner"), SellerID: "verified-partner", MerchantName: "Seller", Generation: 1, Version: 1, Status: "active", KeyID: "key", Ciphertext: "sealed-grant", ScopesHash: hex.EncodeToString(hash[:]), AccessExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now}
	view, e := repo.BindWalmartConnection(context.Background(), storecenter.WalmartBindingIntent{Attempt: attempt, Grant: g}, now)
	require.NoError(t, e)
	require.Equal(t, storecenter.ConnectionStatusConnected, view.Status)
	persisted, e := repo.ReadWalmartAttempt(context.Background(), "org-a", s.ID(), attempt.AttemptID)
	require.NoError(t, e)
	require.Empty(t, persisted.Ciphertext)
	op := uuid.NewString()
	claim, e := repo.ClaimWalmartRefresh(context.Background(), "org-a", s.ID(), op, 1, hex.EncodeToString(hash[:]), now)
	require.NoError(t, e)
	require.True(t, claim.Dispatch)
	replay, e := repo.ClaimWalmartRefresh(context.Background(), "org-a", s.ID(), op, 1, hex.EncodeToString(hash[:]), now)
	require.NoError(t, e)
	require.False(t, replay.Dispatch)
	_, e = repo.ClaimWalmartRefresh(context.Background(), "org-a", s.ID(), uuid.NewString(), 1, hex.EncodeToString(hash[:]), now)
	require.Error(t, e)
	current, e := repo.Get(context.Background(), "org-a", s.ID())
	require.NoError(t, e)
	cmd.AttemptID = uuid.NewString()
	cmd.ExpectedStoreVersion = current.Version()
	_, e = repo.BeginWalmartConnection(context.Background(), cmd, app, hex.EncodeToString(hash[:]), "01234567890123456789012345678901", now)
	require.NoError(t, e)
	next := claim.Grant
	next.Generation++
	next.Version++
	next.Status = "active"
	require.Error(t, repo.CompleteWalmartRefresh(context.Background(), claim, next, now))
	// Permanent identity is independent of app, token and local connection life.
	require.Equal(t, storecenter.WalmartMerchantIdentity("production", "verified-partner"), g.MerchantIdentityHash)
	require.NotEqual(t, storecenter.WalmartMerchantIdentity("sandbox", "verified-partner"), g.MerchantIdentityHash)
}

func TestWalmartSchemaDefinitionsRejectWeakenedContract(t *testing.T) {
	db := newTikTokStoreDatabase(t)
	ctx := context.Background()
	require.NoError(t, storecenter.VerifyCurrentSchema(ctx, db))
	require.NoError(t, db.Exec(`ALTER TABLE workbench_store_walmart_oauth_grants DROP CONSTRAINT walmart_grant_merchant; ALTER TABLE workbench_store_walmart_oauth_grants ADD CONSTRAINT walmart_grant_merchant UNIQUE(organization_id,environment,app_id,merchant_identity_hash)`).Error)
	require.Error(t, storecenter.VerifyCurrentSchema(ctx, db), "same-name tenant-scoped uniqueness must fail preflight")
}

func TestWalmartStoreRuntimePrivilegesRequireExplicitCapability(t *testing.T) {
	db := newTikTokStoreDatabase(t)
	for _, statement := range []string{
		`CREATE ROLE store_center_runtime`, `REVOKE CREATE ON SCHEMA public FROM PUBLIC`, `REVOKE ALL ON DATABASE tiktok_schema FROM PUBLIC`, `GRANT CONNECT ON DATABASE tiktok_schema TO store_center_runtime`, `GRANT USAGE ON SCHEMA public TO store_center_runtime`,
		`GRANT SELECT,INSERT,UPDATE ON workbench_stores,workbench_store_member_grants,workbench_store_service_operations,workbench_store_connections,workbench_store_connection_attempts,workbench_store_groups,workbench_store_profiles TO store_center_runtime`,
		`GRANT SELECT,INSERT ON workbench_store_audit_logs,workbench_store_member_grant_operations,workbench_store_merchant_bindings,workbench_store_metadata_operations TO store_center_runtime`,
	} {
		require.NoError(t, db.Exec(statement).Error)
	}
	ctx := context.Background()
	require.NoError(t, db.Exec(`SET ROLE store_center_runtime`).Error)
	require.NoError(t, storecenter.VerifyRuntimePermissions(ctx, db))
	require.Error(t, storecenter.VerifyRuntimePermissionsForCapabilities(ctx, db, storecenter.RuntimeCapabilities{WalmartUS: true}))
	require.NoError(t, db.Exec(`RESET ROLE`).Error)
	require.NoError(t, db.Exec(`GRANT SELECT,INSERT,UPDATE ON workbench_store_walmart_oauth_grants,workbench_store_walmart_oauth_attempts TO store_center_runtime`).Error)
	require.NoError(t, db.Exec(`SET ROLE store_center_runtime`).Error)
	require.Error(t, storecenter.VerifyRuntimePermissions(ctx, db))
	require.NoError(t, storecenter.VerifyRuntimePermissionsForCapabilities(ctx, db, storecenter.RuntimeCapabilities{WalmartUS: true}))
	require.NoError(t, db.Exec(`RESET ROLE`).Error)
}

func TestWalmartPermanentMerchantOccupancySurvivesDisconnectAndApplicationChange(t *testing.T) {
	db := newTikTokStoreDatabase(t)
	access := &storeMemberAuthorizer{member: "original-member"}
	repo, e := storecenter.NewMemberScopedStoreRepository(db, access)
	require.NoError(t, e)
	ctx := context.Background()
	now := time.Now().UTC()
	hash := storecenter.WalmartMerchantIdentity("production", "merchant")
	makeStore := func(org string) *storecenter.Store {
		in := validCreateStoreInput()
		in.ID = uuid.NewString()
		in.ExternalStoreID = uuid.NewString()
		in.CreateIdempotencyKey = uuid.NewString()
		in.Platform = "walmart"
		in.Region = "US"
		in.OrganizationID = org
		in.ActorSubject = "subject-create"
		candidate, e := storecenter.NewStore(in)
		require.NoError(t, e)
		created, _, e := repo.CreateOrReplay(ctx, org, candidate)
		require.NoError(t, e)
		return created
	}
	bind := func(s *storecenter.Store, appID string) error {
		app := auth.Application{ClientID: appID, Revision: "r1", Environment: "production", CallbackURL: "https://example.org/callback"}
		c := storecenter.OfficialConnectionCommand{OrganizationID: s.OrganizationID(), StoreID: s.ID(), ApplicationID: appID, AttemptID: uuid.NewString(), ExpectedStoreVersion: s.Version()}
		a, e := repo.BeginWalmartConnection(ctx, c, app, hash, "01234567890123456789012345678901", now)
		require.NoError(t, e)
		cb := storecenter.CompleteWalmartConnection{OrganizationID: s.OrganizationID(), StoreID: s.ID(), AttemptID: a.AttemptID, AppID: appID, SellerID: "merchant", CallbackType: "auth"}
		a, dispatch, e := repo.ClaimWalmartExchange(ctx, cb, hash, now)
		require.NoError(t, e)
		require.True(t, dispatch)
		require.NoError(t, repo.SaveWalmartCandidate(ctx, a, "key", "sealed", now))
		g := storecenter.WalmartGrant{OrganizationID: s.OrganizationID(), StoreID: s.ID(), ID: uuid.NewString(), AppID: appID, AppRevision: "r1", Environment: "production", MerchantIdentityHash: hash, SellerID: "merchant", MerchantName: "Merchant", Generation: 1, Version: 1, Status: "active", KeyID: "key", Ciphertext: "sealed-grant", ScopesHash: hash, AccessExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now}
		_, e = repo.BindWalmartConnection(ctx, storecenter.WalmartBindingIntent{Attempt: a, Grant: g}, now)
		return e
	}
	original := makeStore("org-a")
	require.NoError(t, bind(original, "app-a"))
	original, e = repo.Get(ctx, "org-a", original.ID())
	require.NoError(t, e)
	_, e = repo.DisconnectWalmartConnection(ctx, storecenter.OfficialConnectionCommand{OrganizationID: "org-a", StoreID: original.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: original.Version()}, now)
	require.NoError(t, e)
	require.ErrorIs(t, bind(makeStore("org-a"), "app-b"), storecenter.ErrAlreadyExists, "different application cannot take original Store merchant")
	require.ErrorIs(t, bind(makeStore("org-b"), "app-c"), storecenter.ErrAlreadyExists, "another organization cannot claim disconnected merchant")
	var count int64
	require.NoError(t, db.Table("workbench_store_merchant_bindings").Where("identity_hash=? AND organization_id='org-a' AND store_id=?", hash, original.ID()).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

type walmartApplicationProvider struct {
	app                  auth.Application
	tokens               auth.TokenSet
	exchanges, refreshes int
	partner              string
}

func (p *walmartApplicationProvider) Application() auth.Application { return p.app }
func (p *walmartApplicationProvider) AuthorizationURL(state, nonce string) (string, error) {
	return "https://login.account.wal-mart.com/authorize?state=" + url.QueryEscape(state) + "&nonce=" + url.QueryEscape(nonce), nil
}
func (p *walmartApplicationProvider) Exchange(context.Context, string, string) (auth.TokenSet, error) {
	p.exchanges++
	return p.tokens, nil
}
func (p *walmartApplicationProvider) Refresh(_ context.Context, old auth.TokenSet) (auth.TokenSet, error) {
	p.refreshes++
	old.AccessToken = "private-new-access"
	old.AccessExpiresAt = time.Now().Add(time.Hour)
	return old, nil
}
func (p *walmartApplicationProvider) TokenDetail(_ context.Context, t auth.TokenSet) (auth.TokenDetail, error) {
	return auth.TokenDetail{Valid: true, ExpiresAt: t.AccessExpiresAt, Scopes: map[string]string{"item": "full_access", "feeds": "view_only"}}, nil
}
func (p *walmartApplicationProvider) Merchant(context.Context, auth.TokenSet) (auth.Merchant, error) {
	return auth.Merchant{PartnerID: p.partner, Name: "Verified merchant"}, nil
}
func TestWalmartApplicationConfirmsVerifiedMerchantAndRefreshesProtectedGrant(t *testing.T) {
	db := newTikTokStoreDatabase(t)
	access := &storeMemberAuthorizer{member: "original-member"}
	repo, e := storecenter.NewMemberScopedStoreRepository(db, access)
	require.NoError(t, e)
	ctx := context.Background()
	in := validCreateStoreInput()
	in.Platform = "walmart"
	in.Region = "US"
	in.OrganizationID = "org-a"
	in.ActorSubject = "subject-create"
	candidate, e := storecenter.NewStore(in)
	require.NoError(t, e)
	s, _, e := repo.CreateOrReplay(ctx, "org-a", candidate)
	require.NoError(t, e)
	p := &walmartApplicationProvider{app: auth.Application{ClientID: "app", Revision: "r1", Environment: "sandbox", CallbackURL: "https://example.org/callback"}, tokens: auth.TokenSet{AccessToken: "private-access", RefreshToken: "private-refresh", SellerID: "partner", AccessExpiresAt: time.Now().Add(time.Hour), RefreshExpiresAt: time.Now().Add(24 * time.Hour)}, partner: "another-partner"}
	protection, e := integration.NewCredentialProtection("private-key", make([]byte, 32))
	require.NoError(t, e)
	registry, e := storeapp.WithWalmartApplication(nil, p, protection)
	require.NoError(t, e)
	app, e := storeapp.NewOfficialConnections(repo, registry)
	require.NoError(t, e)
	begin, e := app.Begin(ctx, storecenter.OfficialConnectionCommand{OrganizationID: "org-a", StoreID: s.ID(), ApplicationID: "app", AttemptID: uuid.NewString(), ExpectedStoreVersion: s.Version()})
	require.NoError(t, e)
	link, e := url.Parse(begin.AuthorizationURL)
	require.NoError(t, e)
	complete := storecenter.CompleteOfficialConnection{OrganizationID: "org-a", StoreID: s.ID(), AttemptID: begin.AttemptID, AppID: "app", State: "wrong", AuthCode: "one-use-code", SellerID: "partner", CallbackType: "auth"}
	_, e = app.Complete(ctx, complete)
	require.Error(t, e)
	require.Zero(t, p.exchanges)
	complete.State = link.Query().Get("state")
	_, e = app.Complete(ctx, complete)
	require.ErrorIs(t, e, storecenter.ErrOfficialAuthorizationRejected)
	require.Equal(t, 1, p.exchanges)
	access.member = "replacement"
	_, e = app.ResumeQuery(ctx, "org-a", s.ID(), begin.AttemptID)
	require.Error(t, e)
	access.member = "original-member"
	p.partner = "partner"
	view, e := app.ResumeQuery(ctx, "org-a", s.ID(), begin.AttemptID)
	require.NoError(t, e)
	require.Equal(t, "merchant_confirmation_required", view.State)
	require.NotEqual(t, storecenter.ConnectionStatusConnected, view.Status)
	view, e = app.Complete(ctx, storecenter.CompleteOfficialConnection{OrganizationID: "org-a", StoreID: s.ID(), AttemptID: begin.AttemptID, AppID: "app", MerchantConfirmation: "partner"})
	require.NoError(t, e)
	require.Equal(t, storecenter.ConnectionStatusConnected, view.Status)
	raw, e := json.Marshal(view)
	require.NoError(t, e)
	require.NotContains(t, string(raw), "private-")
	grant, e := repo.ReadWalmartGrant(ctx, "org-a", s.ID())
	require.NoError(t, e)
	require.NotContains(t, grant.Ciphertext, "private-access")
	tokens, e := protection.Open(grant.CredentialBinding(), grant.KeyID, grant.Ciphertext)
	require.NoError(t, e)
	require.Equal(t, "private-refresh", tokens.RefreshToken)
	view, e = app.RefreshWalmart(ctx, "org-a", s.ID(), uuid.NewString(), grant.Generation)
	require.NoError(t, e)
	require.EqualValues(t, 2, view.Walmart.Generation)
	require.Equal(t, 1, p.refreshes)
	require.Equal(t, 1, p.exchanges)
	refreshed, e := repo.ReadWalmartGrant(ctx, "org-a", s.ID())
	require.NoError(t, e)
	require.Equal(t, grant.ID, refreshed.ID)
	beforeSKU := marketplace.SellerSKU(marketplace.SKUNamespace(grant.MerchantIdentityHash, grant.ID), "product", "source-sku")
	p.app.ClientID = "replacement-app"
	current, e := repo.Get(ctx, "org-a", s.ID())
	require.NoError(t, e)
	nextBegin, e := app.Begin(ctx, storecenter.OfficialConnectionCommand{OrganizationID: "org-a", StoreID: s.ID(), ApplicationID: p.app.ClientID, AttemptID: uuid.NewString(), ExpectedStoreVersion: current.Version()})
	require.NoError(t, e)
	nextLink, e := url.Parse(nextBegin.AuthorizationURL)
	require.NoError(t, e)
	_, e = app.Complete(ctx, storecenter.CompleteOfficialConnection{OrganizationID: "org-a", StoreID: s.ID(), AttemptID: nextBegin.AttemptID, AppID: p.app.ClientID, State: nextLink.Query().Get("state"), AuthCode: "new-one-use-code", SellerID: "partner", CallbackType: "auth"})
	require.NoError(t, e)
	_, e = app.Complete(ctx, storecenter.CompleteOfficialConnection{OrganizationID: "org-a", StoreID: s.ID(), AttemptID: nextBegin.AttemptID, AppID: p.app.ClientID, MerchantConfirmation: "partner"})
	require.NoError(t, e)
	rebound, e := repo.ReadWalmartGrant(ctx, "org-a", s.ID())
	require.NoError(t, e)
	require.Equal(t, grant.ID, rebound.ID)
	require.Equal(t, beforeSKU, marketplace.SellerSKU(marketplace.SKUNamespace(rebound.MerchantIdentityHash, rebound.ID), "product", "source-sku"), "app replacement and re-consent retain random SKU namespace")
}
