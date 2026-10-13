import { z } from "zod";
import {titlePolicyAdoptionSchema,contentPolicyAdoptionSchema,contentExecutionProvenanceSchema,executionModeSchema,modeRevisionSchema,modeSnapshotRefSchema} from "../contracts/execution-mode";
import { contentSelectionSchema, sameContentSelection, type ContentSelection } from "../contracts/content-selection";
import { productKnowledgeSchema } from "../contracts/product-knowledge";
import { parseWorkbenchErrorEnvelopePayload, type WorkbenchErrorEnvelope } from "./workbench-context";

export const PRODUCT_REVIEW_RESPONSE_BYTES = 128 * 1024;
const bytes = (s: string) => new TextEncoder().encode(s).length;
const unicode = (s: string) => !/[\uD800-\uDFFF]/u.test(s);
const text = z.string().refine(unicode);
export const productTitleIDSchema = z.string().regex(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/)
  .refine((v) => v !== "00000000-0000-0000-0000-000000000000");
const version = z.string().max(19).regex(/^[1-9][0-9]*$/).refine((v) => /^[1-9][0-9]*$/.test(v) && BigInt(v) <= BigInt("9223372036854775807"));
export const productTitleOrganizationSchema = z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/);
const productTitleKeySchema = text.min(1).refine((v) => bytes(v) <= 128 && v.trim() === v && !/\p{Cc}/u.test(v));
// A browser Header cannot preserve Unicode or distinguish a comma-joined pair
// from one value. Use a single visible ASCII key (UUID recommended), unchanged.
export const productTitleIdempotencyKeySchema = z.string().min(1).max(128).regex(/^[\x21-\x7e]+$/).refine((v) => !v.includes(","));
export const productTitleCursorSchema = z.string().min(1).max(256).regex(/^[A-Za-z0-9_-]+$/);
const title = text.min(1).refine((v) => bytes(v) <= 4096 && v.trim() === v && !/\p{Cc}/u.test(v));
const time = z.iso.datetime({ precision: undefined }).regex(/T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/).refine((v) => !v.startsWith("0001-"));
const basis = { schema_version: z.literal(1), coverage: z.literal("product-title-proposals-only") };
const item = z.strictObject({ proposal_id: productTitleIDSchema, product_key: productTitleKeySchema, base_version: version, proposal_revision: version, state: z.enum(["pending", "accepted"]) });
const collection = z.strictObject({ ...basis, items: z.array(item).max(100), next_cursor: productTitleCursorSchema.nullable() })
  .refine((v) => new Set(v.items.map((i) => i.proposal_id)).size === v.items.length);
const typedCollectionItem = z.discriminatedUnion("policy", [item.extend({policy:z.literal("title-review-v1")}),item.extend({policy:z.literal("product-content-review-v1"),content_selection:contentSelectionSchema})]);
const textCollection = z.strictObject({schema_version:z.literal(2),coverage:z.literal("product-text-proposals"),items:z.array(typedCollectionItem).max(100),next_cursor:productTitleCursorSchema.nullable()}).refine(v=>new Set(v.items.map(i=>i.proposal_id)).size===v.items.length);
const proposalCollection = z.union([collection,textCollection]);
const decision = z.strictObject({ action: z.enum(["accept", "edit", "reject"]), actor: text.min(1), revision: version, before: text, after: text, at: time });
const receipt = z.strictObject({ proposal_id: productTitleIDSchema, revision: version, product_version: version, publication_id: text.min(1), actor: text.min(1), at: time });
const proposalBase = z.strictObject({
  policy_adoption:titlePolicyAdoptionSchema.optional(),
  knowledge:productKnowledgeSchema.optional(),
  ...basis, proposal_id: productTitleIDSchema, owner: text.min(1),
  input: z.strictObject({ product_key: productTitleKeySchema, base_version: version }),
  before: text, after: text, original_title: text, policy: z.literal("title-review-v1"),
  state: z.enum(["pending", "accepted", "rejected", "applied"]), revision: version,
  evidence: z.array(z.strictObject({ id: text, reference_type: text, reference_id: text, snapshot_id: text, checksum: text })),
  quality: z.strictObject({ overall: z.number(), evidence_coverage: z.number(), required_field_coverage: z.number() }),
  unresolved: z.array(text), decisions: z.array(decision), apply_receipt: receipt.optional(),
});
const proposal = proposalBase.refine((v) => v.state === "applied" ? !!v.apply_receipt && v.apply_receipt.proposal_id === v.proposal_id : !v.apply_receipt);
const contentFields = ["title", "description", "selling_points", "seo_keywords"] as const;
export const contentValueSchema = z.strictObject({field:z.enum(contentFields),value:text});
export type ContentValue = z.infer<typeof contentValueSchema>;
const contentValues = z.array(contentValueSchema).min(1).max(4).refine(values=>new Set(values.map(v=>v.field)).size===values.length);
function validContentValue(v: ContentValue) {
  if (v.field === "title") return title.safeParse(v.value).success;
  if (v.field === "description") return !!v.value && v.value.trim()===v.value && bytes(v.value)<=65536 && !/\p{Cc}/u.test(v.value.replaceAll("\n", ""));
  try { const values:unknown=JSON.parse(v.value); return Array.isArray(values) && values.length>0 && values.every(value=>title.safeParse(value).success) && JSON.stringify(values)===v.value; } catch { return false; }
}
export const editedContentValuesSchema = contentValues.refine(values=>values.every(validContentValue));
function matchesSelection(values:ContentValue[],s:ContentSelection) {
 const expected=[s.title&&"title",s.description&&"description",s.sellingPoints&&"selling_points",s.seoKeywords&&"seo_keywords"].filter(Boolean);
 return values.length===expected.length && values.every((v,i)=>v.field===expected[i]);
}
const contentPayload = z.strictObject({
 selection:contentSelectionSchema,before:contentValues,after:editedContentValuesSchema,
 decisions:z.array(z.strictObject({action:z.enum(["accept","edit","reject"]),actor:text.min(1),revision:z.number().int().min(2).max(100),before:editedContentValuesSchema,after:editedContentValuesSchema,at:time})).nullable(),
}).refine(v=>matchesSelection(v.before,v.selection)&&matchesSelection(v.after,v.selection)&&(v.decisions??[]).every(d=>matchesSelection(d.before,v.selection)&&matchesSelection(d.after,v.selection)));
const contentProposal = proposalBase.extend({
 schema_version:z.literal(2),coverage:z.literal("product-content-proposals-only"),policy:z.literal("product-content-review-v1"),
 content:contentPayload,execution:contentExecutionProvenanceSchema,policy_adoption:contentPolicyAdoptionSchema.optional(),decisions:z.array(z.never()),
 execution_mode:z.strictObject({mode:executionModeSchema,revision:modeRevisionSchema,snapshot:modeSnapshotRefSchema}).optional(),
}).refine(v=>v.state==="applied"?!!v.apply_receipt&&v.apply_receipt.proposal_id===v.proposal_id:!v.apply_receipt)
.refine(v=>v.execution.scope.ActorID===v.owner&&v.execution.binding.ProductKey===v.input.product_key&&v.execution.binding.CatalogVersion===v.input.base_version&&sameContentSelection(v.content.selection,v.execution.ContentSelection))
.refine(v=>!v.execution_mode||v.execution_mode.snapshot.ID===v.execution.mode.ID&&v.execution_mode.snapshot.Digest===v.execution.mode.Digest)
.refine(v=>!v.policy_adoption||v.state==="applied"&&!(v.content.decisions??[]).length&&JSON.stringify(v.policy_adoption.execution)===JSON.stringify(v.execution));
const textProposal = z.union([proposal,contentProposal]);

const titleDecisionSchema = z.discriminatedUnion("action", [
  z.strictObject({ action: z.literal("accept"), expected_revision: version }),
  z.strictObject({ action: z.literal("reject"), expected_revision: version }),
  z.strictObject({ action: z.literal("edit"), expected_revision: version, title }),
]);
export const productTitleDecisionSchema=z.union([titleDecisionSchema,z.strictObject({action:z.literal("edit"),expected_revision:version,content:editedContentValuesSchema})]).refine(v=>bytes(JSON.stringify(v))<=32768);
export const productTitleApplySchema = z.strictObject({ expected_revision: version });
export type ProductTitleProposalListItem = z.infer<typeof item>|z.infer<typeof typedCollectionItem>;
export type ProductTitleProposalList = Omit<z.infer<typeof proposalCollection>, "items"> & { items: ProductTitleProposalListItem[] };
export type ProductTitleProposal = z.infer<typeof textProposal>;
export type ProductContentProposal = z.infer<typeof contentProposal>;
export type ProductTitleDecisionInput = z.infer<typeof productTitleDecisionSchema>;
export type ProductTitleApplyInput = z.infer<typeof productTitleApplySchema>;
export function parseProductTitleProposalList(value: unknown): ProductTitleProposalList | null { const p = proposalCollection.safeParse(value); return p.success ? p.data : null; }
export function parseProductTitleProposal(value: unknown): ProductTitleProposal | null { const p = textProposal.safeParse(value); return p.success ? p.data : null; }

const domain = z.strictObject({ error: z.enum(["invalid_request", "permission_denied", "not_found", "stale_product_version", "operation_conflict", "input_too_large", "deadline_exceeded", "unavailable"]) });
const domainStatuses = { invalid_request: 400, permission_denied: 403, not_found: 404, stale_product_version: 409, operation_conflict: 409, input_too_large: 413, deadline_exceeded: 504, unavailable: 503 };
const statuses: Record<string, readonly number[]> = {
  INVALID_REQUEST: [400, 405], AUTHENTICATION_REQUIRED: [401], ORGANIZATION_SELECTION_REQUIRED: [409], ORGANIZATION_CONTEXT_CHANGED: [409],
  ORGANIZATION_ACCESS_DENIED: [403], ORGANIZATION_ACCESS_REVOKED: [403], ORGANIZATION_SUSPENDED: [403], PERMISSION_DENIED: [403],
  DEPENDENCY_UNAVAILABLE: [502, 503], INVALID_UPSTREAM_RESPONSE: [502], DEADLINE_EXCEEDED: [504], RESULT_UNVERIFIED: [502, 504], INPUT_TOO_LARGE: [413],
};
const communication = z.strictObject({ code: z.string(), message: text.max(2048), requestId: text.max(256), fieldErrors: z.array(z.never()), outcome: z.enum(["not_sent", "unknown"]) });
export type ProductTitleReviewFailure = z.infer<typeof domain> | WorkbenchErrorEnvelope | z.infer<typeof communication>;
export function parseProductTitleReviewFailure(value: unknown, status: number): ProductTitleReviewFailure | null {
  const d = domain.safeParse(value);
  if (d.success && domainStatuses[d.data.error] === status) return d.data;
  const c = communication.safeParse(value);
  if (c.success && statuses[c.data.code]?.includes(status) && (c.data.outcome === "unknown") === (c.data.code === "RESULT_UNVERIFIED")) return c.data;
  const w = parseWorkbenchErrorEnvelopePayload(value);
  return w.success && w.data.code !== "RESULT_UNVERIFIED" && statuses[w.data.code]?.includes(status) ? w.data : null;
}
