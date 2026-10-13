import { z } from "zod";
import { knowledgeId } from "../api/knowledge";
import { configVersion } from "./agent-configuration";
import { contentSelectionSchema } from "./content-selection";

export const executionModeSchema = z.enum(["default", "auto_approve", "read_only", "full_access"]);
export const modeRevisionSchema = z.union([z.literal("0"), configVersion]);
const definitionId = z.string().min(1).max(128).regex(/^[a-z][a-z0-9_-]*(\.[a-z][a-z0-9_-]*)*$/);
export const modeSubjectSchema = z.discriminatedUnion("kind", [
  z.strictObject({ kind: z.literal("registered_agent"), id: definitionId }),
  z.strictObject({ kind: z.literal("private_delivery"), id: knowledgeId }),
]);
export const executionModeInputSchema = z.strictObject({ mode: executionModeSchema, expectedRevision: modeRevisionSchema });
const timestamp = z.string().datetime({ offset: true });
const executionModeReceiptSchema = z.strictObject({
  commandId: knowledgeId,
  requestKey: knowledgeId,
  operation: z.literal("set-execution-mode"),
  subject: modeSubjectSchema,
  mode: executionModeSchema,
  beforeRevision: modeRevisionSchema,
  revision: configVersion,
  noop: z.boolean(),
  committedAt: timestamp,
  updatedAt: timestamp,
  updatedBy: z.string().max(128),
});
export const executionModeViewSchema = z.strictObject({
  subject: modeSubjectSchema,
  definition: z.strictObject({ id: definitionId, version: z.string().min(1).max(128).refine((v) => v.trim() === v && !/[\u0000-\u001f\u007f]/.test(v)) }),
  mode: executionModeSchema,
  revision: modeRevisionSchema,
  policyVersion: z.literal("agent-execution-policy-v1"),
  canConfigure: z.boolean(),
  updatedAt: timestamp,
  updatedBy: z.string().max(128),
  receipt: executionModeReceiptSchema.optional(),
}).superRefine((value, context) => {
  if (value.subject.kind === "registered_agent" && value.subject.id !== value.definition.id) {
    context.addIssue({ code: "custom", message: "Definition must match the registered subject" });
  }
  const receipt = value.receipt;
  if (receipt && (receipt.subject.kind !== value.subject.kind || receipt.subject.id !== value.subject.id || receipt.mode !== value.mode || receipt.revision !== value.revision)) {
    context.addIssue({ code: "custom", message: "Receipt must match exact mode and subject" });
  }
  if (receipt && (receipt.updatedAt !== value.updatedAt || receipt.updatedBy !== value.updatedBy)) {
    context.addIssue({ code: "custom", message: "Receipt audit must match the saved mode revision" });
  }
});

export type ExecutionMode = z.infer<typeof executionModeSchema>;
export type ModeSubject = z.infer<typeof modeSubjectSchema>;
export type ExecutionModeView = z.infer<typeof executionModeViewSchema>;

const digest = z.string().regex(/^[0-9a-f]{64}$/);
const identifier = z.string().min(1).max(128);
const scopeSchema = z.strictObject({OrganizationID:identifier,ActorID:identifier});
export const modeSnapshotRefSchema = z.strictObject({Kind:z.literal("agent-execution-mode-v1"),ID:knowledgeId,Digest:digest});
const titleExecutionProvenanceSchema = z.strictObject({
 scope:scopeSchema,runId:knowledgeId,requestKey:knowledgeId,
 binding:z.strictObject({ContextKind:identifier,ContextID:identifier,ProductKey:identifier,CatalogVersion:configVersion,PublicationID:identifier,TargetPlatform:z.enum(["shein","temu","amazon"])}),
 configuration:z.strictObject({Kind:z.literal("agent-configuration-v1"),ID:knowledgeId,Digest:digest}),mode:modeSnapshotRefSchema,
});
const policyAdoptionBaseSchema = z.strictObject({
 execution:titleExecutionProvenanceSchema,
 identity:z.strictObject({OrganizationID:identifier,ActorID:identifier,MemberID:identifier}),
 mode:z.enum(["auto_approve","full_access"]),modeRevision:configVersion,
 policyVersion:z.literal("agent-execution-policy-v1"),actionVersion:z.literal("product.title.policy-adopt-v1"),proposalDigest:digest,requiresHuman:z.boolean(),
});
export const titlePolicyAdoptionSchema = policyAdoptionBaseSchema.refine(v=>v.identity.OrganizationID===v.execution.scope.OrganizationID&&v.identity.ActorID===v.execution.scope.ActorID&&!(v.mode==="auto_approve"&&v.requiresHuman));
export const contentExecutionProvenanceSchema = titleExecutionProvenanceSchema.extend({
 ContentSelection: contentSelectionSchema,
 binding: titleExecutionProvenanceSchema.shape.binding.extend({TargetPlatform:z.enum(["product","shein","temu","amazon"])}),
});
export const contentPolicyAdoptionSchema = policyAdoptionBaseSchema.extend({
 execution: contentExecutionProvenanceSchema,
 actionVersion: z.literal("product.content.policy-adopt-v1"),
}).refine(v=>v.identity.OrganizationID===v.execution.scope.OrganizationID&&v.identity.ActorID===v.execution.scope.ActorID&&!(v.mode==="auto_approve"&&v.requiresHuman))
.refine(v=>v.mode!=="auto_approve" || !!v.execution.ContentSelection.title && !v.execution.ContentSelection.description && !v.execution.ContentSelection.sellingPoints && !v.execution.ContentSelection.seoKeywords);

export const imagePolicyAdoptionSchema=z.strictObject({
 identity:z.strictObject({OrganizationID:identifier,ActorID:identifier,MemberID:identifier}),
 mode:z.literal("full_access"),modeRevision:configVersion,modeSnapshotRef:modeSnapshotRefSchema,
 configurationSnapshotRef:z.strictObject({Kind:z.literal("agent-configuration-v1"),ID:knowledgeId,Digest:digest}),
 runId:knowledgeId,planRevision:z.number().int().positive(),resultDigest:digest,
 policyVersion:z.literal("agent-execution-policy-v1"),actionVersion:z.literal("product.image.policy-adopt-v1"),
});
