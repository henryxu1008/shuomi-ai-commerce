package supplychainapp

import (
	"context"
	"time"

	storeapp "task-processor/internal/app/storecenter"
	record "task-processor/internal/listing/record/target"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type TikTokRulesMerchant interface {
	Binding() storecenter.ProductMerchantBinding
	ListingRules(context.Context) (marketplace.ListingRules, error)
	Categories(context.Context, marketplace.ListingMethod) ([]marketplace.Category, error)
	CategoryRules(context.Context, marketplace.RuleSelection) (marketplace.CategoryRules, error)
	Attributes(context.Context, marketplace.RuleSelection) ([]marketplace.AttributeRule, error)
	Warehouses(context.Context) ([]marketplace.Warehouse, error)
	GlobalWarehouses(context.Context) ([]marketplace.GlobalWarehouse, error)
}
type TikTokRuleStore interface {
	TikTokMerchant(context.Context, collection.Scope, string, *storecenter.ProductMerchantBinding) (TikTokRulesMerchant, error)
}
type OfficialTikTokRuleStore struct {
	Access *storeapp.OfficialProductAccess
}

func (s OfficialTikTokRuleStore) TikTokMerchant(ctx context.Context, scope collection.Scope, storeID string, expected *storecenter.ProductMerchantBinding) (TikTokRulesMerchant, error) {
	if s.Access == nil {
		return nil, record.ErrUnavailable
	}
	return s.Access.AuthorizeTikTokRules(ctx, storecenter.ProductExecutionSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: storecenter.ProductPurposeRules}, storeID, expected)
}

type TikTokRuleReader struct{ Stores TikTokRuleStore }

type TikTokCatalog struct {
	Method     marketplace.ListingMethod `json:"method"`
	Listing    marketplace.ListingRules  `json:"listing"`
	Categories []marketplace.Category    `json:"categories"`
}

func (r TikTokRuleReader) ReadCatalog(ctx context.Context, scope collection.Scope, storeID string, method marketplace.ListingMethod) (storecenter.ProductMerchantBinding, TikTokCatalog, error) {
	var binding storecenter.ProductMerchantBinding
	var result TikTokCatalog
	if ctx == nil || scope.Validate() != nil || !collection.ValidID(storeID) || r.Stores == nil || method != "" && method != marketplace.GlobalPublishing && method != marketplace.LocalReplication {
		return binding, result, record.ErrInvalid
	}
	merchant, err := r.Stores.TikTokMerchant(ctx, scope, storeID, nil)
	if err != nil {
		return binding, result, err
	}
	binding = merchant.Binding()
	result.Method = method
	result.Categories = []marketplace.Category{}
	result.Listing, err = merchant.ListingRules(ctx)
	if err != nil {
		return binding, result, err
	}
	if method != "" {
		if result.Listing.RequireMethod(method) != nil {
			return binding, result, record.ErrNotReady
		}
		result.Categories, err = merchant.Categories(ctx, method)
		if err != nil {
			return binding, result, err
		}
	}
	fresh, err := r.Stores.TikTokMerchant(ctx, scope, storeID, &binding)
	if err != nil || fresh.Binding() != binding {
		return binding, result, record.ErrNotReady
	}
	return binding, result, nil
}

func (r TikTokRuleReader) ReadTikTokTargetRules(ctx context.Context, scope collection.Scope, storeID string, selection marketplace.RuleSelection) (storecenter.ProductMerchantBinding, marketplace.RuleSnapshot, error) {
	var binding storecenter.ProductMerchantBinding
	var snapshot marketplace.RuleSnapshot
	if ctx == nil || scope.Validate() != nil || !collection.ValidID(storeID) || !marketplace.ValidSelection(selection) || r.Stores == nil {
		return binding, snapshot, record.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	merchant, err := r.Stores.TikTokMerchant(ctx, scope, storeID, nil)
	if err != nil {
		return binding, snapshot, err
	}
	binding = merchant.Binding()
	if binding.OrganizationID != scope.OrganizationID || binding.StoreID != storeID || !binding.ValidTikTokApplication() {
		return binding, snapshot, record.ErrNotReady
	}
	snapshot.Selection = selection
	snapshot.Listing, err = merchant.ListingRules(ctx)
	if err != nil || snapshot.Listing.RequireMethod(selection.Method) != nil {
		return binding, snapshot, record.ErrNotReady
	}
	categories, err := merchant.Categories(ctx, selection.Method)
	if err != nil {
		return binding, snapshot, err
	}
	for _, category := range categories {
		if category.ID == selection.CategoryID {
			snapshot.Category = category
			break
		}
	}
	if snapshot.Category.ID == "" {
		return binding, snapshot, record.ErrNotReady
	}
	snapshot.Rules, err = merchant.CategoryRules(ctx, selection)
	if err != nil {
		return binding, snapshot, err
	}
	snapshot.Attributes, err = merchant.Attributes(ctx, selection)
	if err != nil {
		return binding, snapshot, err
	}
	snapshot.Warehouses, err = merchant.Warehouses(ctx)
	if err != nil {
		return binding, snapshot, err
	}
	if selection.Method == marketplace.GlobalPublishing {
		snapshot.GlobalWarehouses, err = merchant.GlobalWarehouses(ctx)
		if err != nil {
			return binding, snapshot, err
		}
	}
	// A rule read never renews consent. Recheck the same current material before
	// making the completed snapshot available to the immutable Target builder.
	fresh, err := r.Stores.TikTokMerchant(ctx, scope, storeID, &binding)
	if err != nil || fresh.Binding() != binding {
		return binding, snapshot, record.ErrNotReady
	}
	return binding, snapshot, nil
}
