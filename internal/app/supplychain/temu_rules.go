package supplychainapp

import (
	"context"
	"encoding/json"
	"sort"
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

type TemuSupplierSizeMerchant interface {
	SupplierSizeClass(context.Context, int64) (model.SupplierSizeRequirement, error)
	SupplierSizeCharts(context.Context, int64, int64) (model.SupplierSizePage, error)
	SupplierSizeSettings(context.Context, int64, int64) (model.SupplierSizeSettings, error)
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
		if selection.SupplierSiteID != 0 || selection.SupplierCategoryPath != ([10]int64{}) || selection.SupplierSizeOffset != 0 {
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
		if selection.SupplierCategoryPath != ([10]int64{}) {
			parent := int64(0)
			for index, id := range selection.SupplierCategoryPath {
				if id == 0 {
					break
				}
				pathQuery := model.SupplierCategoryQuery{ParentCatID: parent, SiteID: selection.SupplierSiteID}
				categories := snapshot.SupplierCategories
				if pathQuery != query {
					categories, err = merchant.SupplierCategories(ctx, pathQuery)
					if err != nil {
						return binding, temugoods.RuleSnapshot{}, record.ErrUnavailable
					}
					observe("E33", "bg.goods.cats.get", pathQuery, categories)
				}
				var matched *model.SupplierCategory
				for j := range categories {
					c := &categories[j]
					if c.CatID != id {
						continue
					}
					if matched != nil || c.ParentCatID == nil || *c.ParentCatID != parent || c.Level == nil || *c.Level != int64(index+1) || c.Leaf == nil || c.Hidden == nil || *c.Hidden || c.HiddenType == nil || *c.HiddenType != 0 {
						return binding, temugoods.RuleSnapshot{}, record.ErrNotReady
					}
					matched = c
				}
				if matched == nil || *matched.Leaf && id != selection.CatID {
					return binding, temugoods.RuleSnapshot{}, record.ErrNotReady
				}
				snapshot.SupplierCategoryPath = append(snapshot.SupplierCategoryPath, *matched)
				parent = id
			}
			if selection.CatID > 0 && (parent != selection.CatID || !snapshot.SupplierCategoryPath[len(snapshot.SupplierCategoryPath)-1].SelectableLeaf()) || selection.CatID == 0 && parent != selection.ParentCatID {
				return binding, temugoods.RuleSnapshot{}, record.ErrNotReady
			}
		}
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
			sizeMerchant, ok := merchant.(TemuSupplierSizeMerchant)
			if !ok {
				snapshot.Issues = append(snapshot.Issues, "supplier_size_rules_unverified")
			} else {
				sizes, sizeErr := sizeMerchant.SupplierSizeClass(ctx, selection.CatID)
				if sizeErr != nil || !sizes.Observed || sizes.CatID != selection.CatID {
					snapshot.Issues = append(snapshot.Issues, "supplier_size_rules_unverified")
				} else {
					snapshot.SupplierSizeRequirement = &sizes
					observe("E46", "bg.glo.goods.sizecharts.class.get", selection.CatID, sizes)
					if sizes.Class != nil {
						charts, readErr := sizeMerchant.SupplierSizeCharts(ctx, selection.CatID, selection.SupplierSizeOffset)
						if readErr != nil {
							snapshot.Issues = append(snapshot.Issues, "supplier_size_charts_unverified")
						} else {
							snapshot.SupplierSizeCharts = &charts
							observe("E47", "bg.glo.goods.sizecharts.get", struct{ CatID, Offset int64 }{selection.CatID, selection.SupplierSizeOffset}, charts)
							classes := map[int64]bool{}
							for _, chart := range charts.Charts {
								if chart.ClassID != nil && chart.Reusable != nil && !*chart.Reusable {
									allowed := sizes.Class.ClassType != nil && *sizes.Class.ClassType == 0 && sizes.Class.ClassID != nil && *sizes.Class.ClassID == *chart.ClassID
									if sizes.Class.ClassType != nil && *sizes.Class.ClassType == 1 {
										for _, id := range sizes.Class.RelatedClassIDs {
											if id == *chart.ClassID {
												allowed = true
											}
										}
									}
									if allowed {
										classes[*chart.ClassID] = true
									}
								}
							}
							classIDs := make([]int64, 0, len(classes))
							for id := range classes {
								classIDs = append(classIDs, id)
							}
							sort.Slice(classIDs, func(i, j int) bool { return classIDs[i] < classIDs[j] })
							for _, class := range classIDs {
								if len(snapshot.Observations) >= 31 {
									snapshot.Issues = append(snapshot.Issues, "supplier_size_metadata_budget")
									break
								}
								cat := selection.CatID
								if sizes.Class.ClassType != nil && *sizes.Class.ClassType == 1 {
									cat = 0
								}
								settings, readErr := sizeMerchant.SupplierSizeSettings(ctx, cat, class)
								if readErr != nil {
									snapshot.Issues = append(snapshot.Issues, "supplier_size_settings_unverified")
									continue
								}
								snapshot.SupplierSizeSettings = append(snapshot.SupplierSizeSettings, model.SupplierSizeSettingsObservation{ClassID: class, Settings: settings})
								observe("E49", "bg.glo.goods.sizecharts.settings.get", struct{ CatID, ClassID int64 }{cat, class}, settings)
							}
						}
					}
				}
			}
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
	if input.ParentCatID < 0 || input.CatID < 0 || input.CatID > 0 && input.CatID == input.ParentCatID || input.SupplierSiteID < 0 || input.SupplierSizeOffset < 0 || input.SupplierSizeOffset > 1000000 || input.TrademarkPage < 0 || input.TrademarkSize < 0 || input.TrademarkSize > 100 || (input.TrademarkPage == 0) != (input.TrademarkSize == 0) {
		return false
	}
	seen, ended := map[int64]bool{}, false
	for _, id := range input.SupplierCategoryPath {
		if id == 0 {
			ended = true
			continue
		}
		if id < 0 || ended || seen[id] {
			return false
		}
		seen[id] = true
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
