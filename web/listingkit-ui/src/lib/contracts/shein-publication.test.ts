import {expect,it} from "vitest";
import {supplyPath,supplyUsesKey,supplyMutates,parseSupplyResponse} from "./supply-chain";
import {sheinPublicationStatusSchema,sheinShelfActionSchema} from "./shein-publication";
const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222";
it("keeps SHEIN reads separate from the confirmed mutation",()=>{
 expect(supplyPath("POST",["supply-preparations","shein-publication","status"])).toBe("shein-status");
 expect(supplyPath("POST",["supply-preparations","shein-publication","confirm"])).toBe("shein-shelf-confirm");
 expect(supplyUsesKey("shein-shelf-confirm")).toBe(true);
 expect(supplyMutates("shein-status")).toBe(false);
 expect(supplyMutates("shein-shelf-result")).toBe(false);
 expect(supplyPath("POST",["supply-preparations","shein-publication","force-retry"])).toBeNull();
});
it("retains unknown stock and binds status and action to their original receipt",()=>{
 const value={sourceId:id,storeId:other,receiptId:id,spu:"spu-a",version:"v1",mode:"fully_managed",observedAt:"2026-10-19T00:00:00Z",stockType:"PI",stockOffset:0,stockTotal:1,stockComplete:false,skcs:[{skc:"skc-a",auditState:2,auditLabel:"审核通过",reasons:[],shelfState:1,shelfWay:"1",shelfRequire:"1",hopeOnSaleDate:"",canShelf:false,blockingReason:"等待平台",skus:[{code:"sku-a",mallState:1,usable:null,reportedUsable:12,inventoryScopeKnown:false}]}]};
 expect(sheinPublicationStatusSchema.parse(value).skcs[0]?.skus[0]?.usable).toBeNull();
 expect(parseSupplyResponse("shein-status",value,JSON.stringify({sourceId:id,storeId:other,receiptId:id}))).toBeTruthy();
 expect(parseSupplyResponse("shein-status",value,JSON.stringify({sourceId:other,storeId:id,receiptId:id}))).toBeNull();
 expect(sheinShelfActionSchema.safeParse({key:id,attemptId:id,sourceId:id,storeId:other,receiptId:id,skc:"skc-a",status:"outcome_unknown",message:"待核实"}).success).toBe(true);
});

