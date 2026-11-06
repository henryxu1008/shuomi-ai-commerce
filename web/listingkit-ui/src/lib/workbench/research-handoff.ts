import type {z} from "zod";
import type {DataJob} from "../contracts/data-services";
import type {ResearchCandidate,ResearchFixtureBatch} from "../contracts/market-research";
import type {ResearchInspection, researchAnalysisInput} from "../contracts/research-analysis";

type Choice = z.infer<typeof researchAnalysisInput>["candidates"][number];
type Evidence = ResearchInspection["candidates"][number];
type Query = NonNullable<ResearchCandidate["observation"]>["query"];
export type ResearchJob = DataJob | Pick<DataJob,"id"|"batchId"|"query"|"saved"> & {state:"IMPORTED"};

export function researchFixtureJob(batch:ResearchFixtureBatch):ResearchJob {
  return {id:batch.id,batchId:batch.amazon[0].item.batchId,saved:2,state:"IMPORTED",
    query:{site:"us",mode:"keyword",keyword:batch.keyword,limit:5,fields:["attributes","price","currency"]}};
}

export type ProductRequirements = {
  use: string;
  specs: {sourceKey:string; required:string; difference:string}[];
  targetPrice: string;
  missing: string;
};
export type ResearchHandoff = {
  id: string;
  generation: number;
  scope: string;
  projectId: string;
  job: ResearchJob;
  confirmed?: {
    requirements: ProductRequirements;
    choices: Choice[];
    representative: string;
    after: string;
    confirmedAt: string;
    approvalRevision?:number;
  };
  transfer?: {key:string; revision:number; dispatched?:true};
};
export type ResearchTransferInput = {batchId:string; expectedRevision:number; itemIds?:string[]};

export function createResearchHandoff() {
  let current: ResearchHandoff | null = null;
  let generation = 0;
  const listeners = new Set<()=>void>();
  function set(value: ResearchHandoff | null) {
    current = value;
    listeners.forEach(listener=>listener());
  }
  return {
    subscribe(listener:()=>void) {
      listeners.add(listener);
      return ()=>{listeners.delete(listener)};
    },
    getSnapshot:()=>current,
    clear() { if (current) set(null); },
    start(scope:string, projectId:string, job:ResearchJob) {
      const next = {id:crypto.randomUUID(),generation:++generation,scope,projectId,job};
      set(next);
      return next;
    },
    replace(expected:ResearchHandoff, next:ResearchHandoff) {
      if (current !== expected) return false;
      set(next);
      return true;
    },
  };
}
export type ResearchHandoffStore = ReturnType<typeof createResearchHandoff>;
export const researchScopeKey = (user:string, organization:string, member:string, permissions:string[]) =>
  JSON.stringify([user,organization,member,[...permissions].sort()]);
export function researchHandoffAllowed(permissions:readonly string[], transfer=false) {
  return permissions.includes("workbench.project.read") &&
    permissions.includes("workbench.project.manage") &&
    permissions.includes("workbench.collection.read") &&
    (!transfer || permissions.includes("workbench.supply.manage"));
}

export const handoffToken = (draft:ResearchHandoff) => draft.id + "." + draft.generation;

export function eligibleResearchJob(job:ResearchJob) {
  const {query}=job;
  return job.saved>0 && !!job.batchId && ["SUCCEEDED","PARTIAL","CANCELED","IMPORTED"].includes(job.state) &&
    query.site==="us" && (query.mode==="keyword" && !!query.keyword || query.mode==="category" && !!query.categoryNode);
}
export function sameResearchJob(original:ResearchJob, current:ResearchJob) {
  return eligibleResearchJob(current) && current.id===original.id && current.batchId===original.batchId &&
    (current.state==="IMPORTED")===(original.state==="IMPORTED") && current.saved===original.saved && sameQuery(original.query,current.query);
}
function sameQuery(a:DataJob["query"],b:Pick<DataJob["query"],"mode"|"keyword"|"categoryNode"|"limit">&{fields?:readonly string[]}) {
  return a.mode===b.mode && a.keyword===b.keyword && a.categoryNode===b.categoryNode && a.limit===b.limit &&
    JSON.stringify([...(a.fields??[])].sort())===JSON.stringify([...(b.fields??[])].sort());
}
export function draftMatchesJob(job:ResearchJob, query:Query, jobId:string, batchId:string) {
  return eligibleResearchJob(job) && job.id===jobId && job.batchId===batchId && query.asinCount===0 && sameQuery(job.query,query);
}
export function candidateMatchesJob(candidate:ResearchCandidate, job:ResearchJob) {
  const o=candidate.observation;
  return !!o && !("shein" in o) && o.site==="us" && ("fixtureVersion" in o)===(job.state==="IMPORTED") && draftMatchesJob(job,o.query,o.ref.jobId,o.batchId);
}

export function specificationFacts(row:Evidence) {
  return row.facts.flatMap(fact=>{
    if (!/^attributes\.\d+$/.test(fact.path) || !fact.value || typeof fact.value!=="object" ||
        !("Name" in fact.value) || !("Value" in fact.value)) return [];
    const {Name,Value}=fact.value;
    if (typeof Name!=="string" || typeof Value!=="string" || !Name.trim() || !Value.trim() ||
        row.facts.some(other=>other.path===fact.path+".uncertainty")) return [];
    // Inspect's fact ID also binds the mutable candidate revision. The
    // handoff instead matches the exact selected original and fact value.
    const sourceKey=JSON.stringify([row.research.CandidateID,row.research.SelectedLinkID,row.research.EvidenceDigest,
      row.ref.ItemID,row.ref.ItemRevision,row.ref.PublicationID,row.ref.CatalogVersion,fact.path,Name,Value]);
    return [{sourceKey,name:Name,value:Value}];
  });
}
export function opportunityPriceRange(rows:Evidence[], representative:string) {
  const main=rows.find(row=>row.research.CandidateID===representative);
  if (!main) return null;
  const {price,currency,priceBasis}=main.observation;
  if (!price || currency!=="USD" || (priceBasis!=="displayed_listing_offer" && priceBasis!=="synthetic_listing_offer")) return null;
  const specNames=new Set(specificationFacts(main).map(spec=>spec.name.trim().toLowerCase()));
  const comparable=rows.filter(row=>row.observation.currency===currency &&
    row.observation.priceBasis===priceBasis && !!row.observation.price &&
    specificationFacts(row).some(spec=>specNames.has(spec.name.trim().toLowerCase())));
  if (comparable.length<2) return null;
  const prices=comparable.map(row=>row.observation.price!);
  return {currency,minimum:Math.min(...prices),maximum:Math.max(...prices)};
}

export function advanceHandoffRevision(draft:ResearchHandoff, candidateId:string, before:number, after:number, approved=false) {
  const c=draft.confirmed;
  const rep=c?.choices.find(choice=>choice.candidateId===candidateId);
  if (!c || draft.transfer?.dispatched || candidateId!==c.representative || rep?.expectedRevision!==String(before) || after!==before+1) return draft;
  return {...draft, transfer:undefined, confirmed:{...c,approvalRevision:approved?after:undefined,choices:c.choices.map(choice=>
    choice===rep?{...choice,expectedRevision:String(after)}:choice)}};
}

export function claimResearchTransfer(store:ResearchHandoffStore, draft:ResearchHandoff, input:ResearchTransferInput) {
  const approved=draft.confirmed?.choices.find(choice=>choice.candidateId===draft.confirmed?.representative);
  const transfer=draft.transfer;
  if (!approved || String(draft.confirmed?.approvalRevision)!==approved.expectedRevision || !transfer || transfer.dispatched || input.batchId!==draft.job.batchId ||
      input.expectedRevision!==transfer.revision || input.itemIds?.length!==1 ||
      input.itemIds[0]!==approved.selection.itemId) return null;
  return store.replace(draft,{...draft,transfer:{...transfer,dispatched:true}})?transfer.key:null;
}
