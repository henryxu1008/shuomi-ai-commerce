import {describe,it,expect} from "vitest";
import {walmartDraftInputSchema,walmartObservationSchema} from "./walmart-model";
describe("Walmart US explicit listing contracts",()=>{
 it("rejects other feed types and does not call ingestion publication",()=>{
  expect(walmartDraftInputSchema.safeParse({selection:{feedType:"MP_WFS_ITEM",version:"5",productType:"Shirts"},skus:[]}).success).toBe(false);
  const v=walmartObservationSchema.parse({feedId:"feed",feedStatus:"PROCESSED",itemsReceived:1,itemsSucceeded:1,itemsFailed:0,itemsProcessing:0,items:[{sku:"sku",ingestionStatus:"SUCCESS",publishedStatus:"UNPUBLISHED"}],partial:false,responseHashes:[],observedAt:"2026-10-16T00:00:00Z"});
  expect(v.items?.[0]?.publishedStatus).toBe("UNPUBLISHED");
 });
});
