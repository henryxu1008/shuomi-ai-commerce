package billing

import (
	"context"
	"task-processor/internal/ledger/orgresource"
)

// fulfillZeroPriceOrder uses the same source-bound grant and order CAS as paid
// orders. It never fabricates a money reserve decision or a payment receipt.
func (s *Service) fulfillZeroPriceOrder(ctx context.Context, order Order) (Order, error) {
	if order.Validate() != nil || order.SettlementMode != SettlementZeroPrice || s.grants == nil {
		return order, ErrInvalid
	}
	if order.Status == OrderFulfilled {
		return order, nil
	}
	if order.Status == OrderCancelled {
		return order, cancellationError(order.FailureCode)
	}
	if order.Status == OrderPending {
		order.Status = OrderFulfilling
		order.UpdatedAt = s.now().UTC()
		if err := s.updateOrder(ctx, &order); err != nil {
			return order, ErrReconciliationRequired
		}
	}
	item := order.Items[0]
	grant, err := s.grants.GrantPurchasedResource(ctx, orgresource.PurchasedResourceGrantInput{OrganizationID: order.OrganizationID, OperationID: "grant:" + order.OrderID, CommercialOrderID: order.OrderID, CommercialOrderItemID: item.OrderItemID, ResourceType: item.ResourceType, Quantity: item.ResourceQuantity, Principal: orgresource.Principal{ID: "commercial-billing", Kind: orgresource.PrincipalTrustedCommercial}})
	if err != nil || !matchesGrantProof(order, grant.Snapshot) {
		if terminalGrantError(err) {
			order.Status = OrderCancelled
			order.FailureCode = OrderFailureGrantRejected
		} else {
			order.Status = OrderReconciliationRequired
		}
		order.UpdatedAt = s.now().UTC()
		if s.updateOrder(ctx, &order) != nil {
			return order, ErrReconciliationRequired
		}
		if order.Status == OrderCancelled {
			return order, ErrResourceGrantRejected
		}
		return order, ErrReconciliationRequired
	}
	order.ResourceGrantOperationID = grant.Snapshot.OperationID
	order.ResourceGrantSourceType = grant.Snapshot.SourceType
	order.ResourceGrantSourceIdentity = grant.Snapshot.SourceIdentity
	order.Status = OrderFulfilled
	order.UpdatedAt = s.now().UTC()
	if s.updateOrder(ctx, &order) != nil {
		return order, ErrReconciliationRequired
	}
	return order, nil
}
