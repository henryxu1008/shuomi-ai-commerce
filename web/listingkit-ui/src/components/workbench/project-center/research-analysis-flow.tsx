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
function matches(out:ResearchInspection|undefined,input:unknown){const parsed=researchAnalysisInput.safeParse(input);if(!out||!parsed.success||out.candidates.length!==parsed.data.candidates.length)return false;return parsed.data.candidates.every(c=>out.candidates.some(p=>p.research.CandidateID===c.candidateId&&p.research.CandidateRevision===c.expectedRevision&&p.research.SelectedLinkID===c.linkId&&p.research.EvidenceDigest===c.evidenceDigest&&p.ref.ItemID===c.selection.itemId&&p.ref.ItemRevision===c.selection.expectedRevision&&p.ref.PublicationID===c.selection.originalPublicationId&&p.ref.CatalogVersion===c.selection.originalVersion));}
const recommendationNames={TEST_CANDIDATE:"建议测试",RESEARCH_MORE:"建议继续研究",NOT_RECOMMENDED:"建议暂不选择"};
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
 try{pending.persist(intent);
 if(intent.kind==="prepare"){
 const prepared=await prepareResearch(scope,intent,abort.signal);if(prepared.contextId!==projectId||prepared.requestKey!==intent.key)throw new MarketAPIError("OUTCOME_UNKNOWN",502);
 if(active.current){preparation.persist({key:intent.key,projectId,input:intent.body,prepared});pending.clear(intent);setChosen(intent.key);}
 }else{
 const result=await executeResearch(scope,intent,abort.signal);if(result.contextId!==projectId||result.requestKey!==intent.key)throw new MarketAPIError("OUTCOME_UNKNOWN",502);
 if(active.current){setChosen(intent.key);queries.setQueryData([...keys,"run",intent.key,evidenceReadKey],result);pending.clear(intent);void recent.refetch();}
 }
 }catch(e){if(active.current)setFailure(e);}finally{flight.current=false;if(active.current)setBusy(false);}
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
 <Card id="research-summary" tabIndex={-1} className={styles.section}><div className={styles.heading}><div><h3>{view==="research"?"所选样本汇总":"所选证据与事实比较"}</h3><p>仅包括明确选中的2–5项原观察。同币种、同页面报价口径分别比较；未知不按0计算。</p></div><Button variant="outline" disabled={locked||archived||!input.success||facts.isFetching} onClick={()=>void inspect()}>读取所选样本</Button></div>
 {facts.isFetching?<p role="status">正在核实全部所选证据…</p>:null}{facts.error?<p role="alert">所选证据未能全部核实，请刷新资料后重新选择。</p>:null}
 {!display?<p>先选择2–5个可读候选，再读取事实。销量、需求、趋势和完整利润均未知。</p>:<><div className={styles.summary}><strong>本次样本 {display.summary.selected} 项</strong><span>未参与价格比较 {display.summary.unpriced} 项</span>{display.summary.prices.map(p=><p key={p.currency+"|"+p.basis}>{p.currency} 页面报价：{p.minimum}–{p.maximum} · {p.count} 项同口径观察</p>)}</div><FactTable inspection={{products:display.candidates.map(c=>({ref:c.ref,sourceKind:"amazon_data",facts:c.facts,missing:c.missing}))}}/>{display.candidates.some(c=>c.warnings.length)?<p>保存样本仅供研究；完整字段省略与来源限制见每项缺失信息。</p>:null}</>}
 </Card>
 {failure?<p role="alert">{failure instanceof Error&&!(failure instanceof MarketAPIError)?failure.message:configurationError(failure)}</p>:null}
 {pending.error||preparation.error?<p role="alert">原操作记录无法读取，请恢复浏览器存储后继续。</p>:null}
 {pending.command?<Card className={styles.section}><h3>核实原分析请求</h3><p>保留原请求 {pending.command.key}，当前结果待核实。</p><div className={styles.actions}><Button disabled={busy} onClick={()=>void verify()}>核实原分析请求</Button><Button variant="outline" disabled={busy||archived} onClick={()=>void send(pending.command!)}>{pending.command.kind==="prepare"?"使用原键重试准备":"使用原键重试分析"}</Button>{absent&&pending.command.kind==="prepare"?<Button variant="outline" disabled={busy} onClick={()=>{const intent=pending.command;if(!intent||intent.kind!=="prepare"||intent.key!==absentKey||flight.current)return;pending.clear(intent);setAbsentKey(undefined);setChosen(undefined);setFailure(undefined);void queries.resetQueries({queryKey:[...keys,"facts"]});}}>放弃未提交的准备并重新选择</Button>:null}</div>{absent?<p>原精确请求已确认没有回执，可以使用原键重试；未提交的准备也可显式结束后重新选样。</p>:null}</Card>:null}
 {pointer&&evidencePending?<p role="status">正在重新核实原分析依据，原汇总与建议暂时隐藏。</p>:null}
 {view==="selection"?<Card id="research-analysis" tabIndex={-1} className={styles.section}><div className={styles.heading}><div><h3>研究证据分析</h3><p>AI建议独立保存在原运行历史。人工决定和理由仍由你逐项确认。</p></div><Button asChild variant="outline"><Link href="/workbench/agents/mine/product.research.agent" prefetch={false}>研究分析配置</Link></Button></div>
 {!textReady?<p>当前研究模型尚未就绪。可以继续查看证据、比较事实并作人工决定。</p>:null}
 {catalog.error?<p role="status">当前无法读取研究分析配置。</p>:null}
 <Button variant="ghost" disabled={busy} onClick={()=>{void catalog.refetch();void mode.refetch();}}>刷新分析配置</Button>
 {!saved&&!pending.command?<><label className={styles.goal}>本次分析目标<Input aria-label="本次分析目标" value={goal} disabled={locked} onChange={e=>setGoal(e.target.value)}/></label><Button disabled={locked||archived||!textReady||!mode.data||!!mode.error||!factsMatch||!goalCheck.success} onClick={()=>{if(!input.success||!goalCheck.success||!mode.data)return;const key=crypto.randomUUID();void send({kind:"prepare",projectId,key,body:{...input.data,requestKey:key,expectedModeRevision:mode.data.revision,goalSummary:goalCheck.data}});}}>准备研究分析</Button></>:null}
 {saved&&!masked?<section className={styles.confirmation} aria-label="确认本次研究分析"><h4>确认本次研究分析</h4><p>{saved.input.goalSummary}</p><p>所选 {saved.input.candidates.length} 项 · 最多2次模型调用 · 总量最多 {saved.prepared.maximumTokens} tokens</p><p>总预算上限 {(saved.prepared.maximumCostMicros/1e6).toFixed(6)} {saved.prepared.currency} · 单次调用上限 {(saved.prepared.maximumCallCostMicros/1e6).toFixed(6)} {saved.prepared.currency}</p><p>当前权限模式：{mode.data?executionModeLabels[mode.data.mode]:"待核对"}。点击后才开始调用。</p>{!modeMatch?<p role="status">权限版本尚未核对或已变化，需要重新准备。</p>:null}{!preparedFactsMatch?<p>需重新读取与本次准备一致的所选样本。</p>:null}<div className={styles.actions}><Button disabled={locked||archived||!textReady||!modeMatch||!preparedFactsMatch||run.isFetching||!!run.data||!!run.error&&!noRun} onClick={()=>void send({kind:"start",projectId,key:saved.key,body:{configurationSnapshotRef:saved.prepared.configurationSnapshotRef}})}>确认生成研究建议</Button><Button variant="outline" disabled={!canClear} onClick={()=>{if(!saved||!canClear)return;preparation.clear(saved);setChosen(undefined);setFailure(undefined);}}>准备另一组分析</Button></div><small>准备另一组仅结束本页待确认状态；原快照、运行和已发生用量继续保留。</small></section>:null}
 {masked?<p role="alert">原依据当前不可完整访问，跨候选汇总与建议已隐藏；原请求仍保留。</p>:null}
 {showRun?<section className={styles.result}><h4>{showRun.validated?"研究建议已生成":showRun.phase==="running"?"研究分析执行中":"研究分析尚未完成"}</h4><p>原目标：{showRun.goalSummary}</p>{showRun.stopReason?<p>停止原因：{showRun.stopReason}</p>:null}<p>{showRun.usageStatus==="unknown_reserved"?"原结果或费用未知，请核实原请求。":`已观察 ${showRun.tokens} tokens · ${(showRun.costMicros/1e6).toFixed(6)} ${showRun.currency}`}</p>
 {showRun.validated&&showRun.proposal?<><p>{showRun.proposal.overview}</p><Citations ids={showRun.proposal.overviewEvidenceIds} inspection={showRun.inspection!}/>{showRun.proposal.assessments.map(a=><section key={a.candidateId}><h4>{recommendationNames[a.recommendation]} · {String(showRun.inspection?.candidates.find(c=>c.research.CandidateID===a.candidateId)?.facts.find(f=>f.path==="title")?.value??a.candidateId)}</h4><p>{a.advice}</p><Citations ids={a.evidenceIds} inspection={showRun.inspection!}/></section>)}</>:null}{showRun.unresolved.length?<p>待核实：{showRun.unresolved.join("；")}</p>:null}
 {showRun.phase==="interrupted"?<Button disabled={locked||archived||!textReady} onClick={()=>void send({kind:"resume",projectId,key:showRun.requestKey,body:{configurationSnapshotRef:showRun.configurationSnapshotRef,revision:showRun.revision,feedback:showRun.goalSummary}})}>按原请求继续研究分析</Button>:null}</section>:null}
 {entry?.canReadRuns?<section><h4>本项目本人分析历史</h4>{recent.error?<p role="alert">历史暂未能完整核实。</p>:recent.data?.items.length?<ul>{recent.data.items.map(r=><li key={r.runId}><Button variant="ghost" disabled={locked} onClick={()=>{setChosen(r.requestKey);setFailure(undefined);}}>{new Date(r.startedAt).toLocaleString("zh-CN")} · {r.phase==="human_review_required"?"查看建议":"查看运行"}</Button></li>)}</ul>:<p>当前页暂无可读分析记录。</p>}<div className={styles.actions}><Button variant="outline" disabled={busy} onClick={()=>void recent.refetch()}>刷新分析历史</Button>{after?<Button variant="outline" disabled={busy} onClick={()=>setAfter("")}>历史首页</Button>:null}{recent.data?.nextCursor?<Button variant="outline" disabled={busy} onClick={()=>setAfter(recent.data!.nextCursor)}>下一页分析历史</Button>:null}</div></section>:null}
 </Card>:null}
 </>;
}
function Citations({ids,inspection}:{ids:string[];inspection:ResearchInspection}){return <ul className={styles.citations}>{ids.map(id=>{const candidate=inspection.candidates.find(c=>c.facts.some(f=>f.id===id)),fact=candidate?.facts.find(f=>f.id===id);return <li key={id}>依据 · {candidate?.observation.site.toUpperCase()} {candidate?.observation.asin} · {fact?factLabel(fact.path):"未知字段"}：{JSON.stringify(fact?.value)}</li>;})}</ul>;}
