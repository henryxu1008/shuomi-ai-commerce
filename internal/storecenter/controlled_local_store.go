package storecenter

import "context"

// ReadControlledLocalStore uses existing member access and Store facts. A
// disconnected row alone is insufficient if it retains a merchant identity.
func (r *MemberScopedStoreRepository) ReadControlledLocalStore(ctx context.Context, org, id string) (*Store, error) {
	store, err := r.Get(ctx, org, id)
	if err != nil {
		return nil, err
	}
	if store.Platform() != PlatformShein || store.Region() != "US" || store.RecordStatus() != RecordStatusActive || store.ServiceStatus() != ServiceStatusPendingActivation || store.ConnectionRef() != "" {
		return nil, ErrNotFound
	}
	var n int64
	if err = r.db.WithContext(ctx).Model(&officialMerchantBinding{}).Where("organization_id=? AND store_id=?", org, id).Count(&n).Error; err != nil {
		return nil, err
	}
	if n != 0 {
		return nil, ErrNotFound
	}
	if err = r.db.WithContext(ctx).Model(&officialConnectionRow{}).Where("organization_id=? AND store_id=?", org, id).Count(&n).Error; err != nil {
		return nil, err
	}
	if n != 0 {
		return nil, ErrNotFound
	}
	return store, nil
}
