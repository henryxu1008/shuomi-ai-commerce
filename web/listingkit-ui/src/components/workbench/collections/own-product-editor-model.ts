import {ownProductSchema,type CollectionDetail} from "@/lib/contracts/product-collection";
import type {z} from "zod";
import {newVariantRow,type VariantRow,type Specification,type ProductAttributeRow} from "./own-product-model";

export type OwnProductFormState={title:string;brand:string;description:string;descriptionFormat?:"html-v1";images:string[];mode:"single"|"multiple";rows:VariantRow[];specs:Specification[];attributes:ProductAttributeRow[]};
const unreadable=()=>new Error("当前商品包含无法完整编辑的资料，请返回详情查看；尚未保存任何修改。");
function attributeMap(values:NonNullable<CollectionDetail["product"]["attributes"]>){
 const result:Record<string,string>=Object.create(null);
 for(const value of values){if(value.name===undefined||value.value===undefined||!value.name.trim()||!value.value.trim()||value.name!==value.name.trim()||value.value!==value.value.trim()||Object.hasOwn(result,value.name))throw unreadable();result[value.name]=value.value}
 return result;
}
export function hydrateOwnProduct(detail:CollectionDetail):OwnProductFormState{
 if(detail.item.source.kind!=="own")throw unreadable();
 const p=detail.product;
 if(p.category_path?.length||p.selling_points?.length||p.seo_keywords?.length||p.specifications&&Object.keys(p.specifications).length)throw unreadable();
 if([...(p.images??[]),...(p.variants??[]).flatMap(v=>v.images??[])].some(image=>image.width||image.height))throw unreadable();
 const images=(p.images??[]).map(image=>image.url);
 const variants=(p.variants??[]).map(v=>{
  if(!v.source_id||v.price?.amount===undefined||!v.price.currency||v.barcode||v.is_default||v.price.compare_at!==undefined||v.price.cost_price!==undefined||v.price.wholesale_min!==undefined)throw unreadable();
  return {sourceId:v.source_id,title:v.title??"",sku:v.sku??"",attributes:attributeMap(v.attributes??[]),currency:v.price.currency,price:v.price.amount,stock:v.stock??0,images:(v.images??[]).map(image=>image.url)};
 });
 const raw={title:p.title??"",brand:p.brand??"",description:p.description??"",...(p.description_format?{descriptionFormat:p.description_format}:{}),images,attributes:attributeMap(p.attributes??[]),variants};
 const parsed=ownProductSchema.safeParse(raw);
 if(!parsed.success||raw.title!==raw.title.trim()||raw.brand!==raw.brand.trim())throw unreadable();
 return formFromOwnProduct(parsed.data);
}
export function formFromOwnProduct(raw:z.infer<typeof ownProductSchema>):OwnProductFormState{
 const rows:VariantRow[]=(raw.variants??[]).map(v=>({id:v.sourceId,title:v.title,sku:v.sku,attributes:v.attributes,currency:v.currency,price:String(v.price),stock:String(v.stock),image:v.images?.[0]??"",originalImages:v.images??[],existing:true}));
 const dimensions=new Map<string,Set<string>>();
 for(const row of rows)for(const [name,value] of Object.entries(row.attributes)){if(!dimensions.has(name))dimensions.set(name,new Set());dimensions.get(name)!.add(value)}
 const specs=[...dimensions].map(([name,values],index)=>({id:`dimension-${index}`,name,values:[...values].join(",")}));
 return {title:raw.title,brand:raw.brand??"",description:raw.description,...(raw.descriptionFormat?{descriptionFormat:raw.descriptionFormat}:{}),images:raw.images,rows:rows.length?rows:[newVariantRow()],mode:rows.length>1||dimensions.size?"multiple":"single",specs:specs.length?specs:[{id:"initial",name:"",values:""}],attributes:Object.entries(raw.attributes??{}).map(([name,value],index)=>({id:`attribute-${index}`,name,value}))};
}
