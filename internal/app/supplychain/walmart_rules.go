package supplychainapp

import (
	"context"
	storeapp "task-processor/internal/app/storecenter"
	record "task-processor/internal/listing/record/target"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type WalmartRulesMerchant interface {
	Binding() storecenter.ProductMerchantBinding
	Rules(context.Context, marketplace.RuleSelection) (marketplace.RuleSnapshot, error)
	ProductBySKU(context.Context, string) (*marketplace.Product, error)
	FeedPage(context.Context, string, int) (marketplace.FeedObservation, error)
}
type WalmartRuleStore interface {
	WalmartMerchant(context.Context, collection.Scope, string, *storecenter.ProductMerchantBinding) (WalmartRulesMerchant, error)
}
type OfficialWalmartRuleStore struct {
	Access *storeapp.OfficialProductAccess
}

func (s OfficialWalmartRuleStore) WalmartMerchant(ctx context.Context, scope collection.Scope, id string, expected *storecenter.ProductMerchantBinding) (WalmartRulesMerchant, error) {
	if s.Access == nil {
		return nil, record.ErrUnavailable
	}
	return s.Access.AuthorizeWalmartRules(ctx, storecenter.ProductExecutionSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: storecenter.ProductPurposeRules}, id, expected)
}

type WalmartRuleReader struct{ Stores WalmartRuleStore }

func (r WalmartRuleReader) ReadCatalog(ctx context.Context, scope collection.Scope, id, feedType string) (storecenter.ProductMerchantBinding, marketplace.RuleCatalog, error) {
	var b storecenter.ProductMerchantBinding
	var out marketplace.RuleCatalog
	if scope.Validate() != nil || r.Stores == nil || !collection.ValidID(id) {
		return b, out, record.ErrInvalid
	}
	m, e := r.Stores.WalmartMerchant(ctx, scope, id, nil)
	if e != nil {
		return b, out, e
	}
	b = m.Binding()
	reader, ok := m.(interface {
		Catalog(context.Context, string) (marketplace.RuleCatalog, error)
	})
	if !ok {
		return b, out, record.ErrUnavailable
	}
	out, e = reader.Catalog(ctx, feedType)
	if e != nil {
		return b, out, e
	}
	fresh, e := r.Stores.WalmartMerchant(ctx, scope, id, &b)
	if e != nil || fresh.Binding() != b {
		return b, marketplace.RuleCatalog{}, record.ErrNotReady
	}
	return b, out, nil
}

func (r WalmartRuleReader) ReadWalmartTargetRules(ctx context.Context, scope collection.Scope, id string, s marketplace.RuleSelection) (storecenter.ProductMerchantBinding, marketplace.RuleSnapshot, error) {
	var b storecenter.ProductMerchantBinding
	var out marketplace.RuleSnapshot
	if ctx == nil || scope.Validate() != nil || !collection.ValidID(id) || !s.Valid() || r.Stores == nil {
		return b, out, record.ErrInvalid
	}
	m, e := r.Stores.WalmartMerchant(ctx, scope, id, nil)
	if e != nil {
		return b, out, e
	}
	b = m.Binding()
	if b.OrganizationID != scope.OrganizationID || b.StoreID != id || !b.ValidWalmartApplication() {
		return b, out, record.ErrNotReady
	}
	out, e = m.Rules(ctx, s)
	if e != nil {
		return b, out, e
	}
	fresh, e := r.Stores.WalmartMerchant(ctx, scope, id, &b)
	if e != nil || fresh.Binding() != b {
		return b, marketplace.RuleSnapshot{}, record.ErrNotReady
	}
	return b, out, nil
}
