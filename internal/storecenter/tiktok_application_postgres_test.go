package storecenter_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	storeapp "task-processor/internal/app/storecenter"
	integration "task-processor/internal/integration/tiktokshop"
	"task-processor/internal/storecenter"
	tiktok "task-processor/internal/storecenter/tiktokshop"
)

type tiktokApplicationProvider struct {
	application tiktok.Application
	exchanges   int
	tokens      tiktok.TokenSet
}

func (p *tiktokApplicationProvider) Application() tiktok.Application { return p.application }
func (p *tiktokApplicationProvider) AuthorizationURL(state string) (string, error) {
	return "https://services.us.tiktokshop.com/open/authorize?service_id=fixture&state=" + url.QueryEscape(state), nil
}
func (p *tiktokApplicationProvider) Exchange(context.Context, string) (tiktok.TokenSet, error) {
	p.exchanges++
	return p.tokens, nil
}
func (p *tiktokApplicationProvider) Refresh(context.Context, string) (tiktok.TokenSet, error) {
	return tiktok.TokenSet{}, tiktok.ErrOutcomeUnknown
}
func (p *tiktokApplicationProvider) AuthorizedShops(context.Context, string) ([]tiktok.Shop, error) {
	return []tiktok.Shop{{ID: "actual-shop", Code: "USCODE", Cipher: "actual-cipher", Region: "US", SellerType: "CROSS_BORDER"}}, nil
}

type tiktokProductLiveAccess struct{}

func (tiktokProductLiveAccess) AuthorizeProductExecution(_ context.Context, subject storecenter.ProductExecutionSubject) (storecenter.ProductExecutionAuthorization, error) {
	return storecenter.ProductExecutionAuthorization{Allowed: true, Access: storecenter.StoreMemberAccess{OrganizationID: subject.OrganizationID, ActorID: subject.ActorID, MemberID: subject.MemberID, CanWrite: true}}, nil
}

func TestTikTokApplicationBindsProtectedCanonicalGrantAndReplaysCallback(t *testing.T) {
	db := newTikTokStoreDatabase(t)
	repo, err := storecenter.NewMemberScopedStoreRepository(db, &storeMemberAuthorizer{member: "original-member"})
	require.NoError(t, err)
	input := validCreateStoreInput()
	input.Platform = "tiktokshop"
	input.Region = "US"
	input.ExternalStoreID = "USCODE"
	input.OrganizationID = "org-a"
	input.ActorSubject = "subject-create"
	candidate, err := storecenter.NewStore(input)
	require.NoError(t, err)
	store, _, err := repo.CreateOrReplay(context.Background(), "org-a", candidate)
	require.NoError(t, err)
	provider := &tiktokApplicationProvider{application: tiktok.Application{AppKey: "fixture-app", ServiceID: "fixture-service", Revision: "r1", CallbackURL: "https://example.org/callback"}, tokens: tiktok.TokenSet{AccessToken: "private-access", RefreshToken: "private-refresh", OpenID: "actual-seller", UserType: 4, GrantedScopes: []string{"seller.authorization.info"}, AccessExpiresAt: time.Now().Add(time.Hour), RefreshExpiresAt: time.Now().Add(24 * time.Hour)}}
	protection, err := integration.NewCredentialProtection("test-key", make([]byte, 32))
	require.NoError(t, err)
	registry, err := storeapp.WithTikTokApplication(nil, provider, protection)
	require.NoError(t, err)
	app, err := storeapp.NewOfficialConnections(repo, registry)
	require.NoError(t, err)
	begin, err := app.Begin(context.Background(), storecenter.OfficialConnectionCommand{OrganizationID: "org-a", StoreID: store.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: store.Version(), ApplicationID: "fixture-app"})
	require.NoError(t, err)
	authURL, err := url.Parse(begin.AuthorizationURL)
	require.NoError(t, err)
	complete := storecenter.CompleteOfficialConnection{OrganizationID: "org-a", StoreID: store.ID(), AttemptID: begin.AttemptID, AppID: "fixture-app", State: authURL.Query().Get("state"), AuthCode: "one-use-code"}
	view, err := app.Complete(context.Background(), complete)
	require.NoError(t, err)
	require.Equal(t, storecenter.ConnectionStatusConnected, view.Status)
	require.Equal(t, "actual-shop", view.TikTok.ShopID)
	_, err = app.Complete(context.Background(), complete)
	require.NoError(t, err)
	require.Equal(t, 1, provider.exchanges)
	grant, _, err := repo.ReadTikTokGrant(context.Background(), "org-a", store.ID())
	require.NoError(t, err)
	require.NotContains(t, grant.Ciphertext, "private-access")
	private, err := protection.Open(tiktok.CredentialBinding{OrganizationID: "org-a", AppKey: "fixture-app", AppRevision: "r1", ID: grant.ID, Purpose: "grant", Generation: 1}, grant.KeyID, grant.Ciphertext)
	require.NoError(t, err)
	require.Equal(t, "private-refresh", private.RefreshToken)
	attempt, err := repo.ReadTikTokAttempt(context.Background(), "org-a", store.ID(), begin.AttemptID)
	require.NoError(t, err)
	require.Empty(t, attempt.Ciphertext)
	executionCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	subject := storecenter.ProductExecutionSubject{OrganizationID: "org-a", ActorID: "subject-create", MemberID: "original-member", Purpose: storecenter.ProductPurposeRules}
	_, err = repo.ReadProductExecution(executionCtx, subject, store.ID(), tiktokProductLiveAccess{}, time.Now().UTC())
	require.Error(t, err, "pending service cannot acquire product access")
	require.NoError(t, db.Exec(`UPDATE workbench_stores SET service_status='active',service_started_at=?,service_expires_at=? WHERE organization_id='org-a' AND id=?`, time.Now().Add(-time.Minute), time.Now().Add(time.Hour), store.ID()).Error)
	material, err := repo.ReadProductExecution(executionCtx, subject, store.ID(), tiktokProductLiveAccess{}, time.Now().UTC())
	require.NoError(t, err)
	require.NotNil(t, material.TikTok)
	require.Equal(t, grant, material.TikTok.Grant)
	require.Empty(t, material.Attempt.Ciphertext)
	subject.MemberID = "replacement-member"
	_, err = repo.ReadProductExecution(executionCtx, subject, store.ID(), tiktokProductLiveAccess{}, time.Now().UTC())
	require.Error(t, err)
}

func TestTikTokStoreRuntimePrivilegesRequireExplicitCapability(t *testing.T) {
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
	require.Error(t, storecenter.VerifyRuntimePermissionsForCapabilities(ctx, db, storecenter.RuntimeCapabilities{TikTokShop: true}))
	require.NoError(t, db.Exec(`RESET ROLE`).Error)
	require.NoError(t, db.Exec(`GRANT SELECT,INSERT,UPDATE ON workbench_store_tiktok_oauth_grants,workbench_store_tiktok_oauth_attempts,workbench_store_tiktok_shop_bindings,workbench_store_tiktok_oauth_refresh_operations TO store_center_runtime`).Error)
	require.NoError(t, db.Exec(`GRANT SELECT,INSERT ON workbench_store_tiktok_authorization_events TO store_center_runtime`).Error)
	require.NoError(t, db.Exec(`SET ROLE store_center_runtime`).Error)
	require.Error(t, storecenter.VerifyRuntimePermissions(ctx, db))
	require.NoError(t, storecenter.VerifyRuntimePermissionsForCapabilities(ctx, db, storecenter.RuntimeCapabilities{TikTokShop: true}))
	require.NoError(t, db.Exec(`RESET ROLE`).Error)
}
