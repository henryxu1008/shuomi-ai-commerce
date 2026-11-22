package listingapp

import (
	"context"
	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type TargetMerchantReader interface {
	ReadTargetMerchant(context.Context, collection.Scope, string) (storecenter.ProductMerchantBinding, error)
}

type OfficialTargetMerchantStore struct {
	Access *storeapp.OfficialProductAccess
}

func (s OfficialTargetMerchantStore) ReadTargetMerchant(ctx context.Context, scope collection.Scope, storeID string) (storecenter.ProductMerchantBinding, error) {
	if s.Access == nil {
		return storecenter.ProductMerchantBinding{}, record.ErrUnavailable
	}
	return s.Access.ReadTargetMerchantBinding(ctx, storecenter.ProductExecutionSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: storecenter.ProductPurposeRules}, storeID)
}

func validTargetMerchant(scope collection.Scope, storeID string, b storecenter.ProductMerchantBinding) bool {
	return b.OrganizationID == scope.OrganizationID && b.StoreID == storeID && b.ValidApplication() &&
		((b.Platform != storecenter.PlatformTemu && b.Platform != storecenter.PlatformTikTokShop && b.Platform != storecenter.PlatformWalmart && b.Site == "shein-us") || b.ValidTemuApplication() || b.ValidTikTokApplication() || b.ValidWalmartApplication())
}
