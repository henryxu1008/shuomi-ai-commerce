package storecenter

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"task-processor/internal/authidentity"
	tiktok "task-processor/internal/storecenter/tiktokshop"
)

type ProductExecutionSubject struct {
	OrganizationID, ActorID, MemberID string
	Purpose                           string
}

const (
	ProductPurposeRules   = "product_rules"
	ProductPurposePublish = "product_publish"
	ProductPurposeImage   = "supply_image_transform"
	ProductPurposeStatus  = "product_status"
	ProductPurposeShelfOn = "product_shelf_on"
)

type ProductExecutionAuthorization struct {
	Access  StoreMemberAccess
	Allowed bool
}

// The application verifies current original membership and the exact purpose;
// this port does not manufacture a request identity for a durable command.
type ProductExecutionAuthorizer interface {
	AuthorizeProductExecution(context.Context, ProductExecutionSubject) (ProductExecutionAuthorization, error)
}
type ProductExecutionMaterial struct {
	StoreVersion     int64
	Platform         Platform
	ServiceExpiresAt time.Time
	Connection       OfficialConnectionView
	Attempt          OfficialConnectionAttempt       `json:"-"`
	TikTok           *TikTokProductExecutionMaterial `json:"-"`
	Walmart          *WalmartGrant                   `json:"-"`
}

type TikTokProductExecutionMaterial struct {
	Grant   tiktok.GrantSnapshot
	Binding TikTokShopBinding
}

type TikTokMerchantBinding struct {
	GrantID         string `json:"grant_id"`
	ShopID          string `json:"shop_id"`
	GrantGeneration int64  `json:"grant_generation"`
	GrantRevision   int64  `json:"grant_revision"`
	BindingRevision int64  `json:"binding_revision"`
}

// ProductMerchantBinding is a public reference to current Store authority. It
// contains no credential and never grants durable execution on its own.
type ProductMerchantBinding struct {
	Walmart              WalmartMerchantBinding    `json:"walmart,omitempty,omitzero"`
	TikTok               TikTokMerchantBinding     `json:"tiktok,omitempty,omitzero"`
	Platform             Platform                  `json:"platform,omitempty"`
	IdentityPartition    OfficialIdentityPartition `json:"identity_partition,omitempty"`
	MerchantMode         OfficialMerchantMode      `json:"merchant_mode,omitempty"`
	RegionID             int64                     `json:"region_id,omitempty"`
	CredentialExpiresAt  time.Time                 `json:"credential_expires_at,omitempty,omitzero"`
	VerificationRevision string                    `json:"verification_revision,omitempty"`
	ScopesHash           string                    `json:"scopes_hash,omitempty"`
	OrganizationID       string                    `json:"organization_id"`
	StoreID              string                    `json:"store_id"`
	Site                 string                    `json:"site"`
	StoreVersion         int64                     `json:"store_version"`
	ConnectionRevision   int64                     `json:"connection_revision"`
	ApplicationRevision  string                    `json:"application_revision"`
	ApplicationID        string                    `json:"application_id"`
	ApplicationType      OfficialApplicationType   `json:"application_type"`
	SupplierIdentityHash string                    `json:"supplier_identity_hash"`
	ServiceExpiresAt     time.Time                 `json:"service_expires_at"`
}

func (b ProductMerchantBinding) ValidApplication() bool {
	if b.Platform == PlatformWalmart {
		return b.ValidWalmartApplication()
	}
	if b.Walmart != (WalmartMerchantBinding{}) {
		return false
	}
	if b.Platform == PlatformTikTokShop {
		return b.ValidTikTokApplication()
	}
	if b.TikTok != (TikTokMerchantBinding{}) {
		return false
	}
	if b.Platform == PlatformTemu {
		return b.ValidTemuApplication()
	}
	return b.ApplicationType.Valid() && authidentity.IsBoundedIdentifier(b.ApplicationID) && authidentity.IsBoundedIdentifier(b.ApplicationRevision) && strings.HasSuffix(b.ApplicationRevision, ":"+string(b.ApplicationType))
}

func (b ProductMerchantBinding) ValidTikTokApplication() bool {
	if b.Walmart != (WalmartMerchantBinding{}) {
		return false
	}
	return b.Platform == PlatformTikTokShop && b.Site == "tiktokshop-us" && b.IdentityPartition == "" && b.ApplicationType == "" && b.MerchantMode == "" && b.RegionID == 0 && authidentity.IsBoundedIdentifier(b.ApplicationID) && authidentity.IsBoundedIdentifier(b.ApplicationRevision) && len(b.SupplierIdentityHash) == 64 && len(b.ScopesHash) == 64 && !b.CredentialExpiresAt.IsZero() && uuid.Validate(b.TikTok.GrantID) == nil && authidentity.IsBoundedIdentifier(b.TikTok.ShopID) && b.TikTok.GrantGeneration > 0 && b.TikTok.GrantRevision >= b.TikTok.GrantGeneration && b.TikTok.BindingRevision > 0
}

func (b ProductMerchantBinding) ValidTemuApplication() bool {
	if b.Walmart != (WalmartMerchantBinding{}) {
		return false
	}
	if b.TikTok != (TikTokMerchantBinding{}) {
		return false
	}
	modes := ModesLocal
	if b.IdentityPartition == PartitionCN {
		if b.MerchantMode != MerchantFullyManaged && b.MerchantMode != MerchantSemiManaged || b.RegionID != 0 || b.Site != "temu-supplier" {
			return false
		}
		modes = ModesFullyManaged | ModesSemiManaged
	} else if b.IdentityPartition != PartitionUS && b.IdentityPartition != PartitionEU && b.IdentityPartition != PartitionGlobal || b.MerchantMode != MerchantLocal || b.RegionID < 1 || b.Site != "temu-local" {
		return false
	}
	suffix := ":temu:" + string(b.IdentityPartition) + ":manual_token:" + strconv.Itoa(int(modes)) + ":r1"
	return b.Platform == PlatformTemu && b.ApplicationType == "" && authidentity.IsBoundedIdentifier(b.ApplicationID) && authidentity.IsBoundedIdentifier(b.ApplicationRevision) && strings.HasSuffix(b.ApplicationRevision, suffix) && authidentity.IsBoundedIdentifier(b.VerificationRevision) && len(b.ScopesHash) == 64 && len(b.SupplierIdentityHash) == 64 && !b.CredentialExpiresAt.IsZero()
}

type ProductExecutionReader interface {
	ReadProductExecution(context.Context, ProductExecutionSubject, string, ProductExecutionAuthorizer, time.Time) (ProductExecutionMaterial, error)
}

func (r *MemberScopedStoreRepository) ReadProductExecution(ctx context.Context, subject ProductExecutionSubject, storeID string, authorization ProductExecutionAuthorizer, now time.Time) (ProductExecutionMaterial, error) {
	if isNilDependency(authorization) || (subject.Purpose != ProductPurposeRules && subject.Purpose != ProductPurposePublish && subject.Purpose != ProductPurposeImage && subject.Purpose != ProductPurposeStatus && subject.Purpose != ProductPurposeShelfOn) {
		return ProductExecutionMaterial{}, ErrNotFound
	}
	material, err := r.readOfficialMaterial(ctx, StoreMemberAccess{OrganizationID: subject.OrganizationID, ActorID: subject.ActorID, MemberID: subject.MemberID}, storeID, now, func(ctx context.Context) (ProductExecutionAuthorization, error) {
		return authorization.AuthorizeProductExecution(ctx, subject)
	})
	if err != nil {
		return ProductExecutionMaterial{}, err
	}
	if material.Platform == PlatformTemu && (len(material.Connection.MerchantIdentityHash) != 64 || material.Connection.CredentialExpiresAt == nil || !now.Before(*material.Connection.CredentialExpiresAt)) {
		return ProductExecutionMaterial{}, ErrNotFound
	}
	return material, nil
}

// Borrow the same current Store transaction and material checks for each
// explicitly admitted purpose. The callback is live, never a durable proof.
func (r *MemberScopedStoreRepository) readOfficialMaterial(ctx context.Context, subject StoreMemberAccess, storeID string, now time.Time, authorize func(context.Context) (ProductExecutionAuthorization, error)) (ProductExecutionMaterial, error) {
	if ctx != nil && ctx.Err() != nil {
		return ProductExecutionMaterial{}, ErrDependencyUnavailable
	}
	if ctx == nil || r == nil || r.db == nil || authorize == nil || now.IsZero() {
		return ProductExecutionMaterial{}, ErrNotFound
	}
	if _, ok := ctx.Deadline(); !ok {
		return ProductExecutionMaterial{}, ErrNotFound
	}
	if _, err := canonicalUUID(storeID); err != nil {
		return ProductExecutionMaterial{}, ErrNotFound
	}
	for _, value := range []string{subject.OrganizationID, subject.ActorID, subject.MemberID} {
		if _, err := validateOpaqueIdentity("execution scope", value, MaxSubjectBytes); err != nil {
			return ProductExecutionMaterial{}, ErrNotFound
		}
	}
	approved, err := authorize(ctx)
	if err != nil {
		return ProductExecutionMaterial{}, productExecutionReadError(err)
	}
	if ctx.Err() != nil {
		return ProductExecutionMaterial{}, ErrDependencyUnavailable
	}
	if !approved.Allowed || approved.Access.OrganizationID != subject.OrganizationID || approved.Access.ActorID != subject.ActorID || approved.Access.MemberID != subject.MemberID {
		return ProductExecutionMaterial{}, ErrNotFound
	}
	var material ProductExecutionMaterial
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockMemberStore(tx, subject.OrganizationID, storeID); err != nil {
			return err
		}
		if err := requireMemberGrant(tx, approved.Access, storeID); err != nil {
			return err
		}
		var stored workbenchStoreRecord
		if err := tx.Where("organization_id = ? AND id = ? AND deleted_at IS NULL", subject.OrganizationID, storeID).Take(&stored).Error; err != nil {
			return err
		}
		if stored.Platform != "shein" && stored.Platform != "temu" && stored.Platform != "tiktokshop" && stored.Platform != "walmart" || stored.RecordStatus != "active" || stored.ServiceStatus == nil || *stored.ServiceStatus != "active" || stored.ServiceStartedAt == nil || stored.ServiceExpiresAt == nil || now.Before(*stored.ServiceStartedAt) || !now.Before(*stored.ServiceExpiresAt) || stored.ConnectionRef == "" {
			return ErrNotFound
		}
		var connection officialConnectionRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ? AND store_id = ? AND attempt_id = ?", subject.OrganizationID, storeID, stored.ConnectionRef).Take(&connection).Error; err != nil {
			return err
		}
		if connection.Status != "connected" {
			return ErrNotFound
		}
		if stored.Platform == "walmart" {
			var attempt WalmartConnectionAttempt
			if err := tx.Where("organization_id=? AND store_id=? AND attempt_id=? AND connection_version=? AND state='bound'", subject.OrganizationID, storeID, connection.AttemptID, connection.Version).Take(&attempt).Error; err != nil {
				return ErrNotFound
			}
			var grant WalmartGrant
			if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("organization_id=? AND store_id=? AND id=?", subject.OrganizationID, storeID, attempt.GrantID).Take(&grant).Error; err != nil {
				return ErrNotFound
			}
			if stored.Region != "US" || grant.AppID != attempt.AppID || grant.AppRevision != attempt.AppRevision || grant.Environment != attempt.Environment || !grant.CanRead(now) {
				return ErrNotFound
			}
			material = ProductExecutionMaterial{StoreVersion: stored.Version, Platform: PlatformWalmart, ServiceExpiresAt: stored.ServiceExpiresAt.UTC(), Connection: walmartConnectionView(connection, attempt, grant, now), Walmart: &grant}
			return ctx.Err()
		}
		if stored.Platform == "tiktokshop" {
			var attempt tiktokAttemptRow
			if err := tx.Where("organization_id=? AND store_id=? AND attempt_id=? AND connection_version=? AND state='bound'", subject.OrganizationID, storeID, connection.AttemptID, connection.Version).Take(&attempt).Error; err != nil {
				return ErrNotFound
			}
			var binding tiktokBindingRow
			if err := tx.Where("organization_id=? AND store_id=? AND status='active'", subject.OrganizationID, storeID).Take(&binding).Error; err != nil {
				return ErrNotFound
			}
			if stored.Region != "US" || binding.ShopCode != stored.ExternalStoreID || binding.AppKey != attempt.AppKey {
				return ErrNotFound
			}
			var row tiktokGrantRow
			if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("organization_id=? AND id=? AND app_key=?", subject.OrganizationID, binding.GrantID, binding.AppKey).Take(&row).Error; err != nil {
				return ErrNotFound
			}
			grant, err := tiktok.RestoreGrant(tiktokGrantValue(row))
			if err != nil || !grant.CanSend(now) {
				return ErrNotFound
			}
			material = ProductExecutionMaterial{StoreVersion: stored.Version, Platform: PlatformTikTokShop, ServiceExpiresAt: stored.ServiceExpiresAt.UTC(), Connection: tiktokConnectionView(connection, attempt, grant.Snapshot(), binding, now), TikTok: &TikTokProductExecutionMaterial{Grant: grant.Snapshot(), Binding: tiktokBindingValue(binding)}}
			return ctx.Err()
		}
		var attempt officialAttemptRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ? AND store_id = ? AND attempt_id = ? AND connection_version = ?", subject.OrganizationID, storeID, connection.AttemptID, connection.Version).Take(&attempt).Error; err != nil {
			return err
		}
		if attempt.State != "verified" || attempt.KeyID == "" || attempt.Ciphertext == "" {
			return ErrNotFound
		}
		material = ProductExecutionMaterial{StoreVersion: stored.Version, Platform: Platform(stored.Platform), ServiceExpiresAt: stored.ServiceExpiresAt.UTC(), Connection: connectionView(connection, attempt), Attempt: attemptValue(attempt)}
		return ctx.Err()
	})
	if err != nil {
		return ProductExecutionMaterial{}, productExecutionReadError(err)
	}
	return material, nil
}

func productExecutionReadError(err error) error {
	if errors.Is(err, ErrNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return ErrDependencyUnavailable
}
