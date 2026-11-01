import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {proxyOfflineRecords} from "./offline-shein-records-proxy";
import {WORKBENCH_COOKIE_NAME} from "./workbench-proxy";
const id="550e8400-e29b-41d4-a716-446655440001";
const body={collectionItemId:id,input:{product_key:"p",snapshot_version:"1",store_id:id,country:"US",language:"en",action:"save_draft"}};
const headers={"X-Expected-User-ID":"actor","X-Expected-Organization-ID":"org-a",cookie:`${WORKBENCH_COOKIE_NAME}=org-a`,origin:"https://app.example.com","Content-Type":"application/json","Idempotency-Key":id};
const request=(value=JSON.stringify(body))=>new Request("https://app.example.com/api/offline-shein-records",{method:"POST",headers,body:value});
beforeEach(()=>{for(const key of ["LISTINGKIT_SHEIN_OFFLINE_RECORDS_ENABLED","LISTINGKIT_REPORT_CENTER_ENABLED","LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED"])vi.stubEnv(key,"true");vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","http://localhost:8080/api/v1");vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","https://app.example.com");});
afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs()});
it("rejects disabled feature, scope drift, duplicate JSON and unknown facts before dispatch",async()=>{
 const fetch=vi.fn();vi.stubGlobal("fetch",fetch);
 for(const invalid of [JSON.stringify({...body,payload:"client"}),`{"collectionItemId":"${id}","collectionItemId":"${id}","input":${JSON.stringify(body.input)}}`])expect((await proxyOfflineRecords(request(invalid),"token","actor")).status).toBe(400);
 const drift=request();drift.headers.set("X-Expected-Organization-ID","other");expect((await proxyOfflineRecords(drift,"token","actor")).status).toBe(409);
 const cross=request();cross.headers.set("origin","https://foreign.test");expect((await proxyOfflineRecords(cross,"token","actor")).status).toBe(403);
 vi.stubEnv("LISTINGKIT_SHEIN_OFFLINE_RECORDS_ENABLED","false");expect((await proxyOfflineRecords(request(),"token","actor")).status).toBe(503);expect(fetch).not.toHaveBeenCalled();
});
it("uses normal owned endpoint and preserves unknown response loss without retries",async()=>{
 const fetch=vi.fn().mockRejectedValue(new Error("lost"));vi.stubGlobal("fetch",fetch);
 const result=await proxyOfflineRecords(request(),"server-token","actor");expect(await result.json()).toEqual({code:"OUTCOME_UNKNOWN"});expect(fetch).toHaveBeenCalledTimes(1);
 expect(fetch.mock.calls[0][0]).toBe("http://localhost:8080/api/v1/workbench/offline-shein-records");const init=fetch.mock.calls[0][1];expect(init.redirect).toBe("manual");expect(new Headers(init.headers).get("Authorization")).toBe("Bearer server-token");expect(new Headers(init.headers).has("X-Expected-User-ID")).toBe(false);
});
it("accepts a bounded original receipt without widening fields",async()=>{
 const receipt={record_id:id,input_hash:`sha256:${"a".repeat(64)}`,diagnostic_hash:`sha256:${"b".repeat(64)}`,diagnostic_status:"blocked",diagnostic_only:true};
 const fetch=vi.fn().mockResolvedValue(Response.json(receipt,{status:201}));vi.stubGlobal("fetch",fetch);const result=await proxyOfflineRecords(request(),"server-token","actor");expect(result.status).toBe(201);expect(await result.json()).toEqual(receipt);
 expect(result.headers.get("cache-control")).toBe("private, no-store");
});
