package storecenter

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const shopifyLifecycleTable = "workbench_store_shopify_lifecycle_events"

var (
	shopifyNoticeID            = regexp.MustCompile("^[a-zA-Z0-9_-]{1,128}$")
	ErrShopifyLifecyclePending = fmt.Errorf("Shopify authorization needs verification: %w", ErrNotFound)
)

func shopifyAuthorizationPending(tx *gorm.DB, shop shopifyShopRow) (bool, error) {
	var pending bool
	err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM public.workbench_store_shopify_lifecycle_events
		WHERE organization_id=? AND shop_key=? AND generation=? AND status='pending')`, shop.OrganizationID, shop.ShopKey, shop.Generation).Row().Scan(&pending)
	return pending, err
}

type ShopifyLifecycleNotice struct {
	AppID, ShopID, Topic, EventID, BodyHash string
}

type shopifyLifecycleReceipt struct {
	ShopifyLifecycleNotice          `gorm:"embedded"`
	Revision, Generation, Attempt   int64
	OrganizationID, ShopKey, Status string
	ReceivedAt                      time.Time
	RetryAt                         *time.Time
}

func (shopifyLifecycleReceipt) TableName() string { return shopifyLifecycleTable }

type ShopifyLifecycleCheck struct {
	ShopifyInstallation
	OrganizationID, ShopKey       string
	Revision, Generation, Attempt int64
}

type ShopifyLifecycleDecision string

const (
	ShopifyLifecycleUncertain ShopifyLifecycleDecision = "pending"
	ShopifyLifecycleHealthy   ShopifyLifecycleDecision = "healthy"
	ShopifyLifecycleRevoked   ShopifyLifecycleDecision = "revoked"
)

func (r *MemberScopedStoreRepository) shopifyLifecycleDB(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
}

// The canonical lock also covers the interval before a first Shop row exists.
// A notification never locks a Store or installation proof in reverse order.
func lockShopifyLifecycle(tx *gorm.DB, app, shop string) error {
	if !shopifyAppID.MatchString(app) || !shopifyShopID.MatchString(shop) {
		return ErrNotFound
	}
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", "store-shopify-lifecycle:"+app+":"+shop).Error; err != nil {
		return ErrOfficialConnectionUnavailable
	}
	return nil
}

func shopifyLifecycleRevision(tx *gorm.DB, app, shop string) (int64, error) {
	var revision int64
	err := tx.Model(&shopifyLifecycleReceipt{}).Where("app_id=? AND shop_id=?", app, shop).
		Select("coalesce(max(revision),0)").Scan(&revision).Error
	return revision, err
}

func (r *MemberScopedStoreRepository) ReadShopifyLifecycleRevision(ctx context.Context, installation ShopifyInstallation) (int64, error) {
	if !r.shopifyLifecycle {
		return 0, nil
	}
	if !installation.valid() {
		return 0, ErrNotFound
	}
	return shopifyLifecycleRevision(r.shopifyLifecycleDB(ctx), installation.AppID, installation.ShopID)
}

func (r *MemberScopedStoreRepository) ReceiveShopifyLifecycle(ctx context.Context, notice ShopifyLifecycleNotice, received time.Time) error {
	if !r.shopifyLifecycle || !shopifyNoticeID.MatchString(notice.EventID) || !shopifyDigest.MatchString(notice.BodyHash) ||
		(notice.Topic != "app/uninstalled" && notice.Topic != "app/scopes_update") || received.IsZero() {
		return ErrNotFound
	}
	return r.shopifyLifecycleDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockShopifyLifecycle(tx, notice.AppID, notice.ShopID); err != nil {
			return err
		}
		var prior shopifyLifecycleReceipt
		err := tx.Where("app_id=? AND shop_id=? AND topic=? AND event_id=?", notice.AppID, notice.ShopID, notice.Topic, notice.EventID).Take(&prior).Error
		if err == nil {
			if prior.BodyHash != notice.BodyHash {
				return ErrVersionConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOfficialConnectionUnavailable
		}
		revision, err := shopifyLifecycleRevision(tx, notice.AppID, notice.ShopID)
		if err != nil || revision == math.MaxInt64 {
			return ErrOfficialConnectionUnavailable
		}
		shop, err := findShopifyShop(tx, notice.AppID, notice.ShopID)
		if err != nil {
			return err
		}
		receipt := shopifyLifecycleReceipt{ShopifyLifecycleNotice: notice, Revision: revision + 1, Status: "unbound", ReceivedAt: received.UTC()}
		if shop.ShopKey != "" {
			receipt.OrganizationID, receipt.ShopKey, receipt.Generation = shop.OrganizationID, shop.ShopKey, shop.Generation
			receipt.Status, receipt.RetryAt = "pending", &receipt.ReceivedAt
		}
		if err := tx.Create(&receipt).Error; err != nil {
			return ErrOfficialConnectionUnavailable
		}
		return nil
	})
}

func (r *MemberScopedStoreRepository) ClaimShopifyLifecycle(ctx context.Context, app string, now time.Time) (*ShopifyLifecycleCheck, error) {
	if !r.shopifyLifecycle || !shopifyAppID.MatchString(app) || now.IsZero() {
		return nil, ErrNotFound
	}
	var check *ShopifyLifecycleCheck
	err := r.shopifyLifecycleDB(ctx).Transaction(func(tx *gorm.DB) error {
		var receipt shopifyLifecycleReceipt
		claim := tx.Raw(`WITH due AS (
          SELECT app_id,shop_id,revision FROM public.workbench_store_shopify_lifecycle_events
          WHERE app_id=? AND status='pending' AND retry_at<=? ORDER BY retry_at,shop_id,revision LIMIT 1 FOR UPDATE SKIP LOCKED
        ) UPDATE public.workbench_store_shopify_lifecycle_events AS event SET attempt=event.attempt+1,retry_at=?
          FROM due WHERE event.app_id=due.app_id AND event.shop_id=due.shop_id AND event.revision=due.revision
          RETURNING event.*`, app, now.UTC(), now.Add(time.Minute).UTC()).Scan(&receipt)
		if claim.Error != nil {
			return ErrOfficialConnectionUnavailable
		}
		if claim.RowsAffected == 0 {
			return nil
		}
		var auth shopifyAuthorizationRow
		if err := tx.Where("organization_id=? AND shop_key=? AND generation=?", receipt.OrganizationID, receipt.ShopKey, receipt.Generation).Take(&auth).Error; err != nil {
			return ErrOfficialConnectionUnavailable
		}
		check = &ShopifyLifecycleCheck{
			ShopifyInstallation: ShopifyInstallation{AppID: receipt.AppID, ShopID: receipt.ShopID, InstallationID: auth.InstallationID, Domain: auth.Domain, Currency: auth.Currency},
			OrganizationID:      receipt.OrganizationID, ShopKey: receipt.ShopKey, Revision: receipt.Revision, Generation: receipt.Generation, Attempt: receipt.Attempt,
		}
		return nil
	})
	return check, err
}

func (r *MemberScopedStoreRepository) ResolveShopifyLifecycle(ctx context.Context, check ShopifyLifecycleCheck, decision ShopifyLifecycleDecision, now time.Time) error {
	if !r.shopifyLifecycle || !check.ShopifyInstallation.valid() || check.Revision < 1 || check.Generation < 1 || check.Attempt < 1 || now.IsZero() ||
		(decision != ShopifyLifecycleHealthy && decision != ShopifyLifecycleRevoked && decision != ShopifyLifecycleUncertain) {
		return ErrNotFound
	}
	return r.shopifyLifecycleDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockShopifyLifecycle(tx, check.AppID, check.ShopID); err != nil {
			return err
		}
		shop, err := findShopifyShop(tx, check.AppID, check.ShopID)
		if err != nil {
			return err
		}
		status := string(decision)
		if shop.OrganizationID != check.OrganizationID || shop.ShopKey != check.ShopKey || shop.Generation != check.Generation {
			status = "superseded"
		}
		var retryAt *time.Time
		if status == "pending" {
			retry := now.Add(time.Minute * time.Duration(1<<min(check.Attempt, 6))).UTC()
			retryAt = &retry
		}
		result := tx.Model(&shopifyLifecycleReceipt{}).Where("app_id=? AND shop_id=? AND revision=? AND attempt=? AND generation=? AND shop_key=? AND organization_id=? AND status='pending'",
			check.AppID, check.ShopID, check.Revision, check.Attempt, check.Generation, check.ShopKey, check.OrganizationID).
			Updates(map[string]any{"status": status, "retry_at": retryAt})
		if result.Error != nil {
			return ErrOfficialConnectionUnavailable
		}
		if result.RowsAffected == 0 || status != "revoked" {
			return nil
		}
		auth := tx.Model(&shopifyAuthorizationRow{}).Where("organization_id=? AND shop_key=? AND generation=? AND installation_id=? AND revoked_at IS NULL",
			check.OrganizationID, check.ShopKey, check.Generation, check.InstallationID).
			Update("revoked_at", gorm.Expr("GREATEST(approved_at,?)", now.UTC()))
		if auth.Error != nil {
			return ErrOfficialConnectionUnavailable
		}
		return nil
	})
}

func validateShopifyLifecycleProof(tx *gorm.DB, installation ShopifyInstallation, verifiedRevision int64, created time.Time) error {
	if err := lockShopifyLifecycle(tx, installation.AppID, installation.ShopID); err != nil {
		return err
	}
	revision, err := shopifyLifecycleRevision(tx, installation.AppID, installation.ShopID)
	if err != nil {
		return ErrOfficialConnectionUnavailable
	}
	if revision != verifiedRevision {
		return ErrVersionConflict
	}
	var invalidated bool
	err = tx.Raw(`SELECT EXISTS (SELECT 1 FROM public.workbench_store_shopify_lifecycle_events event
       JOIN public.workbench_store_shopify_authorizations auth ON auth.organization_id=event.organization_id
        AND auth.shop_key=event.shop_key AND auth.generation=event.generation
       WHERE event.app_id=? AND event.shop_id=? AND event.status IN ('pending','revoked')
        AND event.received_at>=? AND auth.installation_id=?)`, installation.AppID, installation.ShopID, created, installation.InstallationID).Row().Scan(&invalidated)
	if err != nil {
		return ErrOfficialConnectionUnavailable
	}
	if invalidated {
		return ErrVersionConflict
	}
	return nil
}
