import {z} from "zod";
import {collectionID} from "./product-collection";

const index=z.number().int().min(0).max(255);
const id=z.number().int().positive().max(Number.MAX_SAFE_INTEGER);
const text=(max:number)=>z.string().min(1).refine(v=>new TextEncoder().encode(v).length<=max&&!v.includes("\0"));
const cmp=(a:string,b:string)=>a<b?-1:a>b?1:0;
export const sheinSalesSelectionSchema=z.object({
 dimensions:z.array(z.object({sourceName:text(200),attributeId:id,main:z.boolean()}).strict()).min(1).max(3),
 values:z.array(z.object({sourceName:text(200),sourceValue:text(8192),optionId:id}).strict()).max(768),
 rows:z.array(z.object({sourceIndex:index,skcIndex:index.max(39),skuIndex:index}).strict()).min(1).max(256),
 images:z.array(z.object({slotIndex:z.number().int().min(0).max(9999)}).strict()).max(10000),
}).strict().transform(s=>({...s,
 dimensions:s.dimensions.sort((a,b)=>cmp(a.sourceName,b.sourceName)),
 values:s.values.sort((a,b)=>cmp(a.sourceName,b.sourceName)||cmp(a.sourceValue,b.sourceValue)),
 rows:s.rows.sort((a,b)=>a.sourceIndex-b.sourceIndex),images:s.images.sort((a,b)=>a.slotIndex-b.slotIndex),
}));
export const sheinSalesRequestSchema=z.object({sourceId:collectionID,baseRecordId:collectionID,completionDraft:z.object({recordId:collectionID,revision:id}).strict(),selection:sheinSalesSelectionSchema}).strict();
export const sheinSalesBindingSchema=sheinSalesRequestSchema.omit({sourceId:true}).extend({version:z.literal("shein-sales-mapping-v1"),targetId:collectionID,rulesHash:z.string().regex(/^[a-f0-9]{64}$/),merchantHash:z.string().regex(/^[a-f0-9]{64}$/),inventoryHash:z.string().regex(/^[a-f0-9]{64}$/)}).strict();
export type SheinSalesSelection=z.infer<typeof sheinSalesSelectionSchema>;
export type SheinSalesRequest=z.infer<typeof sheinSalesRequestSchema>;
