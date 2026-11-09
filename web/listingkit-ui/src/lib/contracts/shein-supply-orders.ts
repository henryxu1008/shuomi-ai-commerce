import {z} from "zod";
import {observationID} from "@/lib/api/store-observations";
const count=z.number().int().min(0).max(200000);
const orderID=z.string().min(1).max(128).refine(v=>v.trim()===v&&!/[\p{Cc},/\\?#]/u.test(v));
const ids=z.array(orderID).max(200);
const quantity=z.object({value:z.string().regex(/^(?:0|[1-9][0-9]{0,20})$/).nullable(),missingLines:count}).strict().refine(v=>v.missingLines===0?v.value!==null:v.value===null);
export const supplyObservationSchema=z.object({
 storeId:observationID,status:z.enum(["not_applicable","not_synced","running","saved","incomplete"]),canSync:z.boolean(),savedOrderIds:ids,
 source:z.object({syncId:observationID,status:z.enum(["pending","running","completed","partial","failed","suspended"]),orderIds:ids.min(1),missingIds:ids,notes:z.array(z.string().max(64)).max(20),startedAt:z.iso.datetime({offset:true}),observedAt:z.iso.datetime({offset:true}).nullable()}).strict().optional(),
 summary:z.object({orderCount:count,motherCount:count,unknownTypeCount:count,quantities:z.object({ordered:quantity,delivered:quantity,received:quantity,stored:quantity,defective:quantity}).strict()}).strict().optional(),
}).strict().superRefine((v,c)=>{
 if(new Set(v.savedOrderIds).size!==v.savedOrderIds.length)c.addIssue({code:"custom",message:"Duplicate saved identity"});
 if(v.source){const requested=new Set(v.source.orderIds),missing=new Set(v.source.missingIds);if(requested.size!==v.source.orderIds.length||missing.size!==v.source.missingIds.length||v.savedOrderIds.some(id=>!requested.has(id)||missing.has(id))||v.source.missingIds.some(id=>!requested.has(id)))c.addIssue({code:"custom",message:"Batch identity mismatch"})}
 if(v.status==="saved"&&(!v.source||!v.summary||v.source.status!=="completed"||v.source.missingIds.length||v.savedOrderIds.length!==v.source.orderIds.length))c.addIssue({code:"custom",message:"Unconfirmed completeness"});
 if(v.summary&&v.summary.orderCount+v.summary.motherCount+v.summary.unknownTypeCount!==v.savedOrderIds.length)c.addIssue({code:"custom",message:"Classification mismatch"});
});
