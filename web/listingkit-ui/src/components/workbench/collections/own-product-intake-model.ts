import type {IntakeCandidate} from "@/lib/contracts/own-product-intake";
import {newVariantRow,variantCombinationKey,type VariantRow} from "./own-product-model";

export function applyIntakeVariants(candidate:IntakeCandidate, active:VariantRow[], images:string[], acceptImages:boolean, removed:ReadonlySet<string>, confirmedPrices:ReadonlySet<number>=new Set(), confirmedStocks:ReadonlySet<number>=new Set()){
 const incoming=candidate.rows??[];
 const onlyBlank=active.length===1&&!Object.keys(active[0].attributes).length&&!Object.values({
  sku:active[0].sku, currency:active[0].currency, price:active[0].price, stock:active[0].stock, image:active[0].image, ...active[0].packaging,
 }).some(value=>value?.trim());
 const rows=onlyBlank?[]:[...active];
 const indices=new Map(rows.map((row,index)=>[variantCombinationKey(row.attributes),index]));
 const assigned=new Set<string>();
 for(const [suggestionIndex,suggested] of incoming.entries()){
  const attributes=suggested.attributes??{};
  const identity=variantCombinationKey(attributes);
  if(removed.has(identity))continue;
  const index=indices.get(identity);
  const prior=index===undefined?newVariantRow(attributes):rows[index];
  const image=suggested.image===undefined?"":images[suggested.image-1];
  const price=suggested.priceSuggestion;
  // A price is a pair. Never combine a new amount with an existing currency,
  // or infer a currency for an incomplete suggestion.
  const acceptPrice=confirmedPrices.has(suggestionIndex)&&!!price?.currency&&!prior.price.trim()&&!prior.currency.trim();
  if(suggested.image!==undefined&&!image)throw new Error("候选引用的图库已变化，请重新生成。");
  const next={
   ...prior,
   sku:prior.sku.trim()?prior.sku:suggested.sku??"",
   currency:acceptPrice?price!.currency!:prior.currency,
   price:acceptPrice?String(price!.amount):prior.price,
   stock:!prior.stock.trim()&&confirmedStocks.has(suggestionIndex)&&suggested.stockSuggestion?String(suggested.stockSuggestion.quantity):prior.stock,
   image:prior.image|| (acceptImages?image:""),
  };
  if(!prior.image&&next.image)assigned.add(next.image);
  if(index===undefined){indices.set(identity,rows.length);rows.push(next)}else rows[index]=next;
 }
 if(rows.length>1&&rows.some(row=>!Object.keys(row.attributes).length)){
  throw new Error("表单已有单规格 SKU。请先核对单规格资料，再采纳多规格候选。");
 }
 if(rows.length>256||rows.reduce((n,row)=>n+1+Object.keys(row.attributes).length+(row.image?1:0),images.length)>1024){
  throw new Error("候选超过保存限制，请减少规格或图片。");
 }
 return {rows,assigned};
}
