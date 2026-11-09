package storecenterapp

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
)

type TemuOrdersProvider interface {
	QueryObservedTemuOrders(context.Context, storecenter.OfficialMerchantCredential, o.TemuOrderQuery) (o.TemuOrderPage, error)
}

func (a *OfficialObservationAccess) authorizeTemuOrders(ctx context.Context, expires time.Time, s storecenter.ObservationSubject, id string, expected *MerchantBinding, m storecenter.ProductExecutionMaterial, entry officialApplicationEntry) (*ObservationHandle, error) {
	if entry.application.SupportedModes != storecenter.ModesLocal|storecenter.ModesSemiManaged {
		return nil, o.ErrUnsupported
	}
	if _, ok := entry.provider.(TemuOrdersProvider); !ok {
		return nil, o.ErrUnavailable
	}
	binding, e := temuOrderBinding(s.OrganizationID, id, m, entry, a.now())
	if e != nil || expected != nil && *expected != binding {
		return nil, o.ErrConflict
	}
	var facts storecenter.OfficialVerificationFacts
	if json.Unmarshal([]byte(m.Attempt.VerificationJSON), &facts) != nil || !slices.Contains(facts.Scopes, "bg.order.list.v2.get") {
		return nil, o.ErrForbidden
	}
	h := &ObservationHandle{owner: a, entry: entry, subject: s, binding: binding, connectionRef: m.Connection.AttemptID, credentialHash: privateCredentialHash(m), expiresAt: expires}
	if e = h.Check(ctx); e != nil {
		return nil, e
	}
	// The identity request cannot release a handle after local revocation.
	if _, e = h.credential(ctx); e != nil {
		return nil, e
	}
	return h, nil
}

func (h *ObservationHandle) checkTemuOrderIdentity(ctx context.Context, credential storecenter.OfficialMerchantCredential) error {
	ctx, cancel := context.WithDeadline(ctx, h.expiresAt)
	defer cancel()
	information, e := h.entry.provider.QueryStore(ctx, credential)
	if e != nil || information.Verification == nil {
		return o.ErrUnavailable
	}
	m, e := h.owner.reader.ReadObservation(ctx, h.subject, h.binding.StoreID, h.owner.authorization, h.owner.now())
	if e != nil || privateCredentialHash(m) != h.credentialHash {
		return o.ErrConflict
	}
	raw, e := json.Marshal(information.Verification)
	if e != nil {
		return o.ErrConflict
	}
	m.Attempt.VerificationJSON = string(raw)
	binding, e := temuOrderBinding(h.subject.OrganizationID, h.binding.StoreID, m, h.entry, h.owner.now())
	if e != nil || binding != h.binding || ctx.Err() != nil || !h.owner.now().Before(h.expiresAt) {
		return o.ErrConflict
	}
	return nil
}

func (a *OfficialObservationAccess) ReadTemuOrders(ctx context.Context, s o.Scope, binding o.Binding, query o.TemuOrderQuery) (o.TemuOrderPage, error) {
	if !binding.ValidTemuOrderApplication() {
		return o.TemuOrderPage{}, o.ErrInvalid
	}
	h, e := a.Authorize(ctx, storecenter.ObservationSubject{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Purpose: storecenter.ObservationPurposeOrders, Sync: true}, binding.StoreID, &binding)
	if e != nil {
		return o.TemuOrderPage{}, e
	}
	return observationCall(ctx, h, storecenter.ObservationPurposeOrders, func(ctx context.Context, credential storecenter.OfficialMerchantCredential) (o.TemuOrderPage, error) {
		return h.entry.provider.(TemuOrdersProvider).QueryObservedTemuOrders(ctx, credential, query)
	})
}

var _ o.TemuOrderReader = (*OfficialObservationAccess)(nil)
