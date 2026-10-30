import {describe,it,expect} from "vitest";
import {walmartDraftInputSchema,walmartObservationSchema,walmartResultSchema} from "./walmart-model";
describe("Walmart US explicit listing contracts",()=>{
 it("rejects other feed types and does not call ingestion publication",()=>{
  expect(walmartDraftInputSchema.safeParse({selection:{feedType:"MP_WFS_ITEM",version:"5",productType:"Shirts"},skus:[]}).success).toBe(false);
  const v=walmartObservationSchema.parse({feedId:"feed",feedStatus:"PROCESSED",itemsReceived:1,itemsSucceeded:1,itemsFailed:0,itemsProcessing:0,items:[{sku:"sku",ingestionStatus:"SUCCESS",publishedStatus:"UNPUBLISHED"}],partial:false,responseHashes:[],observedAt:"2026-10-16T00:00:00Z"});
  expect(v.items?.[0]?.publishedStatus).toBe("UNPUBLISHED");
 });
});

it("retains bounded read progress and historical page times without granting a complete result",()=>{
 const id="11111111-1111-4111-8111-111111111111",hash="a".repeat(64),old="2026-10-30T00:00:00Z",now="2026-10-30T00:01:00Z";
 const observation={feedId:"feed",feedStatus:"PROCESSED",itemsReceived:1000,itemsSucceeded:0,itemsFailed:1000,itemsProcessing:0,items:Array.from({length:50},(_,i)=>({sku:`sku-${i}`,ingestionStatus:"DATA_ERROR"})),partial:true,responseHashes:[hash],observedAt:now};
 const record={id,receiptId:id,generation:2,latest:true,observation,readProgress:{round:1,expectedHash:hash,startOffset:50,nextOffset:100,previousWindowId:id,pages:[{offset:50,count:50,responseHash:hash,observedAt:now}]},observedPages:[{offset:0,observation:{...observation,observedAt:old}}]};
 const parsed=walmartResultSchema.parse(record);
 expect(parsed.readProgress?.nextOffset).toBe(100);expect(parsed.observedPages?.[0]?.observation.observedAt).toBe(old);
 expect(walmartResultSchema.safeParse({...record,observation:{...observation,partial:false}}).success).toBe(false);
 expect(walmartResultSchema.safeParse({...record,readProgress:{...record.readProgress,nextOffset:101}}).success).toBe(false);
 expect(walmartResultSchema.safeParse({...record,observedPages:[{offset:0,observation:{...observation,partial:false}}]}).success).toBe(false);
});
