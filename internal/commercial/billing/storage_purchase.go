package billing

import (
	"context"
	"task-processor/internal/ledger/money"
	"task-processor/internal/ledger/storagecapacity"
)

type StoragePurchaseStore interface {
	PrepareStorageOrder(context.Context, Order) error
	ActivateStorageOrder(context.Context, Order) (Order, error)
}
type StorageOfferCatalog interface {
	ListStorageOffers(context.Context) ([]Offer, error)
}

func InitialStorageOffer() Offer {
	return Offer{OfferID: "storage-50gib-30days-v1", ProductKind: ProductStorageCapacityPack, Currency: CurrencyCNY, UnitPriceMinor: 1990, PricingVersion: "storage-v1-20261012", MinQuantity: 1, MaxQuantity: 10, Status: OfferActive, StorageUnitBytes: storagecapacity.PackBytes, StorageDurationSeconds: storagecapacity.PeriodSeconds}
}
func (s *Service) ConfigureStorage(store StoragePurchaseStore) { s.storage = store }
func (s *Service) ListStorageOffers(ctx context.Context) ([]Offer, error) {
	if s == nil || s.storage == nil {
		return nil, ErrFeatureUnavailable
	}
	catalog, ok := s.offers.(StorageOfferCatalog)
	if !ok {
		return nil, ErrFeatureUnavailable
	}
	offers, e := catalog.ListStorageOffers(ctx)
	if e != nil {
		return nil, e
	}
	if len(offers) > 16 {
		return nil, ErrFeatureUnavailable
	}
	for _, o := range offers {
		if o.Validate() != nil || o.ProductKind != ProductStorageCapacityPack {
			return nil, ErrFeatureUnavailable
		}
	}
	return offers, nil
}
func (s *Service) fulfillStorage(ctx context.Context, order Order) (Order, error) {
	if s.storage == nil || order.Kind != OrderStorageCapacityPurchase || order.Validate() != nil {
		return order, ErrFeatureUnavailable
	}
	if order.Status == OrderFulfilled {
		return order, nil
	}
	if order.Status == OrderCancelled {
		return order, cancellationError(order.FailureCode)
	}
	if order.WalletReservationID == "" {
		return order, ErrReconciliationRequired
	}
	reservation, e := s.wallet.ReadCommercialPurchaseReservation(ctx, order.OrganizationID, order.OrderID, order.WalletReservationID)
	if e != nil || reservation.OrganizationID != order.OrganizationID || reservation.CommercialOrderID != order.OrderID || reservation.ReservationID != order.WalletReservationID || reservation.Currency != order.Currency || reservation.AmountMinor != order.AmountMinor {
		return order, ErrReconciliationRequired
	}
	if reservation.State != money.WalletReservationReserved && reservation.State != money.WalletReservationCommitted {
		return order, ErrReconciliationRequired
	}
	// Preparing a lot adds no capacity. Commit Money before commercial activation.
	if e = s.storage.PrepareStorageOrder(ctx, order); e != nil {
		return order, ErrReconciliationRequired
	}
	if reservation.State == money.WalletReservationReserved {
		order.Status = OrderFulfilling
		order.WalletReservationState = money.WalletReservationReserved
		order.UpdatedAt = s.now().UTC()
		if e = s.updateOrder(ctx, &order); e != nil {
			return order, ErrReconciliationRequired
		}
		reservation, e = s.wallet.CommitCommercialPurchase(ctx, money.CommitWalletReservationInput{OperationID: "commit:" + order.OrderID, OrganizationID: order.OrganizationID, CommercialOrderID: order.OrderID, ReservationID: order.WalletReservationID})
		if e != nil || reservation.State != money.WalletReservationCommitted {
			order.Status = OrderReconciliationRequired
			order.UpdatedAt = s.now().UTC()
			_ = s.updateOrder(ctx, &order)
			return order, ErrReconciliationRequired
		}
	}
	order.WalletReservationState = money.WalletReservationCommitted
	// Recovery may observe Money's original commit before the commercial phase
	// catches up. Keep that recoverable shape valid until the activation UoW.
	if order.Status == OrderFundsReserved {
		order.Status = OrderReconciliationRequired
	}
	active, e := s.storage.ActivateStorageOrder(ctx, order)
	if e != nil {
		return order, ErrReconciliationRequired
	}
	return active, nil
}
func (s *Service) ReconcileStorageOrder(ctx context.Context, org, id string) (Order, error) {
	if s == nil || s.storage == nil {
		return Order{}, ErrFeatureUnavailable
	}
	order, e := s.reader.ReadOrder(ctx, org, id)
	if e != nil {
		return Order{}, e
	}
	if order.Kind != OrderStorageCapacityPurchase {
		return Order{}, ErrInvalid
	}
	if order.Status == OrderFulfilled || order.Status == OrderCancelled {
		return order, nil
	}
	if order.WalletReservationID == "" {
		decision, e := s.wallet.ReadCommercialPurchaseReserveDecision(ctx, org, id, id)
		if e != nil || decision.Validate() != nil || decision.OrganizationID != org || decision.CommercialOrderID != id || decision.OperationID != id || decision.AmountMinor != order.AmountMinor || decision.Currency != order.Currency {
			return order, ErrReconciliationRequired
		}
		if decision.Outcome == money.WalletReserveDecisionRejectedInsufficientFunds {
			order.Status = OrderCancelled
			order.FailureCode = OrderFailureInsufficientFunds
			order.UpdatedAt = s.now().UTC()
			if s.updateOrder(ctx, &order) != nil {
				return order, ErrReconciliationRequired
			}
			return order, ErrInsufficientFunds
		}
		if decision.Outcome != money.WalletReserveDecisionReserved {
			return order, ErrReconciliationRequired
		}
		order.WalletReservationID = decision.ReservationID
		order.WalletReservationState = money.WalletReservationReserved
		order.Status = OrderFundsReserved
		order.UpdatedAt = s.now().UTC()
		if s.updateOrder(ctx, &order) != nil {
			return order, ErrReconciliationRequired
		}
	}
	return s.fulfillStorage(ctx, order)
}
