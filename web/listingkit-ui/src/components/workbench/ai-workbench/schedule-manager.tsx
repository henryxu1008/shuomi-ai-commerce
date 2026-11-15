"use client";
import Link from "next/link";
import {useSearchParams} from "next/navigation";
import {useEffect,useRef,useState} from "react";
import {useInfiniteQuery,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button} from "@/components/ui/button";
import {Select} from "@/components/ui/select";
import {ResourceDialog} from "../resources/resource-dialog";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {requestAIWorkbench,AIWorkbenchError,type AIScope} from "@/lib/api/ai-workbench";
import {aiScheduleInput,aiScheduleControl,aiScheduleEdit,aiWorkbenchPath,type AIScheduleInput,type AIScheduledPlan} from "@/lib/contracts/ai-workbench";
import {ScheduleForm,schedulePoints,scheduleTime} from "./schedule-form";
import {errorText} from "./task-presentation";
import styles from "./schedules.module.css";

const states:Record<string,string>={ACTIVE:"已启用",PAUSED:"已暂停",CLOSED:"已结束",PENDING_REGISTRATION:"正在准备",REGISTRATION_UNKNOWN:"启用状态待确认"};
const occurrenceStates:Record<string,string>={CLAIMED:"已开始处理",DISPATCHED:"已交接执行",SKIPPED:"已跳过",BLOCKED:"执行前受阻",UNKNOWN:"执行结果待核实",RECONCILED:"已完成"};
const reasons:Record<string,string>={REPORT_SAVED:"经营日报已保存，可在我的报告中查看。",REPORT_CAPTURE_UNRESOLVED:"正在核对报告保存结果，请勿重复创建。",RUN_QUIESCENT:"本次执行已结束，生成的建议仍需审核",PLAN_NOT_ACTIVE:"计划当时未启用",PREVIOUS_RUN_ACTIVE:"上一次任务尚未处理完成，本次未启动",MAX_OCCURRENCES:"已达到次数上限",CATCHUP_EXPIRED:"已错过执行时间，本次未启动",MISSED_TRIGGER:"该时刻未能启动，不再补执行",CURRENT_AUTHORIZATION_UNAVAILABLE:"当前账号暂时无权执行这项计划",FROZEN_EXECUTION_UNAVAILABLE:"计划所需的商品、设置或额度暂不可用",EXECUTION_HANDOFF_UNRESOLVED:"任务已提交，执行结果仍在确认",ORIGINAL_RUN_OUTCOME_UNRESOLVED:"上一次执行的结果仍在确认，暂不启动下一次"};
type Command={action:"create"|"edit"|"pause"|"resume"|"close"|"delete";path:string;key:string;body:object};
function validCommand(value:unknown):value is Command{
 if(!value||typeof value!=="object")return false;
 const v=value as Command;if(!["create","edit","pause","resume","close","delete"].includes(v.action)||typeof v.path!=="string"||!/^[-0-9a-f]{36}$/.test(v.key))return false;
 const method=v.action==="edit"?"PATCH":"POST";
 return aiWorkbenchPath(method,v.path.split("/"))===`schedule-${v.action}`&&(v.action==="create"?aiScheduleInput:v.action==="edit"?aiScheduleEdit:aiScheduleControl).safeParse(v.body).success;
}

export function ScheduleManager({scope,authorizationKey,canUse,onBack}:{scope:AIScope;authorizationKey:string;canUse:boolean;onBack:()=>void}){
 const [kindFilter,setKindFilter]=useState("");
 const client=useQueryClient(),[selected,setSelected]=useState<string>(),[form,setForm]=useState<"create"|"edit"|"copy">(),[proposalId,setProposalId]=useState<string>();
 const [pending,setPending]=useState<Command>(),[busy,setBusy]=useState(false),[restored,setRestored]=useState(false),[error,setError]=useState(""),[notice,setNotice]=useState("");
 const [confirmation,setConfirmation]=useState<{action:"close"|"delete";plan:AIScheduledPlan}>();
 const cancelRef=useRef<HTMLButtonElement>(null);
 const storageKey=`ai-workbench-schedule-command:${scope.userId}:${scope.organizationId}`;
 const scheduleSearch=useSearchParams()?.toString()??(typeof window==="undefined"?"":window.location.search);
 useEffect(()=>{let active=true;queueMicrotask(()=>{if(!active)return;const params=new URLSearchParams(scheduleSearch);setSelected(params.get("schedule_id")??undefined);const proposal=params.get("schedule_proposal")??undefined;setProposalId(proposal);setForm(proposal?"create":undefined);});return()=>{active=false};},[scheduleSearch]);
 useEffect(()=>{let active=true;queueMicrotask(()=>{if(!active)return;try{const wire=sessionStorage.getItem(storageKey);if(wire){const command:unknown=JSON.parse(wire);if(validCommand(command))setPending(command);else setError("暂时无法核实上次操作，请保留页面并联系支持人员。");}}catch{setError("浏览器暂时无法记录操作，请检查浏览器设置后重试。");}setRestored(true);});return()=>{active=false};},[storageKey]);
 const prefix=["ai-schedules",scope.userId,scope.organizationId,authorizationKey];
 const availability=useQuery({queryKey:[...prefix,"availability"],queryFn:({signal})=>requestAIWorkbench({route:"schedule-availability",method:"GET",path:"schedules/availability",scope,signal}),retry:false});
 const plans=useInfiniteQuery({queryKey:[...prefix,"list",kindFilter],initialPageParam:"",queryFn:({signal,pageParam})=>requestAIWorkbench({route:"schedule-list",method:"GET",path:`schedules?limit=20${kindFilter?`&kind=${kindFilter}`:""}${pageParam?`&after=${pageParam}`:""}`,scope,signal}),getNextPageParam:(last,_pages,lastParam)=>last.next&&last.next!==lastParam?last.next:undefined,retry:false,refetchOnWindowFocus:false});
 const detail=useQuery({queryKey:[...prefix,selected],enabled:!!selected,queryFn:({signal})=>requestAIWorkbench({route:"schedule-read",method:"GET",path:`schedules/${selected}`,scope,signal}),retry:false,refetchOnWindowFocus:false});
 const history=useInfiniteQuery({queryKey:[...prefix,selected,"history"],enabled:!!selected,initialPageParam:"",queryFn:({signal,pageParam})=>requestAIWorkbench({route:"schedule-history",method:"GET",path:`schedules/${selected}/occurrences?limit=20${pageParam?`&after=${pageParam}`:""}`,scope,signal}),getNextPageParam:(last,_pages,lastParam)=>last.next&&last.next!==lastParam?last.next:undefined,retry:false,refetchOnWindowFocus:false});
 const locked=!restored||busy||!!pending;
 const plan=detail.isSuccess&&(!detail.isFetching||form==="edit"||form==="copy")?detail.data.plan:undefined;
 const planTitle=plan?`${plan.storeName?`${plan.storeName} · `:""}${plan.goalSummary??"当前无权查看任务内容"}`:"";
 function open(id?:string){setForm(undefined);setProposalId(undefined);setSelected(id);setError("");const url=new URL(window.location.href);url.searchParams.delete("schedule_proposal");if(id)url.searchParams.set("schedule_id",id);else url.searchParams.delete("schedule_id");window.history.replaceState(null,"",url);}
 async function execute(command:Command){
  if(busy)return;const recovering=!!pending;setError("");setBusy(true);
  try{
   sessionStorage.setItem(storageKey,JSON.stringify(command));setPending(command);
   const result=await requestAIWorkbench({route:`schedule-${command.action}`,method:command.action==="edit"?"PATCH":"POST",path:command.path,scope,key:command.key,body:command.body});
   sessionStorage.removeItem(storageKey);setPending(undefined);open(command.action==="delete"?undefined:result.plan.id);
   setNotice(command.action==="delete"?"计划已删除，已有结果和执行记录仍保留。":result.plan.registrationState==="MATCHED"?"计划已保存。":"计划已保存，启用状态仍在确认，请勿重复创建。");
   await client.invalidateQueries({queryKey:prefix});
  }catch(err){
   // Only authoritative rejection can retire a mutation key. A server 503 or
   // deadline may follow a committed DB write, just like a lost HTTP response.
   if(err instanceof AIWorkbenchError&&err.commandAccepted===false){sessionStorage.removeItem(storageKey);setPending(undefined);void client.invalidateQueries({queryKey:prefix});}
   else if(!recovering&&err instanceof AIWorkbenchError&&["INVALID_REQUEST","INPUT_TOO_LARGE","FORBIDDEN","NOT_FOUND","IDEMPOTENCY_CONFLICT","PROPOSAL_STALE","REVISION_MISMATCH","MANUAL_REVIEW_REQUIRED","AUTHENTICATION_REQUIRED","ORGANIZATION_SELECTION_REQUIRED","ORGANIZATION_ACCESS_DENIED","ORGANIZATION_ACCESS_REVOKED","ORGANIZATION_SUSPENDED","PERMISSION_DENIED","ORGANIZATION_CONTEXT_CHANGED","IDENTITY_CONTEXT_CHANGED"].includes(err.code)){sessionStorage.removeItem(storageKey);setPending(undefined);if(err.code==="REVISION_MISMATCH"||err.code==="PROPOSAL_STALE")void client.invalidateQueries({queryKey:prefix});}
   setError(err instanceof AIWorkbenchError?err.commandAccepted===false?"已核实原操作没有提交；可以调整内容后重新保存。":err.code==="REVISION_MISMATCH"?"计划已被更新。已重新读取最新状态，请核对后重新操作。":err.code==="MANUAL_REVIEW_REQUIRED"?"请先把智能体切换为默认或只读模式，并重新核对方案。":errorText(err):"暂时无法记录或恢复这次操作，请检查浏览器设置。");
  }finally{setBusy(false);}
 }
 async function save(input:AIScheduleInput){if(locked||!canUse)return;if(form==="edit"&&plan)await execute({action:"edit",path:`schedules/${plan.id}`,key:crypto.randomUUID(),body:{expectedRevision:input.expectedRevision,rule:input.rule,confirmation:input.confirmation,changes:{selection:input.selection,report:input.report,goalSummary:input.goalSummary,maxOccurrences:input.maxOccurrences}}});else await execute({action:"create",path:"schedules",key:crypto.randomUUID(),body:input});}
 function control(action:"pause"|"resume"|"close"|"delete",current:AIScheduledPlan){void execute({action,path:`schedules/${current.id}/${action}`,key:crypto.randomUUID(),body:{expectedRevision:current.revision}});}
 const items=Array.from(new Map(plans.data?.pages.flatMap(p=>p.plans).map(p=>[p.id,p])??[]).values());
 const formTitle=form==="edit"?"编辑计划":form==="copy"?"复制新建定时任务":proposalId?"将 Chat 方案设为定时任务":"创建定时任务";
 return <ConsolePage className={`${styles.page} ${form?styles.editorPage:""}`} title={form?formTitle:"定时任务"} description={form?form==="edit"?"调整后续安排，已经开始的任务和历史记录保持不变":form==="copy"?"带入原计划设置，核对新的执行时间后保存":"选择业务和对象，安排执行时间":"安排业务执行时间，查看每次处理结果"} actions={<Button variant="ghost" disabled={locked} onClick={form?()=>open(selected):onBack}>{form?"返回定时任务":"返回任务中心"}</Button>}>
  {confirmation?<ResourceDialog className={styles.dialog} title={confirmation.action==="delete"?"删除计划":"结束计划"} onClose={()=>setConfirmation(undefined)} initialFocusRef={cancelRef}>
   <p className={styles.dialogDescription}>{confirmation.action==="delete"?"删除后，此计划不再执行，也不再显示在定时列表中。":"结束后，此计划不再执行，且不能恢复。"}</p>
   <p className={styles.hint}>已经开始的任务会继续执行，已有报告和执行记录会保留。</p>
   <div className={styles.dialogActions}><Button ref={cancelRef} variant="outline" onClick={()=>setConfirmation(undefined)}>取消</Button><Button variant="destructive" disabled={locked} onClick={()=>{const current=confirmation;setConfirmation(undefined);control(current.action,current.plan);}}>{confirmation.action==="delete"?"删除计划":"结束计划"}</Button></div>
  </ResourceDialog>:null}
  <div className={styles.content}>
   {error?<p className={styles.error} role="alert">{error}</p>:null}{notice?<p className={styles.notice} role="status">{notice}</p>:null}
   {pending?<section className={styles.recovery}><div><h2>上次操作结果待核实</h2><p>请先核实上次操作的结果，再继续修改计划</p></div><Button disabled={busy} onClick={()=>void execute(pending)}>{busy?"正在核实…":"核实原操作"}</Button></section>:null}
   {availability.data&&!availability.data.ready?<p className={styles.warning}>暂时无法新建或恢复计划，已有计划仍可查看和暂停</p>:null}
   {form?(form!=="create"&&!plan?<ConsoleState kind={detail.isError?"error":"loading"} title="当前计划暂不可编辑"><Button variant="outline" onClick={()=>open(selected)}>返回计划</Button></ConsoleState>:<ScheduleForm key={`${form}:${selected??proposalId??"new"}`} scope={scope} authorizationKey={authorizationKey} plan={form==="edit"?plan:undefined} copyFrom={form==="copy"?plan:undefined} proposalId={proposalId} dailyReportEnabled={availability.data?.storeDailyReportEnabled} contentEnabled={availability.data?.contentEnabled??true} disabled={locked||detail.isFetching||!canUse||!availability.data?.ready} onSave={save} onStale={()=>void client.invalidateQueries({queryKey:[...prefix,selected],exact:true})} onCancel={()=>open(selected)}/>):selected?<>
    <div className={styles.backRow}><Button variant="ghost" disabled={locked} onClick={()=>open()}>返回定时列表</Button></div>
    {detail.isPending||detail.isFetching?<ConsoleState kind="loading" title="正在读取计划"/>:detail.isError?<ConsoleState kind="error" title="计划读取失败">{errorText(detail.error)}<Button variant="outline" onClick={()=>void detail.refetch()}>重试</Button></ConsoleState>:plan?<section className={styles.detail} aria-label="计划信息">
     <header className={styles.detailHeading}><div className={styles.planIdentity}><p className={styles.eyebrow}>{plan.kind==="store.operating-daily-report"?"店铺经营日报":"商品内容计划"}</p><h2 className={planTitle.length>100?styles.longTitle:undefined}>{planTitle}</h2></div><span className={styles.state} data-state={plan.deleted?"CLOSED":plan.state}><span aria-hidden="true"/>{plan.deleted?"已删除":states[plan.state]}</span></header>
     {planTitle.length>100?<details className={styles.request}><summary>查看完整计划内容</summary><p>{planTitle}</p></details>:null}
     <dl className={styles.facts}>
      <div><dt>时间规则</dt><dd>{plan.rule.frequency==="once"?"仅一次":plan.rule.frequency==="daily"?"每天":"每周"}{plan.rule.frequency!=="once"?` ${String(plan.rule.hour).padStart(2,"0")}:${String(plan.rule.minute).padStart(2,"0")}`:""}</dd><dd className={styles.hint}>{plan.rule.timeZone}</dd></div>
      <div><dt>下一次执行</dt><dd>{plan.nextAt?scheduleTime(plan.nextAt,plan.rule.timeZone):plan.state==="CLOSED"?"不再安排执行":plan.state==="PAUSED"?"已暂停":"执行时间待确认"}</dd></div>
      <div><dt>执行次数</dt><dd><span className={styles.figure}>{plan.consumedOccurrences}</span><span className={styles.hint}> / {plan.maxOccurrences} 次</span></dd></div>
      {plan.kind!=="store.operating-daily-report"?<div><dt>单次 / 计划积分上限</dt><dd>{schedulePoints(plan.perRunUpperBoundMicros)} / {schedulePoints(plan.totalUpperBoundMicros)}</dd><dd className={styles.hint}>AI 积分</dd></div>:<div><dt>报告保存</dt><dd>每次自动保存一份</dd></div>}
     </dl>
     {plan.activeOccurrenceId?<p className={styles.warning}>上一次任务仍在处理中，结果确认前会跳过后续执行。暂停、结束或删除计划不会取消已开始的任务。</p>:null}
     {plan.registrationState!=="MATCHED"?<p className={styles.warning}>计划已保存，启用状态仍在确认，请勿重复创建</p>:null}
     {canUse&&!plan.deleted?<div className={styles.detailActions}>
      <div className={styles.actions}>{plan.state!=="CLOSED"?<><Button variant="outline" disabled={locked||!plan.detailsAvailable||!availability.data?.ready} onClick={()=>setForm("edit")}>编辑计划</Button>{plan.state==="PAUSED"?<Button variant="outline" disabled={locked||!availability.data?.ready} onClick={()=>control("resume",plan)}>恢复未来执行</Button>:<Button variant="outline" disabled={locked} onClick={()=>control("pause",plan)}>暂停未来执行</Button>}</>:null}<Button variant="outline" disabled={locked||!plan.detailsAvailable||!availability.data?.ready} onClick={()=>setForm("copy")}>复制新建</Button></div>
      <div className={styles.secondaryActions}>{plan.state!=="CLOSED"?<Button variant="ghost" disabled={locked} onClick={()=>setConfirmation({action:"close",plan})}>结束计划</Button>:null}<Button variant="ghost" className={styles.deleteAction} disabled={locked} onClick={()=>setConfirmation({action:"delete",plan})}>删除计划</Button></div>
     </div>:null}
    </section>:null}
    <section className={styles.history} aria-label="执行历史"><div className={styles.sectionHeading}><h2>执行历史</h2><p>查看每次计划的处理状态与结果</p></div>
     {history.isPending?<p className={styles.hint} role="status">正在读取执行历史…</p>:null}
     {history.isError?<ConsoleState kind="error" title={history.data?"暂时无法刷新执行历史":"历史读取失败"}>{history.data?<p>以下为上次读取的记录</p>:null}<Button variant="outline" onClick={()=>void history.refetch()}>重试</Button></ConsoleState>:null}
     <ol className={styles.timeline}>{history.data?.pages.flatMap(p=>p.occurrences).map(item=><li key={item.id}><time className={styles.historyTime} dateTime={item.scheduledAt}>{scheduleTime(item.scheduledAt,plan?.rule.timeZone??"UTC")}</time><div className={styles.historyResult}><strong className={styles.state} data-state={item.state}><span aria-hidden="true"/>{occurrenceStates[item.state]}</strong>{item.reason?<p>{reasons[item.reason]??"执行结果仍在确认"}</p>:null}<div className={styles.resultLinks}>{item.reportId?<Link href={`/workbench/ai/reports?reportId=${item.reportId}`} prefetch={false}>查看经营日报</Link>:null}{item.businessTaskId?<Link href={`/workbench/ai/tasks/${item.businessTaskId}`} prefetch={false}>查看原任务与人工审核</Link>:null}</div></div></li>)}</ol>
     {history.isSuccess&&!history.data.pages[0].occurrences.length?<div className={styles.emptyHistory}><p>尚无触发记录</p><p className={styles.hint}>计划开始处理后，记录会显示在这里</p></div>:null}
     {history.hasNextPage?<Button className={styles.loadMore} variant="outline" disabled={history.isFetching} onClick={()=>void history.fetchNextPage()}>加载更早记录</Button>:null}
    </section>
   </>:<>
    <div className={styles.toolbar}><label className={styles.filter}>任务类型<Select value={kindFilter} disabled={locked} onChange={e=>setKindFilter(e.target.value)}><option value="">全部</option><option value="product.content.localize">商品内容</option><option value="store.operating-daily-report">店铺经营日报</option></Select></label><div className={styles.actions}>{canUse?<Button disabled={locked||!availability.data?.enabled||!availability.data.ready} onClick={()=>{setForm("create");setNotice("");}}>创建定时任务</Button>:null}<Button variant="outline" disabled={locked||plans.isFetching} onClick={()=>void client.invalidateQueries({queryKey:prefix})}>刷新状态</Button></div></div>
    {plans.isPending?<ConsoleState kind="loading" title="正在读取定时任务"/>:plans.isError?<ConsoleState kind="error" title="定时任务读取失败">{errorText(plans.error)}</ConsoleState>:!items.length?<ConsoleState kind="empty" title="还没有定时任务">选择要安排的业务和执行时间，创建计划</ConsoleState>:null}
    {items.length?<div className={styles.planList}><div className={styles.listHeading} aria-hidden="true"><span>计划</span><span>执行时间</span><span>状态</span><span>执行次数</span></div><ul className={styles.rows}>{items.map(item=><li key={item.id}><button disabled={locked} className={styles.row} onClick={()=>open(item.id)}>
     <span className={styles.rowIdentity}><strong>{item.storeName?`${item.storeName} · `:""}{item.goalSummary??"当前无权查看任务内容"}</strong><small>{item.kind==="store.operating-daily-report"?"店铺经营日报":"商品内容"} · {item.rule.frequency==="once"?"仅一次":item.rule.frequency==="daily"?"每天":"每周"}</small></span>
     <span className={styles.rowTime}>{item.nextAt?scheduleTime(item.nextAt,item.rule.timeZone):item.state==="CLOSED"?"不再安排执行":item.state==="PAUSED"?"已暂停":"执行时间待确认"}<small>{item.rule.timeZone}</small></span>
     <span className={styles.state} data-state={item.state}><span aria-hidden="true"/>{states[item.state]}</span><span className={styles.rowCount}>{item.consumedOccurrences} <small>/ {item.maxOccurrences} 次</small></span>
    </button></li>)}</ul></div>:null}
    {plans.hasNextPage?<Button className={styles.loadMore} variant="outline" disabled={locked||plans.isFetching} onClick={()=>void plans.fetchNextPage()}>加载更多计划</Button>:null}
   </>}
  </div>
 </ConsolePage>;
}
