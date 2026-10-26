package draft

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	"task-processor/internal/listing/record"
	sheinpub "task-processor/internal/marketplace/shein/offlinepackage"
	contract "task-processor/internal/marketplace/validator"
	productasset "task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/catalog/canonical"

	"github.com/stretchr/testify/require"
)

func TestBuilderUsesOnlyControlledExactApprovedAssets(t *testing.T) {
	snapshot := catalog.ProductSnapshot{
		Title:    "Controlled bottle",
		Images:   []catalog.Image{{URL: "https://source.invalid/product.jpg"}},
		Variants: []catalog.Variant{{SKU: "sku-1", Images: []catalog.Image{{URL: "https://source.invalid/variant.jpg"}}}},
	}
	inventory := productasset.ApprovedAssetInventory{
		Scope: productasset.InventoryScope{TenantID: "200", ProductKey: "product", TargetPlatform: "shein", SourceSnapshotVersion: 7},
		Assets: []productasset.ApprovedAsset{
			{ID: "approved-main", RunID: "run", PlanRevision: 1, SlotID: "main", Attempt: 1, Role: productasset.RoleMain, URL: "https://controlled.invalid/main.jpg"},
			{ID: "approved-gallery", RunID: "run", PlanRevision: 1, SlotID: "gallery", Attempt: 1, Role: productasset.RoleGallery, URL: "https://controlled.invalid/gallery.jpg"},
		},
	}
	input := record.Input{ProductKey: "product", SnapshotVersion: 7, StoreID: "11111111-1111-4111-8111-111111111111", Country: "US", Language: "en", Action: contract.SaveDraft}

	raw, err := (Builder{}).Build(context.Background(), snapshot, inventory, input)
	require.NoError(t, err)
	for range 10 {
		again, buildErr := (Builder{}).Build(context.Background(), snapshot, inventory, input)
		require.NoError(t, buildErr)
		require.Equal(t, raw, again, "pure draft mapping must be byte-deterministic")
	}
	var wire map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &wire))
	require.Contains(t, wire, "draft_payload")
	require.Contains(t, wire, "preview_payload")
	for _, key := range []string{"request_draft", "preview_product", "final_draft"} {
		require.NotContains(t, wire, key)
	}
	pkg, err := sheinpub.DecodePersistedPackageStrict(raw)
	require.NoError(t, err)
	require.NotNil(t, pkg.Images)
	require.Equal(t, "https://controlled.invalid/main.jpg", pkg.Images.MainImage)
	require.Contains(t, pkg.Images.Gallery, "https://controlled.invalid/gallery.jpg")
	require.NotContains(t, string(raw), "https://source.invalid/product.jpg")
	require.NotContains(t, string(raw), "https://source.invalid/variant.jpg")
}

func TestBuilderOrdersRawProductAttributesDeterministically(t *testing.T) {
	snapshot := catalog.ProductSnapshot{Title: "Synthetic attributes", Attributes: []catalog.Attribute{
		{Name: "zeta", Value: "  raw z  "}, {Name: "材质", Value: "棉"},
		{Name: "Alpha", Value: "A"}, {Name: " alpha", Value: " leading name "},
		{Name: "alpha", Value: "lower"}, {Name: "Étage", Value: "é"},
	}}
	original := append([]catalog.Attribute(nil), snapshot.Attributes...)
	want := []catalog.Attribute{
		{Name: " alpha", Value: " leading name "}, {Name: "Alpha", Value: "A"},
		{Name: "alpha", Value: "lower"}, {Name: "zeta", Value: "  raw z  "},
		{Name: "Étage", Value: "é"}, {Name: "材质", Value: "棉"},
	}
	attributes := make(map[string]canonical.Attribute, len(original))
	for _, attribute := range original {
		attributes[attribute.Name] = canonical.Attribute{Value: attribute.Value}
	}
	for i, attribute := range buildSourceAttributes(attributes) {
		require.Equal(t, want[i].Name, attribute.Name)
		require.Equal(t, want[i].Value, attribute.Value, "source assembly must not normalize raw values")
	}
	for _, attribute := range original {
		require.Equal(t, attribute.Value, attributes[attribute.Name].Value, "source assembly must not mutate its map")
	}
	require.Nil(t, buildSourceAttributes(nil))
	require.Nil(t, buildSourceAttributes(map[string]canonical.Attribute{}))
	// The existing draft cleanup trims values after source assembly; keep that behavior.
	wantDraftValues := []string{"leading name", "A", "lower", "raw z", "é", "棉"}
	var first []byte
	for range 32 {
		raw, err := (Builder{}).Build(context.Background(), snapshot, productasset.ApprovedAssetInventory{}, record.Input{})
		require.NoError(t, err)
		pkg, err := sheinpub.DecodePersistedPackageStrict(raw)
		require.NoError(t, err)
		require.Len(t, pkg.ProductAttributes, len(want))
		for i, attribute := range want {
			require.Equal(t, attribute.Name, pkg.ProductAttributes[i].Name, "original names must use Go string order")
			require.Equal(t, wantDraftValues[i], pkg.ProductAttributes[i].Value, "existing draft cleanup must remain unchanged")
		}
		require.Equal(t, pkg.ProductAttributes, pkg.DraftPayload.ProductAttributeList)
		if first == nil {
			first = raw
		}
		require.Equal(t, first, raw, "multi-attribute draft bytes must be deterministic")
		require.Equal(t, original, snapshot.Attributes, "building must not mutate source attributes")
	}
}

// Frozen before the owner switch, using only synthetic catalog/approved assets.
// Fixed bytes and hashes are deliberately independent of the implementation.
func TestBuilderFrozenLocalPackageBytes(t *testing.T) {
	cases := []struct {
		name              string
		snapshot          catalog.ProductSnapshot
		country, language string
		assets            []productasset.ApprovedAsset
		want, hash        string
	}{
		{name: "empty", snapshot: catalog.ProductSnapshot{}, want: `{"title_diagnostics":{"source":"structured_fallback"},"site_list":[{"main_site":"US","sub_sites":["US"]}],"skc_list":[{"skc_name":"DEFAULT-001","sale_name":"DEFAULT-001","supplier_code":"DEFAULT-001","skus":[{"sku":"DEFAULT-001","is_default":true}]}],"draft_payload":{"supplier_code":"DEFAULT-001","multi_language_name_list":[{"language":"en_US"},{"language":"en"}],"multi_language_desc_list":[{"language":"en_US"},{"language":"en"}],"site_list":[{"main_site":"US","sub_sites":["US"]}],"skc_list":[{"skc_name":"DEFAULT-001","sale_name":"DEFAULT-001","supplier_code":"DEFAULT-001","sort":1,"multi_language_name_list":[{"language":"en","name":"DEFAULT-001"}],"image_info":{},"sku_list":[{"supplier_sku":"DEFAULT-001","is_default":true,"stock_info_list":[{"warehouse_code":"DEFAULT"}]}]}]},"preview_payload":{"brand_code":null,"brand_series_list":null,"category_id":0,"category_id_list":null,"category_misplace":false,"commodity_type":0,"confirm_size_img":false,"input_size_chart_flag":0,"is_suggest_category":false,"point_key":"","product_type_id":null,"source_system":"listingkit","spp_relate_spu_name":"","spu_name":"","suit_flag":0,"supplier_code":"DEFAULT-001","top_category_id":0,"invalid":null,"product_certificate_list":[],"certificate_list":[],"del_other_certificate_sn_list":null,"multi_language_desc_list":[],"multi_language_name_list":[],"multi_language_makeup_ingredient_list":null,"product_attribute_list":null,"product_name_attribute_list":null,"product_video_list":null,"part_info_list":null,"plm_pattern_id_list":null,"site_list":[{"main_site":"US","sub_site_list":["US"]}],"pop_price_site_info":null,"skc_list":[{"from_pre_product":null,"sale_attribute":{"attribute_id":0,"attribute_value_id":0,"pre_fill_spec":true,"is_spp_sale_attribute":false},"sku_list":[{"sale_attribute_list":null,"spp_relate_sku_code":null,"cost_info":{"cost_price":"","currency":""},"height":"","length":"","width":"","length_unit":"","stock_info_list":[{"inventory_num":0,"merchant_warehouse_code":"DEFAULT"}],"weight":0,"weight_unit":"","stock_count":0,"supplier_sku":"DEFAULT-001","stop_purchase":0,"mall_state":1,"price_info_list":null,"image_info":null,"competing_product_link":null,"competing_cost_price_images":null,"suggested_retail_price":null,"quantity_info":null,"package_type":3,"extra":{"fieldDisabledInfo":{"skuQuantity":null,"skuQuantityType":null,"skuQuantityUnit":null}}}],"image_info":{"image_group_code":null},"is_maternal_style":null,"multi_language_name":{"language":"en","name":"DEFAULT-001"},"multi_language_name_list":[{"language":"en","name":"DEFAULT-001"}],"site_detail_image_info_list":null,"spp_relate_skc_name":null,"hope_on_sale_date":"","shelf_way":0,"shelf_require":0,"proof_of_stock_list":null,"sort":1,"part_code_rel_list":null,"price_filling_tips":null,"skc_material":null,"site_spec_image_info_list":null,"skc_scope_attribute_list":null,"sample_info":null,"is_first_publish":true,"quality_sn":null,"re_quote_file_list":null,"re_quote_reason":null,"supplier_code":"DEFAULT-001","extra":{"isSourceEmbryo":false,"isEmbryoInclude":false,"physical_price_review_success":false}}],"size_attribute_list":null,"back_size_attribute_list":null,"custom_attribute_relation":null,"extra":{"bidding_id":"","best_sale_spu":"","activity_type":"","bidding_key":"","subject_skc":"","best_sale_associate_key":"","switch_to_spu_pic":true,"from_page_id":null,"optimize_skc_name":null,"spu_tag":null,"transformCvSizeImage":false,"useCvTransformImage":true,"confirm_volume_sku":null,"confirm_weight_sku":null,"control_price_data":null,"faddish_price_info":{},"best_sale_add_color_size":"","copy_skc_confirm_updated":false,"extra_clear_bidding_or_best_sale":false,"is_spmb_copy_product":false,"bidding_similar_skc":"","armor_token":"","black_box":""}},"metadata":{"authorized_brand_code":"","authorized_brand_name":"","category_name":"","country":"","english_description_generation":"failed","language":"","source_category_path":"","target_platform":"shein","title_source":"structured_fallback"},"review_notes":["英文描述生成失败，无法进入最终提交"]}`, hash: "sha256:ddb9a6b12a8f8c24b35ab235fa3e54611a69bf4c9e8a054b89a9a8e34425ad0d"},
		{name: "approved_variants", country: "US", language: "en", snapshot: catalog.ProductSnapshot{
			Title: "Controlled bottle", Brand: "Synthetic", Description: "A reusable bottle.", CategoryPath: []string{"Home", "Bottles"},
			Images: []catalog.Image{{URL: "https://source.invalid/unapproved.jpg"}},
			Variants: []catalog.Variant{
				{SKU: "sku-1", Attributes: []catalog.Attribute{{Name: "Color", Value: "Blue"}}, Price: &catalog.Price{Amount: 12.5, CostPrice: 6.25}, Stock: 7, Images: []catalog.Image{{URL: "https://source.invalid/variant.jpg"}}},
				{SKU: "sku-2", Attributes: []catalog.Attribute{{Name: "Color", Value: "Red"}}, Price: &catalog.Price{Currency: "EUR", Amount: 18}, Stock: 2},
			},
		}, assets: []productasset.ApprovedAsset{
			{ID: "main", Role: productasset.RoleMain, URL: "https://controlled.invalid/main.jpg"},
			{ID: "gallery", Role: productasset.RoleGallery, URL: "https://controlled.invalid/gallery.jpg"},
		}, want: `{"spu_name":"Controlled bottle","brand_name":"Synthetic","product_name_en":"Controlled bottle","product_name_multi":"Controlled bottle","title_diagnostics":{"source":"canonical_title","skc_base_title":"Controlled bottle"},"category_name":"Bottles","category_path":["Home","Bottles"],"description":"A reusable bottle.","site_list":[{"main_site":"US","sub_sites":["US"]}],"skc_list":[{"skc_name":"sku-1","sale_name":"sku-1","supplier_code":"sku-1","main_image_url":"https://controlled.invalid/main.jpg","skus":[{"sku":"sku-1","attributes":{"Color":"Blue"},"price":{"amount":12.5,"cost_price":6.25},"stock":7},{"sku":"sku-2","attributes":{"Color":"Red"},"price":{"currency":"EUR","amount":18},"stock":2}]}],"images":{"main_image":"https://controlled.invalid/main.jpg","gallery":["https://controlled.invalid/gallery.jpg"]},"draft_payload":{"spu_name":"Controlled bottle","supplier_code":"sku-1","multi_language_name_list":[{"language":"en","name":"Controlled bottle"},{"language":"en","name":"Controlled bottle"}],"multi_language_desc_list":[{"language":"en","name":"A reusable bottle."},{"language":"en","name":"A reusable bottle."}],"image_info":{"main_image":"https://controlled.invalid/main.jpg","gallery":["https://controlled.invalid/gallery.jpg"]},"site_list":[{"main_site":"US","sub_sites":["US"]}],"skc_list":[{"skc_name":"sku-1","sale_name":"sku-1","supplier_code":"sku-1","sort":1,"multi_language_name_list":[{"language":"en","name":"sku-1"}],"image_info":{"main_image":"https://controlled.invalid/main.jpg","gallery":["https://controlled.invalid/gallery.jpg"]},"sku_list":[{"supplier_sku":"sku-1","attributes":{"Color":"Blue"},"currency":"USD","cost_price":"6.25","base_price":"12.5","stock_count":7,"main_image":"https://controlled.invalid/main.jpg","site_price_list":[{"sub_site":"US","base_price":"12.5","currency":"USD"}],"stock_info_list":[{"warehouse_code":"DEFAULT","inventory_num":7}]},{"supplier_sku":"sku-2","attributes":{"Color":"Red"},"currency":"EUR","base_price":"18","stock_count":2,"main_image":"https://controlled.invalid/main.jpg","site_price_list":[{"sub_site":"US","base_price":"18","currency":"EUR"}],"stock_info_list":[{"warehouse_code":"DEFAULT","inventory_num":2}]}]}]},"preview_payload":{"brand_code":null,"brand_series_list":null,"category_id":0,"category_id_list":null,"category_misplace":false,"commodity_type":0,"confirm_size_img":false,"input_size_chart_flag":0,"is_suggest_category":false,"point_key":"","product_type_id":null,"source_system":"listingkit","spp_relate_spu_name":"","spu_name":"Controlled bottle","suit_flag":0,"supplier_code":"sku-1","top_category_id":0,"invalid":null,"product_certificate_list":[],"certificate_list":[],"del_other_certificate_sn_list":null,"image_info":{"image_group_code":null,"image_info_list":[{"image_type":1,"image_sort":1,"image_url":"https://controlled.invalid/main.jpg","image_item_id":null,"size_img_flag":false,"transformCvSizeImage":false,"ai_status":0,"ps_types":null,"marketing_main_image":true,"commodity_category_flag":null},{"image_type":2,"image_sort":2,"image_url":"https://controlled.invalid/gallery.jpg","image_item_id":null,"size_img_flag":false,"transformCvSizeImage":false,"ai_status":0,"ps_types":null,"marketing_main_image":false,"commodity_category_flag":null}]},"multi_language_desc_list":[{"language":"en","name":"A reusable bottle."},{"language":"en","name":"A reusable bottle."}],"multi_language_name_list":[{"language":"en","name":"Controlled bottle"},{"language":"en","name":"Controlled bottle"}],"multi_language_makeup_ingredient_list":null,"product_attribute_list":null,"product_name_attribute_list":null,"product_video_list":null,"part_info_list":null,"plm_pattern_id_list":null,"site_list":[{"main_site":"US","sub_site_list":["US"]}],"pop_price_site_info":null,"skc_list":[{"from_pre_product":null,"sale_attribute":{"attribute_id":0,"attribute_value_id":0,"pre_fill_spec":true,"is_spp_sale_attribute":false},"sku_list":[{"sale_attribute_list":null,"spp_relate_sku_code":null,"cost_info":{"cost_price":"6.25","currency":"USD"},"height":"","length":"","width":"","length_unit":"","stock_info_list":[{"inventory_num":7,"merchant_warehouse_code":"DEFAULT"}],"weight":0,"weight_unit":"","stock_count":7,"supplier_sku":"sku-1","stop_purchase":0,"mall_state":1,"price_info_list":[{"sub_site":"US","base_price":12.5,"special_price":null,"currency":"USD"}],"image_info":{"image_group_code":null,"image_info_list":[{"image_type":1,"image_sort":1,"image_url":"https://controlled.invalid/main.jpg","image_item_id":null,"size_img_flag":false,"transformCvSizeImage":false,"ai_status":0,"ps_types":null,"marketing_main_image":false,"commodity_category_flag":null}]},"competing_product_link":null,"competing_cost_price_images":null,"suggested_retail_price":null,"quantity_info":null,"package_type":3,"extra":{"fieldDisabledInfo":{"skuQuantity":null,"skuQuantityType":null,"skuQuantityUnit":null}}},{"sale_attribute_list":null,"spp_relate_sku_code":null,"cost_info":{"cost_price":"","currency":"EUR"},"height":"","length":"","width":"","length_unit":"","stock_info_list":[{"inventory_num":2,"merchant_warehouse_code":"DEFAULT"}],"weight":0,"weight_unit":"","stock_count":2,"supplier_sku":"sku-2","stop_purchase":0,"mall_state":1,"price_info_list":[{"sub_site":"US","base_price":18,"special_price":null,"currency":"EUR"}],"image_info":{"image_group_code":null,"image_info_list":[{"image_type":1,"image_sort":1,"image_url":"https://controlled.invalid/main.jpg","image_item_id":null,"size_img_flag":false,"transformCvSizeImage":false,"ai_status":0,"ps_types":null,"marketing_main_image":false,"commodity_category_flag":null}]},"competing_product_link":null,"competing_cost_price_images":null,"suggested_retail_price":null,"quantity_info":null,"package_type":3,"extra":{"fieldDisabledInfo":{"skuQuantity":null,"skuQuantityType":null,"skuQuantityUnit":null}}}],"image_info":{"image_group_code":null,"image_info_list":[{"image_type":1,"image_sort":1,"image_url":"https://controlled.invalid/main.jpg","image_item_id":null,"size_img_flag":false,"transformCvSizeImage":false,"ai_status":0,"ps_types":null,"marketing_main_image":true,"commodity_category_flag":null},{"image_type":2,"image_sort":2,"image_url":"https://controlled.invalid/gallery.jpg","image_item_id":null,"size_img_flag":false,"transformCvSizeImage":false,"ai_status":0,"ps_types":null,"marketing_main_image":false,"commodity_category_flag":null}]},"is_maternal_style":null,"multi_language_name":{"language":"en","name":"sku-1"},"multi_language_name_list":[{"language":"en","name":"sku-1"}],"site_detail_image_info_list":null,"spp_relate_skc_name":null,"hope_on_sale_date":"","shelf_way":0,"shelf_require":0,"proof_of_stock_list":null,"sort":1,"part_code_rel_list":null,"price_filling_tips":null,"skc_material":null,"site_spec_image_info_list":null,"skc_scope_attribute_list":null,"sample_info":null,"is_first_publish":true,"quality_sn":null,"re_quote_file_list":null,"re_quote_reason":null,"supplier_code":"sku-1","extra":{"isSourceEmbryo":false,"isEmbryoInclude":false,"physical_price_review_success":false}}],"size_attribute_list":null,"back_size_attribute_list":null,"custom_attribute_relation":null,"extra":{"bidding_id":"","best_sale_spu":"","activity_type":"","bidding_key":"","subject_skc":"","best_sale_associate_key":"","switch_to_spu_pic":true,"from_page_id":null,"optimize_skc_name":null,"spu_tag":null,"transformCvSizeImage":false,"useCvTransformImage":true,"confirm_volume_sku":null,"confirm_weight_sku":null,"control_price_data":null,"faddish_price_info":{},"best_sale_add_color_size":"","copy_skc_confirm_updated":false,"extra_clear_bidding_or_best_sale":false,"is_spmb_copy_product":false,"bidding_similar_skc":"","armor_token":"","black_box":""}},"metadata":{"authorized_brand_code":"","authorized_brand_name":"","category_name":"Bottles","country":"US","language":"en","source_category_path":"Home \u003e Bottles","target_platform":"shein","title_skc_base":"Controlled bottle","title_source":"canonical_title"}}`, hash: "sha256:40a0b50e59dc98c91ef48024ad526eb1129a39972a7373ef8d620a8d6421c6f9"},
		{name: "prompt_title", country: "GB", language: "fr", snapshot: catalog.ProductSnapshot{Title: "Please design an image, include 2000 pixels", Description: "合成描述"}, want: `{"spu_name":"Please design an image, include 2000 pixels","title_diagnostics":{"source":"unresolved_prompt_title","prompt_contaminated":true,"resolution_note":"prompt-like title candidates could not be safely resolved; llm title extraction unavailable or unsafe"},"site_list":[{"main_site":"GB","sub_sites":["GB"]}],"skc_list":[{"skc_name":"DEFAULT-001","sale_name":"DEFAULT-001","supplier_code":"DEFAULT-001","skus":[{"sku":"DEFAULT-001","is_default":true}]}],"draft_payload":{"spu_name":"Please design an image, include 2000 pixels","supplier_code":"DEFAULT-001","multi_language_name_list":[{"language":"fr"},{"language":"en"}],"multi_language_desc_list":[{"language":"fr"},{"language":"en"}],"site_list":[{"main_site":"GB","sub_sites":["GB"]}],"skc_list":[{"skc_name":"DEFAULT-001","sale_name":"DEFAULT-001","supplier_code":"DEFAULT-001","sort":1,"multi_language_name_list":[{"language":"en","name":"DEFAULT-001"}],"image_info":{},"sku_list":[{"supplier_sku":"DEFAULT-001","is_default":true,"stock_info_list":[{"warehouse_code":"DEFAULT"}]}]}]},"preview_payload":{"brand_code":null,"brand_series_list":null,"category_id":0,"category_id_list":null,"category_misplace":false,"commodity_type":0,"confirm_size_img":false,"input_size_chart_flag":0,"is_suggest_category":false,"point_key":"","product_type_id":null,"source_system":"listingkit","spp_relate_spu_name":"","spu_name":"Please design an image, include 2000 pixels","suit_flag":0,"supplier_code":"DEFAULT-001","top_category_id":0,"invalid":null,"product_certificate_list":[],"certificate_list":[],"del_other_certificate_sn_list":null,"multi_language_desc_list":[],"multi_language_name_list":[],"multi_language_makeup_ingredient_list":null,"product_attribute_list":null,"product_name_attribute_list":null,"product_video_list":null,"part_info_list":null,"plm_pattern_id_list":null,"site_list":[{"main_site":"GB","sub_site_list":["GB"]}],"pop_price_site_info":null,"skc_list":[{"from_pre_product":null,"sale_attribute":{"attribute_id":0,"attribute_value_id":0,"pre_fill_spec":true,"is_spp_sale_attribute":false},"sku_list":[{"sale_attribute_list":null,"spp_relate_sku_code":null,"cost_info":{"cost_price":"","currency":""},"height":"","length":"","width":"","length_unit":"","stock_info_list":[{"inventory_num":0,"merchant_warehouse_code":"DEFAULT"}],"weight":0,"weight_unit":"","stock_count":0,"supplier_sku":"DEFAULT-001","stop_purchase":0,"mall_state":1,"price_info_list":null,"image_info":null,"competing_product_link":null,"competing_cost_price_images":null,"suggested_retail_price":null,"quantity_info":null,"package_type":3,"extra":{"fieldDisabledInfo":{"skuQuantity":null,"skuQuantityType":null,"skuQuantityUnit":null}}}],"image_info":{"image_group_code":null},"is_maternal_style":null,"multi_language_name":{"language":"en","name":"DEFAULT-001"},"multi_language_name_list":[{"language":"en","name":"DEFAULT-001"}],"site_detail_image_info_list":null,"spp_relate_skc_name":null,"hope_on_sale_date":"","shelf_way":0,"shelf_require":0,"proof_of_stock_list":null,"sort":1,"part_code_rel_list":null,"price_filling_tips":null,"skc_material":null,"site_spec_image_info_list":null,"skc_scope_attribute_list":null,"sample_info":null,"is_first_publish":true,"quality_sn":null,"re_quote_file_list":null,"re_quote_reason":null,"supplier_code":"DEFAULT-001","extra":{"isSourceEmbryo":false,"isEmbryoInclude":false,"physical_price_review_success":false}}],"size_attribute_list":null,"back_size_attribute_list":null,"custom_attribute_relation":null,"extra":{"bidding_id":"","best_sale_spu":"","activity_type":"","bidding_key":"","subject_skc":"","best_sale_associate_key":"","switch_to_spu_pic":true,"from_page_id":null,"optimize_skc_name":null,"spu_tag":null,"transformCvSizeImage":false,"useCvTransformImage":true,"confirm_volume_sku":null,"confirm_weight_sku":null,"control_price_data":null,"faddish_price_info":{},"best_sale_add_color_size":"","copy_skc_confirm_updated":false,"extra_clear_bidding_or_best_sale":false,"is_spmb_copy_product":false,"bidding_similar_skc":"","armor_token":"","black_box":""}},"metadata":{"authorized_brand_code":"","authorized_brand_name":"","category_name":"","country":"GB","english_description_generation":"failed","language":"fr","source_category_path":"","target_platform":"shein","title_prompt_contaminated":"true","title_resolution_note":"prompt-like title candidates could not be safely resolved; llm title extraction unavailable or unsafe","title_source":"unresolved_prompt_title"},"review_notes":["英文描述生成失败，无法进入最终提交"]}`, hash: "sha256:bdf2f56e52357a377ae2d6981441e70295c0bf1c6a008f7563ca3d15fe030d85"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := (Builder{}).Build(context.Background(), tc.snapshot, productasset.ApprovedAssetInventory{Assets: tc.assets}, record.Input{Country: tc.country, Language: tc.language})
			require.NoError(t, err)
			hash := fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
			if string(raw) != tc.want || hash != tc.hash {
				t.Fatalf("FROZEN %s %q %s", tc.name, raw, hash)
			}
		})
	}
}
