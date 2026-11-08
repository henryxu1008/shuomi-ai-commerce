import type { z } from "zod";
import type { CockpitScope } from "./operations-cockpit";
import { readBoundedStrictJSON } from "./strict-json-response";

export async function readCockpitMetrics<T>(
  scope: CockpitScope, owner: "supply-preparations" | "tasks" | "walmart-feed" | "tiktok-publish" | "shein-publish", query: string,
  schema: z.ZodType<T>, signal: AbortSignal,
) {
  const path = owner === "walmart-feed" || owner === "tiktok-publish" || owner === "shein-publish"
    ? "supply-preparations/" + owner + "-metrics"
    : owner + "/cockpit-metrics";
  const response = await fetch(`/api/workbench/${path}?${query}`, {
    signal,
    cache: "no-store",
    credentials: "same-origin",
    redirect: "error",
    headers: {
      Accept: "application/json",
      "X-Expected-User-ID": scope.userId,
      "X-Expected-Organization-ID": scope.organizationId,
    },
  });
  if (response.status === 401 || response.status === 403) return { message: "当前身份无查看权限" };
  if (!response.ok) throw new Error("Cockpit metrics unavailable");
  const result = schema.parse(await readBoundedStrictJSON(response, 4096, signal));
  return { result };
}
