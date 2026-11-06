"use client";
import Link from "next/link";
import {useRouter} from "next/navigation";
import {useEffect,useRef,useState} from "react";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {researchRequest,ResearchError} from "@/lib/api/market-research";
import type {ProjectScope} from "@/lib/api/project-center";
import {researchPage,researchReceipt,researchIntent,selectionOf,type ResearchCommand,type ResearchCandidate} from "@/lib/contracts/market-research";
import {CollectionPicker} from "../pod/collection-picker";
import {useResourcePending} from "../resources/resource-pending";
import {ResearchAnalysisFlow,analysisChoices} from "./research-analysis-flow";
import {researchAnalysisInput} from "@/lib/contracts/research-analysis";
import {ResearchComparison} from "./research-comparison";
import {SheinUSTrialPanel} from "./shein-us-trial-panel";
import {ResearchSupplierReview} from "./research-supplier-review";
import {ResearchCostPreview} from "./research-cost-preview";
import {researchStep} from "@/lib/workbench/research-navigation";
import styles from "./research-panel.module.css";
import type {ResearchInspection} from "@/lib/contracts/research-analysis";
import {readCollectionBatch} from "@/lib/api/product-collection";
import {verifyResearchHandoff} from "@/lib/api/research-handoff";
import {advanceHandoffRevision,candidateMatchesJob,handoffToken,type ResearchHandoff} from "@/lib/workbench/research-handoff";
import {researchHandoffScope,useResearchHandoff} from "./research-handoff-provider";
import {ConfirmedOpportunity,ResearchOpportunity} from "./research-opportunity";
function ResearchTransferButton({disabled,prepare}:{disabled:boolean;prepare:()=>Promise<string|undefined>}) {
 const router=useRouter();
 return <Button disabled={disabled} onClick={async()=>{const href=await prepare();if(href)router.push(href);}}>核对 1 件商品</Button>;
}
const states:Record<string,string>={TO_RESEARCH:"待研究",PENDING_CONFIRMATION:"待人工确认",APPROVED_FOR_TEST:"批准测试",REJECTED:"已否决"};
const failures:Record<string,string>={REVISION_CONFLICT:"商品资料或选择已更新，请刷新后重新确认。",PROJECT_ARCHIVED:"项目已归档，只能查看。",NOT_FOUND:"记录不存在或当前不可访问。",FORBIDDEN:"权限已变化，请重新确认企业。",OUTCOME_UNKNOWN:"暂时还不能确认操作结果，请先查询这次操作。",SOURCE_NOT_SUPPORTED:"这份资料暂不能用于调研，本次未添加到项目。请选择已保存且支持调研的资料。"};
export function ResearchPanel({scope,projectId,archived,view="embedded",sample="",initialAfter="",returnQuery="",workflowStep="",showSheinTrial=false,researchToken,opportunityInspection}:{researchToken?:string;opportunityInspection?:ResearchInspection;scope:ProjectScope;projectId:string;archived:boolean;view?:"embedded"|"research"|"selection";sample?:string;initialAfter?:string;returnQuery?:string;workflowStep?:string;showSheinTrial?:boolean}){
 const context=useWorkbenchContext();
 const {store,draft}=useResearchHandoff();
 const scopeKey=researchHandoffScope(context);
 const special=researchToken!==undefined;
 const bound=draft?.scope===scopeKey && draft.projectId===projectId ? draft : null;
 const active=bound?.confirmed && handoffToken(bound)===researchToken ? bound : null;
 const canRead=context.permissions.includes("workbench.collection.read");
 const canWrite=!archived&&context.permissions.includes("workbench.project.manage")&&canRead&&
  (!special||!!active&&!active.transfer?.dispatched);
 const transferFlight=useRef<AbortController|null>(null);
 useEffect(()=>()=>transferFlight.current?.abort(),[]);
 useEffect(()=>{
  if (archived && store?.getSnapshot()?.projectId===projectId) store.clear();
 },[archived,store,projectId]);
 const pending=useResourcePending({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId},["research",projectId],researchIntent,{maxLength:65536});
 const [items,setItems]=useState<ResearchCandidate[]>([]),[after,setAfter]=useState(initialAfter),[next,setNext]=useState(""),[refresh,setRefresh]=useState(0),[loading,setLoading]=useState(true),[incomplete,setIncomplete]=useState(false),[error,setError]=useState(""),[busy,setBusy]=useState(false),[picker,setPicker]=useState(false),[unsupported,setUnsupported]=useState<string[]>([]),[chosen,setChosen]=useState<string[]>([]),[compare,setCompare]=useState(false),[absent,setAbsent]=useState(false);
  const [openedSupplier,setOpenedSupplier]=useState("");
 const [openedCost,setOpenedCost]=useState<string|null>(null),[loadedCostPage,setLoadedCostPage]=useState("");
 const costPageKey=JSON.stringify([scope.userId,scope.organizationId,projectId,after,refresh,canRead,canWrite,archived]);
 const lifetime=useRef<AbortController|null>(null),flight=useRef(false),handoff=useRef(false);
 const section=view==="embedded"?undefined:researchStep(view,workflowStep)?.section;
 useEffect(()=>{if(loading||!section)return;const target=document.getElementById(section);target?.focus({preventScroll:true});target?.scrollIntoView?.({block:"start"});},[loading,section]);
 useEffect(()=>{const controller=new AbortController();lifetime.current=controller;return()=>controller.abort();},[]);
 const register=context.registerOrganizationSwitchGuard;
 useEffect(()=>register(()=>!flight.current&&!pending.command&&!pending.error),[register,pending.command,pending.error]);
 useEffect(()=>{const controller=new AbortController();void Promise.resolve().then(async()=>{if(controller.signal.aborted)return;setLoading(true);setItems([]);setCompare(false);setError("");if(!canRead){setLoading(false);return;}try{const page=await researchRequest(scope,projectId,researchPage,controller.signal,{after});if(!controller.signal.aborted){setItems(page.items);setNext(page.nextCursor??"");setIncomplete(page.incomplete);setLoadedCostPage(costPageKey);
 if(!handoff.current&&sample){handoff.current=true;try{if(sample.length>16384)throw Error();const input=researchAnalysisInput.parse({candidates:JSON.parse(sample)}),current=analysisChoices(page.items.filter(v=>v.observation&&!("shein" in v.observation)));const matching=input.candidates.filter(choice=>current.some(c=>JSON.stringify(c)===JSON.stringify(choice)));if(matching.length!==input.candidates.length){setChosen([]);setError("所选商品资料已更新或暂时无法查看，请重新选择。");}else setChosen(matching.map(c=>c.candidateId));}catch{setChosen([]);setError("之前选择的商品暂时无法确认，请重新选择。");}}
 else setChosen(old=>old.filter(id=>page.items.some(c=>c.id===id&&c.observation)));
}}catch(e){if(!controller.signal.aborted)setError(message(e));}finally{if(!controller.signal.aborted)setLoading(false);}});return()=>controller.abort();},[scope,projectId,after,refresh,canRead,sample,costPageKey]);
 const reload=()=>setRefresh(v=>v+1);
 const locked=busy||loading||!!pending.command||!pending.ready||pending.error||!canWrite;
 const canPreview=!locked&&!error&&!incomplete&&loadedCostPage===costPageKey;
 function invalidateChoice() {
  const current=store?.getSnapshot();
  if (view==="research" && current && current.projectId===projectId && current.scope===scopeKey)
   store?.start(current.scope,current.projectId,current.job);
 }
 function advanceRevision(snapshot:ResearchHandoff|null,candidate:string,before:number,revision?:number,approved=false) {
  if (!snapshot || !store || !revision) return;
  const next=advanceHandoffRevision(snapshot,candidate,before,revision,approved);
  if (next!==snapshot) store.replace(snapshot,next);
 }
 async function startTransfer() {
  if (!active || !store || !active.job.batchId || active.transfer?.dispatched || transferFlight.current || !canWrite ||
      !context.permissions.includes("workbench.supply.manage")) return;
  const controller=new AbortController();
  transferFlight.current=controller;
  setBusy(true);setError("");
  try {
   const [,batch]=await Promise.all([
    verifyResearchHandoff(scope,active,controller.signal,true),
    readCollectionBatch(scope,active.job.batchId,AbortSignal.any([controller.signal,AbortSignal.timeout(10_000)])),
   ]);
   if (controller.signal.aborted || store.getSnapshot()!==active) return;
   if (batch.id!==active.job.batchId || batch.archivedAt || !batch.supplyTransferSupported ||
       active.transfer && active.transfer.revision!==batch.revision) throw new Error("Batch changed");
   const next=active.transfer?active:{...active,transfer:{key:crypto.randomUUID(),revision:batch.revision}};
   if (store.replace(active,next)) return "/workbench/data/mine/research-transfer?handoff="+handoffToken(next);
  } catch {
   if (!controller.signal.aborted && store.getSnapshot()===active) store.start(active.scope,active.projectId,active.job);
  } finally {
   if (transferFlight.current===controller) transferFlight.current=null;
   if (!controller.signal.aborted) setBusy(false);
  }
 }
 async function verifyCostEvidence(binding:string,signal:AbortSignal){
  try {
   const page=await researchRequest(scope,projectId,researchPage,signal,{after});
   return !signal.aborted&&!page.incomplete&&page.items.some(candidate=>costBinding(candidate)===binding);
  } catch (error) {
   if(error instanceof ResearchError){
    // The proxy can use the same error code for 403 and temporary 5xx failures.
    const {code,status}=error;
    const transient=code==="DEPENDENCY_UNAVAILABLE"||code==="DEADLINE_EXCEEDED"||
     (code==="INVALID_UPSTREAM_RESPONSE"&&(status===502||status===503||status===504));
    if((status!==undefined&&status<500)||!transient)return false;
   }
   throw error;
  }
 }
 async function submit(body:ResearchCommand,original=pending.command){
  const controller=lifetime.current;if(!controller||controller.signal.aborted||flight.current)return;
  const intent=original??{key:crypto.randomUUID(),body};
  if (special && (!active || !canWrite || !store ||
      !("candidateId" in intent.body) || active.confirmed?.representative!==intent.body.candidateId ||
      !["review","decide","reset"].includes(intent.body.action))) return;
  flight.current=true;setBusy(true);setError("");setAbsent(false);
  try{
   if (active) {
    try {await verifyResearchHandoff(scope,active,controller.signal);}
    catch {
     if (!controller.signal.aborted && store?.getSnapshot()===active) store.start(active.scope,active.projectId,active.job);
     return;
    }
    if (controller.signal.aborted || store?.getSnapshot()!==active) return;
   } else invalidateChoice();
   try{pending.persist(intent);}catch{setError("无法保存原操作，请恢复会话存储后继续。");return;}
   const receipt=await researchRequest(scope,projectId,researchReceipt,controller.signal,{intent});
   if(!controller.signal.aborted){
    if (active && "candidateId" in intent.body) advanceRevision(active,intent.body.candidateId,intent.body.expectedRevision,receipt.revisions[intent.body.candidateId],intent.body.action==="decide"&&intent.body.state==="APPROVED_FOR_TEST");
    pending.clear(intent);setPicker(false);reload();
   }
  }
  catch(e){if(controller.signal.aborted)return;setError(message(e));if(e instanceof ResearchError&&e.code==="SOURCE_NOT_SUPPORTED"){pending.clear(intent);const rejected=intent.body.action==="link"?intent.body.selections.map(s=>s.itemId):[];setUnsupported(old=>[...old,...rejected]);return;}if(e instanceof ResearchError&&["INVALID_REQUEST","REVISION_CONFLICT","PROJECT_ARCHIVED"].includes(e.code)){pending.clear(intent);reload();}}
  finally{flight.current=false;if(!controller.signal.aborted)setBusy(false);}
 }
 async function lookup(){const intent=pending.command,controller=lifetime.current;if(!intent||!controller||flight.current)return;flight.current=true;setBusy(true);setError("");try{const receipt=await researchRequest(scope,projectId,researchReceipt,controller.signal,{lookup:intent.key});if(!controller.signal.aborted){if(active&&"candidateId" in intent.body)advanceRevision(active,intent.body.candidateId,intent.body.expectedRevision,receipt.revisions[intent.body.candidateId],intent.body.action==="decide"&&intent.body.state==="APPROVED_FOR_TEST");pending.clear(intent);setAbsent(false);reload();}}catch(e){if(!controller.signal.aborted){setAbsent(e instanceof ResearchError&&e.code==="NOT_FOUND");setError(e instanceof ResearchError&&e.code==="NOT_FOUND"?"尚未查到这次操作记录，可以按原内容重试。":message(e));}}finally{flight.current=false;if(!controller.signal.aborted)setBusy(false);}}
 const selected=items.filter(v=>chosen.includes(v.id)&&v.observation&&!("shein" in v.observation));
 const groups=new Map<string,ResearchCandidate[]>();for(const v of selected.filter(v=>v.state==="APPROVED_FOR_TEST")){const batch=v.observation!.batchId;groups.set(batch,[...(groups.get(batch)??[]),v]);}
 const target=(center:"research"|"selection")=>{const query=new URLSearchParams(returnQuery);query.delete("step");if(selected.length>=2){query.set("sample",JSON.stringify(analysisChoices(selected)));if(after)query.set("candidateAfter",after);}return (center==="research"?"/workbench/data/research/":"/workbench/supply/selection/")+projectId+(query.size?"?"+query:"");};
 return <section className={styles.panel+(view!=="embedded"?" "+styles.center:"")} aria-label="研究选品"><header className={styles.heading}><div><small>产品研究</small><h2>{view==="embedded"?"研究选品":view==="research"?"调研资料":"商品评估"}</h2><p>查看收集到的商品资料，比较重点商品，再记录选品理由。</p></div><div className={styles.actions}><Button variant="outline" disabled={busy} onClick={reload}>刷新商品资料</Button>{view!=="embedded" && !(view==="research" && bound)?<Button variant="outline" asChild><Link href={special?"/workbench/data/research/"+projectId:target(view==="research"?"selection":"research")} prefetch={false}>{view==="research"?"进入智能选品":"返回市场调研"}</Link></Button>:null}</div></header>
 {archived?<p role="status">项目已归档，研究记录只读。</p>:null}
 {view==="research" && store && scopeKey && canWrite && context.permissions.includes("workbench.data-market.use")?
  <ResearchOpportunity scope={scope} scopeKey={scopeKey} projectId={projectId} store={store} draft={bound} selected={selected} after={after}
   blocked={locked||!!error||incomplete} onJobChange={()=>{setChosen([]);setCompare(false);}}/>:null}
 {active && opportunityInspection?<ConfirmedOpportunity draft={active} inspection={opportunityInspection}/>:null}
 {pending.error?<p role="alert">本次操作记录暂时无法读取，请检查浏览器设置后刷新。</p>:null}
 {pending.command?<Card className={styles.section}><p>上一次操作还没有确认结果。</p><div className={styles.actions}><Button disabled={busy} onClick={()=>void lookup()}>查询操作结果</Button>{absent&&canWrite?<Button variant="outline" disabled={busy} onClick={()=>void submit(pending.command!.body,pending.command)}>按原内容重试</Button>:null}</div></Card>:null}
 {error?<p role="alert">{error}</p>:null}
 {view!=="selection"?<Card id="research-sources" tabIndex={-1} className={styles.section}><div className={styles.heading}><div><h3>收集与整理资料</h3><p>把已保存的 Amazon 商品资料或 SHEIN 美国站商品卡片添加到项目。SHEIN 卡片可查看和切换资料。</p></div><div className={styles.actions}><Button variant="outline" asChild><Link href="/workbench/data/market" prefetch={false}>去收集资料</Link></Button>{canWrite?<Button disabled={locked} onClick={()=>setPicker(v=>!v)}>添加已有商品资料</Button>:null}</div></div>{picker?<CollectionPicker scope={scope} kind="product" authorizationKey={JSON.stringify([projectId,context.permissions])} disabled={locked} isEligible={item=>(item.source.kind==="amazon_data"||item.source.kind==="shein_us_data")&&!unsupported.includes(item.id)} emptyMessage="这页还没有可添加的商品资料。" onSelect={item=>void submit({action:"link",selections:[selectionOf(item)]},null)}/>:null}
 {view==="research"&&showSheinTrial&&canRead&&context.permissions.includes("workbench.data-market.use")?<SheinUSTrialPanel key={JSON.stringify([scope,projectId])} scope={scope} projectId={projectId} canWrite={canWrite}/>:null}</Card>:null}
 {!special?<Card id="research-evidence" tabIndex={-1} className={styles.section}><div className={styles.heading}><div><h3>已收集的商品</h3><p>这里展示已收集的商品资料。销量、市场需求、利润或合规情况尚未核实，不能据此下结论。</p></div><span>{chosen.length} / 5 项已选</span></div>
 {!canRead?<p>当前身份没有商品资料查看权限。</p>:loading?<p role="status">正在加载商品资料…</p>:!items.length&&!error?<p>还没有添加商品资料。</p>:null}
 {incomplete?<p role="status">部分商品资料暂时无法确认，请刷新后再查看。</p>:null}
 <div className={styles.grid}>{items.map(v=><CandidateCard view={view} key={v.id+":"+v.revision} value={v} checked={selected.some(c=>c.id===v.id)} toggle={()=>{invalidateChoice();setCompare(false);setChosen(old=>old.includes(v.id)?old.filter(id=>id!==v.id):old.length<5?[...old,v.id]:old);}} disabled={locked} canWrite={canWrite} chooseDisabled={!v.observation||"shein" in v.observation||busy||!!pending.command||(!chosen.includes(v.id)&&chosen.length>=5)} command={body=>void submit(body,null)}/>)}</div>
 <div className={styles.actions}>{after?<Button variant="outline" onClick={()=>{invalidateChoice();setChosen([]);setAfter("");}}>返回第一页商品</Button>:null}{next?<Button variant="outline" onClick={()=>{invalidateChoice();setChosen([]);setAfter(next);}}>下一页商品</Button>:null}</div></Card>:null}
 {special?null:view==="embedded"?<Card className={styles.section}><h3>重点商品比较</h3><p>选择2–5件商品，比较价格和特点。需要 AI 建议时，先确认分析设置和费用；现有资料不能代表整个市场。</p><Button disabled={selected.length<2||selected.length>5||busy||loading} onClick={()=>setCompare(true)}>比较所选商品</Button>{compare&&selected.length>=2?<ResearchComparison key={JSON.stringify(selected.map(v=>v.evidenceDigest))} scope={scope} items={selected.map(v=>v.observation!.item)}/>:null}</Card>:<ResearchAnalysisFlow scope={scope} projectId={projectId} selected={selected} archived={archived} view={view} evidenceReadKey={JSON.stringify([refresh,after])} evidenceReadable={canRead&&!loading&&!error&&!incomplete}/>}
 {view==="selection"?<Card id="research-decisions" tabIndex={-1} className={styles.section}>
   <h3>评估与选品决定</h3><p>先查看商品资料，再逐件记录选择理由。AI 建议仅供参考，暂未开通 AI 也可继续作决定。</p>
   <div className={styles.grid}>{items.filter(v=>!special||active?.confirmed?.representative===v.id).map(v=>{
    const binding=canPreview?costBinding(v):null;
    const previewKey=binding?JSON.stringify([costPageKey,binding]):null;
    const open=previewKey!==null&&openedCost===previewKey;
    return <article key={v.id+":"+v.revision} className={styles.candidate}>
     <div className={styles.heading}><h4>{v.observation?.title||"这份资料暂时无法查看"}</h4><span className={styles.badge}>{states[v.state]}</span></div>
     {v.reason?<p>当前理由：{v.reason}</p>:null}
          {v.supplierReview&&canRead&&<div className="grid min-w-0 gap-3">
      <div><Button type="button" variant="secondary" aria-expanded={openedSupplier===v.id} onClick={()=>setOpenedSupplier(openedSupplier===v.id?"":v.id)}>{openedSupplier===v.id?"收起货源核对":"核对已保存货源"}</Button></div>
      {openedSupplier===v.id&&<ResearchSupplierReview key={JSON.stringify([scope,projectId,v.id,v.revision,canWrite,context.permissions])}
       scope={scope} projectId={projectId} candidateId={v.id} canWrite={canWrite} onSaved={(revision,before)=>{if(before!==undefined)advanceRevision(active,v.id,before,revision);reload();}}/>}
     </div>}
     {previewKey&&<div className="grid min-w-0 gap-3">
      <div><Button variant="secondary" aria-expanded={open} onClick={()=>setOpenedCost(open?null:previewKey)}>{open?"收起参考试算":"填写经营假设"}</Button></div>
      {open&&binding&&<ResearchCostPreview key={previewKey} verifyEvidence={signal=>verifyCostEvidence(binding,signal)}/>}
     </div>}
     <CandidateDecision value={v} canWrite={canWrite} disabled={locked} command={body=>void submit(body,null)}/>
    </article>;
   })}</div>
 </Card>:null}
 {special?<Card id="research-transfer" tabIndex={-1} className={styles.section}>
  <h3>转入待上传</h3>
  <p>仅转入本次确认的代表商品。请先批准本次选品，再核对这 1 件商品。若已在之前批准，可点「重新了解这件商品」重新评估。</p>
  <ResearchTransferButton disabled={locked||incomplete||!!error||!context.permissions.includes("workbench.supply.manage")||
   !items.some(candidate=>candidate.id===active?.confirmed?.representative && candidate.state==="APPROVED_FOR_TEST" &&
    !!active && active.confirmed?.approvalRevision===candidate.revision && candidateMatchesJob(candidate,active.job) &&
    active.confirmed?.choices.some(choice=>choice.candidateId===candidate.id && choice.expectedRevision===String(candidate.revision)))}
   prepare={startTransfer}/>
 </Card>:view!=="research"?<Card id="research-transfer" tabIndex={-1} className={styles.section}><h3>转入待上传</h3><p>勾选已批准测试的商品，再到“我的数据”核对。分批收集的资料需要分别确认；批准测试不代表已采购、已上架或已完成合规检查。</p>{groups.size===0?<p>请先选择已批准测试、且资料可以查看的商品。</p>:<div className={styles.actions}>{[...groups].map(([batch,values],i)=><Button key={batch} variant="outline" asChild><Link prefetch={false} href={"/workbench/data/mine?"+new URLSearchParams({batchId:batch,transfer:JSON.stringify(values.map(v=>v.observation!.ref.selection))})}>批次 {i+1} · 核对 {values.length} 件商品</Link></Button>)}</div>}<p className={styles.muted}>是否转入成功，请到“商品刊登”查看处理结果。</p></Card>:null}
 </section>;
}
function costBinding(candidate:ResearchCandidate):string|null{
 const observation=candidate.observation;
 if(!observation||"shein" in observation||observation.item.source.kind!=="amazon_data"||
    observation.site.toLowerCase()!=="us"||!candidate.selectedLinkId||!candidate.evidenceDigest)return null;
 const {itemId,revision,publicationId,version}=observation.ref.selection;
 return JSON.stringify([candidate.id,candidate.revision,candidate.selectedLinkId,candidate.evidenceDigest,itemId,revision,publicationId,version]);
}
function message(e:unknown){return e instanceof ResearchError?failures[e.code]??"服务暂不可用，请重新读取。":"服务暂不可用，请重新读取。";}
function CandidateCard({value:v,checked,toggle,disabled,chooseDisabled,canWrite,command,view}:{view:"embedded"|"research"|"selection";value:ResearchCandidate;checked:boolean;toggle:()=>void;disabled:boolean;chooseDisabled:boolean;canWrite:boolean;command:(body:ResearchCommand)=>void}){
 const o=v.observation;
 return <article className={styles.candidate}><div className={styles.heading}><label className={styles.select}><input type="checkbox" checked={checked} disabled={chooseDisabled} onChange={toggle}/><strong>{o?.title||"这份资料暂时无法查看"}</strong></label><span className={styles.badge}>{states[v.state]}</span></div>
 {o?<>{"shein" in o?<SHEINCardFacts observation={o}/>:<><p>{o.site.toUpperCase()} · {o.asin} · {o.currency??"币种未知"} {o.price??"价格未知"}</p><dl><dt>资料来源</dt><dd>Amazon 公开商品页面</dd><dt>资料收集范围</dt><dd>{o.query?.mode} · {o.query?.keyword||o.query?.categoryNode||(o.query?.asinCount+" 个 ASIN")} · 最多 {o.query?.limit} 条</dd><dt>收集时间</dt><dd>{o.capturedAt}</dd><dt>还缺少的信息</dt><dd>{o.missing.length?o.missing.join("、"):"这份资料没有列出缺少的信息，不代表市场信息已齐全"}</dd></dl></>}<details><summary>资料来源详情</summary><p>资料版本：{o.ref.selection.version} · 解析版本：{o.parserVersion}</p><p>资料校验编号：{o.ref.checksum}</p><p>Publication：{o.ref.selection.publicationId}</p><p>Job：{o.ref.jobId}</p></details><a href={o.sourceUrl} target="_blank" rel="noreferrer">查看来源页面 ↗</a></>:<p>项目仍保留这件商品和已有选择记录；当前无法查看的商品资料不再显示。</p>}
 {v.reason&&view!=="selection"?<p>当前理由：{v.reason}</p>:null}
 {v.evidence.length>0?<details><summary>已收集的资料记录（最近20条及当前使用资料）</summary>{v.evidence.map(l=><div className={styles.evidence} key={l.id}><span>{l.observation?`${l.observation.capturedAt||"收集时间未知"} · v${l.observation.ref.selection.version}`:"这份资料暂时无法查看"}</span>{canWrite&&view!=="selection"?<span className={styles.actions}>{l.observation&&l.id!==v.selectedLinkId?<Button size="sm" variant="outline" disabled={disabled} onClick={()=>command({action:"select",candidateId:v.id,expectedRevision:v.revision,linkId:l.id})}>使用这份资料</Button>:null}<Button size="sm" variant="ghost" disabled={disabled} onClick={()=>command({action:"unlink",candidateId:v.id,expectedRevision:v.revision,linkId:l.id})}>从项目移除</Button></span>:null}</div>)}</details>:null}
 {view!=="selection"?<CandidateDecision value={v} canWrite={canWrite&&view!=="research"} disabled={disabled} command={command}/>:null}</article>;
}
const sheinMissingLabels:Record<string,string>={title:"商品标题",capturedAt:"收集时间",variantId:"具体规格",price:"卡片展示价","price.promotionNote":"优惠条件","price.taxIncluded":"是否含税","price.unit":"计价单位"};
function SHEINCardFacts({observation:o}:{observation:Extract<NonNullable<ResearchCandidate["observation"]>,{shein:unknown}>}){
 const {price,nativeGoodsId}=o.shein;
 return <><p>美国站 · {nativeGoodsId}</p><dl>
  <dt>资料来源</dt><dd>SHEIN 美国站商品卡片</dd>
  <dt>资料收集范围</dt><dd>{o.query.mode==="category"?"类目 "+o.query.categoryRef:"关键词 "+o.query.keyword}</dd>
  <dt>卡片展示价</dt><dd>{price?price.currency+" "+price.amount:"价格未知"}</dd>
  {price?.promotionNote?<><dt>优惠条件</dt><dd>{price.promotionNote}</dd></>:null}
  {price?.taxIncluded!=null?<><dt>税费口径</dt><dd>{price.taxIncluded?"含税":"不含税"}</dd></>:null}
  {price?.unit?<><dt>计价单位</dt><dd>{price.unit}</dd></>:null}
  <dt>收集时间</dt><dd>{o.capturedAt||"未知"}</dd>
  <dt>还缺少的信息</dt><dd>{o.missing.length?o.missing.map(field=>sheinMissingLabels[field]??field).join("、"):"这份资料没有列出缺少的信息"}</dd>
 </dl><p>此卡片可用于查看研究资料。</p></>;
}

function CandidateDecision({value:v,canWrite,disabled,command}:{value:ResearchCandidate;canWrite:boolean;disabled:boolean;command:(body:ResearchCommand)=>void}){
 const [reason,setReason]=useState("");const change={candidateId:v.id,expectedRevision:v.revision,evidenceDigest:v.evidenceDigest!};
 return <>{canWrite&&v.observation&&!("shein" in v.observation)?<div className={styles.decision}>{v.state==="TO_RESEARCH"?<Button disabled={disabled} onClick={()=>command({action:"review",...change})}>开始评估</Button>:v.state==="PENDING_CONFIRMATION"?<><label>选择理由<textarea value={reason} onChange={e=>setReason(e.target.value)} maxLength={4096} rows={3}/></label><div className={styles.actions}><Button disabled={disabled||!reason.trim()} onClick={()=>command({action:"decide",...change,state:"APPROVED_FOR_TEST",reason:reason.trim()})}>批准测试</Button><Button variant="outline" disabled={disabled||!reason.trim()} onClick={()=>command({action:"decide",...change,state:"REJECTED",reason:reason.trim()})}>放弃该商品</Button></div></>:null}{v.state!=="TO_RESEARCH"?<Button variant="outline" disabled={disabled} onClick={()=>command({action:"reset",...change})}>重新了解这件商品</Button>:null}</div>:null}
 {v.history.length?<details><summary>过往选择记录（最近20条）</summary>{v.history.map(h=><div key={h.revision} className={styles.evidence}><p>{states[h.state]} · {h.at} · r{h.revision}</p>{h.reason?<p>{h.reason}</p>:null}<small>{h.evidenceDigest?"当时使用的资料："+h.evidenceDigest:"当时这份资料暂时无法查看"}</small></div>)}</details>:null}</>;
}
