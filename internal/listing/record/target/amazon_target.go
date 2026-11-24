package target

import (
	"context"
	"reflect"
	"time"

	amazonmarket "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/amazonus"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shein/goods"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type AmazonTarget struct {
	Draft      amazonmarket.Draft `json:"draft"`
	ValidUntil time.Time          `json:"validUntil"`
	ImageHash  string             `json:"imageHash"`
}

type AmazonTargetPreparer interface {
	PrepareAmazon(context.Context, collection.Scope, string, amazonmarket.DraftInput, catalog.PublishedSnapshot, asset.ApprovedAssetInventory) (storecenter.ProductMerchantBinding, AmazonTarget, error)
}

func validAmazonInput(input TargetInput) bool {
	return input.Amazon != nil && input.LocalTrial == nil && input.ColorPartition == nil &&
		input.SheinImport == nil && input.SheinSalesMapping == nil && input.CompletionDraft == nil &&
		input.ApplyReceiptID == "" && input.Temu == nil && input.TikTok == nil && input.Walmart == nil &&
		input.TemuRules == nil && input.TikTokRules == nil && input.WalmartRules == nil &&
		reflect.DeepEqual(input.Draft, goods.OfficialDraftInput{})
}

func validAmazonTarget(value TargetRecord) bool {
	target := value.Amazon
	return target != nil && validAmazonInput(value.Input) && value.Merchant.ValidAmazonApplication() &&
		value.Merchant.StoreID == value.Input.StoreID && target.Draft.SKU == amazonmarket.SellerSKU(value.Input.StoreID, value.Source.Source.ProductKey) &&
		target.Draft.ProductType == value.Input.Amazon.ProductType && target.Draft.RuleHash == value.RulesHash && target.Draft.Validate() == nil &&
		!target.ValidUntil.IsZero() && !value.CreatedAt.IsZero() && target.ValidUntil.After(value.CreatedAt) &&
		target.ValidUntil.Sub(value.CreatedAt) <= AmazonReviewLifetime &&
		!target.ValidUntil.After(value.Merchant.CredentialExpiresAt) && len(target.ImageHash) == 64 &&
		value.LocalTrial == nil && value.Temu == nil && value.TemuRuleSnapshot == nil &&
		value.TikTok == nil && value.TikTokRuleSnapshot == nil && value.Walmart == nil && value.WalmartRuleSnapshot == nil &&
		len(value.TemuImageObservations)+len(value.TikTokImageObservations)+len(value.WalmartImageObservations)+len(value.ImageObservations)+len(value.ProofObservations) == 0 &&
		reflect.DeepEqual(value.Result, goods.OfficialDraft{})
}

const AmazonReviewLifetime = 15 * time.Minute
