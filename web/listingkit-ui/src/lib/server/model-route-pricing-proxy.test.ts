import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {proxyModelRoutePricing} from "./model-route-pricing-proxy";
const url="https://console.test/api/model-route-pricing/image/imageagent.generation/publish";
const input={expectedRevision:"NONE",bindingId:"image-a",credentialVersion:"v1",rates:{pointsPerImage:"10"}};
function request(body:unknown=input,extra:Record<string,string>={}){return new Request(url,{method:"POST",headers:{"Content-Type":"application/json","Origin":"https://console.test","Cookie":"shuomi_effective_organization=org-a","X-Expected-User-ID":"maintainer","X-Expected-Organization-ID":"org-a","Idempotency-Key":"original",...extra},body:JSON.stringify(body)})}
beforeEach(()=>{vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","https://console.test");vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","http://localhost:9000/api/v1")});
afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs()});
it("rejects forbidden shapes and changed identity before forwarding",async()=>{const fetch=vi.fn();vi.stubGlobal("fetch",fetch);expect((await proxyModelRoutePricing(request({...input,apiKey:"secret"}),"token","maintainer")).status).toBe(400);expect((await proxyModelRoutePricing(request(input,{"X-Expected-User-ID":"other"}),"token","maintainer")).status).toBe(409);expect(fetch).not.toHaveBeenCalled()});
it("rejects scientific/fractional point rates",async()=>{vi.stubGlobal("fetch",vi.fn());expect((await proxyModelRoutePricing(request({...input,rates:{pointsPerImage:"1e3"}}),"token","maintainer")).status).toBe(400)});
const receipt={id:"r1",scope:{organizationId:"org-a",capability:"imageagent.generation"},parent:"NONE",schemaVersion:1,payload:{bindingId:"image-a",credentialVersion:"v1",rates:{pointsPerImage:"10"},image:{provider:"grsai",model:"gpt-image-2.5",protocol:"grsai-json-sync-v1",endpointDigest:"digest",credentialReference:"db:1:1:image-a",configurationVersion:"r1",priceVersion:"r1",points:10}},digest:"digest",actorId:"maintainer",publishedAt:"2026-10-19T00:00:00Z",key:"original"};
it("forwards original key and selected organization, rejects wrong receipts or secret fields",async()=>{
 const fetch=vi.fn().mockImplementation(()=>Promise.resolve(Response.json({receipt})));vi.stubGlobal("fetch",fetch);
 expect((await proxyModelRoutePricing(request(),"token","maintainer")).status).toBe(200);const headers=new Headers(fetch.mock.calls[0][1].headers);expect(headers.get("Idempotency-Key")).toBe("original");expect(headers.get("X-Requested-Organization-ID")).toBe("org-a");expect(fetch.mock.calls[0][1].cache).toBe("no-store");
 for(const bad of [{...receipt,scope:{...receipt.scope,organizationId:"org-b"}},{...receipt,key:"other"},{...receipt,apiKey:"secret"}]){fetch.mockImplementation(()=>Promise.resolve(Response.json({receipt:bad})));const r=await proxyModelRoutePricing(request(),"token","maintainer");expect(r.status).toBe(502);expect(await r.json()).toEqual({code:"OUTCOME_UNKNOWN"})}
});
it("marks interrupted publication unknown and rejects cross-origin/context changes before send",async()=>{
 const fetch=vi.fn().mockRejectedValue(new Error("lost response"));vi.stubGlobal("fetch",fetch);expect(await (await proxyModelRoutePricing(request(),"token","maintainer")).json()).toEqual({code:"OUTCOME_UNKNOWN"});
 expect((await proxyModelRoutePricing(request(input,{Origin:"https://evil.test"}),"token","maintainer")).status).toBe(403);expect((await proxyModelRoutePricing(request(input,{"X-Expected-Organization-ID":"org-b"}),"token","maintainer")).status).toBe(409);expect(fetch).toHaveBeenCalledTimes(1);
});
