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

it("forwards a single canonical candidate selector without accepting caller facts",async()=>{
 const path=["supply-preparations","records",record,"materials-agent","prepare"];
 const send=(candidateCategoryId:unknown)=>new Request("http://localhost/api/workbench/"+path.join("/"),{method:"POST",headers:{...scope,Origin:"http://localhost","Content-Type":"application/json"},body:JSON.stringify({requestKey:key,expectedModeRevision:"1",candidateCategoryId})});
 const accepted=await buildWorkbenchUpstreamRequest(send("123"),path,"token","actor-a");
 expect(accepted).not.toBeInstanceOf(Response);
 if(!(accepted instanceof Response))expect(JSON.parse(String(accepted.init.body)).candidateCategoryId).toBe("123");
 for(const bad of ["","0","01","1.0","9223372036854775808",123,null])expect(await buildWorkbenchUpstreamRequest(send(bad),path,"token","actor-a")).toBeInstanceOf(Response);
});

it("accepts recorded read-only rule results while rejecting write fields and wrong evidence",async()=>{
 const queryRules={schemaVersion:"shein-candidate-rules-v1",binding:{ContextKind:"listing-candidate-rules-v1",ContextID:record,ProductKey:"product",CatalogVersion:"1",PublicationID:"publication",TargetPlatform:"shein",CandidateRules:{BaseRevision:"1",BaseDigest:"a".repeat(64),StoreID:record,Site:"shein-us",MerchantDigest:"b".repeat(64),CandidateCategoryID:"123",SelectorDigest:"c".repeat(64)}},sourceId:record,originalPublicationId:"publication",originalVersion:"1",applyReceiptId:"",evidenceId:"d".repeat(64),evidence:{tool:{ID:"listing.materials.rules.query",Version:"v1.0.0"},callId:"query-call",ruleDigest:"e".repeat(64),readAt:"2026-11-03T00:00:00Z",algorithmVersion:"shein-candidate-rules-v1",officialVersionStatus:"unknown"},categoryPath:[{id:"123",name:"候选类目",productTypeId:"456"}],fill:{fill_in_standard_list:[{field_key:"name",module:"product",required:null,show:true}],currency:null,default_language:"en",default_language_title_max_length:null,language_title_max_length_list:[],picture_config_list:[],supplier_code_in_spu_dimension:null,support_duplicate_sale_attr_value:null},attributes:{product_type_id:456,main_attribute_status:null,attribute_infos:[]},coverage:[{section:"category",status:"available",reason:"实际叶子"},{section:"fill",status:"unknown",reason:"标志未知"},{section:"attributes",status:"unknown",reason:"标志未知"},{section:"images",status:"unknown",reason:"未取得"},{section:"conditional",status:"not_evaluated",reason:"尚未提供具体属性值"}],unknowns:["条件规则未评价"]};
 const result={purpose:"category-rules",candidateCategoryId:"123",runId:key,requestKey:key,recordId:record,phase:"human_review_required",revision:"2",configurationSnapshotRef:ref,modeSnapshotRef:{Kind:"agent-execution-mode-v1",ID:key,Digest:"b".repeat(64)},executionMode:"full_access",modeRevision:"1",queryRules,ruleExplanation:{schemaVersion:"listing-rule-explanation-v1",summary:"请核对基础模板，具体关联条件未评价。",references:[{evidenceId:queryRules.evidenceId,callId:"query-call",ruleDigest:queryRules.evidence.ruleDigest}],unknowns:[]},unresolved:["未评价"],canRequestAdoption:false,canResume:false,steps:[{step:1,tool:"listing.materials.rules.query",callId:"query-call",auditStatus:"recorded"}],tokens:30,estimatedCostMicros:20,currency:"CNY",usageStatus:"observed"};
 expect(materialsRunSchema.safeParse(result).success).toBe(true);
 // The existing runtime returns human_review_required + repair_limit when
 // deterministic validation exhausts its bounded repair attempts.
 const failed={...result,stopReason:"repair_limit",ruleExplanation:undefined};
 expect(materialsRunSchema.safeParse(failed).success).toBe(true);
 for(const contract of ["supply-materials-start","supply-materials-read"] as const){
  const response=await buildWorkbenchBrowserResponse(Response.json(failed),contract,record+":"+key,{sourceMutation:contract.endsWith("start")});
  expect(response.status).toBe(200);expect(await response.json()).toMatchObject({stopReason:"repair_limit",usageStatus:"observed"});
 }
 expect(materialsRunSchema.safeParse({...failed,ruleExplanation:result.ruleExplanation}).success).toBe(false);
 expect(materialsRunSchema.safeParse({...result,ruleExplanation:undefined}).success).toBe(false);
 for(const extra of [{canRequestAdoption:true},{candidateHash:"f".repeat(64)},{proposal:{brandCode:"bad",evidenceIds:["observed"]}},{adoptionReadUrl:"/adopt"},{before:{}},{ruleExplanation:{...result.ruleExplanation,references:[{...result.ruleExplanation.references[0],callId:"unrecorded"}]}}])expect(materialsRunSchema.safeParse({...result,...extra}).success).toBe(false);
});
