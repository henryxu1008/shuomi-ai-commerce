package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"task-processor/internal/storecenter"
	"task-processor/internal/workbenchcontext"
)

type changingOverviewResolver struct {
	identity  authidentity.AuthenticatedIdentity
	calls     int
	denyAfter int
}

func (r *changingOverviewResolver) Resolve(_ context.Context, _ httproute.OrganizationAccessPolicy, _ workbenchcontext.ResolveInput) (authidentity.AuthenticatedIdentity, error) {
	r.calls++
	out := r.identity
	denyAfter := r.denyAfter
	if denyAfter == 0 {
		denyAfter = 1
	}
	if r.calls > denyAfter {
		out.Roles = []string{"no-store-access"}
	}
	return out, nil
}

func TestStoreHistoryProjectionRealAdapterFailsClosedAfterSnapshot(t *testing.T) {
	now := time.Now()
	identity := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", HomeOrganizationID: "home", EffectiveMemberID: "member", UserID: "actor", TokenExpiresAt: now.Add(time.Hour), Roles: []string{"listingkit_admin"}, OrganizationGrants: []authidentity.OrganizationGrant{{OrganizationID: "org", AuthorizationID: "member"}}}
	ctx, err := (productReviewCapabilityBinder{now: func() time.Time { return now }}).Bind(authidentity.WithAuthenticatedIdentity(context.Background(), identity), "Bearer synthetic-token")
	require.NoError(t, err)
	resolver := &changingOverviewResolver{identity: identity, denyAfter: 2}
	auth, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	app := &aiWorkbenchApplication{resolver: resolver, authorizer: auth, allowedOrganizationIDs: []string{"org"}}
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	raw, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = raw.Close() })
	require.NoError(t, storecenter.AutoMigrateStoreRepository(db))
	require.NoError(t, storecenter.AutoMigrateAuditRepository(db))
	catalog, err := storecenter.NewCatalog(db, currentStoreMemberAuthorizer{authorizer: auth}, nil)
	require.NoError(t, err)
	app.attachStoreOverview(catalog)
	messages := []aiworkbench.Message{{Kind: "STORE_OVERVIEW_REF", StoreQueryStatus: "ACTIVE"}, {Kind: "STORE_OVERVIEW_REF", StoreQueryStatus: "ALL_NON_DELETED"}}
	projected := app.projectStoreMessages(ctx, aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}, messages)
	require.Equal(t, 3, resolver.calls)
	for _, message := range projected {
		require.Nil(t, message.StoreOverview)
		require.True(t, message.StoreOverviewUnavailable)
	}
}

func TestStoreOverviewActualResolverRevocationDuringReadFailsClosed(t *testing.T) {
	now := time.Now()
	identity := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", HomeOrganizationID: "home", EffectiveMemberID: "member", UserID: "actor", TokenExpiresAt: now.Add(time.Hour), Roles: []string{"listingkit_admin"}, OrganizationGrants: []authidentity.OrganizationGrant{{OrganizationID: "org", AuthorizationID: "member"}}}
	ctx, err := (productReviewCapabilityBinder{now: func() time.Time { return now }}).Bind(authidentity.WithAuthenticatedIdentity(context.Background(), identity), "Bearer synthetic-token")
	require.NoError(t, err)
	resolver := &changingOverviewResolver{identity: identity}
	auth, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	app := &aiWorkbenchApplication{resolver: resolver, authorizer: auth, allowedOrganizationIDs: []string{"org"}}
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	raw, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = raw.Close() })
	require.NoError(t, storecenter.AutoMigrateStoreRepository(db))
	require.NoError(t, storecenter.AutoMigrateAuditRepository(db))
	catalog, err := storecenter.NewCatalog(db, currentStoreMemberAuthorizer{authorizer: auth}, nil)
	require.NoError(t, err)
	expected := storecenter.OverviewIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	out, err := catalog.VisibleOverview(ctx, expected, storecenter.OverviewAll, newStoreOverviewLiveAuthorizer(app, expected))
	require.Error(t, err)
	require.Nil(t, out.Platforms)
	require.Equal(t, 2, resolver.calls)
}

func TestStoreOverviewLiveAuthorizerResolvesAndBindsIdentity(t *testing.T) {
	now := time.Now()
	original := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", HomeOrganizationID: "home", EffectiveMemberID: "member", UserID: "actor", TokenExpiresAt: now.Add(time.Hour)}
	ctx, err := (productReviewCapabilityBinder{now: func() time.Time { return now }}).Bind(authidentity.WithAuthenticatedIdentity(context.Background(), original), "Bearer synthetic-token")
	require.NoError(t, err)
	current := original
	current.Roles = []string{"listingkit_admin"}
	current.OrganizationGrants = []authidentity.OrganizationGrant{{OrganizationID: "org", AuthorizationID: "member"}}
	resolver := &productReviewResolverStub{value: current}
	auth, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	app := &aiWorkbenchApplication{resolver: resolver, authorizer: auth, allowedOrganizationIDs: []string{"org"}}
	live := newStoreOverviewLiveAuthorizer(app, storecenter.OverviewIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member"})
	got, err := live.AuthorizeLiveOverview(ctx, "org")
	require.NoError(t, err)
	require.True(t, got.Administrator)
	full := &aiWorkbenchApplication{agent: &productAgentApplication{resolver: resolver, authorizer: auth, config: ProductAgentDependencies{AllowedOrganizationIDs: []string{"org"}}}}
	fullLive := newStoreOverviewLiveAuthorizer(full, storecenter.OverviewIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member"})
	got, err = fullLive.AuthorizeLiveOverview(ctx, "org")
	require.NoError(t, err)
	require.True(t, got.Administrator)
	resolver.value.Roles = []string{"listingkit_operator"}
	_, err = live.AuthorizeLiveOverview(ctx, "org")
	require.Error(t, err) // A role lacking scoped admission fails closed.
	_, err = fullLive.AuthorizeLiveOverview(ctx, "org")
	require.Error(t, err)
	resolver.value = current
	resolver.value.EffectiveMemberID = "rejoined"
	_, err = live.AuthorizeLiveOverview(ctx, "org")
	require.Error(t, err)
	resolver.value = current
	resolver.value.TenantID = "other"
	_, err = live.AuthorizeLiveOverview(ctx, "org")
	require.Error(t, err)
	resolver.value = current
	resolver.value.Roles = []string{"no-store-access"}
	_, err = live.AuthorizeLiveOverview(ctx, "org")
	require.Error(t, err)
}
