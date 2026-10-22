import { z } from "zod";
import { collectionID, collectionVersion } from "./product-collection";

const text = (max: number) => z.string().refine(v => !v.includes("\0") && new TextEncoder().encode(v).length <= max);
const reasons = z.array(text(8192)).max(128);
export const sourceProductInputsSchema = z.object({
  sourceId: collectionID,
  inputs: z.object({
    scope: z.literal("product.inputs"), status: z.enum(["ready", "blocked", "not_evaluated"]),
    product_key: text(128).min(1), product_version: collectionVersion, publication_id: text(128).min(1), target_platform: z.literal("product"),
    input_rule_version: z.literal("listing.readiness.ProductInputs/tool-v1.0.0"), reasons, missing_facts: reasons,
    asset_binding: z.object({ product_version: collectionVersion, status: z.enum(["missing", "exact"]), count: z.number().int().min(0).max(1024), digest: z.string().regex(/^sha256:[a-f0-9]{64}$/).optional() }).strict(),
    marketplace: z.object({ status: z.literal("not_evaluated"), reasons }).strict(),
    diagnostics: z.object({ needs_review: z.boolean(), review_reasons: reasons, warnings: z.array(z.object({ code: text(128), field: text(8192).optional(), message: text(8192) }).strict()).max(128) }).strict(),
  }).strict().refine(v => v.asset_binding.product_version === v.product_version &&
    (v.asset_binding.status === "missing" ? v.asset_binding.count === 0 && !v.asset_binding.digest : v.asset_binding.count > 0 && !!v.asset_binding.digest) &&
    (v.status !== "ready" || v.asset_binding.status === "exact" && !v.diagnostics.needs_review && v.reasons.length === 0)),
}).strict();
export type SourceProductInputs = z.infer<typeof sourceProductInputsSchema>;
