import {expect,it} from "vitest";
import {supplyObservationSchema} from "./shein-supply-orders";
import {observationBeginSchema} from "@/lib/api/store-observations";
const id="d6f6ca0a-27e2-4c4a-b1aa-4505110ae635";
it("accepts scoped partial observations and rejects money or natural dates",()=>{
 const value={storeId:id,status:"incomplete",canSync:false,savedOrderIds:["PO-A"],source:{syncId:id,status:"partial",orderIds:["PO-A","PO-MISSING"],missingIds:["PO-MISSING"],notes:["supply_ids_not_returned"],startedAt:"2026-11-09T01:00:00Z",observedAt:null}};
 expect(supplyObservationSchema.parse(value)).toEqual(value);
 expect(supplyObservationSchema.safeParse({...value,amount:"10"}).success).toBe(false);
 expect(supplyObservationSchema.safeParse({...value,startDate:"2026-11-09"}).success).toBe(false);
});
it("matches the original checkpoint budget before persisting a read intent",()=>{
 const ids=(n:number,prefix:string)=>Array.from({length:n},(_,i)=>prefix+String(i).padStart(3,"0"));
 const parse=(supplyOrderIds:string[])=>observationBeginSchema.safeParse({kind:"supply-orders",stores:[id],supplyOrderIds}).success;
 expect(parse(ids(200,"A".repeat(125)))).toBe(true);
 expect(parse(ids(78,"采".repeat(125)))).toBe(true);
 expect(parse(ids(100,"采".repeat(125)))).toBe(false);
 expect(parse(ids(100,"<".repeat(125)))).toBe(false);
});
