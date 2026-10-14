import {ownProductSchema} from "@/lib/contracts/product-collection";

export type Specification={id:string;name:string;values:string};
export type ProductAttributeRow={id:string;name:string;value:string};
export type VariantRow={id:string;attributes:Record<string,string>;sku:string;currency:string;price:string;stock:string;image:string;title?:string;originalImages?:string[];existing?:boolean};
export function newVariantRow(attributes:Record<string,string>={}):VariantRow{return {id:crypto.randomUUID(),attributes,sku:"",currency:"",price:"",stock:"",image:""}}
const comboKey=(attributes:Record<string,string>)=>JSON.stringify(Object.entries(attributes).sort(([a],[b])=>a.localeCompare(b)));
function parseSpecifications(specs:Specification[]){
 const active=specs.filter(s=>s.name.trim()||s.values.trim());
 if(!active.length)throw new Error("请填写规格名称和值，例如颜色：红色、蓝色。");
 const names=new Set<string>();
 return active.map(s=>{const name=s.name.trim(),values=s.values.split(/[,，、\n]/).map(v=>v.trim()).filter(Boolean);
  if(!name||!values.length)throw new Error("每项规格都需要名称和至少一个规格值。");
  if(names.has(name)||new Set(values).size!==values.length)throw new Error("规格名称和值不能重复。");
  if(new TextEncoder().encode(name).length>200||values.some(v=>new TextEncoder().encode(v).length>8192))throw new Error("规格文字过长。");
  names.add(name);return {name,values};});
}
export function specificationSignature(specs:Specification[]){try{return JSON.stringify(parseSpecifications(specs))}catch{return ""}}
export function buildVariantRows(specs:Specification[],prior:VariantRow[],imageCount=0,attributeCount=0):VariantRow[]{
 const dimensions=parseSpecifications(specs),count=dimensions.reduce((n,s)=>n*s.values.length,1);
 if(!Number.isSafeInteger(count)||count>256||count*(1+dimensions.length)+imageCount+attributeCount>1024)throw new Error("规格组合超过当前保存限制，请减少规格值。");
 const existing=new Map(prior.map(row=>[comboKey(row.attributes),row]));
 let combinations:Record<string,string>[]=[{}];
 for(const dimension of dimensions)combinations=combinations.flatMap(attrs=>dimension.values.map(v=>({...attrs,[dimension.name]:v})));
 const rows=combinations.map(attrs=>existing.get(comboKey(attrs))??newVariantRow(attrs));
 if(count*(1+dimensions.length)+imageCount+attributeCount+rows.reduce((sum,row)=>sum+variantImages(row).length,0)>1024)throw new Error("规格与已有变体图片超过当前保存限制，请减少组合或图片。");
 return rows;
}
export function prepareProductAttributes(rows:ProductAttributeRow[]){
 const entries: [string,string][]=[];const names=new Set<string>();
 for(const row of rows){const name=row.name.trim(),value=row.value.trim();if(!name&&!value)continue;
  if(!name||!value)throw new Error("每项商品属性都需要属性名称和属性值。");
  if(names.has(name))throw new Error("商品属性名称不能重复。");
  names.add(name);entries.push([name,value]);
 }
 const attributes=Object.fromEntries(entries);
 if(!ownProductSchema.shape.attributes.safeParse(attributes).success)throw new Error("商品属性超过保存限制，请减少属性数量或文字长度。");
 return attributes;
}
export function prepareOwnProduct(input:{title:string;brand:string;description:string;images:string[];mode:"single"|"multiple";rows:VariantRow[];attributes?:ProductAttributeRow[]}){
 const {title,brand,description,images,mode,rows}=input;
 if(mode==="multiple"&&!rows.length)throw new Error("请生成至少一个变体。");
 const active=mode==="single"&&!rows[0]?.sku.trim()&&!rows[0]?.existing?undefined:rows;
 const variants=active?.map((row,index)=>{
  if(!row.sku.trim()&&!row.existing||!row.price.trim()||!row.stock.trim()||!row.currency.trim())throw new Error(`第 ${index+1} 个变体：请填写SKU、币种、价格和库存。`);
  if(row.image&&!images.includes(row.image)&&!row.originalImages?.includes(row.image))throw new Error(`第 ${index+1} 个变体的图片已从商品图片中移除，请重新选择。`);
  const skuImages=variantImages(row);
  return {sourceId:row.id,title:row.title??(Object.values(row.attributes).join(" / ")||title),sku:row.existing?row.sku:row.sku.trim(),attributes:row.attributes,currency:row.currency.trim(),price:Number(row.price),stock:Number(row.stock),...(skuImages.length?{images:skuImages}:{})};
 });
 const attributes=prepareProductAttributes(input.attributes??[]);
 const result=ownProductSchema.safeParse({title:title.trim(),brand:brand.trim(),description,images,...(Object.keys(attributes).length?{attributes}:{}),...(variants?{variants}:{})});
 if(!result.success){const issue=result.error.issues[0];throw new Error(issue.code==="custom"?issue.message:"请检查商品名称、图片链接、币种、数值和资料长度。")}
 return result.data;
}

function variantImages(row:VariantRow){return row.originalImages?.[0]===row.image?[...row.originalImages]:row.image?[row.image]:[]}
export function removeVariantImage(row:VariantRow,url:string):VariantRow{
 const originalImages=row.originalImages?.filter(image=>image!==url);
 return {...row,originalImages,image:row.image===url?originalImages?.[0]??"":row.image};
}
