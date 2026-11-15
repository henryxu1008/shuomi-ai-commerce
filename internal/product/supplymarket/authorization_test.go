package supplymarket

import (
	"context"
	"errors"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type liveRoles struct{}

type failingLiveRoles struct{ err error }

func (f failingLiveRoles) ResolveLiveRoles(context.Context, string, string) ([]string, error) {
	return nil, f.err
}

func TestMarketStatusAuthorizationDistinguishesLiveDependencyFailure(t *testing.T) {
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Minute)})
	for _, test := range []struct{ cause, expected error }{{sourcing.ErrPublicationForbidden, ErrForbidden}, {sourcing.ErrSourcePublicationUnavailable, ErrUnavailable}, {errors.New("role store failed"), ErrUnavailable}} {
		a, e := NewContextAuthorizer(failingLiveRoles{test.cause}, permissions)
		require.NoError(t, e)
		_, e = a.Authorize(ctx, collection.PermissionRead)
		require.ErrorIs(t, e, test.expected)
	}
}

func (liveRoles) ResolveLiveRoles(context.Context, string, string) ([]string, error) {
	return []string{"owner"}, nil
}
func TestPlatformAuthorityRequiresVerifiedGlobalContextAndUnexpiredIdentity(t *testing.T) {
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	a, err := NewContextAuthorizer(liveRoles{}, permissions)
	require.NoError(t, err)
	identity := authidentity.AuthenticatedIdentity{UserID: "platform-a", Roles: []string{"platform_admin"}, TokenExpiresAt: time.Now().Add(time.Minute)}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), identity)
	actor, err := a.AuthorizePlatform(ctx)
	require.NoError(t, err)
	require.Equal(t, identity.UserID, actor)
	identity.TenantID = "org-a"
	identity.EffectiveOrganizationID = "org-a"
	identity.EffectiveMemberID = "member-a"
	_, err = a.AuthorizePlatform(authidentity.WithAuthenticatedIdentity(ctx, identity))
	require.ErrorIs(t, err, ErrForbidden)
	identity = authidentity.AuthenticatedIdentity{UserID: "member-a", Roles: []string{"admin", "owner"}, TokenExpiresAt: time.Now().Add(time.Minute)}
	_, err = a.AuthorizePlatform(authidentity.WithAuthenticatedIdentity(ctx, identity))
	require.ErrorIs(t, err, ErrForbidden)
	identity.Roles = []string{"platform_admin"}
	identity.TokenExpiresAt = time.Now().Add(-time.Second)
	_, err = a.AuthorizePlatform(authidentity.WithAuthenticatedIdentity(ctx, identity))
	require.ErrorIs(t, err, ErrForbidden)
	_, err = a.Authorize(nil, collection.PermissionManage)
	require.ErrorIs(t, err, ErrForbidden)
}
