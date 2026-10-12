package commercialbilling

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/commercial/billing"
	capacitystore "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/ledger/money"
	"task-processor/internal/ledger/storagecapacity"
)

func storageLot(o billing.Order) (storagecapacity.PurchaseLot, error) {
	if o.Kind != billing.OrderStorageCapacityPurchase || o.Validate() != nil {
		return storagecapacity.PurchaseLot{}, billing.ErrInvalid
	}
	i := o.Items[0]
	return storagecapacity.PurchaseLot{OrganizationID: o.OrganizationID, OrderID: o.OrderID, ItemID: i.OrderItemID, Bytes: i.ResourceQuantity * i.StorageUnitBytes, DurationSeconds: i.StorageDurationSeconds, TermsFingerprint: fingerprint([]any{o.OrderID, i.OrderItemID, i.StorageUnitBytes, i.StorageDurationSeconds, i.ResourceQuantity, o.AmountMinor, o.Currency})}, nil
}
func (r *Repository) PrepareStorageOrder(ctx context.Context, o billing.Order) error {
	l, e := storageLot(o)
	if e != nil {
		return e
	}
	capacity, e := capacitystore.New(r.db)
	if e != nil {
		return e
	}
	return capacity.PrepareLot(ctx, l)
}
func (r *Repository) ActivateStorageOrder(ctx context.Context, o billing.Order) (out billing.Order, err error) {
	// The service verifies this original reservation against Money's durable
	// committed decision. No HTTP client can supply a wallet or activation proof.
	if o.WalletReservationState != money.WalletReservationCommitted || o.WalletReservationID == "" {
		return out, billing.ErrInvalid
	}
	l, e := storageLot(o)
	if e != nil {
		return out, e
	}
	capacity, e := capacitystore.New(r.db)
	if e != nil {
		return out, e
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row orderRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "organization_id = ? AND order_id = ?", o.OrganizationID, o.OrderID).Error; e != nil {
			return e
		}
		var item orderItemRow
		if e := tx.First(&item, "order_id = ?", o.OrderID).Error; e != nil {
			return e
		}
		current := orderFromRows(row, item)
		currentLot, e := storageLot(current)
		if e != nil || currentLot != l || current.WalletReservationID != o.WalletReservationID {
			return billing.ErrConflict
		}
		if current.Status == billing.OrderFulfilled {
			out = current
			return nil
		}
		if current.Version != o.Version {
			return billing.ErrConflict
		}
		if current.Status == billing.OrderCancelled || current.Status == billing.OrderPending {
			return billing.ErrConflict
		}
		if e := capacity.ActivateLotInTransaction(ctx, tx, l, o.WalletReservationID); e != nil {
			return e
		}
		row.Status = string(billing.OrderFulfilled)
		row.WalletReservationState = string(money.WalletReservationCommitted)
		row.StorageActivationReceipt = o.OrderID
		row.Version++
		row.UpdatedAt = r.now().UTC()
		if e := tx.Save(&row).Error; e != nil {
			return e
		}
		out = orderFromRows(row, item)
		return nil
	})
	return
}
func (r *Repository) ListStorageOffers(ctx context.Context) ([]billing.Offer, error) {
	var rows []offerRow
	if e := r.db.WithContext(ctx).Where("product_kind = ? AND status = ?", billing.ProductStorageCapacityPack, billing.OfferActive).Order("offer_id").Limit(17).Find(&rows).Error; e != nil {
		return nil, e
	}
	if len(rows) > 16 {
		return nil, billing.ErrFeatureUnavailable
	}
	out := []billing.Offer{}
	for _, row := range rows {
		offer, e := r.ReadOffer(ctx, row.OfferID)
		if e == billing.ErrOfferUnavailable {
			continue
		}
		if e != nil {
			return nil, e
		}
		out = append(out, offer)
	}
	return out, nil
}
