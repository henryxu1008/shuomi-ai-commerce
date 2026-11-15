package subjectverification

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMarketNameUsesOnlyVerifiedSameScopeCompany(t *testing.T) {
	a := Application{OrganizationID: "org", Scope: "provider-scope", CompanyName: "已认证企业", State: Verified, ProviderOrganizationID: "provider-org", ProviderAdminID: "provider-admin", ProviderVerifiedAt: time.Now().UTC()}
	store := &memoryStore{apps: map[string]Application{"org": a}}
	service := &Service{Store: store, Scope: "provider-scope"} // No Provider: this read cannot start paid verification.
	name, err := service.VerifiedCompanyName(context.Background(), "org")
	require.NoError(t, err)
	require.Equal(t, a.CompanyName, name)
	for _, state := range []string{Pending, Unknown} {
		a.State = state
		store.apps["org"] = a
		name, err = service.VerifiedCompanyName(context.Background(), "org")
		require.NoError(t, err)
		require.Empty(t, name)
	}
	a.State = Verified
	a.Scope = "another-scope"
	store.apps["org"] = a
	name, err = service.VerifiedCompanyName(context.Background(), "org")
	require.ErrorIs(t, err, ErrConflict)
	require.Empty(t, name)
	a.Scope = service.Scope
	a.ProviderVerifiedAt = time.Time{}
	store.apps["org"] = a
	name, err = service.VerifiedCompanyName(context.Background(), "org")
	require.ErrorIs(t, err, ErrUnavailable)
	require.Empty(t, name)
	name, err = service.VerifiedCompanyName(context.Background(), "missing")
	require.NoError(t, err)
	require.Empty(t, name)
}
