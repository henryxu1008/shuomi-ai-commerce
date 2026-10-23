package productsourcing

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authz"
	"testing"
)

type amazonReadOnlyPolicy struct{}

func (amazonReadOnlyPolicy) RoleModules(_ context.Context, org string, roles []string) (map[string][]string, error) {
	return map[string][]string{authz.EnterpriseRoleKey(org, 1): {"supply-mine"}}, nil
}
func TestAmazonCaptureReadDoesNotRequireCaptureWrite(t *testing.T) {
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	permissions.SetRolePolicyReader(amazonReadOnlyPolicy{})
	live := liveRolesFunc(func(_ context.Context, org, _ string) ([]string, error) {
		return []string{authz.EnterpriseRoleKey(org, 1)}, nil
	})
	auth, err := NewAmazonCaptureContextAuthorizer(live, live, permissions)
	require.NoError(t, err)
	scope, err := auth.Authorize(acquisitionIdentity("amazon-org", "actor"), false)
	require.NoError(t, err)
	require.Equal(t, "membership:actor", scope.MemberID)
	_, err = auth.Authorize(acquisitionIdentity("amazon-org", "actor"), true)
	require.Error(t, err)
}
