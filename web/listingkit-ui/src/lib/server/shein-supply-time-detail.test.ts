import {afterEach,expect,it,vi} from "vitest";
import {proxyStoreObservations} from "./store-observations-proxy";
import {WORKBENCH_COOKIE_NAME} from "./workbench-proxy";
afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs()});
const id="d6f6ca0a-27e2-4c4a-b1aa-4505110ae635",other="c0d81244-c016-442a-8184-3cac18dce4ee";
const fields=["addTime","allocateTime","updateTime","requestDeliveryTime","requestReceiptTime","requestTakeParcelTime","reserveTime","receiptTime","checkTime","storageTime","returnTime","requestCompleteTime","deliveryTime"];
const times={contract:"shein-supply-business-times/v1",source:"/open-api/order/purchase-order-infos",timezone:"unconfirmed",values:Object.fromEntries(fields.map(f=>[f,{state:"missing",rawType:"missing",raw:null}]))};
const record={storeId:id,syncId:id,id:"PO-A",observedAt:"2026-11-09T00:00:00Z",supplyOrder:{id:"PO-A",type:1,status:2,scene:1,countryMarket:null,mother:false,motherRaw:"No",lines:[],businessTimes:times}};
it("allows only bound saved supply detail with complete raw-time metadata",async()=>{
 vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","http://127.0.0.1:9000/api/v1");vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL","http://localhost:3000");
 const fetch=vi.fn();vi.stubGlobal("fetch",fetch);
 const read=()=>proxyStoreObservations(new Request(`http://localhost:3000/api/workbench/store-observations/supply-orders/stores/${id}/syncs/${id}/records/PO-A`,{headers:{"X-Expected-Organization-ID":"org-a","X-Expected-User-ID":"actor-a",cookie:WORKBENCH_COOKIE_NAME+"=org-a"}}),"private","actor-a");
 for(const data of [record,{...record,storeId:other},{...record,syncId:other},{...record,id:"OTHER"},{...record,supplyOrder:{...record.supplyOrder,businessTimes:undefined}},{...record,supplyOrder:{...record.supplyOrder,businessTimes:{...times,timezone:"UTC+8"}}}]){
  fetch.mockResolvedValueOnce(Response.json({organizationId:"org-a",userId:"actor-a",data}));const res=await read();expect(res.status).toBe(data===record?200:502);
 }
});
