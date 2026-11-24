package storecenter

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
)

// InstallIsolatedAmazonOrderConsentTx runs only in the synthetic test assembly.
func InstallIsolatedAmazonOrderConsentTx(ctx context.Context, tx *sql.Tx) error {
	if tx == nil {
		return errors.New("Amazon order consent requires its isolated Store owner transaction")
	}
	_, err := tx.ExecContext(ctx, `CREATE TABLE public.workbench_store_amazon_order_consents (
 organization_id VARCHAR(200) NOT NULL, store_id CHAR(36) NOT NULL, authorization_generation BIGINT NOT NULL CHECK(authorization_generation > 0),
 order_generation BIGINT NOT NULL CHECK(order_generation > 0), granted BOOLEAN NOT NULL, PRIMARY KEY (organization_id, store_id),
 FOREIGN KEY (organization_id, store_id) REFERENCES public.workbench_store_amazon_grants (organization_id, store_id)
)`)
	return err
}

// LockAmazonOrderObservation checks the Store and both authorization
// generations before publishing anything obtained by an isolated order read.
func LockAmazonOrderObservation(tx *gorm.DB, b ProductMerchantBinding) error {
	if tx == nil || !b.ValidAmazonOrderApplication() {
		return ErrNotFound
	}
	var found bool
	err := tx.Raw(`SELECT true FROM public.workbench_stores
		WHERE organization_id=? AND id=? AND platform='amazon' AND region='US' AND record_status='active'
		  AND version=? AND service_status='active' AND service_expires_at>CURRENT_TIMESTAMP FOR SHARE`,
		b.OrganizationID, b.StoreID, b.StoreVersion).Row().Scan(&found)
	if err != nil {
		return amazonOrderLockError(err)
	}
	var boundAt time.Time
	err = tx.Raw(`SELECT bound_at FROM public.workbench_store_amazon_grants
		WHERE organization_id=? AND store_id=? AND app_id=? AND app_revision=? AND beta=TRUE AND generation=?
		  AND bound_at<=CURRENT_TIMESTAMP AND bound_at + INTERVAL '8760 hours'>CURRENT_TIMESTAMP FOR SHARE`,
		b.OrganizationID, b.StoreID, b.ApplicationID, b.ApplicationRevision, b.ConnectionRevision).Row().Scan(&boundAt)
	if err != nil {
		return amazonOrderLockError(err)
	}
	if !boundAt.Add(amazonAuthorizationRenewal).Equal(b.CredentialExpiresAt) {
		return ErrNotFound
	}
	err = tx.Raw(`SELECT true FROM public.workbench_store_amazon_order_consents
		WHERE organization_id=? AND store_id=? AND authorization_generation=? AND order_generation=? AND granted FOR SHARE`,
		b.OrganizationID, b.StoreID, b.ConnectionRevision, b.AmazonOrderGeneration).Row().Scan(&found)
	return amazonOrderLockError(err)
}

func amazonOrderLockError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return ErrDependencyUnavailable
}
