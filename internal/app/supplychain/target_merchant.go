package supplychainapp

import (
	"context"
	storeapp "task-processor/internal/app/storecenter"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
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
		((b.Platform != storecenter.PlatformTemu && b.Platform != storecenter.PlatformTikTokShop && b.Site == "shein-us") || b.ValidTemuApplication() || b.ValidTikTokApplication())
}
