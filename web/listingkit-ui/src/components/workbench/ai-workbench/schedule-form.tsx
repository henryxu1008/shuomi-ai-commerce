"use client";
import {useEffect,useRef,useState} from "react";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {requestAIWorkbench,type AIScope} from "@/lib/api/ai-workbench";
import {aiScheduleInput,type AIScheduleInput,type AIScheduleQuote,type AIScheduledPlan,type AITaskSubject} from "@/lib/contracts/ai-workbench";
import type {ContentSelection} from "@/lib/contracts/content-selection";
import {contentSelectionLabel} from "@/lib/contracts/content-selection";
import {ChatTaskSelection} from "./chat-task-selection";
import {errorText} from "./task-presentation";
import {DailyReportStoreSelect} from "./daily-report-store-select";
import styles from "./schedules.module.css";

export function schedulePoints(micros:string){const n=BigInt(micros),unit=BigInt(1000000);return `${n/unit}${n%unit?`.${String(n%unit).padStart(6,"0").replace(/0+$/,"")}`:""}`;}
export function scheduleTime(at:string,zone:string){return new Intl.DateTimeFormat("zh-CN",{timeZone:zone,dateStyle:"medium",timeStyle:"short"}).format(new Date(at));}

export function ScheduleForm({scope,authorizationKey,plan,proposalId,disabled,onSave,onCancel,dailyReportEnabled=false,contentEnabled=true}:{scope:AIScope;authorizationKey:string;dailyReportEnabled?:boolean;contentEnabled?:boolean;plan?:AIScheduledPlan;proposalId?:string;disabled:boolean;onSave:(input:AIScheduleInput)=>Promise<void>;onCancel:()=>void}){
 const [kind,setKind]=useState(plan?.kind??(proposalId||contentEnabled?"product.content.localize":"store.operating-daily-report"));
 const report=kind==="store.operating-daily-report";
 const [storeId,setStoreId]=useState(plan?.report?.storeId??"");
 const [subject,setSubject]=useState<AITaskSubject|null>(null),[parameters,setParameters]=useState<ContentSelection>({targetLanguage:"en-US",description:true});
 const [goal,setGoal]=useState(""),[frequency,setFrequency]=useState<"once"|"daily"|"weekly">(plan?.rule.frequency??"once");
 const [zone,setZone]=useState(plan?.rule.timeZone??Intl.DateTimeFormat().resolvedOptions().timeZone);
 const [at,setAt]=useState(""),[time,setTime]=useState(plan?`${String(plan.rule.hour).padStart(2,"0")}:${String(plan.rule.minute).padStart(2,"0")}`:"09:00"),[weekday,setWeekday]=useState(plan?.rule.weekday??1);
 const [count,setCount]=useState(plan?.maxOccurrences??"1"),[until,setUntil]=useState(plan?.rule.untilUtc??"");
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
   const input=aiScheduleInput.parse({...(plan?{scheduleId:plan.id}:proposalId?{proposalId}:report?{kind,report:{storeId,version:1,periodPolicy:"previous_natural_day",businessZone:"Asia/Shanghai"}}:{goalSummary:goal,selection:subject?{subject,capability:"product.content.localize",parameters}:undefined}),rule,maxOccurrences:frequency==="once"?"1":count});
   const controller=new AbortController();abort.current=controller;setQuoting(true);
   const value=await requestAIWorkbench({route:"schedule-quote",method:"POST",path:"schedules/quote",scope,body:input,signal:controller.signal});
   if(!controller.signal.aborted)setQuote({value,input});
  }catch(err){if(!abort.current?.signal.aborted)setError(err instanceof RangeError||err instanceof Error&&err.name==="ZodError"?"请完整填写任务、时间和次数，并选择有效的时区。":errorText(err));}
  finally{if(!abort.current?.signal.aborted)setQuoting(false);}
 }
 return <div className={styles.form}>
  <h2>{plan?"修改执行时间":proposalId?"将 Chat 方案设为定时任务":"创建定时任务"}</h2>
  <p>{report?"按所选时间整理一家店铺前一天的经营资料，并自动保存到我的报告。":"自动执行仅生成待人工审核的内容建议，不会自动采用或发布。"}</p>
  {!plan&&!proposalId&&dailyReportEnabled&&contentEnabled?<label>任务类型<select disabled={locked} value={kind} onChange={e=>change(()=>{setKind(e.target.value as typeof kind);if(e.target.value==="store.operating-daily-report"&&frequency==="weekly")setFrequency("daily")})}><option value="product.content.localize">商品内容</option><option value="store.operating-daily-report">店铺经营日报</option></select></label>:null}
  {plan?<p>{plan.storeName?`${plan.storeName} · `:""}{plan.goalSummary}{plan.selection?` · ${contentSelectionLabel(plan.selection.parameters)}`:""}</p>:proposalId?<p>沿用 Chat 方案的商品、目标、模板和工作范围。下一步将核对方案是否仍可执行。</p>:report?<DailyReportStoreSelect scope={scope} authorizationKey={authorizationKey} value={storeId} disabled={locked} onChange={id=>change(()=>setStoreId(id))}/>:<>
   <ChatTaskSelection scope={scope} authorizationKey={authorizationKey} disabled={locked} subject={subject} parameters={parameters} onSubject={value=>change(()=>setSubject(value))} onParameters={value=>change(()=>setParameters(value))}/>
   <label>任务目标<textarea rows={3} maxLength={8192} value={goal} disabled={locked} onChange={event=>change(()=>setGoal(event.target.value))}/></label>
  </>}
  <fieldset disabled={locked}><legend>执行时间</legend><div className={styles.fields}>
   <label>频率<select value={frequency} onChange={event=>change(()=>setFrequency(event.target.value as typeof frequency))}><option value="once" disabled={!!plan&&plan.maxOccurrences!=="1"}>仅一次</option><option value="daily">每天</option>{!report?<option value="weekly">每周</option>:null}</select></label>
   <label>计划时区<Input value={report?"Asia/Shanghai":zone} disabled={report} placeholder="Asia/Shanghai" onChange={event=>change(()=>setZone(event.target.value))}/></label>
   {frequency==="once"?<label>执行日期与时间（本机时区）<Input type="datetime-local" value={at} onChange={event=>change(()=>setAt(event.target.value))}/></label>:<>
    <label>执行时刻（计划时区）<Input type="time" value={time} onChange={event=>change(()=>setTime(event.target.value))}/></label>
    {frequency==="weekly"?<label>星期<select value={weekday} onChange={event=>change(()=>setWeekday(Number(event.target.value)))}>{["日","一","二","三","四","五","六"].map((day,i)=><option value={i} key={i}>星期{day}</option>)}</select></label>:null}
    <label>最多执行次数<Input inputMode="numeric" value={count} disabled={!!plan} onChange={event=>change(()=>setCount(event.target.value))}/></label>
    <label>截止时间（可选，本机时区）<Input type="datetime-local" value={until?new Date(until).toLocaleString("sv-SE").slice(0,16).replace(" ","T"):""} onChange={event=>change(()=>setUntil(event.target.value?new Date(event.target.value).toISOString():""))}/></label>
   </>}
  </div></fieldset>
  {plan?<p>{report?"只修改执行时间，所选店铺、统计口径和次数上限保持不变。":"只修改时间规则；商品、目标、执行配置和次数预算保持原计划。已开始的执行继续由原任务处理。"}</p>:null}
  {error?<p role="alert">{error}</p>:null}
  <div className={styles.actions}><Button variant="outline" disabled={locked} onClick={()=>void preview()}>{quoting?"正在核对…":report?"核对计划":"核对时间与预算"}</Button><Button variant="outline" disabled={locked} onClick={onCancel}>返回</Button></div>
  {quote?<section className={styles.confirm} aria-label="确认定时计划"><h3>确认执行范围</h3><p>{quote.value.storeName?`${quote.value.storeName} · `:""}{quote.value.goalSummary}</p>{quote.value.selection?<p>{contentSelectionLabel(quote.value.selection.parameters)} · 商品版本 {quote.value.selection.subject.catalogVersion}</p>:<p>统计 UTC+8 前一天的已保存资料；缺少数据时会明确说明，不会显示成零。</p>}
   <dl><dt>下一次执行</dt><dd>{scheduleTime(quote.value.nextAt,quote.value.rule.timeZone)}（{quote.value.rule.timeZone}）</dd><dt>最多执行</dt><dd>{quote.value.maxOccurrences} 次</dd>{report?<><dt>报告保存</dt><dd>每次执行自动保存一份个人报告</dd></>:<><dt>单次积分上限</dt><dd>{schedulePoints(quote.value.perRunUpperBoundMicros)} AI 积分</dd><dt>整个计划积分上限</dt><dd>{schedulePoints(quote.value.totalUpperBoundMicros)} AI 积分</dd><dt>执行模式</dt><dd>{quote.value.executionMode==="read_only"?"只读建议":"默认 · 人工审核"}</dd></>}</dl>
   {!report?<p>这里是执行上限，实际扣费由原积分服务按已确认用量结算。结果未知时保留原任务占用，后续时刻会跳过，不重复扣次启动。</p>:null}
   <Button disabled={locked} onClick={()=>void onSave({...quote.input,confirmation:quote.value.confirmation})}>{plan?"确认修改时间":"确认并保存计划"}</Button>
  </section>:null}
 </div>;
}
