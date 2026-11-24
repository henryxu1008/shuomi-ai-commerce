package storecenter

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const shopifyPrivacyDeadline = 30 * 24 * time.Hour

// InstallIsolatedShopifyPrivacySchemaTx is reserved for the product-only synthetic
// trial. The current Store installer does not provision this table or grant access.
func InstallIsolatedShopifyPrivacySchemaTx(ctx context.Context, tx *sql.Tx) error {
	if tx == nil {
		return errors.New("Shopify privacy receipts require an isolated Store owner transaction")
	}
	_, err := tx.ExecContext(ctx, `CREATE TABLE public.workbench_store_shopify_privacy_receipts (
 app_id VARCHAR(128) NOT NULL, topic VARCHAR(32) NOT NULL, event_digest CHAR(64) NOT NULL,
 shop_id VARCHAR(128), status VARCHAR(24) NOT NULL, received_at TIMESTAMPTZ NOT NULL, due_at TIMESTAMPTZ,
 PRIMARY KEY (app_id,topic,event_digest),
 CHECK (
   (topic IN ('customers/data_request','customers/redact') AND status='no_customer_data' AND shop_id IS NULL AND due_at IS NULL)
   OR (topic='shop/redact' AND status='pending_verification' AND shop_id IS NOT NULL AND due_at IS NOT NULL AND due_at>received_at)
 )
)`)
	return err
}

// ReceiveIsolatedShopifyPrivacy persists only receipt state. A shop notification
// cannot identify an installation generation and therefore authorizes no cleanup.
func (r *MemberScopedStoreRepository) ReceiveIsolatedShopifyPrivacy(ctx context.Context, app, topic, eventID, shop string, received time.Time) error {
	if r == nil || !shopifyAppID.MatchString(app) || !shopifyNoticeID.MatchString(eventID) || received.IsZero() {
		return ErrNotFound
	}
	received = received.UTC()
	status := "no_customer_data"
	var due *time.Time
	switch topic {
	case "customers/data_request", "customers/redact":
		if shop != "" {
			return ErrNotFound
		}
	case "shop/redact":
		if !shopifyShopID.MatchString(shop) {
			return ErrNotFound
		}
		status = "pending_verification"
		deadline := received.Add(shopifyPrivacyDeadline)
		due = &deadline
	default:
		return ErrNotFound
	}
	digest := sha256.Sum256([]byte(app + "\x00" + topic + "\x00" + eventID))
	event := hex.EncodeToString(digest[:])
	db := r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	err := db.Transaction(func(tx *gorm.DB) error {
		insert := tx.Exec(`INSERT INTO public.workbench_store_shopify_privacy_receipts
			(app_id,topic,event_digest,shop_id,status,received_at,due_at)
			VALUES (?,?,?,NULLIF(?,''),?,?,?) ON CONFLICT (app_id,topic,event_digest) DO NOTHING`,
			app, topic, event, shop, status, received, due)
		if insert.Error != nil {
			return insert.Error
		}
		if insert.RowsAffected > 0 || shop == "" {
			return nil
		}
		var prior string
		if err := tx.Raw(`SELECT shop_id FROM public.workbench_store_shopify_privacy_receipts
			WHERE app_id=? AND topic=? AND event_digest=?`, app, topic, event).Row().Scan(&prior); err != nil {
			return err
		}
		if prior != shop {
			return ErrVersionConflict
		}
		return nil
	})
	if err != nil && !errors.Is(err, ErrVersionConflict) {
		return ErrOfficialConnectionUnavailable
	}
	return err
}
