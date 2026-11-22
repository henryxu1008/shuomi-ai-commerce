package collection

import (
	"context"
	"errors"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type liveAccess func(context.Context, string, string) ([]string, error)

func (f liveAccess) ResolveLiveRoles(ctx context.Context, org, actor string) ([]string, error) {
	return f(ctx, org, actor)
}
func TestContextAuthorizationRechecksTokenAfterLiveResolution(t *testing.T) {
	now := time.Now()
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	authorizer, err := NewContextAuthorizer(liveAccess(func(context.Context, string, string) ([]string, error) {
		now = now.Add(time.Minute)
		return []string{"listingkit_admin"}, nil
	}), permissions)
	require.NoError(t, err)
	authorizer.now = func() time.Time { return now }
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", EffectiveMemberID: "member", UserID: "actor", TokenExpiresAt: now.Add(10 * time.Second)})
	_, err = authorizer.Authorize(ctx, PermissionManage)
	require.ErrorIs(t, err, ErrForbidden)
	authorizer.live = liveAccess(func(context.Context, string, string) ([]string, error) { return nil, errors.New("revoked") })
	_, err = authorizer.Authorize(ctx, PermissionRead)
	require.ErrorIs(t, err, ErrForbidden)
}
