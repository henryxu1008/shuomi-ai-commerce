"use client";
import { useEffect,useLayoutEffect,useRef,useState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { amazonCaptureSchema,freezeAmazonCapture,amazonCaptureErrors,type AmazonCaptureResult } from "@/lib/contracts/amazon-browser-capture";
import { PriceText } from "@/components/workbench/collections/source-quotes";
import { amazonCaptureRequest,AmazonCaptureError } from "@/lib/api/amazon-browser-capture";
import { useCaptureDraft } from "../use-capture-draft";
import { captureHandoffExpiredMessage,isCaptureHandoffExpired } from "../capture-draft";
import { loadAmazonMarker,saveAmazonMarker,sameAmazonScope,type AmazonMarker } from "./capture-marker";

const unknown="保存结果待核实，本机资料仍保留。请检查原操作。";
type View={attempted:boolean;busy:boolean;result?:AmazonCaptureResult;scopeKey?:string;message:string;retryUntil?:number;handoffExpired?:boolean};
export function AmazonCaptureReceiver(){
 const capture=useCaptureDraft("amazon",amazonCaptureSchema);
 const {context,scope,ready,entry,draft,client}=capture;
 const active=useRef<AbortController|null>(null),busy=useRef(false),latest=useRef(capture);useLayoutEffect(()=>{latest.current=capture;});
 const [view,setView]=useState<View>({attempted:false,busy:false,message:"确认当前账号和企业后保存本次观察资料。"});
 const scopeKey=JSON.stringify(scope);
 const handoffExpired=view.scopeKey===scopeKey&&view.handoffExpired;
 const started=view.attempted||entry?.kind==="recovery"||!!draft&&!draft.canSubmit;
 const stillCurrent=()=>latest.current.stillCurrent()&&JSON.stringify(latest.current.scope)===scopeKey;
 useEffect(()=>{if(!ready)active.current?.abort();return()=>active.current?.abort();},[ready,scopeKey]);
 async function run(action:"submit"|"verify"|"retry"){
  if(!entry||!ready||view.scopeKey&&view.scopeKey!==scopeKey||busy.current||action!=="verify"&&(!client||handoffExpired)||action==="submit"&&started)return;
  busy.current=true;const controller=new AbortController();active.current=controller;
  setView(v=>({...v,busy:true,result:undefined,retryUntil:undefined,scopeKey}));
  let dispatched=false,sendIntent=false,body=draft?.intent?.body,marker:AmazonMarker|undefined;
  const valid=()=>stillCurrent()&&!controller.signal.aborted;
  try{
   const refreshed=await context.retry();
   if(!refreshed||refreshed.user.id!==scope.userId||refreshed.effectiveOrganizationId!==scope.organizationId||refreshed.effectiveMemberId!==scope.memberId||!valid())throw new AmazonCaptureError("IDENTITY_CONTEXT_CHANGED");
   if(draft){
    const frozen=await freezeAmazonCapture(draft.payload);
    if(draft.intent&&draft.intent.body!==frozen.body)throw new AmazonCaptureError("CAPTURE_CONFLICT");
    body=draft.intent?.body??frozen.body;
    marker={...scope,key:entry.key,source:"amazon",site:draft.payload.evidence.site,sourceID:draft.payload.evidence.sourceID,digest:draft.intent?.digest??frozen.digest};
   }else marker=loadAmazonMarker(sessionStorage,scope,entry.key)??undefined;
   if(!marker||!sameAmazonScope(marker,scope))throw new AmazonCaptureError("ORIGINAL_SCOPE_UNAVAILABLE");
   if(action!=="verify"){
    if(!client||!body)throw new AmazonCaptureError("INVALID_CAPTURE");
    if(action==="retry"){
     if(!view.retryUntil||view.retryUntil<=Date.now())throw new AmazonCaptureError("RECHECK_REQUIRED");
     await client.retry();
    }else await client.bind(body);
    if(!valid())throw new AmazonCaptureError("IDENTITY_CONTEXT_CHANGED");
    // The durable byte/scope binding, not sessionStorage, is the dispatch gate.
    history.replaceState(null,"",`/capture/amazon#operationKey=${entry.key}`);
    saveAmazonMarker(sessionStorage,marker);
    setView(v=>({...v,attempted:true,message:"资料已在本机保全，正在保存…"}));
    sendIntent=true;await client.send();
   }
   if(!valid())throw new AmazonCaptureError("IDENTITY_CONTEXT_CHANGED");
   dispatched=true;
   const result=await amazonCaptureRequest(scope,entry.key,action==="verify"?undefined:body,controller.signal);
   if(result.payloadSHA256!==marker.digest)throw new AmazonCaptureError("CAPTURE_CONFLICT");
   if(!valid())return;
   if(client)await client.report("saved_reported",result.captureID).catch(()=>{});
   setView(v=>({...v,result,scopeKey,message:result.archived?"原操作已保存；原引用当前已归档。":"已保存原操作，可在“我的数据”查看固定版本。"}));
  }catch(error){
   if(!stillCurrent())return;
   if(!dispatched&&isCaptureHandoffExpired(error)){
    setView(v=>({...v,handoffExpired:true,message:(action==="retry"?"原请求仍可能完成，请先检查原操作，不要重复保存。":"")+captureHandoffExpiredMessage}));return;
   }
   const code=error instanceof AmazonCaptureError?error.code:error instanceof Error&&error.message.startsWith("DRAFT_")?error.message:"OUTCOME_UNKNOWN";
   if(action==="verify"&&code==="CAPTURE_NOT_FOUND"){
    // Point-in-time absence is not proof the previous POST ended. The existing
    // SRC lock remains authoritative; this only permits one explicit same-body retry.
    let until:number|undefined,expired=false;
    if(client&&body)try{await client.armRetry();until=Date.now()+30000;}catch(error){expired=isCaptureHandoffExpired(error);}
    setView(v=>({...v,retryUntil:until,handoffExpired:expired||v.handoffExpired,message:until?"原键目前没有完整已保存记录；原请求仍可能完成。可明确重试同一份资料，不会换键。":expired?"原键目前没有完整已保存记录；原请求仍可能完成。"+captureHandoffExpiredMessage:"原键目前没有完整已保存记录；未自动重发。请从原插件恢复资料。"}));
   }else{
    const definite=dispatched&&Object.hasOwn(amazonCaptureErrors,code)&&!["OUTCOME_UNKNOWN","CAPTURE_UNAVAILABLE","CAPTURE_STATE_INVALID","DEPENDENCY_UNAVAILABLE"].includes(code);
    if(client&&sendIntent)await client.report(definite?"rejected":"unknown").catch(()=>{});
    const message=code==="ORIGINAL_SCOPE_UNAVAILABLE"?"当前身份、企业和成员下没有原恢复标记。请从原插件恢复本机资料。":code==="RECHECK_REQUIRED"?"请重新检查原操作，再决定是否重试。":definite?"保存被拒绝："+code+"。本机资料保留。":sendIntent?unknown:"保存未发送："+code+"。本机资料保留。";
    setView(v=>({...v,message}));
   }
  }finally{busy.current=false;if(active.current===controller)active.current=null;setView(v=>({...v,busy:false}));}
 }
 const result=ready&&view.scopeKey===scopeKey?view.result:undefined;
 const payload=ready?draft?.payload:undefined;
 return <section className="mx-auto max-w-3xl space-y-5 py-6">
  <header><p className="text-sm text-muted-foreground">Amazon / 插件商品采集</p><h1 className="text-2xl font-semibold">确认并保存来源资料</h1><p className="mt-2 text-sm text-muted-foreground">已保全资料保存在原插件的本机草稿中；页面刷新不会自动提交。</p></header>
  <div className="rounded-xl border p-4 text-sm"><p>账号：{ready?scope.userId:"请先登录"}</p><p>企业：{ready?context.effectiveOrganization?.name:"请选择企业"}</p><p>此插件保存免费。</p></div>
  {payload&&!started?<div className="space-y-2 rounded-xl border p-4"><h2 className="font-medium">{payload.evidence.title}</h2><p className="break-all text-sm">{payload.evidence.sourceURL}</p><ul className="text-sm">{payload.evidence.variants.map((v,i)=>v.price?<li key={i}>{v.sourceID??"规格 "+(i+1)}：<PriceText price={v.price} showUnknown/></li>:null)}</ul><ul className="text-sm text-muted-foreground">{payload.evidence.missingFacts.map((m,i)=><li key={i}>{m.field}：{m.reason}</li>)}</ul></div>:null}
  <p role="status">{!ready?"当前上下文不可用或已改变，请返回原账号、企业与成员。":capture.error??view.message}</p>
  <div className="flex flex-wrap gap-3">{payload&&!started&&!handoffExpired?<Button disabled={!ready||view.busy} onClick={()=>void run("submit")}>确认保存</Button>:null}{started?<Button variant="outline" disabled={!ready||view.busy} onClick={()=>void run("verify")}>检查原操作</Button>:null}{view.retryUntil&&view.scopeKey===scopeKey&&!handoffExpired?<Button disabled={!ready||view.busy} onClick={()=>void run("retry")}>用原键重试同一份资料</Button>:null}{view.busy?<Button variant="outline" onClick={()=>{active.current?.abort();setView(v=>({...v,message:unknown}));}}>停止等待</Button>:null}</div>
  {result?<div className="space-y-2 rounded-xl border p-4 text-sm"><p>已保存：{result.snapshot.title} · 版本 {result.catalogVersion}</p><p className="break-all">{result.captureID}</p>{result.currentBatchID!==result.batchID?<p>原引用已移动到其它批次。</p>:null}<ul>{result.snapshot.warnings?.map((w,i)=><li key={i}>{w.field}：{w.message}</li>)}</ul><Link href="/workbench/data/mine">打开我的数据</Link></div>:null}
  <p className="text-xs text-muted-foreground">交接过期时请使用原插件恢复资料。不要分享此页地址；清理插件存储或卸载插件会删除本机草稿。</p>
 </section>;
}
