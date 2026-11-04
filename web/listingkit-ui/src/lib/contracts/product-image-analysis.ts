import {z} from "zod";
import {aiPointUnit,micropoints,positiveMicropoints} from "./ai-points";
import {isAcquisitionUUID} from "./product-acquisition";
import {catalogVersionSchema} from "./supply-chain";
import {modeRevisionSchema} from "./execution-mode";

const uuid=z.string().refine(isAcquisitionUUID),id=z.string().min(1).max(192),hash=z.string().regex(/^[a-f0-9]{64}$/);
const contextKind=z.enum(["acquisition","collection","own_product"]),kind=z.enum(["source","approved"]);
const reason=z.string().min(1).refine(s=>new TextEncoder().encode(s).length<=512&&s.trim()===s&&!/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/.test(s));
const area=z.enum(["subject","framing","background","clarity"]);
const analysisOpinionSchema=z.strictObject({observations:z.array(z.strictObject({area,status:z.enum(["possible_issue","no_obvious_issue","unknown"]),reason})).max(8),suggestions:z.array(z.strictObject({area,action:z.enum(["check","change_image","open_image_workflow"]),reason})).max(8),unknowns:z.array(reason).max(8)}).refine(v=>v.observations.length+v.unknowns.length>0);
export const analysisSourcesSchema=z.strictObject({contextKind,contextId:uuid,productKey:id,catalogVersion:catalogVersionSchema,publicationId:id,ready:z.boolean(),reason:z.string().max(192),inventoryStatus:z.enum(["READY","NOT_READY","UNKNOWN"]),inventoryDigest:z.union([hash,z.literal("")]),choices:z.array(z.strictObject({kind,id,url:z.string().max(4096).url().refine(s=>/^https?:\/\//.test(s))})).max(4096)}).refine(v=>!v.choices.some(c=>c.kind==="approved")||v.inventoryStatus==="READY"&&!!v.inventoryDigest);
export const analysisResultSchema=z.strictObject({contextKind,contextId:uuid,requestKey:uuid,catalogVersion:catalogVersionSchema,publicationId:id,unitVersion:aiPointUnit,status:z.enum(["prepared","unknown","not_sent","failed","completed"]),stopReason:z.string().max(192),runId:z.union([uuid,z.literal("")]),kind,imageId:id,modelId:id,preparationId:uuid,configurationDigest:hash,quoteDigest:hash,selectionDigest:hash,contentSha256:hash,maximumMicropoints:positiveMicropoints,actualMicropoints:z.union([micropoints,z.literal("")]),width:z.number().int().min(1).max(8192),height:z.number().int().min(1).max(8192),opinion:analysisOpinionSchema.optional()}).refine(v=>v.width*v.height<=16777216&&(!v.actualMicropoints||BigInt(v.actualMicropoints)<=BigInt(v.maximumMicropoints))&&(v.status==="completed"?!!v.opinion&&!!v.runId&&v.actualMicropoints!=="":!v.opinion)&&(v.status!=="not_sent"||v.actualMicropoints==="0"));
export const analysisRecentSchema=z.strictObject({contextKind,contextId:uuid,items:z.array(z.strictObject({runId:uuid,operationId:uuid,contextKind,requestKey:uuid,productKey:id,targetPlatform:z.literal("product"),phase:z.enum(["running","interrupted","human_review_required","stopped"]),revision:catalogVersionSchema,startedAt:z.string().datetime({offset:true})})).max(20),nextCursor:z.union([uuid,z.literal("")])}).refine(v=>v.items.every(i=>i.contextKind===v.contextKind&&i.operationId===v.contextId));
export type AnalysisSources=z.infer<typeof analysisSourcesSchema>;
export type AnalysisResult=z.infer<typeof analysisResultSchema>;
export type AnalysisRecent=z.infer<typeof analysisRecentSchema>;
export const analysisRoutes=["sources","prepare","read","confirm","recent"] as const;
export type AnalysisRoute=typeof analysisRoutes[number];
export const analysisMethods:Record<AnalysisRoute,"GET"|"POST">={sources:"GET",prepare:"POST",read:"GET",confirm:"POST",recent:"GET"};
export function analysisRequestSchema(action:AnalysisRoute):z.ZodType {
 if(action==="prepare")return z.strictObject({kind,imageId:id,catalogVersion:catalogVersionSchema,publicationId:id,inventoryDigest:hash.optional(),expectedModeRevision:modeRevisionSchema}).refine(v=>v.kind==="approved"?!!v.inventoryDigest:!v.inventoryDigest);
 if(action==="confirm")return z.strictObject({confirmed:z.literal(true),preparationId:uuid,configurationDigest:hash,quoteDigest:hash,selectionDigest:hash,contentSha256:hash,modelId:id,maximumMicropoints:positiveMicropoints});
 return z.never();
}
export function analysisResponseSchema(action:AnalysisRoute):z.ZodType{return action==="sources"?analysisSourcesSchema:action==="recent"?analysisRecentSchema:analysisResultSchema}
export const analysisBindingSchema=z.strictObject({contextKind,contextId:uuid,requestKey:uuid.optional()});
export type AnalysisBinding=z.infer<typeof analysisBindingSchema>;
export function analysisBinding(path:string[],requestKey?:string):AnalysisBinding{
 const acquisition=path[0]==="sourcing",own=path[0]==="collections",rest=path.slice(acquisition?6:5);
 return {contextKind:acquisition?"acquisition":own?"own_product":"collection",contextId:path[acquisition?3:2],...rest[0]==="by-key"?{requestKey:rest[1]}:requestKey?{requestKey}:{}};
}
export function analysisPath(path:string[],action:AnalysisRoute):string|null{
 const acquisition=path[0]==="sourcing"&&path[1]==="1688"&&path[2]==="acquisitions"&&isAcquisitionUUID(path[3]??"");
 const source=(path[0]==="supply-preparations"&&path[1]==="sources"||path[0]==="collections"&&path[1]==="items")&&isAcquisitionUUID(path[2]??"");
 const offset=acquisition?4:source?3:-1;if(offset<0||path[offset]!=="images"||path[offset+1]!=="analysis")return null;
 const rest=path.slice(offset+2),match=action==="sources"?rest.length===0:action==="prepare"?rest.length===1&&rest[0]==="prepare":action==="recent"?rest.length===1&&rest[0]==="runs":rest.length===(action==="confirm"?3:2)&&rest[0]==="by-key"&&isAcquisitionUUID(rest[1]??"")&&(action!=="confirm"||rest[2]==="confirm");
 return match?path.join("/"):null;
}
export function analysisMatches(data:unknown,binding:AnalysisBinding):boolean{
 if(!data||typeof data!=="object")return false;const v=data as Record<string,unknown>;
 return v.contextKind===binding.contextKind&&v.contextId===binding.contextId&&(!binding.requestKey||v.requestKey===binding.requestKey);
}
