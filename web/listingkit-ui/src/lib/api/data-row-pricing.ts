import {z} from "zod";
import {request,resourceOfferSchema} from "./resource-purchase";
export const dataRowPriceInput=z.object({expected_version:z.string().min(1).max(128),unit_price_minor:z.string().regex(/^(0|[1-9][0-9]{0,18})$/).refine(v=>BigInt(v)<=BigInt("9223372036854775807")),reason:z.string().trim().min(1).max(500)}).strict();
export const dataRowPriceOffer=resourceOfferSchema.refine(v=>v.product_kind==="DATA_ROW");
export const dataRowPriceReceipt=z.object({id:z.string().min(1),actor_id:z.string().min(1),organization_id:z.string().min(1),previous_offer_id:z.string().min(1),previous_version:z.string().min(1),offer:dataRowPriceOffer,reason:z.string().max(500),created_at:z.string().datetime()}).strict();
export const dataRowPriceHistory=z.object({items:z.array(dataRowPriceReceipt).max(50),next_cursor:z.string().max(512)}).strict();
export type PriceScope={userId:string;organizationId:string};
export function dataRowPriceRequest<T>(scope:PriceScope,path:string,schema:z.ZodType<T>,options:{body?:object;idempotencyKey?:string;signal?:AbortSignal}={}){
 return request(`/api/workbench/commercial/data-row-pricing${path}`,scope.userId,scope.organizationId,v=>{const r=schema.safeParse(v);return r.success?r.data:null},options);
}
