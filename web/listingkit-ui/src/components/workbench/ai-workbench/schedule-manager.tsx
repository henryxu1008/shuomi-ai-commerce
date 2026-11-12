"use client";
import Link from "next/link";
import {useEffect,useState} from "react";
import {useInfiniteQuery,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button} from "@/components/ui/button";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {requestAIWorkbench,AIWorkbenchError,type AIScope} from "@/lib/api/ai-workbench";
import {aiScheduleInput,aiScheduleControl,aiScheduleEdit,aiWorkbenchPath,type AIScheduleInput,type AIScheduledPlan} from "@/lib/contracts/ai-workbench";
import {ScheduleForm,schedulePoints,scheduleTime} from "./schedule-form";
import {errorText} from "./task-presentation";
import styles from "./schedules.module.css";

const states:Record<string,string>={ACTIVE:"已启用",PAUSED:"已暂停",CLOSED:"已结束",PENDING_REGISTRATION:"等待调度登记",REGISTRATION_UNKNOWN:"调度登记待核实"};
const occurrenceStates:Record<string,string>={CLAIMED:"已创建任务",DISPATCHED:"已交接执行",SKIPPED:"已跳过",BLOCKED:"执行前受阻",UNKNOWN:"执行结果待核实",RECONCILED:"原任务已核实"};
const reasons:Record<string,string>={RUN_QUIESCENT:"原任务已停止自动执行，已知用量已核实；建议仍需人工处理",PLAN_NOT_ACTIVE:"计划当时未启用",PREVIOUS_RUN_ACTIVE:"前一次执行仍被占用",MAX_OCCURRENCES:"已达到次数上限",CATCHUP_EXPIRED:"已超过 15 分钟补触发窗口",MISSED_TRIGGER:"调度中断期间的历史时刻，不补付费执行",CURRENT_AUTHORIZATION_UNAVAILABLE:"当前企业、成员或操作权限校验未通过",FROZEN_EXECUTION_UNAVAILABLE:"冻结的商品、范围、配置或预算当前不可执行",EXECUTION_HANDOFF_UNRESOLVED:"已交接原任务，尚未确认执行结果",ORIGINAL_RUN_OUTCOME_UNRESOLVED:"原任务或计量结果尚未确认，继续保留占用"};
type Command={action:"create"|"edit"|"pause"|"resume"|"close";path:string;key:string;body:object};
function validCommand(value:unknown):value is Command{
 if(!value||typeof value!=="object")return false;
 const v=value as Command;if(!["create","edit","pause","resume","close"].includes(v.action)||typeof v.path!=="string"||!/^[-0-9a-f]{36}$/.test(v.key))return false;
 const method=v.action==="edit"?"PATCH":"POST";
 return aiWorkbenchPath(method,v.path.split("/"))===`schedule-${v.action}`&&(v.action==="create"?aiScheduleInput:v.action==="edit"?aiScheduleEdit:aiScheduleControl).safeParse(v.body).success;
}

export function ScheduleManager({scope,authorizationKey,canUse,onBack}:{scope:AIScope;authorizationKey:string;canUse:boolean;onBack:()=>void}){
 const client=useQueryClient(),[selected,setSelected]=useState<string>(),[form,setForm]=useState<"create"|"edit">(),[proposalId,setProposalId]=useState<string>();
 const [pending,setPending]=useState<Command>(),[busy,setBusy]=useState(false),[restored,setRestored]=useState(false),[error,setError]=useState(""),[notice,setNotice]=useState("");
 const storageKey=`ai-workbench-schedule-command:${scope.userId}:${scope.organizationId}`;
 useEffect(()=>{let active=true;queueMicrotask(()=>{if(!active)return;const params=new URLSearchParams(window.location.search);setSelected(params.get("schedule_id")??undefined);if(params.get("schedule_proposal")){setProposalId(params.get("schedule_proposal")!);setForm("create");}try{const wire=sessionStorage.getItem(storageKey);if(wire){const command:unknown=JSON.parse(wire);if(validCommand(command))setPending(command);else setError("保存的命令无法核实，请保留页面并联系维护者。");}}catch{setError("浏览器存储不可用，不能安全保存重试命令。");}setRestored(true);});return()=>{active=false};},[storageKey]);
 const prefix=["ai-schedules",scope.userId,scope.organizationId,authorizationKey];
 const availability=useQuery({queryKey:[...prefix,"availability"],queryFn:({signal})=>requestAIWorkbench({route:"schedule-availability",method:"GET",path:"schedules/availability",scope,signal}),retry:false});
 const plans=useInfiniteQuery({queryKey:[...prefix,"list"],initialPageParam:"",queryFn:({signal,pageParam})=>requestAIWorkbench({route:"schedule-list",method:"GET",path:`schedules?limit=20${pageParam?`&after=${pageParam}`:""}`,scope,signal}),getNextPageParam:(last,_pages,lastParam)=>last.next&&last.next!==lastParam?last.next:undefined,retry:false,refetchOnWindowFocus:false});
 const detail=useQuery({queryKey:[...prefix,selected],enabled:!!selected,queryFn:({signal})=>requestAIWorkbench({route:"schedule-read",method:"GET",path:`schedules/${selected}`,scope,signal}),retry:false,refetchOnWindowFocus:false});
 const history=useInfiniteQuery({queryKey:[...prefix,selected,"history"],enabled:!!selected,initialPageParam:"",queryFn:({signal,pageParam})=>requestAIWorkbench({route:"schedule-history",method:"GET",path:`schedules/${selected}/occurrences?limit=20${pageParam?`&after=${pageParam}`:""}`,scope,signal}),getNextPageParam:(last,_pages,lastParam)=>last.next&&last.next!==lastParam?last.next:undefined,retry:false,refetchOnWindowFocus:false});
 const locked=!restored||busy||!!pending;
 const plan=detail.isSuccess&&(!detail.isFetching||form==="edit")?detail.data.plan:undefined;
 function open(id?:string){setForm(undefined);setProposalId(undefined);setSelected(id);setError("");const url=new URL(window.location.href);url.searchParams.delete("schedule_proposal");if(id)url.searchParams.set("schedule_id",id);else url.searchParams.delete("schedule_id");window.history.replaceState(null,"",url);}
 async function execute(command:Command){
  if(busy)return;const recovering=!!pending;setError("");setBusy(true);
  try{
   sessionStorage.setItem(storageKey,JSON.stringify(command));setPending(command);
   const result=await requestAIWorkbench({route:`schedule-${command.action}`,method:command.action==="edit"?"PATCH":"POST",path:command.path,scope,key:command.key,body:command.body});
   sessionStorage.removeItem(storageKey);setPending(undefined);open(result.plan.id);
   setNotice(result.plan.registrationState==="MATCHED"?"计划已保存，调度登记已核实。":"计划已保存；调度登记尚未核实，后台会继续核对，请勿重复创建。");
   await client.invalidateQueries({queryKey:prefix});
  }catch(err){
   // Only authoritative rejection can retire a mutation key. A server 503 or
   // deadline may follow a committed DB write, just like a lost HTTP response.
   if(err instanceof AIWorkbenchError&&err.commandAccepted===false){sessionStorage.removeItem(storageKey);setPending(undefined);void client.invalidateQueries({queryKey:prefix});}
   else if(!recovering&&err instanceof AIWorkbenchError&&["INVALID_REQUEST","INPUT_TOO_LARGE","FORBIDDEN","NOT_FOUND","IDEMPOTENCY_CONFLICT","PROPOSAL_STALE","REVISION_MISMATCH","MANUAL_REVIEW_REQUIRED","AUTHENTICATION_REQUIRED","ORGANIZATION_SELECTION_REQUIRED","ORGANIZATION_ACCESS_DENIED","ORGANIZATION_ACCESS_REVOKED","ORGANIZATION_SUSPENDED","PERMISSION_DENIED","ORGANIZATION_CONTEXT_CHANGED","IDENTITY_CONTEXT_CHANGED"].includes(err.code)){sessionStorage.removeItem(storageKey);setPending(undefined);if(err.code==="REVISION_MISMATCH"||err.code==="PROPOSAL_STALE")void client.invalidateQueries({queryKey:prefix});}
   setError(err instanceof AIWorkbenchError?err.commandAccepted===false?"已核实原操作没有提交；可以调整内容后重新保存。":err.code==="REVISION_MISMATCH"?"计划已被更新。已重新读取最新状态，请核对后重新操作。":err.code==="MANUAL_REVIEW_REQUIRED"?"请先把智能体切换为默认或只读模式，并重新核对方案。":errorText(err):"浏览器无法保存或恢复重试命令，请检查存储是否可用。");
  }finally{setBusy(false);}
 }
 async function save(input:AIScheduleInput){if(locked||!canUse)return;if(form==="edit"&&plan)await execute({action:"edit",path:`schedules/${plan.id}`,key:crypto.randomUUID(),body:{expectedRevision:plan.revision,rule:input.rule,confirmation:input.confirmation}});else await execute({action:"create",path:"schedules",key:crypto.randomUUID(),body:input});}
 function control(action:"pause"|"resume"|"close",current:AIScheduledPlan){void execute({action,path:`schedules/${current.id}/${action}`,key:crypto.randomUUID(),body:{expectedRevision:current.revision}});}
 const items=Array.from(new Map(plans.data?.pages.flatMap(p=>p.plans).map(p=>[p.id,p])??[]).values());
 return <ConsolePage title="定时任务" description="为选定的商品安排有次数和积分上限的内容任务。" actions={<Button variant="outline" disabled={locked} onClick={onBack}>返回业务任务</Button>}>
  {error?<p role="alert">{error}</p>:null}{notice?<p role="status">{notice}</p>:null}
  {pending?<section className={styles.confirm}><h2>上次操作结果待核实</h2><p>使用已保存的同一命令核实结果，避免重复创建或覆盖更新。</p><Button disabled={busy} onClick={()=>void execute(pending)}>{busy?"正在核实…":"核实原操作"}</Button></section>:null}
  {availability.data&&!availability.data.ready?<p className={styles.warning}>调度服务当前未就绪。已有计划仍可查看和暂停；新建与恢复请等待服务恢复。</p>:null}
  {form?(form==="edit"&&!plan?<ConsoleState kind={detail.isError?"error":"loading"} title="当前计划暂不可编辑"><Button variant="outline" onClick={()=>open(selected)}>返回计划</Button></ConsoleState>:<ScheduleForm key={`${form}:${selected??proposalId??"new"}`} scope={scope} authorizationKey={authorizationKey} plan={form==="edit"?plan:undefined} proposalId={proposalId} disabled={locked||detail.isFetching||!canUse||!availability.data?.ready} onSave={save} onCancel={()=>open(selected)}/>):selected?<>
   <Button variant="outline" disabled={locked} onClick={()=>open()}>返回定时列表</Button>
   {detail.isPending||detail.isFetching?<ConsoleState kind="loading" title="正在读取计划"/>:detail.isError?<ConsoleState kind="error" title="计划读取失败">{errorText(detail.error)}<Button variant="outline" onClick={()=>void detail.refetch()}>重试</Button></ConsoleState>:plan?<section className={styles.detail}>
    <h2>{plan.goalSummary??"当前无权查看任务内容"}</h2><p>{states[plan.state]} · 修订 {plan.revision}</p>
    <dl><dt>时间规则</dt><dd>{plan.rule.frequency==="once"?"仅一次":plan.rule.frequency==="daily"?"每天":"每周"} · {plan.rule.timeZone}{plan.rule.frequency!=="once"?` · ${String(plan.rule.hour).padStart(2,"0")}:${String(plan.rule.minute).padStart(2,"0")}`:""}</dd><dt>下一次执行</dt><dd>{plan.nextAt?scheduleTime(plan.nextAt,plan.rule.timeZone):"当前无可确认的下次执行时间"}</dd><dt>已消耗次数 / 上限</dt><dd>{plan.consumedOccurrences} / {plan.maxOccurrences}</dd><dt>单次 / 计划积分上限</dt><dd>{schedulePoints(plan.perRunUpperBoundMicros)} / {schedulePoints(plan.totalUpperBoundMicros)} AI 积分</dd></dl>
    {plan.activeOccurrenceId?<p className={styles.warning}>前一次执行仍占用本计划；在原任务及计量结果核实前，后续触发会跳过。暂停、修改时间和关闭都不会取消原任务或释放占用。</p>:null}
    {plan.registrationState!=="MATCHED"?<p className={styles.warning}>计划事实已保存，调度登记尚未核实。此时不显示“已启用”的成功状态。</p>:null}
    {canUse&&plan.state!=="CLOSED"?<div className={styles.actions}>
     <Button variant="outline" disabled={locked||!plan.detailsAvailable||!availability.data?.ready} onClick={()=>setForm("edit")}>修改时间</Button>
     {plan.state==="PAUSED"?<Button disabled={locked||!availability.data?.ready} onClick={()=>control("resume",plan)}>恢复未来执行</Button>:<Button variant="outline" disabled={locked} onClick={()=>control("pause",plan)}>暂停未来执行</Button>}
     <Button variant="outline" disabled={locked} onClick={()=>{if(window.confirm("结束此计划的未来执行？已开始的原任务仍保留，结束后不能恢复。"))control("close",plan)}}>结束计划</Button>
    </div>:null}
   </section>:null}
   <section className={styles.history}><h2>执行历史</h2><p>记录每个触发时刻。跳过和执行前阻断不消耗次数；“原任务已核实”不代表建议已被采用。</p>
    {history.isError?<ConsoleState kind="error" title="历史读取失败"><Button variant="outline" onClick={()=>void history.refetch()}>重试</Button></ConsoleState>:null}
    <ol>{history.data?.pages.flatMap(p=>p.occurrences).map(item=><li key={item.id}><strong>{scheduleTime(item.scheduledAt,plan?.rule.timeZone??"UTC")} · {occurrenceStates[item.state]}</strong><p>{reasons[item.reason??""]??(item.reason?"原任务状态尚待核对":"")}</p>{item.businessTaskId?<Link href={`/workbench/ai/tasks/${item.businessTaskId}`} prefetch={false}>查看原任务与人工审核</Link>:null}</li>)}</ol>
    {history.isSuccess&&!history.data.pages[0].occurrences.length?<p>尚无触发记录。</p>:null}
    {history.hasNextPage?<Button variant="outline" disabled={history.isFetching} onClick={()=>void history.fetchNextPage()}>加载更早记录</Button>:null}
   </section>
  </>:<>
   <div className={styles.actions}>{canUse?<Button disabled={locked||!availability.data?.enabled||!availability.data.ready} onClick={()=>{setForm("create");setNotice("");}}>创建定时任务</Button>:null}<Button variant="outline" disabled={locked||plans.isFetching} onClick={()=>void client.invalidateQueries({queryKey:prefix})}>刷新状态</Button></div>
   {plans.isPending?<ConsoleState kind="loading" title="正在读取定时任务"/>:plans.isError?<ConsoleState kind="error" title="定时任务读取失败">{errorText(plans.error)}</ConsoleState>:!items.length?<ConsoleState kind="empty" title="还没有定时任务">选择商品、时间和次数，确认预算后保存。</ConsoleState>:null}
   <ul className={styles.rows}>{items.map(item=><li key={item.id}><button disabled={locked} className={styles.row} onClick={()=>open(item.id)}><strong>{item.goalSummary??"当前无权查看任务内容"}</strong><span>{states[item.state]} · 已消耗 {item.consumedOccurrences} / {item.maxOccurrences} 次</span><small>{item.nextAt?`下一次：${scheduleTime(item.nextAt,item.rule.timeZone)}（${item.rule.timeZone}）`:"当前无可确认的下次执行时间"}</small></button></li>)}</ul>
   {plans.hasNextPage?<Button variant="outline" disabled={locked||plans.isFetching} onClick={()=>void plans.fetchNextPage()}>加载更多计划</Button>:null}
  </>}
 </ConsolePage>;
}
