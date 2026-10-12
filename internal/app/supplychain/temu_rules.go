package supplychainapp

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	storeapp "task-processor/internal/app/storecenter"
	record "task-processor/internal/listing/record/target"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/marketplace/temu/rules"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type TemuRulesMerchant interface {
	Binding() storecenter.ProductMerchantBinding
	SupplierCategories(context.Context, model.SupplierCategoryQuery) ([]model.SupplierCategory, error)
	LocalCategories(context.Context, model.LocalCategoryQuery) ([]model.LocalCategory, error)
	SupplierAttributes(context.Context, int64) (model.SupplierAttributeTemplate, error)
	LocalCategoryRules(context.Context, model.CategoryRuleQuery) (model.LocalCategoryRules, error)
	LocalAttributes(context.Context, model.LocalAttributeQuery) (model.LocalAttributeTemplate, error)
	LocalFreightTemplates(context.Context) ([]model.LocalFreightTemplate, error)
	LocalTrademarks(context.Context, model.LocalTrademarkQuery) (model.LocalTrademarkPage, error)
	LocalVariations(context.Context, model.CategoryRuleQuery) (model.LocalVariationTemplate, error)
}

type TemuRuleStore interface {
	TemuMerchant(context.Context, collection.Scope, string, *storecenter.ProductMerchantBinding) (TemuRulesMerchant, error)
}

type OfficialTemuRuleStore struct {
	Access *storeapp.OfficialProductAccess
}

func (s OfficialTemuRuleStore) TemuMerchant(ctx context.Context, scope collection.Scope, storeID string, expected *storecenter.ProductMerchantBinding) (TemuRulesMerchant, error) {
	if s.Access == nil {
		return nil, record.ErrUnavailable
	}
	return s.Access.AuthorizeTemuRules(ctx, storecenter.ProductExecutionSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: storecenter.ProductPurposeRules}, storeID, expected)
}

type TemuTargetRuleReader interface {
	ReadTemuTargetRules(context.Context, collection.Scope, string, model.RuleSelection) (storecenter.ProductMerchantBinding, temugoods.RuleSnapshot, error)
}

type TemuRuleReader struct {
	Stores  TemuRuleStore
	Catalog *rules.Catalog
}

func (r TemuRuleReader) ReadTemuTargetRules(ctx context.Context, scope collection.Scope, storeID string, selection model.RuleSelection) (storecenter.ProductMerchantBinding, temugoods.RuleSnapshot, error) {
	var binding storecenter.ProductMerchantBinding
	var snapshot temugoods.RuleSnapshot
	if ctx == nil || scope.Validate() != nil || !collection.ValidID(storeID) || !validTemuRuleSelection(selection) {
		return binding, snapshot, record.ErrInvalid
	}
	if r.Stores == nil || r.Catalog == nil {
		return binding, snapshot, record.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	merchant, err := r.Stores.TemuMerchant(ctx, scope, storeID, nil)
	if err != nil || merchant == nil {
		return binding, snapshot, record.ErrNotReady
	}
	binding = merchant.Binding()
	if !binding.ValidTemuApplication() || binding.OrganizationID != scope.OrganizationID || binding.StoreID != storeID || binding.StoreVersion < 1 || binding.ConnectionRevision < 1 || !time.Now().Before(binding.CredentialExpiresAt) || !time.Now().Before(binding.ServiceExpiresAt) {
		return binding, snapshot, record.ErrNotReady
	}
	snapshot = temugoods.RuleSnapshot{Selection: selection, Mode: string(binding.MerchantMode), SpecificationRevision: r.Catalog.Revision(), SpecificationSources: r.Catalog.Sources(), Observations: []temugoods.RuleObservation{}, Issues: []string{}}
	observe := func(id, api string, input, output any) {
		snapshot.Observations = append(snapshot.Observations, temugoods.RuleObservation{EvidenceID: id, API: api, InputHash: collection.Digest(input), DecodedHash: collection.Digest(output)})
	}
	if binding.MerchantMode == storecenter.MerchantLocal {
		if selection.SupplierSiteID != 0 {
			return binding, temugoods.RuleSnapshot{}, record.ErrInvalid
		}
		target := r.Catalog.LocalTarget(string(binding.IdentityPartition), binding.RegionID)
		snapshot.LocalTarget = &target
		query := model.LocalCategoryQuery{ParentCatID: selection.ParentCatID, Language: selection.Language}
		snapshot.LocalCategories, err = merchant.LocalCategories(ctx, query)
		if err != nil {
			return binding, temugoods.RuleSnapshot{}, record.ErrUnavailable
		}
		observe("E34", "bg.local.goods.cats.get", query, snapshot.LocalCategories)
		for _, category := range snapshot.LocalCategories {
			if category.CatID == selection.CatID && category.ParentID != nil && *category.ParentID == selection.ParentCatID {
				snapshot.SelectableLeaf = category.SelectableLeaf()
			}
		}
		snapshot.FreightTemplates, err = merchant.LocalFreightTemplates(ctx)
		if err != nil {
			return binding, temugoods.RuleSnapshot{}, record.ErrUnavailable
		}
		observe("E37", "bg.freight.template.list.query", struct{}{}, snapshot.FreightTemplates)
		if snapshot.SelectableLeaf {
			query := model.CategoryRuleQuery{CatID: selection.CatID, Language: selection.Language}
			fields, readErr := merchant.LocalCategoryRules(ctx, query)
			if readErr != nil {
				return binding, temugoods.RuleSnapshot{}, record.ErrUnavailable
			}
			snapshot.LocalFieldRules = &fields
			observe("E29", "temu.local.goods.cats.rules.get", query, fields)
			freightVerified := false
			for _, freight := range snapshot.FreightTemplates {
				if freight.ID == selection.CostTemplateID {
					freightVerified = true
				}
			}
			if freightVerified {
				attributesQuery := model.LocalAttributeQuery{CatID: selection.CatID, Language: selection.Language, CostTemplateID: selection.CostTemplateID}
				attributes, readErr := merchant.LocalAttributes(ctx, attributesQuery)
				if readErr != nil {
					return binding, temugoods.RuleSnapshot{}, record.ErrUnavailable
				}
				snapshot.LocalAttributes = &attributes
				observe("E36", "temu.local.product.attributes.get", attributesQuery, attributes)
			} else {
				snapshot.Issues = append(snapshot.Issues, "selected_freight_unverified")
			}
			variations, readErr := merchant.LocalVariations(ctx, query)
			if readErr != nil {
				return binding, temugoods.RuleSnapshot{}, record.ErrUnavailable
			}
			snapshot.LocalVariations = &variations
			observe("E39", "temu.local.product.variation.get", query, variations)
		}
		if selection.TrademarkPage > 0 {
			query := model.LocalTrademarkQuery{Page: selection.TrademarkPage, Size: selection.TrademarkSize}
			brands, readErr := merchant.LocalTrademarks(ctx, query)
			if readErr != nil {
				return binding, temugoods.RuleSnapshot{}, record.ErrUnavailable
			}
			snapshot.Trademarks = &brands
			observe("E38", "temu.local.goods.brand.trademark.V2.get", query, brands)
		}
	} else {
		nameSource, readErr := rules.SupplierNameSource()
		if readErr != nil {
			return binding, temugoods.RuleSnapshot{}, record.ErrUnavailable
		}
		snapshot.SpecificationSources = append(snapshot.SpecificationSources, nameSource)
		snapshot.SpecificationRevision += ";supplier-name-" + nameSource.ContentHash
		if selection.Language != "" || selection.CostTemplateID != "" || selection.TrademarkPage != 0 || binding.MerchantMode == storecenter.MerchantFullyManaged && selection.SupplierSiteID != 0 {
			return binding, temugoods.RuleSnapshot{}, record.ErrInvalid
		}
		snapshot.SemiSites = r.Catalog.SemiSites()
		if selection.SupplierSiteID > 0 {
			site, known := r.Catalog.SemiSite(selection.SupplierSiteID)
			if !known || site.Opening != rules.Open {
				return binding, temugoods.RuleSnapshot{}, record.ErrNotReady
			}
		}
		query := model.SupplierCategoryQuery{ParentCatID: selection.ParentCatID, SiteID: selection.SupplierSiteID}
		snapshot.SupplierCategories, err = merchant.SupplierCategories(ctx, query)
		if err != nil {
			return binding, temugoods.RuleSnapshot{}, record.ErrUnavailable
		}
		observe("E33", "bg.goods.cats.get", query, snapshot.SupplierCategories)
		for _, category := range snapshot.SupplierCategories {
			if category.CatID == selection.CatID && category.ParentCatID != nil && *category.ParentCatID == selection.ParentCatID {
				snapshot.SelectableLeaf = category.SelectableLeaf()
			}
		}
		if snapshot.SelectableLeaf {
			attributes, readErr := merchant.SupplierAttributes(ctx, selection.CatID)
			if readErr != nil {
				return binding, temugoods.RuleSnapshot{}, record.ErrUnavailable
			}
			snapshot.SupplierAttributes = &attributes
			observe("E35", "bg.goods.attrs.get", selection.CatID, attributes)
		}
	}
	// Both provider reads and metadata preparation can outlast the first handle.
	// Revalidate the original exact binding; do not combine a rotated connection.
	current, err := r.Stores.TemuMerchant(ctx, scope, storeID, &binding)
	if err != nil || current == nil || current.Binding() != binding || ctx.Err() != nil {
		return binding, temugoods.RuleSnapshot{}, record.ErrNotReady
	}
	snapshot.ObservedAt = time.Now().UTC()
	raw, err := json.Marshal(snapshot)
	if err != nil || len(raw) > record.MaxPayloadBytes {
		return binding, temugoods.RuleSnapshot{}, record.ErrTooLarge
	}
	return binding, snapshot, nil
}

func validTemuRuleSelection(input model.RuleSelection) bool {
	if input.ParentCatID < 0 || input.CatID < 0 || input.CatID > 0 && input.CatID == input.ParentCatID || input.SupplierSiteID < 0 || input.TrademarkPage < 0 || input.TrademarkSize < 0 || input.TrademarkSize > 100 || (input.TrademarkPage == 0) != (input.TrademarkSize == 0) {
		return false
	}
	for _, value := range []struct {
		text    string
		maximum int
	}{{input.Language, 32}, {input.CostTemplateID, 256}} {
		if len(value.text) > value.maximum || !utf8.ValidString(value.text) || strings.TrimSpace(value.text) != value.text || strings.ContainsAny(value.text, "\r\n\x00") {
			return false
		}
	}
	return true
}

var _ TemuTargetRuleReader = TemuRuleReader{}
