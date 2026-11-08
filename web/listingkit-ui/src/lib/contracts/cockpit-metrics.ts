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

export function isCockpitMetricsQuery(url: URL) {
  if (url.search.length >= 16000 || /%(?![0-9A-Fa-f]{2})/.test(url.search)) return false;
  const query = url.searchParams;
  if ([...query.keys()].some(key => !["startDate", "endDate", "storeId"].includes(key))) return false;
  for (const key of ["startDate", "endDate"]) {
    const dates = query.getAll(key);
    if (dates.length !== 1 || !z.iso.date().safeParse(dates[0]).success) return false;
  }
  const stores = query.getAll("storeId");
  return stores.length > 0 && stores.length <= 500 &&
    new Set(stores).size === stores.length && stores.every(isAcquisitionUUID);
}
