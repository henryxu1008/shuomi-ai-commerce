package storecenterapp

import (
	"context"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	auth "task-processor/internal/storecenter/tiktokshop"
)

// ReadSavedTikTokOrderBinding is an identity projection for local saved-order
// reads, not an online orders-scope proof. It never opens a credential or calls
// AuthorizedShops/refresh/provider I/O, and does not borrow product authority.
func (a *OfficialObservationAccess) ReadSavedTikTokOrderBinding(ctx context.Context, s storecenter.ObservationSubject, id string) (MerchantBinding, error) {
	if a == nil || ctx == nil || ctx.Err() != nil || a.applications == nil || !a.applications.TikTokConfigured() || s.Purpose != storecenter.ObservationPurposeOrders || s.Sync {
		return MerchantBinding{}, o.ErrUnavailable
	}
	now := a.now()
	m, e := a.reader.ReadObservation(ctx, s, id, a.authorization, now)
	if e != nil {
		return MerchantBinding{}, observationError(e)
	}
	if m.Platform != storecenter.PlatformTikTokShop || m.TikTok == nil {
		return MerchantBinding{}, o.ErrNotFound
	}
	saved := m.TikTok
	g := saved.Grant
	app := a.applications.tiktokProvider.Application()
	grant, e := auth.RestoreGrant(g)
	if e != nil || !grant.CanSend(now) || g.OrganizationID != s.OrganizationID || g.AppKey != app.AppKey || g.AppRevision != app.Revision || saved.Binding.OrganizationID != s.OrganizationID || saved.Binding.StoreID != id || saved.Binding.GrantID != g.ID || saved.Binding.AppKey != app.AppKey || saved.Binding.Status != "active" || saved.Binding.Version < 1 || m.StoreVersion < 1 || m.Connection.Version < 1 || !now.Before(m.ServiceExpiresAt) {
		return MerchantBinding{}, o.ErrConflict
	}
	shop, e := auth.SelectUSCrossBorderShop([]auth.Shop{saved.Binding.Shop}, saved.Binding.Shop.Code)
	if e != nil {
		return MerchantBinding{}, o.ErrConflict
	}
	return tikTokOrderBinding(s.OrganizationID, id, m, app, shop), nil
}

func tikTokOrderBinding(org, id string, m storecenter.ProductExecutionMaterial, app auth.Application, shop auth.Shop) MerchantBinding {
	g := m.TikTok.Grant
	return MerchantBinding{Platform: storecenter.PlatformTikTokShop, OrganizationID: org, StoreID: id, Site: "tiktokshop-us", StoreVersion: m.StoreVersion, ConnectionRevision: m.Connection.Version, ApplicationID: app.AppKey, ApplicationRevision: app.Revision, SupplierIdentityHash: g.SellerIdentityHash, ServiceExpiresAt: m.ServiceExpiresAt.UTC(), CredentialExpiresAt: g.AccessExpiresAt.UTC(), TikTok: storecenter.TikTokMerchantBinding{GrantID: g.ID, ShopID: shop.ID, GrantGeneration: g.Generation, GrantRevision: g.Revision, BindingRevision: m.TikTok.Binding.Version}}
}
