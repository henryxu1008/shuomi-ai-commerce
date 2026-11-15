/* eslint-disable @next/next/no-img-element */
"use client";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import Link from "next/link";
import {ConsolePage,ConsoleState,ConsoleToolbar} from "@/components/workbench/console/console-page";
import {WorkbenchViewNav,workbenchViewPage} from "@/components/workbench/console/workbench-view-nav";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {listMarket,readMarket,type MarketScope} from "@/lib/api/supply-market";
import {MarketBoundary,MarketReadError,CursorButtons,useMarketCommands} from "./shared";
type Channel="official"|"selected";
const channels={official:{title:"硕米自营",href:"/workbench/supply/official"},selected:{title:"硕米优选",href:"/workbench/supply/selected"}};
function MyDataLink(){return <Button variant="outline" asChild><Link href="/workbench/data/mine" prefetch={false}>我的数据</Link></Button>}
function ChannelNavigation({channel}:{channel:Channel}){return <WorkbenchViewNav label="货源渠道" className="mb-6">{(Object.keys(channels) as Channel[]).map(key=><Link key={key} href={channels[key].href} prefetch={false} aria-current={channel===key?"page":undefined}>{channels[key].title}</Link>)}</WorkbenchViewNav>}
export function SupplyMarketPage({channel}:{channel:"official"|"selected"}){return <MarketBoundary>{scope=><MarketList scope={scope} channel={channel}/>}</MarketBoundary>}
function MarketList({scope,channel}:{scope:MarketScope;channel:"official"|"selected"}){
 const [keyword,setKeyword]=useState(""),[filter,setFilter]=useState(""),[after,setAfter]=useState<string>();
 const q=useQuery({queryKey:["supply-market",scope.userId,scope.organizationId,"releases",channel,filter,after],queryFn:({signal})=>listMarket(scope,{kind:channel,keyword:filter,after},signal),retry:false});
 return <ConsolePage className={workbenchViewPage} title={channels[channel].title} breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:channels[channel].title}]} description="浏览已发布的商品与供货说明，加入我的数据后继续准备店铺刊登。" actions={<MyDataLink/>}><ChannelNavigation channel={channel}/><ConsoleToolbar><form className="flex w-full flex-wrap gap-2" onSubmit={e=>{e.preventDefault();setFilter(keyword);setAfter(undefined)}}><Input className="min-w-0 max-w-sm flex-1" aria-label="商品关键词" value={keyword} maxLength={80} onChange={e=>setKeyword(e.target.value)} placeholder="搜索商品名称"/><Button type="submit">搜索</Button>{filter?<Button type="button" variant="ghost" onClick={()=>{setKeyword("");setFilter("");setAfter(undefined)}}>清除筛选</Button>:null}</form></ConsoleToolbar>
  {q.isPending?<ConsoleState kind="loading" title="正在读取商品"/>:q.error?<MarketReadError error={q.error} retry={()=>void q.refetch()}/>:q.data.items.length===0?<ConsoleState kind="empty" title={filter?"没有找到匹配商品":"暂无已发布商品"}>{filter?"试试其他关键词，或清除筛选查看全部商品。":"商品明确发布后会出现在这里，可查看详情并加入我的数据。"}</ConsoleState>:<div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">{q.data.items.map(p=><Card key={p.id} className="gap-0 overflow-hidden p-0"><img src={p.product.images[0]} alt={p.product.title} loading="lazy" className="aspect-[4/3] w-full object-contain bg-muted"/><div className="space-y-3 p-5"><h2 className="font-semibold leading-relaxed break-words">{p.product.title}</h2><p className="text-sm text-muted-foreground">{p.supply.province} · {p.supply.city}</p><div className="flex flex-wrap gap-x-5 gap-y-1 border-y py-3 text-sm"><span>起订 {p.supply.minimumQuantity} 件</span><span>供货周期 {p.supply.leadDays} 天</span></div><p className="text-sm text-muted-foreground">{p.supply.stock>0?"声明库存 "+p.supply.stock+" 件":p.supply.capacity}</p><Button asChild variant="outline"><Link href={"/workbench/supply/products/"+p.id} prefetch={false}>查看商品</Link></Button></div></Card>)}</div>}
  {q.data?<CursorButtons after={after} next={q.data.nextCursor} total={q.data.total} onAfter={setAfter}/>:null}
 </ConsolePage>;
}
export function SupplyProductPage({id}:{id:string}){return <MarketBoundary>{scope=><MarketProduct scope={scope} id={id}/>}</MarketBoundary>}
function MarketProduct({scope,id}:{scope:MarketScope;id:string}){
 const q=useQuery({queryKey:["supply-market",scope.userId,scope.organizationId,"release",id],queryFn:({signal})=>readMarket(scope,id,signal),retry:false}),commands=useMarketCommands(scope);
 if(q.isPending)return <ConsoleState kind="loading" title="正在读取商品"/>;if(q.error)return <MarketReadError error={q.error} retry={()=>void q.refetch()}/>;const p=q.data;
 const channel=channels[p.channel];
 return <ConsolePage className={workbenchViewPage} title={p.product.title} breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:channel.title,href:channel.href},{label:"商品详情"}]} description="仅包含明确发布的商品字段和供货声明。加入后会保存该发布版本。" actions={<><Button variant="outline" asChild><Link href={channel.href} prefetch={false}>返回{channel.title}</Link></Button><MyDataLink/></>}>{commands.notice}{commands.saved?.collection?<Card className="p-4" role="status">已选入我的数据。<Link href="/workbench/data/mine" prefetch={false}>打开我的数据</Link><p>在我的数据中将该批次转入“商品刊登”，继续适配和上传。</p></Card>:null}
  <div className="grid gap-6 lg:grid-cols-2"><Card className="p-4"><div className="grid grid-cols-2 gap-3">{p.product.images.map((src,i)=><img key={src} src={src} alt={p.product.title+" 商品图 "+(i+1)} className="aspect-square w-full object-contain"/>)}</div></Card><Card className="space-y-4 p-6"><h2>供货声明</h2><p>发货地：{p.supply.province} {p.supply.city}</p><p>库存：{p.supply.stock} 件；产能：{p.supply.capacity||"未声明"}</p><p>起订量：{p.supply.minimumQuantity} 件；供货周期：{p.supply.leadDays} 天</p><p>{p.supply.priceNote}</p><p>{p.supply.afterSaleNote}</p><Button disabled={commands.locked||!!commands.saved?.collection} onClick={()=>void commands.execute({action:"select_release",id:p.id,expectedRevision:p.revision})}>加入我的数据</Button><p className="text-sm text-muted-foreground">本批不提供在线采购、付款或结算。</p></Card></div>
  <Card className="space-y-3 p-6"><h2>商品详情</h2><p className="whitespace-pre-wrap">{p.product.description}</p>{p.product.brand?<p>品牌：{p.product.brand}</p>:null}<dl className="grid grid-cols-2 gap-2">{Object.entries(p.product.attributes??{}).map(([k,v])=><div key={k}><dt>{k}</dt><dd>{v}</dd></div>)}</dl>{p.product.variants?.map((v,i)=><div key={i} className="border-t pt-3"><h3>{v.title||v.sku||"款式 "+(i+1)}</h3><p>{Object.entries(v.attributes??{}).map(([k,x])=>k+"："+x).join("，")}</p>{v.currency?<p>{v.currency} {v.price}</p>:null}<div className="flex gap-2">{v.images?.map(src=><img key={src} src={src} alt="款式图" className="h-24 w-24 object-contain"/>)}</div></div>)}</Card>
 </ConsolePage>;
}
