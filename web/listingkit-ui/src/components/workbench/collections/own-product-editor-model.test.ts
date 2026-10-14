import {expect,it} from "vitest";
import type {CollectionDetail} from "@/lib/contracts/product-collection";
import {hydrateOwnProduct} from "./own-product-editor-model";
import {prepareOwnProduct} from "./own-product-model";

const a="https://images.example.org/public.png",b="https://images.example.org/sku.png",c="https://images.example.org/sku-detail.png";
export const savedProduct={item:{source:{kind:"own"}},product:{title:"原名",description:"原描述",brand:"原品牌",attributes:[{name:"材质",value:"棉"}],images:[{url:a}],variants:[{source_id:"stable-id",title:"自定义SKU名",sku:"SKU",attributes:[{name:"颜色",value:"红色"}],price:{currency:"CNY",amount:0},images:[{url:b},{url:c}]}]}} as CollectionDetail;
it("hydrates a single attributed SKU without promoting variant images or regenerating combinations",()=>{
 const form=hydrateOwnProduct(savedProduct);
 expect(form.mode).toBe("multiple");expect(form.images).toEqual([a]);expect(form.rows).toHaveLength(1);
 const output=prepareOwnProduct({...form,rows:form.rows});
 expect(output.variants?.[0]).toEqual({sourceId:"stable-id",title:"自定义SKU名",sku:"SKU",attributes:{颜色:"红色"},currency:"CNY",price:0,stock:0,images:[b,c]});
 expect(prepareOwnProduct({...form,title:"改商品名",rows:form.rows}).variants?.[0].title).toBe("自定义SKU名");
 expect(prepareOwnProduct({...form,rows:form.rows.map(row=>({...row,image:a}))}).variants?.[0].images).toEqual([a]);
});
it("rejects missing identity/price and unsupported current facts rather than overwriting them",()=>{
 expect(()=>hydrateOwnProduct({...savedProduct,product:{...savedProduct.product,variants:[{sku:"SKU"}]}})).toThrow();
 expect(()=>hydrateOwnProduct({...savedProduct,product:{...savedProduct.product,selling_points:["未支持资料"]}})).toThrow();
});
it("keeps only saved rows when dimensions describe a larger Cartesian product",()=>{
 const variants=[savedProduct.product.variants![0],{...savedProduct.product.variants![0],source_id:"blue-m",sku:"BLUE-M",attributes:[{name:"颜色",value:"蓝色"},{name:"尺码",value:"M"}]}];
 const form=hydrateOwnProduct({...savedProduct,product:{...savedProduct.product,variants}});
 expect(form.rows).toHaveLength(2);expect(form.specs).toHaveLength(2);
});
