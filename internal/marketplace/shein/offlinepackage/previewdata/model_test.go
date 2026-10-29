package previewdata_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
)

// These field shapes are frozen from the original wire contract before cutover.
func TestPreviewDataPreservesWireFieldsAndZeroValues(t *testing.T) {
	cases := []struct {
		name   string
		value  any
		fields []string
	}{
		{"Extra", previewdata.Extra{}, []string{"BiddingID|string|json:\"bidding_id\"", "BestSaleSPU|string|json:\"best_sale_spu\"", "ActivityType|string|json:\"activity_type\"", "BiddingKey|string|json:\"bidding_key\"", "SubjectSKC|string|json:\"subject_skc\"", "BestSaleAssociateKey|string|json:\"best_sale_associate_key\"", "SwitchToSPUPic|bool|json:\"switch_to_spu_pic\"", "FromPageID|*string|json:\"from_page_id\"", "OptimizeSKCName|*string|json:\"optimize_skc_name\"", "SPUTag|[]string|json:\"spu_tag\"", "TransformCVSizeImage|bool|json:\"transformCvSizeImage\"", "UseCVTransformImage|bool|json:\"useCvTransformImage\"", "ConfirmVolumeSKU|[]string|json:\"confirm_volume_sku\"", "ConfirmWeightSKU|[]string|json:\"confirm_weight_sku\"", "ControlPriceData|map[string]string|json:\"control_price_data\"", "FaddishPriceInfo|struct {}|json:\"faddish_price_info\"", "BestSaleAddColorSize|string|json:\"best_sale_add_color_size\"", "CopySKCConfirmUpdated|bool|json:\"copy_skc_confirm_updated\"", "ExtraClearBiddingOrBestSale|bool|json:\"extra_clear_bidding_or_best_sale\"", "IsSpmbCopyProduct|bool|json:\"is_spmb_copy_product\"", "BiddingSimilarSKC|string|json:\"bidding_similar_skc\"", "ArmorToken|string|json:\"armor_token\"", "BlackBox|string|json:\"black_box\""}},
		{"LanguageContent", previewdata.LanguageContent{}, []string{"Language|string|json:\"language\"", "Name|string|json:\"name\""}},
		{"Product", previewdata.Product{}, []string{"BrandCode|*string|json:\"brand_code\"", "BrandSeriesList|[]string|json:\"brand_series_list\"", "CategoryID|int|json:\"category_id\"", "CategoryIDList|[]int|json:\"category_id_list\"", "CategoryMisplace|bool|json:\"category_misplace\"", "CommodityType|int|json:\"commodity_type\"", "ConfirmSizeImg|bool|json:\"confirm_size_img\"", "InputSizeChartFlag|int|json:\"input_size_chart_flag\"", "IsSuggestCategory|bool|json:\"is_suggest_category\"", "PointKey|string|json:\"point_key\"", "ProductTypeID|*int|json:\"product_type_id\"", "SourceSystem|string|json:\"source_system\"", "SPPRelateSPUName|string|json:\"spp_relate_spu_name\"", "SPUName|string|json:\"spu_name\"", "SuitFlag|int|json:\"suit_flag\"", "SupplierCode|string|json:\"supplier_code\"", "TopCategoryID|int|json:\"top_category_id\"", "Invalid|*string|json:\"invalid\"", "ProductCertificateList|[]int|json:\"product_certificate_list\"", "CertificateList|[]int|json:\"certificate_list\"", "DelOtherCertificateSNList|[]string|json:\"del_other_certificate_sn_list\"", "ImageInfo|*ImageInfo|json:\"image_info,omitempty\"", "MultiLanguageDescList|[]LanguageContent|json:\"multi_language_desc_list\"", "MultiLanguageNameList|[]LanguageContent|json:\"multi_language_name_list\"", "MultiLanguageMakeupIngredientList|[]interface {}|json:\"multi_language_makeup_ingredient_list\"", "ProductAttributeList|[]ProductAttribute|json:\"product_attribute_list\"", "ProductNameAttributeList|[]interface {}|json:\"product_name_attribute_list\"", "ProductVideoList|[]ProductVideo|json:\"product_video_list\"", "PartInfoList|[]interface {}|json:\"part_info_list\"", "PLMPatternIDList|[]interface {}|json:\"plm_pattern_id_list\"", "SiteList|[]SiteInfo|json:\"site_list\"", "PopPriceSiteInfo|*string|json:\"pop_price_site_info\"", "SKCList|[]SKC|json:\"skc_list\"", "SizeAttributeList|[]SizeAttribute|json:\"size_attribute_list\"", "SampleSkuBackSize|*string|json:\"sample_sku_back_size,omitempty\"", "BackSizeAttributeList|[]interface {}|json:\"back_size_attribute_list\"", "CustomAttributeRelation|[]CustomAttributeRelation|json:\"custom_attribute_relation\"", "Extra|Extra|json:\"extra\""}},
		{"ProductAttribute", previewdata.ProductAttribute{}, []string{"AttributeID|int|json:\"attribute_id\"", "AttributeValueID|*int|json:\"attribute_value_id\"", "CVSuggestType|string|json:\"cv_suggest_type\"", "AttributeExtraValue|string|json:\"attribute_extra_value\""}},
		{"ProductVideo", previewdata.ProductVideo{}, []string{"Sites|[]string|json:\"sites\"", "URL|*string|json:\"url\"", "OriginFileURL|string|json:\"origin_file_url\""}},
		{"SiteInfo", previewdata.SiteInfo{}, []string{"MainSite|string|json:\"main_site\"", "SubSiteList|[]string|json:\"sub_site_list\""}},
		{"SizeAttribute", previewdata.SizeAttribute{}, []string{"AttributeID|int|json:\"attribute_id\"", "AttributeExtraValue|string|json:\"attribute_extra_value\"", "RelateSaleAttributeID|int|json:\"relate_sale_attribute_id\"", "RelateSaleAttributeValueID|int|json:\"relate_sale_attribute_value_id\""}},
		{"CostInfo", previewdata.CostInfo{}, []string{"CostPrice|string|json:\"cost_price\"", "Currency|string|json:\"currency\""}},
		{"DetailImage", previewdata.DetailImage{}, []string{"ImageSort|int|json:\"image_sort\"", "ImageURL|string|json:\"image_url\"", "ImageItemID|*string|json:\"image_item_id\""}},
		{"FieldDisabledInfo", previewdata.FieldDisabledInfo{}, []string{"SkuQuantity|*int|json:\"skuQuantity\"", "SkuQuantityType|*int|json:\"skuQuantityType\"", "SkuQuantityUnit|*int|json:\"skuQuantityUnit\""}},
		{"ImageDetail", previewdata.ImageDetail{}, []string{"ImageType|int|json:\"image_type\"", "ImageSort|int|json:\"image_sort\"", "ImageURL|string|json:\"image_url\"", "ImageItemID|*string|json:\"image_item_id\"", "SizeImgFlag|bool|json:\"size_img_flag\"", "TransformCVSizeImage|bool|json:\"transformCvSizeImage\"", "AISStatus|int|json:\"ai_status\"", "PSTypes|[]string|json:\"ps_types\"", "MarketingMainImage|bool|json:\"marketing_main_image\"", "CommodityCategoryFlag|*string|json:\"commodity_category_flag\""}},
		{"ImageInfo", previewdata.ImageInfo{}, []string{"ImageGroupCode|*string|json:\"image_group_code\"", "ImageInfoList|[]ImageDetail|json:\"image_info_list,omitempty\"", "OriginalImageInfoList|*[]interface {}|json:\"original_image_info_list,omitempty\""}},
		{"PriceInfo", previewdata.PriceInfo{}, []string{"SubSite|string|json:\"sub_site\"", "BasePrice|float64|json:\"base_price\"", "SpecialPrice|*float64|json:\"special_price\"", "Currency|string|json:\"currency\""}},
		{"QuantityInfo", previewdata.QuantityInfo{}, []string{"Quantity|*int|json:\"quantity\"", "QuantityType|*int|json:\"quantity_type\"", "QuantityUnit|*int|json:\"quantity_unit\""}},
		{"SKC", previewdata.SKC{}, []string{"FromPreProduct|*string|json:\"from_pre_product\"", "SaleAttribute|SaleAttribute|json:\"sale_attribute\"", "SKUS|[]SKU|json:\"sku_list\"", "ImageInfo|ImageInfo|json:\"image_info,omitempty\"", "IsMaternalStyle|*bool|json:\"is_maternal_style\"", "MultiLanguageName|LanguageContent|json:\"multi_language_name\"", "MultiLanguageNameList|[]LanguageContent|json:\"multi_language_name_list\"", "SiteDetailImageInfoList|[]SiteDetailImageInfo|json:\"site_detail_image_info_list\"", "SppRelateSkcName|*string|json:\"spp_relate_skc_name\"", "HopeOnSaleDate|string|json:\"hope_on_sale_date\"", "ShelfWay|int|json:\"shelf_way\"", "ShelfRequire|int|json:\"shelf_require\"", "ProofOfStockList|[]interface {}|json:\"proof_of_stock_list\"", "Sort|int|json:\"sort\"", "PartCodeRelList|*interface {}|json:\"part_code_rel_list\"", "PriceFillingTips|*string|json:\"price_filling_tips\"", "SKCMaterial|*interface {}|json:\"skc_material\"", "SiteSpecImageInfoList|[]interface {}|json:\"site_spec_image_info_list\"", "SKCScopeAttributeList|[]interface {}|json:\"skc_scope_attribute_list\"", "SampleInfo|*interface {}|json:\"sample_info\"", "IsFirstPublish|bool|json:\"is_first_publish\"", "QualitySN|*string|json:\"quality_sn\"", "ReQuoteFileList|*[]interface {}|json:\"re_quote_file_list\"", "ReQuoteReason|*string|json:\"re_quote_reason\"", "SupplierCode|*string|json:\"supplier_code\"", "Extra|SkcExtra|json:\"extra\""}},
		{"SKU", previewdata.SKU{}, []string{"SaleAttributeList|[]SaleAttribute|json:\"sale_attribute_list\"", "SppRelateSkuCode|*string|json:\"spp_relate_sku_code\"", "CostInfo|*CostInfo|json:\"cost_info\"", "Height|string|json:\"height\"", "Length|string|json:\"length\"", "Width|string|json:\"width\"", "LengthUnit|string|json:\"length_unit\"", "StockInfoList|[]StockInfo|json:\"stock_info_list\"", "Weight|float64|json:\"weight\"", "WeightUnit|string|json:\"weight_unit\"", "StockCount|*int|json:\"stock_count\"", "SupplierSKU|string|json:\"supplier_sku\"", "StopPurchase|int|json:\"stop_purchase\"", "MallState|int|json:\"mall_state\"", "PriceInfoList|[]PriceInfo|json:\"price_info_list\"", "ImageInfo|*ImageInfo|json:\"image_info\"", "CompetingProductLink|*interface {}|json:\"competing_product_link\"", "CompetingCostPriceImages|[]interface {}|json:\"competing_cost_price_images\"", "SuggestedRetailPrice|*interface {}|json:\"suggested_retail_price\"", "QuantityInfo|*QuantityInfo|json:\"quantity_info\"", "PackageType|int|json:\"package_type\"", "Extra|SkuExtra|json:\"extra\""}},
		{"SaleAttribute", previewdata.SaleAttribute{}, []string{"AttributeID|int|json:\"attribute_id\"", "AttributeValueID|int|json:\"attribute_value_id\"", "PreFillSpec|bool|json:\"pre_fill_spec\"", "IsSPPSaleAttribute|bool|json:\"is_spp_sale_attribute\""}},
		{"SiteDetailImageInfo", previewdata.SiteDetailImageInfo{}, []string{"SiteAbbrList|[]string|json:\"site_abbr_list\"", "ImageGroupCode|*string|json:\"image_group_code\"", "ImageInfoList|[]DetailImage|json:\"image_info_list\""}},
		{"SkcExtra", previewdata.SkcExtra{}, []string{"IsSourceEmbryo|bool|json:\"isSourceEmbryo\"", "IsEmbryoInclude|bool|json:\"isEmbryoInclude\"", "PhysicalPriceReviewSuccess|bool|json:\"physical_price_review_success\""}},
		{"SkuExtra", previewdata.SkuExtra{}, []string{"FieldDisabledInfo|FieldDisabledInfo|json:\"fieldDisabledInfo\""}},
		{"StockInfo", previewdata.StockInfo{}, []string{"InventoryNum|int|json:\"inventory_num\"", "MerchantWarehouseCode|string|json:\"merchant_warehouse_code\""}},
		{"CustomAttributeRelation", previewdata.CustomAttributeRelation{}, []string{"PreAttributeValueID|int64|json:\"pre_attribute_value_id\"", "AttributeValueID|int64|json:\"attribute_value_id\""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			typ := reflect.TypeOf(tc.value)
			var got []string
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				shape := field.Type.String()
				for _, prefix := range []string{"previewdata."} {
					shape = strings.ReplaceAll(shape, prefix, "")
				}
				got = append(got, field.Name+"|"+shape+"|"+string(field.Tag))
			}
			if !reflect.DeepEqual(got, tc.fields) {
				t.Fatalf("wire fields/order/tags changed: got %v, want %v", got, tc.fields)
			}
			raw, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			decoded := reflect.New(typ)
			if err := json.Unmarshal(raw, decoded.Interface()); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded.Elem().Interface(), tc.value) {
				t.Fatalf("zero wire roundtrip changed: %s", raw)
			}
		})
	}
}

func TestPreviewDataPreservesJSONPointersNumbersAndEmptySlices(t *testing.T) {
	zero := 0.0
	empty := []any{}
	cases := []struct {
		value any
		want  string
	}{
		{previewdata.CustomAttributeRelation{PreAttributeValueID: 9007199254740993, AttributeValueID: 9007199254740995}, `{"pre_attribute_value_id":9007199254740993,"attribute_value_id":9007199254740995}`},
		{previewdata.PriceInfo{SubSite: "us", BasePrice: 12.5, SpecialPrice: &zero, Currency: "USD"}, `{"sub_site":"us","base_price":12.5,"special_price":0,"currency":"USD"}`},
		{previewdata.ImageInfo{OriginalImageInfoList: &empty}, `{"image_group_code":null,"original_image_info_list":[]}`},
		{previewdata.ProductAttribute{AttributeID: 7}, `{"attribute_id":7,"attribute_value_id":null,"cv_suggest_type":"","attribute_extra_value":""}`},
		{previewdata.SiteInfo{MainSite: "us", SubSiteList: []string{}}, `{"main_site":"us","sub_site_list":[]}`},
		{previewdata.SiteInfo{MainSite: "us"}, `{"main_site":"us","sub_site_list":null}`},
	}
	for _, tc := range cases {
		raw, err := json.Marshal(tc.value)
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) != tc.want {
			t.Fatalf("JSON changed: got %s, want %s", raw, tc.want)
		}
	}
}
