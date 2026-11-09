package storecenterapp

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	auth "task-processor/internal/storecenter/walmart"
	"testing"
	"time"
)

type orderObservationProvider struct {
	auth.Authorization
	app    auth.Application
	scopes map[string]string
	after  func()
	calls  int
}

func (p *orderObservationProvider) Application() auth.Application { return p.app }
func (p *orderObservationProvider) TokenDetail(context.Context, auth.TokenSet) (auth.TokenDetail, error) {
	return auth.TokenDetail{Valid: true, ExpiresAt: time.Now().Add(time.Hour), Scopes: p.scopes}, nil
}
func (p *orderObservationProvider) Orders(context.Context, auth.TokenSet, o.WalmartOrderQuery) (o.WalmartOrderPage, error) {
	p.calls++
	if p.after != nil {
		p.after()
	}
	return o.WalmartOrderPage{Total: 0}, nil
}

type orderObservationProtection struct{ auth.CredentialProtection }

func (orderObservationProtection) Open(auth.CredentialBinding, string, string) (auth.TokenSet, error) {
	return auth.TokenSet{SellerID: "seller", AccessToken: "synthetic"}, nil
}
func TestWalmartOrdersKeepPurposeBindingAndLateRevocation(t *testing.T) {
	now := time.Now()
	store := uuid.NewString()
	provider := &orderObservationProvider{app: auth.Application{ClientID: "app", Revision: "r1", Environment: "sandbox"}, scopes: map[string]string{"orders": "view_only"}}
	g := &storecenter.WalmartGrant{OrganizationID: "org", StoreID: store, ID: uuid.NewString(), AppID: "app", AppRevision: "r1", Environment: "sandbox", SellerID: "seller", MerchantIdentityHash: storecenter.WalmartMerchantIdentity("sandbox", "seller"), ScopesHash: marketplace.Hash(provider.scopes), Generation: 1, Version: 1, Status: "active", AccessExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(time.Hour)}
	reader := &observationMaterialReader{material: storecenter.ProductExecutionMaterial{Platform: storecenter.PlatformWalmart, Walmart: g, StoreVersion: 1, ServiceExpiresAt: now.Add(time.Hour), Connection: storecenter.OfficialConnectionView{AttemptID: uuid.NewString(), Version: 1}}}
	registry, e := WithWalmartApplication(nil, provider, orderObservationProtection{})
	require.NoError(t, e)
	a, e := NewOfficialObservationAccess(reader, observationLiveAccess{}, registry)
	require.NoError(t, e)
	scope := o.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	subject := storecenter.ObservationSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: storecenter.ObservationPurposeOrders, Sync: true}
	h, e := a.Authorize(context.Background(), subject, store, nil)
	require.NoError(t, e)
	// Durable JSON round-trip strips monotonic clocks and normalizes the zone.
	raw, e := json.Marshal(h.Binding())
	require.NoError(t, e)
	var persisted o.Binding
	require.NoError(t, json.Unmarshal(raw, &persisted))
	_, e = a.ReadWalmartOrders(context.Background(), scope, persisted, o.WalmartOrderQuery{})
	require.NoError(t, e)
	require.Equal(t, 1, provider.calls)
	provider.after = func() { g.Status = "revoked" }
	_, e = a.ReadWalmartOrders(context.Background(), scope, persisted, o.WalmartOrderQuery{})
	require.Error(t, e)
	require.Equal(t, 2, provider.calls)
	g.Status = "active"
	provider.after = nil
	provider.scopes = map[string]string{"item": "full_access"}
	g.ScopesHash = marketplace.Hash(provider.scopes)
	_, e = a.Authorize(context.Background(), subject, store, nil)
	require.Error(t, e)
	require.Equal(t, 2, provider.calls)
}
