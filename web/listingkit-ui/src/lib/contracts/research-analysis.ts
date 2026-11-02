import {z} from "zod";
import {collectionID} from "./product-collection";
import {configVersion} from "./agent-configuration";
import {modeRevisionSchema} from "./execution-mode";
import {selectionInspectionSchema,selectionInputSchema,selectionPreparedSchema,selectionRecentSchema,selectionRunBaseSchema,selectionIntentSchema} from "./product-selection";
const text=(n:number)=>z.string().refine(v=>!v.includes("\0")&&new TextEncoder().encode(v).length<=n),digest=z.string().regex(/^[a-f0-9]{64}$/),factID=z.string().regex(/^fact:[a-f0-9]{64}$/);
const original=selectionInspectionSchema.shape.products.element.shape;
const researchRef=z.strictObject({CandidateID:collectionID,CandidateRevision:configVersion,SelectedLinkID:collectionID,EvidenceDigest:digest});
const query=z.strictObject({mode:text(32),keyword:text(512).optional(),categoryNode:text(512).optional(),asinCount:z.number().int().min(0).max(200),limit:z.number().int().min(0).max(200),fields:z.array(text(128)).max(32)});
export const researchInspectionSchema=z.strictObject({version:z.literal("market-research-inspection-v1"),projectId:collectionID,candidates:z.array(z.strictObject({ref:original.ref,research:researchRef,observation:z.strictObject({site:text(4).min(1),asin:text(10).length(10),capturedAt:text(64),parserVersion:text(128),checksum:text(256).min(1),query,currency:text(8).optional(),price:z.number().finite().nonnegative().optional(),priceBasis:text(64).optional()}),facts:original.facts,missing:original.missing,warnings:z.array(text(128)).max(32)})).min(2).max(5),summary:z.strictObject({selected:z.number().int().min(2).max(5),unpriced:z.number().int().min(0).max(5),prices:z.array(z.strictObject({currency:text(8),basis:z.literal("displayed_listing_offer"),count:z.number().int().min(1).max(5),minimum:z.number().finite().nonnegative(),maximum:z.number().finite().nonnegative()})).max(5)})}).superRefine((v,ctx)=>{
 if(new TextEncoder().encode(JSON.stringify(v)).length>32768||v.summary.selected!==v.candidates.length||new Set(v.candidates.map(c=>c.research.CandidateID)).size!==v.candidates.length||new Set(v.candidates.map(c=>c.ref.ItemID)).size!==v.candidates.length)ctx.addIssue({code:"custom",message:"研究样本不一致或超限"});
});
const choice=z.strictObject({candidateId:collectionID,expectedRevision:configVersion,linkId:collectionID,evidenceDigest:digest,selection:selectionInputSchema.shape.candidates.element});
export const researchAnalysisInput=z.strictObject({candidates:z.array(choice).min(2).max(5)}).refine(v=>new Set(v.candidates.map(c=>c.candidateId)).size===v.candidates.length&&new Set(v.candidates.map(c=>c.selection.itemId)).size===v.candidates.length);
export const researchPrepareSchema=researchAnalysisInput.safeExtend({requestKey:collectionID,expectedModeRevision:modeRevisionSchema,goalSummary:text(512).min(1).refine(v=>v.trim()===v)});
export const researchPreparedSchema=selectionPreparedSchema;
const proposal=z.strictObject({overview:text(512).min(1),overviewEvidenceIds:z.array(factID).min(1).max(12),recommendedCandidateId:collectionID.optional(),assessments:z.array(z.strictObject({candidateId:collectionID,recommendation:z.enum(["TEST_CANDIDATE","RESEARCH_MORE","NOT_RECOMMENDED"]),advice:text(512).min(1),evidenceIds:z.array(factID).min(1).max(6)})).min(1).max(5)});
export const researchRunSchema=selectionRunBaseSchema.extend({inspection:researchInspectionSchema.optional(),proposal:proposal.optional()}).superRefine((r,ctx)=>{
 const fail=()=>ctx.addIssue({code:"custom",message:"研究建议与原依据不一致"});
 if(r.inspection&&r.inspection.projectId!==r.contextId)fail();
 if(!r.validated){if(r.proposal)fail();return;}
 if(r.phase!=="human_review_required"||r.stopReason||!r.proposal||!r.inspection){fail();return;}
 const rows=new Map(r.inspection.candidates.map(c=>[c.research.CandidateID,c])),all=new Set(r.inspection.candidates.flatMap(c=>c.facts.map(f=>f.id))),seen=new Set<string>();
 if(new Set(r.proposal.overviewEvidenceIds).size!==r.proposal.overviewEvidenceIds.length||r.proposal.overviewEvidenceIds.some(id=>!all.has(id)))fail();
 for(const a of r.proposal.assessments){const row=rows.get(a.candidateId);if(!row||seen.has(a.candidateId)||new Set(a.evidenceIds).size!==a.evidenceIds.length||a.evidenceIds.some(id=>!row.facts.some(f=>f.id===id)))fail();seen.add(a.candidateId);}
 if(r.proposal.recommendedCandidateId&&!seen.has(r.proposal.recommendedCandidateId))fail();
});
export const researchRecentSchema=z.strictObject({items:z.array(selectionRecentSchema.shape.items.element.extend({contextKind:z.literal("market-research-selection-v1")})).max(10),nextCursor:z.union([z.literal(""),collectionID])});
const start=selectionIntentSchema.options[1].shape.body,resume=selectionIntentSchema.options[2].shape.body;
export const researchAnalysisIntent=z.discriminatedUnion("kind",[
 z.strictObject({kind:z.literal("prepare"),projectId:collectionID,key:collectionID,body:researchPrepareSchema}).refine(v=>v.key===v.body.requestKey),
 z.strictObject({kind:z.literal("start"),projectId:collectionID,key:collectionID,body:start}),
 z.strictObject({kind:z.literal("resume"),projectId:collectionID,key:collectionID,body:resume})
]);
export const researchSavedPreparation=z.strictObject({key:collectionID,projectId:collectionID,input:researchPrepareSchema,prepared:researchPreparedSchema}).refine(v=>v.key===v.prepared.requestKey&&v.projectId===v.prepared.contextId&&v.input.requestKey===v.prepared.requestKey);
export const researchAnalysisFailure=z.strictObject({code:z.string().regex(/^[A-Z_]{1,80}$/),contextId:collectionID.optional(),requestKey:collectionID.optional(),operationMayExist:z.boolean().optional()});
export type ResearchInspection=z.infer<typeof researchInspectionSchema>;
export type ResearchAnalysisIntent=z.infer<typeof researchAnalysisIntent>;
export function researchAnalysisEndpoint(url:URL,method:string):{path:string;input?:z.ZodType;output:z.ZodType}|null{
 const match=url.pathname.match(/^\/api\/workbench\/research\/projects\/([0-9a-f-]{36})\/analysis\/(.+)$/);
 if(!match||url.hash||!collectionID.safeParse(match[1]).success)return null;
 const action=match[2],path=match[1]+"/analysis/"+action;
 if(action==="runs"&&method==="GET"){if([...url.searchParams.keys()].some(k=>k!=="after")||url.searchParams.getAll("after").length>1||url.searchParams.has("after")&&!collectionID.safeParse(url.searchParams.get("after")).success)return null;return {path,output:researchRecentSchema};}
 if(url.search)return null;
 if(action==="inspect"&&method==="POST")return {path,input:researchAnalysisInput,output:researchInspectionSchema};
 if(action==="prepare"&&method==="POST")return {path,input:researchPrepareSchema,output:researchPreparedSchema};
 if(action==="runs"&&method==="POST")return {path,input:start,output:researchRunSchema};
 const parts=action.split("/");if(parts.length===2&&method==="GET"&&collectionID.safeParse(parts[1]).success)return parts[0]==="preparations"?{path,output:researchPreparedSchema}:parts[0]==="runs"?{path,output:researchRunSchema}:null;
 if(parts.length===3&&parts[0]==="runs"&&parts[2]==="resume"&&method==="POST"&&collectionID.safeParse(parts[1]).success)return {path,input:resume,output:researchRunSchema};
 return null;
}
