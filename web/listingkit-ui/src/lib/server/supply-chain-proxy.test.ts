import { beforeEach, describe, expect, it, vi } from "vitest";
import { buildWorkbenchUpstreamRequest, buildWorkbenchBrowserResponse } from "./workbench-proxy";
const id="550e8400-e29b-41d4-a716-446655440000";
const scope={cookie:"shuomi_effective_organization=org-a","X-Expected-Organization-ID":"org-a","X-Expected-User-ID":"actor-a"};
describe("private supply chain proxy",()=>{
 beforeEach(()=>vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","http://localhost"));
 it("keeps official status reads bound to source/store and accepts no browser product selector",async()=>{
  const path=["supply-preparations","sources",id,"publications",id,"status"],url=`http://localhost/api/workbench/${path.join("/")}`;
  const read=await buildWorkbenchUpstreamRequest(new Request(url,{headers:scope}),path,"server-token","actor-a");
  expect(read).not.toBeInstanceOf(Response);if(read instanceof Response)return;
  expect(read.sourceMutation).toBe(false);expect(read.expectedStoreId).toBe(`${id}:${id}`);
  expect(new Headers(read.init.headers).get("X-Requested-Organization-ID")).toBe("org-a");
  const selector=await buildWorkbenchUpstreamRequest(new Request(url+"?productId=other",{headers:scope}),path,"server-token","actor-a");expect(selector).toBeInstanceOf(Response);if(selector instanceof Response)expect(selector.status).toBe(400);
  const wrongActor=await buildWorkbenchUpstreamRequest(new Request(url,{headers:scope}),path,"server-token","other");expect(wrongActor).toBeInstanceOf(Response);if(wrongActor instanceof Response)expect(wrongActor.status).toBe(409);
 });
 it("binds unbound text saves to the same verified actor and forbids Store or authority fields",async()=>{
  const input={sourceId:id,expectedRevision:0,content:{title:"",description:"真实待补全内容"}};
  const path=["supply-preparations","completion-drafts"];
  const request=(body:string)=>new Request("http://localhost/api/workbench/supply-preparations/completion-drafts",{method:"POST",headers:{...scope,Origin:"http://localhost","Content-Type":"application/json","Idempotency-Key":id},body});
  const accepted=await buildWorkbenchUpstreamRequest(request(JSON.stringify(input)),path,"server-token","actor-a");expect(accepted).not.toBeInstanceOf(Response);if(accepted instanceof Response)return;expect(accepted.sourceMutation).toBe(true);expect(accepted.url).toMatch(/completion-drafts$/);
  for(const body of [JSON.stringify({...input,storeId:id}),JSON.stringify(input).replace('"title":""','"title":"","title":"other"'),JSON.stringify({...input,content:{title:"",description:"a".repeat(70*1024)}})]){
   const rejected=await buildWorkbenchUpstreamRequest(request(body),path,"server-token","actor-a");expect(rejected).toBeInstanceOf(Response);if(rejected instanceof Response)expect(rejected.status).toBeGreaterThanOrEqual(400);
  }
  const wrongActor=await buildWorkbenchUpstreamRequest(request(JSON.stringify(input)),path,"server-token","other");expect(wrongActor).toBeInstanceOf(Response);if(wrongActor instanceof Response)expect(wrongActor.status).toBe(409);
 });
 it("allows bounded draft lists and rejects records from another preparation",async()=>{
  const path=["supply-preparations",id,"completion-drafts"];
  const read=await buildWorkbenchUpstreamRequest(new Request(`http://localhost/api/workbench/${path.join("/")}?limit=20&keyword=test`,{headers:scope}),path,"server-token","actor-a");expect(read).not.toBeInstanceOf(Response);
  const other="550e8400-e29b-41d4-a716-446655440001";
  const record={id,draftId:id,revision:1,source:{id,preparationId:other,collectionItemId:id,collectionRevision:1,source:{productKey:"product",publicationId:id,version:"1",kind:"own"}},content:{title:"saved",description:""},status:"pending_completion",createdAt:"2026-10-12T00:00:00Z"};
  const rejected=await buildWorkbenchBrowserResponse(Response.json({items:[record],total:1}),"supply-completion-drafts",id);expect(rejected.status).toBe(502);
 });
 it("binds a full batch transfer to the verified original actor",async()=>{
  const request=new Request("http://localhost/api/workbench/supply-preparations/transfer",{method:"POST",headers:{...scope,Origin:"http://localhost","Content-Type":"application/json","Idempotency-Key":id},body:JSON.stringify({batchId:id,expectedRevision:1})});
  const result=await buildWorkbenchUpstreamRequest(request,["supply-preparations","transfer"],"server-token","actor-a");
  expect(result).not.toBeInstanceOf(Response);if(result instanceof Response)return;
  expect(result.url).toMatch(/\/api\/v1\/workbench\/supply-preparations\/transfer$/);expect(result.sourceMutation).toBe(true);
  expect(new Headers(result.init.headers).get("X-Requested-Organization-ID")).toBe("org-a");
 });
 it("strips private material from a retained transfer receipt",async()=>{
  const upstream=Response.json({preparation:{id,sourceBatchId:id,sourceRevision:1,name:"测试批次",count:205,revision:1,createdAt:"2026-10-08T00:00:00Z",owner:"private",credential:"private"},replayed:true,credential:"private"});
  const response=await buildWorkbenchBrowserResponse(upstream,"supply-transfer",undefined,{sourceMutation:true});
  expect(response.status).toBe(200);expect(await response.text()).not.toContain("private");
 });
 it("rejects a different expected actor before any upload",async()=>{
  const request=new Request(`http://localhost/api/workbench/supply-preparations/operations/${id}/ensure`,{method:"POST",headers:{...scope,"X-Expected-User-ID":"other",Origin:"http://localhost"}});
  const result=await buildWorkbenchUpstreamRequest(request,["supply-preparations","operations",id,"ensure"],"server-token","actor-a");expect(result).toBeInstanceOf(Response);if(result instanceof Response)expect(result.status).toBe(409);
 });
 it("binds official resolution to the current actor and rejects browser evidence",async()=>{
  const path=["supply-preparations","uploads","resolve"];
  const input={recordId:id,attemptId:id,spu:"spu-a"};
  const request=(body:unknown,origin="http://localhost")=>new Request("http://localhost/api/workbench/supply-preparations/uploads/resolve",{method:"POST",headers:{...scope,Origin:origin,"Content-Type":"application/json"},body:JSON.stringify(body)});
  const safe=await buildWorkbenchUpstreamRequest(request(input),path,"server-token","actor-a");expect(safe).not.toBeInstanceOf(Response);if(safe instanceof Response)return;expect(safe.sourceMutation).toBe(true);expect(new Headers(safe.init.headers).get("Idempotency-Key")).toBeNull();
  for(const r of [request({...input,evidence:{outcome:"succeeded"}}),request(input,"https://another.example")]){const rejected=await buildWorkbenchUpstreamRequest(r,path,"server-token","actor-a");expect(rejected).toBeInstanceOf(Response);if(rejected instanceof Response)expect(rejected.status).toBeGreaterThanOrEqual(400)}
 });
 it("forwards only a bounded canonical source filter",async()=>{
  const path=["supply-preparations",id,"stages"];
  const result=await buildWorkbenchUpstreamRequest(new Request(`http://localhost/api/workbench/supply-preparations/${id}/stages?storeId=${id}&stage=review&sourceKind=own`,{headers:scope}),path,"server-token","actor-a");expect(result).not.toBeInstanceOf(Response);if(!(result instanceof Response))expect(result.url).toContain("sourceKind=own");
  for(const filter of ["sourceKind=fake","sourceKind=own&sourceKind=acquisition"]){const rejected=await buildWorkbenchUpstreamRequest(new Request(`http://localhost/api/workbench/supply-preparations/${id}/stages?storeId=${id}&stage=review&${filter}`,{headers:scope}),path,"server-token","actor-a");expect(rejected).toBeInstanceOf(Response);if(rejected instanceof Response)expect(rejected.status).toBe(400);}
 });
 it("bounds whole-batch stage queries before dispatch",async()=>{
  for(const query of [`storeId=${id}&stage=ready&stage=review`,`storeId=${id}&stage=fake`,`storeId=${id}&stage=ready&limit=101`]){
   const result=await buildWorkbenchUpstreamRequest(new Request(`http://localhost/api/workbench/supply-preparations/${id}/stages?${query}`,{headers:scope}),["supply-preparations",id,"stages"],"server-token","actor-a");expect(result).toBeInstanceOf(Response);if(result instanceof Response)expect(result.status).toBe(400);
  }
 });
});
