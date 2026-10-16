package storecenterapp

import (
	"context"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	auth "task-processor/internal/storecenter/walmart"
	"time"
)

type WalmartProductsReader interface {
	Products(context.Context, auth.TokenSet, int) (marketplace.ProductPage, error)
}

func (a *OfficialObservationAccess) authorizeWalmartObservation(ctx context.Context, started time.Time, s storecenter.ObservationSubject, id string, expected *MerchantBinding, m storecenter.ProductExecutionMaterial) (*ObservationHandle, error) {
	if s.Purpose != storecenter.ObservationPurposeProducts {
		return nil, o.ErrUnsupported
	}
	if !a.applications.WalmartConfigured() || m.Walmart == nil {
		return nil, o.ErrUnavailable
	}
	g := *m.Walmart
	app := a.applications.walmartProvider.Application()
	if g.OrganizationID != s.OrganizationID || g.StoreID != id || g.AppID != app.ClientID || g.AppRevision != app.Revision || g.Environment != app.Environment || !g.CanRead(a.now()) || !a.now().Before(m.ServiceExpiresAt) {
		return nil, o.ErrConflict
	}
	if _, ok := a.applications.walmartProvider.(WalmartProductsReader); !ok {
		return nil, o.ErrUnavailable
	}
	binding := MerchantBinding{Platform: storecenter.PlatformWalmart, Site: "walmart-us", OrganizationID: s.OrganizationID, StoreID: id, StoreVersion: m.StoreVersion, ConnectionRevision: m.Connection.Version, ApplicationRevision: g.AppRevision, ApplicationID: g.AppID, SupplierIdentityHash: g.MerchantIdentityHash, ServiceExpiresAt: m.ServiceExpiresAt, CredentialExpiresAt: g.AccessExpiresAt, ScopesHash: g.ScopesHash, Walmart: storecenter.WalmartMerchantBinding{Environment: g.Environment, Market: "US", GrantID: g.ID, Generation: g.Generation, GrantVersion: g.Version}}
	if !binding.ValidWalmartApplication() || expected != nil && *expected != binding {
		return nil, o.ErrConflict
	}
	h := &ObservationHandle{owner: a, subject: s, binding: binding, connectionRef: m.Connection.AttemptID, expiresAt: started.Add(10 * time.Second), walmartGrant: &g}
	if _, e := h.walmartCredential(ctx); e != nil {
		return nil, e
	}
	return h, nil
}
func (h *ObservationHandle) walmartCredential(ctx context.Context) (auth.TokenSet, error) {
	var zero auth.TokenSet
	if h == nil || h.owner == nil || h.walmartGrant == nil || ctx == nil || ctx.Err() != nil || !h.owner.now().Before(h.expiresAt) || h.subject.Purpose != storecenter.ObservationPurposeProducts {
		return zero, o.ErrForbidden
	}
	ctx, cancel := context.WithDeadline(ctx, h.expiresAt)
	defer cancel()
	m, e := h.owner.reader.ReadObservation(ctx, h.subject, h.binding.StoreID, h.owner.authorization, h.owner.now())
	if e != nil || m.Walmart == nil || *m.Walmart != *h.walmartGrant || m.StoreVersion != h.binding.StoreVersion || m.Connection.AttemptID != h.connectionRef || m.Connection.Version != h.binding.ConnectionRevision || !m.ServiceExpiresAt.Equal(h.binding.ServiceExpiresAt) {
		return zero, o.ErrConflict
	}
	app := h.owner.applications.walmartProvider.Application()
	g := h.walmartGrant
	if app.ClientID != g.AppID || app.Revision != g.AppRevision || app.Environment != g.Environment {
		return zero, o.ErrConflict
	}
	tokens, e := h.owner.applications.walmartProtection.Open(g.CredentialBinding(), g.KeyID, g.Ciphertext)
	if e != nil || tokens.SellerID != g.SellerID {
		return zero, o.ErrConflict
	}
	detail, e := h.owner.applications.walmartProvider.TokenDetail(ctx, tokens)
	if e != nil || !detail.Allows(storecenter.ObservationPurposeProducts, h.owner.now()) || marketplace.Hash(detail.Scopes) != g.ScopesHash {
		return zero, o.ErrConflict
	}
	return tokens, nil
}
func (h *ObservationHandle) walmartProducts(ctx context.Context, page int) (o.ProductPage, error) {
	tokens, e := h.walmartCredential(ctx)
	if e != nil {
		return o.ProductPage{}, e
	}
	result, e := h.owner.applications.walmartProvider.(WalmartProductsReader).Products(ctx, tokens, page)
	if e != nil {
		return o.ProductPage{}, o.ErrUnavailable
	}
	if _, e = h.walmartCredential(ctx); e != nil {
		return o.ProductPage{}, e
	}
	return o.ProductPage{Total: result.Total, WalmartItems: result.Items}, nil
}
