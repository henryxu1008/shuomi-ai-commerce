"use client";
import { useEffect,useRef,useState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { amazonCaptureSchema,freezeAmazonCapture,type AmazonCapturePayload,type AmazonCaptureResult } from "@/lib/contracts/amazon-browser-capture";
import { PriceText } from "@/components/workbench/collections/source-quotes";
import { amazonCaptureRequest,AmazonCaptureError } from "@/lib/api/amazon-browser-capture";
import { parseCaptureEntry,readCaptureHandoff,notifyCaptureStatus,type CaptureEntry } from "../1688/capture-handoff";
import { loadAmazonMarker,saveAmazonMarker,sameAmazonScope,type AmazonScope,type AmazonMarker } from "./capture-marker";

const unknown="保存结果待核实。请检查原操作，保留此页面与原键。";
type View={entry:CaptureEntry|null;payload?:AmazonCapturePayload;body?:string;marker?:AmazonMarker;started:boolean;busy:boolean;result?:AmazonCaptureResult;message:string};
export function AmazonCaptureReceiver(){
 const context=useWorkbenchContext();const current=useRef(context);const active=useRef<AbortController|null>(null);const busy=useRef(false);
 const [view,setView]=useState<View>({entry:null,started:false,busy:false,message:"正在读取插件交接…"});
 const scope:AmazonScope={userId:context.user?.id??"",organizationId:context.effectiveOrganization?.id??"",memberId:context.effectiveMemberId??""};
 const ready=!context.isLoading&&!context.isSwitching&&!context.error&&!context.blockingError&&!context.selectionRequired&&!!scope.userId&&!!scope.organizationId&&!!scope.memberId;
 const scoped=ready&&(!view.marker||sameAmazonScope(scope,view.marker));
 useEffect(()=>{current.current=context;if(!scoped)active.current?.abort();},[context,scoped]);
 useEffect(()=>{const controller=new AbortController();void(async()=>{
  await Promise.resolve();const entry=parseCaptureEntry(window.location.href,"amazon");if(controller.signal.aborted)return;
  if(!entry){setView(v=>({...v,message:"交接地址无效。请从插件打开此页。"}));return;}
  if(entry.kind==="recovery"){setView(v=>({...v,entry,started:true,message:unknown}));return;}
  try{const payload=await readCaptureHandoff(entry,controller.signal,amazonCaptureSchema);if(!controller.signal.aborted)setView(v=>({...v,entry,payload,message:"确认当前账号和企业后保存本次观察资料。"}));}catch{if(!controller.signal.aborted)setView(v=>({...v,entry,message:"插件交接不可读，本页面未发送保存请求。"}));}
 })();return()=>{controller.abort();active.current?.abort();};},[]);
 async function run(submit:boolean){
  if(!view.entry||!scoped||busy.current||submit&&view.started)return;
  busy.current=true;const controller=new AbortController();active.current=controller;setView(v=>({...v,busy:true,result:undefined}));
  let dispatched=false;let marker=view.marker;
  const stillCurrent=()=>{const c=current.current;return !controller.signal.aborted&&!c.isLoading&&!c.isSwitching&&!c.error&&!c.blockingError&&c.user?.id===scope.userId&&c.effectiveOrganization?.id===scope.organizationId&&c.effectiveMemberId===scope.memberId;};
  try{
   const refreshed=await context.retry();if(!refreshed||refreshed.user.id!==scope.userId||refreshed.effectiveOrganizationId!==scope.organizationId||refreshed.effectiveMemberId!==scope.memberId||!stillCurrent())throw new AmazonCaptureError("IDENTITY_CONTEXT_CHANGED");
   let body:string|undefined;
   if(submit){
    if(!view.payload)throw new AmazonCaptureError("INVALID_CAPTURE");const frozen=await freezeAmazonCapture(view.payload);body=frozen.body;
    marker={...scope,key:view.entry.key,source:"amazon",site:view.payload.evidence.site,sourceID:view.payload.evidence.sourceID,digest:frozen.digest};
    if(!stillCurrent()||!saveAmazonMarker(sessionStorage,marker))throw new AmazonCaptureError("MARKER_UNRETAINED");
    window.history.replaceState(null,"",`/capture/amazon#operationKey=${view.entry.key}`);
    setView(v=>({...v,marker,body,started:true,message:"正在保存冻结的来源资料…"}));
   }else{
    marker=marker??loadAmazonMarker(sessionStorage,scope,view.entry.key)??undefined;
    if(!marker)throw new AmazonCaptureError("ORIGINAL_SCOPE_UNAVAILABLE");setView(v=>({...v,marker}));
   }
   if(!stillCurrent())throw new AmazonCaptureError("IDENTITY_CONTEXT_CHANGED");
   dispatched=true;const result=await amazonCaptureRequest(scope,view.entry.key,body,controller.signal);
   if(result.payloadSHA256!==marker.digest)throw new AmazonCaptureError("CAPTURE_CONFLICT");
   if(!stillCurrent())return;
   setView(v=>({...v,result,message:result.archived?"原操作已保存；原引用当前已归档。":"已保存原操作，可在“我的数据”查看固定版本。"}));
   if(view.entry.kind==="handoff")void notifyCaptureStatus(view.entry,"published",result.captureID);
  }catch(error){
   if(!stillCurrent())return;const code=error instanceof AmazonCaptureError?error.code:"OUTCOME_UNKNOWN";
   const message=code==="CAPTURE_NOT_FOUND"?"原键目前无完整已保存记录。未自动重发；原 payload 丢失时请暂停该保存。":code==="CAPTURE_CONFLICT"?"原键绑定的内容不一致，请保留原操作。":code==="ORIGINAL_SCOPE_UNAVAILABLE"?"当前身份、企业和成员下没有原恢复标记。请返回原上下文。":!dispatched?"保存未发送：上下文或恢复标记不可用。":unknown;
   setView(v=>({...v,message}));if(dispatched&&view.entry.kind==="handoff")void notifyCaptureStatus(view.entry,"outcome_unknown");
  }finally{busy.current=false;if(active.current===controller)active.current=null;setView(v=>({...v,busy:false}));}
 }
 const result=scoped?view.result:undefined;
 return <section className="mx-auto max-w-3xl space-y-5 py-6">
  <header><p className="text-sm text-muted-foreground">Amazon / 插件单商品采集</p><h1 className="text-2xl font-semibold">确认并保存来源资料</h1><p className="mt-2 text-sm text-muted-foreground">仅保存已观察的商品资料。未观察的规格、库存和图片仍保留缺失提示。</p></header>
  <div className="rounded-xl border p-4 text-sm"><p>账号：{ready?scope.userId:"请先登录"}</p><p>企业：{ready?context.effectiveOrganization?.name:"请选择企业"}</p><p>此插件保存免费。</p></div>
  {view.payload&&!view.started?<div className="space-y-2 rounded-xl border p-4"><h2 className="font-medium">{view.payload.evidence.title}</h2><p className="break-all text-sm">{view.payload.evidence.sourceURL}</p><ul className="text-sm">{view.payload.evidence.variants.map((v,i)=>v.price?<li key={i}>{v.sourceID??`规格 ${i+1}`}：<PriceText price={v.price} showUnknown/></li>:null)}</ul><ul className="text-sm text-muted-foreground">{view.payload.evidence.missingFacts.map((m,i)=><li key={i}>{m.field}：{m.reason}</li>)}</ul></div>:null}
  <p role="status">{scoped?view.message:"当前上下文不可用或已改变，请返回原账号、企业与成员。"}</p>
  <div className="flex gap-3">{view.payload&&!view.started?<Button disabled={!scoped||view.busy} onClick={()=>void run(true)}>确认保存</Button>:null}{view.started?<Button variant="outline" disabled={!scoped||view.busy} onClick={()=>void run(false)}>检查原操作</Button>:null}{view.busy?<Button variant="outline" onClick={()=>{active.current?.abort();setView(v=>({...v,message:unknown}));}}>停止等待</Button>:null}</div>
  {result?<div className="space-y-2 rounded-xl border p-4 text-sm"><p>已保存：{result.snapshot.title} · 版本 {result.catalogVersion}</p><p className="break-all">{result.captureID}</p>{result.currentBatchID!==result.batchID?<p>原引用已移动到其它批次。</p>:null}<ul>{result.snapshot.warnings?.map((w,i)=><li key={i}>{w.field}：{w.message}</li>)}</ul><Link href="/workbench/data/mine">打开我的数据</Link></div>:null}
  {view.started?<p className="text-xs text-muted-foreground">刷新不会提交。完整资料仅保留在本页内存；恢复标记保留在当前标签会话中。请勿分享恢复地址。</p>:null}
 </section>;
}
