import {expect,it} from "vitest";
import {ownProductSchema} from "./product-collection";

it("accepts multiple declared variants with their own image and explicit zero values",()=>{
 const variant={sourceId:"red-s",title:"红色 / S",sku:"RED-S",attributes:{颜色:"红色",尺码:"S"},currency:"CNY",price:0,stock:0,images:["https://images.example.org/red.png"]};
 const parsed=ownProductSchema.safeParse({title:"多规格上衣",description:"",images:[],variants:[variant,{...variant,sourceId:"blue-m",sku:"BLUE-M",attributes:{颜色:"蓝色",尺码:"M"}}]});
 expect(parsed.success).toBe(true);
});

it("rejects repeated SKU and actual source-publication budget overflow",()=>{
 const variant={sourceId:"red-s",title:"红色 / S",sku:"RED-S",attributes:{颜色:"红色",尺码:"S"},currency:"CNY",price:0,stock:0};
 expect(ownProductSchema.safeParse({title:"商品",description:"",images:[],variants:[variant,{...variant,sourceId:"other"}]}).success).toBe(false);
 expect(ownProductSchema.safeParse({title:"商品",description:"",images:[],variants:Array.from({length:257},(_,i)=>({...variant,sourceId:String(i),sku:String(i)}))}).success).toBe(false);
});
