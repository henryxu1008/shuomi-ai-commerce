import { z } from "zod";
import { googleContracts, type GoogleAction } from "@/lib/contracts/google-merchant";
import { readBoundedStrictJSON } from "./strict-json-response";

export type GoogleScope = { userId: string; organizationId: string };
export type GoogleResponse<A extends GoogleAction> = z.output<(typeof googleContracts)[A]["response"]>;

export class GoogleRequestError extends Error {
  constructor(readonly code: string) {
    super("Google Merchant request failed");
  }
}

export async function googleRequest<A extends GoogleAction>(
  scope: GoogleScope, action: A, id: string, body?: unknown, key?: string,
): Promise<GoogleResponse<A>> {
  const contract = googleContracts[action];
  const listing = action === "prepare" || action === "confirm" || action === "result" || action === "options";
  const base = listing ? "/api/workbench/supply-preparations/google" : "/api/workbench/stores/" + z.uuid().parse(id) + "/connection/google";
  const path = action === "status" ? base : base + "/" + action + (action === "options" ? "/" + z.uuid().parse(id) : "");
  const headers: Record<string, string> = {
    "X-Expected-User-ID": scope.userId,
    "X-Expected-Organization-ID": scope.organizationId,
    Accept: "application/json",
  };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (key) headers["Idempotency-Key"] = key;
  const response = await fetch(path, { method: contract.method, headers, credentials: "same-origin", cache: "no-store",
    body: body === undefined ? undefined : JSON.stringify(body) });
  const payload = await readBoundedStrictJSON(response, 256 * 1024);
  if (!response.ok) {
    const parsed = z.object({ code: z.string() }).safeParse(payload);
    throw new GoogleRequestError(parsed.success ? parsed.data.code : "DEPENDENCY_UNAVAILABLE");
  }
  return contract.response.parse(payload) as GoogleResponse<A>;
}
