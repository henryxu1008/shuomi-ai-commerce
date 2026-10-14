import type {CollectionDetail} from "@/lib/contracts/product-collection";
type Product=CollectionDetail["product"];
type Field={label:string;value:string;images?:boolean};
export type OwnDiffRow={key:string;label:string;before:string;after:string;status:"added"|"removed"|"changed";images?:boolean};
function fields(product:Product){
 const result=new Map<string,Field>();
 const put=(key:string[],label:string,value:string,images=false)=>result.set(JSON.stringify(key),{label,value,images});
 for(const [key,label] of [["title","商品名称"],["brand","品牌"],["description","商品描述"]] as const)put([key],label,product[key]??"");
 const attrs=(key:string[],label:string,values:Product["attributes"])=>{const seen=new Set<string>();for(const attr of values??[]){if(!attr.name||seen.has(attr.name))throw new Error("属性名称不完整或重复，无法准确对比");seen.add(attr.name);put([...key,attr.name],`${label} · ${attr.name}`,attr.value??"")}};
 attrs(["attributes"],"商品属性",product.attributes);put(["images"],"公共图片",(product.images??[]).map(image=>image.url).join("\n"),true);
 const ids=new Set<string>();
 for(const variant of product.variants??[]){
  if(!variant.source_id||ids.has(variant.source_id))throw new Error("变体身份不完整或重复，无法准确对比");ids.add(variant.source_id);
  const key=["sku",variant.source_id],label=`SKU ${variant.sku||variant.source_id}`;
  put([...key,"变体"],label,variant.sku||variant.source_id);
  for(const [name,value] of [["SKU名称",variant.title??""],["SKU编码",variant.sku??""],["币种",variant.price?.currency??"未填写"],["价格",variant.price?.amount===undefined?"未填写":String(variant.price.amount)],["库存",String(variant.stock??0)]] as const)put([...key,name],`${label} · ${name}`,value);
  attrs([...key,"attributes"],`${label} · 销售规格`,variant.attributes);put([...key,"images"],`${label} · 图片`,(variant.images??[]).map(image=>image.url).join("\n"),true);
 }
 return result;
}
export function compareOwnProducts(before:Product,after:Product):OwnDiffRow[]{
 const left=fields(before),right=fields(after),result:OwnDiffRow[]=[];
 for(const key of new Set([...left.keys(),...right.keys()])){
  const a=left.get(key),b=right.get(key);if(a?.value===b?.value)continue;
  const field=b??a!;result.push({key,label:field.label,before:a?.value??"未提供",after:b?.value??"未提供",status:!a?"added":!b?"removed":"changed",images:field.images});
 }
 return result;
}
