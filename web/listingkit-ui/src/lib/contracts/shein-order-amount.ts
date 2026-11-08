import { z } from "zod";
const count=z.number().int().min(0).max(250);
const stamp=z.iso.datetime({offset:true});
const source=z.strictObject({
 syncId:z.uuid(),observedAt:stamp.nullable(),range:z.strictObject({start:stamp,end:stamp}).nullable(),
});
const completeSource=source.refine(value=>value.observedAt!==null&&value.range!==null);
const complete=z.strictObject({
 status:z.literal("complete"),capturedAt:stamp,source:completeSource,matchedOrderCount:count,
 amounts:z.array(z.strictObject({
  currency:z.string().regex(/^[A-Z]{3}$/),amount:z.string().max(32).regex(/^(0|[1-9][0-9]{0,22})\.[0-9]{2,8}$/),
  orderCount:count.min(1),
 })).max(250),
}).refine(value=>new Set(value.amounts.map(item=>item.currency)).size===value.amounts.length&&value.amounts.reduce((sum,item)=>sum+item.orderCount,0)===value.matchedOrderCount);
const missing=z.strictObject({
 status:z.literal("missing_fields"),capturedAt:stamp,source:completeSource,matchedOrderCount:count,
 missing:z.strictObject({amountCount:count,currencyCount:count,unknownTimeCount:count}),
}).refine(value=>value.missing.amountCount<=value.matchedOrderCount&&value.missing.currencyCount<=value.matchedOrderCount&&
 value.matchedOrderCount+value.missing.unknownTimeCount<=250&&
 value.missing.amountCount+value.missing.currencyCount+value.missing.unknownTimeCount>0);
export const savedOrderAmountMetrics=z.discriminatedUnion("status",[
 complete,missing,
 z.strictObject({status:z.literal("coverage_incomplete"),capturedAt:stamp,source:source.optional(),
  reason:z.enum(["no_saved_sync","sync_incomplete","newer_sync_incomplete","range_not_covered"])}),
 z.strictObject({status:z.literal("limited"),capturedAt:stamp,source:completeSource}),
]);
export type SavedOrderAmountMetrics=z.infer<typeof savedOrderAmountMetrics>;
