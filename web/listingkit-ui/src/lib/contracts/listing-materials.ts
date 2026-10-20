import {z} from "zod";
import {collectionID} from "./product-collection";
import {templateRefSchema,configVersion} from "./agent-configuration";
import {executionModeSchema,modeRevisionSchema,modeSnapshotRefSchema} from "./execution-mode";
import {isAcquisitionUUID} from "./product-acquisition";

const hash=z.string().regex(/^[0-9a-f]{64}$/);
const text=(limit:number)=>z.string().refine(v=>new TextEncoder().encode(v).length<=limit);
const summary=text(512).refine(v=>v.trim()===v);
const bounded=z.number().int().min(0).max(Number.MAX_SAFE_INTEGER);
const materialsConfigurationRefSchema=z.strictObject({Kind:z.literal("agent-configuration-v1"),ID:collectionID,Digest:hash});
export const materialsPrepareSchema=z.strictObject({requestKey:collectionID,expectedModeRevision:modeRevisionSchema,template:templateRefSchema.optional(),goalSummary:summary.optional()});
export const materialsStartSchema=z.strictObject({configurationSnapshotRef:materialsConfigurationRefSchema});
export const materialsResumeSchema=materialsStartSchema.extend({revision:configVersion,feedback:summary});
export const materialsAdoptSchema=materialsStartSchema.extend({candidateHash:hash,humanConfirmed:z.boolean()});
export const materialsPreparedSchema=z.strictObject({recordId:collectionID,requestKey:collectionID,configurationSnapshotRef:materialsConfigurationRefSchema,modeSnapshotRef:modeSnapshotRefSchema,modeRevision:modeRevisionSchema,templateSelection:templateRefSchema.nullable()});

// Compose the exact existing Target / official attribute / image contracts.
// This module does not duplicate their owner schema or import its consumer.
export function listingMaterialsSchemas<I extends z.ZodType,R extends z.ZodType,A extends z.ZodType,M extends z.ZodType>(input:I,receipt:R,attribute:A,image:M){
 const attrs=z.array(attribute).max(1000);
 const proposal=z.strictObject({categoryId:bounded.min(1).optional(),brandCode:text(128).min(1).optional(),attributes:attrs.optional(),images:z.array(image).max(10000).optional(),skcs:z.array(z.strictObject({index:bounded.max(39),saleAttribute:attribute.optional(),skus:z.array(z.strictObject({index:bounded.max(999),saleAttributes:attrs.optional(),attributes:attrs.optional()})).max(1000).optional()})).max(40).optional(),evidenceIds:z.array(text(128).min(1)).min(1).max(64)}).refine(p=>p.categoryId!==undefined||p.brandCode!==undefined||p.attributes!==undefined||p.images!==undefined||!!p.skcs?.length);
 const run=z.strictObject({runId:collectionID,requestKey:collectionID,recordId:collectionID,phase:z.enum(["running","interrupted","human_review_required","stopped"]),revision:configVersion,stopReason:text(128).optional(),configurationSnapshotRef:materialsConfigurationRefSchema,modeSnapshotRef:modeSnapshotRefSchema,executionMode:executionModeSchema,modeRevision:modeRevisionSchema,templateSelection:templateRefSchema.optional(),proposal:proposal.optional(),candidateHash:hash.optional(),before:input,after:input.optional(),unresolved:z.array(text(8192)).max(4096),canRequestAdoption:z.boolean(),adoption:receipt.optional(),steps:z.array(z.strictObject({step:bounded.max(16),tool:text(128).optional(),callId:text(128).optional(),invocationId:text(128).optional(),auditStatus:z.enum(["recorded","record_failed"]).optional()})).max(32),tokens:bounded,estimatedCostMicros:bounded,currency:text(16),usageStatus:z.enum(["observed","unknown_reserved"])}).superRefine((v,ctx)=>{
  if(v.canRequestAdoption && (v.executionMode==="read_only"||v.phase!=="human_review_required"||v.stopReason||!v.proposal||!v.after||!v.candidateHash||v.adoption))ctx.addIssue({code:"custom",message:"Adoption must bind the exact reviewable run"});
 });
 return {proposal,run};
}
export type MaterialsRoute="materials-prepare"|"materials-start"|"materials-read"|"materials-resume"|"materials-adopt"|"materials-adoption";
export const materialsRouteMethods:ReadonlyArray<readonly["GET"|"POST",MaterialsRoute]>=[["POST","materials-prepare"],["POST","materials-start"],["GET","materials-read"],["POST","materials-resume"],["POST","materials-adopt"],["GET","materials-adoption"]];
export function materialsPath(method:string,p:string[]):MaterialsRoute|null{
 if(p[0]!=="supply-preparations"||p[1]!=="records"||!isAcquisitionUUID(p[2]??"")||p[3]!=="materials-agent")return null;
 if(method==="POST"&&p.length===5)return p[4]==="prepare"?"materials-prepare":p[4]==="runs"?"materials-start":null;
 if(p[4]!=="runs"||!isAcquisitionUUID(p[5]??""))return null;
 if(method==="GET")return p.length===6?"materials-read":p.length===7&&p[6]==="adoption"?"materials-adoption":null;
 if(method==="POST"&&p.length===7)return p[6]==="resume"?"materials-resume":p[6]==="adopt"?"materials-adopt":null;
 return null;
}
export const materialsRequestSchema=(action:MaterialsRoute)=>action==="materials-prepare"?materialsPrepareSchema:action==="materials-start"?materialsStartSchema:action==="materials-resume"?materialsResumeSchema:action==="materials-adopt"?materialsAdoptSchema:null;
export const materialsErrorStatuses:Record<string,number>={INVALID_REQUEST:400,PERMISSION_DENIED:403,NOT_FOUND:404,HUMAN_CONFIRMATION_REQUIRED:409,OUTCOME_UNKNOWN:409,AGENT_CONFLICT:409,NOT_READY:409,CONFIGURATION_CHANGED:409,AGENT_NOT_ENABLED:409,TEMPLATE_ARCHIVED:409,REVISION_MISMATCH:409,PRECONDITION_REQUIRED:409,AGENT_DEFINITION_UNAVAILABLE:409,INPUT_TOO_LARGE:413,DEPENDENCY_UNAVAILABLE:503};
