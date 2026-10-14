import {afterEach,expect,it,vi} from "vitest";
import {proxySkill} from "./skill-proxy";
afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs()});
const info={available:true,canCreate:true,reason:""};
it("personal reads ignore selected organization and never forward browser authority",async()=>{
 vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","http://service.test/api/v1");
 const fetch=vi.fn().mockResolvedValue(Response.json(info));vi.stubGlobal("fetch",fetch);
 const r=await proxySkill(new Request("http://app.test/api/workbench/skills/personal/info",{headers:{"X-Expected-User-ID":"alice","X-Expected-Organization-ID":"spoof","X-User-ID":"bob"}}),"token","alice");
 expect(r.status).toBe(200);const h=new Headers(fetch.mock.calls[0][1].headers);expect(h.has("X-Requested-Organization-ID")).toBe(false);expect(h.has("X-User-ID")).toBe(false);
});
it("blocks identity and organization changes before dispatch",async()=>{
 const fetch=vi.fn();vi.stubGlobal("fetch",fetch);
 let r=await proxySkill(new Request("http://app.test/api/workbench/skills/personal/info",{headers:{"X-Expected-User-ID":"bob"}}),"token","alice");expect(r.status).toBe(409);
 r=await proxySkill(new Request("http://app.test/api/workbench/skills/organization/info",{headers:{"X-Expected-User-ID":"alice","X-Expected-Organization-ID":"org"}}),"token","alice");expect(r.status).toBe(409);expect(fetch).not.toHaveBeenCalled();
});
it("rejects owner injection and unsafe paths before dispatch",async()=>{

 vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","http://app.test");
 const fetch=vi.fn();vi.stubGlobal("fetch",fetch);
 const r=await proxySkill(new Request("http://app.test/api/workbench/skills/personal",{method:"POST",headers:{"X-Expected-User-ID":"alice",Origin:"http://app.test","Content-Type":"application/json","Idempotency-Key":"11111111-1111-4111-8111-111111111111"},body:JSON.stringify({files:[{path:"SKILL.md",text:"test"}],allowCopy:false,allowExport:false,owner:{id:"bob"}})}),"token","alice");expect(r.status).toBe(400);expect(fetch).not.toHaveBeenCalled();
});
