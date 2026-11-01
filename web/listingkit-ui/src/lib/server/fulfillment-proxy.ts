import { z } from "zod";
import {
  fulfillmentCapabilities,
  fulfillmentID,
  fulfillmentInput,
  fulfillmentResponseSchema,
} from "@/lib/api/fulfillment";
import { readBoundedStrictJSON } from "@/lib/api/strict-json-response";
import { serviceOrigin } from "./account-proxy";
import { hasTrustedSameOriginWrite } from "./same-origin-write";
import { WORKBENCH_COOKIE_NAME } from "./workbench-proxy";
const failure = (status: number, code: string) =>
  Response.json(
    { code },
    {
      status,
      headers: {
        "Cache-Control": "private, no-store",
        "X-Content-Type-Options": "nosniff",
      },
    },
  );
export async function proxyFulfillment(
  request: Request,
  token: string,
  user: string,
): Promise<Response> {
  if (!token || !fulfillmentID.safeParse(user).success)
    return failure(401, "AUTHENTICATION_REQUIRED");
  if (request.headers.get("X-Expected-User-ID") !== user)
    return failure(409, "IDENTITY_CONTEXT_CHANGED");
  const cookies = (request.headers.get("cookie") ?? "")
    .split(";")
    .map((s) => s.trim())
    .filter((s) => s.startsWith(`${WORKBENCH_COOKIE_NAME}=`));
  let org = "";
  try {
    if (cookies.length === 1)
      org = decodeURIComponent(
        cookies[0].slice(WORKBENCH_COOKIE_NAME.length + 1),
      );
  } catch {}
  if (
    !fulfillmentID.safeParse(org).success ||
    request.headers.get("X-Expected-Organization-ID") !== org
  )
    return failure(409, "ORGANIZATION_CONTEXT_CHANGED");
  const url = new URL(request.url);
  const capability =
    request.method === "GET" &&
    url.pathname === "/api/workbench/fulfillment/capabilities";
  const command =
    request.method === "POST" &&
    url.pathname === "/api/workbench/fulfillment/commands";
  if (
    (!capability && !command) ||
    url.search ||
    request.headers.has("idempotency-key")
  )
    return failure(400, "FULFILLMENT_INVALID");
  if (command && !hasTrustedSameOriginWrite(request))
    return failure(403, "FULFILLMENT_FORBIDDEN");
  if (
    capability &&
    (request.body !== null ||
      request.headers.has("transfer-encoding") ||
      (request.headers.has("content-length") &&
        request.headers.get("content-length") !== "0"))
  )
    return failure(400, "FULFILLMENT_INVALID");
  const origin = serviceOrigin();
  if (!origin) return failure(503, "FULFILLMENT_UNAVAILABLE");
  const signal = AbortSignal.any([request.signal, AbortSignal.timeout(60000)]);
  let input: z.infer<typeof fulfillmentInput> | null = null;
  try {
    if (command)
      input = fulfillmentInput.parse(
        await readBoundedStrictJSON(
          new Response(request.body, { headers: request.headers }),
          256 * 1024,
          signal,
        ),
      );
  } catch {
    return failure(400, "FULFILLMENT_INVALID");
  }
  try {
    const response = await fetch(
      new URL(
        `/api/v1/workbench/fulfillment/${capability ? "capabilities" : "commands"}`,
        origin,
      ),
      {
        method: request.method,
        headers: {
          Authorization: `Bearer ${token}`,
          "X-Requested-Organization-ID": org,
          Accept: "application/json",
          ...(command ? { "Content-Type": "application/json" } : {}),
        },
        body: input ? JSON.stringify(input) : undefined,
        cache: "no-store",
        redirect: "manual",
        signal,
      },
    );
    const value = await readBoundedStrictJSON(
      response,
      2 * 1024 * 1024,
      signal,
    );
    if (!response.ok) {
      const error = z
        .object({ code: z.string().max(80) })
        .strict()
        .safeParse(value);
      return failure(
        [400, 401, 403, 404, 409, 503].includes(response.status)
          ? response.status
          : 503,
        error.success ? error.data.code : "FULFILLMENT_UNKNOWN",
      );
    }
    const parsed = capability
      ? fulfillmentCapabilities.parse(value)
      : fulfillmentResponseSchema(input!.action).parse(value);
    if (input && "operation" in parsed) {
      const expected = ["read", "confirm", "ensure", "cancel"].includes(
        input.action,
      )
        ? input.payload.operation
        : input.operation_id;
      if (parsed.operation !== expected)
        return failure(409, "FULFILLMENT_UNKNOWN");
    }
    if (
      input?.action === "source" &&
      "source" in parsed &&
      "kind" in parsed.source &&
      (parsed.source.id !== input.source_id ||
        parsed.source.kind !== input.kind)
    )
      return failure(409, "FULFILLMENT_UNKNOWN");
    if (
      input?.action === "details" &&
      "source" in parsed &&
      "organization" in parsed.source &&
      (parsed.source.organization !== org ||
        parsed.source.store !== input.store ||
        parsed.source.source_id !== input.source_id ||
        parsed.source.source_kind !== input.kind)
    )
      return failure(409, "FULFILLMENT_UNKNOWN");
    return Response.json(parsed, {
      headers: {
        "Cache-Control": "private, no-store",
        "X-Content-Type-Options": "nosniff",
      },
    });
  } catch {
    return failure(409, "FULFILLMENT_UNKNOWN");
  }
}
