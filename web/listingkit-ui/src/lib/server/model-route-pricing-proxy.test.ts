import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {proxyModelRoutePricing} from "./model-route-pricing-proxy";

it("reads automatic FX status through the authenticated platform path",async()=>{
 vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({state:"CHECKED",checkedAt:"2026-11-12T00:00:00Z",stopRequested:false})));
 const result=await proxyModelRoutePricing(request(undefined,{},"/platform-prices/automatic-fx","GET"),"token","maintainer");
 expect(result.status).toBe(200);
 expect((await result.json()).state).toBe("CHECKED");
});
const base="https://console.test/api/model-route-pricing";
const selection={schemaVersion:1,unitVersion:"ai-micropoint-v1",defaultChannel:"openai-direct",text:{id:"mini",version:"v1"},image:{id:"image",version:"v1"},multimodal:{id:"mini",version:"v1"}};
const input={expectedRevision:"NONE",selection};
function request(body:unknown=input,extra:Record<string,string>={},path="/enterprise/publish",method="POST"){return new Request(base+path,{method,headers:{"Content-Type":"application/json","Origin":"https://console.test","Cookie":"shuomi_effective_organization=org-a","X-Expected-User-ID":"maintainer","X-Expected-Organization-ID":"org-a","Idempotency-Key":"original",...extra},...method==="POST"?{body:JSON.stringify(body)}:{}})}
const receipt=()=>({receipt:{id:"r1",scope:{organizationId:"org-a",capability:"ai.models.enterprise"},parent:"NONE",schemaVersion:1,payload:{bindingId:"",credentialVersion:"",rates:{},sharedModels:selection},digest:"digest",actorId:"maintainer",publishedAt:"2026-11-03T00:00:00Z",key:"original"}});
beforeEach(()=>{vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","https://console.test");vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","http://localhost:9000/api/v1")});
afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs()});

const credentialResult=()=>({receipt:{organizationId:"org-a",actorId:"maintainer",requestKey:"original",connectionId:"shared",expectedRevision:"0",action:"set-key",connection:{organizationId:"org-a",connectionId:"shared",provider:"openai",channel:"openai-direct",region:"global",account:"account",revision:"1",enabled:true,keyPresent:true,models:[]}}});
it("forwards a bounded credential command through the original trusted BFF without echoing its Key",async()=>{
 const fetch=vi.fn().mockResolvedValue(Response.json(credentialResult()));vi.stubGlobal("fetch",fetch);
 const body={action:"set-key",expectedRevision:"0",account:"account",apiKey:"synthetic-only-key"};
 const result=await proxyModelRoutePricing(request(body,{},"/connections/shared/commands"),"token","maintainer");
 expect(result.status).toBe(200);expect(await result.text()).not.toContain(body.apiKey);
 expect(fetch).toHaveBeenCalledTimes(1);expect(JSON.parse(fetch.mock.calls[0][1].body)).toEqual(body);
 expect(new Headers(fetch.mock.calls[0][1].headers).get("Idempotency-Key")).toBe("original");
});
it("rejects credential identity injection and keeps malformed or mismatched post-send replies unknown",async()=>{
 const body={action:"set-key",expectedRevision:"0",account:"account",apiKey:"synthetic-only-key"};
 const fetch=vi.fn();vi.stubGlobal("fetch",fetch);
 for(const input of [{...body,writerDatabase:{}},{...body,organizationId:"org-b"},{...body,baseURL:"https://other.test"}])expect((await proxyModelRoutePricing(request(input,{},"/connections/shared/commands"),"token","maintainer")).status).toBe(400);
 expect(fetch).not.toHaveBeenCalled();
 for(const change of [(r:ReturnType<typeof credentialResult>)=>{r.receipt.actorId="other"},(r:ReturnType<typeof credentialResult>)=>{r.receipt.connectionId="other"},(r:ReturnType<typeof credentialResult>)=>{r.receipt.connection.account="different"}]){
  const value=credentialResult();change(value);fetch.mockResolvedValue(Response.json(value));
  expect(await(await proxyModelRoutePricing(request(body,{},"/connections/shared/commands"),"token","maintainer")).json()).toEqual({code:"OUTCOME_UNKNOWN"});
 }
 fetch.mockResolvedValue(Response.json({...credentialResult(),apiKey:body.apiKey}));
 expect(await(await proxyModelRoutePricing(request(body,{},"/connections/shared/commands"),"token","maintainer")).json()).toEqual({code:"OUTCOME_UNKNOWN"});
});
it("reads an original credential receipt without retransmitting a secret and binds its target",async()=>{
 const fetch=vi.fn().mockResolvedValue(Response.json(credentialResult()));vi.stubGlobal("fetch",fetch);
 const response=await proxyModelRoutePricing(request(undefined,{},"/connections/shared/receipts/original","GET"),"token","maintainer");
 expect(response.status).toBe(200);expect(fetch.mock.calls[0][1].body).toBeUndefined();
 const value=credentialResult();value.receipt.connection.organizationId="org-b";fetch.mockResolvedValue(Response.json(value));
 expect((await proxyModelRoutePricing(request(undefined,{},"/connections/shared/receipts/original","GET"),"token","maintainer")).status).toBe(502);
});

it.each(["publish","history","validate","options","commands","receipts"])("keeps original receipt reads independent of the valid connection identifier %s",async id=>{
 const value=credentialResult();value.receipt.connectionId=id;value.receipt.connection.connectionId=id;
 const fetch=vi.fn().mockResolvedValue(Response.json(value));vi.stubGlobal("fetch",fetch);
 const response=await proxyModelRoutePricing(request(undefined,{},"/connections/"+id+"/receipts/original","GET"),"token","maintainer");
 expect(response.status).toBe(200);expect(await response.json()).toEqual(value);
 expect(fetch).toHaveBeenCalledTimes(1);expect(fetch.mock.calls[0][1].method).toBe("GET");expect(fetch.mock.calls[0][1].body).toBeUndefined();
});
it("retires both independent point tariffs and every old per-purpose write URL",async()=>{const fetch=vi.fn();vi.stubGlobal("fetch",fetch);for(const body of [{...input,pointsPerImage:"10"},{...input,inputPoints:"2"},{...input,selection:{...selection,price:{rate:1}}}])expect((await proxyModelRoutePricing(request(body),"token","maintainer")).status).toBe(400);expect((await proxyModelRoutePricing(request(input,{},"/image/imageagent.generation/publish"),"token","maintainer")).status).toBe(400);expect(fetch).not.toHaveBeenCalled()});
it("rejects changed user, organization, cross-origin and duplicate JSON before forwarding",async()=>{const fetch=vi.fn();vi.stubGlobal("fetch",fetch);for(const extra of [{"X-Expected-User-ID":"other"},{"X-Expected-Organization-ID":"other"}] as Record<string,string>[])expect((await proxyModelRoutePricing(request(input,extra),"token","maintainer")).status).toBe(409);expect((await proxyModelRoutePricing(request(input,{Origin:"https://other.test"}),"token","maintainer")).status).toBe(403);const duplicate=new Request(request(),{body:JSON.stringify(input).replace('"expectedRevision":"NONE"','"expectedRevision":"NONE","expectedRevision":"NONE"')});expect((await proxyModelRoutePricing(duplicate,"token","maintainer")).status).toBe(400);expect(fetch).not.toHaveBeenCalled()});
it("forwards the complete enterprise selection with its original key and verified organization",async()=>{const fetch=vi.fn().mockResolvedValue(Response.json(receipt()));vi.stubGlobal("fetch",fetch);const result=await proxyModelRoutePricing(request(),"token","maintainer");expect(result.status).toBe(200);expect(fetch.mock.calls[0][0]).toBe("http://localhost:9000/api/v1/model-route-pricing/enterprise/publish");const headers=new Headers(fetch.mock.calls[0][1].headers);expect(headers.get("Authorization")).toBe("Bearer token");expect(headers.get("Idempotency-Key")).toBe("original");expect(headers.get("X-Requested-Organization-ID")).toBe("org-a");expect(JSON.parse(fetch.mock.calls[0][1].body)).toEqual(input)});
it("does not confirm a receipt for another organization, actor, key, parent or model choice",async()=>{for(const change of [r=>{r.receipt.scope.organizationId="other"},r=>{r.receipt.actorId="other"},r=>{r.receipt.key="other"},r=>{r.receipt.parent="other"},r=>{r.receipt.payload.sharedModels={...selection,image:{id:"different",version:"v1"}}}] as ((r:ReturnType<typeof receipt>)=>void)[]){const value=receipt();change(value);vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json(value)));const response=await proxyModelRoutePricing(request(),"token","maintainer");expect(await response.json()).toEqual({code:"OUTCOME_UNKNOWN"})}});
it("keeps dispatch failures uncertain and preserves a definite pre-dispatch rejection",async()=>{vi.stubGlobal("fetch",vi.fn().mockRejectedValue(new Error("ack lost")));expect(await(await proxyModelRoutePricing(request(),"token","maintainer")).json()).toEqual({code:"OUTCOME_UNKNOWN"});vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({code:"NEEDS_CONFIGURATION"},{status:409})));expect((await proxyModelRoutePricing(request(),"token","maintainer")).status).toBe(409)});
it.each([[401,"AUTHENTICATION_REQUIRED"],[403,"PERMISSION_DENIED"],[403,"ORGANIZATION_ACCESS_REVOKED"],[409,"ORGANIZATION_SELECTION_REQUIRED"]] as const)("preserves a proved initial middleware rejection %s/%s without leaking metadata",async(status,code)=>{vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({code,message:"private diagnostic",requestId:"private-request",fieldErrors:[]},{status})));const response=await proxyModelRoutePricing(request(),"token","maintainer");expect(response.status).toBe(status);expect(await response.json()).toEqual({code})});
it("keeps mismatched or malformed denials and every server failure uncertain",async()=>{for(const[status,payload]of [[401,{code:"PERMISSION_DENIED",message:"mismatch",requestId:"r",fieldErrors:[]}],[403,{code:"PERMISSION_DENIED",message:"missing fields"}],[403,{code:"ARBITRARY_REJECTION"}],[503,{code:"PERMISSION_DENIED",message:"wrong status",requestId:"r",fieldErrors:[]}]] as const){vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json(payload,{status})));expect(await(await proxyModelRoutePricing(request(),"token","maintainer")).json()).toEqual({code:"OUTCOME_UNKNOWN"})}});
it("rejects mixed-unit selections without querying the backend",async()=>{const fetch=vi.fn();vi.stubGlobal("fetch",fetch);expect((await proxyModelRoutePricing(request({...input,selection:{...selection,unitVersion:"whole-point"}}),"token","maintainer")).status).toBe(400);expect(fetch).not.toHaveBeenCalled()});

it("verifies official price receipts after exact timestamp normalization without losing fractions",async()=>{
 const price={id:"mini",region:"global",serviceType:"standard-sync",provider:"openai",channel:"openai-direct",model:"synthetic-mini",currency:"CNY",usageVersion:"openai-text-v1",rates:{inputText:1,cachedInputText:0,inputImage:0,cachedInputImage:0,outputText:1,outputImage:0,outputImagePerItem:0},sourceUrl:"https://developers.openai.com/api/docs/pricing",verifiedAt:"2026-11-03T08:00:00.000+08:00",effectiveFrom:"2026-11-03T08:00:00.123456789+08:00",effectiveUntil:"2026-11-04T08:00:00+08:00"};
 const body={expectedRevision:"NONE",prices:[price],fx:[]};const value=receipt();const stored={...price,version:"price-version",verifiedAt:"2026-11-03T00:00:00Z",effectiveFrom:"2026-11-03T00:00:00.123456789Z",effectiveUntil:"2026-11-04T00:00:00Z",rates:{...price.rates,outputImagePerItem:undefined}};
 const result={receipt:{...value.receipt,scope:{organizationId:"platform-ai-models",capability:"ai.models.official-prices"},payload:{bindingId:"",credentialVersion:"",rates:{},platformPrices:{schemaVersion:1,unitVersion:"ai-micropoint-v1",prices:[stored],fx:[]}}}};
 vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json(result)));expect((await proxyModelRoutePricing(request(body,{},"/platform-prices/publish"),"token","maintainer")).status).toBe(200);
 result.receipt.payload.platformPrices.prices[0].rates.outputText=2;vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json(result)));expect(await(await proxyModelRoutePricing(request(body,{},"/platform-prices/publish"),"token","maintainer")).json()).toEqual({code:"OUTCOME_UNKNOWN"});
});
it("rejects price timestamps beyond persisted nanosecond precision before forwarding",async()=>{const price={id:"mini",region:"global",serviceType:"standard-sync",provider:"openai",channel:"openai-direct",model:"synthetic-mini",currency:"CNY",usageVersion:"openai-text-v1",rates:{inputText:1,cachedInputText:0,inputImage:0,cachedInputImage:0,outputText:1,outputImage:0},sourceUrl:"https://developers.openai.com/api/docs/pricing",verifiedAt:"2026-11-03T00:00:00Z",effectiveFrom:"2026-11-03T00:00:00Z",effectiveUntil:"2026-11-04T00:00:00Z"};for(const field of ["verifiedAt","effectiveFrom","effectiveUntil"]){const fetch=vi.fn();vi.stubGlobal("fetch",fetch);const input={expectedRevision:"NONE",prices:[{...price,[field]:"2026-11-03T00:00:00.1234567891Z"}],fx:[]};expect((await proxyModelRoutePricing(request(input,{},"/platform-prices/publish"),"token","maintainer")).status).toBe(400);expect(fetch).not.toHaveBeenCalled()}});
