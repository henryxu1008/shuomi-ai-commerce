"use client";
import {useEffect,useRef,useState} from "react";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {Textarea} from "@/components/ui/textarea";
import {requestAIWorkbench,AIWorkbenchError,type AIScope} from "@/lib/api/ai-workbench";
import {aiScheduleInput,type AIScheduleInput,type AIScheduleQuote,type AIScheduledPlan,type AITaskSubject} from "@/lib/contracts/ai-workbench";
import type {ContentSelection} from "@/lib/contracts/content-selection";
import {contentSelectionLabel} from "@/lib/contracts/content-selection";
import {ChatTaskSelection} from "./chat-task-selection";
import {errorText} from "./task-presentation";
import {DailyReportStoreSelect} from "./daily-report-store-select";
import styles from "./schedules.module.css";

export function schedulePoints(micros:string){const n=BigInt(micros),unit=BigInt(1000000);return `${n/unit}${n%unit?`.${String(n%unit).padStart(6,"0").replace(/0+$/,"")}`:""}`;}
export function scheduleTime(at:string,zone:string){return new Intl.DateTimeFormat("zh-CN",{timeZone:zone,dateStyle:"medium",timeStyle:"short"}).format(new Date(at));}

export function ScheduleForm({scope,authorizationKey,plan,copyFrom,proposalId,disabled,onSave,onCancel,onStale,dailyReportEnabled=false,contentEnabled=true}:{scope:AIScope;authorizationKey:string;dailyReportEnabled?:boolean;contentEnabled?:boolean;plan?:AIScheduledPlan;copyFrom?:AIScheduledPlan;proposalId?:string;disabled:boolean;onSave:(input:AIScheduleInput)=>Promise<void>;onCancel:()=>void;onStale?:()=>void}){
 const initial=plan??copyFrom;
 const [kind,setKind]=useState(initial?.kind??(initial||proposalId||contentEnabled?"product.content.localize":"store.operating-daily-report"));
 const report=kind==="store.operating-daily-report";
 const [storeId,setStoreId]=useState(initial?.report?.storeId??"");
 const [subject,setSubject]=useState<AITaskSubject|null>(initial?.selection?.subject??null),[parameters,setParameters]=useState<ContentSelection>(initial?.selection?.parameters??{targetLanguage:"en-US",description:true});
 const [goal,setGoal]=useState(initial?.goalSummary??""),[frequency,setFrequency]=useState<"once"|"daily"|"weekly">(initial?.rule.frequency??"once");
 const [zone,setZone]=useState(initial?.rule.timeZone??Intl.DateTimeFormat().resolvedOptions().timeZone);
 const [at,setAt]=useState(plan?.rule.atUtc?new Date(plan.rule.atUtc).toLocaleString("sv-SE").slice(0,16).replace(" ","T"):""),[time,setTime]=useState(initial?`${String(initial.rule.hour).padStart(2,"0")}:${String(initial.rule.minute).padStart(2,"0")}`:"09:00"),[weekday,setWeekday]=useState(initial?.rule.weekday??1);
 const [count,setCount]=useState(initial?.maxOccurrences??"1"),[until,setUntil]=useState(plan?.rule.untilUtc??"");
 const [quote,setQuote]=useState<{value:AIScheduleQuote;input:AIScheduleInput}>(),[quoting,setQuoting]=useState(false),[error,setError]=useState("");
 const abort=useRef<AbortController|null>(null);useEffect(()=>()=>abort.current?.abort(),[]);
 const revision=plan?.revision;useEffect(()=>{let active=true;queueMicrotask(()=>{if(active)setQuote(undefined)});return()=>{active=false}},[revision]);
 const locked=disabled||quoting;
 function change(update:()=>void){setQuote(undefined);setError("");update();}
 async function preview(){
  setError("");setQuote(undefined);
  try{
   const [hour,minute]=time.split(":").map(Number);
   const rule={frequency,timeZone:report?"Asia/Shanghai":zone,hour:frequency==="once"?0:hour,minute:frequency==="once"?0:minute,weekday:frequency==="weekly"?weekday:0,...(frequency==="once"?{atUtc:new Date(at).toISOString()}:{}),...(until?{untilUtc:new Date(until).toISOString()}: {})};
   const input=aiScheduleInput.parse({...(plan?{scheduleId:plan.id,expectedRevision:plan.revision}:copyFrom?{sourceScheduleId:copyFrom.id}:{}),...(proposalId?{proposalId}:report?{kind,report:{storeId,version:1,periodPolicy:"previous_natural_day",businessZone:"Asia/Shanghai"}}:{goalSummary:goal,selection:subject?{subject,capability:"product.content.localize",parameters}:undefined}),rule,maxOccurrences:frequency==="once"?"1":count});
   if(plan&&BigInt(input.maxOccurrences)<=BigInt(plan.consumedOccurrences)){setError(`次数上限需大于已执行的 ${plan.consumedOccurrences} 次，请调整后重新核对。`);return}
   const controller=new AbortController();abort.current=controller;setQuoting(true);
   const value=await requestAIWorkbench({route:"schedule-quote",method:"POST",path:"schedules/quote",scope,body:input,signal:controller.signal});
   if(!controller.signal.aborted)setQuote({value,input});
  }catch(err){if(err instanceof AIWorkbenchError&&["REVISION_MISMATCH","PROPOSAL_STALE"].includes(err.code))onStale?.();if(!abort.current?.signal.aborted)setError(err instanceof RangeError||err instanceof Error&&err.name==="ZodError"?"请完整填写任务、时间和次数，并选择有效的时区。":errorText(err));}
  finally{if(!abort.current?.signal.aborted)setQuoting(false);}
 }
 const quotedConsumed=quote?.value.consumedOccurrences??"0";
 const remaining=quote?BigInt(quote.value.maxOccurrences)-BigInt(quotedConsumed):BigInt(0);
 const futureUpperBound=quote?(BigInt(quote.value.perRunUpperBoundMicros)*remaining).toString():"0";
 return <div className={styles.form}>
  <header className={styles.formHeading}><h2>{plan?"编辑计划":copyFrom?"复制新建定时任务":proposalId?"将 Chat 方案设为定时任务":"创建定时任务"}</h2><p>{plan?"修改只用于后续执行，已经开始的任务和历史记录保持不变":copyFrom?"已带入原计划设置，核对新的执行时间后保存":report?"按计划整理一家店铺前一天的经营资料，自动保存到我的报告":"生成待审核的内容建议，由你确认后使用"}</p></header>
  {!initial&&!proposalId&&dailyReportEnabled&&contentEnabled?<label className={styles.kindField}>任务类型<Select disabled={locked} value={kind} onChange={e=>change(()=>{setKind(e.target.value as typeof kind);if(e.target.value==="store.operating-daily-report"&&frequency==="weekly")setFrequency("daily")})}><option value="product.content.localize">商品内容</option><option value="store.operating-daily-report">店铺经营日报</option></Select></label>:null}
  {proposalId?<p className={styles.selectedSummary}>沿用 Chat 方案的商品、目标和设置，确认后安排执行时间</p>:report?<section className={styles.formSection} aria-label="报告店铺"><DailyReportStoreSelect initialStore={initial?.report?{id:initial.report.storeId,name:initial.storeName||`店铺标识 ${initial.report.storeId}`}:undefined} scope={scope} authorizationKey={authorizationKey} value={storeId} disabled={locked} onChange={id=>change(()=>setStoreId(id))}/></section>:<>
   <div className={styles.subjectGroup}><ChatTaskSelection compactSummary scope={scope} authorizationKey={authorizationKey} disabled={locked} subject={subject} parameters={parameters} onSubject={value=>change(()=>setSubject(value))} onParameters={value=>change(()=>setParameters(value))}/></div>
   <label className={styles.field}>任务目标<Textarea rows={3} maxLength={8192} value={goal} disabled={locked} placeholder="说明希望处理的内容与目标" onChange={event=>change(()=>setGoal(event.target.value))}/></label>
  </>}
  <fieldset className={styles.formSection} disabled={locked}><legend>执行时间</legend><div className={styles.fields}>
   <label className={styles.field}>频率<Select value={frequency} onChange={event=>change(()=>setFrequency(event.target.value as typeof frequency))}><option value="once" disabled={!!plan&&plan.consumedOccurrences!=="0"}>仅一次</option><option value="daily">每天</option>{!report?<option value="weekly">每周</option>:null}</Select></label>
   <label className={styles.field}>计划时区<Input value={report?"Asia/Shanghai":zone} disabled={report} placeholder="Asia/Shanghai" onChange={event=>change(()=>setZone(event.target.value))}/></label>
   {frequency==="once"?<label className={styles.field}>执行日期与时间（本机时区）<Input type="datetime-local" value={at} onChange={event=>change(()=>setAt(event.target.value))}/></label>:<>
    <label className={styles.field}>执行时刻（计划时区）<Input type="time" value={time} onChange={event=>change(()=>setTime(event.target.value))}/></label>
    {frequency==="weekly"?<label className={styles.field}>星期<Select value={weekday} onChange={event=>change(()=>setWeekday(Number(event.target.value)))}>{["日","一","二","三","四","五","六"].map((day,i)=><option value={i} key={i}>星期{day}</option>)}</Select></label>:null}
    <label className={styles.field}>最多执行次数<Input inputMode="numeric" value={count} onChange={event=>change(()=>setCount(event.target.value))}/></label>
    <label className={styles.field}>截止时间（可选，本机时区）<Input type="datetime-local" value={until?new Date(until).toLocaleString("sv-SE").slice(0,16).replace(" ","T"):""} onChange={event=>change(()=>setUntil(event.target.value?new Date(event.target.value).toISOString():""))}/></label>
   </>}
  </div></fieldset>
  {plan?<p className={styles.hint}>已执行 {plan.consumedOccurrences} 次，次数上限应大于已执行次数。{plan.state==="PAUSED"?"保存后仍保持暂停。":""}</p>:null}
  {error?<p className={styles.error} role="alert">{error}</p>:null}
  <div className={styles.formActions}><Button variant={quote?"outline":"default"} disabled={locked} onClick={()=>void preview()}>{quoting?"正在核对…":report?"核对计划":"核对时间与预算"}</Button><Button variant="ghost" disabled={locked} onClick={onCancel}>返回</Button></div>
  {quote?<section className={styles.confirm} aria-label="确认定时计划"><header className={styles.sectionHeading}><h3>确认执行范围</h3><p>核对以下内容，确认后保存计划</p></header><p className={styles.confirmTitle}>{quote.value.storeName?`${quote.value.storeName} · `:""}{quote.value.goalSummary}</p>{quote.value.selection?<p className={styles.hint}>{contentSelectionLabel(quote.value.selection.parameters)}</p>:<p className={styles.hint}>统计 UTC+8 前一天的已保存资料；缺少数据时会明确说明，不会显示成零。</p>}
   {quote.value.selection?<details className={styles.request}><summary>查看所选商品资料</summary><p>商品标识 {quote.value.selection.subject.id}</p><p>资料版本 {quote.value.selection.subject.catalogVersion}</p></details>:null}
   <dl className={styles.confirmFacts}><div><dt>下一次执行</dt><dd>{scheduleTime(quote.value.nextAt,quote.value.rule.timeZone)}</dd><dd className={styles.hint}>{quote.value.rule.timeZone}</dd></div><div><dt>最多执行</dt><dd>{quote.value.maxOccurrences} 次</dd>{plan?<dd className={styles.hint}>已执行 {quotedConsumed} 次，后续可执行 {remaining.toString()} 次</dd>:null}</div>{report?<div><dt>报告保存</dt><dd>每次执行自动保存一份个人报告</dd></div>:<><div><dt>单次积分上限</dt><dd>{schedulePoints(quote.value.perRunUpperBoundMicros)} AI 积分</dd></div><div><dt>{plan?"后续执行积分上限":"整个计划积分上限"}</dt><dd>{schedulePoints(plan?futureUpperBound:quote.value.totalUpperBoundMicros)} AI 积分</dd></div><div><dt>执行方式</dt><dd>{quote.value.executionMode==="read_only"?"只读建议":"生成建议 · 人工审核"}</dd></div></>}</dl>
   {!report?<p className={styles.hint}>以上为执行上限，实际按已确认用量扣除积分。上次结果未确认时，后续执行会跳过，避免重复启动。</p>:null}
   <div className={styles.actions}><Button disabled={locked} onClick={()=>void onSave({...quote.input,confirmation:quote.value.confirmation})}>{plan?"确认修改计划":"确认并保存计划"}</Button></div>
  </section>:null}
 </div>;
}
