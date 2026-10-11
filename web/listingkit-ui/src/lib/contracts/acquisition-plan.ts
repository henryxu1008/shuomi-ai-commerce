import { z } from "zod";
import { canonical1688Source, isAcquisitionUUID } from "./product-acquisition";

const id=z.string().refine(isAcquisitionUUID);
const revision=z.number().int().positive().max(Number.MAX_SAFE_INTEGER);
const decimal=z.string().regex(/^(?:0|[1-9][0-9]{0,15})(?:\.[0-9]{1,8})?$/);
const optionalDecimal=decimal.or(z.literal(""));
const exact=(v:string)=>{const [whole,part=""]=v.split(".");return BigInt(whole!)*BigInt(100000000)+BigInt(part.padEnd(8,"0"));};
export const acquisitionRuleSchema=z.object({minPrice:optionalDecimal.optional(),maxPrice:optionalDecimal.optional(),maxMOQ:z.number().int().min(0).max(1000000000).optional()}).strict().refine(r=>!r.minPrice||!r.maxPrice||exact(r.minPrice)<=exact(r.maxPrice),"最低价不能大于最高价");
export const createPlanSchema=z.object({mode:z.enum(["product","page","keyword"]),inputs:z.array(z.string().min(1).max(2048).refine(v=>!/[\r\n\0]/.test(v))).min(1).max(20),templateId:id.optional(),templateRevision:revision.optional(),autoTransfer:z.boolean()}).strict().refine(v=>!!v.templateId===!!v.templateRevision).refine(v=>v.mode!=="product"||v.inputs.every(s=>canonical1688Source(s)!==null)).refine(v=>v.mode!=="keyword"||v.inputs.every(s=>new TextEncoder().encode(s.trim()).length<=200));
export const advancePlanSchema=z.object({expectedRevision:revision}).strict();
export const ruleMutationSchema=z.object({action:z.enum(["create","update","archive"]),id:id.optional(),name:z.string().trim().min(1).max(100).refine(s=>new TextEncoder().encode(s).length<=400&&!/[\r\n\0]/.test(s)).optional(),rule:acquisitionRuleSchema,expectedRevision:revision.optional()}).strict().refine(v=>v.action==="create"?!v.id&&!v.expectedRevision&&!!v.name:!!v.id&&!!v.expectedRevision&&(v.action==="archive"||!!v.name)).refine(v=>v.action!=="archive"||!v.name&&!v.rule.minPrice&&!v.rule.maxPrice&&!v.rule.maxMOQ);
export const ruleTemplateSchema=z.object({id,name:z.string().min(1).max(100),rule:acquisitionRuleSchema,revision,archived:z.boolean(),createdAt:z.iso.datetime({offset:true})}).strict();
const evaluation=z.object({state:z.enum(["matched","not_matched","unknown"]),reason:z.string().max(64).optional(),startingPrice:decimal.optional(),minimumQuantity:z.string().regex(/^[1-9][0-9]{0,18}$/).optional()}).strict();
const item=z.object({source:z.string().max(128).refine(s=>canonical1688Source(s)===s),operationId:id.optional(),state:z.enum(["pending","published","failed","needs_verification"]),failureCode:z.string().max(64).optional(),evaluation:evaluation.optional(),collectionItemId:id.optional()}).strict().refine(v=>v.state!=="published"||!!v.operationId&&!!v.evaluation);
export const acquisitionPlanSchema=z.object({id,input:createPlanSchema.safeExtend({rule:acquisitionRuleSchema}),revision,discoveryIndex:z.number().int().min(0).max(20),items:z.array(item).max(200),state:z.enum(["pending","paused","transfer_pending","completed"]),truncated:z.boolean(),failureCode:z.string().max(64).optional(),discoveryFailures:z.array(z.object({inputIndex:z.number().int().min(0).max(19),code:z.literal("source_unavailable")}).strict()).max(20),preparationId:id.optional(),createdAt:z.iso.datetime({offset:true})}).strict().refine(v=>v.discoveryIndex<=v.input.inputs.length&&new Set(v.items.map(i=>i.source)).size===v.items.length);
export const planListSchema=z.object({schemaVersion:z.literal(1),plans:z.array(acquisitionPlanSchema).max(100),canAutoTransfer:z.boolean(),nextCursor:id.optional()}).strict();
export const ruleListSchema=z.object({schemaVersion:z.literal(1),rules:z.array(ruleTemplateSchema).max(100)}).strict();
export type AcquisitionPlan=z.infer<typeof acquisitionPlanSchema>;
export type AcquisitionRule=z.infer<typeof acquisitionRuleSchema>;
export type RuleTemplate=z.infer<typeof ruleTemplateSchema>;
export type CreatePlan=z.infer<typeof createPlanSchema>;
export type RuleMutation=z.infer<typeof ruleMutationSchema>;
export type PlanRoute="list"|"create"|"read"|"by-key"|"advance"|"rules"|"mutate-rule";
export const planMethods:Record<PlanRoute,"GET"|"POST">={list:"GET",create:"POST",read:"GET","by-key":"GET",advance:"POST",rules:"GET","mutate-rule":"POST"};
export function planPath(method:string,path:string[]):PlanRoute|null{
 if(path[0]!=="sourcing"||path[1]!=="1688")return null;
 if(path.length===3&&path[2]==="rules")return method==="GET"?"rules":method==="POST"?"mutate-rule":null;
 if(path[2]!=="plans")return null;
 if(path.length===3)return method==="GET"?"list":method==="POST"?"create":null;
 if(path.length===4&&isAcquisitionUUID(path[3]!))return method==="GET"?"read":null;
 if(path.length===5&&path[3]==="by-key"&&isAcquisitionUUID(path[4]!))return method==="GET"?"by-key":null;
 if(path.length===5&&isAcquisitionUUID(path[3]!)&&path[4]==="advance")return method==="POST"?"advance":null;
 return null;
}
export function planRequestSchema(route:PlanRoute){return route==="create"?createPlanSchema:route==="advance"?advancePlanSchema:route==="mutate-rule"?ruleMutationSchema:null;}
export function planResponseSchema(route:PlanRoute){return route==="list"?planListSchema:route==="rules"?ruleListSchema:route==="mutate-rule"?ruleTemplateSchema:acquisitionPlanSchema;}
