import { z } from "zod";
import { collectionID } from "./product-collection";

const digest = z.string().regex(/^[a-f0-9]{64}$/);
const sku = z.string().regex(/^shuomi-[a-f0-9]{32}$/);
const scope = { sourceId: collectionID, storeId: collectionID };
const bounded = z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/);

export const amazonDetailsSchema = z.object({
  productType: z.string().regex(/^[A-Z0-9_]{1,128}$/),
  title: z.string().trim().min(1).max(200),
  description: z.string().trim().min(1).max(2000).optional(),
  brand: z.string().trim().min(1).max(100),
  gtin: z.string().regex(/^(?:[0-9]{12}|[0-9]{13}|[0-9]{14})$/),
  priceCents: z.number().int().positive().max(100_000_000),
  quantity: z.number().int().min(0).max(100_000),
  mainImageId: z.string().min(1).max(128),
}).strict();
export const amazonListingInputSchema = z.object(scope).strict();
export const amazonReviewInputSchema = z.object({
  ...scope, effectiveVersion: z.string().regex(/^[1-9][0-9]{0,18}$/),
  expectedRevision: z.number().int().min(0), details: amazonDetailsSchema,
}).strict();
export const amazonConfirmSchema = z.object({ recordId: collectionID, wireHash: digest }).strict();
export const amazonReviewSchema = z.object({
  ...scope, ...amazonConfirmSchema.shape, sku, revision: z.number().int().positive(),
  validUntil: z.iso.datetime({ offset: true }), details: amazonDetailsSchema,
}).strict();

const issue = z.object({
  code: bounded, severity: z.enum(["ERROR", "WARNING", "INFO"]),
  attributes: z.array(bounded).max(20).optional(),
}).strict();
export const amazonListingResultSchema = z.object({
  ...scope, recordId: collectionID, attemptId: collectionID, sku,
  submission: z.enum(["accepted", "rejected", "unknown", "not_sent"]),
  receipt: z.object({ accepted: z.boolean(), submissionId: bounded, issues: z.array(issue).max(50).optional() }).strict().optional(),
  observation: z.object({
    asin: z.string().regex(/^(?:[A-Z0-9]{10})?$/), buyable: z.boolean(), discoverable: z.boolean(),
    observedAt: z.iso.datetime({offset:true}), issues: z.array(issue).max(50).optional(),
  }).strict().optional(),
  check: z.enum(["not_found", "unavailable"]).optional(),
}).strict().refine(value => value.submission === "accepted" ? value.receipt?.accepted === true
  : value.submission === "rejected" ? value.receipt?.accepted === false && !value.observation
  : !value.receipt);

export type AmazonListingResult = z.infer<typeof amazonListingResultSchema>;
export type AmazonListingReview = z.infer<typeof amazonReviewSchema>;
export const amazonListingActions = ["details", "review", "confirm", "result", "check"] as const;
export type AmazonListingRoute = `amazon-${(typeof amazonListingActions)[number]}`;

export function amazonListingRequest(route: string) {
  if (route === "amazon-review") return amazonReviewInputSchema;
  if (route === "amazon-confirm") return amazonConfirmSchema;
  return amazonListingInputSchema;
}

export function amazonListingResponse(route: string) {
  return route === "amazon-review" || route === "amazon-details" ? amazonReviewSchema : amazonListingResultSchema;
}

export function matchesAmazonListingResponse(route: string, value: unknown, raw: string) {
  try {
    const input = amazonListingRequest(route).parse(JSON.parse(raw));
    const output = amazonListingResponse(route).parse(value);
    if ("recordId" in input) return output.recordId === input.recordId;
    if ("expectedRevision" in input &&
        (!("details" in output) || output.revision !== input.expectedRevision + 1 ||
          JSON.stringify(output.details) !== JSON.stringify(input.details))) return false;
    return output.sourceId === input.sourceId && output.storeId === input.storeId;
  } catch {
    return false;
  }
}
