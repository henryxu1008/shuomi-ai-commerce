import {parseSourcePriceFact,type CollectionDetail} from "@/lib/contracts/product-collection";

type Price=NonNullable<ReturnType<typeof parseSourcePriceFact>>;
function PriceText({price}:{price:Price}){
  return <>{price.amount} · {price.currency??"币种待确认"} · {price.minQuantity?`起订数量 ${price.minQuantity}`:"起订数量未提供"}</>;
}
export function SourceQuotes({quotes}:{quotes:CollectionDetail["sourceQuotes"]}){
  if(!quotes||quotes.state==="not_applicable")return null;
  return <section aria-label="来源报价（参考）" className="space-y-2 rounded-xl border border-border p-4">
    <h4 className="font-semibold">来源报价（参考）</h4>
    <p className="text-sm text-muted-foreground">供货来源参考信息；本店销售价和库存需另行填写。</p>
    {quotes.state==="unavailable"?<p className="text-sm">来源报价暂未能读取。</p>:quotes.prices.length?<ul className="space-y-1 text-sm">{quotes.prices.map((price,index)=><li key={index}><PriceText price={price}/></li>)}</ul>:<p className="text-sm">本次未记录商品级报价，规格报价见规格资料。</p>}
  </section>;
}
export function VariantSourcePrice({attributes}:{attributes:NonNullable<CollectionDetail["product"]["variants"]>[number]["attributes"]}){
  const facts=attributes?.filter(attribute=>attribute.name==="source.price.fact")??[];
  if(!facts.length)return null;
  const price=facts.length===1?parseSourcePriceFact(facts[0]?.value):null;
  return <span className="ml-3">{price?<>来源金额 <PriceText price={price}/></>:"来源规格报价无法识别，需人工确认。"}</span>;
}
