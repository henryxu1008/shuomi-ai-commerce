import {afterEach,expect,it,vi} from "vitest";
import {proxyStoreCenter} from "./store-center-proxy";
import {WORKBENCH_COOKIE_NAME} from "./workbench-proxy";
const headers={"X-Expected-Organization-ID":"org-a","X-Expected-User-ID":"actor-a",cookie:`${WORKBENCH_COOKIE_NAME}=org-a`,Origin:"http://localhost:3000"};
afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs()});
it("rejects context drift and unknown filters before dispatch",async()=>{
 const fetch=vi.fn();vi.stubGlobal("fetch",fetch);
 for(const path of ["catalog?q=x&q=y","catalog?tenantId=foreign","catalog?attention=active","catalog?q=%FF"]){
  expect((await proxyStoreCenter(new Request(`http://localhost:3000/api/workbench/store-center/${path}`,{headers}),"secret","actor-a")).status).toBe(400);
 }
 expect((await proxyStoreCenter(new Request("http://localhost:3000/api/workbench/store-center/groups",{headers:{...headers,"X-Expected-User-ID":"old"}}),"secret","actor-a")).status).toBe(409);
 expect(fetch).not.toHaveBeenCalled();
});
it("checks upstream scope and retains a command key after an ambiguous transport result",async()=>{
 vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","http://127.0.0.1:9000/api/v1");vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","http://localhost:3000");
 const fetch=vi.fn().mockResolvedValue(Response.json({schemaVersion:"store-center-v1",organizationId:"foreign",userId:"actor-a",data:{items:[]}}));vi.stubGlobal("fetch",fetch);
 expect((await proxyStoreCenter(new Request("http://localhost:3000/api/workbench/store-center/groups",{headers}),"secret","actor-a")).status).toBe(409);
 fetch.mockRejectedValueOnce(new Error("lost"));
 const key="00000000-0000-4000-8000-000000000001";
 const response=await proxyStoreCenter(new Request("http://localhost:3000/api/workbench/store-center/metadata",{method:"POST",headers:{...headers,"Content-Type":"application/json","Idempotency-Key":key},body:JSON.stringify({action:"create_group",name:"新组",groupId:null,groupVersion:0,stores:[],responsibleMemberId:null,lowStockThreshold:null})}),"secret","actor-a");
 expect(response.status).toBe(502);expect((await response.json()).code).toBe("OUTCOME_UNKNOWN");expect(new Headers(fetch.mock.calls[1][1].headers).get("Idempotency-Key")).toBe(key);
});
