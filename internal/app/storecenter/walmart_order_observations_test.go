package storecenterapp

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/storecenter"
	auth "task-processor/internal/storecenter/walmart"
	"testing"
	"time"
)

func TestSavedWalmartOrdersUseOnlyCurrentLocalOrderAuthority(t *testing.T) {
	now := time.Now()
	shop := uuid.NewString()
	g := &storecenter.WalmartGrant{OrganizationID: "org", StoreID: shop, ID: uuid.NewString(), AppID: "app", AppRevision: "r1", Environment: "sandbox", MerchantIdentityHash: strings.Repeat("a", 64), ScopesHash: strings.Repeat("b", 64), Generation: 1, Version: 1, Status: "active", AccessExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(time.Hour)}
	reader := &observationMaterialReader{material: storecenter.ProductExecutionMaterial{Platform: storecenter.PlatformWalmart, Walmart: g, ServiceExpiresAt: now.Add(time.Hour), StoreVersion: 1}}
	registry := &OfficialApplicationRegistry{walmartProvider: walmartBindingMetadataOnly{app: auth.Application{ClientID: "app", Revision: "r1", Environment: "sandbox"}}}
	access, e := NewOfficialObservationAccess(reader, observationLiveAccess{}, registry)
	require.NoError(t, e)
	access.now = func() time.Time { return now }
	subject := storecenter.ObservationSubject{OrganizationID: "org", ActorID: "actor", MemberID: "member", Purpose: storecenter.ObservationPurposeOrders}
	b, e := access.ReadSavedWalmartOrderBinding(context.Background(), subject, shop)
	require.NoError(t, e)
	require.Equal(t, g.MerchantIdentityHash, b.SupplierIdentityHash)
	require.Equal(t, subject, reader.subject)
	g.Status = "revoked"
	_, e = access.ReadSavedWalmartOrderBinding(context.Background(), subject, shop)
	require.Error(t, e)
	g.Status = "active"
	subject.Purpose = storecenter.ObservationPurposeProducts
	_, e = access.ReadSavedWalmartOrderBinding(context.Background(), subject, shop)
	require.Error(t, e)
}
