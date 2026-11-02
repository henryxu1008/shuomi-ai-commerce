import {afterEach,expect,it,vi} from "vitest";
import {prepareMaterials,startMaterials} from "./listing-materials";

const record="11111111-1111-4111-8111-111111111111",key="22222222-2222-4222-8222-222222222222";
const scope={userId:"actor",organizationId:"org"};
const prepared={recordId:record,requestKey:key,configurationSnapshotRef:{Kind:"agent-configuration-v1" as const,ID:key,Digest:"a".repeat(64)},modeSnapshotRef:{Kind:"agent-execution-mode-v1" as const,ID:key,Digest:"b".repeat(64)},modeRevision:"1",templateSelection:null,purpose:"category-rules" as const,candidateCategoryId:"123"};
const reply=(value:unknown)=>Response.json(value);
afterEach(()=>vi.unstubAllGlobals());

it("binds preparation to the retained public purpose and candidate",async()=>{
 const fetcher=vi.fn().mockResolvedValue(reply(prepared));vi.stubGlobal("fetch",fetcher);
 const input={requestKey:key,expectedModeRevision:"1",candidateCategoryId:"123"};
 await expect(prepareMaterials(scope,record,input,new AbortController().signal)).resolves.toEqual(prepared);
 fetcher.mockResolvedValue(reply({...prepared,candidateCategoryId:"124"}));
 await expect(prepareMaterials(scope,record,input,new AbortController().signal)).rejects.toMatchObject({code:"OUTCOME_UNKNOWN"});
 fetcher.mockResolvedValue(reply({...prepared,purpose:"materials-edit",candidateCategoryId:undefined}));
 await expect(prepareMaterials(scope,record,input,new AbortController().signal)).rejects.toMatchObject({code:"OUTCOME_UNKNOWN"});
});

it("does not accept a valid run for a different query selector",async()=>{
 const run={...prepared,templateSelection:undefined,runId:key,phase:"stopped",revision:"1",executionMode:"default",unresolved:[],canRequestAdoption:false,steps:[],tokens:0,estimatedCostMicros:0,currency:"CNY",usageStatus:"observed"};
 const fetcher=vi.fn().mockResolvedValue(reply(run));vi.stubGlobal("fetch",fetcher);
 await expect(startMaterials(scope,record,key,prepared,new AbortController().signal)).resolves.toMatchObject({candidateCategoryId:"123"});
 fetcher.mockResolvedValue(reply({...run,candidateCategoryId:"124"}));
 await expect(startMaterials(scope,record,key,prepared,new AbortController().signal)).rejects.toMatchObject({code:"OUTCOME_UNKNOWN"});
 expect(fetcher.mock.calls).toHaveLength(2);
});
