package storecenterapp

import (
	"context"
	"errors"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"time"
)

type OfficialObservationProvider interface {
	QueryObservedProducts(context.Context, storecenter.OfficialMerchantCredential, int) (o.ProductPage, error)
	QueryObservedOrders(context.Context, storecenter.OfficialMerchantCredential, o.Window, int) (o.OrderPage, error)
	QueryObservedOrderDetails(context.Context, storecenter.OfficialMerchantCredential, []string) ([]o.Order, error)
	QueryObservedTrack(context.Context, storecenter.OfficialMerchantCredential, string, string) ([]o.Track, error)
}
type OfficialSupplyOrderProvider interface {
	QueryObservedSupplyOrders(context.Context, storecenter.OfficialMerchantCredential, []string) (o.SupplyPage, error)
}

func observationBinding(s storecenter.ObservationSubject, store string, m storecenter.ProductExecutionMaterial, k storecenter.OfficialMerchantCredential, mode storecenter.OfficialApplicationType) MerchantBinding {
	b := officialMerchantBinding(s.OrganizationID, store, m, k, mode)
	if s.Purpose == storecenter.ObservationPurposeSupplyOrders {
		b.Site = "unknown"
		b.Platform = storecenter.PlatformShein
	}
	return b
}

type OfficialObservationAccess struct {
	reader        storecenter.ObservationReader
	authorization storecenter.ObservationAuthorizer
	applications  *OfficialApplicationRegistry
	now           func() time.Time
}

// Credentials and authorization are private process-local fields. In
// particular this type has no method for any product or fulfillment mutation.
type ObservationHandle struct {
	tiktokOrders                  *tikTokOrderAuthority
	walmartGrant                  *storecenter.WalmartGrant
	owner                         *OfficialObservationAccess
	entry                         officialApplicationEntry
	subject                       storecenter.ObservationSubject
	binding                       MerchantBinding
	connectionRef, credentialHash string
	expiresAt                     time.Time
}

func NewOfficialObservationAccess(r storecenter.ObservationReader, a storecenter.ObservationAuthorizer, apps *OfficialApplicationRegistry) (*OfficialObservationAccess, error) {
	if r == nil || a == nil || apps == nil {
		return nil, o.ErrUnavailable
	}
	return &OfficialObservationAccess{r, a, apps, time.Now}, nil
}
func observationError(e error) error {
	if errors.Is(e, storecenter.ErrNotFound) {
		return o.ErrForbidden
	}
	return o.ErrUnavailable
}
func (a *OfficialObservationAccess) Authorize(ctx context.Context, s storecenter.ObservationSubject, store string, expected *MerchantBinding) (*ObservationHandle, error) {
	if a == nil || ctx == nil || ctx.Err() != nil {
		return nil, o.ErrUnavailable
	}
	if s.Purpose != storecenter.ObservationPurposeProducts && s.Purpose != storecenter.ObservationPurposeOrders && s.Purpose != storecenter.ObservationPurposeSupplyOrders {
		return nil, o.ErrForbidden
	}
	callerCtx, startedAt := ctx, a.now()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	m, e := a.reader.ReadObservation(ctx, s, store, a.authorization, a.now())
	if e != nil {
		return nil, observationError(e)
	}
	if s.Purpose == storecenter.ObservationPurposeSupplyOrders && m.Platform != storecenter.PlatformShein {
		return nil, o.ErrUnsupported
	}
	if m.Platform == storecenter.PlatformWalmart {
		return a.authorizeWalmartObservation(callerCtx, startedAt, s, store, expected, m)
	}
	if m.Platform == storecenter.PlatformTikTokShop {
		return a.authorizeTikTokOrders(callerCtx, startedAt, s, store, expected, m)
	}
	entry, e := a.applications.resolve(m.Attempt.AppID, m.Attempt.AppVersion, m.Platform, m.Attempt.IdentityPartition)
	if e != nil {
		return nil, o.ErrConflict
	}
	if m.Platform == storecenter.PlatformTemu {
		return a.authorizeTemuObservation(callerCtx, startedAt, s, store, expected, m, entry)
	}
	if _, ok := entry.provider.(OfficialObservationProvider); !ok {
		return nil, o.ErrUnavailable
	}
	if s.Purpose == storecenter.ObservationPurposeOrders && entry.mode == storecenter.ApplicationFullyManaged {
		return nil, o.ErrUnsupported
	}
	if s.Purpose == storecenter.ObservationPurposeSupplyOrders {
		if _, ok := entry.provider.(OfficialSupplyOrderProvider); !ok || entry.mode != storecenter.ApplicationFullyManaged {
			return nil, o.ErrUnsupported
		}
	}
	k, e := entry.protection.Open(m.Attempt, m.Attempt.KeyID, m.Attempt.Ciphertext)
	if e != nil || !validOfficialMaterial(s.OrganizationID, store, m, k, entry, a.now()) {
		return nil, o.ErrConflict
	}
	if s.Purpose == storecenter.ObservationPurposeOrders && entry.mode == storecenter.ApplicationSelfOperated && k.SheinBusinessMode != "" {
		return nil, o.ErrUnsupported
	}
	binding := observationBinding(s, store, m, k, entry.mode)
	if expected != nil && *expected != binding {
		return nil, o.ErrConflict
	}
	expires, _ := ctx.Deadline()
	if !a.now().Before(expires) {
		return nil, o.ErrUnavailable
	}
	return &ObservationHandle{owner: a, entry: entry, subject: s, binding: binding, connectionRef: m.Connection.AttemptID, credentialHash: privateCredentialHash(m), expiresAt: expires}, nil
}
func (h *ObservationHandle) Binding() MerchantBinding {
	if h == nil {
		return MerchantBinding{}
	}
	return h.binding
}
func (h *ObservationHandle) credential(ctx context.Context) (storecenter.OfficialMerchantCredential, error) {
	if h == nil || h.owner == nil || ctx == nil || ctx.Err() != nil {
		return storecenter.OfficialMerchantCredential{}, o.ErrUnavailable
	}
	if !h.owner.now().Before(h.expiresAt) {
		return storecenter.OfficialMerchantCredential{}, o.ErrUnavailable
	}
	ctx, cancel := context.WithDeadline(ctx, h.expiresAt)
	defer cancel()
	m, e := h.owner.reader.ReadObservation(ctx, h.subject, h.binding.StoreID, h.owner.authorization, h.owner.now())
	if e != nil {
		return storecenter.OfficialMerchantCredential{}, observationError(e)
	}
	entry, e := h.owner.applications.resolve(m.Attempt.AppID, m.Attempt.AppVersion, m.Platform, m.Attempt.IdentityPartition)
	if e != nil || entry.application != h.entry.application || entry.mode != h.entry.mode || m.Connection.AttemptID != h.connectionRef || privateCredentialHash(m) != h.credentialHash {
		return storecenter.OfficialMerchantCredential{}, o.ErrConflict
	}
	k, e := entry.protection.Open(m.Attempt, m.Attempt.KeyID, m.Attempt.Ciphertext)
	if m.Platform == storecenter.PlatformTemu && h.subject.Purpose == storecenter.ObservationPurposeOrders {
		binding, err := temuOrderBinding(h.subject.OrganizationID, h.binding.StoreID, m, entry, h.owner.now())
		if e != nil || err != nil || !validTemuPrivateCredential(k) || k.AppID != entry.application.AppID || binding != h.binding || ctx.Err() != nil || !h.owner.now().Before(h.expiresAt) {
			return storecenter.OfficialMerchantCredential{}, o.ErrConflict
		}
		return k, nil
	}
	if m.Platform == storecenter.PlatformTemu {
		binding, valid := verifiedTemuMerchantBinding(h.subject.OrganizationID, h.binding.StoreID, m, k, entry, h.owner.now())
		if e != nil || !valid || binding != h.binding || ctx.Err() != nil || !h.owner.now().Before(h.expiresAt) {
			return storecenter.OfficialMerchantCredential{}, o.ErrConflict
		}
		return k, nil
	}
	if e != nil || !validOfficialMaterial(h.subject.OrganizationID, h.binding.StoreID, m, k, entry, h.owner.now()) || observationBinding(h.subject, h.binding.StoreID, m, k, entry.mode) != h.binding {
		return storecenter.OfficialMerchantCredential{}, o.ErrConflict
	}
	if h.subject.Purpose == storecenter.ObservationPurposeOrders && entry.mode == storecenter.ApplicationSelfOperated && k.SheinBusinessMode != "" {
		return storecenter.OfficialMerchantCredential{}, o.ErrUnsupported
	}
	return k, nil
}
func (h *ObservationHandle) Check(ctx context.Context) error {
	if h != nil && h.tiktokOrders != nil {
		_, e := h.tikTokOrderCredential(ctx)
		return e
	}
	if h != nil && h.walmartGrant != nil {
		_, e := h.walmartCredential(ctx)
		return e
	}
	k, e := h.credential(ctx)
	if e == nil && h.binding.Platform == storecenter.PlatformTemu && h.subject.Purpose == storecenter.ObservationPurposeOrders {
		return h.checkTemuOrderIdentity(ctx, k)
	}
	return e
}
func observationCall[T any](ctx context.Context, h *ObservationHandle, purpose string, call func(context.Context, storecenter.OfficialMerchantCredential) (T, error)) (T, error) {
	var zero T
	if h == nil || h.subject.Purpose != purpose {
		return zero, o.ErrForbidden
	}
	ctx, cancel := context.WithDeadline(ctx, h.expiresAt)
	defer cancel()
	k, e := h.credential(ctx)
	if e != nil {
		return zero, e
	}
	result, e := call(ctx, k)
	if e != nil {
		return zero, e
	}
	if e = h.Check(ctx); e != nil {
		return zero, e
	}
	return result, nil
}
func (h *ObservationHandle) Products(ctx context.Context, page int) (o.ProductPage, error) {
	if h != nil && h.walmartGrant != nil {
		return h.walmartProducts(ctx, page)
	}
	if h == nil || h.binding.Platform == storecenter.PlatformTemu {
		return o.ProductPage{}, o.ErrUnsupported
	}
	return observationCall(ctx, h, storecenter.ObservationPurposeProducts, func(ctx context.Context, k storecenter.OfficialMerchantCredential) (o.ProductPage, error) {
		return h.entry.provider.(OfficialObservationProvider).QueryObservedProducts(ctx, k, page)
	})
}
func (h *ObservationHandle) Orders(ctx context.Context, w o.Window, page int) (o.OrderPage, error) {
	return observationCall(ctx, h, storecenter.ObservationPurposeOrders, func(ctx context.Context, k storecenter.OfficialMerchantCredential) (o.OrderPage, error) {
		return h.entry.provider.(OfficialObservationProvider).QueryObservedOrders(ctx, k, w, page)
	})
}
func (h *ObservationHandle) SupplyOrders(ctx context.Context, ids []string) (o.SupplyPage, error) {
	return observationCall(ctx, h, storecenter.ObservationPurposeSupplyOrders, func(ctx context.Context, k storecenter.OfficialMerchantCredential) (o.SupplyPage, error) {
		provider, ok := h.entry.provider.(OfficialSupplyOrderProvider)
		if !ok || h.entry.mode != storecenter.ApplicationFullyManaged {
			return o.SupplyPage{}, o.ErrUnsupported
		}
		return provider.QueryObservedSupplyOrders(ctx, k, ids)
	})
}
func (h *ObservationHandle) OrderDetails(ctx context.Context, ids []string) ([]o.Order, error) {
	return observationCall(ctx, h, storecenter.ObservationPurposeOrders, func(ctx context.Context, k storecenter.OfficialMerchantCredential) ([]o.Order, error) {
		return h.entry.provider.(OfficialObservationProvider).QueryObservedOrderDetails(ctx, k, ids)
	})
}
func (h *ObservationHandle) Track(ctx context.Context, order, pkg string) ([]o.Track, error) {
	return observationCall(ctx, h, storecenter.ObservationPurposeOrders, func(ctx context.Context, k storecenter.OfficialMerchantCredential) ([]o.Track, error) {
		return h.entry.provider.(OfficialObservationProvider).QueryObservedTrack(ctx, k, order, pkg)
	})
}

var _ o.Merchant = (*ObservationHandle)(nil)
