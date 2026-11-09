import {z} from "zod";
import {observationID} from "@/lib/api/store-observations";

export const supplyTimeLabels={addTime:"创建时间",allocateTime:"分单／下发时间",updateTime:"状态变更时间",requestDeliveryTime:"要求发货时间",requestReceiptTime:"要求收货时间",requestTakeParcelTime:"要求取件时间",reserveTime:"预约时间",receiptTime:"收货时间",checkTime:"验货时间",storageTime:"入库时间",returnTime:"退货时间",requestCompleteTime:"要求完成时间",deliveryTime:"发货时间"};
const rawText=z.string().refine(v=>new TextEncoder().encode(v).length<=256&&!v.includes("\0")&&v.isWellFormed());
const value=z.object({state:z.enum(["missing","null","empty","local_datetime","offset_datetime","unrecognized","unexpected_type"]),rawType:z.enum(["missing","null","string","number","boolean"]),raw:rawText.nullable()}).strict().superRefine((v,c)=>{
 let valid=true;
 if(v.rawType==="missing"||v.rawType==="null")valid=v.raw===null&&v.state===v.rawType;
 else if(v.raw===null)valid=false;
 else if(v.rawType==="string")valid=v.raw===""?v.state==="empty":["local_datetime","offset_datetime","unrecognized"].includes(v.state);
 else if(v.rawType==="boolean")valid=v.state==="unexpected_type"&&["true","false"].includes(v.raw);
 else valid=v.state==="unexpected_type"&&/^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?$/.test(v.raw);
 if(!valid)c.addIssue({code:"custom",message:"Inconsistent raw time metadata"});
});
const times=z.object({contract:z.literal("shein-supply-business-times/v1"),source:z.literal("/open-api/order/purchase-order-infos"),timezone:z.literal("unconfirmed"),values:z.object(Object.fromEntries(Object.keys(supplyTimeLabels).map(k=>[k,value])) as Record<keyof typeof supplyTimeLabels,typeof value>).strict()}).strict();
const qty=z.number().int().min(0).max(100000000).nullable();
const code=z.number().int().min(0).max(1000000).nullable();
const identity=z.string().min(1).max(128).refine(v=>v.trim()===v&&!/[\p{Cc},/\\?#]/u.test(v));
export const supplyTimeDetailSchema=z.object({storeId:observationID,syncId:observationID,id:identity,observedAt:z.iso.datetime({offset:true}),stale:z.literal(false).optional(),supplyOrder:z.object({id:identity,type:code,status:code,scene:code,countryMarket:code,mother:z.boolean().nullable(),motherRaw:z.string().max(24),lines:z.array(z.object({sku:z.string().min(1).max(128),ordered:qty,delivered:qty,received:qty,stored:qty,defective:qty}).strict()).max(1000),businessTimes:times}).strict()}).strict().refine(v=>v.id===v.supplyOrder.id);
