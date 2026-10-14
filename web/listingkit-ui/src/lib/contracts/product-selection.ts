import { z } from "zod";
import {
  collectionID,
  collectionDetailSchema,
  collectionSourceKindSchema,
} from "./product-collection";
import { configVersion } from "./agent-configuration";
import {
  executionModeSchema,
  modeRevisionSchema,
  modeSnapshotRefSchema,
} from "./execution-mode";
import { materialsStartSchema } from "./listing-materials";
const materialsConfigurationRefSchema =
  materialsStartSchema.shape.configurationSnapshotRef;
const text = (max: number) =>
  z
    .string()
    .refine(
      (v) => new TextEncoder().encode(v).length <= max && !v.includes("\0"),
    );
const count = z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER);
const ref = z.strictObject({
  ItemID: collectionID,
  ItemRevision: configVersion,
  ProductKey: text(128).min(1),
  CatalogVersion: configVersion,
  PublicationID: text(128).min(1),
});
const product = collectionDetailSchema.shape.product.shape;
const pair = z.tuple([text(512), text(512)]);
const value = z
  .union([
    text(512),
    z.number().finite(),
    z.boolean(),
    z.array(text(512)),
    z.array(pair),
    z.strictObject({ Name: text(512), Value: text(512) }),
    z.strictObject({
      amount: z.number().finite().nonnegative(),
      currency: text(16).min(1),
    }),
    product.specifications.unwrap(),
    z.strictObject({
      needs_review: z.boolean(),
      reasons: z.array(text(512)).optional(),
    }),
    product.warnings.unwrap(),
  ])
  .refine((v) => new TextEncoder().encode(JSON.stringify(v)).length <= 512);
export const selectionInspectionSchema = z
  .strictObject({
    products: z
      .array(
        z.strictObject({
          ref,
          sourceKind: collectionSourceKindSchema,
          facts: z
            .array(
              z.strictObject({
                id: z.string().regex(/^fact:[0-9a-f]{64}$/),
                path: text(128).min(1),
                value,
              }),
            )
            .max(32),
          missing: z.array(text(128)).max(32),
        }),
      )
      .min(2)
      .max(5),
  })
  .superRefine((v, ctx) => {
    const ids = new Set<string>(),
      products = new Set<string>();
    for (const p of v.products) {
      if (ids.has(p.ref.ItemID) || products.has(p.ref.ProductKey))
        ctx.addIssue({ code: "custom", message: "候选商品不可重复" });
      ids.add(p.ref.ItemID);
      products.add(p.ref.ProductKey);
    }
    if (new TextEncoder().encode(JSON.stringify(v)).length > 32768)
      ctx.addIssue({ code: "custom", message: "对比事实超过读取限制" });
  });
const selectionChoiceSchema = z.strictObject({
  itemId: collectionID,
  expectedRevision: configVersion,
  originalPublicationId: text(128).min(1),
  originalVersion: configVersion,
});
export const selectionInputSchema = z
  .strictObject({
    contextId: collectionID,
    candidates: z.array(selectionChoiceSchema).min(2).max(5),
  })
  .refine(
    (v) =>
      new Set(v.candidates.map((c) => c.itemId)).size === v.candidates.length,
  );
export const selectionPrepareSchema = selectionInputSchema.safeExtend({
  requestKey: collectionID,
  expectedModeRevision: modeRevisionSchema,
  goalSummary: text(512).refine((v) => v.trim() === v),
});
export const selectionPreparedSchema = z.strictObject({
  contextId: collectionID,
  requestKey: collectionID,
  configurationSnapshotRef: materialsConfigurationRefSchema,
  modeSnapshotRef: modeSnapshotRefSchema,
  modeRevision: modeRevisionSchema,
  maximumTokens: count,
  maximumCostMicros: count,
  maximumCallCostMicros: count,
  currency: text(16).min(1),
});
const start = z.strictObject({
  configurationSnapshotRef: materialsConfigurationRefSchema,
});
const resume = start.extend({
  revision: configVersion,
  feedback: text(512).refine((v) => v.trim() === v),
});
const proposal = z.strictObject({
  overview: text(512).min(1),
  recommendedItemId: collectionID.optional(),
  assessments: z
    .array(
      z.strictObject({
        itemId: collectionID,
        advice: text(512).min(1),
        evidenceIds: z
          .array(z.string().regex(/^fact:[0-9a-f]{64}$/))
          .min(1)
          .max(6),
      }),
    )
    .min(1)
    .max(5),
});
export const selectionRunSchema = z
  .strictObject({
    runId: collectionID,
    contextId: collectionID,
    requestKey: collectionID,
    phase: z.enum([
      "running",
      "interrupted",
      "human_review_required",
      "stopped",
    ]),
    revision: configVersion,
    stopReason: text(128).optional(),
    configurationSnapshotRef: materialsConfigurationRefSchema,
    modeSnapshotRef: modeSnapshotRefSchema,
    executionMode: executionModeSchema,
    modeRevision: modeRevisionSchema,
    goalSummary: text(512),
    inspection: selectionInspectionSchema.optional(),
    proposal: proposal.optional(),
    validated: z.boolean(),
    unresolved: z.array(text(8192)).max(4096),
    tokens: count,
    costMicros: count,
    currency: text(16),
    usageStatus: z.enum(["observed", "unknown_reserved"]),
  })
  .superRefine((r, ctx) => {
    if (!r.validated) {
      if (r.proposal)
        ctx.addIssue({ code: "custom", message: "未通过检查不能标记为建议" });
      return;
    }
    if (
      r.phase !== "human_review_required" ||
      r.stopReason ||
      !r.proposal ||
      !r.inspection
    ) {
      ctx.addIssue({ code: "custom", message: "建议必须来自完整有效运行" });
      return;
    }
    const rows = new Map(r.inspection.products.map((p) => [p.ref.ItemID, p]));
    if (r.proposal.recommendedItemId && !rows.has(r.proposal.recommendedItemId))
      ctx.addIssue({ code: "custom", message: "推荐商品不在候选范围" });
    const seen = new Set<string>();
    for (const a of r.proposal.assessments) {
      const row = rows.get(a.itemId);
      if (
        !row ||
        seen.has(a.itemId) ||
        a.evidenceIds.some((id) => !row.facts.some((f) => f.id === id)) ||
        new Set(a.evidenceIds).size !== a.evidenceIds.length
      )
        ctx.addIssue({ code: "custom", message: "建议引用与实际候选不符" });
      seen.add(a.itemId);
    }
  });
export const selectionRecentSchema = z.strictObject({
  items: z
    .array(
      z.strictObject({
        runId: collectionID,
        operationId: collectionID,
        requestKey: collectionID,
        productKey: z.literal(""),
        targetPlatform: z.literal("product"),
        phase: z.enum([
          "running",
          "interrupted",
          "human_review_required",
          "stopped",
        ]),
        revision: configVersion,
        startedAt: z.string().datetime({ offset: true }),
      }),
    )
    .max(20),
  nextCursor: z.string().max(180),
});
export const selectionIntentSchema = z.discriminatedUnion("kind", [
  z
    .strictObject({
      kind: z.literal("prepare"),
      key: collectionID,
      body: selectionPrepareSchema,
    })
    .refine((v) => v.key === v.body.requestKey),
  z.strictObject({
    kind: z.literal("start"),
    key: collectionID,
    contextId: collectionID,
    body: start,
  }),
  z.strictObject({
    kind: z.literal("resume"),
    key: collectionID,
    contextId: collectionID,
    body: resume,
  }),
]);
export const selectionSavedPreparationSchema = z
  .strictObject({
    input: selectionPrepareSchema,
    prepared: selectionPreparedSchema,
  })
  .refine(
    (v) =>
      v.input.contextId === v.prepared.contextId &&
      v.input.requestKey === v.prepared.requestKey,
  );
export type SelectionIntent = z.infer<typeof selectionIntentSchema>;
export type SelectionInspection = z.infer<typeof selectionInspectionSchema>;
export type SelectionPreparation = z.infer<
  typeof selectionSavedPreparationSchema
>;
export function selectionEndpoint(
  url: URL,
  method: string,
): { path: string; input?: z.ZodType; output: z.ZodType } | null {
  if (
    url.search ||
    url.hash ||
    !url.pathname.startsWith("/api/workbench/product-selection/")
  )
    return null;
  const p = url.pathname
    .slice("/api/workbench/product-selection/".length)
    .split("/");
  if (p.length === 1 && method === "POST")
    return p[0] === "inspect"
      ? {
          path: p.join("/"),
          input: selectionInputSchema,
          output: selectionInspectionSchema,
        }
      : p[0] === "prepare"
        ? {
            path: p.join("/"),
            input: selectionPrepareSchema,
            output: selectionPreparedSchema,
          }
        : null;
  if (
    p[0] !== "contexts" ||
    !collectionID.safeParse(p[1]).success ||
    p[2] !== "runs"
  )
    return null;
  if (p.length === 3 && method === "POST")
    return { path: p.join("/"), input: start, output: selectionRunSchema };
  if (
    p.length === 4 &&
    method === "GET" &&
    collectionID.safeParse(p[3]).success
  )
    return { path: p.join("/"), output: selectionRunSchema };
  if (
    p.length === 5 &&
    method === "POST" &&
    collectionID.safeParse(p[3]).success &&
    p[4] === "resume"
  )
    return { path: p.join("/"), input: resume, output: selectionRunSchema };
  return null;
}
