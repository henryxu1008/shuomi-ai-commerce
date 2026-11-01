package authz

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFulfillmentDoesNotInheritObservationWritePermissions(t *testing.T) {
	a, err := NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	for _, role := range []string{"listingkit_viewer", "listingkit_operator", "listingkit_admin"} {
		for _, permission := range FulfillmentPermissions() {
			require.Equal(t, role == "listingkit_admin" || permission == "workbench.fulfillment.read", a.Authorize("actor", []string{role}, permission), role+permission)
		}
	}
	key := EnterpriseRoleKey("org", 1)
	policy := rolePolicyFixture{"org": {key: {"store-products"}}}
	a.SetRolePolicyReader(policy)
	allowed, err := a.AuthorizeScoped(context.Background(), "actor", "org", []string{key}, "workbench.fulfillment.execute")
	require.NoError(t, err)
	require.False(t, allowed)
	policy["org"][key] = []string{"store-orders"}
	for _, permission := range FulfillmentPermissions() {
		allowed, err = a.AuthorizeScoped(context.Background(), "actor", "org", []string{key}, permission)
		require.NoError(t, err)
		require.Equal(t, permission == "workbench.fulfillment.read", allowed, "order observation module must not grant a new sensitive purpose: "+permission)
	}
}
