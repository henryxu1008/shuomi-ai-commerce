package storecenterapp

import (
	"context"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
)

// ReadSavedWalmartOrderBinding checks current local order/Store authority only.
// It never opens credentials, uses product rules permissions, or calls a provider.
func (a *OfficialObservationAccess) ReadSavedWalmartOrderBinding(ctx context.Context, subject storecenter.ObservationSubject, id string) (MerchantBinding, error) {
	if a == nil || ctx == nil || ctx.Err() != nil || a.applications == nil || !a.applications.WalmartConfigured() || subject.Purpose != storecenter.ObservationPurposeOrders || subject.Sync {
		return MerchantBinding{}, o.ErrUnavailable
	}
	now := a.now()
	m, e := a.reader.ReadObservation(ctx, subject, id, a.authorization, now)
	if e != nil {
		return MerchantBinding{}, observationError(e)
	}
	if m.Platform != storecenter.PlatformWalmart || m.Walmart == nil {
		return MerchantBinding{}, o.ErrNotFound
	}
	g := *m.Walmart
	app := a.applications.walmartProvider.Application()
	if g.OrganizationID != subject.OrganizationID || g.StoreID != id || g.AppID != app.ClientID || g.AppRevision != app.Revision || g.Environment != app.Environment || !g.CanRead(now) || !now.Before(m.ServiceExpiresAt) {
		return MerchantBinding{}, o.ErrConflict
	}
	b := walmartMerchantBinding(m, g)
	if !b.ValidWalmartApplication() {
		return MerchantBinding{}, o.ErrConflict
	}
	return b, nil
}
