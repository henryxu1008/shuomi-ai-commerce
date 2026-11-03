import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {buildWorkbenchBrowserResponse,buildWorkbenchUpstreamRequest} from "./workbench-proxy";
const id="11111111-1111-4111-8111-111111111111",key="22222222-2222-4222-8222-222222222222",path=["supply-preparations","sources",id,"images","analysis"],hash="a".repeat(64);
function request(method:string,tail:string[],body?:unknown){return new Request(`http://localhost:3000/api/workbench/${[...path,...tail].join("/")}`,{method,headers:{Origin:"http://localhost:3000","Sec-Fetch-Site":"same-origin",Cookie:"shuomi_effective_organization=B","X-Expected-Organization-ID":"B","X-Expected-User-ID":"actor",...body?{"Content-Type":"application/json"}:{},...tail[0]==="prepare"?{"Idempotency-Key":key}:{}},body:body?JSON.stringify(body):undefined})}
beforeEach(()=>vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","http://localhost:3000"));afterEach(()=>vi.unstubAllEnvs());
it("binds the original source, actor and preparation key with write uncertainty only on POST",async()=>{
 const p=await buildWorkbenchUpstreamRequest(request("POST",["prepare"],{kind:"source",imageId:"img",catalogVersion:"1",publicationId:"pub",expectedModeRevision:"0"}),[...path,"prepare"],"token","actor");
 expect(p).not.toBeInstanceOf(Response);if(p instanceof Response)return;
 expect(p.sourceMutation).toBe(true);expect(JSON.parse(p.expectedStoreId!)).toEqual({contextKind:"collection",contextId:id,requestKey:key});
 const read=await buildWorkbenchUpstreamRequest(request("GET",["by-key",key]),[...path,"by-key",key],"token","actor");
 expect(read).not.toBeInstanceOf(Response);if(read instanceof Response)return;expect(read.sourceMutation).toBe(false);
 expect(await buildWorkbenchUpstreamRequest(request("POST",["prepare"],{kind:"source",imageId:"img",catalogVersion:"1",publicationId:"pub",expectedModeRevision:"0",url:"https://evil.test/x"}),[...path,"prepare"],"token","actor")).toBeInstanceOf(Response);
});
it("rejects crossed result identities and preserves UNKNOWN instead of exposing raw output or guessing no-send",async()=>{
 const payload={contextKind:"collection",contextId:id,requestKey:key,catalogVersion:"1",publicationId:"pub",unitVersion:"ai-micropoint-v1",status:"unknown",stopReason:"",runId:id,kind:"source",imageId:"img",modelId:"mini",preparationId:id,configurationDigest:hash,quoteDigest:hash,selectionDigest:hash,contentSha256:hash,maximumMicropoints:"10",actualMicropoints:"",width:2,height:2};
 const binding=JSON.stringify({contextKind:"collection",contextId:id,requestKey:key});
 expect((await buildWorkbenchBrowserResponse(Response.json(payload),"product-agent-image-analysis-read",binding)).status).toBe(200);
 const bad=await buildWorkbenchBrowserResponse(Response.json({...payload,contextKind:"acquisition"}),"product-agent-image-analysis-confirm",binding,{sourceMutation:true});
 expect(bad.status).toBe(503);expect((await bad.json()).code).toBe("OUTCOME_UNKNOWN");
 expect((await buildWorkbenchBrowserResponse(Response.json({...payload,modelOutput:"raw"}),"product-agent-image-analysis-read",binding)).status).toBe(502);
});
