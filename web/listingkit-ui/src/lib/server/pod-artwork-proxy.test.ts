import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {proxyPOD} from "./pod-proxy";
const id="12752596-6056-4316-9f2f-380c97df9675";
function request(path:string,init:RequestInit={}){return new Request("https://console.example/api/workbench/pod-artwork/"+path,{...init,headers:{"X-Expected-User-ID":"user-a","X-Expected-Organization-ID":"org-a",cookie:"shuomi_effective_organization=org-a",...init.headers}})}
beforeEach(()=>{vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","https://api.example/api/v1");vi.stubEnv("LISTINGKIT_SDS_POD_ENABLED","false");vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","https://console.example")});afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs()});
it("creative preparation has its own path and is not blocked by absent SDS; tenant drift and provider fields are rejected",async()=>{
 const fetcher=vi.fn(async(_url:RequestInfo|URL,_init?:RequestInit)=>Response.json({code:"IMAGE_AGENT_DISABLED"},{status:409}));vi.stubGlobal("fetch",fetcher);
 const body={theme:"原创狐狸",requirements:"水彩",contextId:id,expectedModeRevision:"0"};const write={method:"POST",headers:{Origin:"https://console.example","Content-Type":"application/json","Idempotency-Key":id},body:JSON.stringify(body)};
 expect((await proxyPOD(request("prepare",write),"private-token","user-a")).status).toBe(409);expect(fetcher.mock.calls[0]?.[0]).toBe("https://api.example/api/v1/workbench/pod-artwork/prepare");
 fetcher.mockClear();expect((await proxyPOD(request("prepare",{...write,body:JSON.stringify({...body,provider:"other"})}),"private-token","user-a")).status).toBe(400);expect((await proxyPOD(request("runs/"+id,{headers:{"X-Expected-Organization-ID":"other"}}),"private-token","user-a")).status).toBe(409);expect(fetcher).not.toHaveBeenCalled();
});
it("confirmation requires the frozen quote, plan and original idempotency key",async()=>{const fetcher=vi.fn();vi.stubGlobal("fetch",fetcher);const r=await proxyPOD(request("runs/"+id+"/confirm",{method:"POST",headers:{Origin:"https://console.example","Content-Type":"application/json","Idempotency-Key":id},body:JSON.stringify({planRevision:1})}),"private-token","user-a");expect(r.status).toBe(400);expect(fetcher).not.toHaveBeenCalled()});
