package storecenter

import (
	"context"
	"database/sql"
	"errors"

	"gorm.io/gorm"
)

// InstallIsolatedShopifyOrderConsentTx is only used by the synthetic trial.
// The production Store installer never creates this table or grants access.
func InstallIsolatedShopifyOrderConsentTx(ctx context.Context, tx *sql.Tx) error {
	if tx == nil {
		return errors.New("Shopify order consent requires its isolated Store owner transaction")
	}
	_, err := tx.ExecContext(ctx, `CREATE TABLE public.workbench_store_shopify_order_consents (
 organization_id VARCHAR(200) NOT NULL, shop_key CHAR(36) NOT NULL, authorization_generation BIGINT NOT NULL,
 order_generation BIGINT NOT NULL CHECK(order_generation > 0), granted BOOLEAN NOT NULL,
 PRIMARY KEY (organization_id, shop_key),
 FOREIGN KEY (organization_id, shop_key, authorization_generation)
 REFERENCES public.workbench_store_shopify_authorizations(organization_id, shop_key, generation)
)`)
	return err
}

// LockShopifyOrderObservation binds an observation page to current local
// Store authority inside the page transaction. A product authorization alone
// never provides order consent, and an old grant generation cannot promote a head.
func LockShopifyOrderObservation(tx *gorm.DB, binding ProductMerchantBinding) error {
	if tx == nil || !binding.ValidShopifyOrderApplication() {
		return ErrNotFound
	}
	var found bool
	err := tx.Raw(`SELECT true FROM public.workbench_stores
		WHERE organization_id=? AND id=? AND platform='shopify' AND record_status='active'
		  AND version=? AND connection_ref=? AND service_status='active' AND service_expires_at=?
		  AND service_expires_at>CURRENT_TIMESTAMP FOR SHARE`,
		binding.OrganizationID, binding.StoreID, binding.StoreVersion, binding.Shopify.ShopKey, binding.ServiceExpiresAt).Row().Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return ErrDependencyUnavailable
	}
	err = tx.Raw(`SELECT true FROM public.workbench_store_shopify_shops shop
		JOIN public.workbench_store_shopify_authorizations approval
		  ON approval.organization_id=shop.organization_id AND approval.shop_key=shop.shop_key AND approval.generation=shop.generation
		JOIN public.workbench_store_shopify_order_consents consent
		  ON consent.organization_id=shop.organization_id AND consent.shop_key=shop.shop_key AND consent.authorization_generation=shop.generation
		WHERE shop.organization_id=? AND shop.store_id=? AND shop.shop_key=? AND shop.app_id=? AND shop.generation=?
		  AND approval.revoked_at IS NULL AND approval.currency=? AND consent.granted AND consent.order_generation=?
		  AND NOT EXISTS (SELECT 1 FROM public.workbench_store_shopify_lifecycle_events event
		    WHERE event.organization_id=shop.organization_id AND event.shop_key=shop.shop_key
		      AND event.generation=shop.generation AND event.status='pending')
		FOR SHARE OF shop, approval, consent`,
		binding.OrganizationID, binding.StoreID, binding.Shopify.ShopKey, binding.ApplicationID, binding.ConnectionRevision,
		binding.Shopify.Currency, binding.Shopify.OrderGeneration).Row().Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return ErrDependencyUnavailable
	}
	return nil
}
