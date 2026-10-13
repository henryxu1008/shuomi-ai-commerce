import {z} from "zod";
import {collectionID} from "@/lib/contracts/product-collection";
import {materialsPrepareSchema,materialsPreparedSchema,materialsResumeSchema,materialsAdoptSchema} from "@/lib/contracts/listing-materials";
const identity=z.string().min(1).max(128);
export const materialsIntentSchema=z.strictObject({recordId:collectionID,userId:identity,organizationId:identity,key:collectionID,prepare:materialsPrepareSchema,prepared:materialsPreparedSchema.optional()}).superRefine((v,ctx)=>{
 if(v.key!==v.prepare.requestKey||v.prepared&&(v.prepared.recordId!==v.recordId||v.prepared.requestKey!==v.key))ctx.addIssue({code:"custom",message:"Original materials intent binding differs"});
});
export type MaterialsIntent=z.infer<typeof materialsIntentSchema>;
export function readMaterialsIntent(raw:string|null):MaterialsIntent|null {
 if(!raw||new TextEncoder().encode(raw).length>8192)return null;
 try{const value=materialsIntentSchema.safeParse(JSON.parse(raw));return value.success?value.data:null}catch{return null}
}
export function storeMaterialsIntent(intent:MaterialsIntent){
 const exact=materialsIntentSchema.parse(intent),url=new URL(window.location.href);
 url.searchParams.set("materials_intent",JSON.stringify(exact));
 url.searchParams.set("agent_key",exact.key);url.searchParams.set("agent_actor",exact.userId);url.searchParams.set("agent_org",exact.organizationId);
 window.history.replaceState(null,"",url);
}
const mutationBinding={recordId:collectionID,userId:identity,organizationId:identity,key:collectionID};
export const materialsMutationSchema=z.discriminatedUnion("action",[
 z.strictObject({...mutationBinding,action:z.literal("resume"),input:materialsResumeSchema}),
 z.strictObject({...mutationBinding,action:z.literal("adopt"),input:materialsAdoptSchema}),
]);
export type MaterialsMutation=z.infer<typeof materialsMutationSchema>;
export function readMaterialsMutation(raw:string|null):MaterialsMutation|null {
 if(!raw||new TextEncoder().encode(raw).length>8192)return null;
 try{const value=materialsMutationSchema.safeParse(JSON.parse(raw));return value.success?value.data:null}catch{return null}
}
export function storeMaterialsMutation(value:MaterialsMutation|null){
 const url=new URL(window.location.href);
 if(value)url.searchParams.set("materials_mutation",JSON.stringify(materialsMutationSchema.parse(value)));else url.searchParams.delete("materials_mutation");
 window.history.replaceState(null,"",url);
}
