package storecenterapp

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/storecenter"
	auth "task-processor/internal/storecenter/tiktokshop"
	"testing"
	"time"
)

type tiktokOrderMetadataOnly struct{ auth.SellerAuthorization }

func (tiktokOrderMetadataOnly) Application() auth.Application {
	return auth.Application{AppKey: "app", Revision: "r1"}
}
func TestSavedTikTokOrdersUseOnlyCurrentLocalOrderAuthority(t *testing.T) {
	now := time.Now()
	id := uuid.NewString()
	grant := auth.GrantSnapshot{OrganizationID: "org", ID: uuid.NewString(), AppKey: "app", AppRevision: "r1", SellerIdentityHash: strings.Repeat("a", 64), Generation: 1, Revision: 1, Status: auth.GrantActive, KeyID: "key", Ciphertext: "not-decrypted", AccessExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now}
	reader := &observationMaterialReader{material: storecenter.ProductExecutionMaterial{Platform: storecenter.PlatformTikTokShop, StoreVersion: 1, ServiceExpiresAt: now.Add(time.Hour), Connection: storecenter.OfficialConnectionView{Version: 1, Status: storecenter.ConnectionStatusConnected}, TikTok: &storecenter.TikTokProductExecutionMaterial{Grant: grant, Binding: storecenter.TikTokShopBinding{OrganizationID: "org", StoreID: id, GrantID: grant.ID, AppKey: "app", Shop: auth.Shop{ID: "shop", Code: "code", Cipher: "private-cipher", Region: "US", SellerType: "CROSS_BORDER"}, Status: "active", Version: 1}}}}
	access, e := NewOfficialObservationAccess(reader, observationLiveAccess{}, &OfficialApplicationRegistry{tiktokProvider: tiktokOrderMetadataOnly{}})
	require.NoError(t, e)
	access.now = func() time.Time { return now }
	subject := storecenter.ObservationSubject{OrganizationID: "org", ActorID: "actor", MemberID: "member", Purpose: storecenter.ObservationPurposeOrders}
	b, e := access.ReadSavedTikTokOrderBinding(context.Background(), subject, id)
	require.NoError(t, e)
	require.Empty(t, b.ScopesHash)
	require.Equal(t, "shop", b.TikTok.ShopID)
	require.Equal(t, subject, reader.subject)
	for _, mutate := range []func(){func() { reader.material.TikTok.Grant.Status = auth.GrantRevoked }, func() { reader.material.TikTok.Binding.Shop.Region = "GB" }, func() { reader.material.TikTok.Binding.Shop.SellerType = "LOCAL" }, func() { reader.material.TikTok.Binding.GrantID = uuid.NewString() }, func() { reader.material.TikTok.Grant.AppRevision = "r2" }} {
		saved := *reader.material.TikTok
		mutate()
		_, e = access.ReadSavedTikTokOrderBinding(context.Background(), subject, id)
		require.Error(t, e)
		*reader.material.TikTok = saved
	}
	subject.Sync = true
	_, e = access.ReadSavedTikTokOrderBinding(context.Background(), subject, id)
	require.Error(t, e)
	subject.Sync = false
	subject.Purpose = storecenter.ProductPurposeRules
	_, e = access.ReadSavedTikTokOrderBinding(context.Background(), subject, id)
	require.Error(t, e)
}
