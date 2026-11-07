"use client";
import Link from "next/link";
import {useEffect,useRef,useState} from "react";
import {useQuery,useQueryClient} from "@tanstack/react-query";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {configurationRequest,type ConfigurationScope} from "@/lib/api/agent-configuration";
import {MarketAPIError} from "@/lib/api/supply-market";
import {inspectResearch,prepareResearch,executeResearch,readResearchPreparation,readResearchRun,recentResearchRuns} from "@/lib/api/research-analysis";
import {matchesResearchInspection} from "@/lib/api/research-handoff";
import {catalogEntrySchema} from "@/lib/contracts/agent-configuration";
import {executionModeViewSchema} from "@/lib/contracts/execution-mode";
import {researchAnalysisInput,researchAnalysisIntent,researchPrepareSchema,researchSavedPreparation,type ResearchAnalysisIntent,type ResearchInspection} from "@/lib/contracts/research-analysis";
import type {ResearchCandidate} from "@/lib/contracts/market-research";
import {FactTable,factLabel} from "../agents/product-selection-flow";
import {executionModeLabels} from "../agents/execution-mode-settings";
import {configurationError} from "../agents/agent-configuration-hooks";
import {useResourcePending} from "../resources/resource-pending";
import styles from "./research-panel.module.css";
export function analysisChoices(selected:ResearchCandidate[]){return selected.map(c=>({candidateId:c.id,expectedRevision:String(c.revision),linkId:c.selectedLinkId,evidenceDigest:c.evidenceDigest,selection:{itemId:c.observation?.ref.selection.itemId,expectedRevision:String(c.observation?.ref.selection.revision),originalPublicationId:c.observation?.ref.selection.publicationId,originalVersion:c.observation?.ref.selection.version}}));}
function matches(out:ResearchInspection|undefined,input:unknown){const parsed=researchAnalysisInput.safeParse(input);return !!out&&parsed.success&&matchesResearchInspection(out,parsed.data);}
const recommendationNames={TEST_CANDIDATE:"建议测试",RESEARCH_MORE:"建议进一步了解",NOT_RECOMMENDED:"建议暂不选择"};
export function ResearchAnalysisFlow({scope,projectId,selected,archived,view,evidenceReadKey="",evidenceReadable=true}:{scope:ConfigurationScope;projectId:string;selected:ResearchCandidate[];archived:boolean;view:"research"|"selection";evidenceReadKey?:string;evidenceReadable?:boolean}){
 const context=useWorkbenchContext(),queries=useQueryClient();
 const identity={expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId};
 const namespace=["research-analysis",projectId,context.effectiveMemberId??""];
 const pending=useResourcePending(identity,namespace,researchAnalysisIntent,{storage:"local",maxLength:16384});
 const preparation=useResourcePending(identity,[...namespace,"prepared"],researchSavedPreparation,{storage:"local",maxLength:16384});
 const saved=preparation.command;
 const [goal,setGoal]=useState("请根据本次保存样本比较差异，说明缺项和后续核实建议。"),[busy,setBusy]=useState(false),[failure,setFailure]=useState<unknown>(),[chosen,setChosen]=useState<string>(),[after,setAfter]=useState(""),[absentKey,setAbsentKey]=useState<string>();
 const absent=!!pending.command&&absentKey===pending.command.key;
 const active=useRef(true),flight=useRef(false),controller=useRef<AbortController|null>(null);
 useEffect(()=>{active.current=true;return()=>{active.current=false;controller.current?.abort();};},[]);
 const register=context.registerOrganizationSwitchGuard;
 useEffect(()=>register(()=>!flight.current&&!pending.command&&!pending.error&&!preparation.error),[register,pending.command,pending.error,preparation.error]);
 const keys=[...namespace,scope.userId,scope.organizationId],input=researchAnalysisInput.safeParse({candidates:analysisChoices(selected)}),signature=JSON.stringify(input.success?input.data:null);
 const facts=useQuery({queryKey:[...keys,"facts",signature],queryFn:({signal})=>inspectResearch(scope,projectId,input.success?input.data:null,signal),enabled:false,retry:false});
 const catalog=useQuery({queryKey:[...keys,"catalog"],queryFn:({signal})=>configurationRequest(scope,"product.research.agent",catalogEntrySchema,{signal}),enabled:view==="selection",retry:false});
 const entry=catalog.data;
 const mode=useQuery({queryKey:[...keys,"mode"],queryFn:({signal})=>configurationRequest(scope,"execution-modes/registered_agent/product.research.agent",executionModeViewSchema,{signal}),enabled:view==="selection"&&!!entry&&entry.agent.activation!=="NOT_ENABLED",retry:false});
 const recent=useQuery({queryKey:[...keys,"recent",after],queryFn:({signal})=>recentResearchRuns(scope,projectId,after,signal),enabled:view==="selection"&&!!entry?.canReadRuns,retry:false});
 const pointer=chosen??(pending.command?.kind!=="prepare"?pending.command?.key:undefined)??saved?.key;
 const runKey=[...keys,"run",pointer,evidenceReadKey];
 const run=useQuery({queryKey:runKey,queryFn:({signal})=>readResearchRun(scope,projectId,pointer!,signal),enabled:!!pointer&&evidenceReadable,retry:false,refetchInterval:q=>q.state.data?.phase==="running"?3000:false,refetchIntervalInBackground:false});
 const denied=(e:unknown)=>e instanceof MarketAPIError&&[403,401].includes(e.status);
 const masked=denied(failure)||denied(facts.error)||denied(run.error)||!!run.error&&!(run.error instanceof MarketAPIError&&run.error.status===404);
 const evidencePending=!evidenceReadable||run.isFetching;
 const display=!evidencePending&&!masked&&!facts.error?(run.data&&!run.error?run.data.inspection:!pointer||pointer===saved?.key?facts.data:undefined):undefined;
 const locked=busy||!evidenceReadable||!!pending.command||pending.error||!pending.ready||preparation.error||!preparation.ready;
 const textReady=!!entry?.canUse&&entry.agent.activation==="ENABLED"&&entry.capabilities.some(c=>c.id==="text.generate"&&c.readiness==="AVAILABLE");
 const goalCheck=researchPrepareSchema.shape.goalSummary.safeParse(goal.trim());
 const factsMatch=!facts.error&&input.success&&matches(facts.data,input.data);
 const preparedFactsMatch=!!saved&&!facts.error&&matches(facts.data,{candidates:saved.input.candidates});
 const modeMatch=!!saved&&!mode.error&&mode.data?.revision===saved.prepared.modeRevision;
 const showRun=run.data&&!run.error&&!masked&&!evidencePending?run.data:undefined;
 async function inspect(){if(flight.current||archived||!input.success)return;flight.current=true;setBusy(true);setFailure(undefined);try{await facts.refetch({throwOnError:true});if(active.current&&!saved&&!pending.command)setChosen(undefined);}catch(e){if(active.current)setFailure(e);}finally{flight.current=false;if(active.current)setBusy(false);}}
 async function send(intent:ResearchAnalysisIntent){
 if(flight.current||!active.current||!pending.ready||intent.projectId!==projectId||archived)return;
 if(intent.kind==="start"&&!pending.command&&(!preparedFactsMatch||!modeMatch)){setFailure(new Error("请重新核对本次准备的依据和权限版本。"));return;}
 flight.current=true;setBusy(true);setFailure(undefined);setAbsentKey(undefined);const abort=new AbortController();controller.current=abort;
 let freshStart=false;
 try{
 freshStart=intent.kind==="start"&&!pending.read();
 pending.persist(intent);
 if(intent.kind==="prepare"){
 const prepared=await prepareResearch(scope,intent,abort.signal);if(prepared.contextId!==projectId||prepared.requestKey!==intent.key)throw new MarketAPIError("OUTCOME_UNKNOWN",502);
 if(active.current){preparation.persist({key:intent.key,projectId,input:intent.body,prepared});pending.clear(intent);setChosen(intent.key);}
 }else{
 const result=await executeResearch(scope,intent,abort.signal);if(result.contextId!==projectId||result.requestKey!==intent.key)throw new MarketAPIError("OUTCOME_UNKNOWN",502);
 if(active.current){setChosen(intent.key);queries.setQueryData([...keys,"run",intent.key,evidenceReadKey],result);pending.clear(intent);void recent.refetch();}
 }
 }catch(e){
  if(!active.current)return;
  if(freshStart&&e instanceof MarketAPIError&&e.status===409&&e.code==="CONFIGURATION_CHANGED"&&e.operationMayExist===false){
   try{
    const original=preparation.read();
    if(original?.key===intent.key)preparation.clear(original);
    pending.clear(intent);
   }catch(storageError){setFailure(storageError);return;}
   setChosen(undefined);
   void queries.resetQueries({queryKey:[...keys,"facts"]});
   void mode.refetch();
   setFailure(new Error("本次分析尚未开始。请重新查看商品资料，并按当前分析设置重新准备。"));
  }else setFailure(e);
 }finally{flight.current=false;if(active.current)setBusy(false);}
 }
 async function verify(){const intent=pending.command;if(!intent||flight.current)return;flight.current=true;setBusy(true);setFailure(undefined);setAbsentKey(undefined);const abort=new AbortController();controller.current=abort;
 try{if(intent.kind==="prepare"){
 const prepared=await readResearchPreparation(scope,projectId,intent.key,abort.signal);if(prepared.contextId!==projectId||prepared.requestKey!==intent.key)throw new MarketAPIError("OUTCOME_UNKNOWN",502);
 if(active.current){preparation.persist({key:intent.key,projectId,input:intent.body,prepared});pending.clear(intent);setChosen(intent.key);}
 }else{
 const result=await readResearchRun(scope,projectId,intent.key,abort.signal);if(!active.current)return;
 setChosen(intent.key);queries.setQueryData([...keys,"run",intent.key,evidenceReadKey],result);
 if(intent.kind==="start"||BigInt(result.revision)>BigInt(intent.body.revision))pending.clear(intent);
 }
 }catch(e){if(active.current){setFailure(e);setAbsentKey(e instanceof MarketAPIError&&e.status===404&&e.operationMayExist===false?intent.key:undefined);}}finally{flight.current=false;if(active.current)setBusy(false);}}
 const noRun=run.error instanceof MarketAPIError&&run.error.status===404;
 const canClear=!!saved&&!locked&&!masked&&!run.isFetching&&(!!showRun&&showRun.phase!=="running"&&showRun.usageStatus!=="unknown_reserved"||noRun);
 return <>
 <Card id="research-summary" tabIndex={-1} className={styles.section}><div className={styles.heading}><div><h3>{view==="research"?"重点商品比较":"价格与特点比较"}</h3><p>这里只比较你选中的2–5件商品。不同币种、不同价格条件分开查看；没有价格的信息保留为空，不当作0元。</p></div><Button variant="outline" disabled={locked||archived||!input.success||facts.isFetching} onClick={()=>void inspect()}>查看所选商品比较</Button></div>
 {facts.isFetching?<p role="status">正在确认所选商品资料…</p>:null}{facts.error?<p role="alert">部分商品资料暂时无法确认，请刷新后重新选择。</p>:null}
 {!display?<p>先选择2–5件资料可以查看的商品，再比较价格和特点。销量、市场需求、趋势和完整利润尚未核实。</p>:<><div className={styles.summary}><strong>本次比较 {display.summary.selected} 件</strong><span>缺少可比价格的商品 {display.summary.unpriced} 件</span>{display.summary.prices.map(p=><p key={p.currency+"|"+p.basis}>{p.currency} 页面报价：{p.minimum}–{p.maximum} · {p.count} 件商品使用相同价格条件</p>)}</div><FactTable inspection={{products:display.candidates.map(c=>({ref:c.ref,sourceKind:"amazon_data",facts:c.facts,missing:c.missing}))}}/>{display.candidates.some(c=>c.warnings.length)?<p>这些资料供调研参考；请结合每件商品缺少的信息判断，不把资料不全当作已经了解清楚。</p>:null}</>}
 </Card>
 {failure?<p role="alert">{failure instanceof Error&&!(failure instanceof MarketAPIError)?failure.message:configurationError(failure)}</p>:null}
 {pending.error||preparation.error?<p role="alert">本次操作记录暂时无法读取，请检查浏览器设置后继续。</p>:null}
 {pending.command?<Card className={styles.section}><h3>查询本次分析结果</h3><p>操作编号 {pending.command.key}，分析结果还待确认。</p><div className={styles.actions}><Button disabled={busy} onClick={()=>void verify()}>查询本次分析结果</Button><Button variant="outline" disabled={busy||archived} onClick={()=>void send(pending.command!)}>{pending.command.kind==="prepare"?"按原内容重新准备":"按原内容重试分析"}</Button>{absent&&pending.command.kind==="prepare"?<Button variant="outline" disabled={busy} onClick={()=>{const intent=pending.command;if(!intent||intent.kind!=="prepare"||intent.key!==absentKey||flight.current)return;pending.clear(intent);setAbsentKey(undefined);setChosen(undefined);setFailure(undefined);void queries.resetQueries({queryKey:[...keys,"facts"]});}}>取消本次准备并重新选择</Button>:null}</div>{absent?<p>尚未查到这次操作记录，可以按原内容重试；尚未提交的准备也可以取消后重新选择商品。</p>:null}</Card>:null}
 {pointer&&evidencePending?<p role="status">正在重新确认本次分析使用的资料，比较结果和建议暂时不显示。</p>:null}
 {view==="selection"?<Card id="research-analysis" tabIndex={-1} className={styles.section}><div className={styles.heading}><div><h3>AI分析建议</h3><p>AI 建议会保存在本次分析记录中。是否选择商品以及选择理由，仍由你逐件确认。</p></div><Button asChild variant="outline"><Link href="/workbench/agents/mine/product.research.agent" prefetch={false}>AI分析设置</Link></Button></div>
 {!textReady?<p>AI 分析暂未开通。你仍可比较商品资料并记录选品决定。</p>:null}
 {catalog.error?<p role="status">暂时无法查看 AI 分析设置。</p>:null}
 <Button variant="ghost" disabled={busy} onClick={()=>{void catalog.refetch();void mode.refetch();}}>刷新分析设置</Button>
 {!saved&&!pending.command?<><label className={styles.goal}>这次想重点了解什么<Input aria-label="这次想重点了解什么" value={goal} disabled={locked} onChange={e=>setGoal(e.target.value)}/></label><Button disabled={locked||archived||!textReady||!mode.data||mode.isFetching||!!mode.error||!factsMatch||!goalCheck.success} onClick={()=>{if(!input.success||!goalCheck.success||!mode.data)return;const key=crypto.randomUUID();void send({kind:"prepare",projectId,key,body:{...input.data,requestKey:key,expectedModeRevision:mode.data.revision,goalSummary:goalCheck.data}});}}>准备AI分析</Button></>:null}
 {saved&&!masked?<section className={styles.confirmation} aria-label="确认本次AI分析"><h4>确认本次AI分析</h4><p>{saved.input.goalSummary}</p><p>所选 {saved.input.candidates.length} 件商品 · 最多调用AI2次 · 处理用量上限 {saved.prepared.maximumTokens} tokens</p><p>本次费用估算上限 {(saved.prepared.maximumCostMicros/1e6).toFixed(6)} {saved.prepared.currency} · 每次AI处理费用估算上限 {(saved.prepared.maximumCallCostMicros/1e6).toFixed(6)} {saved.prepared.currency}</p>{saved.prepared.referencePriceVersion?<p>参考价版本：{saved.prepared.referencePriceVersion}。费用按此版本估算，不代表厂商最终账单。</p>:null}<p>本次操作确认方式：{mode.data?executionModeLabels[mode.data.mode]:"待核对"}。点击确认后才开始分析。</p>{!modeMatch?<p role="status">分析的操作设置尚未确认或已变化，请重新确认后再准备。</p>:null}{!preparedFactsMatch?<p>请重新查看本次选中的商品资料，确认与准备分析时一致。</p>:null}<div className={styles.actions}><Button disabled={locked||archived||!textReady||!modeMatch||!preparedFactsMatch||run.isFetching||!!run.data||!!run.error&&!noRun} onClick={()=>void send({kind:"start",projectId,key:saved.key,body:{configurationSnapshotRef:saved.prepared.configurationSnapshotRef}})}>确认生成AI建议</Button><Button variant="outline" disabled={!canClear} onClick={()=>{if(!saved||!canClear)return;preparation.clear(saved);setChosen(undefined);setFailure(undefined);}}>分析另一组商品</Button></div><small>更换商品只结束本页的待确认准备；已有分析记录和已发生的费用仍会保留。</small></section>:null}
 {masked?<p role="alert">本次分析使用的部分资料现在无法查看，比较结果和建议暂时不显示；已有操作记录仍会保留。</p>:null}
 {showRun?<section className={styles.result}><h4>{showRun.validated?"AI建议已生成":showRun.phase==="running"?"AI正在分析":"AI分析还未完成"}</h4><p>本次关注的问题：{showRun.goalSummary}</p>{showRun.stopReason?<p>停止原因：{showRun.stopReason}</p>:null}<p>{showRun.usageStatus==="unknown_reserved"?"这次分析的结果或费用还未确认，请查询操作结果。":`已记录处理用量 ${showRun.tokens} tokens · 费用估算 ${(showRun.costMicros/1e6).toFixed(6)} ${showRun.currency}`}</p>
 {showRun.validated&&showRun.proposal?<><p>{showRun.proposal.overview}</p><Citations ids={showRun.proposal.overviewEvidenceIds} inspection={showRun.inspection!}/>{showRun.proposal.assessments.map(a=><section key={a.candidateId}><h4>{recommendationNames[a.recommendation]} · {String(showRun.inspection?.candidates.find(c=>c.research.CandidateID===a.candidateId)?.facts.find(f=>f.path==="title")?.value??a.candidateId)}</h4><p>{a.advice}</p><Citations ids={a.evidenceIds} inspection={showRun.inspection!}/></section>)}</>:null}{showRun.unresolved.length?<p>待核实：{showRun.unresolved.join("；")}</p>:null}
 {showRun.phase==="interrupted"?<Button disabled={locked||archived||!textReady} onClick={()=>void send({kind:"resume",projectId,key:showRun.requestKey,body:{configurationSnapshotRef:showRun.configurationSnapshotRef,revision:showRun.revision,feedback:showRun.goalSummary}})}>继续本次分析</Button>:null}</section>:null}
 {entry?.canReadRuns?<section><h4>我的分析记录</h4>{recent.error?<p role="alert">分析记录暂时无法完整显示。</p>:recent.data?.items.length?<ul>{recent.data.items.map(r=><li key={r.runId}><Button variant="ghost" disabled={locked} onClick={()=>{setChosen(r.requestKey);setFailure(undefined);}}>{new Date(r.startedAt).toLocaleString("zh-CN")} · {r.phase==="human_review_required"?"查看建议":"查看分析进度"}</Button></li>)}</ul>:<p>这页还没有可查看的分析记录。</p>}<div className={styles.actions}><Button variant="outline" disabled={busy} onClick={()=>void recent.refetch()}>刷新分析记录</Button>{after?<Button variant="outline" disabled={busy} onClick={()=>setAfter("")}>返回第一页记录</Button>:null}{recent.data?.nextCursor?<Button variant="outline" disabled={busy} onClick={()=>setAfter(recent.data!.nextCursor)}>下一页分析记录</Button>:null}</div></section>:null}
 </Card>:null}
 </>;
}
function Citations({ids,inspection}:{ids:string[];inspection:ResearchInspection}){return <ul className={styles.citations}>{ids.map(id=>{const candidate=inspection.candidates.find(c=>c.facts.some(f=>f.id===id)),fact=candidate?.facts.find(f=>f.id===id);return <li key={id}>依据 · {candidate?.observation.site.toUpperCase()} {candidate?.observation.asin} · {fact?factLabel(fact.path):"未提供的信息"}：{JSON.stringify(fact?.value)}</li>;})}</ul>;}
