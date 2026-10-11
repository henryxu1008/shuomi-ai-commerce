"use client";
import Image from "next/image";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {ConsolePage,ConsoleState} from "@/components/workbench/console/console-page";
import {ResourceDialog} from "@/components/workbench/resources/resource-dialog";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {ecoRequest,ecoPageSchema,type EcoListing,type EcoScope} from "@/lib/api/ecoservices";
import {EcoBoundary,EcoPagination,ReadFailure,categories,yuan,useEcoCommands,NonPaymentNotice} from "./shared";
import {FileUpload} from "./files";
import "./market.css";

// Figma service-type introductions, never published listings or provider promises.
const introductions = {
 COMPANY_REGISTRATION: {title:"公司注册",description:"为企业设立提供完整办理支持",items:["注册方案咨询","申请资料指导","办理进度跟进"]},
 TRADEMARK_REGISTRATION: {title:"商标注册",description:"为品牌保护提供注册申请支持",items:["注册前查询","申请材料整理","注册进度跟进"]},
 STORE_OPENING: {title:"店铺开通服务",description:"协助完成店铺开通前后的准备工作",items:["开通流程指导","申请资料准备","开通过程协助"]},
 STORE_OPERATION: {title:"店铺代运营",description:"持续协助店铺完成日常经营管理",items:["商品与店铺管理","日常运营执行","经营数据分析"]},
} satisfies Record<EcoListing["category"],{title:string;description:string;items:string[]}>;

function ServiceItems({category,items,introduction=false}:{category:EcoListing["category"];items:string[];introduction?:boolean}){
 const dot=category==="STORE_OPENING"?"cfdd2.svg":category==="STORE_OPERATION"?"03b8d.svg":"eea41.svg";
 return <div className="eco-includes"><strong>{introduction?"服务类型介绍":"包含服务"}</strong><ul>{items.slice(0,3).map((item,i)=><li key={i}><Image width={5} height={5} src={"/figma/ecoservices/"+dot} alt=""/>{item}</li>)}</ul></div>;
}

function EmptyCatalog({group}:{group:string}){
 return <div className="eco-market-grid">{categories.filter(category=>!group||category.group===group).map(category=>{
  const intro=introductions[category.id];
  return <Card className="eco-service-card" key={category.id} data-tone={category.id}><div className="eco-card-top"><span className="eco-tag">{category.group==="enterprise"?"企业服务":"店铺服务"}</span><span>{String(categories.indexOf(category)+1).padStart(2,"0")}</span></div><h2>{intro.title}</h2><p className="eco-card-description">{intro.description}</p><ServiceItems category={category.id} items={intro.items} introduction/><Button variant="outline" disabled>暂无已发布服务</Button></Card>;
 })}</div>;
}
export function EcoservicesMarket({nonPaymentOnly=false}:{nonPaymentOnly?:boolean}={}){return <EcoBoundary>{scope=><Market key={scope.userId+scope.organizationId} scope={scope} nonPaymentOnly={nonPaymentOnly}/>}</EcoBoundary>}
function Market({scope,nonPaymentOnly}:{scope:EcoScope;nonPaymentOnly:boolean}){
 const context=useWorkbenchContext(),[group,setGroup]=useState(""),[page,setPage]=useState(1),[selected,setSelected]=useState<EcoListing|null>(null),[description,setDescription]=useState(""),[files,setFiles]=useState<string[]>([]);
 const commands=useEcoCommands(scope);const canRead=context.permissions.includes("workbench.ecoservices.read"),canPurchase=!nonPaymentOnly&&context.permissions.includes("workbench.ecoservices.purchase");
 const query=useQuery({queryKey:["ecoservices",scope.userId,scope.organizationId,"catalog",group,page,context.permissions.join("|")],queryFn:({signal})=>ecoRequest(scope,"catalog?page="+page+"&pageSize=20"+(group?"&group="+group:""),ecoPageSchema,{signal}),enabled:canRead&&!context.error&&!context.blockingError&&!context.isLoading&&!context.isSwitching});
 const counts=query.data?.counts;const enterprise=(counts?.COMPANY_REGISTRATION??0)+(counts?.TRADEMARK_REGISTRATION??0),shop=(counts?.STORE_OPENING??0)+(counts?.STORE_OPERATION??0);
 return <ConsolePage className="eco-page eco-market-page" title="服务市场" description="围绕企业设立、品牌保护、店铺开通和日常运营，提供专业服务支持。" breadcrumbs={[{label:"生态服务"},{label:"服务市场"}]}>
  {commands.notice}{nonPaymentOnly?<NonPaymentNotice/>:null}
  {!canRead?<ConsoleState kind="unavailable" title="当前身份没有服务市场读取权限"/>:<>
  <Card className="eco-catalog-filter"><div><h2>服务分类</h2><p>{query.isSuccess&&!query.data.listings?.length?"暂无已发布服务，以下仅为服务类型介绍":"按服务类型快速查看当前开放的专业服务"}</p></div><div className="eco-category-tabs">{[["","全部服务",enterprise+shop],["enterprise","企业服务",enterprise],["shop","店铺服务",shop]].map(([id,label,count])=><Button key={id} variant="outline" aria-pressed={group===id} onClick={()=>{setGroup(String(id));setPage(1)}}>{label} <span>{query.data?count:"—"}</span></Button>)}</div></Card>
  {query.isPending?<ConsoleState kind="loading" title="正在读取服务目录"/>:query.error?<ReadFailure error={query.error} retry={()=>void query.refetch()}/>:query.data?.listings?.length?<><div className="eco-market-grid">{query.data.listings.map((listing,index)=>{const category=categories.find(c=>c.id===listing.category)!;return <Card className="eco-service-card" key={listing.id} data-tone={listing.category}><div className="eco-card-top"><span className="eco-tag">{category.group==="enterprise"?"企业服务":"店铺服务"}</span><span>{String((page-1)*20+index+1).padStart(2,"0")}</span></div><h2>{listing.title}</h2><p className="eco-card-description">{listing.description}</p><ServiceItems category={listing.category} items={listing.items}/><p className="eco-provider">{listing.providerName} · {yuan(listing.priceMinor)}起</p><Button variant="outline" onClick={()=>{setSelected(listing);setDescription("");setFiles([])}}>查看详情 →</Button></Card>})}</div><EcoPagination page={page} total={query.data.total} onPage={setPage}/></>:<EmptyCatalog group={group}/>}
  <Card className="eco-market-explainer"><div><small>服务流程</small><h2>查看详情 → 提交需求 → 服务商确认 → 我的服务跟进</h2><p>提交后可在“我的服务”查看报价、交付节点和售后进度。</p></div><div><small>服务说明</small><h2>具体服务信息在详情页明确展示</h2><p>服务地区、费用和交付周期由服务商根据实际能力提供。</p></div></Card></>}
  {selected?<ResourceDialog title={selected.title} onClose={()=>setSelected(null)} locked={commands.locked}><div className="eco-detail"><p>{selected.description}</p><dl><dt>服务商</dt><dd>{selected.providerName}</dd><dt>参考价</dt><dd>{yuan(selected.priceMinor)}，成交以确认报价为准</dd><dt>预计周期</dt><dd>{selected.deliveryDays} 天</dd><dt>服务地区</dt><dd>{selected.regions.join("、")}</dd><dt>适用平台</dt><dd>{selected.platforms?.join("、")||"按报价约定"}</dd></dl><ul>{selected.items.map((v,i)=><li key={i}>{v}</li>)}</ul><form onSubmit={event=>{event.preventDefault();void commands.json("catalog/"+selected.id+"/requests",{description,fileIds:files}).then(()=>setSelected(null)).catch(()=>undefined)}}><fieldset disabled={!canPurchase||commands.locked}><label>具体需求<textarea required maxLength={10000} value={description} onChange={e=>setDescription(e.target.value)}/></label><FileUpload commands={commands} path="requests/files" ids={files} onFiles={setFiles}/><Button type="submit">提交需求</Button></fieldset></form>{!canPurchase?<p>当前身份没有购买或提交需求权限。</p>:null}{commands.notice}</div></ResourceDialog>:null}
 </ConsolePage>;
}
