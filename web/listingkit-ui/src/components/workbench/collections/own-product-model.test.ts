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
 expect(()=>prepareOwnProduct({...base,mode:"single",rows:[{...row,sku:""}]})).toThrow("请填写SKU、币种、价格和库存");
});
it.each([
 {currency:"CNY"}, {price:"0"}, {stock:"0"},
 {currency:"CNY",price:"12.5",stock:"8"},
 {image:"https://images.example.org/red.png"},
 {title:"已填写的SKU名称"}, {attributes:{颜色:"红色"}},
])("rejects a partially populated single row instead of dropping its facts: %j",patch=>{
 const row={...newVariantRow(),...patch,sku:"   "};
 expect(()=>prepareOwnProduct({title:"衣服",brand:"",description:"",images:["https://images.example.org/red.png"],mode:"single",rows:[row]})).toThrow("请填写SKU、币种、价格和库存");
});
it("allows a completely blank single row while retaining the common product facts",()=>{
 const base={title:"衣服",brand:"品牌",description:"基础资料",images:["https://images.example.org/common.png"],mode:"single" as const};
 for(const row of [newVariantRow(),{...newVariantRow(),sku:"  ",currency:" ",price:" ",stock:" ",image:" ",title:" "}]){
  expect(prepareOwnProduct({...base,rows:[row]})).toMatchObject({title:base.title,brand:base.brand,description:base.description,images:base.images});
  expect(prepareOwnProduct({...base,rows:[row]}).variants).toBeUndefined();
 }
});
it("keeps explicit zero and the stable identity of a complete single row, including an existing row without SKU",()=>{
 const row={...newVariantRow(),sku:"SINGLE",currency:"CNY",price:"0",stock:"0"};
 const base={title:"衣服",brand:"",description:"",images:[],mode:"single" as const};
 expect(prepareOwnProduct({...base,rows:[row]}).variants?.[0]).toMatchObject({sourceId:row.id,sku:"SINGLE",price:0,stock:0,currency:"CNY"});
 expect(prepareOwnProduct({...base,rows:[{...row,existing:true,sku:"",title:"原SKU名称"}]}).variants?.[0]).toMatchObject({sourceId:row.id,sku:"",title:"原SKU名称",price:0,stock:0,currency:"CNY"});
});
it("preserves common attributes separately from SKU specifications and rejects incomplete or repeated names",()=>{
 const base={title:"衣服",brand:"",description:"",images:[],mode:"single" as const,rows:[newVariantRow()]};
 const attributes=[{id:"a",name:" 材质 ",value:" 棉 "},{id:"b",name:"适用季节",value:"夏季"},{id:"empty",name:"",value:""}];
 expect(prepareOwnProduct({...base,attributes}).attributes).toEqual({材质:"棉",适用季节:"夏季"});
 expect(()=>prepareOwnProduct({...base,attributes:[{id:"a",name:"材质",value:""}]})).toThrow("名称和属性值");
 expect(()=>prepareOwnProduct({...base,attributes:[{id:"a",name:"材质",value:"棉"},{id:"b",name:" 材质 ",value:"麻"}]})).toThrow("属性名称不能重复");
 expect(()=>prepareOwnProduct({...base,attributes:Array.from({length:257},(_,i)=>({id:String(i),name:`属性${i}`,value:"值"}))})).toThrow();
});
it("counts common attributes before allocating specification combinations",()=>{
 const specs=[{id:"a",name:"a",values:"1,2,3"},{id:"b",name:"b",values:"1,2,3,4,5"},{id:"c",name:"c",values:Array.from({length:17},(_,i)=>String(i)).join(",")}];
 expect(()=>buildVariantRows(specs,[],1,4)).toThrow("保存限制");
});
