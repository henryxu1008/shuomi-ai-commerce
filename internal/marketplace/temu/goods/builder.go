package goods

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"task-processor/internal/marketplace/temu/rules"
	"task-processor/internal/product/asset"
)

// Build preserves incomplete editable facts. A source snapshot, a selectable
// category or a numerical check never substitutes for the remaining rules,
// approved assets, merchant qualification or the eventual submission consent.
func Build(input DraftInput, snapshot RuleSnapshot, inventory asset.ApprovedAssetInventory, observations []ImageObservation) Draft {
	result := Draft{Mode: input.Mode, Issues: []Issue{}}
	add := func(field, code, message string) { result.Issues = append(result.Issues, Issue{field, code, message}) }
	raw, err := json.Marshal(input)
	var copied DraftInput
	if err != nil || len(raw) > 2<<20 || json.Unmarshal(raw, &copied) != nil {
		add("product", "input_invalid", "商品资料超过限制或包含无效值")
		return result
	}
	input = copied
	result.Local, result.Supplier, result.Images = input.Local, input.Supplier, input.Images
	if snapshot.Mode != input.Mode || input.Mode != "local" && input.Mode != "fully_managed" && input.Mode != "semi_managed" || input.Mode == "local" && (input.Local == nil || input.Supplier != nil) || input.Mode != "local" && (input.Supplier == nil || input.Local != nil) {
		add("mode", "mode_payload_mismatch", "商品资料类型必须与官方核实的店铺模式一致")
		return result
	}
	if snapshot.Selection != input.Selection || snapshot.ObservedAt.IsZero() || snapshot.SpecificationRevision == "" {
		add("rules", "rules_unverified", "重新读取与当前选择一致的官方规则")
	}
	if !snapshot.SelectableLeaf || input.Selection.CatID < 1 {
		add("category", "category_unverified", "选择本店当前可用的末级类目")
	}
	for _, reason := range snapshot.Issues {
		add("rules", reason, "当前官方资料仍缺少明确依据")
	}
	if inventory.Scope.TargetPlatform != "temu" || inventory.Scope.TenantID == "" || inventory.Scope.ProductKey == "" || inventory.Scope.SourceSnapshotVersion == 0 {
		add("images", "asset_scope_unverified", "使用当前商品版本已批准的TEMU图片")
	}
	if containsImageURLs(input) {
		add("images", "unapproved_image_url", "图片必须从当前已批准资产选择")
	}
	if input.Mode == "local" {
		checkLocal(&result, input, snapshot, add)
	} else {
		checkSupplier(&result, input, snapshot, add)
	}
	if len(input.Images) == 0 {
		add("images", "images_missing", "选择商品及SKU需要的已批准图片")
	}
	var categoryType *int64
	if snapshot.LocalFieldRules != nil {
		categoryType = snapshot.LocalFieldRules.ExpandCatType
	}
	wire, imageIssues := resolveImages(input, inventory, observations, categoryType)
	result.Issues = append(result.Issues, imageIssues...)
	result.ReadyForUpload = len(result.Issues) == 0
	if result.ReadyForUpload {
		var payload any = wire.Local
		if wire.Supplier != nil {
			payload = wire.Supplier
		}
		result.SubmissionPayload, err = json.Marshal(payload)
		if err != nil || len(result.SubmissionPayload) > 2<<20 {
			add("product", "payload_invalid", "提交资料超过限制")
			result.ReadyForUpload = false
			result.SubmissionPayload = nil
		}
	}
	return result
}

func checkLocal(result *Draft, input DraftInput, snapshot RuleSnapshot, add func(string, string, string)) {
	p := input.Local
	if !boundedText(p.Basic.Name, 500) {
		add("goodsName", "title_invalid", "填写符合长度要求的商品标题")
	}
	if p.Basic.CatID != input.Selection.CatID {
		add("catId", "category_mismatch", "商品类目必须与当前规则选择一致")
	}
	if p.Basic.SecondHand != nil {
		add("secondHand", "second_hand_contract_unverified", "二手商品编号的官方合同仍有冲突")
	}
	if snapshot.LocalTarget == nil {
		add("site", "local_specifications_missing", "尚未核实本店站点规格")
	} else {
		for _, code := range snapshot.LocalTarget.Issues {
			add("site", string(code), "本店站点规则仍未核实")
		}
		catalog, err := rules.NewOfficialCatalog()
		if err != nil {
			add("site", "local_specifications_missing", "官方规格不可用")
		} else {
			checked := catalog.CheckLocalNumbers(snapshot.LocalTarget.ApplicationPartition, snapshot.LocalTarget.Region.RegionID, p.SKUs, input.PackageUnits)
			for _, issue := range checked.Issues {
				add(issue.Field, issue.Code, "填写与本站币种、精度和单位一致的明确数值")
			}
		}
	}
	if p.ServicePromise.ShipmentLimitDay == nil || *p.ServicePromise.ShipmentLimitDay < 1 {
		add("shipmentLimitDay", "shipment_missing", "填写明确的备货时长")
	}
	freight := false
	for _, value := range snapshot.FreightTemplates {
		if value.ID == p.ServicePromise.CostTemplateID && value.ID == input.Selection.CostTemplateID {
			freight = true
		}
	}
	if !freight {
		add("costTemplateId", "freight_unverified", "选择本店当前有效的运费模板")
	}
	if snapshot.LocalVariations == nil {
		add("specDetails", "variation_rules_unverified", "读取当前规格模板")
	} else {
		for _, issue := range rules.CheckLocalVariations(input.Selection.CatID, *snapshot.LocalVariations, p.SKUs).Issues {
			add(issue.Field, issue.Code, "按当前官方模板补全规格")
		}
	}
	if snapshot.LocalFieldRules == nil {
		add("fields", "field_rules_unverified", "核实当前类目的全部条件性字段要求")
	} else {
		for _, issue := range checkLocalFields(input, *snapshot.LocalFieldRules) {
			add(issue.Field, issue.Code, issue.Message)
		}
	}
	if snapshot.LocalAttributes == nil || snapshot.LocalAttributes.CatID == nil || *snapshot.LocalAttributes.CatID != p.Basic.CatID {
		add("attributes", "attributes_unverified", "读取当前类目及运费模板对应的属性")
	} else {
		// Full attribute/conditional-field coverage is verified by the typed
		// checks below, rather than trusting an empty template as approval.
		for _, issue := range CheckLocalProperties(p.Properties, *snapshot.LocalAttributes) {
			add(issue.Field, issue.Code, issue.Message)
		}
	}
	if p.Basic.Brand != nil && p.Basic.Brand.TrademarkID != nil {
		known := false
		if snapshot.Trademarks != nil {
			for _, value := range snapshot.Trademarks.Trademarks {
				if value.TrademarkID != nil && *value.TrademarkID == *p.Basic.Brand.TrademarkID {
					known = true
				}
			}
		}
		if !known {
			add("trademarkId", "trademark_unverified", "核实本店当前商标资料")
		}
	}
	_ = result
}

func checkSupplier(result *Draft, input DraftInput, snapshot RuleSnapshot, add func(string, string, string)) {
	p := input.Supplier
	result.Issues = append(result.Issues, checkSupplierCategory(*p, snapshot)...)
	// E45 is a weighted first-level-category rule, not a universal 250-rune cap.
	if !boundedText(p.Name, 16<<10) || p.Cat1ID != nil && !rules.SupplierNameAllowed(p.Name, *p.Cat1ID) {
		add("productName", "title_invalid", "填写明确的商品名称")
	}
	nameSource, sourceErr := rules.SupplierNameSource()
	nameVerified := false
	for _, source := range snapshot.SpecificationSources {
		if source == nameSource && sourceErr == nil {
			nameVerified = true
		}
	}
	if !nameVerified {
		add("productName", "supplier_name_rules_unverified", "读取当前货品名称长度规则")
	}
	for _, issue := range rules.CheckSupplierPriceLayout(input.Mode, *p).Issues {
		add(issue.Field, issue.Code, "按店铺模式补全价格和初始数量")
	}
	if input.Mode == "semi_managed" && p.SemiManaged != nil {
		catalog, err := rules.NewOfficialCatalog()
		if err != nil {
			add("sites", "site_rules_unverified", "读取官方站点规则")
		} else {
			for _, code := range catalog.SemiSelection(p.SemiManaged.SiteIDs).Issues {
				add("sites", string(code), "核实所选站点及关联资格")
			}
		}
	}
	if snapshot.SupplierAttributes == nil {
		add("attributes", "attributes_unverified", "读取当前供应商属性模板")
	} else {
		result.Issues = append(result.Issues, CheckSupplierProperties(*p, *snapshot.SupplierAttributes)...)
		result.Issues = append(result.Issues, checkSupplierSpecifications(*p, *snapshot.SupplierAttributes)...)
	}
	result.Issues = append(result.Issues, checkSupplierPackaging(*p)...)
	result.Issues = append(result.Issues, checkSupplierSize(*p, snapshot)...)
	if p.Compliance != nil {
		add("compliance", "supplier_legal_facts_unapproved", "协议资料需要已有批准依据，保存不代表签署")
	}
	if input.Mode == "semi_managed" && (p.Shipment == nil || !boundedText(p.Shipment.FreightTemplateID, 256) || p.Shipment.LimitSecond == nil || *p.Shipment.LimitSecond < 1) {
		add("shipment", "supplier_shipment_missing", "填写本店明确的运费模板与备货时长")
	}
	_ = result
}

func boundedText(value string, maximum int) bool {
	return utf8.ValidString(value) && strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= maximum && !strings.ContainsRune(value, '\x00')
}

func containsImageURLs(input DraftInput) bool {
	if p := input.Local; p != nil {
		if p.Basic.Gallery != nil && (len(p.Basic.Gallery.DetailImages) > 0 || len(p.Basic.Gallery.CarouselImages) > 0) || p.Size != nil && len(p.Size.Images) > 0 {
			return true
		}
		for _, sku := range p.SKUs {
			if len(sku.Images) > 0 {
				return true
			}
		}
	}
	if p := input.Supplier; p != nil {
		if p.MaterialImage != "" || len(p.CarouselImages) > 0 || len(p.OuterImages) > 0 {
			return true
		}
		for _, skc := range p.SKCs {
			if len(skc.PreviewURLs) > 0 || skc.ColorURL != "" {
				return true
			}
			for _, sku := range skc.SKUs {
				if sku.ThumbURL != "" {
					return true
				}
			}
		}
		for _, detail := range p.Details {
			for _, content := range detail.Content {
				if content.ImageURL != "" {
					return true
				}
			}
		}
	}
	return false
}
