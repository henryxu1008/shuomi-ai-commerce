/* eslint-disable @next/next/no-img-element */
"use client";
import {useEffect,useId,useRef,useState,type KeyboardEvent} from "react";
import {useQuery} from "@tanstack/react-query";
import Link from "next/link";
import {useRouter} from "next/navigation";
import {ConsolePage,ConsoleState,ConsoleToolbar} from "@/components/workbench/console/console-page";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {Textarea} from "@/components/ui/textarea";
import {listOwnProducts} from "@/lib/api/product-collection";
import {listMarketRecords,listRecordReleases,readMarketChoice,type MarketScope} from "@/lib/api/supply-market";
import {MarketBoundary,MarketReadError,CursorButtons,marketStages,useMarketCommands} from "./shared";
import {SupplyApplicationOverview} from "./application-overview";
import {SelectedApplicationForm} from "./selected-application-form";
import {OfficialPublishingAccess,OfficialMarketActions} from "./official-access";
import {CatalogConnectionForm} from "./catalog-connection-form";
import {isShippingLocation, SupplyLocationFields, useShippingLocations} from "./supply-location-fields";
type ApplicationView="progress"|"records";
const applicationKinds=[{id:"selected",label:"优选申请"},{id:"connection",label:"货盘对接"},{id:"official",label:"自营商品"}] as const;
export function SupplyApplicationsPage({admin=false,expectedUserId,view,connection=false,initialKind}:{admin?:boolean;expectedUserId?:string;view?:ApplicationView;connection?:boolean;initialKind?:"official"}){return <MarketBoundary admin={admin} expectedUserId={expectedUserId}>{scope=>!admin&&!view&&!connection?<SupplyApplicationOverview scope={scope}/>:<Applications key={view??initialKind??"all"} scope={scope} admin={admin} initialKind={initialKind} view={view} connection={!admin&&connection}/>}</MarketBoundary>}
function Applications({scope,admin,view,connection=false,initialKind}:{scope:MarketScope;admin:boolean;view?:ApplicationView;connection?:boolean;initialKind?:"official"}){
 const [applying,setApplying]=useState(false);
 const panelId=useId(),tabRefs=useRef<Array<HTMLButtonElement|null>>([]);
 const [endedFilter,setEnded]=useState(false),[kindFilter,setKind]=useState<"selected"|"official"|"connection">(initialKind??"selected"),[after,setAfter]=useState<string>();
 const selectedView=!admin?view:undefined, ended=selectedView?selectedView==="records":endedFilter, kind=connection?"connection":selectedView?"selected":kindFilter;
 const title=connection?(selectedView==="records"?"对接记录":"对接进度"):admin?kind==="official"?"自营商品管理":"供应发布与人工评估":selectedView==="progress"?"申请进度":selectedView==="records"?"申请记录":"优选申请";
 function changeKind(next:typeof kindFilter){setKind(next);setAfter(undefined)}
 function tabKey(event:KeyboardEvent<HTMLButtonElement>,index:number){
  const next=event.key==="ArrowRight"?(index+1)%applicationKinds.length:event.key==="ArrowLeft"?(index-1+applicationKinds.length)%applicationKinds.length:event.key==="Home"?0:event.key==="End"?applicationKinds.length-1:undefined;
  if(next===undefined)return;event.preventDefault();changeKind(applicationKinds[next].id);tabRefs.current[next]?.focus();
 }
 const q=useQuery({queryKey:["supply-market",scope.userId,scope.organizationId,"records",admin,kind,kind==="official"?false:ended,after],queryFn:({signal})=>listMarketRecords(scope,{kind,ended:kind==="official"?false:ended,after},admin,signal),retry:false});
 return <ConsolePage title={title} description={connection?"查看本人货盘对接申请的人工评估与方案进展。":admin?kind==="official"?"查看自营草稿与实际发布状态，打开商品后继续预览和发布。":"按实际资质和线下合作评估，审批后明确发布。":selectedView==="records"?"查看本人已结束的优选申请及最终审核结果。":selectedView==="progress"?"查看本人进行中的优选申请与人工评估进展。":"提交本人自有且已有实际优化结果的商品，查看人工评估进展。"} breadcrumbs={admin?undefined:[{label:"供应市场",href:"/workbench/supply"},{label:connection?"货盘集成":"优选申请",...(selectedView?{href:connection?"/workbench/supply/catalogs":"/workbench/supply/applications"}:{})},...(selectedView?[{label:title}]:[])]} actions={admin&&kind==="official"?<OfficialMarketActions showManage={false}/>:!admin?connection?<Button onClick={()=>setApplying(true)}>发起申请</Button>:<Link href="/workbench/supply/applications/new" prefetch={false}>发起申请</Link>:null}>
  <ConsoleToolbar><div className="flex flex-wrap items-center gap-3">{selectedView?<><Button asChild variant={ended?"outline":"default"}><Link href={connection?"/workbench/supply/catalogs/apply?view=progress":"/workbench/supply/applications/progress"} prefetch={false} aria-current={!ended?"page":undefined}>申请进度</Link></Button><Button asChild variant={ended?"default":"outline"}><Link href={connection?"/workbench/supply/catalogs/apply?view=records":"/workbench/supply/applications/records"} prefetch={false} aria-current={ended?"page":undefined}>申请记录</Link></Button></>:<><div role="tablist" aria-label="申请类型" className="flex gap-1 border-b border-border">{applicationKinds.map((entry,index)=><Button key={entry.id} ref={element=>{tabRefs.current[index]=element}} role="tab" id={panelId+"-"+entry.id} aria-controls={panelId} aria-selected={kind===entry.id} tabIndex={kind===entry.id?0:-1} variant="ghost" className={"rounded-none border-b-2 "+(kind===entry.id?"border-primary text-primary":"border-transparent text-muted-foreground")} onClick={()=>changeKind(entry.id)} onKeyDown={event=>tabKey(event,index)}>{entry.label}</Button>)}</div>{!(admin&&kind==="official")?<><Button variant={ended?"outline":"default"} onClick={()=>{setEnded(false);setAfter(undefined)}}>进行中</Button><Button variant={ended?"default":"outline"} onClick={()=>{setEnded(true);setAfter(undefined)}}>已结束</Button></>:null}</>}</div></ConsoleToolbar>
  <div role={admin?"tabpanel":undefined} id={admin?panelId:undefined} aria-labelledby={admin?panelId+"-"+kind:undefined} tabIndex={admin?0:undefined}>
  {q.isPending?<ConsoleState kind="loading" title={admin&&kind==="official"?"正在读取自营商品":"正在读取申请"}/>:q.error?<MarketReadError error={q.error} retry={()=>void q.refetch()}/>:q.data.items.length===0?<ConsoleState kind="empty" title={admin&&kind==="official"?"暂无自营商品":"暂无申请记录"}/>:<div className="space-y-3">{q.data.items.map(r=><Card key={r.id} className="flex items-center justify-between gap-3 p-4"><div><h2>{r.product?.title||r.connection?.name}</h2><p>{admin&&r.kind==="official"?<OfficialReleaseStatus scope={scope} id={r.id}/>:marketStages[r.stage]} · {new Date(r.createdAt).toLocaleString("zh-CN")}</p>{r.stage==="APPROVED"?<p>已通过人工评估；市场发布为单独操作。</p>:null}</div><Link href={(admin?"/workbench/supply/operator/":"/workbench/supply/applications/")+r.id} prefetch={false}>{admin&&r.kind==="official"?"查看并继续发布":"查看进展"}</Link></Card>)}</div>}
  {connection&&applying?<CatalogConnectionForm scope={scope} onClose={()=>setApplying(false)}/>:null}
  {q.data?<CursorButtons after={after} next={q.data.nextCursor} total={q.data.total} onAfter={setAfter}/>:null}
  </div>
 </ConsolePage>;
}
export function SupplyApplicationForm({official=false,initialItemId}:{official?:boolean;initialItemId?:string}){return <MarketBoundary>{scope=>official?<OfficialPublishingAccess><OfficialApplicationForm key={initialItemId??"choose"} scope={scope} initialItemId={initialItemId}/></OfficialPublishingAccess>:<SelectedApplicationForm scope={scope}/>}</MarketBoundary>}
function OfficialApplicationForm({scope,initialItemId}:{scope:MarketScope;initialItemId?:string}){
 const router=useRouter();
 const locations=useShippingLocations(scope.userId);
 const [keyword,setKeyword]=useState(""),[after,setAfter]=useState<string>(),[item,setItem]=useState(initialItemId??""),[disclosure,setDisclosure]=useState(false);
 const [supply,setSupply]=useState({stock:0,capacity:"",minimumQuantity:1,leadDays:0,province:"",city:"",priceNote:"",afterSaleNote:""});
 const own=useQuery({queryKey:["supply-market",scope.userId,scope.organizationId,"own",keyword,after],queryFn:({signal})=>listOwnProducts(scope,{keyword,after,limit:20},signal),retry:false});
 const choice=useQuery({queryKey:["supply-market",scope.userId,scope.organizationId,"choice",item],queryFn:({signal})=>readMarketChoice(scope,item,signal),enabled:!!item,retry:false}),commands=useMarketCommands(scope);
 const current=!choice.isFetching&&!choice.error&&choice.data?.selection.itemId===item?choice.data:undefined;
 useEffect(()=>{if(commands.savedAction==="create_official_draft"&&commands.saved?.recordId)router.push("/workbench/supply/operator/"+commands.saved.recordId)},[commands.saved,commands.savedAction,router]);
 const blocked=commands.locked||!!commands.saved||!current?.product||!disclosure||!isShippingLocation(locations.data?.country,supply);
 return <ConsolePage title="添加自营商品" description="选择商品 → 填写供货信息 → 预览并确认发布。保存草稿不会自动公开。" breadcrumbs={[{label:"硕米自营",href:"/workbench/supply/official"},{label:"添加自营商品"}]} actions={<Button variant="outline" asChild><Link href="/workbench/supply/official" prefetch={false}>返回硕米自营</Link></Button>}>{commands.notice}{commands.saved?.recordId?<Card className="p-4" role="status">草稿已保存，尚未公开。<Link href={"/workbench/supply/operator/"+commands.saved.recordId} prefetch={false}>继续预览与发布</Link></Card>:null}
  <Card className="space-y-4 p-6"><div className="flex flex-wrap items-center justify-between gap-3"><h2>1. 选择自有商品</h2><Button variant="outline" asChild><Link href="/workbench/data/mine/own/new?continuation=official" prefetch={false}>新建商品</Link></Button></div><p className="text-sm text-muted-foreground">从本人的自有商品库选择，或新建商品后继续。</p><Input aria-label="筛选本人商品" placeholder="商品关键词" value={keyword} maxLength={80} onChange={e=>{setKeyword(e.target.value);setAfter(undefined)}}/>{own.isPending?<p>正在读取商品</p>:own.error?<MarketReadError error={own.error} retry={()=>void own.refetch()}/>:<><Select aria-label="自有商品" value={item} disabled={commands.locked} onChange={e=>setItem(e.target.value)}><option value="">请选择商品</option>{initialItemId&&current?.product&&!own.data.items.some(i=>i.id===initialItemId)?<option value={initialItemId}>{current.product.title}</option>:null}{own.data.items.map(i=><option key={i.id} value={i.id}>{i.title||i.source.productKey}</option>)}</Select><CursorButtons after={after} next={own.data.nextCursor} total={own.data.total} onAfter={setAfter}/></>}
   {choice.isFetching?<p>正在核对商品版本与优化记录</p>:choice.error?<MarketReadError error={choice.error} retry={()=>void choice.refetch()}/>:current?.product?<div className="flex gap-4"><img src={current.product.images[0]} alt={current.product.title} className="h-24 w-24 object-contain"/><div><h3>{current.product.title}</h3><p>{current.optimized?"已应用商品优化":"自有商品原始资料"}</p></div></div>:current?<p role="status">商品资料不符合发布要求，请先完善自有商品。</p>:null}
  </Card>
  <form className="space-y-4" onSubmit={e=>{e.preventDefault();if(blocked||!current?.product)return;void commands.execute({action:"create_official_draft",selection:current.selection,supply,disclosure:true})}}><Card className="space-y-4 p-6"><h2>2. 填写供货信息</h2><div className="grid gap-4 sm:grid-cols-2">{[{key:"stock",label:"现有库存（件）",min:0,max:1e9},{key:"minimumQuantity",label:"最低起订量（件）",min:1,max:1e9},{key:"leadDays",label:"供货周期（天）",min:0,max:3650}].map(f=><label key={f.key}>{f.label}<Input type="number" required min={f.min} max={f.max} value={supply[f.key as "stock"|"minimumQuantity"|"leadDays"]} onChange={e=>setSupply({...supply,[f.key]:Number(e.target.value)})}/></label>)}<label>生产能力<Input value={supply.capacity} required={supply.stock===0} maxLength={4000} onChange={e=>setSupply({...supply,capacity:e.target.value})}/></label><SupplyLocationFields locations={locations} province={supply.province} city={supply.city} onChange={location=>setSupply(previous=>({...previous,...location}))}/></div><label className="block">供货价格说明<Textarea value={supply.priceNote} maxLength={4000} onChange={e=>setSupply({...supply,priceNote:e.target.value})}/></label><label className="block">售后说明<Textarea value={supply.afterSaleNote} maxLength={4000} onChange={e=>setSupply({...supply,afterSaleNote:e.target.value})}/></label>
   <label className="flex gap-2"><input type="checkbox" checked={disclosure} onChange={e=>setDisclosure(e.target.checked)}/>我确认这是本人/企业自有商品，并允许平台评估；明确发布后，列明的商品字段及供货声明可供其他获授权企业查看。</label><Button type="submit" disabled={blocked}>{commands.busy?"正在保存…":"保存草稿并预览"}</Button><p className="text-sm text-muted-foreground">下一步核对公开内容，再由当前账号确认发布；也可以保存后稍后继续。</p>
  </Card></form>
 </ConsolePage>;
}

function OfficialReleaseStatus({scope,id}:{scope:MarketScope;id:string}){
 const q=useQuery({queryKey:["supply-market",scope.userId,scope.organizationId,"record-releases",true,id,undefined],queryFn:({signal})=>listRecordReleases(scope,id,undefined,true,signal),retry:false});
 return <span>{q.isPending?"正在读取发布状态":q.error?"发布状态读取失败":q.data.items.some(r=>r.active)?"已发布":q.data.nextCursor?"打开详情核对发布状态":q.data.total>0?"已撤销发布":"草稿 · 待发布"}</span>;
}
