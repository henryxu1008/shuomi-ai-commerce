// Isolated unit/component data; never used by a running application.
export const importIds={source:"11111111-1111-4111-8111-111111111111",batch:"22222222-2222-4222-8222-222222222222",store:"33333333-3333-4333-8333-333333333333",base:"44444444-4444-4444-8444-444444444444",candidate:"55555555-5555-4555-8555-555555555555",saved:"66666666-6666-4666-8666-666666666666"};
export function sheinImportFixture(){
 const id=importIds,time="2026-10-16T00:00:00Z",hash="a".repeat(64);
 const source={id:id.source,preparationId:id.batch,collectionItemId:id.batch,collectionRevision:1,source:{kind:"own",productKey:"fixed",publicationId:id.batch,version:"1"}};
 const detail={source,images:[],product:{title:"source",variants:[{sku:"",title:"第一款",attributes:[{name:"颜色",value:"蓝"}]}]}};
 const candidate={id:id.candidate,draftId:id.candidate,revision:3,source,content:{title:"candidate",description:"",structured:{attributes:[{name:"材质",value:"棉"}],variants:[{sourceIndex:0,price:{currency:"USD",amount:12.5},stock:0}]}},status:"pending_completion",createdAt:time};
 const sku={supplier_sku:"target-sku",length:"12",mall_state:1,price_info_list:[{base_price:8,currency:"USD",sub_site:"shein-us"}],stock_info_list:[{inventory_num:7,supplier_warehouse_id:"a",supplier_warehouse_name:"仓库A"}],sale_attribute_list:[]};
 const draft={product:{category_id:123,product_type_id:456,source_system:"supply",suit_flag:"0",is_spu_pic:false,multi_language_name_list:[{language:"en",name:"original"}],multi_language_desc_list:[{language:"en",name:"keep"}],product_attribute_list:[],skc_list:[{sale_attribute:{attribute_id:0},image_info:{image_info_list:[]},sku_list:[sku]}]},images:[]};
 const input={sourceId:id.source,storeId:id.store,expectedRevision:0,effectiveVersion:"1",draft};
 const merchant={organization_id:"org",store_id:id.store,site:"shein-us",store_version:1,connection_revision:1,application_revision:"v1:self_operated",application_id:"app",application_type:"self_operated",supplier_identity_hash:hash,service_expires_at:"2027-01-01T00:00:00Z"};
 const base={id:id.base,targetId:id.base,revision:1,source,effectiveVersion:"1",productHash:hash,inventoryHash:hash,rulesHash:hash,merchant,input,result:{product:draft.product,images:[],issues:[],ready_for_upload:false},createdAt:time};
 const rules={merchant,rules:{application_type:"self_operated",categories:[],brands:[],sites:[],warehouses:[{warehouseCode:"a",warehouseName:"仓库A",saleCountryList:["US"],warehouseType:1}],fill:{fill_in_standard_list:[],currency:"USD",default_language:"en",default_language_title_max_length:150,language_title_max_length_list:[],picture_config_list:[],supplier_code_in_spu_dimension:false,support_duplicate_sale_attr_value:false},attributes:{product_type_id:456,main_attribute_status:0,attribute_infos:[{attribute_id:9,attribute_name:"Material",attribute_is_show:1,attribute_type:4,attribute_label:0,attribute_mode:2,attribute_input_num:1,attribute_status:2,data_dimension:1,value_info_list:[{attribute_value_id:11,attribute_value:"Cotton",is_show:1,is_custom_attribute_value:0,supplier_id:0}],rule_info_list:[]}]},linked:[]}};
 const selection={title:true,description:false,attributes:[],skus:[]};
 const request={sourceId:id.source,baseRecordId:id.base,completionDraft:{recordId:id.candidate,revision:3},selection};
 const preview={input:{...input,expectedRevision:1,draft:{...draft,product:{...draft.product,multi_language_name_list:[{language:"en",name:"candidate"}]}},sheinImport:{targetId:id.base,baseRecordId:id.base,completionDraft:request.completionDraft,selection,rulesHash:hash,merchantHash:hash}}};
 const saved={record:{...base,id:id.saved,revision:2,input:preview.input},replayed:false};
 return {detail,candidate,base,rules,request,preview,saved};
}
