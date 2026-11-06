"use client";

import Link from "next/link";
import Image from "next/image";
import {useEffect,useRef,useState,type ReactNode,type Ref} from "react";
import {z} from "zod";
import {Badge} from "@/components/ui/badge";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {Table,TableBody,TableCell,TableHead,TableHeader,TableRow} from "@/components/ui/table";
import {Textarea} from "@/components/ui/textarea";
import {ConsolePage} from "../console/console-page";
import {findConsoleRoute} from "@/lib/workbench/console-navigation";
import {AcquisitionAPIError,type AcquisitionContext} from "@/lib/api/product-acquisition";
import * as api from "@/lib/api/acquisition-plan";
import {createPlanSchema,ruleMutationSchema,acquisitionRuleSchema,type AcquisitionPlan,type CreatePlan,type RuleTemplate,type RuleMutation} from "@/lib/contracts/acquisition-plan";
import {canonical1688Source,isAcquisitionUUID} from "@/lib/contracts/product-acquisition";
import {productAgentOperationHref} from "@/lib/contracts/product-agent";
import styles from "./acquisition-plans.module.css";

const modes=[{id:"product",label:"指定商品",field:"1688 商品链接",placeholder:"粘贴 1688 商品链接，支持批量输入，每行一条"},{id:"page",label:"按页采集",field:"1688 页面链接",placeholder:"粘贴公开搜索结果页或店铺商品列表页链接，每行一条"},{id:"keyword",label:"按关键词",field:"采集关键词",placeholder:"输入关键词，支持批量输入，每行一个关键词"}] as const;
const messages:Record<string,string>={INVALID_ACQUISITION:"请检查来源链接、价格范围和起批量。",INVALID_REQUEST:"请检查输入。",FORBIDDEN:"当前身份没有所需权限，请确认当前企业和授权。",ACQUISITION_UNAVAILABLE:"采集服务暂不可用，已发出的请求请按原计划核实。",ACQUISITION_NOT_FOUND:"原计划尚未读到，可用原请求再次确认创建。",IDEMPOTENCY_CONFLICT:"计划正在处理或版本已变化，请先读取最新计划。",OUTCOME_UNKNOWN:"请求结果暂未确认，请读取原计划后继续。",DEADLINE_EXCEEDED:"请求超时，请读取原计划后继续。",SOURCE_TOO_LARGE:"输入或返回内容超过上限。",collector_capacity_or_cooldown:"采集器处于限速或冷却状态，请稍后继续原计划。",discovery_unavailable:"公开列表暂时无法读取，请稍后继续原计划。",transfer_source_conflict:"来源批次已移动、归档或版本变化，原计划不能继续转入；已保存商品和已转入批次保留，请从我的数据重新选择。",transfer_needs_verification:"转入结果待核实，继续时将读取原转入回执。",capacity_or_quota_unavailable:"数据额度不足或采集器繁忙，请补充额度或稍后继续。",outcome_unknown:"该商品结果待核实，继续时按原请求恢复。",source_unavailable:"来源不可用或需要登录、验证。",source_facts_unavailable:"规则所需公开事实缺失。",source_currency_unavailable:"公开价格缺少明确人民币币种。",source_moq_unavailable:"公开最小起订量缺失。"};
const failureText=(e:unknown)=>e instanceof AcquisitionAPIError?messages[e.code]??`请求未完成（${e.code}）`:"请求未完成，请先读取原计划核实。";
const storage=(scope:AcquisitionContext,kind:string)=>`listingkit:acquisition:${kind}:${scope.userId}:${scope.organizationId}`;
const pendingPlanSchema=z.object({key:z.string().refine(isAcquisitionUUID),input:createPlanSchema}).strict();
type PendingPlan=z.infer<typeof pendingPlanSchema>;
function loadPending(scope:AcquisitionContext):PendingPlan|null{
 try{const raw=sessionStorage.getItem(storage(scope,"create"));const parsed=pendingPlanSchema.safeParse(raw?JSON.parse(raw):null);return parsed.success?parsed.data:null;}catch{return null;}
}
const planStateLabels: Record<AcquisitionPlan["state"], string> = {
 pending: "待继续", paused: "已暂停", transfer_pending: "待转入", completed: "已完成",
};
const storageFailure="无法保存原请求，未提交操作。请恢复浏览器存储后重试；已有原请求请先核实。";
function persist(scope:AcquisitionContext,kind:string,value:unknown):boolean{
 try{
  const slot=storage(scope,kind),before=sessionStorage.getItem(slot);
  if(value===null){sessionStorage.removeItem(slot);return sessionStorage.getItem(slot)===null;}
  const raw=JSON.stringify(value);
  // Preserve an unread or different original instead of replacing its key.
  if(before!==null&&before!==raw)return false;
  sessionStorage.setItem(slot,raw);return sessionStorage.getItem(slot)===raw;
 }catch{return false;}
}

export function AcquisitionPlansPage({scope,agentKind}:{scope:AcquisitionContext;agentKind?:string|null}){
 const [cursor,setCursor]=useState<string>();
 const [plans,setPlans]=useState<AcquisitionPlan[]>([]),[rules,setRules]=useState<RuleTemplate[]>([]);
 const [open,setOpen]=useState(false),[ruleOpen,setRuleOpen]=useState(false),[loading,setLoading]=useState(true),[canTransfer,setCanTransfer]=useState(false);
 const [error,setError]=useState<string|null>(null),[busy,setBusy]=useState(false),[pending,setPending]=useState<PendingPlan|null>(()=>loadPending(scope)),[activePlan,setActivePlan]=useState<AcquisitionPlan|null>(null);
 const planHeading = useRef<HTMLHeadingElement>(null);
 const resultHeading = useRef<HTMLHeadingElement>(null);
 const activePlanId = activePlan?.id;
 useEffect(() => {
  if (activePlanId) resultHeading.current?.focus();
 }, [activePlanId]);

 function closeResults() {
  if (!activePlanId) return;
  setActivePlan(null);
  const trigger = document.getElementById(`plan-results-${activePlanId}`);
  (trigger instanceof HTMLButtonElement && !trigger.disabled ? trigger : planHeading.current)?.focus();
 }

 const alive=useRef(true),controller=useRef<AbortController|null>(null),running=useRef(false);
 useEffect(()=>{
  alive.current=true;const abort=new AbortController();controller.current=abort;
  void Promise.all([api.listAcquisitionPlans(scope,abort.signal),api.listAcquisitionRules(scope,abort.signal)]).then(([list,templates])=>{
   if(!alive.current||abort.signal.aborted)return;setPlans(list.plans);setCursor(list.nextCursor);setRules(templates.rules);setCanTransfer(list.canAutoTransfer);setLoading(false);
  }).catch(e=>{if(alive.current&&!abort.signal.aborted){setError(failureText(e));setLoading(false);}});
  return ()=>{alive.current=false;running.current=false;controller.current?.abort();};
 },[scope]);
 const remember=(p:AcquisitionPlan,select=true)=>{
  if(!alive.current)return;
  setActivePlan(active=>select||active?.id===p.id?p:active);
  setPlans(values=>values.some(value=>value.id===p.id)?values.map(value=>value.id===p.id?p:value):[p,...values]);
 };
 async function advance(p:AcquisitionPlan,abort:AbortController){
  running.current=true;
  for(let step=0;step<422&&running.current&&!abort.signal.aborted&&alive.current;step++){
   if(p.state==="completed"||p.failureCode==="transfer_source_conflict")break;
   p=await api.advanceAcquisitionPlan(scope,p,abort.signal);
   if(abort.signal.aborted||!alive.current)break;remember(p,false);
   if(p.state==="paused"||p.failureCode)break;
  }
  running.current=false;
 }
 async function more(){if(busy||!cursor)return;const abort=new AbortController();controller.current=abort;setBusy(true);try{const page=await api.listAcquisitionPlans(scope,abort.signal,cursor);if(!alive.current||abort.signal.aborted)return;setPlans(values=>[...values,...page.plans.filter(p=>!values.some(v=>v.id===p.id))]);setCursor(page.nextCursor);}catch(e){if(alive.current&&!abort.signal.aborted)setError(failureText(e));}finally{if(alive.current)setBusy(false);}}
 async function submit(input:CreatePlan){
  if(busy||pending)return;const abort=new AbortController();controller.current=abort;
  const intent={key:crypto.randomUUID(),input};if(!persist(scope,"create",intent)){setError(storageFailure);return;}setPending(intent);setBusy(true);setError(null);
  try{
   const plan=await api.createAcquisitionPlan(scope,intent.key,intent.input,abort.signal);
   if(!alive.current||abort.signal.aborted)return;
   remember(plan);setPending(null);persist(scope,"create",null);setOpen(false);await advance(plan,abort);
  }catch(e){if(alive.current&&!abort.signal.aborted){setError(failureText(e));if(e instanceof AcquisitionAPIError&&["INVALID_ACQUISITION","SOURCE_TOO_LARGE"].includes(e.code)){setPending(null);persist(scope,"create",null);}}}
  finally{if(alive.current){setBusy(false);running.current=false;}}
 }
 async function recoverCreation(replay=false){
  if(!pending||busy)return;if(replay&&!persist(scope,"create",pending)){setError(storageFailure);return;}const abort=new AbortController();controller.current=abort;setBusy(true);setError(null);
  try{
   const plan=replay?await api.createAcquisitionPlan(scope,pending.key,pending.input,abort.signal):await api.readAcquisitionPlanByKey(scope,pending.key,abort.signal);
   if(!alive.current||abort.signal.aborted)return;remember(plan);setPending(null);persist(scope,"create",null);setOpen(false);
  }catch(e){if(alive.current&&!abort.signal.aborted)setError(failureText(e));}
  finally{if(alive.current)setBusy(false);}
 }
 async function inspect(id:string,resume=false){
  if(busy)return;
  if(id===activePlanId)resultHeading.current?.focus();
  const abort=new AbortController();controller.current=abort;setBusy(true);setError(null);
  try{const p=await api.readAcquisitionPlan(scope,id,abort.signal);if(!alive.current||abort.signal.aborted)return;remember(p);if(resume)await advance(p,abort);}
  catch(e){if(alive.current&&!abort.signal.aborted)setError(failureText(e));}
  finally{if(alive.current){setBusy(false);running.current=false;}}
 }
 function pause(){running.current=false;controller.current?.abort();setBusy(false);setError("已停止发出新步骤；已发出的操作请读取原计划确认。");}
 return <ConsolePage title="1688在线采集" breadcrumbs={findConsoleRoute("/workbench/supply/acquisition")?.trail} description="采集成功的商品保存到我的数据，按需转入待上传。" actions={<Button disabled={loading||busy||!!pending} onClick={()=>{setError(null);setOpen(true);}}>1688在线采集</Button>}>
  {error&&!open&&!ruleOpen?<p role="alert" className={styles.feedback}>{error}</p>:null}
  {pending?<Card className="mb-4 space-y-3 p-5"><h2>创建结果待核实</h2><p className="text-sm text-muted-foreground">原请求标识：{pending.key}。读取不会开始采集。</p><div className="flex flex-wrap gap-2"><Button variant="outline" disabled={busy} onClick={()=>void recoverCreation()}>读取原计划</Button><Button variant="outline" disabled={busy} onClick={()=>void recoverCreation(true)}>使用原请求确认创建</Button></div></Card>:null}
  <Card className="space-y-4 p-5"><div className="flex items-center justify-between"><h2 ref={planHeading} tabIndex={-1} className="font-semibold">采集计划</h2>{busy?<Button variant="outline" onClick={pause}>暂停</Button>:null}</div>
   <p className="text-sm text-muted-foreground">每计划最多 20 行输入、200 个去重商品；只读当前页可见商品。离开页面会暂停发出新请求。</p>
   {loading ? <p role="status">正在读取采集计划…</p> : plans.length === 0 ? <p>还没有采集计划。</p> : (
    <div className={styles.planList}>{plans.map(plan => {
     const input = plan.input.inputs[0] ?? "";
     const source = canonical1688Source(input);
     const summary = plan.input.mode === "product" && source ? `商品 ${offerId(source)}` : input;
     const selected = plan.id === activePlanId;
     return <div key={plan.id}>
      <div className="min-w-0 flex-1 basis-52">
       <p className="font-medium">{modes.find(mode => mode.id === plan.input.mode)?.label} · {new Date(plan.createdAt).toLocaleString("zh-CN")}</p>
       <p className="truncate text-xs text-muted-foreground" title={input}>
        {summary}{plan.input.inputs.length > 1 ? ` 等 ${plan.input.inputs.length} 个输入` : ""}
       </p>
      </div>
      <span>{plan.items.filter(item => item.state === "published").length}/{plan.items.length} 已保存 · {planStateLabels[plan.state]}</span>
      <Button id={`plan-results-${plan.id}`} variant="outline" size="sm" disabled={busy} aria-expanded={selected}
       aria-controls={selected ? "acquisition-plan-result-panel" : undefined} onClick={() => void inspect(plan.id)}>查看结果</Button>
      {plan.state !== "completed" ? <Button size="sm" disabled={busy || plan.failureCode === "transfer_source_conflict"}
       onClick={() => void inspect(plan.id, true)}>继续原计划</Button> : null}
     </div>;
    })}</div>
   )}
   {cursor?<Button variant="outline" disabled={busy} onClick={()=>void more()}>更多计划</Button>:null}
  </Card>
  {activePlan ? <PlanResults key={activePlan.id} plan={activePlan} agentKind={agentKind} heading={resultHeading} onClose={closeResults} /> : null}
  {open?<CaptureDialog closeLocked={busy} rules={rules} canTransfer={canTransfer} disabled={busy||loading||!!pending} error={error} onClose={()=>{if(!busy)setOpen(false);}} onRules={()=>setRuleOpen(true)} onSubmit={input=>void submit(input)}/>:null}
  {ruleOpen?<RuleDialog scope={scope} rules={rules} onRules={setRules} onClose={()=>setRuleOpen(false)}/>:null}
 </ConsolePage>;
}

function offerId(source: string) {
 return source.slice(source.lastIndexOf("/") + 1).replace(/\.html$/, "");
}

type PlanItem = AcquisitionPlan["items"][number];
const resultStates = {
 published: { label: "已保存", variant: "success" },
 failed: { label: "采集失败", variant: "danger" },
 needs_verification: { label: "待核实", variant: "warning" },
 pending: { label: "待采集", variant: "neutral" },
} as const;
const planItemStates = Object.keys(resultStates) as PlanItem["state"][];
const ruleReasons: Record<string,string> = {
 price_below_minimum: "公开起售价低于规则下限",
 price_above_maximum: "公开起售价高于规则上限",
 moq_above_maximum: "起订量超过规则上限",
 source_facts_invalid: "来源事实无效",
 source_price_unavailable: "未取得可用于判断的公开价格",
};
const ruleResults = {
 matched: "通过规则",
 not_matched: "未通过规则，保留在我的数据",
 unknown: "规则事实不足，保留在我的数据",
} as const;

function PlanResults({plan, agentKind, heading, onClose}: {
 plan: AcquisitionPlan;
 agentKind?: string | null;
 heading: Ref<HTMLHeadingElement>;
 onClose: () => void;
}) {
 const [filter, setFilter] = useState<PlanItem["state"] | "all">("all");
 const counts = { published: 0, failed: 0, needs_verification: 0, pending: 0 };
 for (const item of plan.items) counts[item.state]++;
 const items = filter === "all" ? plan.items : plan.items.filter(item => item.state === filter);

 return <Card id="acquisition-plan-result-panel" role="region" aria-labelledby="acquisition-plan-results" className="mt-4 space-y-4 p-5">
  <div className="flex flex-wrap items-start justify-between gap-3">
   <div>
    <h2 ref={heading} tabIndex={-1} id="acquisition-plan-results" className="text-lg font-semibold">采集结果</h2>
    <p className="mt-1 text-sm text-muted-foreground">已读取 {plan.discoveryIndex} / {plan.input.inputs.length} 个输入 · {new Date(plan.createdAt).toLocaleString("zh-CN")}</p>
   </div>
   <Button variant="outline" onClick={onClose}>返回采集计划</Button>
  </div>
  <div className="flex flex-wrap gap-2">
   {planItemStates.map(state => <Badge key={state} variant={resultStates[state].variant}>{resultStates[state].label} {counts[state]}</Badge>)}
  </div>
  <details className="text-xs text-muted-foreground">
   <summary className="w-fit cursor-pointer">计划信息</summary>
   <p className="mt-1 break-all">计划 ID {plan.id}</p>
  </details>
  {plan.truncated ? <p role="status" className="text-sm text-warning-foreground">结果达到读取或商品数量上限，已截断；本计划不代表整页全部商品。</p> : null}
  {plan.failureCode ? <p role="status" className="text-sm text-warning-foreground">{messages[plan.failureCode] ?? plan.failureCode}</p> : null}
  {plan.discoveryFailures.length ? <ul className="list-disc space-y-1 pl-5 text-sm text-warning-foreground">
   {plan.discoveryFailures.map(failure => <li key={failure.inputIndex}>第 {failure.inputIndex + 1} 行：公开来源不可用，未开始商品采集。</li>)}
  </ul> : null}
  {plan.preparationIds.length ? <div className="flex flex-wrap gap-2">
   {plan.preparationIds.map((id,index) => <Button key={id} asChild variant="outline"><Link href={`/workbench/listing/waiting?preparation=${id}`}>查看待上传 {index + 1}</Link></Button>)}
  </div> : null}
  {plan.items.length ? <label className="block max-w-56 space-y-1 text-sm">
   <span>按保存状态筛选</span>
   <Select value={filter} onChange={event => setFilter(event.target.value as typeof filter)}>
    <option value="all">全部商品（{plan.items.length}）</option>
    {planItemStates.map(state => <option key={state} value={state}>{resultStates[state].label}（{counts[state]}）</option>)}
   </Select>
  </label> : null}
  {items.length ? <Table aria-label="1688 商品结果">
   <TableHeader className="bg-muted"><TableRow>
    <TableHead scope="col">1688 商品</TableHead><TableHead scope="col">保存状态</TableHead>
    <TableHead scope="col">公开起售价 / 起订量</TableHead><TableHead scope="col">规则结果</TableHead><TableHead scope="col">操作</TableHead>
   </TableRow></TableHeader>
   <TableBody>{items.map(item => <TableRow key={item.source}>
    <TableCell><a href={item.source} target="_blank" rel="noreferrer" className="font-medium text-primary underline-offset-4 hover:underline">商品 {offerId(item.source)}</a></TableCell>
    <TableCell>
     <Badge variant={resultStates[item.state].variant}>{resultStates[item.state].label}</Badge>
     {item.failureCode ? <p className="mt-1 min-w-32 text-xs text-muted-foreground">{messages[item.failureCode] ?? item.failureCode}</p> : null}
    </TableCell>
    <TableCell>
     <p>{item.evaluation?.startingPrice ? `¥${item.evaluation.startingPrice}` : "起售价未记录"}</p>
     <p className="text-xs text-muted-foreground">{item.evaluation?.minimumQuantity ? `起订量 ${item.evaluation.minimumQuantity}` : "起订量未记录"}</p>
    </TableCell>
    <TableCell className="min-w-32 text-sm">
     {item.evaluation ? ruleResults[item.evaluation.state] : "尚无规则结果"}
     {item.evaluation?.reason ? <p className="mt-1 text-xs text-muted-foreground">{ruleReasons[item.evaluation.reason] ?? messages[item.evaluation.reason] ?? item.evaluation.reason}</p> : null}
    </TableCell>
    <TableCell>{item.state === "published" && item.operationId ?
     <Button asChild variant="outline" size="sm"><Link href={productAgentOperationHref(item.operationId, agentKind)} prefetch={false}>查看商品</Link></Button> : null}</TableCell>
   </TableRow>)}</TableBody>
  </Table> : <p className="text-sm text-muted-foreground">{plan.items.length ? "当前筛选没有商品。" : "此计划暂无商品结果。"}</p>}
 </Card>;
}

function NativeDialog({title,className,onClose,children}:{title:string;className:string;onClose:()=>void;children:ReactNode}){
 const ref=useRef<HTMLDialogElement>(null);
 useEffect(()=>{const dialog=ref.current;dialog?.showModal();return()=>dialog?.close();},[]);
 return <dialog ref={ref} aria-label={title} className={`${styles.dialog} ${className}`} onCancel={e=>{e.preventDefault();onClose();}}>{children}</dialog>;
}

function CaptureDialog({closeLocked,rules,canTransfer,disabled,error,onClose,onRules,onSubmit}:{closeLocked:boolean;rules:RuleTemplate[];canTransfer:boolean;disabled:boolean;error:string|null;onClose:()=>void;onRules:()=>void;onSubmit:(v:CreatePlan)=>void}){
 const [mode,setMode]=useState<CreatePlan["mode"]>("product"),[values,setValues]=useState<Record<string,string>>({product:"",page:"",keyword:""}),[auto,setAuto]=useState(false),[ruleID,setRuleID]=useState(""),[inputError,setInputError]=useState<string|null>(null);
 const selected=modes.find(m=>m.id===mode)!;
 function start(){const template=rules.find(r=>r.id===ruleID);const parsed=createPlanSchema.safeParse({mode,inputs:values[mode]!.split(/\r?\n/).map(s=>s.trim()).filter(Boolean),autoTransfer:auto, ...(template?{templateId:template.id,templateRevision:template.revision}:{})});if(!parsed.success){setInputError("请填写 1–20 行有效输入；关键词最多 200 字节，商品需提供公开链接或 offer ID。");return;}setInputError(null);onSubmit(parsed.data);}
 return <NativeDialog title="1688在线采集" className={styles.captureDialog} onClose={onClose}><form className={`${styles.capture} ${error||inputError?styles.withError:""}`} onSubmit={e=>{e.preventDefault();start();}}>
  <header><h2>1688在线采集</h2><span className={styles.billing}>1688 · 按数据量计费</span><p>支持指定商品、按页、按关键词三种采集方式。</p></header>
  <section className={styles.config}><span>采集方式</span><div role="tablist" aria-label="采集方式">{modes.map(m=><button key={m.id} type="button" role="tab" aria-selected={mode===m.id} disabled={disabled} onClick={()=>setMode(m.id)}>{m.label}</button>)}</div></section>
  <div className={styles.source}><label htmlFor="capture-input">{selected.field}</label><Textarea id="capture-input" value={values[mode]} onChange={e=>setValues(v=>({...v,[mode]:e.target.value}))} disabled={disabled} placeholder={selected.placeholder}/><p>最多 20 行输入、200 个去重商品；{mode==="product"?"每行一条。":"仅当前页可见商品，不自动翻页。"}</p></div>
  <p className={styles.charge}>按实际采集成功的数据条数扣除数据额度；采集规则不影响计费。</p>
  <section className={styles.execution}><div><h3>采集后处理</h3><p>采集结果全部保存至「数据服务 → 我的数据」</p><div className={styles.transfer}><span>采集后自动转入待上传</span><button type="button" role="switch" aria-label="采集后自动转入待上传" aria-checked={auto} disabled={disabled||!canTransfer} className={styles.switch} onClick={()=>setAuto(!auto)}><Image src="/figma/acquisition/switch-off.svg" alt="" width={18} height={18}/></button></div><p>{canTransfer?"开启后进入待上传，补全平台必填信息并准备上架；关闭后仅保留在「我的数据」。":"当前身份缺少自动转入权限，采集结果仍保存到「我的数据」。"}</p></div></section>
  <div className={styles.ruleChooser}><label htmlFor="capture-rule">采集规则</label><Select id="capture-rule" value={ruleID} disabled={disabled} onChange={e=>setRuleID(e.target.value)}><option value="">选择采集规则（可选）</option>{rules.map(r=><option value={r.id} key={r.id}>{r.name}</option>)}</Select><Button type="button" variant="outline" disabled={disabled} onClick={onRules}>设置规则</Button></div>
  {error||inputError?<p role="alert" className={styles.dialogError}>{inputError??error}</p>:null}
  <footer><Button type="button" variant="outline" disabled={closeLocked} onClick={onClose}>取消</Button><Button type="submit" disabled={disabled}>开始采集</Button></footer>
 </form></NativeDialog>;
}

function RuleDialog({scope,rules,onRules,onClose}:{scope:AcquisitionContext;rules:RuleTemplate[];onRules:(v:RuleTemplate[])=>void;onClose:()=>void}){
 const [id,setID]=useState(rules[0]?.id??""),[name,setName]=useState(rules[0]?.name??""),[min,setMin]=useState(rules[0]?.rule.minPrice??""),[max,setMax]=useState(rules[0]?.rule.maxPrice??""),[moq,setMOQ]=useState(String(rules[0]?.rule.maxMOQ||""));
 const [pending,setPending]=useState<{key:string;input:RuleMutation}|null>(()=>{
  try{const raw=sessionStorage.getItem(storage(scope,"rule"));const p=z.object({key:z.string().refine(isAcquisitionUUID),input:ruleMutationSchema}).strict().safeParse(raw?JSON.parse(raw):null);return p.success?p.data:null;}catch{return null;}
 });
 const [busy,setBusy]=useState(false),[error,setError]=useState<string|null>(()=>pending?"模板操作结果待确认，请使用原请求核实。":null);const abort=useRef<AbortController|null>(null),alive=useRef(true);
 useEffect(()=>{alive.current=true;return()=>{alive.current=false;abort.current?.abort();};},[]);
 function select(rule?:RuleTemplate){if(busy||pending)return;setID(rule?.id??"");setName(rule?.name??"");setMin(rule?.rule.minPrice??"");setMax(rule?.rule.maxPrice??"");setMOQ(String(rule?.rule.maxMOQ||""));setError(null);}
 async function save(action:"save"|"rename"|"archive",replay=false){
  if(busy||pending&&!replay)return;
  const existing=rules.find(r=>r.id===id);
  const parsed=acquisitionRuleSchema.safeParse({minPrice:min,maxPrice:max,maxMOQ:moq===""?0:Number(moq)});
  if(!replay&&action!=="archive"&&(!parsed.success||moq!==""&&!/^[1-9][0-9]*$/.test(moq))){setError("采购价应为非负十进制，最低价不大于最高价；起批量应为正整数。");return;}
  const input:RuleMutation=action==="archive"?{action:"archive",id,expectedRevision:existing?.revision,rule:{}}:existing?{action:"update",id,expectedRevision:existing.revision,name,rule:action==="rename"?existing.rule:parsed.success?parsed.data:{}}:{action:"create",name,rule:parsed.success?parsed.data:{}};
  const validated=ruleMutationSchema.safeParse(input);if(!replay&&!validated.success){setError("请填写模板名称（最多100字）。");return;}
  const intent=replay?pending!:{key:crypto.randomUUID(),input:validated.success?validated.data:input};if(!persist(scope,"rule",intent)){setError(storageFailure);return;}setPending(intent);setBusy(true);setError(null);const controller=new AbortController();abort.current=controller;
  try{const saved=await api.mutateAcquisitionRule(scope,intent.key,intent.input,controller.signal);if(!alive.current||controller.signal.aborted)return;
   const next=[...rules.filter(r=>r.id!==saved.id),...(saved.archived?[]:[saved])];onRules(next);setPending(null);persist(scope,"rule",null);if(saved.archived){setID(next[0]?.id??"");setName(next[0]?.name??"");setMin(next[0]?.rule.minPrice??"");setMax(next[0]?.rule.maxPrice??"");setMOQ(String(next[0]?.rule.maxMOQ||""));}else{setID(saved.id);setName(saved.name);setMin(saved.rule.minPrice??"");setMax(saved.rule.maxPrice??"");setMOQ(String(saved.rule.maxMOQ||""));}
  }catch(e){if(alive.current&&!controller.signal.aborted){setError(failureText(e));if(e instanceof AcquisitionAPIError&&["INVALID_ACQUISITION","IDEMPOTENCY_CONFLICT","ACQUISITION_NOT_FOUND"].includes(e.code)){setPending(null);persist(scope,"rule",null);}}}
  finally{if(alive.current)setBusy(false);}
 }
 const locked=busy||!!pending;
 return <NativeDialog title="1688采集规则" className={styles.rulesDialog} onClose={()=>{if(!busy)onClose();}}><div className={styles.rules}><header><h2>1688采集规则</h2><p>管理并复用1688采集筛选模板。</p></header><div className={styles.ruleGrid}>
  <aside><h3>规则模板</h3><div>{rules.map(r=><button key={r.id} disabled={locked} aria-pressed={r.id===id} onClick={()=>select(r)}><strong>{r.name}</strong><span>{r.rule.minPrice||r.rule.maxPrice?`价${r.rule.minPrice||"不限"}–${r.rule.maxPrice||"不限"}`:"采购价不限"}{r.rule.maxMOQ?` · 起批≤${r.rule.maxMOQ}`:""}</span></button>)}</div><Button variant="outline" disabled={locked||rules.length>=100} onClick={()=>select()}>＋ 新建模板</Button></aside>
  <main><div className={styles.templateName}><label htmlFor="template-name">模板名称</label><Input id="template-name" value={name} maxLength={100} disabled={locked} onChange={e=>setName(e.target.value)}/><Button variant="outline" disabled={locked||!id} onClick={()=>void save("rename")}>保存名称</Button><Button variant="outline" disabled={locked||!id} className={styles.deleteRule} onClick={()=>void save("archive")}>删除模板</Button></div>
   <div className={styles.ruleNotice}><span>规则说明</span><p>1688在线采集会先采集并正常计费，再按当前规则筛选进入后续处理。<br/>不符合规则的商品仍保留在「我的数据」。</p></div>
   <div className={styles.conditionTitle}><h3>筛选条件</h3><span>同时满足</span></div><div className={styles.priceCondition}><span>采购价</span><Input aria-label="最低采购价" inputMode="decimal" placeholder="最低价" value={min} disabled={locked} onChange={e=>setMin(e.target.value)}/><span>—</span><Input aria-label="最高采购价" inputMode="decimal" placeholder="最高价" value={max} disabled={locked} onChange={e=>setMax(e.target.value)}/></div><div className={styles.moqCondition}><span>起批量</span><Input aria-label="最大起批量" inputMode="numeric" placeholder="不超过" value={moq} disabled={locked} onChange={e=>setMOQ(e.target.value)}/></div><p className={styles.ruleHint}>未填写的条件不参与筛选。</p>
   {error?<p role="alert" className={styles.feedback}>{error}</p>:null}{pending?<Button variant="outline" disabled={busy} onClick={()=>void save("save",true)}>使用原请求核实模板</Button>:null}
  </main></div><footer><Button variant="outline" disabled={busy} onClick={onClose}>取消</Button><Button variant="outline" disabled={locked} onClick={()=>void save("save")}>保存模板</Button></footer></div></NativeDialog>;
}
