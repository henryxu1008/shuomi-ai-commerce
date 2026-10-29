import {parseSourcePriceFact,type CollectionDetail} from "@/lib/contracts/product-collection";

type Price=NonNullable<ReturnType<typeof parseSourcePriceFact>>;
export function PriceText({price,showUnknown=false}:{price:Price;showUnknown?:boolean}){
  const label=(p:Price)=>p.condition==='new'?'新品':p.condition==='used'?'二手':showUnknown?'状态未记录':null;
  const render=(p:Price)=><>{p.amount} · {p.currency??"币种待确认"} · {p.minQuantity?`起订数量 ${p.minQuantity}`:"起订数量未提供"}{label(p)?` · ${label(p)}`:""}{p.conditionNote?` · ${p.conditionNote}`:""}</>;
  return <>{render(price)}{price.alternative?<>；{render(price.alternative)}</>:null}</>;
}
export function SourceQuotes({quotes,showUnknown=false}:{quotes:CollectionDetail["sourceQuotes"];showUnknown?:boolean}){
  if(!quotes||quotes.state==="not_applicable")return null;
  return <section aria-label="来源报价（参考）" className="space-y-2 rounded-xl border border-border p-4">
    <h4 className="font-semibold">来源报价（参考）</h4>
    <p className="text-sm text-muted-foreground">供货来源参考信息；本店销售价和库存需另行填写。</p>
    {quotes.state==="unavailable"?<p className="text-sm">来源报价暂未能读取。</p>:quotes.prices.length?<ul className="space-y-1 text-sm">{quotes.prices.map((price,index)=><li key={index}><PriceText price={price} showUnknown={showUnknown}/></li>)}</ul>:<p className="text-sm">本次未记录商品级报价，规格报价见规格资料。</p>}
  </section>;
}
export function VariantSourcePrice({attributes,showUnknown=false}:{attributes:NonNullable<CollectionDetail["product"]["variants"]>[number]["attributes"];showUnknown?:boolean}){
  const facts=attributes?.filter(attribute=>attribute.name==="source.price.fact")??[];
  if(!facts.length)return null;
  const price=facts.length===1?parseSourcePriceFact(facts[0]?.value):null;
  return <span className="ml-3">{price?<>来源金额 <PriceText price={price} showUnknown={showUnknown}/></>:"来源规格报价无法识别，需人工确认。"}</span>;
}
