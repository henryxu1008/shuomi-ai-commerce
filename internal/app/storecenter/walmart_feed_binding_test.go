package storecenterapp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"task-processor/internal/storecenter"
	walmart "task-processor/internal/storecenter/walmart"
)

type walmartBindingMetadataOnly struct {
	walmart.Authorization
	app walmart.Application
}

func (p walmartBindingMetadataOnly) Application() walmart.Application { return p.app }

func TestSavedWalmartFeedBindingUsesLocalAuthorityOnly(t *testing.T) {
	now := time.Now()
	org, shop := "org", "550e8400-e29b-41d4-a716-446655440000"
	grant := &storecenter.WalmartGrant{
		OrganizationID: org, StoreID: shop, ID: "550e8400-e29b-41d4-a716-446655440001",
		AppID: "app", AppRevision: "r1", Environment: "production",
		MerchantIdentityHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ScopesHash:           "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Generation:           1, Version: 1, Status: "active", AccessExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(time.Hour),
	}
	reader := &productMaterialReader{material: storecenter.ProductExecutionMaterial{
		StoreVersion: 2, Platform: storecenter.PlatformWalmart, Walmart: grant, ServiceExpiresAt: now.Add(time.Hour),
	}}
	// The fixture supplies neither decryption nor TokenDetail. An attempted call fails.
	registry := &OfficialApplicationRegistry{walmartProvider: walmartBindingMetadataOnly{
		app: walmart.Application{ClientID: "app", Revision: "r1", Environment: "production"},
	}}
	access, err := NewOfficialProductAccess(reader, productLiveAccess{}, registry)
	require.NoError(t, err)
	access.now = func() time.Time { return now }
	subject := storecenter.ProductExecutionSubject{OrganizationID: org, ActorID: "actor", MemberID: "member", Purpose: storecenter.ProductPurposeRules}
	binding, err := access.ReadSavedWalmartFeedBinding(context.Background(), subject, shop)
	require.NoError(t, err)
	require.Equal(t, grant.MerchantIdentityHash, binding.SupplierIdentityHash)
	require.Equal(t, 1, reader.reads)

	grant.Status = "revoked"
	_, err = access.ReadSavedWalmartFeedBinding(context.Background(), subject, shop)
	require.ErrorIs(t, err, ErrProductAccessChanged)
}
