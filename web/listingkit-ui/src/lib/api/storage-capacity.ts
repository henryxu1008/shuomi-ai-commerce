import { z } from "zod";
import {positiveInt64, request} from "./resource-purchase";
import {parseCommercialOrder} from "./commercial-billing";
const id=z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/);
const number=z.string().refine(v=>v==="0"||positiveInt64(v));
const timestamp=z.string().max(40).regex(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,9})?Z$/).refine(v=>Number.isFinite(Date.parse(v)));
const lot=z.object({order_id:id,bytes:number,activated_at:timestamp,expires_at:timestamp}).strict().refine(v=>Date.parse(v.expires_at)-Date.parse(v.activated_at)===2592000000);
const summary=z.object({more_lots:z.boolean(),base_bytes:z.literal("5368709120"),quota_bytes:number,occupied_bytes:number,reserved_bytes:number,remaining_bytes:number,overage_bytes:number,categories:z.object({product_media:number.optional(),ai_image:number.optional(),knowledge_document:number.optional(),business_attachment:number.optional()}).strict(),active_lots:z.array(lot).max(50),observed_at:timestamp}).strict().refine(v=>{
  if (![v.quota_bytes,v.occupied_bytes,v.reserved_bytes,v.remaining_bytes,v.overage_bytes,...Object.values(v.categories)].every(n=>n==="0"||positiveInt64(n))) return false;
  const q=BigInt(v.quota_bytes),o=BigInt(v.occupied_bytes),r=BigInt(v.reserved_bytes);
  return q>=BigInt(v.base_bytes)&&BigInt(v.remaining_bytes)===(q>o+r?q-o-r:BigInt(0))&&BigInt(v.overage_bytes)===(o>q?o-q:BigInt(0))&&Object.values(v.categories).reduce((a,b)=>a+BigInt(b??"0"),BigInt(0))===o;
});
const response=z.object({organization_id:id,summary}).strict();
const offer=z.object({offer_id:id,product_kind:z.literal("STORAGE_CAPACITY_PACK"),currency:z.literal("CNY"),unit_price_minor:z.string().refine(positiveInt64),min_quantity:z.literal("1"),max_quantity:z.literal("10"),storage_unit_bytes:z.literal("53687091200"),storage_duration_seconds:z.literal("2592000"),pricing_version:z.string().min(1).max(128)}).strict();
const offers=z.object({schema_version:z.literal("storage-price-catalog-v1"),organization_id:id,items:z.array(offer).max(16)}).strict();
const quote=z.object({quote_id:id,organization_id:id,offer_id:id,product_kind:z.literal("STORAGE_CAPACITY_PACK"),resource_type:z.literal(""),resource_quantity:z.string().refine(v=>/^[1-9]$|^10$/.test(v)),currency:z.literal("CNY"),total_minor:z.string().refine(positiveInt64),pricing_version:z.string().min(1).max(128),expires_at:timestamp,fingerprint:z.string().regex(/^[a-f0-9]{64}$/),created_at:timestamp,storage_unit_bytes:z.literal("53687091200"),storage_duration_seconds:z.literal("2592000")}).strict();
export type StorageSummary=z.infer<typeof summary>;
export type StorageQuote=z.infer<typeof quote>;
export const parseStorageSummary=(v:unknown)=>{const p=response.safeParse(v);return p.success?p.data:null};
export const parseStorageOffers=(v:unknown)=>{const p=offers.safeParse(v);return p.success?p.data:null};
export function getStorageSummary(user:string,org:string,signal?:AbortSignal){return request("/api/workbench/storage/summary",user,org,v=>{const p=parseStorageSummary(v);return p?.organization_id===org?p:null},{signal})}
export function getStorageOffers(user:string,org:string,signal?:AbortSignal){return request("/api/workbench/commercial/storage-offers",user,org,v=>{const p=parseStorageOffers(v);return p?.organization_id===org?p:null},{signal})}
export function createStorageQuote(user:string,org:string,offerId:string,quantity:string,signal?:AbortSignal){return request("/api/workbench/commercial/quotes",user,org,v=>{const p=quote.safeParse(v);return p.success&&p.data.organization_id===org&&p.data.offer_id===offerId&&p.data.resource_quantity===quantity?p.data:null},{body:{offer_id:offerId,quantity},signal})}
export function createStorageOrder(user:string,org:string,quoteId:string,key:string,signal?:AbortSignal){return request("/api/workbench/commercial/orders",user,org,v=>{const p=parseCommercialOrder(v);return p?.kind==="STORAGE_CAPACITY_PURCHASE"&&p.organization_id===org&&p.quote_id===quoteId?p:null},{body:{quote_id:quoteId},idempotencyKey:key,signal})}
export function readStorageOrder(user:string,org:string,id:string,signal?:AbortSignal){return request(`/api/workbench/commercial/orders/${encodeURIComponent(id)}`,user,org,v=>{const p=parseCommercialOrder(v);return p?.kind==="STORAGE_CAPACITY_PURCHASE"&&p.organization_id===org&&p.order_id===id?p:null},{signal})}
