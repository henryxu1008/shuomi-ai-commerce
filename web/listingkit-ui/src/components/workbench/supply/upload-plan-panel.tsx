"use client";
import {useEffect,useRef,useState} from "react";
import Link from "next/link";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {CollectionDialog} from "../collections/collection-page";
import {listUploadPlans,readUploadPlan,previewUploadPlan,type SupplyScope} from "@/lib/api/supply-chain";
import type {UploadPlan,UploadPlanPreview} from "@/lib/contracts/upload-plan";
import {supplyStageHref} from "@/lib/workbench/supply-navigation";
import type {SupplyCommandState} from "./use-supply-command";

const labels:Record<string,string>={adapt:"生成平台资料",optimize:"智能体优化",upload:"提交到平台",missing:"待补全",review:"待审核并应用",pending:"等待执行",running:"处理中",succeeded:"已上传",unknown:"原结果待核实",failed:"原操作失败",denied:"权限不足",cancelled:"已取消",rejected:"审核已拒绝",unavailable:"原结果暂不可读",channel_exists:"该渠道已有其他版本回执"};
type Props={scope:SupplyScope;planId:string;autoPreview?:string;reload:number;command:SupplyCommandState;blocked:boolean;canSubmit:boolean;onOpen:(id:string)=>void;onLoaded:(plan:UploadPlan)=>void;onItem:(source:string)=>void;onOperation:(id:string)=>void;onReview:(source:string,operation:string)=>void};
export function UploadPlanPanel({scope,planId,autoPreview,reload,command,blocked,canSubmit,onOpen,onLoaded,onItem,onOperation,onReview}:Props){
 const [storedPlan,setPlan]=useState<UploadPlan>(),[loadedKey,setLoadedKey]=useState(""),[plans,setPlans]=useState<UploadPlan[]>([]),[after,setAfter]=useState<string>(),[next,setNext]=useState<string>(),[approved,setApproved]=useState<{key:string;value:UploadPlanPreview}>(),[confirm,setConfirm]=useState(false),[busy,setBusy]=useState(false),[error,setError]=useState("");
 const loaded=useRef(onLoaded);useEffect(()=>{loaded.current=onLoaded},[onLoaded]);
 const autoDone=useRef("");
 const active=useRef(false),request=useRef<AbortController|null>(null);
 useEffect(()=>()=>request.current?.abort(),[]);
 const loadKey=JSON.stringify([scope,planId,reload,after]);
 const plan=loadedKey===loadKey?storedPlan:undefined;
 const preview=approved?.key===loadKey?approved.value:undefined;
 useEffect(()=>{const c=new AbortController();let live=true;request.current=c;
  void (planId?readUploadPlan(scope,planId,c.signal):listUploadPlans(scope,{limit:20,after},c.signal)).then(async v=>{
   if(!live)return;setError("");setLoadedKey(loadKey);setApproved(undefined);setConfirm(false);
   if("items" in v){setPlans(v.items);setNext(v.nextCursor)}else{
    setPlan(v);loaded.current(v);
    if(autoPreview===v.id&&autoDone.current!==v.id&&!blocked){
     autoDone.current=v.id;setBusy(true);
     const checked=await previewUploadPlan(scope,v.id,{expectedRevision:v.revision},c.signal);
     if(live){setApproved({key:loadKey,value:checked});setPlan(checked.plan);setConfirm(true);setBusy(false)}
    }
   }
  }).catch(()=>{if(live){setBusy(false);setError("无法读取本人当前企业的上传计划或当前进度，请刷新重试。")}});
  return()=>{live=false;c.abort();request.current?.abort()};
 },[scope,planId,reload,after,autoPreview,blocked,loadKey]);
 async function inspect(){if(!plan||blocked||active.current)return;active.current=true;setBusy(true);setError("");setApproved(undefined);const c=new AbortController();request.current=c;
  try{const v=await previewUploadPlan(scope,plan.id,{expectedRevision:plan.revision},c.signal);if(!c.signal.aborted){setApproved({key:loadKey,value:v});setPlan(v.plan);setConfirm(true)}}catch{if(!c.signal.aborted)setError("资料、店铺或原操作已变化，无法继续。请刷新后重新核对。")}finally{active.current=false;if(!c.signal.aborted)setBusy(false)}
 }
 function proceed(){if(!preview||blocked||preview.actionable===0||!!preview.counts.upload&&!canSubmit)return;setConfirm(false);void command.execute("continue-upload-plan",{planId:preview.plan.id,expectedRevision:preview.plan.revision,previewHash:preview.hash});setApproved(undefined);}
 const context=plan?{preparation:plan.input.preparationId,store:plan.input.storeId,plan:plan.id,mode:"platform" as const}:{};
 return <Card className="space-y-4 p-5" aria-label="原上传计划">
  <div className="flex flex-wrap items-center justify-between gap-3"><div><h2 className="font-semibold">{planId?"继续原上传计划":"已保存的上传计划"}</h2><p className="mt-1 text-sm text-muted-foreground">补全保存、审核应用后，回到原计划核对并继续。刷新不会重新提交。</p></div>{planId?<Button variant="outline" disabled={blocked||busy||!plan||plan.cancelled} onClick={()=>void inspect()}>{busy?"正在核对完整计划…":"检查进度并继续"}</Button>:null}</div>
  {error?<p role="alert" className="text-sm text-destructive">{error}</p>:null}
  {!planId?<><ul className="space-y-2">{(loadedKey===loadKey?plans:[]).map(p=><li key={p.id} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border p-3"><div><p className="text-sm">{p.count} 件 · {new Date(p.createdAt).toLocaleString("zh-CN")}</p><p className="break-all text-xs text-muted-foreground">计划 {p.id} · 店铺 {p.input.storeId}</p></div><Button variant="outline" disabled={blocked} onClick={()=>onOpen(p.id)}>打开原计划</Button></li>)}</ul>{after?<Button size="sm" variant="ghost" onClick={()=>setAfter(undefined)}>返回最近计划</Button>:null}{next?<Button size="sm" variant="outline" onClick={()=>setAfter(next)}>下一页计划</Button>:null}</>:plan?<><p className="break-all text-xs text-muted-foreground">计划 {plan.id} · 修订 {plan.revision} · {plan.cancelled?"后续步骤已停止":"进行中"} · 固定 {plan.count} 件 · {plan.input.publicationMode==="platform_draft"?"TikTok 平台草稿":"按原生平台方式提交"}</p><p className="break-all text-xs text-muted-foreground">规则：标准适配 · 智能体：{plan.input.titleTemplateId?`${plan.input.titleTemplateId} v${plan.input.titleTemplateRevision}`:"未启用"}</p><div className="flex flex-wrap gap-3">{!plan.cancelled?<Button variant="outline" disabled={blocked||busy} onClick={()=>void command.execute("cancel-upload-plan",{planId:plan.id,expectedRevision:plan.revision})}>停止后续步骤</Button>:null}<Button asChild variant="outline"><Link href={supplyStageHref("missing",context)}>补全原计划资料</Link></Button><Button asChild variant="outline"><Link href={supplyStageHref("review",context)}>审核原计划建议</Link></Button></div></>:<p role="status" className="text-sm">正在读取原计划…</p>}
  {confirm&&preview?<CollectionDialog title="确认继续上传" onClose={()=>setConfirm(false)}><p className="text-sm">原计划 {preview.plan.id} · 固定 {preview.plan.count} 件 · 本次核对 {preview.count} 件</p><p className="break-all text-sm">目标店铺：{preview.plan.input.storeId} · {preview.plan.input.publicationMode==="platform_draft"?"保存 TikTok 平台草稿":"按已保存的原生方式提交"}</p><dl className="grid gap-2 text-sm sm:grid-cols-2">{Object.entries(preview.counts).map(([step,count])=><div key={step}><dt className="text-muted-foreground">{labels[step]}</dt><dd>{count} 件</dd></div>)}</dl><p className="text-sm text-muted-foreground">只继续当前可执行的 {preview.actionable} 件。已成功、拒绝、失败、取消及结果待核实的原操作保持原结果。平台时间与提交方式沿所列准确资料版本。</p>
   <div className="max-h-72 space-y-2 overflow-y-auto">{preview.items.map(v=><div key={v.item.sourceId} className="space-y-2 rounded-lg border border-border p-3 text-sm"><p className="break-all">{v.item.sourceId} · {labels[v.step]}{v.facts.hasTarget?` · 目标版本 ${v.facts.target.revision}`:""}</p>{v.facts.hasTarget?<p className="break-all text-xs text-muted-foreground">资料 {v.facts.target.recordId}</p>:null}{v.facts.publicationSummary?.map((text,i)=><p key={i} className="break-all text-xs">{text}</p>)}<div className="flex flex-wrap gap-2">{v.step==="missing"?<Button size="sm" variant="outline" onClick={()=>{setConfirm(false);onItem(v.item.sourceId)}}>补全此商品</Button>:null}{v.step==="review"&&v.item.optimizeOperationId?<Button size="sm" variant="outline" onClick={()=>{setConfirm(false);onReview(v.item.sourceId,v.item.optimizeOperationId!)}}>审核并应用</Button>:null}{(v.item.uploadOperationId||v.item.optimizeOperationId||v.item.adaptOperationId)?<Button size="sm" variant="ghost" onClick={()=>{setConfirm(false);onOperation((v.item.uploadOperationId||v.item.optimizeOperationId||v.item.adaptOperationId)!)}}>查看原操作</Button>:null}</div></div>)}</div>{preview.count>preview.items.length?<p className="text-xs text-muted-foreground">明细展示前 {preview.items.length} 件，以上计数与本次确认覆盖完整 {preview.count} 件；其余商品可在原批次列表查看。</p>:null}
   {!!preview.counts.upload&&!canSubmit?<p role="alert" className="text-sm text-destructive">当前身份没有平台提交权限。</p>:null}<Button disabled={blocked||preview.actionable===0||!!preview.counts.upload&&!canSubmit} onClick={proceed}>确认继续 {preview.actionable} 件</Button>
  </CollectionDialog>:null}
 </Card>;
}
