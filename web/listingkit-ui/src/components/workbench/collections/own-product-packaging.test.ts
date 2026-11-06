import {expect,it} from "vitest";
import {ownProductSchema,collectionCommandSchema,type CollectionDetail} from "@/lib/contracts/product-collection";
import {buildVariantRows,newVariantRow,prepareOwnProduct} from "./own-product-model";
import {hydrateOwnProduct} from "./own-product-editor-model";
import {compareOwnProducts} from "./own-product-diff";

it("strictly accepts partial metric packaging and keeps stock zero distinct from unknown",()=>{
 const base={title:"商品",description:"",images:[],variants:[{sourceId:"red",title:"红",sku:"RED",attributes:{颜色:"红"},currency:"CNY",price:0,stock:0,packaging:{weightGrams:125.5,lengthCm:10,widthCm:null}}]};
 const product=ownProductSchema.parse(base);
 expect(JSON.parse(JSON.stringify(product)).variants[0]).toMatchObject({stock:0,packaging:{weightGrams:125.5,lengthCm:10}});
 for(const packaging of [{weightGrams:0},{lengthCm:-1},{heightCm:Infinity},{widthCm:"10"},{weightGrams:12,unit:"kg"}])expect(ownProductSchema.safeParse({...base,variants:[{...base.variants[0],packaging}]}).success).toBe(false);
 expect(ownProductSchema.safeParse({...base,variants:[{...base.variants[0],packaging:null}]}).success).toBe(true);
});

it("keeps packaging with stable rows across reordering and leaves new combinations unknown",()=>{
 const specs=[{id:"color",name:"颜色",values:"红,蓝"}];
 const rows=buildVariantRows(specs,[]).map((row,index)=>Object.assign(row,{sku:index?"BLUE":"RED",currency:"CNY",price:"0",stock:"0",packaging:{weightGrams:index?"250":"125.5",lengthCm:index?"20":"10"}}));
 const next=buildVariantRows([{...specs[0],values:"蓝,红,绿"}],rows);
 expect(next.slice(0,2).map(row=>row.id)).toEqual([rows[1].id,rows[0].id]);
 const product=prepareOwnProduct({title:"商品",brand:"",description:"",images:[],mode:"multiple",rows:next.slice(0,2)});
 expect(product.variants).toEqual([expect.objectContaining({sourceId:rows[1].id,packaging:{weightGrams:250,lengthCm:20}}),expect.objectContaining({sourceId:rows[0].id,stock:0,packaging:{weightGrams:125.5,lengthCm:10}})]);
 expect(prepareOwnProduct({title:"商品",brand:"",description:"",images:[],mode:"multiple",rows:[Object.assign(next[2],{sku:"GREEN",currency:"CNY",price:"0",stock:"0"})]}).variants?.[0]).not.toHaveProperty("packaging");
 expect(()=>prepareOwnProduct({title:"商品",brand:"",description:"",images:[],mode:"single",rows:[Object.assign(newVariantRow(),{packaging:{weightGrams:"12"}})]})).toThrow("请填写SKU、币种、价格和库存");
});

it("hydrates exact packaging independently from sales dimensions and includes it in pending revisions and history differences",()=>{
 const detail=JSON.parse('{"item":{"source":{"kind":"own"}},"product":{"title":"商品","description":"","images":[],"variants":[{"source_id":"red","sku":"RED","attributes":[{"name":"颜色","value":"红"}],"price":{"currency":"CNY","amount":0},"stock":0,"own_stock_declared":true,"packaging":{"weightGrams":125.5,"lengthCm":10}}]}}') as CollectionDetail;
 const form=hydrateOwnProduct(detail);
 expect(form.specs.map(spec=>spec.name)).toEqual(["颜色"]);
 const product=prepareOwnProduct(form);
 expect(product.variants?.[0]).toMatchObject({sourceId:"red",stock:0,packaging:{weightGrams:125.5,lengthCm:10}});
 const id="550e8400-e29b-41d4-a716-446655440002";
 const pending=collectionCommandSchema.parse({action:"revise_product",itemId:id,expectedRevision:1,expectedSourceVersion:"1",expectedPublicationId:id,product});
 expect(JSON.parse(JSON.stringify(pending)).product.variants[0].packaging).toEqual({weightGrams:125.5,lengthCm:10});
 const changed=JSON.parse(JSON.stringify(detail.product));changed.variants[0].sku="RENAMED";changed.variants[0].packaging.weightGrams=250;
 expect(compareOwnProducts(detail.product,changed)).toEqual(expect.arrayContaining([expect.objectContaining({label:expect.stringContaining("含包装重量"),before:"125.5",after:"250",status:"changed"})]));
});
