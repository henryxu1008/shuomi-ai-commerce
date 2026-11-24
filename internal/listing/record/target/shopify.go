package target

import (
	"context"
	"reflect"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shein/goods"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type ShopifyTargetStore interface {
	ShopifyTargetBinding(context.Context, collection.Scope, string) (storecenter.ProductMerchantBinding, error)
}

func validShopifyTargetInput(input TargetInput) bool {
	return input.Shopify != nil && input.LocalTrial == nil && input.ColorPartition == nil && input.WalmartRules == nil &&
		input.TemuRules == nil && input.TikTokRules == nil && input.Temu == nil && input.TikTok == nil && input.Walmart == nil &&
		input.CompletionDraft == nil && input.SheinSalesMapping == nil && input.SheinImport == nil &&
		reflect.DeepEqual(input.Draft, goods.OfficialDraftInput{})
}

// An empty Shopify input requests the current store currency only.
func IsShopifyRulesQuery(input TargetInput) bool {
	return validShopifyTargetInput(input) && input.Shopify.Currency == "" && len(input.Shopify.Prices) == 0 && len(input.Shopify.ImageIDs) == 0
}

func validShopifyRecord(value TargetRecord) bool {
	return validShopifyTargetInput(value.Input) && value.Shopify != nil && value.Merchant.ValidShopifyApplication() &&
		value.Merchant.StoreID == value.Input.StoreID && value.Shopify.Currency == value.Merchant.Shopify.Currency &&
		value.Shopify.Title != "" && value.Shopify.SourceKey == "" && len(value.Shopify.Variants) > 0 &&
		value.RulesHash == collection.Digest(value.Merchant.Shopify) && value.LocalTrial == nil &&
		value.Temu == nil && value.TikTok == nil && value.Walmart == nil &&
		value.TemuRuleSnapshot == nil && value.TikTokRuleSnapshot == nil && value.WalmartRuleSnapshot == nil &&
		len(value.TemuImageObservations) == 0 && len(value.TikTokImageObservations) == 0 && len(value.WalmartImageObservations) == 0 &&
		len(value.ImageObservations) == 0 && len(value.ProofObservations) == 0 && reflect.DeepEqual(value.Result, goods.OfficialDraft{})
}
