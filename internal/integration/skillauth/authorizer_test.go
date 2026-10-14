package skillauth

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"task-processor/internal/workbenchcontext"
	"testing"
	"time"
)

type liveResolver struct {
	identity authidentity.AuthenticatedIdentity
	calls    int
	changed  bool
}

func (r *liveResolver) Resolve(_ context.Context, policy httproute.OrganizationAccessPolicy, input workbenchcontext.ResolveInput) (authidentity.AuthenticatedIdentity, error) {
	r.calls++
	if policy != httproute.OrganizationAccessPolicyLiveWrite || input.RequestedOrganizationID != r.identity.EffectiveOrganizationID {
		panic("unexpected scope or freshness")
	}
	out := r.identity
	if r.changed {
		out.Roles = []string{"listingkit_viewer"}
	}
	return out, nil
}
func TestPersonalIdentityDoesNotRequireOrBorrowAnOrganization(t *testing.T) {
	policy, e := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, e)
	a, e := New(nil, policy)
	require.NoError(t, e)
	identity := authidentity.AuthenticatedIdentity{UserID: "alice", HomeOrganizationID: "home-is-not-active", TokenExpiresAt: time.Now().Add(time.Hour)}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), identity)
	p, e := a.Resolve(ctx)
	require.NoError(t, e)
	require.Equal(t, "alice", p.UserID)
	require.Empty(t, p.OrganizationID)
	require.False(t, p.PlatformMaintainer)
	identity.TenantID, identity.EffectiveOrganizationID, identity.EffectiveMemberID = "org-a", "org-a", "member"
	selected := authidentity.WithAuthenticatedIdentity(context.Background(), identity)
	selected, e = a.Bind(selected, "", false)
	require.NoError(t, e)
	p, e = a.Resolve(selected)
	require.NoError(t, e)
	require.Empty(t, p.OrganizationID)
	identity.TokenExpiresAt = time.Now().Add(-time.Second)
	_, e = a.Resolve(authidentity.WithAuthenticatedIdentity(ctx, identity))
	require.Error(t, e)
}
func TestOrganizationGrantsAreFreshForEveryOperation(t *testing.T) {
	policy, e := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, e)
	identity := authidentity.AuthenticatedIdentity{UserID: "alice", TenantID: "org-a", EffectiveOrganizationID: "org-a", EffectiveMemberID: "member", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)}
	live := &liveResolver{identity: identity}
	a, e := New(live, policy)
	require.NoError(t, e)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), identity)
	ctx, e = a.Bind(ctx, "Bearer controlled-token", true)
	require.NoError(t, e)
	p, e := a.Resolve(ctx)
	require.NoError(t, e)
	require.True(t, p.UseOrganization)
	require.Equal(t, "member", p.MemberID)
	live.changed = true
	p, e = a.Resolve(ctx)
	require.NoError(t, e)
	require.False(t, p.UseOrganization)
	require.Equal(t, 2, live.calls)
	_, e = a.Resolve(authidentity.WithAuthenticatedIdentity(context.Background(), identity))
	require.Error(t, e)
}
