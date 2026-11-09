package storecenterapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	auth "task-processor/internal/storecenter/tiktokshop"
	"time"
)

type TikTokOrdersReader interface {
	Orders(context.Context, string, string, o.TikTokOrderQuery) (o.TikTokOrderPage, error)
}
type tikTokOrderAuthority struct {
	material    storecenter.TikTokProductExecutionMaterial
	application auth.Application
	shop        auth.Shop
}

func tikTokOrderScopes(t auth.TokenSet) bool {
	return t.UserType == 0 && hasTikTokScopes(t, "seller.authorization.info") && (hasTikTokScopes(t, "seller.order.info") || hasTikTokScopes(t, "seller.fs.order&fufillment.management"))
}

func (a *OfficialObservationAccess) authorizeTikTokOrders(ctx context.Context, started time.Time, s storecenter.ObservationSubject, id string, expected *MerchantBinding, m storecenter.ProductExecutionMaterial) (*ObservationHandle, error) {
	if s.Purpose != storecenter.ObservationPurposeOrders {
		return nil, o.ErrUnsupported
	}
	if !a.applications.TikTokConfigured() || a.applications.tiktokProtection == nil || m.TikTok == nil {
		return nil, o.ErrUnavailable
	}
	if _, ok := a.applications.tiktokProvider.(TikTokOrdersReader); !ok {
		return nil, o.ErrUnavailable
	}
	app := a.applications.tiktokProvider.Application()
	private := *m.TikTok
	g := private.Grant
	grant, e := auth.RestoreGrant(g)
	if e != nil || !grant.CanSend(a.now()) || g.OrganizationID != s.OrganizationID || g.AppKey != app.AppKey || g.AppRevision != app.Revision || private.Binding.OrganizationID != s.OrganizationID || private.Binding.StoreID != id || private.Binding.GrantID != g.ID || private.Binding.AppKey != app.AppKey || private.Binding.Status != "active" || private.Binding.Version < 1 || m.StoreVersion < 1 || m.Connection.Version < 1 || !a.now().Before(m.ServiceExpiresAt) {
		return nil, o.ErrConflict
	}
	local, e := auth.SelectUSCrossBorderShop([]auth.Shop{private.Binding.Shop}, private.Binding.Shop.Code)
	if e != nil {
		return nil, o.ErrConflict
	}
	tokens, e := a.applications.tiktokProtection.Open(grantCredentialBinding(g), g.KeyID, g.Ciphertext)
	if e != nil || !tikTokOrderScopes(tokens) || sellerIdentityHash(tokens.OpenID) != g.SellerIdentityHash || !tokens.AccessExpiresAt.Equal(g.AccessExpiresAt) || !a.now().Before(tokens.AccessExpiresAt) {
		return nil, o.ErrForbidden
	}
	ctx, cancel := context.WithDeadline(ctx, started.Add(10*time.Second))
	defer cancel()
	shops, e := a.applications.tiktokProvider.AuthorizedShops(ctx, tokens.AccessToken)
	if e != nil {
		return nil, o.ErrUnavailable
	}
	shop, e := auth.SelectUSCrossBorderShop(shops, local.Code)
	if e != nil || shop.ID != local.ID {
		return nil, o.ErrConflict
	}
	binding := tikTokOrderBinding(s.OrganizationID, id, m, app, shop)
	scopes := append([]string(nil), tokens.GrantedScopes...)
	sort.Strings(scopes)
	raw, _ := json.Marshal(scopes)
	hash := sha256.Sum256(raw)
	binding.ScopesHash = hex.EncodeToString(hash[:])
	if !binding.ValidTikTokApplication() || expected != nil && *expected != binding {
		return nil, o.ErrConflict
	}
	h := &ObservationHandle{owner: a, subject: s, binding: binding, connectionRef: m.Connection.AttemptID, expiresAt: started.Add(10 * time.Second), tiktokOrders: &tikTokOrderAuthority{material: private, application: app, shop: shop}}
	if _, e = h.tikTokOrderCredential(ctx); e != nil {
		return nil, e
	}
	return h, nil
}

func (h *ObservationHandle) tikTokOrderCredential(ctx context.Context) (auth.TokenSet, error) {
	var zero auth.TokenSet
	if h == nil || h.owner == nil || h.tiktokOrders == nil || ctx == nil || ctx.Err() != nil || h.subject.Purpose != storecenter.ObservationPurposeOrders || !h.owner.now().Before(h.expiresAt) {
		return zero, o.ErrForbidden
	}
	ctx, cancel := context.WithDeadline(ctx, h.expiresAt)
	defer cancel()
	a := h.owner
	authority := h.tiktokOrders
	if a.applications.tiktokProvider.Application() != authority.application {
		return zero, o.ErrConflict
	}
	m, e := a.reader.ReadObservation(ctx, h.subject, h.binding.StoreID, a.authorization, a.now())
	if e != nil {
		return zero, observationError(e)
	}
	if m.Platform != storecenter.PlatformTikTokShop || m.TikTok == nil || *m.TikTok != authority.material || m.StoreVersion != h.binding.StoreVersion || m.Connection.Version != h.binding.ConnectionRevision || m.Connection.AttemptID != h.connectionRef || !m.ServiceExpiresAt.Equal(h.binding.ServiceExpiresAt) || !a.now().Before(m.ServiceExpiresAt) {
		return zero, o.ErrConflict
	}
	grant, e := auth.RestoreGrant(m.TikTok.Grant)
	if e != nil || !grant.CanSend(a.now()) {
		return zero, o.ErrConflict
	}
	tokens, e := a.applications.tiktokProtection.Open(grantCredentialBinding(m.TikTok.Grant), m.TikTok.Grant.KeyID, m.TikTok.Grant.Ciphertext)
	if e != nil || !tikTokOrderScopes(tokens) || sellerIdentityHash(tokens.OpenID) != h.binding.SupplierIdentityHash || !tokens.AccessExpiresAt.Equal(h.binding.CredentialExpiresAt) || !a.now().Before(tokens.AccessExpiresAt) {
		return zero, o.ErrConflict
	}
	return tokens, nil
}

func (a *OfficialObservationAccess) ReadTikTokOrders(ctx context.Context, s o.Scope, binding o.Binding, q o.TikTokOrderQuery) (o.TikTokOrderPage, error) {
	h, e := a.Authorize(ctx, storecenter.ObservationSubject{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Purpose: storecenter.ObservationPurposeOrders, Sync: true}, binding.StoreID, &binding)
	if e != nil {
		return o.TikTokOrderPage{}, e
	}
	ctx, cancel := context.WithDeadline(ctx, h.expiresAt)
	defer cancel()
	tokens, e := h.tikTokOrderCredential(ctx)
	if e != nil {
		return o.TikTokOrderPage{}, e
	}
	page, e := a.applications.tiktokProvider.(TikTokOrdersReader).Orders(ctx, tokens.AccessToken, h.tiktokOrders.shop.Cipher, q)
	if e != nil {
		return o.TikTokOrderPage{}, o.ErrUnavailable
	}
	if _, e = h.tikTokOrderCredential(ctx); e != nil {
		return o.TikTokOrderPage{}, e
	}
	return page, nil
}
