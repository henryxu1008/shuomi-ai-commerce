import {z} from "zod";
import {temuIntegerSchema,temuModeSchema} from "./temu-model";

const id=temuIntegerSchema.refine(v=>BigInt(v)>0);
const text=z.string().max(2*1024*1024).refine(v=>!v.includes("\0"));
const image=text.refine(v=>{if(!v)return true;try{const u=new URL(v);return u.protocol==="https:"&&!u.username&&!u.password&&!u.hash}catch{return false}});
export const temuObservedProductSchema=z.object({
 id,productId:id,entity:z.enum(["local_goods","supplier_skc"]),mode:temuModeSchema,title:text,externalCode:text,imageUrl:image,saleState:z.enum(["active","off","unknown"]),rawStatus:text,
 createdAt:temuIntegerSchema.optional(),statusChangedAt:temuIntegerSchema.optional(),skuCount:temuIntegerSchema.optional(),selectionStatus:temuIntegerSchema.optional(),
 skus:z.array(z.object({id,externalCode:text,serialNumber:text,stock:temuIntegerSchema.optional(),stockKind:z.enum(["stock","virtualStock"]),specifications:z.array(z.object({id:z.union([id,z.literal("")]),parentId:z.union([id,z.literal("")]),name:text,parentName:text}).strict()).max(256),applyJitStatus:temuIntegerSchema.optional(),suggestCloseJit:z.boolean().optional()}).strict()).max(1000),
 sites:z.array(z.object({id,name:text}).strict()).max(100),publishStatus:z.object({status:temuIntegerSchema.nullable(),subStatus:temuIntegerSchema.nullable()}).strict().optional()
}).strict().refine(v=>v.mode==="local"?v.entity==="local_goods"&&v.id===v.productId&&v.sites.length===0:v.entity==="supplier_skc"&&!v.publishStatus);
export type TemuObservedProduct=z.infer<typeof temuObservedProductSchema>;
