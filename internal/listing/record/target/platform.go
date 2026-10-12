package target

import (
	"context"
	"reflect"
	"task-processor/internal/marketplace/shein/goods"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

// ValidPlatformRecord rejects ambiguous payloads at persistence boundaries.
// The current qualified SHEIN path and the TEMU variant stay explicit.
func ValidPlatformRecord(value TargetRecord) bool {
	if value.Input.Temu == nil {
		return value.Temu == nil && value.TemuRuleSnapshot == nil && len(value.TemuImageObservations) == 0 && value.Input.TemuRules == nil && value.Merchant.Site == "shein-us" && value.Merchant.Platform != storecenter.PlatformTemu
	}
	return value.Temu != nil && value.TemuRuleSnapshot != nil && value.Input.TemuRules == nil &&
		value.Merchant.ValidTemuApplication() && value.Input.Temu.Mode == string(value.Merchant.MerchantMode) &&
		value.Temu.Mode == value.Input.Temu.Mode && value.TemuRuleSnapshot.Mode == value.Input.Temu.Mode &&
		value.TemuRuleSnapshot.Selection == value.Input.Temu.Selection && value.RulesHash == temugoods.RuleHash(*value.TemuRuleSnapshot) &&
		reflect.DeepEqual(value.Input.Draft, goods.OfficialDraftInput{}) && reflect.DeepEqual(value.Result, goods.OfficialDraft{}) &&
		len(value.ImageObservations) == 0 && len(value.ProofObservations) == 0
}

type preparedPlatform struct {
	merchant   storecenter.ProductMerchantBinding
	rulesHash  string
	shein      goods.OfficialDraft
	images     []goods.OfficialImageObservation
	proofs     []goods.OfficialStockProofObservation
	temu       *temugoods.Draft
	temuRules  *temugoods.RuleSnapshot
	temuImages []temugoods.ImageObservation
}

func (s *TargetService) preparePlatform(ctx context.Context, scope collection.Scope, input TargetInput, inventory asset.ApprovedAssetInventory) (preparedPlatform, error) {
	if input.Temu != nil {
		if s.dependencies.TemuRules == nil {
			return preparedPlatform{}, ErrUnavailable
		}
		merchant, snapshot, err := s.dependencies.TemuRules.ReadTemuTargetRules(ctx, scope, input.StoreID, input.Temu.Selection)
		if err != nil {
			return preparedPlatform{}, err
		}
		if merchant.OrganizationID != scope.OrganizationID || merchant.StoreID != input.StoreID || !merchant.ValidTemuApplication() || merchant.StoreVersion < 1 || merchant.ConnectionRevision < 1 || input.Temu.Mode != string(merchant.MerchantMode) || snapshot.Mode != string(merchant.MerchantMode) || !time.Now().Before(merchant.ServiceExpiresAt) || !time.Now().Before(merchant.CredentialExpiresAt) {
			return preparedPlatform{}, ErrNotReady
		}
		images := ProbeTemuTargetImages(ctx, s.dependencies.TemuImages, *input.Temu, inventory)
		result := temugoods.Build(*input.Temu, snapshot, inventory, images)
		return preparedPlatform{merchant: merchant, rulesHash: temugoods.RuleHash(snapshot), temu: &result, temuRules: &snapshot, temuImages: images}, nil
	}
	merchant, rules, err := s.dependencies.Rules.ReadTargetRules(ctx, scope, input.StoreID, input.Draft)
	if err != nil {
		return preparedPlatform{}, err
	}
	if merchant.OrganizationID != scope.OrganizationID || merchant.StoreID != input.StoreID || merchant.Site != "shein-us" || merchant.StoreVersion < 1 || merchant.ConnectionRevision < 1 || !merchant.ValidApplication() || string(rules.ApplicationMode) != string(merchant.ApplicationType) || len(merchant.SupplierIdentityHash) != 64 || !time.Now().Before(merchant.ServiceExpiresAt) {
		return preparedPlatform{}, ErrNotReady
	}
	observations, err := ProbeTargetImages(ctx, s.dependencies.Images, input.Draft, inventory)
	if err != nil {
		return preparedPlatform{}, err
	}
	proofs, err := ProbeTargetStockProofs(ctx, s.dependencies.Images, input.Draft)
	if err != nil {
		return preparedPlatform{}, err
	}
	result := goods.BuildOfficial(input.Draft, rules, inventory, observations, proofs)
	return preparedPlatform{merchant: merchant, rulesHash: collection.Digest(rules), shein: result, images: observations, proofs: proofs}, nil
}
