package storecenter

import (
	"context"
	"crypto/subtle"
	"errors"
	"math"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const shopifyPairingLifetime = 10 * time.Minute

var (
	ErrShopifyOwnerRequired = errors.New("Shopify store owner authorization required")
	shopifyDigest           = regexp.MustCompile(`^[a-f0-9]{64}$`)
	shopifyAppID            = regexp.MustCompile(`^gid://shopify/App/[1-9][0-9]{0,19}$`)
	shopifyShopID           = regexp.MustCompile(`^gid://shopify/Shop/[1-9][0-9]{0,19}$`)
	shopifyInstallID        = regexp.MustCompile(`^gid://shopify/AppInstallation/[1-9][0-9]{0,19}$`)
	shopifyDomain           = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.myshopify\.com$`)
	shopifyCurrencyCode     = regexp.MustCompile(`^[A-Z]{3}$`)
)

type ShopifyInstallation struct {
	AppID, ShopID, InstallationID string
	Domain, Currency              string
}

func (v ShopifyInstallation) valid() bool {
	return shopifyAppID.MatchString(v.AppID) && shopifyShopID.MatchString(v.ShopID) &&
		shopifyInstallID.MatchString(v.InstallationID) && shopifyDomain.MatchString(v.Domain) && shopifyCurrencyCode.MatchString(v.Currency)
}

type ShopifyPairingKey struct{ OrganizationID, StoreID, AttemptID string }

type ShopifyPairing struct {
	OrganizationID, StoreID, AttemptID string
	ActorID, MemberID                  string
	StoreVersion                       int64
	StateDigest, SessionDigest, State  string
	ShopifyInstallation                `gorm:"embedded"`
	Generation                         int64
	KeyID, Ciphertext                  string `json:"-"`
	ShopKey                            *string
	CreatedAt, ExpiresAt               time.Time
}

func (ShopifyPairing) TableName() string { return "workbench_store_shopify_pairings" }
func (p ShopifyPairing) Key() ShopifyPairingKey {
	return ShopifyPairingKey{p.OrganizationID, p.StoreID, p.AttemptID}
}

type shopifyShopRow struct {
	OrganizationID, ShopKey, StoreID, AppID, ShopID string
	Generation                                      int64
}

func (shopifyShopRow) TableName() string { return "workbench_store_shopify_shops" }

type shopifyAuthorizationRow struct {
	OrganizationID, ShopKey, InstallationID, Domain, Currency string
	Generation                                                int64
	ApprovedAt                                                time.Time
	RevokedAt                                                 *time.Time
}

func (shopifyAuthorizationRow) TableName() string { return "workbench_store_shopify_authorizations" }

type ShopifyBinding struct {
	OrganizationID, StoreID, ShopKey string
	ShopifyInstallation
	Generation int64
}

func (r *MemberScopedStoreRepository) shopifyTx(ctx context.Context, org, storeID string, write bool, operation func(*gorm.DB, StoreMemberAccess, workbenchStoreRecord) error) error {
	access, err := r.authorize(ctx, org, write)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Transaction(func(tx *gorm.DB) error {
		store, err := connectionStore(tx, access, storeID)
		if err != nil {
			return err
		}
		if store.Platform != string(PlatformShopify) || store.RecordStatus != string(RecordStatusActive) {
			return ErrNotFound
		}
		return operation(tx, access, store)
	})
}

func (r *MemberScopedStoreRepository) BeginShopifyPairing(ctx context.Context, c OfficialConnectionCommand, appID, session, state string, now time.Time) (ShopifyPairing, error) {
	if !validConnectionCommand(c) || !shopifyAppID.MatchString(appID) || !shopifyDigest.MatchString(session) || !shopifyDigest.MatchString(state) || now.IsZero() {
		return ShopifyPairing{}, ErrNotFound
	}
	now = now.UTC().Truncate(time.Microsecond)
	var pairing ShopifyPairing
	err := r.shopifyTx(ctx, c.OrganizationID, c.StoreID, true, func(tx *gorm.DB, access StoreMemberAccess, store workbenchStoreRecord) error {
		if store.Version != c.ExpectedStoreVersion {
			return ErrVersionConflict
		}
		expired := tx.Model(&ShopifyPairing{}).Where("organization_id=? AND store_id=? AND expires_at<=? AND state IN ('created','exchanging','staged')", c.OrganizationID, c.StoreID, now).
			Updates(map[string]any{"state": "cancelled", "ciphertext": "", "key_id": "", "shop_id": "", "installation_id": "", "domain": "", "currency": "", "generation": 0})
		if expired.Error != nil {
			return ErrOfficialConnectionUnavailable
		}
		pairing = ShopifyPairing{OrganizationID: c.OrganizationID, StoreID: c.StoreID, AttemptID: c.AttemptID,
			ActorID: access.ActorID, MemberID: access.MemberID, StoreVersion: store.Version,
			StateDigest: state, SessionDigest: session, State: "created", ShopifyInstallation: ShopifyInstallation{AppID: appID},
			CreatedAt: now, ExpiresAt: now.Add(shopifyPairingLifetime)}
		insert := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&pairing)
		if insert.Error != nil {
			return ErrOfficialConnectionUnavailable
		}
		if insert.RowsAffected != 1 {
			return ErrAlreadyExists
		}
		return nil
	})
	if err != nil {
		return ShopifyPairing{}, err
	}
	return pairing, nil
}

func lockShopifyPairing(tx *gorm.DB, access StoreMemberAccess, store workbenchStoreRecord, key ShopifyPairingKey, session string, now time.Time) (ShopifyPairing, error) {
	var row ShopifyPairing
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=? AND attempt_id=?", key.OrganizationID, key.StoreID, key.AttemptID).Take(&row).Error
	if err != nil || row.ActorID != access.ActorID || row.MemberID != access.MemberID ||
		subtle.ConstantTimeCompare([]byte(row.SessionDigest), []byte(session)) != 1 {
		return ShopifyPairing{}, ErrNotFound
	}
	if now.IsZero() || !now.Before(row.ExpiresAt) || row.State != "confirmed" && store.Version != row.StoreVersion {
		return ShopifyPairing{}, ErrVersionConflict
	}
	return row, nil
}

func (r *MemberScopedStoreRepository) updateShopifyPairing(ctx context.Context, key ShopifyPairingKey, session string, now time.Time, update func(*gorm.DB, *ShopifyPairing) error) (ShopifyPairing, error) {
	if _, err := canonicalUUID(key.AttemptID); err != nil {
		return ShopifyPairing{}, ErrNotFound
	}
	var row ShopifyPairing
	err := r.shopifyTx(ctx, key.OrganizationID, key.StoreID, true, func(tx *gorm.DB, access StoreMemberAccess, store workbenchStoreRecord) error {
		current, err := lockShopifyPairing(tx, access, store, key, session, now)
		if err != nil {
			return err
		}
		if err := update(tx, &current); err != nil {
			return err
		}
		row = current
		return nil
	})
	return row, err
}

func updateShopifyPairingFields(tx *gorm.DB, row ShopifyPairing, fields map[string]any) error {
	result := tx.Model(&ShopifyPairing{}).Where("organization_id=? AND store_id=? AND attempt_id=? AND state=?", row.OrganizationID, row.StoreID, row.AttemptID, row.State).Updates(fields)
	if result.Error != nil {
		return ErrOfficialConnectionUnavailable
	}
	if result.RowsAffected != 1 {
		return ErrVersionConflict
	}
	return nil
}

func (r *MemberScopedStoreRepository) ClaimShopifyPairing(ctx context.Context, key ShopifyPairingKey, session, state string, now time.Time) (ShopifyPairing, error) {
	return r.updateShopifyPairing(ctx, key, session, now, func(tx *gorm.DB, row *ShopifyPairing) error {
		if subtle.ConstantTimeCompare([]byte(row.StateDigest), []byte(state)) != 1 {
			return ErrNotFound
		}
		if row.State != "created" {
			return ErrVersionConflict
		}
		if err := updateShopifyPairingFields(tx, *row, map[string]any{"state": "exchanging"}); err != nil {
			return err
		}
		row.State = "exchanging"
		return nil
	})
}

func findShopifyShop(tx *gorm.DB, app, shop string) (shopifyShopRow, error) {
	var row shopifyShopRow
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("app_id=? AND shop_id=?", app, shop).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return shopifyShopRow{}, nil
	}
	if err != nil {
		return shopifyShopRow{}, ErrOfficialConnectionUnavailable
	}
	return row, nil
}

func (r *MemberScopedStoreRepository) ShopifyPairingGeneration(ctx context.Context, key ShopifyPairingKey, session string, identity ShopifyInstallation, now time.Time) (int64, error) {
	if !identity.valid() {
		return 0, ErrNotFound
	}
	var generation int64
	_, err := r.updateShopifyPairing(ctx, key, session, now, func(tx *gorm.DB, row *ShopifyPairing) error {
		if row.State != "exchanging" || row.AppID != identity.AppID {
			return ErrVersionConflict
		}
		current, err := findShopifyShop(tx, identity.AppID, identity.ShopID)
		if err != nil {
			return err
		}
		if current.ShopKey != "" && current.OrganizationID != row.OrganizationID {
			return ErrAlreadyExists
		}
		generation = current.Generation
		return nil
	})
	return generation, err
}

func (r *MemberScopedStoreRepository) StageShopifyPairing(ctx context.Context, key ShopifyPairingKey, session string, identity ShopifyInstallation, generation int64, keyID, ciphertext string, now time.Time) (ShopifyPairing, error) {
	if !identity.valid() || keyID == "" || len(keyID) > 128 || ciphertext == "" || len(ciphertext) > 16384 {
		return ShopifyPairing{}, ErrNotFound
	}
	return r.updateShopifyPairing(ctx, key, session, now, func(tx *gorm.DB, row *ShopifyPairing) error {
		if row.State != "exchanging" || row.AppID != identity.AppID {
			return ErrVersionConflict
		}
		shop, err := findShopifyShop(tx, identity.AppID, identity.ShopID)
		if err != nil {
			return err
		}
		if shop.ShopKey != "" && shop.OrganizationID != row.OrganizationID {
			return ErrAlreadyExists
		}
		if shop.Generation != generation {
			return ErrVersionConflict
		}
		if err := updateShopifyPairingFields(tx, *row, map[string]any{"state": "staged", "shop_id": identity.ShopID, "installation_id": identity.InstallationID,
			"domain": identity.Domain, "currency": identity.Currency, "generation": generation, "key_id": keyID, "ciphertext": ciphertext}); err != nil {
			return err
		}
		row.State, row.ShopifyInstallation, row.Generation = "staged", identity, generation
		row.KeyID, row.Ciphertext = keyID, ciphertext
		return nil
	})
}

func (r *MemberScopedStoreRepository) ReadShopifyPairing(ctx context.Context, key ShopifyPairingKey, session string, now time.Time) (ShopifyPairing, error) {
	return r.updateShopifyPairing(ctx, key, session, now, func(*gorm.DB, *ShopifyPairing) error { return nil })
}

func (r *MemberScopedStoreRepository) ConfirmShopifyPairing(ctx context.Context, key ShopifyPairingKey, session string, identity ShopifyInstallation, now time.Time) (ShopifyBinding, error) {
	if !identity.valid() {
		return ShopifyBinding{}, ErrNotFound
	}
	var result ShopifyBinding
	err := r.shopifyTx(ctx, key.OrganizationID, key.StoreID, true, func(tx *gorm.DB, access StoreMemberAccess, store workbenchStoreRecord) error {
		row, err := lockShopifyPairing(tx, access, store, key, session, now)
		if err != nil {
			return err
		}
		shop, err := findShopifyShop(tx, identity.AppID, identity.ShopID)
		if err != nil {
			return err
		}
		if row.State == "confirmed" {
			known, err := currentShopifyBinding(tx, shop, key)
			if err != nil || row.ShopKey == nil || *row.ShopKey != known.ShopKey || row.Generation != known.Generation || known.ShopifyInstallation != identity {
				return ErrVersionConflict
			}
			result = known
			return nil
		}
		if row.State != "staged" || row.ShopifyInstallation != identity || row.Generation != shop.Generation {
			return ErrVersionConflict
		}
		result, err = bindShopifyPairing(tx, access, store, row, shop, now.UTC().Truncate(time.Microsecond))
		return err
	})
	if err != nil {
		return ShopifyBinding{}, err
	}
	return result, nil
}

func bindShopifyPairing(tx *gorm.DB, access StoreMemberAccess, store workbenchStoreRecord, row ShopifyPairing, shop shopifyShopRow, now time.Time) (ShopifyBinding, error) {
	identity := row.ShopifyInstallation
	if shop.ShopKey == "" {
		shop = shopifyShopRow{OrganizationID: row.OrganizationID, ShopKey: uuid.NewString(), StoreID: row.StoreID,
			AppID: identity.AppID, ShopID: identity.ShopID, Generation: 1}
		insert := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&shop)
		if insert.Error != nil || insert.RowsAffected != 1 {
			return ShopifyBinding{}, ErrAlreadyExists
		}
	} else {
		if shop.OrganizationID != row.OrganizationID || shop.Generation == math.MaxInt64 {
			return ShopifyBinding{}, ErrAlreadyExists
		}
		if shop.StoreID != row.StoreID {
			if err := requireMemberGrant(tx, access, shop.StoreID); err != nil {
				return ShopifyBinding{}, err
			}
		}
		if err := tx.Model(&shopifyAuthorizationRow{}).Where("organization_id=? AND shop_key=? AND generation=? AND revoked_at IS NULL",
			shop.OrganizationID, shop.ShopKey, shop.Generation).Update("revoked_at", now).Error; err != nil {
			return ShopifyBinding{}, ErrOfficialConnectionUnavailable
		}
		if err := tx.Model(&shopifyShopRow{}).Where("organization_id=? AND shop_key=? AND generation=?", shop.OrganizationID, shop.ShopKey, shop.Generation).
			Updates(map[string]any{"generation": shop.Generation + 1, "store_id": row.StoreID}).Error; err != nil {
			return ShopifyBinding{}, ErrAlreadyExists
		}
		shop.Generation++
		shop.StoreID = row.StoreID
	}
	consent := shopifyAuthorizationRow{OrganizationID: shop.OrganizationID, ShopKey: shop.ShopKey, Generation: shop.Generation,
		InstallationID: identity.InstallationID, Domain: identity.Domain, Currency: identity.Currency, ApprovedAt: now}
	if err := tx.Create(&consent).Error; err != nil {
		return ShopifyBinding{}, ErrOfficialConnectionUnavailable
	}
	if err := updateShopifyPairingFields(tx, row, map[string]any{"state": "confirmed", "shop_key": shop.ShopKey, "generation": shop.Generation,
		"shop_id": "", "installation_id": "", "domain": "", "currency": "", "key_id": "", "ciphertext": ""}); err != nil {
		return ShopifyBinding{}, err
	}
	if err := updateConnectionRef(tx, store, shop.ShopKey, access.ActorID, now); err != nil {
		return ShopifyBinding{}, err
	}
	return shopifyBindingValue(shop, consent), nil
}

func shopifyBindingValue(shop shopifyShopRow, consent shopifyAuthorizationRow) ShopifyBinding {
	return ShopifyBinding{OrganizationID: shop.OrganizationID, StoreID: shop.StoreID, ShopKey: shop.ShopKey,
		ShopifyInstallation: ShopifyInstallation{AppID: shop.AppID, ShopID: shop.ShopID, InstallationID: consent.InstallationID, Domain: consent.Domain, Currency: consent.Currency},
		Generation:          shop.Generation}
}

func currentShopifyBinding(tx *gorm.DB, shop shopifyShopRow, key ShopifyPairingKey) (ShopifyBinding, error) {
	if shop.ShopKey == "" || shop.OrganizationID != key.OrganizationID || shop.StoreID != key.StoreID {
		return ShopifyBinding{}, ErrNotFound
	}
	var auth shopifyAuthorizationRow
	if err := tx.Where("organization_id=? AND shop_key=? AND generation=? AND revoked_at IS NULL", shop.OrganizationID, shop.ShopKey, shop.Generation).Take(&auth).Error; err != nil {
		return ShopifyBinding{}, ErrNotFound
	}
	return shopifyBindingValue(shop, auth), nil
}

func (r *MemberScopedStoreRepository) ReadShopifyBinding(ctx context.Context, org, storeID string) (ShopifyBinding, error) {
	var result ShopifyBinding
	err := r.shopifyTx(ctx, org, storeID, false, func(tx *gorm.DB, _ StoreMemberAccess, store workbenchStoreRecord) error {
		var shop shopifyShopRow
		if err := tx.Where("organization_id=? AND store_id=? AND shop_key=?", org, storeID, store.ConnectionRef).Take(&shop).Error; err != nil {
			return ErrNotFound
		}
		var err error
		result, err = currentShopifyBinding(tx, shop, ShopifyPairingKey{OrganizationID: org, StoreID: storeID})
		return err
	})
	return result, err
}
