/* eslint-disable @next/next/no-img-element */
"use client";
import {useState} from "react";
import {useQuery,useQueryClient} from "@tanstack/react-query";
import Link from "next/link";
import {ConsolePage,ConsoleState,ConsoleToolbar} from "@/components/workbench/console/console-page";
import {WorkbenchViewNav,workbenchViewPage} from "@/components/workbench/console/workbench-view-nav";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {listMarket,readMarket,type MarketScope} from "@/lib/api/supply-market";
import {MarketBoundary,MarketReadError,CursorButtons,useMarketCommands} from "./shared";
import styles from "./market-product.module.css";
import {OfficialMarketActions} from "./official-access";
import {MarketFiltersForm,type MarketCriteria} from "./market-filters";
import {MarketCard,MarketMerchant} from "./market-card";
import {formatMarketPrice as priceText} from "./market-price";
import type {MarketView} from "@/lib/contracts/supply-market";
type Channel="official"|"selected";
const channels={official:{title:"硕米自营",href:"/workbench/supply/official"},selected:{title:"硕米优选",href:"/workbench/supply/selected"}};
function MyDataLink(){return <Button variant="outline" asChild><Link href="/workbench/data/mine" prefetch={false}>我的数据</Link></Button>}
function ChannelNavigation({channel}:{channel:Channel}){return <WorkbenchViewNav label="货源渠道" className="mb-6">{(Object.keys(channels) as Channel[]).map(key=><Link key={key} href={channels[key].href} prefetch={false} aria-current={channel===key?"page":undefined}>{channels[key].title}</Link>)}</WorkbenchViewNav>}
function ProductPrice({variants,note}:{variants?:readonly {currency?:string;price?:number}[];note?:string}){
 const groups=new Map<string,number[]>();
 for(const variant of variants??[]){if(variant.currency&&variant.price!==undefined){const values=groups.get(variant.currency)??[];values.push(variant.price);groups.set(variant.currency,values)}}
 return <div className={styles.price} role="group" aria-label="商品价格"><p className={styles.note}>价格</p>{groups.size?[...groups].map(([currency,values])=>{const low=Math.min(...values),high=Math.max(...values);return <p key={currency} className={styles.priceValue}>{low===high?priceText(low,currency):`${priceText(low,currency)} – ${priceText(high,currency)}`}</p>}):<p className={styles.priceDescription}>{note||"暂未提供"}</p>}</div>
}
export function SupplyMarketPage({channel}:{channel:"official"|"selected"}){return <MarketBoundary>{scope=><MarketList scope={scope} channel={channel}/>}</MarketBoundary>}
function MarketList({scope,channel}:{scope:MarketScope;channel:"official"|"selected"}){
 const [criteria,setCriteria]=useState<MarketCriteria>({keyword:""}),[after,setAfter]=useState<string>(),[adding,setAdding]=useState<string>();
 const commands=useMarketCommands(scope),client=useQueryClient();
 const filter=criteria.keyword!==""||Object.keys(criteria).length>1;
 const prefix=["supply-market",scope.userId,scope.organizationId,"releases"];
 const q=useQuery({queryKey:[...prefix,channel,criteria,after],queryFn:({signal})=>listMarket(scope,{kind:channel,...criteria,after},signal),retry:false});
 async function add(product:MarketView){
  if(commands.locked||product.added!==false)return;
  setAdding(product.id);
  try{const receipt=await commands.execute({action:"select_release",id:product.id,expectedRevision:product.revision});if(receipt?.releaseId===product.id&&receipt.collection){client.setQueriesData<Awaited<ReturnType<typeof listMarket>>>({queryKey:prefix},page=>page?{...page,items:page.items.map(item=>item.id===product.id?{...item,added:true}:item)}:page)}}finally{setAdding(undefined)}
 }
 return <ConsolePage className={workbenchViewPage} title={channels[channel].title} breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:channels[channel].title}]} description="挑选合适的货源，加入我的数据后继续准备商品上架。" actions={<><MyDataLink/>{channel==="official"?<OfficialMarketActions/>:null}</>}><ChannelNavigation channel={channel}/><ConsoleToolbar><MarketFiltersForm subject={scope.userId} onApply={next=>{setCriteria(next);setAfter(undefined)}}/></ConsoleToolbar>
  {commands.notice}
  {q.isPending?<ConsoleState kind="loading" title="正在加载商品"/>:q.error?<MarketReadError error={q.error} retry={()=>void q.refetch()}/>:q.data.items.length===0?<ConsoleState kind="empty" title={filter?"没有找到匹配商品":"暂无商品"}>{filter?"试试其他关键词，或清除筛选查看全部商品。":"这里暂时还没有商品，请稍后再来看看。"}</ConsoleState>:<div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">{q.data.items.map(p=><MarketCard key={p.id} product={p} locked={!!commands.locked} adding={adding===p.id} onAdd={()=>void add(p)}/>)}</div>}
  {q.data?<CursorButtons after={after} next={q.data.nextCursor} total={q.data.total} onAfter={setAfter}/>:null}
 </ConsolePage>;
}
export function SupplyProductPage({id}:{id:string}){return <MarketBoundary>{scope=><MarketProduct scope={scope} id={id}/>}</MarketBoundary>}
function MarketProduct({scope,id}:{scope:MarketScope;id:string}){
 const q=useQuery({queryKey:["supply-market",scope.userId,scope.organizationId,"release",id],queryFn:({signal})=>readMarket(scope,id,signal),retry:false}),commands=useMarketCommands(scope);
 if(q.isPending)return <ConsoleState kind="loading" title="正在加载商品"/>;if(q.error)return <MarketReadError error={q.error} retry={()=>void q.refetch()}/>;const p=q.data;
 const channel=channels[p.channel],joined=p.added===true||(commands.saved?.releaseId===p.id&&!!commands.saved?.collection);
 return <ConsolePage className={`${workbenchViewPage} ${styles.page}`} title={p.product.title} breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:channel.title,href:channel.href},{label:"商品详情"}]} description="查看商品和供货信息，加入我的数据后可继续编辑" actions={<><Button variant="ghost" asChild><Link href={channel.href} prefetch={false}>返回{channel.title}</Link></Button><MyDataLink/></>}>
  <div className={styles.content}>
   {commands.notice}
   {joined?<Card className={styles.saved} role="status"><div>已加入我的数据 <Link href="/workbench/data/mine" prefetch={false}>打开我的数据</Link></div><p>可在“我的数据”中转入待上传，继续完善商品资料</p></Card>:null}
   <div className={styles.overview}>
    <Card className={styles.gallery}>
     <img src={p.product.images[0]} alt={p.product.title+" 商品图 1"} className={styles.mainImage}/>
     {p.product.images.length>1?<div className={styles.moreImages}>{p.product.images.slice(1).map((src,i)=><img key={src} src={src} alt={p.product.title+" 商品图 "+(i+2)} loading="lazy"/>)}</div>:null}
    </Card>
    <Card className={styles.supply}>
     <h2>供货信息</h2>
     {p.channel==="selected"?<MarketMerchant merchant={p.merchant}/>:null}
     <dl className={styles.metrics}>
      <div><dt>库存</dt><dd>{p.supply.stock}<span>件</span></dd></div>
      <div><dt>起订量</dt><dd>{p.supply.minimumQuantity}<span>件</span></dd></div>
      <div><dt>供货周期</dt><dd>{p.supply.leadDays}<span>天</span></dd></div>
     </dl>
     <dl className={styles.fields}>
      <div><dt>发货地</dt><dd>{p.supply.province} {p.supply.city}</dd></div>
      <div><dt>生产能力</dt><dd>{p.supply.capacity||"未填写"}</dd></div>
      {p.supply.priceNote?<div><dt>价格说明</dt><dd>{p.supply.priceNote}</dd></div>:null}
      {p.supply.afterSaleNote?<div><dt>售后说明</dt><dd>{p.supply.afterSaleNote}</dd></div>:null}
     </dl>
     <p className={styles.note}>库存由供货方提供，实际供货以确认结果为准</p>
     <div className={styles.selection}>
      <ProductPrice variants={p.product.variants} note={p.supply.priceNote}/>
      <Button className={styles.selectButton} disabled={!!commands.locked||joined||p.added!==false} onClick={()=>void commands.execute({action:"select_release",id:p.id,expectedRevision:p.revision})}>加入我的数据</Button>
      <p className={`${styles.note} ${styles.selectionNote}`}>你可以先保存商品、准备上架资料，暂不支持在线下单</p>
     </div>
    </Card>
   </div>
   <Card className={styles.details}>
    <h2>商品详情</h2>
    {p.product.description?<p className={styles.description}>{p.product.description}</p>:null}
    <dl className={styles.attributes}>
     {p.product.brand?<div><dt>品牌</dt><dd>{p.product.brand}</dd></div>:null}
     {Object.entries(p.product.attributes??{}).map(([k,v])=><div key={k}><dt>{k}</dt><dd>{v}</dd></div>)}
    </dl>
    {p.product.variants?.map((v,i)=><div key={i} className={styles.variant}>
     <div className={styles.variantHeading}><h3>{v.title||v.sku||"款式 "+(i+1)}</h3>{v.currency?<p className="tabular-nums">{v.currency} {v.price}</p>:null}</div>
     <dl className={styles.attributes}>{Object.entries(v.attributes??{}).map(([k,x])=><div key={k}><dt>{k}</dt><dd>{x}</dd></div>)}</dl>
     {v.images?.length?<div className={styles.variantImages}>{v.images.map(src=><img key={src} src={src} alt="款式图" loading="lazy"/>)}</div>:null}
    </div>)}
   </Card>
  </div>
 </ConsolePage>;
}
