import {beforeEach,expect,it,vi} from "vitest";
import {buildWorkbenchUpstreamRequest,buildWorkbenchBrowserResponse} from "./workbench-proxy";
import {materialsRunSchema} from "../contracts/supply-chain";
import {initialSupplyDraft} from "../../components/workbench/supply/draft-fields";
const record="550e8400-e29b-41d4-a716-446655440000",key="550e8400-e29b-41d4-a716-446655440001";
const ref={Kind:"agent-configuration-v1",ID:key,Digest:"a".repeat(64)};
const scope={cookie:"shuomi_effective_organization=org-a","X-Expected-Organization-ID":"org-a","X-Expected-User-ID":"actor-a"};
beforeEach(()=>vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","http://localhost"));
it("reads owner-valid material attributes through the proposal and saved draft",()=>{
 const attributes=Array.from({length:1000},(_,index)=>({attribute_id:index+1,attribute_value_id:1}));
 const draft=initialSupplyDraft({product:{}} as Parameters<typeof initialSupplyDraft>[0]);
 draft.product.product_attribute_list=attributes;
 const input={sourceId:record,storeId:record,expectedRevision:0,effectiveVersion:"1",draft};
 const result={runId:key,requestKey:key,recordId:record,phase:"human_review_required",revision:"2",configurationSnapshotRef:ref,modeSnapshotRef:{Kind:"agent-execution-mode-v1",ID:key,Digest:"b".repeat(64)},executionMode:"default",modeRevision:"1",before:input,after:input,proposal:{attributes,evidenceIds:["observed"]},candidateHash:"c".repeat(64),unresolved:[],canRequestAdoption:true,steps:[],tokens:30,estimatedCostMicros:20,currency:"CNY",usageStatus:"observed"};
 expect(materialsRunSchema.safeParse(result).success).toBe(true);
 expect(materialsRunSchema.safeParse({...result,proposal:{...result.proposal,attributes:[...attributes,attributes[0]]}}).success).toBe(false);
});
it("forwards only the original materials configuration and refuses caller authority/candidate",async()=>{
 const path=["supply-preparations","records",record,"materials-agent","runs"];
 const request=(body:string,origin="http://localhost")=>new Request("http://localhost/api/workbench/"+path.join("/"),{method:"POST",headers:{...scope,Origin:origin,"Content-Type":"application/json","Idempotency-Key":key},body});
 const accepted=await buildWorkbenchUpstreamRequest(request(JSON.stringify({configurationSnapshotRef:ref})),path,"token","actor-a");
 expect(accepted).not.toBeInstanceOf(Response);if(accepted instanceof Response)return;
 expect(accepted.sourceMutation).toBe(true);expect(accepted.url).toMatch(/materials-agent\/runs$/);
 expect(new Headers(accepted.init.headers).get("Idempotency-Key")).toBe(key);
 expect(accepted.expectedStoreId).toBe(record+":"+key);
 for(const body of [JSON.stringify({configurationSnapshotRef:ref,candidate:{title:"bad"}}),JSON.stringify({configurationSnapshotRef:ref,organizationId:"other"}),`{"configurationSnapshotRef":${JSON.stringify(ref)},"configurationSnapshotRef":${JSON.stringify(ref)}}`]){
  const denied=await buildWorkbenchUpstreamRequest(request(body),path,"token","actor-a");expect(denied).toBeInstanceOf(Response);if(denied instanceof Response)expect(denied.status).toBe(400);
 }
 expect(await buildWorkbenchUpstreamRequest(request(JSON.stringify({configurationSnapshotRef:ref}),"https://other.example"),path,"token","actor-a")).toBeInstanceOf(Response);
});
it("preserves human/UNKNOWN semantics and accepts only empty original-key reads",async()=>{
 const path=["supply-preparations","records",record,"materials-agent","runs",key,"adoption"];
 const request=new Request("http://localhost/api/workbench/"+path.join("/"),{headers:scope});
 const accepted=await buildWorkbenchUpstreamRequest(request,path,"token","actor-a");expect(accepted).not.toBeInstanceOf(Response);
 for(const code of ["HUMAN_CONFIRMATION_REQUIRED","OUTCOME_UNKNOWN","AGENT_CONFLICT"]){
  const response=await buildWorkbenchBrowserResponse(Response.json({code},{status:409}),"supply-materials-adopt",record+":"+key,{sourceMutation:true});expect(response.status).toBe(409);expect((await response.json()).code).toBe(code);
 }
 const query=await buildWorkbenchUpstreamRequest(new Request(request.url+"?key=other",{headers:scope}),path,"token","actor-a");expect(query).toBeInstanceOf(Response);
 if(query instanceof Response)expect(query.status).toBe(400);
});
