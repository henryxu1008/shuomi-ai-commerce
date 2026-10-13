package storecenterapp

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"sync"
	"sync/atomic"
	integration "task-processor/internal/integration/tiktokshop"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	tiktok "task-processor/internal/storecenter/tiktokshop"
	"testing"
	"time"
)

type tiktokMutationProvider struct {
	tiktokProductProvider
	TikTokMutationProvider
	uploads atomic.Int64
}

func (p *tiktokMutationProvider) UploadProductImage(_ context.Context, token, useCase, media string, _ []byte) (marketplace.UploadedImage, error) {
	if token != "private-access" || useCase != "MAIN_IMAGE" || media != "image/png" {
		return marketplace.UploadedImage{}, ErrProductAccessChanged
	}
	p.uploads.Add(1)
	return marketplace.UploadedImage{URI: "actual-uri", URL: "https://images.example.org/image.png", Width: 1024, Height: 1024, UseCase: useCase, ResponseHash: collection.Digest("actual response")}, nil
}

func TestTikTokMutationHandleSendsOnceAndRejectsRevokedGrant(t *testing.T) {
	now := time.Now().UTC()
	protection, err := integration.NewCredentialProtection("key", make([]byte, 32))
	require.NoError(t, err)
	grant := tiktok.GrantSnapshot{OrganizationID: "org-a", ID: uuid.NewString(), AppKey: "app", AppRevision: "r1", SellerIdentityHash: sellerIdentityHash("seller"), Generation: 1, Revision: 1, Status: tiktok.GrantActive, AccessExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now}
	tokens := tiktok.TokenSet{AccessToken: "private-access", RefreshToken: "private-refresh", OpenID: "seller", UserType: 4, GrantedScopes: []string{"seller.authorization.info", "seller.product.basic"}, AccessExpiresAt: grant.AccessExpiresAt, RefreshExpiresAt: grant.RefreshExpiresAt}
	grant.KeyID, grant.Ciphertext, err = protection.Seal(grantCredentialBinding(grant), tokens)
	require.NoError(t, err)
	storeID := uuid.NewString()
	reader := &productMaterialReader{material: storecenter.ProductExecutionMaterial{Platform: storecenter.PlatformTikTokShop, StoreVersion: 2, ServiceExpiresAt: now.Add(time.Hour), Connection: storecenter.OfficialConnectionView{Status: storecenter.ConnectionStatusConnected, Version: 1, AttemptID: uuid.NewString()}, TikTok: &storecenter.TikTokProductExecutionMaterial{Grant: grant, Binding: storecenter.TikTokShopBinding{OrganizationID: "org-a", StoreID: storeID, GrantID: grant.ID, AppKey: "app", Shop: tiktok.Shop{ID: "shop", Code: "USCODE", Cipher: "old-cipher", Region: "US", SellerType: "CROSS_BORDER"}, Status: "active", Version: 1}}}}
	provider := &tiktokMutationProvider{tiktokProductProvider: tiktokProductProvider{app: tiktok.Application{AppKey: "app", ServiceID: "service", Revision: "r1", CallbackURL: "https://example.org/callback"}, shopID: "shop"}}
	registry, err := WithTikTokApplication(nil, provider, protection)
	require.NoError(t, err)
	access, err := NewOfficialProductAccess(reader, productLiveAccess{}, registry)
	require.NoError(t, err)
	subject := storecenter.ProductExecutionSubject{OrganizationID: "org-a", ActorID: "actor", MemberID: "member", Purpose: storecenter.ProductPurposeImage}
	handle, err := access.AuthorizeTikTokMutation(context.Background(), subject, storeID, nil)
	require.NoError(t, err)
	var sent sync.WaitGroup
	for range 2 {
		sent.Add(1)
		go func() {
			defer sent.Done()
			_, _ = handle.UploadImage(context.Background(), "MAIN_IMAGE", "image/png", []byte("approved image"))
		}()
	}
	sent.Wait()
	require.EqualValues(t, 1, provider.uploads.Load(), "concurrent callers share one consume permission")
	_, err = handle.CreateGlobal(context.Background(), marketplace.CreateGlobalRequest{})
	require.Error(t, err, "image purpose cannot publish")
	handle, err = access.AuthorizeTikTokMutation(context.Background(), subject, storeID, nil)
	require.NoError(t, err)
	reader.material.TikTok.Grant.Revision++
	reader.material.TikTok.Grant.Status = tiktok.GrantRevoked
	_, err = handle.UploadImage(context.Background(), "MAIN_IMAGE", "image/png", []byte("approved image"))
	require.Error(t, err)
	require.EqualValues(t, 1, provider.uploads.Load())
}
