import { expect,it } from "vitest";
import { initialSupplyDraft,applyApplicationMode,sampleForSKU,applyCompletionText } from "./draft-fields";
import type { SupplySourceDetail } from "@/lib/contracts/supply-chain";
it("explicitly brings in candidate text while preserving target category, images and other languages",()=>{
 const draft=initialSupplyDraft({product:{title:"original"}} as SupplySourceDetail);draft.product.category_id=123;draft.product.multi_language_name_list.push({language:"zh",name:"中文"});draft.images=[{group:"spu",skc:0,sku:0,asset_id:"approved",sort:1,type:1}];
 const brought=applyCompletionText(draft,{title:"saved candidate",description:"saved description"});
 expect(brought.product.category_id).toBe(123);expect(brought.images).toEqual(draft.images);expect(brought.product.multi_language_name_list).toEqual([{language:"en",name:"saved candidate"},{language:"zh",name:"中文"}]);expect(draft.product.multi_language_name_list[0].name).toBe("original");
});
it("copies source names and SKU identities while leaving target price and stock unfilled",()=>{
 const value=initialSupplyDraft({product:{title:"来源标题",description:"来源说明",variants:[{sku:"source-sku",price:{amount:2},stock:99}]}} as SupplySourceDetail);
 expect(value.product.multi_language_name_list).toEqual([{language:"en",name:"来源标题"}]);
 expect(value.product.skc_list[0]?.sku_list[0]).toMatchObject({supplier_sku:"source-sku",stock_info_list:[]});
 expect(value.product.skc_list[0]?.sku_list[0]?.price_info_list).toBeUndefined();
});
it("selects sample specifications from an exact real SKU combination",()=>{
 const draft=initialSupplyDraft({product:{}} as SupplySourceDetail);
 expect(sampleForSKU(draft,0,0)).toBeUndefined();
 draft.product.skc_list[0]!.sale_attribute={attribute_id:12,attribute_value_id:34};
 draft.product.skc_list[0]!.sku_list[0]!.sale_attribute_list=[{attribute_id:56,attribute_value_id:78}];
 expect(sampleForSKU(draft,0,0)).toEqual({sample_judge_type:2,reserve_sample_flag:2,spot_flag:0,sample_spec:{main_spec:{attribute_id:12,attribute_value_id:34},sub_spec_list:[{attribute_id:"56",attribute_value_id:"78"}]}});
 expect(sampleForSKU(draft,0,1)).toBeUndefined();
});
it("requires explicit re-entry of incompatible mode fields when switching targets",()=>{
 const value=initialSupplyDraft({product:{}} as SupplySourceDetail);
 value.product.site_list=[{main_site:"shein",sub_site_list:["shein-us"]}];
 value.product.skc_list[0]!.sku_list[0]!.price_info_list=[{base_price:3,currency:"USD",sub_site:"shein-us"}];
 value.product.skc_list[0]!.sku_list[0]!.stock_info_list=[{inventory_num:42,supplier_warehouse_id:"old"}];
 const full=applyApplicationMode(value,"fully_managed");
 expect(full.product.site_list).toBeUndefined();expect(full.product.skc_list[0]?.sku_list[0]?.price_info_list).toBeUndefined();
 expect(full.product.skc_list[0]?.sku_list[0]?.stock_info_list).toEqual([]);
 expect(full.product.skc_list[0]?.sku_list[0]?.cost_info).toBeUndefined();
});
