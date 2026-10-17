import {expect,it} from "vitest";
import {compareOwnProducts} from "./own-product-diff";
import type {CollectionDetail} from "@/lib/contracts/product-collection";
type Product=CollectionDetail["product"];
it("matches renamed SKUs by stable identity and preserves zero and image order",()=>{
 const before:Product={title:"旧名称",attributes:[{name:"材质",value:"棉"},{name:"季节",value:"夏"}],images:[{url:"https://a/x"},{url:"https://a/y"}],variants:[{source_id:"red",sku:"OLD",price:{currency:"CNY",amount:0}},{source_id:"removed",sku:"REMOVED"}]};
 const after:Product={...before,title:"新名称",attributes:[{name:"季节",value:"夏"},{name:"材质",value:"棉"}],images:[{url:"https://a/y"},{url:"https://a/x"}],variants:[{source_id:"red",sku:"NEW",price:{currency:"CNY",amount:5},stock:0},{source_id:"added",sku:"ADDED"}]};
 const rows=compareOwnProducts(before,after);
 expect(rows.find(row=>row.label==="商品名称")).toMatchObject({before:"旧名称",after:"新名称",status:"changed"});
 expect(rows.some(row=>row.label.includes("商品属性"))).toBe(false);
 expect(rows.find(row=>row.label.includes("公共图片"))).toMatchObject({status:"changed"});
 expect(rows.find(row=>row.key===JSON.stringify(["sku","red","价格"]))).toMatchObject({before:"0",after:"5",status:"changed"});
 expect(rows.find(row=>row.key===JSON.stringify(["sku","red","库存"]))).toBeUndefined();
 expect(rows.some(row=>row.status==="added"&&row.label.includes("ADDED"))).toBe(true);
 expect(rows.some(row=>row.status==="removed"&&row.label.includes("REMOVED"))).toBe(true);
});
it("distinguishes missing prices from declared zero and rejects ambiguous SKU identities",()=>{
 const a:Product={variants:[{source_id:"one",sku:"SKU"}]};
 const rows=compareOwnProducts(a,{variants:[{source_id:"one",sku:"SKU",price:{amount:0}}]});
 expect(rows.find(row=>row.label.includes("价格"))).toMatchObject({before:"未填写",after:"0"});
 expect(()=>compareOwnProducts(a,{variants:[{source_id:"same"},{source_id:"same"}]})).toThrow();
 expect(()=>compareOwnProducts(a,{variants:[{sku:"missing"}]})).toThrow();
});

it("includes a description format change even when stored characters match",()=>{
 const rows=compareOwnProducts({description:"<p>same</p>"},{description:"<p>same</p>",description_format:"html-v1"});
 expect(rows).toEqual(expect.arrayContaining([expect.objectContaining({label:"描述格式",before:"纯文本",after:"富文本"})]));
});
