import {z} from "zod";
import {collectionID} from "./product-collection";

const integer=z.number().int().safe().nonnegative();
const revision=integer.positive();
const version=z.string().regex(/^[1-9][0-9]*$/);
const hash=z.string().regex(/^[a-f0-9]{64}$/);
const selection=z.array(collectionID).min(1).max(1000).refine(v=>new Set(v).size===v.length).optional();
export const uploadPlanInputSchema=z.object({
 preparationId:collectionID,expectedRevision:revision,storeId:collectionID,sourceIds:selection,
 rule:z.literal("standard"),publicationMode:z.enum(["platform","platform_draft"]),
 titleTemplateId:collectionID.optional(),titleTemplateRevision:version.optional(),titleQuoteHash:hash.optional(),titleModeRevision:z.string().regex(/^(0|[1-9][0-9]*)$/).optional(),
}).strict().refine(v=>!!v.titleTemplateId===!!v.titleTemplateRevision&&!!v.titleTemplateId===!!v.titleQuoteHash&&!!v.titleTemplateId===(v.titleModeRevision!==undefined));
export const uploadPlanSchema=z.object({cancelled:z.boolean().default(false),id:collectionID,input:uploadPlanInputSchema,storeBindingHash:hash,count:revision,revision,createdAt:z.iso.datetime({offset:true})});
export const uploadPlanPageSchema=z.object({items:z.array(uploadPlanSchema).max(100),nextCursor:collectionID.optional()});
export const uploadPlanReceiptSchema=z.object({plan:uploadPlanSchema,replayed:z.boolean()});
export const uploadPlanSelectionSchema=z.object({expectedRevision:revision,sourceIds:selection}).strict();
export const uploadPlanContinueSchema=z.object({expectedRevision:revision,previewHash:hash,sourceIds:selection}).strict();
export const uploadPlanCancelSchema=z.object({expectedRevision:revision}).strict();
export const uploadPlanCancelIntentSchema=uploadPlanCancelSchema.extend({planId:collectionID});
export const uploadPlanContinueIntentSchema=uploadPlanContinueSchema.extend({planId:collectionID});
export const uploadPlanAdvanceSchema=z.object({planId:collectionID,operationIds:z.array(collectionID).min(1).max(10000),replayed:z.boolean()});
const status=z.enum(["pending","running","succeeded","missing","review","unknown","failed","denied","cancelled"]);
const step=z.enum(["adapt","optimize","upload","missing","review","unknown","pending","running","failed","denied","cancelled","succeeded","rejected","unavailable","channel_exists"]);
const item=z.object({sourceId:collectionID,sourceVersion:version,revision,adaptOperationId:collectionID.optional(),optimizeOperationId:collectionID.optional(),uploadOperationId:collectionID.optional()});
const facts=z.object({publicationSummary:z.array(z.string().max(2048)).max(40).optional(),hasTarget:z.boolean(),ready:z.boolean(),target:z.object({sourceId:z.union([collectionID,z.literal("")]),recordId:z.union([collectionID,z.literal("")]),revision:integer}),publicationMode:z.string().max(64),adaptStatus:status.optional(),optimizationStatus:status.optional(),reviewRequired:z.boolean(),reviewRejected:z.boolean(),uploadStatus:status.optional(),uploadConfirmed:z.boolean(),existingPublicationRecordId:collectionID.optional()}).refine(v=>!v.hasTarget||!!v.target.sourceId&&!!v.target.recordId&&v.target.revision>0);
export const uploadPlanPreviewSchema=z.object({plan:uploadPlanSchema,items:z.array(z.object({item,facts,step}).refine(v=>!v.facts.hasTarget||v.facts.target.sourceId===v.item.sourceId)).max(100),count:integer,actionable:integer,counts:z.partialRecord(step,integer),hash}).refine(v=>Object.values(v.counts).reduce((n,x)=>n+(x??0),0)===v.count&&v.actionable===(v.counts.adapt??0)+(v.counts.optimize??0)+(v.counts.upload??0)&&v.items.length<=v.count);
export type UploadPlanInput=z.infer<typeof uploadPlanInputSchema>;
export type UploadPlan=z.infer<typeof uploadPlanSchema>;
export type UploadPlanPreview=z.infer<typeof uploadPlanPreviewSchema>;
export type UploadPlanRoute="cancel-upload-plan"|"create-upload-plan"|"upload-plans"|"upload-plan-key"|"upload-plan"|"upload-plan-preview"|"continue-upload-plan"|"upload-plan-command";
export const uploadPlanRoutes:ReadonlyArray<readonly["GET"|"POST",UploadPlanRoute]>=[["POST","cancel-upload-plan"],["POST","create-upload-plan"],["GET","upload-plans"],["GET","upload-plan-key"],["GET","upload-plan"],["POST","upload-plan-preview"],["POST","continue-upload-plan"],["GET","upload-plan-command"]];
export function uploadPlanPath(method:string,p:string[]):UploadPlanRoute|null{
 if(p[0]!=="supply-preparations"||p[1]!=="upload-plans")return null;
 const valid=(v:string|undefined)=>collectionID.safeParse(v).success;
 if(p.length===2)return method==="GET"?"upload-plans":method==="POST"?"create-upload-plan":null;
 if(method==="GET"&&p.length===4&&p[2]==="by-key"&&valid(p[3]))return "upload-plan-key";
 if(!valid(p[2]))return null;
 if(method==="GET"&&p.length===3)return "upload-plan";
 if(method==="GET"&&p.length===5&&p[3]==="commands"&&valid(p[4]))return "upload-plan-command";
 if(method==="POST"&&p.length===4)return p[3]==="cancel"?"cancel-upload-plan":p[3]==="preview"?"upload-plan-preview":p[3]==="continue"?"continue-upload-plan":null;
 return null;
}
export function uploadPlanRequest(route:string){return route==="cancel-upload-plan"?uploadPlanCancelSchema:route==="create-upload-plan"?uploadPlanInputSchema:route==="continue-upload-plan"?uploadPlanContinueSchema:route==="upload-plan-preview"?uploadPlanSelectionSchema:null;}
export function uploadPlanResponse(route:string){return route==="create-upload-plan"||route==="upload-plan-key"?uploadPlanReceiptSchema:route==="upload-plan"||route==="cancel-upload-plan"?uploadPlanSchema:route==="upload-plans"?uploadPlanPageSchema:route==="upload-plan-preview"?uploadPlanPreviewSchema:route==="continue-upload-plan"||route==="upload-plan-command"?uploadPlanAdvanceSchema:null;}
