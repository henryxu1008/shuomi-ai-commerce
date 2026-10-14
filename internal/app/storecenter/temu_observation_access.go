package storecenterapp

import (
	"context"
	"encoding/json"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"time"
)

type TemuObservationProvider interface {
	QueryObservedTemuProducts(context.Context, storecenter.OfficialMerchantCredential, o.ProductQuery) (o.ProductPage, error)
}

func (a *OfficialObservationAccess) authorizeTemuObservation(ctx context.Context, startedAt time.Time, s storecenter.ObservationSubject, store string, expected *MerchantBinding, m storecenter.ProductExecutionMaterial, entry officialApplicationEntry) (*ObservationHandle, error) {
	// One fixed read-only page budget includes initial admission. Current
	// authority is still reread before and after every provider observation.
	expires := startedAt.Add(30 * time.Second)
	if deadline, ok := ctx.Deadline(); ok && deadline.Before(expires) {
		expires = deadline
	}
	ctx, cancel := context.WithDeadline(ctx, expires)
	defer cancel()
	if s.Purpose != storecenter.ObservationPurposeProducts {
		return nil, o.ErrUnsupported
	}
	if _, ok := entry.provider.(TemuObservationProvider); !ok {
		return nil, o.ErrUnavailable
	}
	credential, err := entry.protection.Open(m.Attempt, m.Attempt.KeyID, m.Attempt.Ciphertext)
	if err != nil {
		return nil, o.ErrConflict
	}
	binding, valid := verifiedTemuMerchantBinding(s.OrganizationID, store, m, credential, entry, a.now())
	if !valid || expected != nil && *expected != binding {
		return nil, o.ErrConflict
	}
	information, err := entry.provider.QueryStore(ctx, credential)
	if err != nil || information.Verification == nil {
		return nil, o.ErrUnavailable
	}
	current := m
	raw, err := json.Marshal(information.Verification)
	if err != nil {
		return nil, o.ErrUnavailable
	}
	current.Attempt.VerificationJSON = string(raw)
	actual, valid := verifiedTemuMerchantBinding(s.OrganizationID, store, current, credential, entry, a.now())
	if !valid || actual != binding {
		return nil, o.ErrConflict
	}
	fresh, err := a.reader.ReadObservation(ctx, s, store, a.authorization, a.now())
	if err != nil {
		return nil, observationError(err)
	}
	actual, valid = verifiedTemuMerchantBinding(s.OrganizationID, store, fresh, credential, entry, a.now())
	if !valid || actual != binding || privateCredentialHash(fresh) != privateCredentialHash(m) || ctx.Err() != nil {
		return nil, o.ErrConflict
	}
	if ctx.Err() != nil || !a.now().Before(expires) {
		return nil, o.ErrUnavailable
	}
	return &ObservationHandle{owner: a, entry: entry, subject: s, binding: binding, connectionRef: m.Connection.AttemptID, credentialHash: privateCredentialHash(m), expiresAt: expires}, nil
}

func (h *ObservationHandle) TemuProducts(ctx context.Context, query o.ProductQuery) (o.ProductPage, error) {
	if h == nil || h.binding.Platform != storecenter.PlatformTemu || query.Page < 1 || len(query.PageToken) > 8192 {
		return o.ProductPage{}, o.ErrInvalid
	}
	return observationCall(ctx, h, storecenter.ObservationPurposeProducts, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (o.ProductPage, error) {
		return h.entry.provider.(TemuObservationProvider).QueryObservedTemuProducts(ctx, c, query)
	})
}
