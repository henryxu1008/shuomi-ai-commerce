import {afterEach,expect,it,vi} from "vitest";
import {proxyStoreObservations} from "./store-observations-proxy";
import {WORKBENCH_COOKIE_NAME} from "./workbench-proxy";
afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs()});
const id="d6f6ca0a-27e2-4c4a-b1aa-4505110ae635";
const quantity={value:"5",missingLines:0};
const dates={range:{startDate:"2026-11-01",endDate:"2026-11-03"},urgent:{orderIds:["PO-A"],ordered:quantity},stock:{orderIds:[],ordered:{value:"0",missingLines:0}},undated:[],motherIds:[],unknownMotherIds:[],unknownPurchaseTypeIds:[],outsideRangeIds:[]};
const payload={storeId:id,status:"saved",canSync:false,savedOrderIds:["PO-A"],source:{syncId:id,status:"completed",orderIds:["PO-A"],missingIds:[],notes:[],startedAt:"2026-11-09T00:00:00Z",observedAt:"2026-11-09T00:00:01Z"},summary:{orderCount:1,motherCount:0,unknownTypeCount:0,quantities:{ordered:quantity,delivered:quantity,received:quantity,stored:quantity,defective:quantity}},originalDates:dates};
it("binds original-date observations to the requested range and complete disjoint saved identities",async()=>{
 vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","http://127.0.0.1:9000/api/v1");vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","http://localhost:3000");
 const fetch=vi.fn();vi.stubGlobal("fetch",fetch);
 const read=(query:string)=>proxyStoreObservations(new Request(`http://localhost:3000/api/workbench/store-observations/supply-orders/observation?storeId=${id}${query}`,{headers:{"X-Expected-Organization-ID":"org-a","X-Expected-User-ID":"actor-a",cookie:WORKBENCH_COOKIE_NAME+"=org-a"}}),"private","actor-a");
 for(const data of [payload,{...payload,status:"running"},{...payload,originalDates:undefined},{...payload,originalDates:{...dates,range:{startDate:"2026-11-02",endDate:"2026-11-03"}}},{...payload,originalDates:{...dates,outsideRangeIds:["PO-A"]}},{...payload,originalDates:{...dates,urgent:{...dates.urgent,orderIds:["OTHER"]}}}]){
  fetch.mockResolvedValueOnce(Response.json({organizationId:"org-a",userId:"actor-a",data}));expect((await read("&startDate=2026-11-01&endDate=2026-11-03")).status).toBe(data===payload?200:502);
 }
 fetch.mockResolvedValueOnce(Response.json({organizationId:"org-a",userId:"actor-a",data:payload}));expect((await read("")).status).toBe(502);
 fetch.mockClear();for(const q of ["&startDate=2026-11-01","&startDate=2026-02-30&endDate=2026-03-01","&startDate=2026-11-04&endDate=2026-11-03","&startDate=2026-11-01&startDate=2026-11-02&endDate=2026-11-03"]){expect((await read(q)).status).toBe(400)}expect(fetch).not.toHaveBeenCalled();
});
