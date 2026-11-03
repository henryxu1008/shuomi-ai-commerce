"use client";

import {useEffect,useMemo,useRef,useState} from "react";
import Image from "next/image";
import {z} from "zod";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Select} from "@/components/ui/select";
import {AnalysisError,analysisRequest,type AnalysisScope} from "@/lib/api/product-image-analysis";
import {analysisSourcesSchema,analysisResultSchema,analysisRecentSchema,type AnalysisSources,type AnalysisResult,type AnalysisRecent} from "@/lib/contracts/product-image-analysis";
import {formatMicropoints} from "@/lib/contracts/ai-points";
import {isAcquisitionUUID} from "@/lib/contracts/product-acquisition";
import {executionModeViewSchema} from "@/lib/contracts/execution-mode";
import {useConfigurationRead} from "../agents/agent-configuration-hooks";

type Props={kind:AnalysisScope["kind"];contextId:string};
const pendingSchema=z.strictObject({key:z.string().refine(isAcquisitionUUID),uncertain:z.boolean()});
type Pending=z.infer<typeof pendingSchema>;
const areas={subject:"主体",framing:"构图",background:"背景",clarity:"清晰度"};
const observationLabels={possible_issue:"可能有问题",no_obvious_issue:"未发现明显问题",unknown:"无法判断"};
const suggestionLabels={check:"人工核实",change_image:"考虑更换图片",open_image_workflow:"到原图片流程处理"};
const reasons:Record<string,string>={MODEL_NOT_READY:"企业多模态模型尚未准入，请由管理员在统一模型配置中处理。",AGENT_NOT_ENABLED:"商品图片智能体未启用。",RUN_PERMISSION_REQUIRED:"当前身份可查看，但没有运行权限。",CONFIGURATION_CHANGED:"图片、模型或配置已变化。原请求保留，请读取当前来源后重新准备。",REVISION_MISMATCH:"执行模式已变化，请重新读取并准备。",FORBIDDEN:"当前身份无权执行此操作。",NOT_FOUND:"暂未找到原请求，请保留编号继续核实。",OUTCOME_UNKNOWN:"结果尚未确认，请核实原请求，不要重新发起检查。"};
function definitePreparationRejection(action:string,error:unknown){
 if(action!=="prepare"||!(error instanceof AnalysisError))return false;
 const statuses:Record<string,number>={INVALID_REQUEST:400,INVALID_AGENT_REQUEST:400,FORBIDDEN:403,CONFIGURATION_CHANGED:409,AGENT_NOT_ENABLED:409,AGENT_DEFINITION_UNAVAILABLE:409,REVISION_MISMATCH:412,PRECONDITION_REQUIRED:428};
 return statuses[error.code]===error.status;
}

export function ProductImageAnalysisPanel(props:Props){
 const c=useWorkbenchContext(),userId=c.user?.id,organizationId=c.effectiveOrganization?.id;
 const permissions=c.permissions??[];
 if(!userId||!organizationId||c.isLoading||c.isSwitching||c.error||c.blockingError||c.selectionRequired||!permissions.includes("workbench.agent.use"))return null;
 return <ScopedAnalysis key={JSON.stringify([userId,organizationId,props,permissions])} scope={{userId,organizationId,...props}} canRun={permissions.includes("local_agent.write")}/>;
}

function ScopedAnalysis({scope,canRun}:{scope:AnalysisScope;canRun:boolean}){
 const mode=useConfigurationRead(scope,"execution-modes/registered_agent/product.image.agent",executionModeViewSchema);
 const storageKey=`product-image-analysis:${scope.userId}:${scope.organizationId}:${scope.kind}:${scope.contextId}`;
 const [sources,setSources]=useState<AnalysisSources>(),[recent,setRecent]=useState<AnalysisRecent>(),[result,setResult]=useState<AnalysisResult>();
 const [selected,setSelected]=useState(""),[consent,setConsent]=useState(false),[pending,setPending]=useState<Pending|undefined>(()=>{if(typeof window==="undefined")return;try{const raw=localStorage.getItem(storageKey),v=raw?pendingSchema.safeParse(JSON.parse(raw)):null;return v?.success?v.data:undefined}catch{return}}),[busy,setBusy]=useState(false),[error,setError]=useState(""),[nonce,setNonce]=useState(0);
 const active=useRef(true),flight=useRef(false),controller=useRef<AbortController|null>(null);
 useEffect(()=>{active.current=true;return()=>{active.current=false;controller.current?.abort()}},[]);
 const {userId,organizationId,kind,contextId}=scope;
 const stableScope=useMemo(()=>({userId,organizationId,kind,contextId}),[userId,organizationId,kind,contextId]);
 useEffect(()=>{const abort=new AbortController();void Promise.allSettled([
  analysisRequest(stableScope,"sources",analysisSourcesSchema,{signal:abort.signal}).then(v=>{if(!abort.signal.aborted)setSources(v)}),
  analysisRequest(stableScope,"recent",analysisRecentSchema,{signal:abort.signal}).then(v=>{if(!abort.signal.aborted)setRecent(v)})
 ]).then(results=>{if(!abort.signal.aborted&&results.some(v=>v.status==="rejected"))setError("图片来源或历史暂不可读；不会自动执行检查。")});return()=>abort.abort()},[stableScope,nonce]);
 function remember(p:Pending){localStorage.setItem(storageKey,JSON.stringify(p));setPending(p)}
 async function run(action:"prepare"|"confirm"|"read",key?:string){
  if(flight.current)return;flight.current=true;setBusy(true);setError("");setConsent(false);const abort=new AbortController();controller.current=abort;
  try{
   let body:unknown;
   if(action==="prepare"){
    if(!sources?.ready||!canRun||!mode.data||pending?.uncertain)throw new AnalysisError("CONFIGURATION_CHANGED",409);
    const image=sources.choices.find(v=>`${v.kind}:${v.id}`===selected);if(!image)throw new AnalysisError("INVALID_REQUEST",400);
    key=crypto.randomUUID();remember({key,uncertain:true});
    body={kind:image.kind,imageId:image.id,catalogVersion:sources.catalogVersion,publicationId:sources.publicationId,expectedModeRevision:mode.data.revision,...image.kind==="approved"?{inventoryDigest:sources.inventoryDigest}:{}};
   }else if(action==="confirm"){
    if(!canRun||!result||result.status!=="prepared"||!consent||!selectionMatches)throw new AnalysisError("CONFIGURATION_CHANGED",409);
    key=result.requestKey;remember({key,uncertain:true});
    const {preparationId,configurationDigest,quoteDigest,selectionDigest,contentSha256,modelId,maximumMicropoints}=result;
    body={confirmed:true,preparationId,configurationDigest,quoteDigest,selectionDigest,contentSha256,modelId,maximumMicropoints};
   }
   const value=await analysisRequest(scope,action,analysisResultSchema,{requestKey:key,body,signal:abort.signal});
   if(!active.current||abort.signal.aborted)return;
   setResult(value);
   if(action==="read"&&value.status==="prepared"&&sources?.catalogVersion===value.catalogVersion&&sources.publicationId===value.publicationId&&sources.choices.some(v=>v.kind===value.kind&&v.id===value.imageId))setSelected(`${value.kind}:${value.imageId}`);
   if(action!=="read"||pending?.key===key){
    const unresolved=value.status==="unknown";
    remember({key:value.requestKey,uncertain:unresolved});
   }
   if(value.status==="unknown")setError(reasons.OUTCOME_UNKNOWN);
   if(action==="confirm"||action==="read")setNonce(v=>v+1);
  }catch(e){if(active.current&&!abort.signal.aborted){const definite=definitePreparationRejection(action,e),code=e instanceof AnalysisError?action!=="read"&&!definite?"OUTCOME_UNKNOWN":e.code:"LOCAL_RECEIPT_UNAVAILABLE";setError(reasons[code]??(code==="LOCAL_RECEIPT_UNAVAILABLE"?"无法保留原请求编号，请检查浏览器存储后再操作。":"当前操作未完成，请保留原请求并核实。"));if(definite&&key)remember({key,uncertain:false})}}
  finally{flight.current=false;if(active.current)setBusy(false)}
 }
 const selectionMatches=!!sources&&!!result&&sources.catalogVersion===result.catalogVersion&&sources.publicationId===result.publicationId&&selected===`${result.kind}:${result.imageId}`;
 const selectedImage=sources?.choices.find(v=>`${v.kind}:${v.id}`===selected);
 const ready=canRun&&sources?.ready&&!!mode.data;
 return <Card className="space-y-4 p-4" aria-label="单图只读检查">
  <div><h3 className="font-semibold">单图只读检查</h3><p className="text-sm text-muted-foreground">检查一张已有图片的主体、构图、背景与清晰度疑点。使用企业统一多模态模型；建议不会改资料、生成或批准图片。</p></div>
  {!sources?<p role="status">正在读取图片来源</p>:<>
   {!sources.ready&&<p role="status">{reasons[sources.reason]??"当前模型或图片能力尚未准入。"}</p>}
   {sources.ready&&!mode.data&&<p role="status">正在核实原智能体执行模式；核实前不会发送图片。</p>}
   {sources.inventoryStatus!=="READY"&&<p className="text-sm text-muted-foreground">批准素材{sources.inventoryStatus==="NOT_READY"?"尚未准备好":"状态未知"}；已有来源图仍按自身准入检查。</p>}
   <label className="block space-y-2"><span>待检查图片</span><Select value={selected} disabled={busy||pending?.uncertain} onChange={e=>{setSelected(e.target.value);setConsent(false);setResult(undefined)}}><option value="">选择一张已有图片</option>{sources.choices.map(v=><option key={`${v.kind}:${v.id}`} value={`${v.kind}:${v.id}`}>{v.kind==="source"?"来源图":"批准素材"} · {v.id}</option>)}</Select></label>
   {selectedImage&&<Image src={selectedImage.url} alt="待检查的已有图片" width={160} height={160} unoptimized className="max-h-40 object-contain"/>}
   <Button disabled={!ready||!selected||busy||pending?.uncertain} onClick={()=>void run("prepare")}>准备单图检查</Button>
  </>}
  {error&&<p role="alert">{error}</p>}
  {pending&&<div className="space-y-2 text-sm"><p className="break-all">原请求编号：{pending.key}</p><Button variant="outline" disabled={busy} onClick={()=>void run("read",pending.key)}>核实原请求</Button></div>}
  {result&&<section className="space-y-3" aria-label="原检查结果">
   <p className="break-all">{result.kind==="source"?"来源图":"批准素材"} {result.imageId} · 商品版本 {result.catalogVersion} · {result.width}×{result.height}</p>
   <p>模型 {result.modelId} · 费用上界 {formatMicropoints(result.maximumMicropoints)} 点{result.actualMicropoints!==""?` · 实际 ${formatMicropoints(result.actualMicropoints)} 点`:""}</p>
   {result.status==="prepared"?<><p>准备完成，尚未向模型发送图片。</p><label className="flex items-start gap-2"><input type="checkbox" checked={consent} disabled={!ready||!selectionMatches||busy||pending?.uncertain} onChange={e=>setConsent(e.target.checked)}/><span>我同意将上述准确图片发送给已配置模型，费用不超过上述上界。</span></label><Button disabled={!ready||!selectionMatches||!consent||busy||pending?.uncertain} onClick={()=>void run("confirm")}>确认发送图片并检查</Button></>:result.status==="unknown"?<p role="status">结果尚未确认；保留原请求和原占用，仅查询原结果。</p>:result.status==="not_sent"?<p>已确认未发送，实际用量为零；原请求不会自动重试。</p>:result.status==="failed"?<p>检查未完成可展示结果；已确认用量仍按原账本结算，不自动重试。</p>:<>
    <h4 className="font-medium">只读建议</h4>
    {result.opinion?.observations.map((v,i)=><p key={`o${i}`}>{areas[v.area]} · {observationLabels[v.status]}：{v.reason}</p>)}
    {result.opinion?.suggestions.map((v,i)=><p key={`s${i}`}>{areas[v.area]} · {suggestionLabels[v.action]}：{v.reason}</p>)}
    {result.opinion?.unknowns.map((v,i)=><p key={`u${i}`}>{v}</p>)}
   </>}
  </section>}
  <section className="space-y-2" aria-label="图片检查历史"><h4 className="font-medium">图片检查历史</h4>{recent?.items.map(v=><Button key={v.requestKey} variant="outline" disabled={busy} onClick={()=>void run("read",v.requestKey)}>查看原检查 · {v.requestKey}</Button>)}{recent&&!recent.items.length&&<p className="text-sm text-muted-foreground">当前商品暂无已开始的检查。</p>}{recent?.nextCursor&&<Button variant="outline" disabled={busy} onClick={()=>{const abort=new AbortController();controller.current=abort;void analysisRequest(scope,"recent",analysisRecentSchema,{cursor:recent.nextCursor,signal:abort.signal}).then(v=>{if(active.current&&!abort.signal.aborted)setRecent({...v,items:[...recent.items,...v.items]})}).catch(()=>{if(active.current)setError("历史暂不可读，请稍后核实原请求。")})}}>更早记录</Button>}</section>
 </Card>;
}
