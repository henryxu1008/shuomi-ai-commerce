import { afterEach, expect, it, vi } from "vitest";
import { buildWorkbenchUpstreamRequest, buildWorkbenchBrowserResponse } from "./workbench-proxy";
const op = "11111111-1111-4111-8111-111111111111", key = "22222222-2222-4222-8222-222222222222";
const path = ["sourcing", "1688", "acquisitions", op, "product-agent", "runs"];
const contentPath = ["sourcing", "1688", "acquisitions", op, "content-agent", "runs"];
it("preserves a definite manual-only rejection without converting it to UNKNOWN",async()=>{
 const response=await buildWorkbenchBrowserResponse(Response.json({code:"MANUAL_REVIEW_REQUIRED"},{status:409}),"product-agent-content-result",key,{sourceMutation:true});
 expect(response.status).toBe(409);
 expect((await response.json()).code).toBe("MANUAL_REVIEW_REQUIRED");
 const uncertain=await buildWorkbenchBrowserResponse(Response.json({code:"UNRECOGNIZED"},{status:409}),"product-agent-content-result",key,{sourceMutation:true});
 expect(uncertain.status).toBe(503);expect((await uncertain.json()).code).toBe("OUTCOME_UNKNOWN");
});
afterEach(() => vi.unstubAllEnvs());
it("seals an own-product source in its own namespace and keeps mutation checks",async()=>{
 vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","http://localhost:3000");
 const ownPath=["collections","items",op,"content-agent","runs"];
 const payload={expectedModeRevision:"0",targetPlatform:"product",contentSelection:{targetLanguage:"zh-CN",title:true},ownSource:{catalogVersion:"2",publicationId:key}};
 const ownRequest=(body:unknown,extra:Record<string,string>={})=>new Request(`http://localhost:3000/api/workbench/${ownPath.join("/")}`,{method:"POST",headers:{...Object.fromEntries(req().headers),...extra},body:JSON.stringify(body)});
 const result=await buildWorkbenchUpstreamRequest(ownRequest(payload),ownPath,"token","actor");
 expect(result).not.toBeInstanceOf(Response);
 if(!(result instanceof Response)){expect(JSON.parse(result.init.body as string)).toEqual(payload);expect(result.sourceMutation).toBe(true)}
 for(const body of [{...payload,ownSource:undefined},{...payload,targetPlatform:"shein"},{...payload,ownSource:{...payload.ownSource,productKey:"forged"}}])expect(await buildWorkbenchUpstreamRequest(ownRequest(body),ownPath,"token","actor")).toBeInstanceOf(Response);
 expect(await buildWorkbenchUpstreamRequest(ownRequest(payload,{Origin:"http://evil.test"}),ownPath,"token","actor")).toBeInstanceOf(Response);
 expect(await buildWorkbenchUpstreamRequest(req(JSON.stringify(payload)),contentPath,"token","actor")).toBeInstanceOf(Response);
 const readPath=[...ownPath,key];
 const read=await buildWorkbenchUpstreamRequest(new Request(`http://localhost:3000/api/workbench/${readPath.join("/")}`,{headers:req().headers}),readPath,"token","actor");
 expect(read).not.toBeInstanceOf(Response);if(!(read instanceof Response))expect(read.expectedStoreId).toBe(key);
});
it("preserves readable content diagnostics while refusing an upstream adoption claim", async () => {
 const payload = {
  executionMode:"full_access",modeRevision:"1",modeSnapshotRef:{Kind:"agent-execution-mode-v1",ID:op,Digest:"a".repeat(64)},
  runId:op,requestKey:key,operationId:op,productKey:"product",catalogVersion:"1",publicationId:"publication",targetPlatform:"product",
  contentSelection:{targetLanguage:"en-US",description:true},phase:"human_review_required",revision:"2",humanReviewRequired:true,canSubmitReview:false,
  candidate:{Changes:[{Field:"description",Value:"Readable suggestion",EvidenceIDs:["source"]}],Warnings:[{Code:"missing_material",Field:"description",Message:"Missing material",Metadata:null}],Rejections:[]},
  confidence:[],unresolved:["missing_material: Missing material"],steps:[],tokens:50,estimatedCostMicros:30,currency:"CNY",usageStatus:"observed",
 };
 const readable=await buildWorkbenchBrowserResponse(Response.json(payload),"product-agent-content-result",key);
 expect(readable.status).toBe(200);
 await expect(readable.json()).resolves.toMatchObject({canSubmitReview:false,unresolved:payload.unresolved,candidate:payload.candidate});
 const unsafe=await buildWorkbenchBrowserResponse(Response.json({...payload,canSubmitReview:true}),"product-agent-content-result",key);
 expect(unsafe.status).not.toBe(200);
});
it("forwards the fixed content namespace and rejects missing selection or title/content confusion", async () => {
 vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://localhost:3000");
 const payload={expectedModeRevision:"0",targetPlatform:"product",contentSelection:{targetLanguage:"ja",description:true}};
 const contentRequest=(body:unknown)=>new Request(`http://localhost:3000/api/workbench/${contentPath.join("/")}`,{method:"POST",headers:req().headers,body:JSON.stringify(body)});
 const result=await buildWorkbenchUpstreamRequest(contentRequest(payload),contentPath,"token","actor");
 expect(result).not.toBeInstanceOf(Response);
 if(!(result instanceof Response))expect(JSON.parse(result.init.body as string)).toEqual(payload);
 expect(await buildWorkbenchUpstreamRequest(contentRequest({expectedModeRevision:"0",targetPlatform:"product"}),contentPath,"token","actor")).toBeInstanceOf(Response);
 expect(await buildWorkbenchUpstreamRequest(req(JSON.stringify({...payload,targetPlatform:"shein"})),path,"token","actor")).toBeInstanceOf(Response);
});
function req(body = '{"targetPlatform":"shein","expectedModeRevision":"0"}',  extra: Record<string, string> = {}) { return new Request(`http://localhost:3000/api/workbench/${path.join("/")}`, { method: "POST", headers: { Origin: "http://localhost:3000", "Sec-Fetch-Site": "same-origin", Cookie: "shuomi_effective_organization=B", "X-Expected-Organization-ID": "B", "X-Expected-User-ID": "actor", "Content-Type": "application/json", "Idempotency-Key": key, ...extra }, body }); }
it("admits an explicitly selected platform for Agent Start and rejects client authority and stale identity", async () => {
    vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://localhost:3000");
    const result = await buildWorkbenchUpstreamRequest(req(), path, "token", "actor");
    expect(result).not.toBeInstanceOf(Response);
    if (result instanceof Response)
        return;
    expect(JSON.parse(result.init.body as string)).toEqual({targetPlatform:"shein",expectedModeRevision:"0"});
    expect(new Headers(result.init.headers).get("Idempotency-Key")).toBe(key);
    expect(result.sourceMutation).toBe(true);
    for (const request of [req('{}'), req('{"targetPlatform":"product"}'), req('{"model":"other"}'), req('{"organizationId":"A"}'), req('{}', { "X-Expected-User-ID": "other" }), req('{}', { Origin: "http://evil.test" }), req('{}', { "Idempotency-Key": "" })]) {
        expect(await buildWorkbenchUpstreamRequest(request, path, "token", "actor")).toBeInstanceOf(Response);
    }
});
it("does not turn a mismatched result or unmounted Agent into successful generation", async () => {
    const unavailable = await buildWorkbenchBrowserResponse(Response.json({ code: "PRODUCT_AGENT_UNAVAILABLE" }, { status: 503 }), "product-agent-result", key, { sourceMutation: true });
    expect(unavailable.status).toBe(503);
    expect((await unavailable.json()).code).toBe("PRODUCT_AGENT_UNAVAILABLE");
    const wrong = await buildWorkbenchBrowserResponse(Response.json({ proposalId: op, requestKey: op, operationId: op }), "product-agent-review", key, { sourceMutation: true });
    expect(wrong.status).toBe(503);
    expect((await wrong.json()).code).toBe("OUTCOME_UNKNOWN");
});

it("passes only normalized optional Knowledge selection and rejects context authority", async () => {
 vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://localhost:3000");
 const body=JSON.stringify({targetPlatform:"shein",expectedModeRevision:"0",knowledgeSelection:{knowledgeBaseId:key}});
 const result=await buildWorkbenchUpstreamRequest(req(body),path,"token","actor");
 expect(result).not.toBeInstanceOf(Response);
 if(!(result instanceof Response)) expect(JSON.parse(result.init.body as string)).toEqual(JSON.parse(body));
 for(const knowledgeSelection of [null,{}, {knowledgeBaseId:key,revisionId:op},{knowledgeBaseId:key,text:"secret"},{bundleId:key}, {knowledgeBaseId:"bad"}]) {
  const denied=await buildWorkbenchUpstreamRequest(req(JSON.stringify({targetPlatform:"shein",expectedModeRevision:"0",knowledgeSelection})),path,"token","actor");
  expect(denied).toBeInstanceOf(Response);
 }
 const unavailable=await buildWorkbenchBrowserResponse(Response.json({code:"KNOWLEDGE_CONTEXT_TOO_LARGE"},{status:409}),"product-agent-result",key,{sourceMutation:true});
 expect(unavailable.status).toBe(409);expect((await unavailable.json()).code).toBe("KNOWLEDGE_CONTEXT_TOO_LARGE");
});
