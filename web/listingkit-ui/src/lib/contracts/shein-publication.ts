import {z} from "zod";
import {collectionID} from "./product-collection";
const bounded=(n:number)=>z.string().max(n).refine(v=>!v.includes("\0"));
const time=z.iso.datetime({offset:true});
const count=z.number().int().min(0).max(Number.MAX_SAFE_INTEGER);
const state=z.number().int().nullable();
const mode=z.enum(["self_operated","semi_managed","fully_managed"]);
const status=z.enum(["claimed","outcome_unknown","succeeded","failed_definitive","cancelled"]);
export const sheinShelfInputSchema=z.object({sourceId:collectionID,storeId:collectionID,receiptId:collectionID,skc:bounded(128).min(1)}).strict();
export const sheinStatusInputSchema=sheinShelfInputSchema.omit({skc:true}).extend({stockOffset:count.max(15900).multipleOf(100).optional()});
export const sheinShelfConfirmInputSchema=sheinShelfInputSchema.extend({token:collectionID});
export const sheinShelfResultInputSchema=sheinShelfConfirmInputSchema.extend({key:collectionID});
export const sheinPublicationStatusSchema=z.object({
 sourceId:collectionID,storeId:collectionID,receiptId:collectionID,spu:bounded(128).min(1),version:bounded(128),mode,observedAt:time,
 auditAt:time.optional(),liveAt:time.optional(),stockAt:time.optional(),auditError:bounded(512).optional(),liveError:bounded(512).optional(),stockError:bounded(512).optional(),
 stockType:z.enum(["VI","PI","JI"]),stockOffset:count.max(15900),stockTotal:count.max(16000),nextStockOffset:count.max(15900).optional(),stockComplete:z.boolean(),
 skcs:z.array(z.object({skc:bounded(128).min(1),auditState:state,auditLabel:bounded(128),reasons:z.array(z.object({language:bounded(32),content:bounded(2048)})).max(20).nullable().transform(v=>v??[]),shelfState:state,shelfWay:bounded(16),hopeOnSaleDate:bounded(64),shelfRequire:bounded(16),canShelf:z.boolean(),blockingReason:bounded(512),
 attempt:z.object({id:collectionID,key:collectionID,status}).optional(),
 skus:z.array(z.object({code:bounded(128).min(1),mallState:state,usable:count.nullable(),reportedUsable:count.nullable(),inventoryScopeKnown:z.boolean()}).refine(v=>v.inventoryScopeKnown===(v.usable!==null))).max(400)
 })).max(40)
}).refine(v=>v.mode!=="fully_managed"||v.skcs.every(s=>!s.canShelf));
export const sheinShelfConfirmationSchema=z.object({key:collectionID,sourceId:collectionID,storeId:collectionID,receiptId:collectionID,skc:bounded(128).min(1),spu:bounded(128).min(1),version:bounded(128).min(1),site:z.literal("shein-us"),mode:mode.exclude(["fully_managed"]),token:collectionID,expiresAt:time,shelfWay:bounded(16),hopeOnSaleDate:bounded(64)}).strict();
export const sheinShelfResponseSchema=z.object({code:bounded(128),message:bounded(512),responseHash:z.string().regex(/^[a-f0-9]{64}$/),failures:z.array(z.object({identity:bounded(128),code:bounded(128),message:bounded(512)})).max(20)});
export const sheinShelfActionSchema=z.object({key:collectionID,attemptId:collectionID,sourceId:collectionID,storeId:collectionID,receiptId:collectionID,skc:bounded(128).min(1),status,message:bounded(512),response:sheinShelfResponseSchema.optional(),observation:sheinPublicationStatusSchema.optional()}).refine(v=>!v.observation||v.observation.receiptId===v.receiptId&&v.observation.sourceId===v.sourceId&&v.observation.storeId===v.storeId);
export function sheinRequestSchema(route:string){
 return route==="shein-status"?sheinStatusInputSchema:route==="shein-shelf-prepare"?sheinShelfInputSchema:route==="shein-shelf-confirm"?sheinShelfConfirmInputSchema:route==="shein-shelf-result"?sheinShelfResultInputSchema:null;
}
export function sheinResponseSchema(route:string):z.ZodType{
 return route==="shein-status"?sheinPublicationStatusSchema:route==="shein-shelf-prepare"?sheinShelfConfirmationSchema:sheinShelfActionSchema;
}
export function matchesSheinResponse(route:string,value:unknown,expected:string):boolean{
 try{const input=JSON.parse(expected),v=sheinResponseSchema(route).parse(value) as {sourceId:string;storeId:string;receiptId:string;skc?:string;key?:string};return v.sourceId===input.sourceId&&v.storeId===input.storeId&&v.receiptId===input.receiptId&&(route==="shein-status"||v.skc===input.skc&&(!input.key||v.key===input.key));}catch{return false}
}
export type SheinPublicationStatus=z.infer<typeof sheinPublicationStatusSchema>;
export type SheinShelfConfirmation=z.infer<typeof sheinShelfConfirmationSchema>;
export type SheinShelfInput=z.infer<typeof sheinShelfInputSchema>;

