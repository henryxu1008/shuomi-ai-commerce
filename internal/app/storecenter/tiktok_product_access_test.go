package storecenterapp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	integration "task-processor/internal/integration/tiktokshop"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/storecenter"
	tiktok "task-processor/internal/storecenter/tiktokshop"
)

type tiktokProductProvider struct {
	tiktok.SellerAuthorization
	TikTokRulesProvider
	app         tiktok.Application
	shopID      string
	calls       int
	statusCalls int
	afterStatus func()
}

func (p *tiktokProductProvider) QueryProductStatus(_ context.Context, token, cipher string, expected marketplace.ProductStatusExpectation) (marketplace.ProductStatus, error) {
	p.statusCalls++
	if token != "private-access" || cipher != "fresh-cipher" {
		return marketplace.ProductStatus{}, ErrProductAccessChanged
	}
	if p.afterStatus != nil {
		p.afterStatus()
	}
	return marketplace.ProductStatus{ProductID: expected.ProductID, Status: "ACTIVATE", AuditStatus: "APPROVED", Reasons: []string{}}, nil
}

func (p *tiktokProductProvider) Application() tiktok.Application { return p.app }
func (p *tiktokProductProvider) AuthorizedShops(context.Context, string) ([]tiktok.Shop, error) {
	return []tiktok.Shop{{ID: p.shopID, Code: "USCODE", Cipher: "fresh-cipher", Region: "US", SellerType: "CROSS_BORDER"}}, nil
}
func (p *tiktokProductProvider) QueryListingRules(_ context.Context, token, cipher string) (marketplace.ListingRules, error) {
	p.calls++
	if token != "private-access" || cipher != "fresh-cipher" {
		return marketplace.ListingRules{}, ErrProductAccessChanged
	}
	return marketplace.ListingRules{Methods: []marketplace.ListingMethod{marketplace.GlobalPublishing}, Fingerprint: strings.Repeat("a", 64)}, nil
}

func TestTikTokProductAccessRequiresFreshShopScopesAndCanonicalGeneration(t *testing.T) {
	now := time.Now().UTC()
	protection, err := integration.NewCredentialProtection("key", make([]byte, 32))
	require.NoError(t, err)
	grant := tiktok.GrantSnapshot{OrganizationID: "org-a", ID: uuid.NewString(), AppKey: "app", AppRevision: "r1", SellerIdentityHash: sellerIdentityHash("seller"), Generation: 1, Revision: 1, Status: tiktok.GrantActive, AccessExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now}
	tokens := tiktok.TokenSet{AccessToken: "private-access", RefreshToken: "private-refresh", OpenID: "seller", UserType: 4, GrantedScopes: []string{"seller.authorization.info", "seller.product.basic"}, AccessExpiresAt: grant.AccessExpiresAt, RefreshExpiresAt: grant.RefreshExpiresAt}
	grant.KeyID, grant.Ciphertext, err = protection.Seal(grantCredentialBinding(grant), tokens)
	require.NoError(t, err)
	storeID := uuid.NewString()
	reader := &productMaterialReader{material: storecenter.ProductExecutionMaterial{Platform: storecenter.PlatformTikTokShop, StoreVersion: 2, ServiceExpiresAt: now.Add(time.Hour), Connection: storecenter.OfficialConnectionView{Status: storecenter.ConnectionStatusConnected, Version: 1, AttemptID: uuid.NewString()}, TikTok: &storecenter.TikTokProductExecutionMaterial{Grant: grant, Binding: storecenter.TikTokShopBinding{OrganizationID: "org-a", StoreID: storeID, GrantID: grant.ID, AppKey: "app", Shop: tiktok.Shop{ID: "shop", Code: "USCODE", Cipher: "old-cipher", Region: "US", SellerType: "CROSS_BORDER"}, Status: "active", Version: 1}}}}
	provider := &tiktokProductProvider{app: tiktok.Application{AppKey: "app", ServiceID: "service", Revision: "r1", CallbackURL: "https://example.org/callback"}, shopID: "shop"}
	registry, err := WithTikTokApplication(nil, provider, protection)
	require.NoError(t, err)
	access, err := NewOfficialProductAccess(reader, productLiveAccess{}, registry)
	require.NoError(t, err)
	subject := storecenter.ProductExecutionSubject{OrganizationID: "org-a", ActorID: "actor", MemberID: "member", Purpose: storecenter.ProductPurposeRules}
	handle, err := access.AuthorizeTikTokRules(context.Background(), subject, storeID, nil)
	require.NoError(t, err)
	require.True(t, handle.Binding().ValidTikTokApplication())
	_, err = handle.ListingRules(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, provider.calls)
	_, err = handle.Categories(context.Background(), marketplace.GlobalPublishing)
	require.Error(t, err, "ungranted global-category scope must block resource I/O")
	require.Equal(t, 1, provider.calls)
	_, err = handle.ProductStatus(context.Background(), marketplace.ProductStatusExpectation{ProductID: "local", LocalSKUIDs: []string{"sku"}})
	require.NoError(t, err)
	require.Equal(t, 1, provider.statusCalls)
	provider.afterStatus = func() { reader.material.TikTok.Grant.Revision++ }
	_, err = handle.ProductStatus(context.Background(), marketplace.ProductStatusExpectation{ProductID: "local", LocalSKUIDs: []string{"sku"}})
	require.ErrorIs(t, err, ErrProductAccessChanged, "canonical authorization change during response blocks observation")
	provider.afterStatus = nil
	reader.material.TikTok.Grant.Revision++
	reader.material.TikTok.Grant.Status = tiktok.GrantRevoked
	_, err = handle.ListingRules(context.Background())
	require.Error(t, err)
	require.Equal(t, 1, provider.calls)
	_, err = handle.ProductStatus(context.Background(), marketplace.ProductStatusExpectation{ProductID: "local", LocalSKUIDs: []string{"sku"}})
	require.Error(t, err)
	require.Equal(t, 2, provider.statusCalls, "revoked credentials cannot issue status I/O")
	reader.material.TikTok.Grant = grant
	provider.shopID = "another-shop"
	_, err = access.AuthorizeTikTokRules(context.Background(), subject, storeID, nil)
	require.Error(t, err)
	provider.shopID = "shop"
	subject.Purpose = storecenter.ProductPurposePublish
	_, err = access.AuthorizeTikTokRules(context.Background(), subject, storeID, nil)
	require.Error(t, err)
}
