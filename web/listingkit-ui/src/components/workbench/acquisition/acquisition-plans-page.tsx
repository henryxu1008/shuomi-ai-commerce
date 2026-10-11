"use client";

import Link from "next/link";
import Image from "next/image";
import {useEffect,useRef,useState,type ReactNode} from "react";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {Textarea} from "@/components/ui/textarea";
import {ConsolePage} from "../console/console-page";
import {findConsoleRoute} from "@/lib/workbench/console-navigation";
import {AcquisitionAPIError,type AcquisitionContext} from "@/lib/api/product-acquisition";
import * as api from "@/lib/api/acquisition-plan";
import {createPlanSchema,ruleMutationSchema,acquisitionRuleSchema,type AcquisitionPlan,type CreatePlan,type RuleTemplate,type RuleMutation} from "@/lib/contracts/acquisition-plan";
import {isAcquisitionUUID} from "@/lib/contracts/product-acquisition";
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
function persist(scope:AcquisitionContext,kind:string,value:unknown){try{if(value===null)sessionStorage.removeItem(storage(scope,kind));else sessionStorage.setItem(storage(scope,kind),JSON.stringify(value));}catch{/* The server remains the durable recovery owner. */}}

export function AcquisitionPlansPage({scope,initialOpen=false}:{scope:AcquisitionContext;initialOpen?:boolean}){
 const [cursor,setCursor]=useState<string>();
 const [plans,setPlans]=useState<AcquisitionPlan[]>([]),[rules,setRules]=useState<RuleTemplate[]>([]);
 const [open,setOpen]=useState(initialOpen),[ruleOpen,setRuleOpen]=useState(false),[loading,setLoading]=useState(true),[canTransfer,setCanTransfer]=useState(false);
 const [error,setError]=useState<string|null>(null),[busy,setBusy]=useState(false),[pending,setPending]=useState<PendingPlan|null>(()=>loadPending(scope)),[activePlan,setActivePlan]=useState<AcquisitionPlan|null>(null),[operationID,setOperationID]=useState("");
 const alive=useRef(true),controller=useRef<AbortController|null>(null),running=useRef(false);
 useEffect(()=>{
  alive.current=true;const abort=new AbortController();controller.current=abort;
  void Promise.all([api.listAcquisitionPlans(scope,abort.signal),api.listAcquisitionRules(scope,abort.signal)]).then(([list,templates])=>{
   if(!alive.current||abort.signal.aborted)return;setPlans(list.plans);setCursor(list.nextCursor);setRules(templates.rules);setCanTransfer(list.canAutoTransfer);setLoading(false);
  }).catch(e=>{if(alive.current&&!abort.signal.aborted){setError(failureText(e));setLoading(false);}});
  return ()=>{alive.current=false;running.current=false;controller.current?.abort();};
 },[scope]);
 const remember=(p:AcquisitionPlan)=>{if(!alive.current)return;setActivePlan(p);setPlans(values=>[p,...values.filter(v=>v.id!==p.id)]);};
 async function advance(p:AcquisitionPlan,abort:AbortController){
  running.current=true;
  for(let step=0;step<422&&running.current&&!abort.signal.aborted&&alive.current;step++){
   if(p.state==="completed"||p.failureCode==="transfer_source_conflict")break;
   p=await api.advanceAcquisitionPlan(scope,p,abort.signal);
   if(abort.signal.aborted||!alive.current)break;remember(p);
   if(p.state==="paused"||p.failureCode)break;
  }
  running.current=false;
 }
 async function more(){if(busy||!cursor)return;const abort=new AbortController();controller.current=abort;setBusy(true);try{const page=await api.listAcquisitionPlans(scope,abort.signal,cursor);if(!alive.current||abort.signal.aborted)return;setPlans(values=>[...values,...page.plans.filter(p=>!values.some(v=>v.id===p.id))]);setCursor(page.nextCursor);}catch(e){if(alive.current&&!abort.signal.aborted)setError(failureText(e));}finally{if(alive.current)setBusy(false);}}
 async function submit(input:CreatePlan){
  if(busy||pending)return;const abort=new AbortController();controller.current=abort;
  const intent={key:crypto.randomUUID(),input};setPending(intent);persist(scope,"create",intent);setBusy(true);setError(null);
  try{
   const plan=await api.createAcquisitionPlan(scope,intent.key,intent.input,abort.signal);
   if(!alive.current||abort.signal.aborted)return;
   remember(plan);setPending(null);persist(scope,"create",null);setOpen(false);await advance(plan,abort);
  }catch(e){if(alive.current&&!abort.signal.aborted){setError(failureText(e));if(e instanceof AcquisitionAPIError&&["INVALID_ACQUISITION","SOURCE_TOO_LARGE"].includes(e.code)){setPending(null);persist(scope,"create",null);}}}
  finally{if(alive.current){setBusy(false);running.current=false;}}
 }
 async function recoverCreation(replay=false){
  if(!pending||busy)return;const abort=new AbortController();controller.current=abort;setBusy(true);setError(null);
  try{
   const plan=replay?await api.createAcquisitionPlan(scope,pending.key,pending.input,abort.signal):await api.readAcquisitionPlanByKey(scope,pending.key,abort.signal);
   if(!alive.current||abort.signal.aborted)return;remember(plan);setPending(null);persist(scope,"create",null);setOpen(false);
  }catch(e){if(alive.current&&!abort.signal.aborted)setError(failureText(e));}
  finally{if(alive.current)setBusy(false);}
 }
 async function inspect(id:string,resume=false){
  if(busy)return;const abort=new AbortController();controller.current=abort;setBusy(true);setError(null);
  try{const p=await api.readAcquisitionPlan(scope,id,abort.signal);if(!alive.current||abort.signal.aborted)return;remember(p);if(resume)await advance(p,abort);}
  catch(e){if(alive.current&&!abort.signal.aborted)setError(failureText(e));}
  finally{if(alive.current){setBusy(false);running.current=false;}}
 }
 function pause(){running.current=false;controller.current?.abort();setBusy(false);setError("已停止发出新步骤；已发出的操作请读取原计划确认。");}
 return <ConsolePage title="1688在线采集" breadcrumbs={findConsoleRoute("/workbench/supply/acquisition")?.trail} description="采集成功的商品保存到我的数据，按需转入待适配。" actions={<Button disabled={loading||busy||!!pending} onClick={()=>{setError(null);setOpen(true);}}>1688在线采集</Button>}>
  {error&&!open&&!ruleOpen?<p role="alert" className={styles.feedback}>{error}</p>:null}
  {pending?<Card className="mb-4 space-y-3 p-5"><h2>创建结果待核实</h2><p className="text-sm text-muted-foreground">原请求标识：{pending.key}。读取不会开始采集。</p><div className="flex flex-wrap gap-2"><Button variant="outline" disabled={busy} onClick={()=>void recoverCreation()}>读取原计划</Button><Button variant="outline" disabled={busy} onClick={()=>void recoverCreation(true)}>使用原请求确认创建</Button></div></Card>:null}
  <Card className="space-y-4 p-5"><div className="flex items-center justify-between"><h2 className="font-semibold">采集计划</h2>{busy?<Button variant="outline" onClick={pause}>暂停</Button>:null}</div>
   <p className="text-sm text-muted-foreground">每计划最多 20 行输入、200 个去重商品；只读当前页可见商品。离开页面会暂停发出新请求。</p>
   {loading?<p role="status">正在读取采集计划…</p>:plans.length===0?<p>还没有采集计划。</p>:<div className={styles.planList}>{plans.map(p=><div key={p.id}><span>{p.input.mode==="product"?"指定商品":p.input.mode==="page"?"按页采集":"按关键词"} · {new Date(p.createdAt).toLocaleString("zh-CN")}</span><span>{p.items.filter(i=>i.state==="published").length}/{p.items.length} 已保存 · {p.state==="completed"?"已完成":p.state==="paused"?"已暂停":p.state==="transfer_pending"?"待转入":"待继续"}</span><Button variant="outline" size="sm" disabled={busy} onClick={()=>void inspect(p.id)}>查看</Button>{p.state!=="completed"?<Button size="sm" disabled={busy||p.failureCode==="transfer_source_conflict"} onClick={()=>void inspect(p.id,true)}>继续原计划</Button>:null}</div>)}</div>}
   {cursor?<Button variant="outline" disabled={busy} onClick={()=>void more()}>更多计划</Button>:null}
  </Card>
  {activePlan?<Card className="mt-4 space-y-3 p-5"><h2 className="font-semibold">当前计划</h2><p className="break-all text-xs text-muted-foreground">{activePlan.id}</p><p>已读取 {activePlan.discoveryIndex}/{activePlan.input.inputs.length} 个输入 · 已保存 {activePlan.items.filter(i=>i.state==="published").length} 个商品 · 失败 {activePlan.items.filter(i=>i.state==="failed").length} 个</p>
   {activePlan.truncated?<p role="status">结果达到读取或商品数量上限，已截断；本计划不代表整页全部商品。</p>:null}
   {activePlan.failureCode?<p role="status">{messages[activePlan.failureCode]??activePlan.failureCode}</p>:null}
   {activePlan.discoveryFailures.map(f=><p key={f.inputIndex}>第 {f.inputIndex+1} 行：公开来源不可用，未开始商品采集。</p>)}
   {activePlan.preparationIds.map((id,index)=><Button key={id} asChild variant="outline"><Link href={`/workbench/supply/mine/waiting?preparation=${id}`}>查看待适配 {index+1}</Link></Button>)}
   <div className={styles.itemList}>{activePlan.items.map(i=><div key={i.source}><span className="break-all">{i.source}</span><span>{i.state==="published"?"已保存":i.state==="failed"?"来源失败":i.state==="needs_verification"?"结果待核实":"待采集"}{i.evaluation?` · ${i.evaluation.state==="matched"?"规则通过":i.evaluation.state==="not_matched"?"未通过规则，保留在我的数据":"规则事实不足，保留在我的数据"}`:""}</span>{i.failureCode?<span>{messages[i.failureCode]??i.failureCode}</span>:null}{i.operationId&&i.state==="published"?<Link href={`/workbench/supply/acquisition/operation/${i.operationId}`}>查看商品</Link>:null}</div>)}</div>
  </Card>:null}
  <Card className="mt-4 space-y-3 p-5"><h2 className="font-semibold">找回单商品采集结果</h2><p className="text-sm text-muted-foreground">已有操作 ID 可直接读取原结果。</p><div className="flex max-w-xl gap-2"><Input aria-label="操作 ID" value={operationID} onChange={e=>setOperationID(e.target.value)}/><Button asChild variant="outline" disabled={!isAcquisitionUUID(operationID)}><Link aria-disabled={!isAcquisitionUUID(operationID)} href={isAcquisitionUUID(operationID)?`/workbench/supply/acquisition/operation/${operationID}`:"#"}>读取</Link></Button></div></Card>
  {open?<CaptureDialog closeLocked={busy} rules={rules} canTransfer={canTransfer} disabled={busy||loading||!!pending} error={error} onClose={()=>{if(!busy)setOpen(false);}} onRules={()=>setRuleOpen(true)} onSubmit={input=>void submit(input)}/>:null}
  {ruleOpen?<RuleDialog scope={scope} rules={rules} onRules={setRules} onClose={()=>setRuleOpen(false)}/>:null}
 </ConsolePage>;
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
  <section className={styles.execution}><div><h3>采集后处理</h3><p>采集结果全部保存至「数据服务 → 我的数据」</p><div className={styles.transfer}><span>采集后自动转入待适配</span><button type="button" role="switch" aria-label="采集后自动转入待适配" aria-checked={auto} disabled={disabled||!canTransfer} className={styles.switch} onClick={()=>setAuto(!auto)}><Image src="/figma/acquisition/switch-off.svg" alt="" width={18} height={18}/></button></div><p>{canTransfer?"开启后进入待适配，补全平台必填信息并准备上架；关闭后仅保留在「我的数据」。":"当前身份缺少自动转入权限，采集结果仍保存到「我的数据」。"}</p></div></section>
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
  const intent=replay?pending!:{key:crypto.randomUUID(),input:validated.success?validated.data:input};setPending(intent);persist(scope,"rule",intent);setBusy(true);setError(null);const controller=new AbortController();abort.current=controller;
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
