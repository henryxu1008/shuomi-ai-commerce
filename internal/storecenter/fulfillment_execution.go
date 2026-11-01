package storecenter

import (
	"context"
	"time"
)

type FulfillmentSubject struct{ OrganizationID, ActorID, MemberID, Purpose string }
type FulfillmentAuthorizer interface {
	AuthorizeFulfillment(context.Context, FulfillmentSubject) (ProductExecutionAuthorization, error)
}
type FulfillmentReader interface {
	ReadFulfillment(context.Context, FulfillmentSubject, string, FulfillmentAuthorizer, time.Time) (ProductExecutionMaterial, error)
}

func (r *MemberScopedStoreRepository) ReadFulfillment(ctx context.Context, s FulfillmentSubject, store string, a FulfillmentAuthorizer, now time.Time) (ProductExecutionMaterial, error) {
	if isNilDependency(a) || !ValidFulfillmentPurpose(s.Purpose) {
		return ProductExecutionMaterial{}, ErrNotFound
	}
	return r.readOfficialMaterial(ctx, StoreMemberAccess{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID}, store, now, func(ctx context.Context) (ProductExecutionAuthorization, error) {
		return a.AuthorizeFulfillment(ctx, s)
	})
}
func ValidFulfillmentPurpose(p string) bool {
	switch p {
	case "fulfillment.read", "fulfillment.purchase", "fulfillment.stock", "fulfillment.address", "fulfillment.execute":
		return true
	}
	return false
}
