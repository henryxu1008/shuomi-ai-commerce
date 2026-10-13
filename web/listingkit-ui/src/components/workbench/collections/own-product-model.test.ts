import {expect,it} from "vitest";
import {buildVariantRows,newVariantRow,prepareOwnProduct} from "./own-product-model";

it("builds six combinations and preserves their entered facts and identity on regeneration",()=>{
 const specs=[{id:"color",name:"颜色",values:"红色,蓝色"},{id:"size",name:"尺码",values:"S,M,L"}];
 const rows=buildVariantRows(specs,[]);expect(rows).toHaveLength(6);
 rows[0]={...rows[0],sku:"RED-S",price:"0",stock:"0",currency:"CNY",image:"https://images.example.org/red.png"};
 const next=buildVariantRows([...specs,{id:"empty",name:"",values:""}],rows);
 expect(next[0]).toEqual(rows[0]);expect(new Set(next.map(v=>v.id)).size).toBe(6);
});
it("rejects duplicate specification names or values and computes limits before allocating combinations",()=>{
 expect(()=>buildVariantRows([{id:"a",name:"颜色",values:"红,红"}],[])).toThrow();
 expect(()=>buildVariantRows([{id:"a",name:"颜色",values:"红"},{id:"b",name:"颜色",values:"蓝"}],[])).toThrow();
 expect(()=>buildVariantRows([{id:"a",name:"颜色",values:Array.from({length:257},(_,i)=>String(i)).join(",")}],[])).toThrow();
});
it("includes retained variant images in the aggregate limit when regenerating",()=>{
 const specs=[{id:"a",name:"a",values:"1,2,3"},{id:"b",name:"b",values:"1,2,3,4,5"},{id:"c",name:"c",values:Array.from({length:17},(_,i)=>String(i)).join(",")}];
 const rows=buildVariantRows(specs,[],1).map((row,i)=>({...row,image:i<4?"https://images.example.org/a.png":""}));
 expect(()=>buildVariantRows(specs,rows,1)).toThrow("保存限制");
});
it("keeps explicit zero and rejects empty numbers or duplicate SKU before submission",()=>{
 const row={...newVariantRow({颜色:"红"}),sku:"RED",currency:"CNY",price:"0",stock:"0"};
 const base={title:"衣服",brand:"",description:"",images:[],mode:"multiple" as const,rows:[row]};
 expect(prepareOwnProduct(base).variants?.[0]).toMatchObject({price:0,stock:0,sourceId:row.id});
 expect(()=>prepareOwnProduct({...base,rows:[{...row,price:""}]})).toThrow();
 expect(()=>prepareOwnProduct({...base,rows:[row,{...row,id:"other"}]})).toThrow();
 expect(prepareOwnProduct({...base,mode:"single",rows:[{...row,sku:""}]}).variants).toBeUndefined();
});
