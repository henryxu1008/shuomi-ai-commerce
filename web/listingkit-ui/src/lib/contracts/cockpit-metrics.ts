import { z } from "zod";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";

const count = z.number().int().min(0).max(250);
const capturedAt = z.iso.datetime();
const limited = z.strictObject({ status: z.literal("limited"), capturedAt });

export const listingMetrics = z.discriminatedUnion("status", [
  limited,
  z.strictObject({
    status: z.literal("complete"),
    capturedAt,
    counts: z.strictObject({ pending: count, review: count })
      .refine(value => value.pending + value.review <= 250),
  }),
]);

export const publicationMetrics = z.discriminatedUnion("status", [
  limited,
  z.strictObject({
    status: z.literal("complete"),
    capturedAt,
    counts: z.strictObject({ confirmed: count, unknown: count })
      .refine(value => value.confirmed + value.unknown <= 250),
  }),
]);

export const storeSubmissionMetrics = z.discriminatedUnion("status", [
  limited,
  z.strictObject({
    status: z.literal("complete"),
    capturedAt,
    counts: z.strictObject({ accepted: count, rejected: count, unknown: count })
      .refine(value => value.accepted + value.rejected + value.unknown <= 250),
  }),
]);

function overviewRow(platform: "walmart" | "tiktokshop" | "shein" | "temu") {
 const failureCovered = platform === "walmart" || platform === "tiktokshop";
 const counts = z.strictObject({
  confirmed: count, unknown: count, ...(failureCovered ? { rejected: count } : {}),
 }).refine(value => value.confirmed + value.unknown + ("rejected" in value ? Number(value.rejected) : 0) <= 250);
 return z.discriminatedUnion("status", [
  z.strictObject({ platform: z.literal(platform), status: z.literal("empty"), stores: z.literal(0) }),
  z.strictObject({ platform: z.literal(platform), status: z.enum(["limited", "unavailable"]), stores: z.number().int().min(1).max(500) }),
  z.strictObject({ platform: z.literal(platform), status: z.literal("complete"), stores: z.number().int().min(1).max(500), counts }),
 ]);
}
const overviewShape = {
 capturedAt, rows: z.tuple([overviewRow("walmart"), overviewRow("tiktokshop"), overviewRow("shein"), overviewRow("temu")]),
 excludedStores: z.number().int().min(0).max(500),
};
export const submissionOverviewMetrics = z.discriminatedUnion("status", [
 z.strictObject({ ...overviewShape, status: z.literal("complete"), unknownTotal: z.number().int().min(0).max(1000) })
  .refine(value => value.excludedStores === 0 && value.rows.every(row => row.status === "complete" || row.status === "empty") &&
   value.unknownTotal === value.rows.reduce((sum,row) => sum + (row.status === "complete" ? row.counts.unknown : 0), 0)),
 z.strictObject({ ...overviewShape, status: z.literal("incomplete") })
  .refine(value => value.excludedStores > 0 || value.rows.some(row => row.status === "limited" || row.status === "unavailable")),
]);
export type SubmissionOverview = z.infer<typeof submissionOverviewMetrics>;

export const taskMetrics = z.discriminatedUnion("status", [
  limited,
  z.strictObject({
    status: z.literal("complete"),
    capturedAt,
    counts: z.strictObject({
      RUNNING: count,
      WAITING_CONFIRMATION: count,
      COMPLETED: count,
      ERROR: count,
      PAUSED: count,
      UNKNOWN: count,
    }).refine(value => Object.values(value).reduce((sum, item) => sum + item, 0) <= 250),
  }),
]);

export type TaskCounts = Extract<z.infer<typeof taskMetrics>, { status: "complete" }>["counts"];

export function isCockpitMetricsQuery(url: URL, maxStores = 500) {
  if (url.search.length >= 16000 || /%(?![0-9A-Fa-f]{2})/.test(url.search)) return false;
  const query = url.searchParams;
  if ([...query.keys()].some(key => !["startDate", "endDate", "storeId"].includes(key))) return false;
  for (const key of ["startDate", "endDate"]) {
    const dates = query.getAll(key);
    if (dates.length !== 1 || !z.iso.date().safeParse(dates[0]).success) return false;
  }
  const stores = query.getAll("storeId");
  return stores.length > 0 && stores.length <= maxStores &&
    new Set(stores).size === stores.length && stores.every(isAcquisitionUUID);
}
