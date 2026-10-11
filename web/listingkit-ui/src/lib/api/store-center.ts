import {z} from "zod";
import {workbenchStoreSchema} from "./workbench-stores";
import {workbenchStorePlatformSchema} from "@/lib/validation/workbench-store";
import {readBoundedStrictJSON} from "./strict-json-response";

export const storeCenterID=z.string().uuid().refine(value=>value!=="00000000-0000-0000-0000-000000000000" && value===value.toLowerCase());
const identity=z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/);
const integer=z.number().int().min(0).max(Number.MAX_SAFE_INTEGER);
const name=(max:number)=>z.string().refine(value=>value===value.trim() && Array.from(value).length>0 && Array.from(value).length<=max && !/\p{Cc}/u.test(value));
const application=z.enum(["self_operated","semi_managed","fully_managed"]);
export const profileSchema=z.object({storeId:storeCenterID,groupId:storeCenterID.nullable(),responsibleMemberId:identity.nullable(),lowStockThreshold:integer.nullable(),profileVersion:integer,updatedAt:z.iso.datetime({offset:true}).nullable()}).strict();
const groupSchema=z.object({id:storeCenterID,name:name(60),groupVersion:integer.min(1),updatedAt:z.iso.datetime({offset:true})}).strict();
const metricSchema=z.object({value:integer.nullable(),unknown:integer,complete:z.boolean(),reason:z.string().max(64)}).strict();
const signalsSchema=z.object({connectionStatus:z.enum(["disconnected","connected","expired","unavailable"]),applicationType:z.union([application,z.literal("")]),products:metricSchema,orders:metricSchema,inventory:metricSchema,productSync:z.string().max(32),orderSync:z.string().max(32),syncHealthy:z.boolean(),observedAt:z.iso.datetime({offset:true}).nullable(),reason:z.string().max(64)}).strict();
export const catalogFiltersSchema=z.object({page:integer.min(1),pageSize:integer.min(1).max(100),q:z.string().refine(value=>value===value.trim() && new TextEncoder().encode(value).length<=200 && !/\p{Cc}/u.test(value)).optional(),platform:workbenchStorePlatformSchema.optional(),region:name(64).optional(),applicationType:application.optional(),groupId:z.union([storeCenterID,z.literal("ungrouped")]).optional(),attention:z.enum(["normal","attention","unknown"]).optional(),status:z.enum(["active","disabled","deleting"]).optional()}).strict();
export const catalogItemSchema=workbenchStoreSchema.extend({profile:profileSchema,signals:signalsSchema,attention:z.enum(["normal","attention","unknown"])}).refine(item=>item.id===item.profile.storeId && item.connectionStatus===item.signals.connectionStatus);
export const catalogSchema=z.object({items:z.array(catalogItemSchema).max(100),pagination:z.object({page:integer.min(1),pageSize:integer.min(1).max(100),total:integer}).strict(),metrics:z.object({total:integer,normal:integer.nullable(),attention:integer.nullable(),syncHealthy:integer.nullable(),unknown:integer,complete:z.boolean(),observedAt:z.iso.datetime({offset:true})}).strict(),facets:z.object({regions:z.array(name(64)),applicationTypes:z.array(application),groups:z.array(groupSchema.extend({count:integer})),canManageGroups:z.boolean()}).strict()}).strict().refine(result=>result.items.length<=result.pagination.pageSize && result.items.length<=result.pagination.total && result.metrics.total===result.pagination.total && new Set(result.items.map(item=>item.id)).size===result.items.length);
export const metadataCommandSchema=z.object({action:z.enum(["create_group","rename_group","delete_group","set_profile","assign_group"]),name:z.string().max(240),groupId:storeCenterID.nullable(),groupVersion:integer,stores:z.array(z.object({storeId:storeCenterID,profileVersion:integer}).strict()).max(100),responsibleMemberId:identity.nullable(),lowStockThreshold:integer.nullable()}).strict().superRefine((command,ctx)=>{
 const issue=()=>ctx.addIssue({code:"custom",message:"Invalid Store metadata command"});
 if(new Set(command.stores.map(item=>item.storeId)).size!==command.stores.length) issue();
 if(["create_group","rename_group","delete_group"].includes(command.action)) {
  if(command.stores.length || command.responsibleMemberId!==null || command.lowStockThreshold!==null) issue();
  if(command.action==="create_group" ? command.groupId!==null || command.groupVersion!==0 : command.groupId===null || command.groupVersion<1) issue();
  if(command.action==="delete_group" ? command.name!=="" : !name(60).safeParse(command.name).success) issue();
 } else {
  if(!command.stores.length || command.groupVersion!==0 || command.name!=="" || command.action==="set_profile" && command.stores.length!==1 || command.action==="assign_group" && (command.responsibleMemberId!==null || command.lowStockThreshold!==null)) issue();
 }
});
export const metadataResultSchema=z.object({operationId:storeCenterID,profiles:z.array(profileSchema).max(100),group:groupSchema.nullable(),replayed:z.boolean()}).strict();
export const groupsSchema=z.object({items:z.array(groupSchema)}).strict();
export const storeCenterEnvelope=<T>(schema:z.ZodType<T>)=>z.object({schemaVersion:z.literal("store-center-v1"),organizationId:identity,userId:identity,data:schema}).strict();
export type CatalogFilters=z.infer<typeof catalogFiltersSchema>;
export type CatalogItem=z.infer<typeof catalogItemSchema>;
export type Group=z.infer<typeof groupSchema>;
export type MetadataCommand=z.infer<typeof metadataCommandSchema>;
export type StoreCenterScope={organizationId:string;userId:string};
export class StoreCenterError extends Error {constructor(public status:number,public code:string){super(code)}}
export async function storeCenterRequest<T>(path:string,scope:StoreCenterScope,schema:z.ZodType<T>,options:{command?:MetadataCommand;key?:string;signal?:AbortSignal}={}):Promise<T> {
 identity.parse(scope.organizationId);identity.parse(scope.userId);
 const headers=new Headers({Accept:"application/json","X-Expected-Organization-ID":scope.organizationId,"X-Expected-User-ID":scope.userId});
 let body:string|undefined;
 if(options.command) {headers.set("Content-Type","application/json");headers.set("Idempotency-Key",storeCenterID.parse(options.key));body=JSON.stringify(metadataCommandSchema.parse(options.command));}
 const response=await fetch(`/api/workbench/store-center/${path}`,{method:body?"POST":"GET",headers,body,signal:options.signal,credentials:"same-origin",redirect:"error",cache:"no-store"});
 const raw=await readBoundedStrictJSON(response,2<<20,options.signal);
 if(!response.ok) {const error=z.object({code:z.string().max(80)}).passthrough().safeParse(raw);throw new StoreCenterError(response.status,error.success?error.data.code:"INVALID_UPSTREAM_RESPONSE")}
 const result=storeCenterEnvelope(schema).safeParse(raw);
 if(!result.success) throw new StoreCenterError(502,"INVALID_UPSTREAM_RESPONSE");
 if(result.data.organizationId!==scope.organizationId || result.data.userId!==scope.userId) throw new StoreCenterError(409,"ORGANIZATION_CONTEXT_CHANGED");
 return schema.parse(result.data.data);
}
export function catalogSearch(filters:CatalogFilters) {const search=new URLSearchParams(); for(const[key,value]of Object.entries(catalogFiltersSchema.parse(filters)))if(value!==undefined)search.set(key,String(value));return search.toString()}
