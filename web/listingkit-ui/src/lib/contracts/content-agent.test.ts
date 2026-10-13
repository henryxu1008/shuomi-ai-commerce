import { expect, it } from "vitest";
import { contentAgentStartRequestSchema, contentAgentResultSchema, agentResultSchema } from "./product-agent";

export const contentResult = {
  executionMode: "default", modeRevision: "0", modeSnapshotRef: { Kind: "agent-execution-mode-v1", ID: "11111111-1111-4111-8111-111111111111", Digest: "a".repeat(64) },
  runId: "11111111-1111-4111-8111-111111111111", requestKey: "22222222-2222-4222-8222-222222222222", operationId: "11111111-1111-4111-8111-111111111111", productKey: "product", catalogVersion: "1", publicationId: "publication", targetPlatform: "product",
  contentSelection: { targetLanguage: "ja", title: true, description: true }, phase: "human_review_required", revision: "2", humanReviewRequired: true, canSubmitReview: true,
  candidate: { Changes: [{ Field: "title", Value: "New title", EvidenceIDs: ["source"] }, { Field: "description", Value: "New description", EvidenceIDs: ["source"] }], Warnings: [], Rejections: [] },
  confidence: [], unresolved: [], steps: [], tokens: 50, estimatedCostMicros: 30, currency: "CNY", usageStatus: "observed",
};
it("freezes explicit content selection and accepts only a complete selected candidate for review", () => {
  expect(contentAgentStartRequestSchema.safeParse({ expectedModeRevision: "0", targetPlatform: "product", contentSelection: contentResult.contentSelection }).success).toBe(true);
  expect(contentAgentStartRequestSchema.safeParse({ expectedModeRevision: "0", targetPlatform: "product" }).success).toBe(false);
  expect(contentAgentResultSchema.safeParse(contentResult).success).toBe(true);
  expect(agentResultSchema.safeParse(contentResult).success).toBe(false);
  for (const changes of [contentResult.candidate.Changes.slice(0, 1), [contentResult.candidate.Changes[0], contentResult.candidate.Changes[0]], [...contentResult.candidate.Changes, { Field: "price", Value: "1", EvidenceIDs: [] }]]) {
    expect(contentAgentResultSchema.safeParse({ ...contentResult, candidate: { ...contentResult.candidate, Changes: changes } }).success).toBe(false);
  }
});
it("shows invalid or unresolved stopped suggestions without approving them", () => {
  expect(contentAgentResultSchema.safeParse({ ...contentResult, phase: "stopped", stopReason: "repair_limit", canSubmitReview: false, candidate: { ...contentResult.candidate, Changes: [{ Field: "invalid", Value: "unknown", EvidenceIDs: [] }] } }).success).toBe(true);
  expect(contentAgentResultSchema.safeParse({ ...contentResult, unresolved: ["missing source"] }).success).toBe(false);
  expect(contentAgentResultSchema.safeParse({ ...contentResult, canSubmitReview: false, unresolved: ["missing source"] }).success).toBe(true);
  const candidate = {...contentResult.candidate, Warnings:[{Code:"missing_material",Field:"description",Message:"Missing material",Metadata:null}]};
  expect(contentAgentResultSchema.safeParse({...contentResult,candidate}).success).toBe(false);
  expect(contentAgentResultSchema.safeParse({...contentResult,candidate,canSubmitReview:false}).success).toBe(true);
});
