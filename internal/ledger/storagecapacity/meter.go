package storagecapacity

import "context"

// Meter is scoped to the physical storage namespace, never to a caller supplied
// bucket. The composition root supplies both the owner port and namespace.
type Meter struct {
	Writer    Writer
	Namespace string
}

func (m *Meter) Admit(ctx context.Context, org, owner, operation, fingerprint string, o Object) (Reservation, error) {
	if m == nil || m.Writer == nil || !ValidID(m.Namespace, 256) {
		return Reservation{}, ErrUnavailable
	}
	o.Namespace = m.Namespace
	r, e := m.Writer.ReserveWrite(ctx, WriteIntent{OrganizationID: org, Owner: owner, OperationID: operation, Fingerprint: fingerprint, MaxBytes: o.SizeBytes, MaxObjects: 1})
	if e != nil {
		return r, e
	}
	if e = m.Writer.BindObjects(ctx, org, r.ID, []Object{o}); e != nil {
		return r, e
	}
	if e = m.Writer.ClaimDispatch(ctx, org, r.ID); e != nil {
		return r, e
	}
	return r, nil
}
func (m *Meter) Confirm(ctx context.Context, org string, r Reservation, o Object) error {
	if m == nil || m.Writer == nil {
		return ErrUnavailable
	}
	return m.Writer.ConfirmStored(ctx, org, r.ID, m.Namespace, o.Key, o.SHA256, o.SizeBytes)
}
