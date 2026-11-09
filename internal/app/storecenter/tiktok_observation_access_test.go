package storecenterapp

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	integration "task-processor/internal/integration/tiktokshop"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	auth "task-processor/internal/storecenter/tiktokshop"
	"testing"
	"time"
)

type tikTokOrderProvider struct {
	tiktokProductProvider
	orderCalls  int
	afterOrders func()
}

func (p *tikTokOrderProvider) Orders(_ context.Context, token, cipher string, _ o.TikTokOrderQuery) (o.TikTokOrderPage, error) {
	p.orderCalls++
	if token != "private-access" || cipher != "fresh-cipher" {
		return o.TikTokOrderPage{}, o.ErrForbidden
	}
	if p.afterOrders != nil {
		p.afterOrders()
	}
	return o.TikTokOrderPage{}, nil
}

func TestTikTokOrdersRequireOrderScopeFreshShopAndRetainBinding(t *testing.T) {
	for _, scope := range []string{"seller.order.info", "seller.fs.order&fufillment.management", "seller.product.basic"} {
		t.Run(scope, func(t *testing.T) {
			now := time.Now()
			store := uuid.NewString()
			protection, e := integration.NewCredentialProtection("key", make([]byte, 32))
			require.NoError(t, e)
			grant := auth.GrantSnapshot{OrganizationID: "org", ID: uuid.NewString(), AppKey: "app", AppRevision: "r1", SellerIdentityHash: sellerIdentityHash("seller"), Generation: 1, Revision: 1, Status: auth.GrantActive, AccessExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now}
			token := auth.TokenSet{AccessToken: "private-access", RefreshToken: "private-refresh", OpenID: "seller", UserType: 0, GrantedScopes: []string{"seller.authorization.info", scope}, AccessExpiresAt: grant.AccessExpiresAt, RefreshExpiresAt: grant.RefreshExpiresAt}
			seal := func() {
				grant.KeyID, grant.Ciphertext, e = protection.Seal(grantCredentialBinding(grant), token)
				require.NoError(t, e)
			}
			seal()
			reader := &observationMaterialReader{material: storecenter.ProductExecutionMaterial{Platform: storecenter.PlatformTikTokShop, StoreVersion: 1, ServiceExpiresAt: now.Add(time.Hour), Connection: storecenter.OfficialConnectionView{AttemptID: uuid.NewString(), Version: 1}, TikTok: &storecenter.TikTokProductExecutionMaterial{Grant: grant, Binding: storecenter.TikTokShopBinding{OrganizationID: "org", StoreID: store, GrantID: grant.ID, AppKey: "app", Shop: auth.Shop{ID: "shop", Code: "USCODE", Cipher: "old-cipher", Region: "US", SellerType: "CROSS_BORDER"}, Status: "active", Version: 1}}}}
			provider := &tikTokOrderProvider{tiktokProductProvider: tiktokProductProvider{app: auth.Application{AppKey: "app", Revision: "r1"}, shopID: "shop"}}
			registry, e := WithTikTokApplication(nil, provider, protection)
			require.NoError(t, e)
			access, e := NewOfficialObservationAccess(reader, observationLiveAccess{}, registry)
			require.NoError(t, e)
			subject := storecenter.ObservationSubject{OrganizationID: "org", ActorID: "actor", MemberID: "member", Purpose: storecenter.ObservationPurposeOrders, Sync: true}
			h, e := access.Authorize(context.Background(), subject, store, nil)
			if scope == "seller.product.basic" {
				require.Error(t, e)
				require.Zero(t, provider.shopCalls)
				return
			}
			require.NoError(t, e)
			require.True(t, h.Binding().ValidTikTokApplication())
			raw, e := json.Marshal(h.Binding())
			require.NoError(t, e)
			var saved o.Binding
			require.NoError(t, json.Unmarshal(raw, &saved))
			_, e = access.ReadTikTokOrders(context.Background(), o.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, saved, o.TikTokOrderQuery{})
			require.NoError(t, e)
			require.Equal(t, 1, provider.orderCalls)
			provider.afterOrders = func() { reader.material.TikTok.Grant.Status = auth.GrantRevoked }
			_, e = access.ReadTikTokOrders(context.Background(), o.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, saved, o.TikTokOrderQuery{})
			require.Error(t, e)
			require.Equal(t, 2, provider.orderCalls)
			reader.material.TikTok.Grant.Status = auth.GrantActive
			provider.afterOrders = nil
			provider.shopID = "foreign-shop"
			_, e = access.Authorize(context.Background(), subject, store, nil)
			require.Error(t, e)
			require.Equal(t, 2, provider.orderCalls)
			provider.shopID = "shop"
			token.UserType = 4
			seal()
			reader.material.TikTok.Grant = grant
			_, e = access.Authorize(context.Background(), subject, store, nil)
			require.Error(t, e)
			require.Equal(t, 2, provider.orderCalls)
		})
	}
}
