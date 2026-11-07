import {describe, expect, it} from "vitest";
import {createResearchHandoff, draftMatchesJob, opportunityPriceRange, claimResearchTransfer, advanceHandoffRevision, specificationFacts} from "./research-handoff";
import {parseSupplyIntent} from "../api/supply-intent";
import type {DataJob} from "../contracts/data-services";
import type {ResearchInspection} from "../contracts/research-analysis";

const A = "92000000-0000-4000-8000-000000000001";
const B = "92000000-0000-4000-8000-000000000002";
const batchId = "92000000-0000-4000-8000-000000000003";
const job = {id:A,batchId,saved:2,state:"SUCCEEDED",query:{site:"us",mode:"keyword",keyword:"under sink organizer",limit:5,fields:["attributes","price","currency"]}} satisfies Partial<DataJob>;
const nextJob = {...job,id:B,query:{...job.query,keyword:"under sink mat"}} satisfies Partial<DataJob>;
const requirements = {use:"水槽下收纳", targetPrice:"", missing:"", specs:[{sourceKey:"dimension",required:"可调宽",difference:"最小 40 cm"}]};
const choices = [A,B].map(candidateId=>({candidateId,expectedRevision:"1",linkId:candidateId,evidenceDigest:"a".repeat(64),selection:{itemId:candidateId,expectedRevision:"2",originalPublicationId:candidateId,originalVersion:"3"}}));
const confirmed = {requirements, choices, representative:A, after:"", confirmedAt:"2026-11-06T12:00:00Z"};
const price = (candidateId:string, amount:number) => ({research:{CandidateID:candidateId,CandidateRevision:"1",SelectedLinkID:candidateId,EvidenceDigest:"a".repeat(64)},ref:{ItemID:candidateId,ItemRevision:"2",PublicationID:candidateId,CatalogVersion:"3",ProductKey:candidateId},observation:{price:amount,currency:"USD",priceBasis:"displayed_listing_offer"},facts:[{path:"attributes.0",id:"dimension",value:{Name:"宽度",Value:"40 cm"}}]}) as ResearchInspection["candidates"][number];

describe("research opportunity handoff",()=>{
  it("binds a specification to its unchanged evidence across a candidate revision",()=>{
    const row = price(A,29.99);
    const newer = {...row,research:{...row.research,CandidateRevision:"2"},facts:[{...row.facts[0],id:"new-fact-id"}]};
    const key = specificationFacts(row)[0].sourceKey;
    expect(key).toBeTruthy();
    expect(specificationFacts(newer)[0].sourceKey).toBe(key);
    expect(specificationFacts({...newer,ref:{...newer.ref,CatalogVersion:"4"}})[0].sourceKey).not.toBe(key);
    expect(specificationFacts({...newer,facts:[{...newer.facts[0],value:{Name:"宽度",Value:"42 cm"}}]})[0].sourceKey).not.toBe(key);
  });
  it("W1 requires two observed comparable samples and claims exactly the approved item once",()=>{
    const store=createResearchHandoff();
    const selected=store.start("member",A,job as DataJob);
    expect(opportunityPriceRange([price(A,29.99),price(B,24.99)],A)).toEqual({currency:"USD",minimum:24.99,maximum:29.99});
    const ready={...selected,confirmed:{...confirmed,approvalRevision:1},transfer:{key:B,revision:4}};
    expect(store.replace(selected,ready)).toBe(true);
    const dispatched=claimResearchTransfer(store,ready,{batchId,expectedRevision:4,itemIds:[A]});
    expect(dispatched).toBe(B);
    expect(claimResearchTransfer(store,ready,{batchId,expectedRevision:4,itemIds:[A]})).toBeNull();
  });

  it("W2 rejects old work after choosing a different query and checks the exact job and batch",()=>{
    const store=createResearchHandoff();
    const old=store.start("member",A,job as DataJob);
    const current=store.start("member",A,nextJob as DataJob);
    expect(store.replace(old,{...old,confirmed})).toBe(false);
    expect(store.getSnapshot()).toBe(current);
    expect(draftMatchesJob(job as DataJob,{...job.query,asinCount:0},A,batchId)).toBe(true);
    expect(draftMatchesJob(nextJob as DataJob,{...job.query,asinCount:0},A,batchId)).toBe(false);
    expect(draftMatchesJob(job as DataJob,{...job.query,asinCount:0},A,B)).toBe(false);
  });

  it("W3 gives no comparable range when a sample has only a title",()=>{
    const onlyTitle={...price(B,15),observation:{},facts:[{id:"title",path:"title",value:"Organizer"}]} as unknown as ResearchInspection["candidates"][number];
    expect(opportunityPriceRange([onlyTitle],B)).toBeNull();
    expect(opportunityPriceRange([price(A,24.99),onlyTitle],A)).toBeNull();
  });

  it("advances only the representative's matching revision from its own receipt",()=>{
    const store=createResearchHandoff();
    const draft=store.start("member",A,job as DataJob);
    const ready={...draft,confirmed};
    expect(advanceHandoffRevision(ready,B,1,2)).toBe(ready);
    expect(advanceHandoffRevision(ready,A,2,3)).toBe(ready);
    expect(advanceHandoffRevision(ready,A,1,2).confirmed?.approvalRevision).toBeUndefined();
    const approved=advanceHandoffRevision(ready,A,1,2,true);
    expect(approved.confirmed?.choices[0].expectedRevision).toBe("2");
    expect(approved.confirmed?.approvalRevision).toBe(2);
  });

  it("retains the transfer's lookup-only mode across strict intent parsing and storage reload",()=>{
    const command={batchId,expectedRevision:4,itemIds:[A]};
    const intent={userId:"user",organizationId:"org",key:B,route:"transfer",command,lookupOnly:true};
    expect(parseSupplyIntent(JSON.stringify(intent))).toEqual(intent);
    expect(parseSupplyIntent(JSON.stringify({...intent,lookupOnly:false}))).toBeNull();
    expect(parseSupplyIntent(JSON.stringify({...intent,command:{...command,itemIds:[A,B]}}))).toBeNull();
  });
});
