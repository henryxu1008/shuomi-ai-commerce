/* eslint-disable @next/next/no-img-element */
import Link from "next/link";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import type {MarketView} from "@/lib/contracts/supply-market";
import {formatMarketPrice} from "./market-price";
import styles from "./market-card.module.css";

export function MarketMerchant({merchant}:{merchant:MarketView["merchant"]}){
 return <p className={styles.merchant}>{merchant?.state==="verified"?<><span>{merchant.name}</span><span className={styles.verified}>企业已认证</span></>:merchant?.state==="unverified"?"商家待认证":"商家信息暂不可用"}</p>;
}
export function MarketCard({product:p,locked,adding,onAdd}:{product:MarketView;locked:boolean;adding:boolean;onAdd:()=>void}){
 return <Card className={styles.card}>
  <img src={p.product.images[0]} alt={p.product.title} loading="lazy" className={styles.image}/>
  <div className={styles.body}>
   <h2>{p.product.title}</h2>
   {p.channel==="selected"?<MarketMerchant merchant={p.merchant}/>:null}
   <p className={styles.location}>{p.supply.province} · {p.supply.city}</p>
   <div className={styles.facts}><span>起订 {p.supply.minimumQuantity} 件</span><span>供货周期 {p.supply.leadDays} 天</span></div>
   <p className={styles.location}>{p.supply.stock>0?`库存 ${p.supply.stock} 件`:p.supply.capacity}</p>
   <div className={styles.footer}>
    <div className={styles.prices} aria-label="商品价格">{p.prices.length?p.prices.map(price=><p key={price.currency}>{price.minimum===price.maximum?formatMarketPrice(price.minimum,price.currency):`${formatMarketPrice(price.minimum,price.currency)} – ${formatMarketPrice(price.maximum,price.currency)}`}</p>):<p className={styles.noPrice}>价格待确认</p>}</div>
    {p.added===true?<><p className={styles.added}>已加入我的数据</p><Button variant="secondary" asChild><Link href="/workbench/data/mine" prefetch={false}>查看我的数据</Link></Button></>:<Button disabled={locked||p.added!==false} onClick={onAdd}>{adding?"正在加入…":p.added===false?"加入我的数据":"暂不可加入"}</Button>}
    <Link className={styles.details} href={"/workbench/supply/products/"+p.id} prefetch={false}>查看商品</Link>
   </div>
  </div>
 </Card>;
}
