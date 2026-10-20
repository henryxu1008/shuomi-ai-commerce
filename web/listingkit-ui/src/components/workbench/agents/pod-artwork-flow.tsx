"use client";
import Link from "next/link";
import {useEffect,useRef,useState,useSyncExternalStore} from "react";
import {useQuery} from "@tanstack/react-query";
import {Card} from "@/components/ui/card";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {configurationRequest,type ConfigurationScope} from "@/lib/api/agent-configuration";
import {readArtworkRun,resolveArtworkPrepare,writeArtwork} from "@/lib/api/pod-artwork";
import {MarketAPIError} from "@/lib/api/supply-market";
import {knowledgeId} from "@/lib/api/knowledge";
import {artworkIntentSchema,artworkPrepareSchema,artworkRecentSchema,type ArtworkRun,type ArtworkIntent} from "@/lib/contracts/pod-artwork";
import {executionModeViewSchema} from "@/lib/contracts/execution-mode";
import type {CatalogEntry,AgentTemplate} from "@/lib/contracts/agent-configuration";
import {useResourcePending} from "../resources/resource-pending";
import {ResourceDialog} from "../resources/resource-dialog";
import {configurationError} from "./agent-configuration-hooks";
import {executionModeLabels} from "./execution-mode-settings";
import styles from "./pod-artwork-flow.module.css";
const statusNames:Record<ArtworkRun["status"],string>={planning:"准备中",awaiting_plan_approval:"待确认报价",executing:"正在生成",evaluating:"正在核对图案",repairing:"正在恢复原结果",awaiting_final_approval:"待确认保存作品",blocked:"原操作待处理",completed:"作品已保存",failed:"生成未完成",cancelled:"已取消"};
export function PODArtworkFlow({scope,entry,templates,nonce,podAvailable=false}:{scope:ConfigurationScope;entry:CatalogEntry;templates:AgentTemplate[];nonce:number;podAvailable?:boolean}){
 const context=useWorkbenchContext();
 const pending=useResourcePending({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId},["pod-artwork"],artworkIntentSchema,{storage:"local",maxLength:16384});
 const [theme,setTheme]=useState(""),[requirements,setRequirements]=useState(""),[template,setTemplate]=useState(""),[selectedRun,setRunId]=useState<string|null>(null),[busy,setBusy]=useState(false),[failure,setFailure]=useState<unknown>(),[preview,setPreview]=useState(false),[after,setAfter]=useState("");
 const flight=useRef(false),active=useRef(true),controller=useRef<AbortController|null>(null);
 const pointer=`pod.artwork.last:${JSON.stringify([scope.userId,scope.organizationId])}`;
 const restored=useSyncExternalStore(listener=>{window.addEventListener("storage",listener);window.addEventListener("resource-pending",listener);return()=>{window.removeEventListener("storage",listener);window.removeEventListener("resource-pending",listener)}},()=>{try{return localStorage.getItem(pointer)??new URLSearchParams(window.location.search).get("artwork_run")??""}catch{return "unreadable"}},()=>"");
 const runId=selectedRun??(knowledgeId.safeParse(restored).success?restored:"");
 useEffect(()=>{active.current=true;return()=>{active.current=false;controller.current?.abort()}},[]);
 const register=context.registerOrganizationSwitchGuard;
 useEffect(()=>register(()=>!flight.current&&!pending.command&&!pending.error),[register,pending.command,pending.error]);
 const key=["pod-artwork",scope.userId,scope.organizationId];
 const mode=useQuery({queryKey:[...key,"mode",nonce],queryFn:({signal})=>configurationRequest(scope,"execution-modes/registered_agent/pod.design.agent",executionModeViewSchema,{signal}),enabled:entry.agent.activation!=="NOT_ENABLED"});
 const run=useQuery({queryKey:[...key,"run",runId],queryFn:({signal})=>readArtworkRun(scope,runId,signal),enabled:!!runId,refetchInterval:q=>q.state.data&&(["executing","evaluating","repairing"].includes(q.state.data.status)||!!q.state.data.pendingCommand)?3000:false,refetchIntervalInBackground:false});
 const recent=useQuery({queryKey:[...key,"recent",after,nonce,run.data?.status],queryFn:({signal})=>configurationRequest(scope,`pod.design.agent/recent-runs?pageSize=20${after?"&cursor="+after:""}`,artworkRecentSchema,{signal}),enabled:entry.canReadRuns});
 const current=run.data,locked=busy||!!pending.command||!pending.ready||pending.error;
 function remember(r:ArtworkRun){localStorage.setItem(pointer,r.runId);if(localStorage.getItem(pointer)!==r.runId)throw new Error("无法保留原运行编号");setRunId(r.runId)}
 async function send(intent:ArtworkIntent,replay=false){
  if(flight.current||!active.current||!pending.ready)return;
  flight.current=true;setBusy(true);setFailure(undefined);const abort=new AbortController();controller.current=abort;
  try{pending.persist(intent);const result=await writeArtwork(scope,intent,abort.signal);if(!active.current)return;
   if(intent.kind==="prepare"&&(result.contextId!==intent.body.contextId||result.requestKey!==intent.key)||intent.kind!=="prepare"&&result.runId!==intent.runId)throw new MarketAPIError("OUTCOME_UNKNOWN",502);
   remember(result);pending.clear(intent);if(result.runId===runId)await run.refetch();
  }catch(error){if(active.current){setFailure(error);if(!replay&&error instanceof MarketAPIError&&error.code!=="OUTCOME_UNKNOWN"&&[400,403,404,409,412,422].includes(error.status))pending.clear(intent)}}finally{flight.current=false;if(active.current)setBusy(false)}
 }
 async function verify(){const i=pending.command;if(!i||flight.current)return;flight.current=true;setBusy(true);setFailure(undefined);try{
  const result=i.kind==="prepare"?await resolveArtworkPrepare(scope,i.body.contextId,i.key):await readArtworkRun(scope,i.runId);if(!active.current)return;remember(result);
  const accepted=i.kind==="prepare"||i.kind==="confirm"&&result.confirmationActionId===i.key&&!["planning","awaiting_plan_approval","executing"].includes(result.status)||(i.kind==="save"||i.kind==="resume")&&(result.saved||result.pendingCommand?.ActionID===i.key)||i.kind==="cancel"&&result.status==="cancelled";
  if(accepted)pending.clear(i);else setFailure(new Error("原请求尚未核实，可再次读取或明确重试同一请求。"));if(result.runId===runId)await run.refetch();
 }catch(error){if(active.current)setFailure(error)}finally{flight.current=false;if(active.current)setBusy(false)}}
 const options=templates.filter(t=>t.agentId==="pod.design.agent"&&t.lifecycle==="ACTIVE");
 const newAllowed=entry.agent.activation==="ENABLED"&&entry.canUse&&!locked&&!!mode.data;
 const candidate=current?.candidates[0];
 return <Card className={styles.panel}>
  <h2>AI 生成新图案</h2><p>输入主题，核对点数后生成一张全新图案。确认保存作品后，继续模板布局与效果图。</p>
  {failure?<p role="alert">{failure instanceof Error&&!(failure instanceof MarketAPIError)?failure.message:configurationError(failure)}</p>:null}
  {restored==="unreadable"?<p role="alert">原运行记录暂不可读，请恢复浏览器存储。</p>:null}
  {pending.error?<p role="alert">原请求记录不可用，请恢复浏览器存储后核实。</p>:null}
  {pending.command?<div className={styles.actions}><p role="status">原请求编号：{pending.command.key}。结果未核实前不会创建另一请求。</p><Button disabled={busy} onClick={()=>void verify()}>核实原操作</Button><Button variant="outline" disabled={busy} onClick={()=>void send(pending.command!,true)}>重试同一请求</Button></div>:null}
  {!current&&!runId?<form className={styles.form} onSubmit={e=>{e.preventDefault();if(!mode.data)return;const chosen=options.find(t=>t.templateId===template);const body=artworkPrepareSchema.safeParse({theme,requirements,contextId:crypto.randomUUID(),expectedModeRevision:mode.data.revision,...(chosen?{template:{templateId:chosen.templateId,revision:chosen.version}}:{})});if(!body.success){setFailure(new Error("请填写有效主题与创作要求，并选择原创图案模板。"));return}void send({kind:"prepare",key:crypto.randomUUID(),body:body.data})}}>
   <label>图案主题<Input required maxLength={512} value={theme} disabled={locked} onChange={e=>setTheme(e.target.value)} placeholder="例如：水彩风格的森林小狐狸"/></label>
   <label>设计要求<textarea value={requirements} maxLength={4096} disabled={locked} onChange={e=>setRequirements(e.target.value)} placeholder="构图、配色、需要避免的元素…"/></label>
   <label>原创图案模板<select value={template} disabled={locked} onChange={e=>setTemplate(e.target.value)}><option value="">使用企业默认精确版本</option>{options.map(t=><option key={t.templateId} value={t.templateId}>{t.name} · v{t.version}</option>)}</select></label>
   <p>当前输出为一张 1024 × 1024 图案。画布适配不代表达到所有款式的印刷清晰度。</p>
   <Button type="submit" disabled={!newAllowed}>准备图案并查看报价</Button>
   {!entry.canUse?<p>请先完成本企业启用、模板和图片模型配置。未配置时不会调用模型。</p>:null}
  </form>:null}
  {run.error?<p role="alert">{configurationError(run.error)} <Button variant="outline" onClick={()=>void run.refetch()}>读取原运行</Button></p>:runId&&!current?<p role="status">正在读取原图案运行…</p>:null}
  {current?<>
   <div className={styles.heading}><div><h3>{current.theme}</h3><p role="status">{statusNames[current.status]}</p><p>{executionModeLabels[current.mode]} · 点数上限 {current.points} · 已结算 {current.settledPoints}</p></div><Button variant="outline" disabled={busy} onClick={()=>{void run.refetch();void mode.refetch()}}>核实当前状态</Button></div>
   <p>{current.requirements}</p>
   {current.status==="awaiting_plan_approval"?<><p>确认后将调用图片模型生成一个原创图案，最多使用 {current.points} 点。请核对主题、要求和预算。</p><Button disabled={locked||!current.generationAllowed} onClick={()=>void send({kind:"confirm",key:crypto.randomUUID(),runId:current.runId,body:{planRevision:current.planRevision,planDigest:current.planDigest,quoteDigest:current.quoteDigest}})}>确认点数并生成新图案</Button></>:null}
   {candidate?<div className={styles.result}><div className={styles.preview}><button aria-label="放大查看生成图案" onClick={()=>setPreview(true)}><img src={candidate.url} alt={current.theme}/></button><p>{candidate.width} × {candidate.height} · 原生成图案</p></div><div className={styles.controls}><h3>确认作品</h3><p>检查实际图案与使用权。保存后仍需明确批准为印花图案。</p>{current.saveAvailable?<Button disabled={locked||!entry.canUse} onClick={()=>void send({kind:"save",key:crypto.randomUUID(),runId:current.runId,body:{planRevision:current.planRevision,resultDigest:current.resultDigest}})}>确认并保存作品</Button>:null}{current.saved?<><p role="status">作品已保存到我的数据。</p>{podAvailable?<Link href={`/workbench/supply/catalogs/sds/design?artwork=${current.saved.itemId}`} prefetch={false}>选择模板，进入布局与效果图 →</Link>:<p>SDS 模板与效果图暂不可用；已生成作品保留，配置就绪后可继续。</p>}<Link href="/workbench/data/mine" prefetch={false}>查看我的数据</Link></>:null}</div></div>:null}
   {!current.generationAllowed&&!current.saved?<p>只读模式或当前授权、配置已变化；新生成和保存操作不可执行，原结果保留。</p>:null}
   {current.recoverableEffects?.map(effect=>typeof effect.SlotID==="string"&&effect.SlotID==="artwork"&&typeof effect.Attempt==="number"?<Button key={effect.Attempt} variant="outline" disabled={locked} onClick={()=>void send({kind:"recover",key:crypto.randomUUID(),runId:current.runId,body:{planRevision:current.planRevision,slotId:"artwork",attempt:effect.Attempt as number}})}>恢复原图案结果（不重新生成）</Button>:null)}
   {current.pendingCommand?.Kind==="approve_results"&&current.pendingCommand.PlanRevision===current.planRevision?<Button disabled={locked} variant="outline" onClick={()=>void send({kind:"resume",key:current.pendingCommand!.ActionID,runId:current.runId,body:{planRevision:current.planRevision}})}>继续核实原保存操作</Button>:null}
   {!["completed","failed","cancelled"].includes(current.status)?<Button variant="outline" disabled={locked} onClick={()=>void send({kind:"cancel",key:crypto.randomUUID(),runId:current.runId,body:{planRevision:current.planRevision}})}>取消原运行</Button>:<Button variant="outline" disabled={locked} onClick={()=>{localStorage.removeItem(pointer);setRunId("");setFailure(undefined)}}>创作另一个图案</Button>}
   <p className={styles.reference}>原运行编号：{current.runId} · 请求编号：{current.requestKey}</p>
  </>:null}
  {entry.canReadRuns?<section><h3>图案运行历史</h3>{recent.error?<p role="alert">{configurationError(recent.error)}</p>:recent.data?.items.length?<ul>{recent.data.items.map(r=><li key={r.runId}><Button variant="ghost" disabled={locked} onClick={()=>{localStorage.setItem(pointer,r.runId);setRunId(r.runId)}}>{statusNames[r.status]} · {new Date(r.createdAt).toLocaleString("zh-CN")} · {r.runId}</Button></li>)}</ul>:<p>暂无当前企业及本人的图案运行。</p>}{recent.data?.nextCursor?<Button variant="outline" onClick={()=>setAfter(recent.data!.nextCursor)}>下一页历史</Button>:null}{after?<Button variant="outline" onClick={()=>setAfter("")}>返回第一页</Button>:null}</section>:null}
  {preview&&candidate?<ResourceDialog title="生成图案预览" locked={false} onClose={()=>setPreview(false)} className={styles.dialog}><img className={styles.large} src={candidate.url} alt={current?.theme??"生成图案"}/><Button variant="outline" onClick={()=>setPreview(false)}>关闭预览</Button></ResourceDialog>:null}
 </Card>;
}
