"use client";

import Link from "next/link";
import {useRouter} from "next/navigation";
import {useEffect,useRef,useState} from "react";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {dataRequest} from "@/lib/api/data-services";
import {researchRequest} from "@/lib/api/market-research";
import {readCollectionBatch} from "@/lib/api/product-collection";
import type {ProjectScope} from "@/lib/api/project-center";
import {verifyResearchHandoff} from "@/lib/api/research-handoff";
import {jobSchema} from "@/lib/contracts/data-services";
import {researchFixtureStatus,type ResearchCandidate} from "@/lib/contracts/market-research";
import {researchAnalysisInput,type ResearchInspection} from "@/lib/contracts/research-analysis";
import {candidateMatchesJob,eligibleResearchJob,handoffToken,opportunityPriceRange,researchFixtureJob,specificationFacts,type ProductRequirements,type ResearchHandoff,type ResearchHandoffStore,type ResearchJob} from "@/lib/workbench/research-handoff";
import {analysisChoices} from "./research-analysis-flow";

const emptyRequirements:ProductRequirements={use:"",specs:[],targetPrice:"",missing:""};

function queryLabel(job:ResearchJob) {
  const query=job.query;
  if (query.mode==="category") return "类目 "+query.categoryNode;
  return "关键词 "+query.keyword+(query.categoryNode?" · 类目 "+query.categoryNode:"");
}

export function ResearchOpportunity({scope,scopeKey,projectId,store,draft,selected,after,blocked,canStartQuery=false,showResearchFixture=false,onJobChange}:{
  scope:ProjectScope; scopeKey:string; projectId:string; store:ResearchHandoffStore; draft:ResearchHandoff|null;
  selected:ResearchCandidate[]; after:string; blocked:boolean; canStartQuery?:boolean; showResearchFixture?:boolean; onJobChange:()=>void;
}) {
  const [jobs,setJobs]=useState<ResearchJob[]>([]);
  const [canImport,setCanImport]=useState(false);
  const [importing,setImporting]=useState(false);
  const [importError,setImportError]=useState("");
  const importRequest=useRef<AbortController|null>(null);
  const [readStatus,setReadStatus]=useState<"loading"|"ready"|"error">("loading");
  const [refresh,setRefresh]=useState(0);
  useEffect(()=>{
    const controller=new AbortController();
    const fixture=showResearchFixture?researchRequest(scope,projectId,researchFixtureStatus,controller.signal,{fixture:"read"}):Promise.resolve(null);
    Promise.all([dataRequest(scope,"amazon/jobs",z.array(jobSchema).max(100),undefined,controller.signal),fixture])
      .then(([value,fixtureStatus])=>{
        if (controller.signal.aborted) return;
        const available:ResearchJob[]=value.filter(eligibleResearchJob);
        if(fixtureStatus?.ready)available.unshift(researchFixtureJob(fixtureStatus.batch));
        setCanImport(fixtureStatus?.ready===false);
        setJobs(available);
        setReadStatus("ready");
      }).catch(()=>{if (!controller.signal.aborted) setReadStatus("error");});
    return ()=>controller.abort();
  },[scope,projectId,refresh,showResearchFixture]);
  useEffect(()=>()=>importRequest.current?.abort(),[]);
  async function importFixture(){
    if(importRequest.current||blocked)return;
    const controller=new AbortController();
    importRequest.current=controller;
    setImporting(true);
    setImportError("");
    try{
      await researchRequest(scope,projectId,researchFixtureStatus,controller.signal,{fixture:"import"});
      if(!controller.signal.aborted)setRefresh(value=>value+1);
    }catch{
      if(!controller.signal.aborted)setImportError("暂时无法确认导入结果，请刷新查看。");
    }finally{
      importRequest.current=null;
      if(!controller.signal.aborted)setImporting(false);
    }
  }
  const current=draft?.projectId===projectId && draft.scope===scopeKey?draft:null;
  return <Card className="space-y-3">
    <h3>从已保存的样本整理选品方向</h3>
    <p>选择一批 Amazon 美国站关键词或类目查询的结果，再在下方勾选同批的 2–5 件原版商品。这里由你整理方向。</p>
    <div className="flex flex-wrap gap-2">
      {canStartQuery?<Button asChild variant="outline"><Link href="/workbench/data/market" prefetch={false}>前往数据市场</Link></Button>:null}
      {canImport?<Button variant="outline" disabled={blocked||importing} onClick={()=>void importFixture()}>导入合成验收资料</Button>:null}
      <Button variant="outline" onClick={()=>{setReadStatus("loading");setRefresh(value=>value+1);}}>刷新查询结果</Button>
    </div>
    {importError?<p role="alert">{importError}</p>:null}
    {readStatus==="loading"?<p role="status">正在读取已保存的查询…</p>:readStatus==="error"?<p role="alert">暂时无法读取已保存的查询。</p>:null}
    <label className="grid gap-1">本次查询
      <select className="rounded border bg-background p-2" disabled={readStatus!=="ready" || blocked}
        value={current?.job.id??""} onChange={event=>{
          const job=jobs.find(value=>value.id===event.target.value);
          if (job) store.start(scopeKey,projectId,job);
          else store.clear();
          onJobChange();
        }}>
        <option value="">选择已保存的查询</option>
        {jobs.map(job=><option key={job.id} value={job.id}>{job.state==="IMPORTED"?"合成样本 · ":""}{queryLabel(job)} · {job.state==="IMPORTED"?"已导入":new Date(job.createdAt).toLocaleString()} · {job.saved} 件</option>)}
      </select>
    </label>
    {readStatus==="ready" && !jobs.length?<p>暂无可用的已保存样本。</p>:null}
    {current && <OpportunityEditor key={current.job.id} scope={scope} store={store} draft={current}
      selected={selected} after={after} blocked={blocked || readStatus!=="ready" || !jobs.some(job=>job.id===current.job.id)}/>}
  </Card>;
}

function OpportunityEditor({scope,store,draft,selected,after,blocked}:{
  scope:ProjectScope; store:ResearchHandoffStore; draft:ResearchHandoff; selected:ResearchCandidate[]; after:string; blocked:boolean;
}) {
  const router=useRouter();
  const [inspected,setInspected]=useState<{signature:string;inspection:ResearchInspection}|null>(null);
  const [requirements,setRequirements]=useState<ProductRequirements>(emptyRequirements);
  const [representative,setRepresentative]=useState("");
  const [busyFor,setBusyFor]=useState("");
  const busy=busyFor===draft.id;
  const [error,setError]=useState("");
  const read=useRef<AbortController|null>(null);
  useEffect(()=>()=>{read.current?.abort();read.current=null;},[draft.id]);
  const input=researchAnalysisInput.safeParse({candidates:analysisChoices(selected)});
  const signature=input.success && selected.every(candidate=>candidateMatchesJob(candidate,draft.job))?
    JSON.stringify([draft.job.id,after,input.data]):"";
  const inspection=signature && inspected?.signature===signature?inspected.inspection:null;
  const range=inspection && opportunityPriceRange(inspection.candidates,representative);
  const facts=inspection?.candidates.flatMap(row=>specificationFacts(row).map(spec=>({...spec,asin:row.observation.asin})))??[];
  const missing=selected.flatMap(candidate=>candidate.observation?.missing??[]);
  function edit(next:ProductRequirements) {
    if (draft.confirmed) store.start(draft.scope,draft.projectId,draft.job);
    setRequirements(next);
  }
  async function check(confirm:boolean) {
    if (blocked || read.current || !signature || !input.success) return;
    const base=store.getSnapshot();
    if (!base || base!==draft || !base.job.batchId) return;
    const controller=new AbortController();
    read.current=controller;
    setBusyFor(draft.id);
    setError("");
    const next={...base,confirmed:{requirements,choices:input.data.candidates,representative,after,confirmedAt:new Date().toISOString()}};
    try {
      const [{inspection:result},batch]=await Promise.all([
        verifyResearchHandoff(scope,{...next,confirmed:{...next.confirmed,requirements:confirm?requirements:{...requirements,specs:[]},representative:representative||input.data.candidates[0].candidateId}},controller.signal),
        readCollectionBatch(scope,base.job.batchId,AbortSignal.any([controller.signal,AbortSignal.timeout(10_000)])),
      ]);
      if (controller.signal.aborted || store.getSnapshot()!==base) return;
      if (batch.id!==base.job.batchId || batch.archivedAt) throw new Error("Batch changed");
      setInspected({signature,inspection:result});
      if (confirm) {
        if (!opportunityPriceRange(result.candidates,representative) || !requirements.use.trim() || !requirements.specs.length ||
            requirements.specs.some(spec=>!spec.required.trim())) throw new Error("Insufficient evidence");
        if (store.replace(base,next)) router.push("/workbench/supply/selection/"+base.projectId+"?handoff="+handoffToken(next));
      }
    } catch {
      if (!controller.signal.aborted && store.getSnapshot()===base) {
        if (confirm && base.confirmed) store.start(base.scope,base.projectId,base.job);
        setInspected(null);
        setError("暂时无法确认这批原件或商品要求。请核对权限、所选原版与缺项。");
      }
    } finally {
      if (read.current===controller) {
        read.current=null;
        setBusyFor("");
      }
    }
  }
  return <div className="space-y-3">
    <p>本批已保存 {draft.job.saved} 件，当前选择 {selected.length} 件。</p>
    {!signature?<p>请在本项目选择同批 2–5 件可读的原版商品。已有同款时，先在资料记录中点「使用这份资料」切换到本批原件。</p>:null}
    {!signature && selected.length>0 && missing.length>0?<p>待补证 · 缺少 {[...new Set(missing)].join("、")}</p>:null}
    <Button disabled={blocked||busy||!signature} variant="outline" onClick={()=>void check(false)}>核对所选原件</Button>
    {error?<p role="alert">{error}</p>:null}
    {inspection?<fieldset disabled={blocked||busy} className="space-y-3">
      <legend>人工整理 · Amazon 美国站</legend>
      {inspection.candidates.map(row=><div key={row.research.CandidateID}>
        <strong>{row.observation.asin}</strong> · {row.observation.price&&row.observation.currency?
          row.observation.currency+" "+row.observation.price:"缺可比售价"}
        <ul>{specificationFacts(row).map(fact=><li key={fact.sourceKey}>{fact.name}：{fact.value}</li>)}</ul>
        {row.missing.length?<p>缺项：{row.missing.join("、")}</p>:null}
      </div>)}
      <label className="grid gap-1">代表候选<select className="rounded border bg-background p-2" value={representative} onChange={event=>{
        if (draft.confirmed) store.start(draft.scope,draft.projectId,draft.job);
        setRepresentative(event.target.value);
      }}><option value="">请选择</option>{inspection.candidates.map(row=><option key={row.research.CandidateID} value={row.research.CandidateID}>{row.observation.asin}</option>)}</select></label>
      <p>{range?((draft.job.state==="IMPORTED"?"合成样本价 ":"样本展示价 ")+range.currency+" "+range.minimum.toFixed(2)+"–"+range.maximum.toFixed(2)):"需要两件具有同类规格和可比售价的样本，才可继续。"}</p>
      <label className="grid gap-1">商品用途<Input maxLength={200} value={requirements.use} onChange={event=>edit({...requirements,use:event.target.value})}/></label>
      {requirements.specs.map((spec,index)=>{
        const change=(next:typeof spec)=>edit({...requirements,specs:requirements.specs.map((item,i)=>i===index?next:item)});
        return <div key={index} className="space-y-2 rounded border p-3">
          <label className="grid gap-1">第 {index+1} 条规格依据<select className="rounded border bg-background p-2" value={spec.sourceKey} onChange={event=>change({...spec,sourceKey:event.target.value})}>
            <option value="">选择样本规格</option>{facts.map(fact=><option key={fact.sourceKey} value={fact.sourceKey}>{fact.asin} · {fact.name}：{fact.value}</option>)}
          </select></label>
          <label className="grid gap-1">必要规格<Input maxLength={200} value={spec.required} onChange={event=>change({...spec,required:event.target.value})}/></label>
          <label className="grid gap-1">可接受差异<Input maxLength={200} value={spec.difference} onChange={event=>change({...spec,difference:event.target.value})}/></label>
          <Button variant="ghost" onClick={()=>edit({...requirements,specs:requirements.specs.filter((_,i)=>i!==index)})}>移除此条规格</Button>
        </div>;
      })}
      <Button variant="outline" disabled={requirements.specs.length>=8 || !facts.length} onClick={()=>edit({...requirements,specs:[...requirements.specs,{sourceKey:"",required:"",difference:""}]})}>添加必要规格</Button>
      <label className="grid gap-1">目标售价（USD，可留空）<Input inputMode="decimal" maxLength={12} value={requirements.targetPrice} onChange={event=>edit({...requirements,targetPrice:event.target.value})}/></label>
      <label className="grid gap-1">待核实事项<Input maxLength={300} value={requirements.missing} onChange={event=>edit({...requirements,missing:event.target.value})}/></label>
      <Button disabled={!range||!requirements.use.trim()||!requirements.specs.length||
        requirements.specs.some(spec=>!spec.required.trim()||!facts.some(fact=>fact.sourceKey===spec.sourceKey))||
        (!!requirements.targetPrice&&(!/^\d+(\.\d{1,2})?$/.test(requirements.targetPrice)||Number(requirements.targetPrice)<=0))}
        onClick={()=>void check(true)}>确认并进入选品</Button>
    </fieldset>:null}
  </div>;
}

export function ConfirmedOpportunity({draft,inspection}:{draft:ResearchHandoff;inspection:ResearchInspection}) {
  const confirmed=draft.confirmed!;
  const {requirements,representative}=confirmed;
  const evidence=inspection.candidates.flatMap(row=>specificationFacts(row).map(fact=>({...fact,asin:row.observation.asin})));
  const range=opportunityPriceRange(inspection.candidates,representative);
  const missing=[...new Set(inspection.candidates.flatMap(row=>row.missing))];
  return <Card className="space-y-3">
    <h3>已确认的商品要求</h3>
    <p>{draft.job.state==="IMPORTED"?"合成样本 · ":""}Amazon 美国站 · {queryLabel(draft.job)} · {requirements.use}</p>
    {range?<p>{draft.job.state==="IMPORTED"?"合成样本价":"样本展示价"} {range.currency} {range.minimum.toFixed(2)}–{range.maximum.toFixed(2)}</p>:null}
    {requirements.targetPrice?<p>人工目标售价 USD {requirements.targetPrice}</p>:null}
    <ul>{requirements.specs.map((spec,index)=>{
      const fact=evidence.find(item=>item.sourceKey===spec.sourceKey);
      return <li key={index}>{spec.required}{spec.difference?" · 可接受："+spec.difference:""}
        {fact?" · 依据 "+fact.asin+" "+fact.name+"："+fact.value:""}</li>;
    })}</ul>
    <p>原版证据 {inspection.candidates.map(row=>row.observation.asin+(row.observation.fixtureVersion?" 合成参考 v"+row.observation.fixtureVersion+" · 本地 v":" v")+row.ref.CatalogVersion).join("、")}</p>
    {missing.length>0?<p>样本缺项：{missing.join("、")}</p>:null}
    {requirements.missing?<p>待核实：{requirements.missing}</p>:null}
  </Card>;
}
