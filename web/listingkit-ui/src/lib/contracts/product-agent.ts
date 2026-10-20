import { z } from "zod";
import {executionModeSchema,modeRevisionSchema,modeSnapshotRefSchema,titlePolicyAdoptionSchema,contentPolicyAdoptionSchema} from "./execution-mode";
import { contentSelectionSchema, sameContentSelection } from "./content-selection";
import { isAcquisitionUUID } from "./product-acquisition";
import { knowledgeSelectionSchema, productKnowledgeSchema } from "./product-knowledge";
import {templateRefSchema} from "./agent-configuration";
import {skillSelectionSchema,skillRefSchema,sameSkill} from "./skill";
const id = z.string().refine(isAcquisitionUUID);
const version = z.string().max(19).regex(/^[1-9][0-9]*$/).refine(v => BigInt(v) <= BigInt("9223372036854775807"));
// Candidate Action is bounded to 64 KiB by the runtime. HTTP and both BFF/client
// readers independently enforce the 128 KiB aggregate response bound. Do not
// reject valid backend arrays/strings with a smaller unrelated presentation cap.
const text = z.string();
const diagnostic = z.strictObject({ Code: text, Field: text, Message: text, Metadata: z.record(z.string(), z.string()).nullable() });
const change = z.strictObject({ Field: text, Value: text, EvidenceIDs: z.array(text).nullable() });
export const agentEmptyRequestSchema = z.strictObject({});
const agentTargetPlatformSchema = z.enum(["shein", "temu", "amazon"]);
export const agentStartRequestSchema = z.strictObject({ expectedModeRevision:modeRevisionSchema,targetPlatform: agentTargetPlatformSchema, knowledgeSelection:knowledgeSelectionSchema.optional(),templateSelection:templateRefSchema.optional() });
export const contentAgentStartRequestSchema = agentStartRequestSchema.extend({targetPlatform:z.enum(["product","shein","temu","amazon"]),manualReviewOnly:z.literal(true).optional(),contentSelection:contentSelectionSchema,skillSelection:skillSelectionSchema.optional()}).refine(v=>!v.skillSelection||v.contentSelection.title&&!v.contentSelection.description&&!v.contentSelection.sellingPoints&&!v.contentSelection.seoKeywords);
const ownSourceSelectionSchema=z.strictObject({catalogVersion:version,publicationId:z.string().min(1).max(128)});
export type OwnSourceSelection=z.infer<typeof ownSourceSelectionSchema>;
export const ownContentStartRequestSchema=contentAgentStartRequestSchema.safeExtend({targetPlatform:z.literal("product"),ownSource:ownSourceSelectionSchema});
export const ownContentSourceSchema=z.strictObject({itemId:id,productKey:z.string().min(1).max(128),catalogVersion:version,publicationId:z.string().min(1).max(128),originalVersion:z.number().int().positive().max(Number.MAX_SAFE_INTEGER),originalPublicationId:z.string().min(1).max(128),title:z.string().max(20000)});
export const agentResumeRequestSchema = z.strictObject({ revision: version, feedback: z.string().refine(v => new TextEncoder().encode(v).length <= 8192) });
export const agentReviewLinkSchema = z.strictObject({ proposalId: id, requestKey: id, operationId: id });
const baseAgentResultSchema = z.strictObject({
    executionMode:executionModeSchema.optional(),modeRevision:modeRevisionSchema.optional(),modeSnapshotRef:modeSnapshotRefSchema.optional(),modeUnavailableReason:z.literal("EXECUTION_MODE_NOT_FROZEN").optional(),policyAdoption:z.union([titlePolicyAdoptionSchema,contentPolicyAdoptionSchema]).optional(),reviewId:id.optional(),
    knowledge:productKnowledgeSchema.optional(),
    templateSelection:templateRefSchema.optional(),
    contextKind:z.literal("own_product").optional(),
    runId: id, requestKey: id, operationId: id, productKey: z.string().min(1).max(128), catalogVersion: version, publicationId: z.string().min(1).max(128), targetPlatform: z.enum(["product","shein","temu","amazon"]),
    phase: z.enum(["running", "interrupted", "human_review_required", "stopped"]), revision: version, stopReason: z.string().max(128).optional(), humanReviewRequired: z.boolean(),
    candidate: z.strictObject({ Changes: z.array(change).nullable(), Warnings: z.array(diagnostic).nullable(), Rejections: z.array(diagnostic).nullable() }),
    confidence: z.array(z.strictObject({ Field: text, Value: z.number().min(0).max(1), Known: z.boolean() })).nullable(),
    unresolved: z.array(text).nullable(),
    steps: z.array(z.strictObject({ step: z.number().int().min(0).max(16), tool: z.string().max(128).optional(), callId: z.string().max(128).optional(), invocationId: z.string().max(128).optional(), auditStatus: z.string().max(64).optional() })).max(24),
    tokens: z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER), estimatedCostMicros: z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER), currency: z.string().regex(/^[A-Z]{3}$/), usageStatus: z.enum(["observed", "unknown_reserved"]), canSubmitReview: z.boolean(),
});
function authorityMatches(v:z.infer<typeof baseAgentResultSchema>) {
    if(v.contextKind && (v.targetPlatform!=="product" || v.policyAdoption&&v.policyAdoption.execution.binding.ContextKind!==v.contextKind))return false;
    if (v.modeUnavailableReason ? (v.executionMode || v.modeRevision || v.modeSnapshotRef || v.canSubmitReview || !v.humanReviewRequired) : (!v.executionMode || !v.modeRevision || !v.modeSnapshotRef)) return false;
    if (!v.humanReviewRequired && (!v.policyAdoption || !v.reviewId || v.canSubmitReview)) return false;
    const p=v.policyAdoption;
    return !p || p.execution.runId===v.runId && p.execution.requestKey===v.requestKey && p.execution.binding.ContextID===v.operationId && p.execution.binding.ProductKey===v.productKey && p.execution.binding.CatalogVersion===v.catalogVersion && p.execution.binding.PublicationID===v.publicationId && p.execution.binding.TargetPlatform===v.targetPlatform && p.mode===v.executionMode && p.modeRevision===v.modeRevision && p.execution.mode.ID===v.modeSnapshotRef?.ID && p.execution.mode.Digest===v.modeSnapshotRef?.Digest;
}
export const agentResultSchema = baseAgentResultSchema.extend({targetPlatform:agentTargetPlatformSchema,policyAdoption:titlePolicyAdoptionSchema.optional()})
.refine(v => !v.canSubmitReview || (v.phase === "human_review_required" && !v.stopReason && v.candidate.Changes?.length === 1 && v.candidate.Changes[0]?.Field === "title"))
.refine(authorityMatches);
export const contentAgentResultSchema = baseAgentResultSchema.extend({targetPlatform:z.enum(["product","shein","temu","amazon"]),contentSelection:contentSelectionSchema,policyAdoption:contentPolicyAdoptionSchema.optional(),skillVersionRef:skillRefSchema.optional()})
.refine(v=>!v.skillVersionRef||v.contentSelection.title&&!v.contentSelection.description&&!v.contentSelection.sellingPoints&&!v.contentSelection.seoKeywords)
.refine(v=>!v.policyAdoption||sameSkill(v.skillVersionRef?{skillId:v.skillVersionRef.id,version:v.skillVersionRef.version,digest:v.skillVersionRef.digest}:undefined,v.policyAdoption.execution.skillVersionRef?{skillId:v.policyAdoption.execution.skillVersionRef.id,version:v.policyAdoption.execution.skillVersionRef.version,digest:v.policyAdoption.execution.skillVersionRef.digest}:undefined))
.refine(authorityMatches)
.refine(v=>!v.canSubmitReview || v.phase==="human_review_required" && !v.stopReason && !v.unresolved?.length && !v.candidate.Warnings?.length && !v.candidate.Rejections?.length && !!v.candidate.Changes && (()=>{
 const fields=[v.contentSelection.title&&"title",v.contentSelection.description&&"description",v.contentSelection.sellingPoints&&"selling_points",v.contentSelection.seoKeywords&&"seo_keywords"].filter(Boolean);
 return v.candidate.Changes.length===fields.length && new Set(v.candidate.Changes.map(c=>c.Field)).size===fields.length && v.candidate.Changes.every(c=>fields.includes(c.Field));
})())
.refine(v=>!v.policyAdoption || sameContentSelection(v.policyAdoption.execution.ContentSelection,v.contentSelection));
export type ProductAgentResult = z.infer<typeof agentResultSchema> | z.infer<typeof contentAgentResultSchema>;
export function ownAgentPath(path:string[]):string|null {
 if(path[0]!=="collections"||path[1]!=="items"||!isAcquisitionUUID(path[2]??"")||path[3]!=="content-agent")return null;
 if(path.length===5&&path[4]==="source")return path.join("/");
 if(path[4]!=="runs")return null;
 return path.length===5||path.length===6&&isAcquisitionUUID(path[5]??"")||path.length===7&&isAcquisitionUUID(path[5]??"")&&["resume","review"].includes(path[6]??"")?path.join("/"):null;
}
export function agentPath(path: string[], namespace?: "product-agent" | "content-agent"): string | null {
    if (path[0] !== "sourcing" || path[1] !== "1688" || path[2] !== "acquisitions" || !isAcquisitionUUID(path[3] ?? "") || (namespace ? path[4] !== namespace : !["product-agent", "content-agent"].includes(path[4] ?? "")) || path[5] !== "runs")
        return null;
    if (path.length === 6 || path.length === 7 && isAcquisitionUUID(path[6] ?? "") || path.length === 8 && isAcquisitionUUID(path[6] ?? "") && ["resume", "review"].includes(path[7] ?? ""))
        return path.join("/");
    return null;
}

// Navigation selects an existing operation; it never supplies execution intent
// or authority. Unknown kinds are discarded before constructing a detail link.
export function productAgentOperationHref(operationId: string, kind?: string | null): string {
    return `/workbench/supply/acquisition/operation/${operationId}${kind === "title" || kind === "content" ? `?agent_kind=${kind}` : ""}`;
}
