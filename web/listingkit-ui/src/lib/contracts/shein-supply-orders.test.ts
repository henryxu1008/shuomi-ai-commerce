import {expect,it} from "vitest";
import {supplyObservationSchema} from "./shein-supply-orders";
const id="d6f6ca0a-27e2-4c4a-b1aa-4505110ae635";
it("accepts scoped partial observations and rejects money or natural dates",()=>{
 const value={storeId:id,status:"incomplete",canSync:false,savedOrderIds:["PO-A"],source:{syncId:id,status:"partial",orderIds:["PO-A","PO-MISSING"],missingIds:["PO-MISSING"],notes:["supply_ids_not_returned"],startedAt:"2026-11-09T01:00:00Z",observedAt:null}};
 expect(supplyObservationSchema.parse(value)).toEqual(value);
 expect(supplyObservationSchema.safeParse({...value,amount:"10"}).success).toBe(false);
 expect(supplyObservationSchema.safeParse({...value,startDate:"2026-11-09"}).success).toBe(false);
});
