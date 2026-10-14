package target

import (
	"context"
	"reflect"
	"task-processor/internal/marketplace/shein/goods"
	temugoods "task-processor/internal/marketplace/temu/goods"
	tiktok "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

// ValidPlatformRecord rejects ambiguous payloads at persistence boundaries.
// The current qualified SHEIN path and the TEMU variant stay explicit.
func ValidPlatformRecord(value TargetRecord) bool {
	if value.Input.TikTokRules != nil {
		return false
	}
	if value.Input.TikTok != nil {
		return value.TikTok != nil && value.TikTokRuleSnapshot != nil && value.Input.Temu == nil && value.Input.TemuRules == nil && value.Temu == nil && value.TemuRuleSnapshot == nil && len(value.TemuImageObservations) == 0 && value.Merchant.ValidTikTokApplication() && value.TikTokRuleSnapshot.Selection == value.Input.TikTok.Selection && value.RulesHash == tiktok.RuleHash(*value.TikTokRuleSnapshot) && reflect.DeepEqual(value.Input.Draft, goods.OfficialDraftInput{}) && reflect.DeepEqual(value.Result, goods.OfficialDraft{}) && len(value.ImageObservations) == 0 && len(value.ProofObservations) == 0
	}
	if value.TikTok != nil || value.TikTokRuleSnapshot != nil || len(value.TikTokImageObservations) > 0 || value.Merchant.Platform == storecenter.PlatformTikTokShop {
		return false
	}
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

func PublicationReady(value TargetRecord) bool {
	if !ValidPlatformRecord(value) {
		return false
	}
	if value.Merchant.Platform == storecenter.PlatformTikTokShop {
		return value.TikTok.ReadyForUpload
	}
	if value.Merchant.Platform == storecenter.PlatformTemu {
		return value.Temu.ReadyForUpload && len(value.Temu.SubmissionPayload) > 0
	}
	return value.Result.ReadyForUpload
}

type preparedPlatform struct {
	merchant     storecenter.ProductMerchantBinding
	rulesHash    string
	shein        goods.OfficialDraft
	images       []goods.OfficialImageObservation
	proofs       []goods.OfficialStockProofObservation
	temu         *temugoods.Draft
	temuRules    *temugoods.RuleSnapshot
	temuImages   []temugoods.ImageObservation
	tiktok       *tiktok.Draft
	tiktokRules  *tiktok.RuleSnapshot
	tiktokImages []tiktok.ImageObservation
	sheinRules   goods.OfficialRuleSnapshot
}

func (s *TargetService) preparePlatform(ctx context.Context, scope collection.Scope, input TargetInput, inventory asset.ApprovedAssetInventory) (preparedPlatform, error) {
	if input.TikTok != nil {
		if s.dependencies.TikTokRules == nil {
			return preparedPlatform{}, ErrUnavailable
		}
		merchant, snapshot, err := s.dependencies.TikTokRules.ReadTikTokTargetRules(ctx, scope, input.StoreID, input.TikTok.Selection)
		if err != nil {
			return preparedPlatform{}, err
		}
		if merchant.OrganizationID != scope.OrganizationID || merchant.StoreID != input.StoreID || !merchant.ValidTikTokApplication() || merchant.StoreVersion < 1 || merchant.ConnectionRevision < 1 || !time.Now().Before(merchant.ServiceExpiresAt) || !time.Now().Before(merchant.CredentialExpiresAt) {
			return preparedPlatform{}, ErrNotReady
		}
		var observations []tiktok.ImageObservation
		if s.dependencies.TikTokImages != nil {
			for _, required := range tiktok.RequiredImages(*input.TikTok) {
				for _, approved := range inventory.Assets {
					if approved.ID == required.AssetID {
						observed, e := s.dependencies.TikTokImages.ProbeTikTok(ctx, approved, required.UseCase)
						if e == nil && observed.AssetID == approved.ID && observed.UseCase == required.UseCase {
							observations = append(observations, observed)
						}
						break
					}
				}
			}
		}
		result := tiktok.BuildDraft(*input.TikTok, snapshot, inventory, observations)
		return preparedPlatform{merchant: merchant, rulesHash: tiktok.RuleHash(snapshot), tiktok: &result, tiktokRules: &snapshot, tiktokImages: observations}, nil
	}
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
	return preparedPlatform{merchant: merchant, rulesHash: collection.Digest(rules), shein: result, sheinRules: rules, images: observations, proofs: proofs}, nil
}
