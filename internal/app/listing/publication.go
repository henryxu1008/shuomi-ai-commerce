package listingapp

import (
	"context"
	"errors"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	model "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shein/model"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shopify"
	temugoods "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/temu/goods"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"time"
)

// PublicationView retains confirmed creation facts. A separately timestamped
// TikTokStatus may contain a manual read; neither grants authority to send.
type PublicationView struct {
	SheinStatus          *SheinPublicationStatus          `json:"sheinStatus,omitempty"`
	Shopify              *shopify.ProductObservation      `json:"shopify,omitempty"`
	ShopifyStatus        *shopify.ProductObservation      `json:"shopifyStatus,omitempty"`
	ShopifyAdminURL      string                           `json:"shopifyAdminUrl,omitempty"`
	ShopifyAttemptID     string                           `json:"shopifyAttemptId,omitempty"`
	ShopifyAttemptStatus submission.ExecutionStatus       `json:"shopifyAttemptStatus,omitempty"`
	Walmart              *submission.WalmartFeedReceipt   `json:"walmart,omitempty"`
	WalmartResults       []submission.WalmartResultRecord `json:"walmartResults,omitempty"`
	binding              storecenter.ProductMerchantBinding
	shopifyTarget        submission.ExecutionTarget
	shopifyScope         collection.Scope
	SourceID             string                          `json:"sourceId"`
	StoreID              string                          `json:"storeId"`
	RecordID             string                          `json:"recordId,omitempty"`
	ReceiptID            string                          `json:"receiptId,omitempty"`
	Product              *model.PublishResult            `json:"product,omitempty"`
	TemuProduct          *temugoods.CreatedGoods         `json:"temuProduct,omitempty"`
	TikTok               *submission.TikTokEffectReceipt `json:"tiktok,omitempty"`
	TikTokStatus         *TikTokStatusObservation        `json:"tiktokStatus,omitempty"`
	ObservedAt           *time.Time                      `json:"observedAt,omitempty"`
}

func (a *Application) Publication(ctx context.Context, sourceID, storeID string) (PublicationView, error) {
	value := PublicationView{SourceID: sourceID, StoreID: storeID}
	if ctx == nil || !collection.ValidID(sourceID) || !collection.ValidID(storeID) {
		return value, preparation.ErrInvalid
	}
	if a == nil || a.PublicationReceipts == nil || a.TargetMerchants == nil {
		return value, record.ErrUnavailable
	}
	scope, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return value, err
	}
	proof, err := a.Sources.Select(ctx, sourceID)
	if err != nil {
		return value, err
	}
	owner, source, _, err := proof.Read(ctx)
	if err != nil || owner != scope {
		return value, preparation.ErrForbidden
	}
	binding, err := a.TargetMerchants.ReadTargetMerchant(ctx, scope, storeID)
	if err != nil {
		return value, record.ErrForbidden
	}
	if !validTargetMerchant(scope, storeID, binding) {
		return value, record.ErrConflict
	}
	if binding.ValidShopifyApplication() {
		value.shopifyTarget, err = submission.ShopifyDraftTarget(binding, source.Source.ProductKey)
		if err != nil {
			return value, err
		}
		value.binding, value.shopifyScope = binding, scope
	}
	receipt, err := a.readPublicationReceipt(ctx, scope, source.Source.ProductKey, binding)
	if err != nil || receipt == nil {
		return value, err
	}
	if _, _, _, err = proof.Read(ctx); err != nil {
		return value, err
	}
	current, err := a.TargetMerchants.ReadTargetMerchant(ctx, scope, storeID)
	if err != nil || current != binding {
		return value, record.ErrForbidden
	}
	if receipt.Target.Platform == "shopify" {
		value.Shopify = receipt.Shopify
	} else if receipt.Target.Platform == "walmart" {
		if !submission.WalmartCompatibleBinding(receipt.Binding, binding) || receipt.Walmart == nil {
			return value, record.ErrConflict
		}
		value.Walmart = receipt.Walmart
		if a.WalmartResults != nil {
			value.WalmartResults, err = a.WalmartResults.ReadWalmartResults(ctx, scope, receipt.Target)
			if err != nil {
				return value, err
			}
		}
	} else if receipt.Target.Platform == "tiktokshop" {
		if !submission.TikTokCompatibleBinding(receipt.Binding, binding) {
			return value, record.ErrConflict
		}
		value.TikTok = receipt.TikTok
	} else if receipt.Target.Platform == "temu" {
		value.TemuProduct = receipt.TemuProduct
	} else {
		value.Product = &receipt.Product
	}
	value.ReceiptID = receipt.ID
	value.binding = binding
	value.ObservedAt = &receipt.ObservedAt
	if receipt.Owner == scope {
		value.RecordID = receipt.RecordID
	}
	if value.Walmart != nil {
		return boundWalmartPublication(value)
	}
	return value, nil
}

func (a *Application) readPublicationReceipt(ctx context.Context, scope collection.Scope, productKey string, binding storecenter.ProductMerchantBinding) (*submission.OfficialReceipt, error) {
	storeID := binding.StoreID
	subject, err := submission.ProductSubjectID(productKey, binding.Site)
	platform := "shein"
	if binding.Platform == storecenter.PlatformTemu {
		platform = "temu"
		subject, err = submission.TemuProductSubjectID(productKey, binding)
	}
	if binding.Platform == storecenter.PlatformTikTokShop {
		platform = "tiktokshop"
		subject, err = submission.TikTokProductSubjectID(productKey, submission.TikTokPublishStage)
	}
	if binding.Platform == storecenter.PlatformShopify {
		var target submission.ExecutionTarget
		target, err = submission.ShopifyDraftTarget(binding, productKey)
		platform, storeID, subject = target.Platform, target.StoreID, target.SubjectID
	}
	if binding.Platform == storecenter.PlatformWalmart {
		platform = "walmart"
		subject, err = submission.WalmartSubjectID(productKey, binding.SupplierIdentityHash)
	}
	if err != nil {
		return nil, err
	}
	receipt, err := a.PublicationReceipts.FindOfficialTarget(ctx, scope.OrganizationID, submission.ExecutionTarget{Platform: platform, StoreID: storeID, SubjectID: subject})
	if errors.Is(err, submission.ErrExecutionNotFound) || errors.Is(err, submission.ErrExecutionTargetClaimed) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if submission.ValidateOfficialReceipt(receipt) != nil || receipt.Kind != "publish" || receipt.ProductKey != productKey || receipt.Binding.SupplierIdentityHash != binding.SupplierIdentityHash || (platform != "shopify" && receipt.Binding.StoreID != storeID || platform == "shopify" && receipt.Binding.Shopify.ShopKey != storeID) || receipt.Target.SubjectID != subject {
		return nil, record.ErrConflict
	}
	return &receipt, nil
}
