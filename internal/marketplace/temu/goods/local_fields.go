package goods

import (
	"strings"
	"task-processor/internal/marketplace/temu/model"
	"unicode/utf8"
)

// E29: 1 required, 2 optional, 3 disabled. Unknown required fields remain a
// visible gap; optional, absent wire fields do not become invented requirements.
func checkLocalFields(input DraftInput, template model.LocalCategoryRules) []Issue {
	p := input.Local
	issues := []Issue{}
	add := func(field, code string) {
		issues = append(issues, Issue{field, code, "按当前类目的官方字段规则补全资料"})
	}
	present := map[string]bool{
		"goodsName": p.Basic.Name != "", "catId": p.Basic.CatID > 0, "goodsDesc": p.Basic.Description != "",
		"brand": p.Basic.Brand != nil, "bulletPoints": len(p.Basic.BulletPoints) > 0, "externalGoodsId": p.Basic.ExternalGoodsID != "",
		"productType": p.Basic.ProductType != nil, "itemTaxCode": p.Basic.ItemTaxCode != "", "secondHand": p.Basic.SecondHand != nil,
		"shipmentLimitDay": p.ServicePromise.ShipmentLimitDay != nil, "fulfillmentType": p.ServicePromise.FulfillmentType != nil, "costTemplateId": p.ServicePromise.CostTemplateID != "",
		"goodsProperty": len(p.Properties) > 0, "goodsOriginInfo": p.Origin != nil, "goodsSize": p.Size != nil, "skuList": len(p.SKUs) > 0,
		"goodsVehiclePropertyRelation": p.VehiclePropertyRelation != nil, "referenceGoods": p.ReferenceGoods != nil, "certificationInfo": p.Certification != nil,
	}
	for _, slot := range input.Images {
		switch slot.Field {
		case "detail":
			present["detailImage"] = true
			present["goodsGallery"] = true
		case "gallery":
			present["goodsCarouselImage"] = true
			present["goodsGallery"] = true
		case "size":
			present["goodsSizeImage"] = true
			present["goodsSize"] = true
		}
	}
	if p.Basic.Gallery != nil {
		present["detailVideoId"] = p.Basic.Gallery.DetailVideoID != ""
		present["carouselVideoId"] = p.Basic.Gallery.CarouselVideoID != ""
	}
	if p.Origin != nil {
		present["originRegion1"] = p.Origin.Region1 != ""
		present["originRegion2"] = p.Origin.Region2 != ""
		present["manufacturingLocations"] = len(p.Origin.ManufacturingLocations) > 0
	}
	if p.Basic.Brand != nil {
		present["trademarkId"] = p.Basic.Brand.TrademarkID != nil
		present["noTrademark"] = p.Basic.Brand.NoTrademark != nil
	}
	for _, rule := range template.FieldRules {
		if rule.RequirementType == nil {
			add(rule.FieldName, "field_rules_unverified")
			continue
		}
		value, known := present[rule.FieldName]
		switch *rule.RequirementType {
		case 1:
			if !known {
				add(rule.FieldName, "field_rules_unverified")
			} else if !value {
				add(rule.FieldName, "field_required")
			}
		case 2:
		case 3:
			if value {
				add(rule.FieldName, "field_disabled")
			}
		default:
			add(rule.FieldName, "field_rules_unverified")
		}
	}
	if p.Basic.ExternalGoodsID != "" && (utf8.RuneCountInString(p.Basic.ExternalGoodsID) > 40 || strings.TrimSpace(p.Basic.ExternalGoodsID) != p.Basic.ExternalGoodsID) {
		add("externalGoodsId", "external_goods_id_invalid")
	}
	if utf8.RuneCountInString(p.Basic.Description) > 2000 || len(p.Basic.BulletPoints) > 5 {
		add("goodsDesc", "text_length_invalid")
	}
	for _, v := range p.Basic.BulletPoints {
		if !boundedText(v, 200) {
			add("bulletPoints", "text_length_invalid")
		}
	}
	if p.Basic.ProductType != nil && *p.Basic.ProductType != 1 {
		add("productType", "product_type_qualification_unverified")
	}
	if p.ServicePromise.FulfillmentType != nil && *p.ServicePromise.FulfillmentType != 1 {
		add("fulfillmentType", "fulfillment_type_unverified")
	}
	if p.Basic.Gallery != nil && (p.Basic.Gallery.DetailVideoID != "" || p.Basic.Gallery.CarouselVideoID != "") {
		add("goodsGallery", "video_asset_unverified")
	}
	if p.Certification != nil {
		add("certificationInfo", "governance_template_unverified")
	}
	if p.Origin != nil {
		add("goodsOriginInfo", "origin_dictionary_unverified")
	}
	if p.Basic.Brand != nil && p.Basic.Brand.NoTrademark != nil && *p.Basic.Brand.NoTrademark && p.Basic.Brand.TrademarkID != nil {
		add("brand", "brand_conflict")
	}
	return issues
}
