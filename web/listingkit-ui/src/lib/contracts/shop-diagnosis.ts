import { z } from "zod";
import { collectionID } from "./product-collection";
import { cockpitPeriod } from "./operations-cockpit";
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
export const diagnosisInspectionSchema = z
  .strictObject({
    ref: z
      .strictObject({
        StoreID: collectionID,
        StartDate: z.string(),
        EndDate: z.string(),
        EvidenceDigest: z.string().regex(/^[0-9a-f]{64}$/),
      })
      .refine((v) => validPeriod(v.StartDate, v.EndDate)),
    facts: z
      .array(
        z.strictObject({
          id: z.string().regex(/^fact:[0-9a-f]{64}$/),
          path: text(128).min(1),
          value: text(512).min(1),
        }),
      )
      .max(32),
    missing: z.array(text(128)).max(32),
  })
  .refine(
    (v) =>
      new TextEncoder().encode(JSON.stringify(v)).length <= 32768 &&
      new Set(v.facts.map((f) => f.path)).size === v.facts.length,
  );
export const diagnosisInputSchema = z
  .strictObject({
    contextId: collectionID,
    storeId: collectionID,
    startDate: z.string(),
    endDate: z.string(),
  })
  .refine((v) => validPeriod(v.startDate, v.endDate));
const diagnosisPrepareSchema = diagnosisInputSchema.safeExtend({
  requestKey: collectionID,
  expectedModeRevision: modeRevisionSchema,
  goalSummary: text(512).refine((v) => v.trim() === v),
});
export const diagnosisPreparedSchema = z.strictObject({
  contextId: collectionID,
  requestKey: collectionID,
  inspectionRef: diagnosisInspectionSchema.shape.ref,
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
  recommendations: z
    .array(
      z.strictObject({
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
export const diagnosisRunSchema = z
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
    inspection: diagnosisInspectionSchema.optional(),
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
    const ids = new Set(r.inspection.facts.map((f) => f.id));
    for (const a of r.proposal.recommendations) {
      if (
        a.evidenceIds.some((id) => !ids.has(id)) ||
        new Set(a.evidenceIds).size !== a.evidenceIds.length
      )
        ctx.addIssue({ code: "custom", message: "建议必须引用本次实际事实" });
    }
  });
export const diagnosisRecentSchema = z.strictObject({
  items: z
    .array(
      z.strictObject({
        runId: collectionID,
        operationId: collectionID,
        requestKey: collectionID,
        productKey: z.literal(""),
        targetPlatform: z.literal("business"),
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
export const diagnosisIntentSchema = z.discriminatedUnion("kind", [
  z
    .strictObject({
      kind: z.literal("prepare"),
      key: collectionID,
      body: diagnosisPrepareSchema,
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
export const diagnosisSavedPreparationSchema = z
  .strictObject({
    input: diagnosisPrepareSchema,
    prepared: diagnosisPreparedSchema,
  })
  .refine(
    (v) =>
      v.input.contextId === v.prepared.contextId &&
      v.input.requestKey === v.prepared.requestKey &&
      v.input.storeId === v.prepared.inspectionRef.StoreID &&
      v.input.startDate === v.prepared.inspectionRef.StartDate &&
      v.input.endDate === v.prepared.inspectionRef.EndDate,
  );
export type DiagnosisIntent = z.infer<typeof diagnosisIntentSchema>;
export type DiagnosisInspection = z.infer<typeof diagnosisInspectionSchema>;
export type DiagnosisPreparation = z.infer<
  typeof diagnosisSavedPreparationSchema
>;
export function diagnosisEndpoint(
  url: URL,
  method: string,
): { path: string; input?: z.ZodType; output: z.ZodType } | null {
  if (
    url.search ||
    url.hash ||
    !url.pathname.startsWith("/api/workbench/shop-diagnosis/")
  )
    return null;
  const p = url.pathname
    .slice("/api/workbench/shop-diagnosis/".length)
    .split("/");
  if (p.length === 1 && method === "POST")
    return p[0] === "inspect"
      ? {
          path: p.join("/"),
          input: diagnosisInputSchema,
          output: diagnosisInspectionSchema,
        }
      : p[0] === "prepare"
        ? {
            path: p.join("/"),
            input: diagnosisPrepareSchema,
            output: diagnosisPreparedSchema,
          }
        : null;
  if (
    p[0] !== "contexts" ||
    !collectionID.safeParse(p[1]).success ||
    p[2] !== "runs"
  )
    return null;
  if (p.length === 3 && method === "POST")
    return { path: p.join("/"), input: start, output: diagnosisRunSchema };
  if (
    p.length === 4 &&
    method === "GET" &&
    collectionID.safeParse(p[3]).success
  )
    return { path: p.join("/"), output: diagnosisRunSchema };
  if (
    p.length === 5 &&
    method === "POST" &&
    collectionID.safeParse(p[3]).success &&
    p[4] === "resume"
  )
    return { path: p.join("/"), input: resume, output: diagnosisRunSchema };
  return null;
}

function validPeriod(startDate: string, endDate: string) {
  const parsed = cockpitPeriod.safeParse({ startDate, endDate });
  const days =
    (Date.parse(endDate + "T00:00:00Z") -
      Date.parse(startDate + "T00:00:00Z")) /
    86400000;
  return (
    parsed.success &&
    startDate.slice(0, 4) !== "0000" &&
    days >= 0 &&
    days < 366
  );
}
