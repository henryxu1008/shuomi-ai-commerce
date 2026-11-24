package storecenter

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// ShopifyInstallProof belongs to Store Center. Until a person selects a Store,
// it is a browser-bound owner proof without authority in any organization.
type ShopifyInstallProof struct {
	AttemptID, SessionDigest, StateDigest, CodeDigest, State string
	ShopifyInstallation                                      `gorm:"embedded"`
	Generation                                               int64
	OrganizationID, StoreID, ActorID, MemberID               string
	StoreVersion                                             int64
	ShopKey                                                  string
	KeyID, Ciphertext                                        string `json:"-"`
	CreatedAt, ExpiresAt                                     time.Time
}

func (ShopifyInstallProof) TableName() string { return "workbench_store_shopify_installs" }

func (r *MemberScopedStoreRepository) BeginShopifyInstall(ctx context.Context, attempt, app, domain, session, state string, now time.Time) error {
	if _, err := canonicalUUID(attempt); err != nil || !shopifyAppID.MatchString(app) || !shopifyDomain.MatchString(domain) ||
		!shopifyDigest.MatchString(session) || !shopifyDigest.MatchString(state) || now.IsZero() {
		return ErrNotFound
	}
	now = now.UTC().Truncate(time.Microsecond)
	proof := ShopifyInstallProof{AttemptID: attempt, SessionDigest: session, StateDigest: state, State: "created",
		ShopifyInstallation: ShopifyInstallation{AppID: app, Domain: domain}, CreatedAt: now, ExpiresAt: now.Add(shopifyPairingLifetime)}
	return r.shopifyInstallDatabase(ctx).Transaction(func(tx *gorm.DB) error {
		if err := expireShopifyInstallProofs(tx, now); err != nil {
			return err
		}
		insert := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&proof)
		if insert.Error != nil {
			return ErrOfficialConnectionUnavailable
		}
		if insert.RowsAffected != 1 {
			return ErrVersionConflict
		}
		return nil
	})
}

func (r *MemberScopedStoreRepository) shopifyInstallDatabase(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
}

func expireShopifyInstallProofs(tx *gorm.DB, now time.Time) error {
	// Bounded cleanup never waits on a proof being confirmed concurrently.
	result := tx.Exec(`DELETE FROM public.workbench_store_shopify_installs
		WHERE attempt_id IN (SELECT attempt_id FROM public.workbench_store_shopify_installs
			WHERE expires_at<=?
			ORDER BY expires_at LIMIT 128 FOR UPDATE SKIP LOCKED)`, now)
	if result.Error != nil {
		return ErrOfficialConnectionUnavailable
	}
	return nil
}

func (r *MemberScopedStoreRepository) ExpireShopifyInstallProofs(ctx context.Context, now time.Time) error {
	return expireShopifyInstallProofs(r.shopifyInstallDatabase(ctx), now)
}

func lockShopifyInstallProof(tx *gorm.DB, attempt, session string, now time.Time) (ShopifyInstallProof, error) {
	var proof ShopifyInstallProof
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("attempt_id=?", attempt).Take(&proof).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return proof, ErrNotFound
	}
	if err != nil {
		return proof, ErrOfficialConnectionUnavailable
	}
	if subtle.ConstantTimeCompare([]byte(proof.SessionDigest), []byte(session)) != 1 {
		return proof, ErrNotFound
	}
	if now.IsZero() || !now.Before(proof.ExpiresAt) {
		return proof, ErrVersionConflict
	}
	return proof, nil
}

func updateShopifyInstall(tx *gorm.DB, proof ShopifyInstallProof, fields map[string]any) error {
	result := tx.Model(&ShopifyInstallProof{}).Where("attempt_id=? AND state=?", proof.AttemptID, proof.State).Updates(fields)
	if result.Error != nil {
		return ErrOfficialConnectionUnavailable
	}
	if result.RowsAffected != 1 {
		return ErrVersionConflict
	}
	return nil
}

func (r *MemberScopedStoreRepository) ClaimShopifyInstall(ctx context.Context, attempt, app, domain, session, state, codeDigest string, now time.Time) (ShopifyInstallProof, error) {
	if !shopifyDigest.MatchString(codeDigest) {
		return ShopifyInstallProof{}, ErrNotFound
	}
	var found ShopifyInstallProof
	err := r.shopifyInstallDatabase(ctx).Transaction(func(tx *gorm.DB) error {
		proof, err := lockShopifyInstallProof(tx, attempt, session, now)
		if err != nil {
			return err
		}
		if proof.AppID != app || proof.Domain != domain || subtle.ConstantTimeCompare([]byte(proof.StateDigest), []byte(state)) != 1 {
			return ErrNotFound
		}
		if proof.State == "staged" || proof.State == "selected" {
			if subtle.ConstantTimeCompare([]byte(proof.CodeDigest), []byte(codeDigest)) != 1 {
				return ErrVersionConflict
			}
			// Replaying a completed callback only recovers its display domain.
			found = ShopifyInstallProof{State: proof.State, ShopifyInstallation: ShopifyInstallation{Domain: proof.Domain}}
			return nil
		}
		if proof.State != "created" {
			return ErrVersionConflict
		}
		if err := updateShopifyInstall(tx, proof, map[string]any{"state": "exchanging", "code_digest": codeDigest}); err != nil {
			return err
		}
		proof.State, proof.CodeDigest = "exchanging", codeDigest
		found = proof
		return nil
	})
	return found, err
}

func (r *MemberScopedStoreRepository) ShopifyInstallGeneration(ctx context.Context, identity ShopifyInstallation) (int64, error) {
	if !identity.valid() {
		return 0, ErrNotFound
	}
	var generation int64
	err := r.shopifyInstallDatabase(ctx).Transaction(func(tx *gorm.DB) error {
		shop, err := findShopifyShop(tx, identity.AppID, identity.ShopID)
		generation = shop.Generation
		return err
	})
	return generation, err
}

func (r *MemberScopedStoreRepository) StageShopifyInstall(ctx context.Context, claimed ShopifyInstallProof, identity ShopifyInstallation, generation int64, keyID, ciphertext string, now time.Time) (ShopifyInstallProof, error) {
	if !identity.valid() || keyID == "" || len(keyID) > 128 || ciphertext == "" || len(ciphertext) > 16384 {
		return ShopifyInstallProof{}, ErrNotFound
	}
	var saved ShopifyInstallProof
	err := r.shopifyInstallDatabase(ctx).Transaction(func(tx *gorm.DB) error {
		proof, err := lockShopifyInstallProof(tx, claimed.AttemptID, claimed.SessionDigest, now)
		if err != nil {
			return err
		}
		if proof.State != "exchanging" || proof.AppID != identity.AppID || proof.Domain != identity.Domain {
			return ErrVersionConflict
		}
		shop, err := findShopifyShop(tx, identity.AppID, identity.ShopID)
		if err != nil {
			return err
		}
		if shop.Generation != generation {
			return ErrVersionConflict
		}
		if err := updateShopifyInstall(tx, proof, map[string]any{"state": "staged", "shop_id": identity.ShopID, "installation_id": identity.InstallationID,
			"currency": identity.Currency, "generation": generation, "key_id": keyID, "ciphertext": ciphertext}); err != nil {
			return err
		}
		proof.State, proof.ShopifyInstallation, proof.Generation = "staged", identity, generation
		proof.KeyID, proof.Ciphertext = keyID, ciphertext
		saved = proof
		return nil
	})
	return saved, err
}

func (p ShopifyInstallProof) matchesSelection(command OfficialConnectionCommand, access StoreMemberAccess) bool {
	return p.OrganizationID == command.OrganizationID && p.StoreID == command.StoreID && p.StoreVersion == command.ExpectedStoreVersion &&
		p.ActorID == access.ActorID && p.MemberID == access.MemberID
}

type ShopifyInstallSelection struct {
	Domain       string `json:"domain"`
	StoreVersion int64  `json:"storeVersion"`
	Confirmed    bool   `json:"confirmed"`
}

// InspectShopifyInstall exposes only the verified domain and the selected
// version to an authenticated member authorized to edit this exact Store.
func (r *MemberScopedStoreRepository) InspectShopifyInstall(ctx context.Context, key ShopifyPairingKey, app, session string, now time.Time) (ShopifyInstallSelection, error) {
	var view ShopifyInstallSelection
	err := r.shopifyTx(ctx, key.OrganizationID, key.StoreID, true, func(tx *gorm.DB, access StoreMemberAccess, store workbenchStoreRecord) error {
		proof, err := lockShopifyInstallProof(tx, key.AttemptID, session, now)
		if err != nil {
			return err
		}
		if proof.AppID != app {
			return ErrNotFound
		}
		if proof.State == "staged" {
			view = ShopifyInstallSelection{Domain: proof.Domain, StoreVersion: store.Version}
			return nil
		}
		bound := OfficialConnectionCommand{OrganizationID: key.OrganizationID, StoreID: key.StoreID, ExpectedStoreVersion: proof.StoreVersion}
		if !proof.matchesSelection(bound, access) {
			return ErrNotFound
		}
		switch proof.State {
		case "selected":
			view = ShopifyInstallSelection{Domain: proof.Domain, StoreVersion: proof.StoreVersion}
		case "confirmed":
			binding, err := readShopifyBinding(tx, key.OrganizationID, key.StoreID, store.ConnectionRef)
			if err != nil || binding.ShopKey != proof.ShopKey || binding.Generation != proof.Generation || binding.AppID != proof.AppID {
				return ErrVersionConflict
			}
			view = ShopifyInstallSelection{Domain: binding.Domain, StoreVersion: proof.StoreVersion, Confirmed: true}
		default:
			return ErrVersionConflict
		}
		return nil
	})
	return view, err
}

// SelectShopifyInstall fixes the current identity and selected Store before
// the confirmation checks the original online token outside the transaction.
func (r *MemberScopedStoreRepository) SelectShopifyInstall(ctx context.Context, command OfficialConnectionCommand, session string, now time.Time) (ShopifyInstallProof, error) {
	var selected ShopifyInstallProof
	err := r.shopifyTx(ctx, command.OrganizationID, command.StoreID, true, func(tx *gorm.DB, access StoreMemberAccess, store workbenchStoreRecord) error {
		proof, err := lockShopifyInstallProof(tx, command.AttemptID, session, now)
		if err != nil {
			return err
		}
		switch proof.State {
		case "confirmed", "selected":
			if !proof.matchesSelection(command, access) {
				return ErrNotFound
			}
		case "staged":
			if store.Version != command.ExpectedStoreVersion {
				return ErrVersionConflict
			}
			shop, err := findShopifyShop(tx, proof.AppID, proof.ShopID)
			if err != nil {
				return err
			}
			if shop.ShopKey != "" && shop.OrganizationID != command.OrganizationID {
				return ErrAlreadyExists
			}
			if shop.Generation != proof.Generation {
				return ErrVersionConflict
			}
			if err := updateShopifyInstall(tx, proof, map[string]any{"state": "selected", "organization_id": command.OrganizationID,
				"store_id": command.StoreID, "store_version": command.ExpectedStoreVersion, "actor_id": access.ActorID, "member_id": access.MemberID}); err != nil {
				return err
			}
			proof.State, proof.OrganizationID, proof.StoreID, proof.StoreVersion = "selected", command.OrganizationID, command.StoreID, command.ExpectedStoreVersion
			proof.ActorID, proof.MemberID = access.ActorID, access.MemberID
		default:
			return ErrVersionConflict
		}
		selected = proof
		return nil
	})
	return selected, err
}

func (r *MemberScopedStoreRepository) ConfirmShopifyInstall(ctx context.Context, command OfficialConnectionCommand, session string, verified ShopifyInstallation, now time.Time) (ShopifyBinding, error) {
	var result ShopifyBinding
	err := r.shopifyTx(ctx, command.OrganizationID, command.StoreID, true, func(tx *gorm.DB, access StoreMemberAccess, store workbenchStoreRecord) error {
		proof, err := lockShopifyInstallProof(tx, command.AttemptID, session, now)
		if err != nil {
			return err
		}
		if !proof.matchesSelection(command, access) {
			return ErrNotFound
		}
		if proof.State == "confirmed" {
			binding, err := readShopifyBinding(tx, command.OrganizationID, command.StoreID, store.ConnectionRef)
			if err != nil || binding.ShopKey != proof.ShopKey || binding.Generation != proof.Generation || binding.AppID != proof.AppID {
				return ErrVersionConflict
			}
			result = binding
			return nil
		}
		if proof.State != "selected" || store.Version != command.ExpectedStoreVersion || !verified.valid() || proof.ShopifyInstallation != verified {
			return ErrVersionConflict
		}
		shop, err := findShopifyShop(tx, verified.AppID, verified.ShopID)
		if err != nil {
			return err
		}
		if shop.Generation != proof.Generation {
			return ErrVersionConflict
		}
		result, err = bindShopifyAuthorization(tx, access, store, verified, shop, now.UTC().Truncate(time.Microsecond))
		if err != nil {
			return err
		}
		return updateShopifyInstall(tx, proof, map[string]any{"state": "confirmed", "shop_key": result.ShopKey,
			"generation": result.Generation, "domain": "", "shop_id": "", "installation_id": "", "currency": "", "key_id": "", "ciphertext": "", "code_digest": ""})
	})
	if err != nil {
		return ShopifyBinding{}, err
	}
	return result, nil
}
