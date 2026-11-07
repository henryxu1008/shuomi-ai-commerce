import { z } from "zod";
import { announcementContent, announcementPage, announcementPublish, announcementWithdraw, notificationCommandSchema, notificationItemSchema, notificationListSchema, notificationRef, notificationSnapshotSchema, notificationUUID } from "@/lib/api/notifications";
import { readBoundedStrictJSON } from "@/lib/api/strict-json-response";
import { hasTrustedSameOriginWrite } from "./same-origin-write";
import { WORKBENCH_COOKIE_NAME } from "./workbench-proxy";
import { hasEmptyBody } from "./members-proxy";
const safeHeaders = { "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff" };
export function notificationFailure(status: number, code: string) { return Response.json({ code }, { status, headers: safeHeaders }); }

const platformPath = "/api/v1/platform/notifications/official";
function announcementEndpoint(url: URL, method: string, parts: string[]) {
  const [, , id, action] = parts;
  if (url.toString().endsWith("?") || url.search.length > 64) return null;
  if (method === "GET" && parts.length === 2) {
    const after = url.searchParams.get("after");
    if (url.search && (!notificationUUID.safeParse(after).success || [...url.searchParams].length !== 1)) return null;
    return { path: platformPath, schema: announcementPage };
  }
  if (url.search) return null;
  if (method === "GET" && parts.length === 3 && notificationUUID.safeParse(id).success) {
    return { path: platformPath + "/" + id, schema: announcementContent };
  }
  if (method === "POST" && parts.length === 2) {
    return { path: platformPath, schema: notificationCommandSchema, input: announcementPublish, operation: "publish" };
  }
  if (method === "POST" && parts.length === 4 && action === "withdraw" && notificationUUID.safeParse(id).success) {
    return { path: platformPath + "/" + id + "/withdraw", schema: notificationCommandSchema, input: announcementWithdraw, operation: "withdraw", resultId: id };
  }
  return null;
}

function endpoint(url: URL, method: string) {
  const parts = url.pathname.slice("/api/notifications/".length).split("/");
  const [channel, action, id] = parts;
  if (channel === "platform" && action === "official") {
    const route = announcementEndpoint(url, method, parts);
    return route && { ...route, channel };
  }
  if (!["official", "business", "personal"].includes(channel) || !["GET", "POST"].includes(method) || url.toString().endsWith("?")) return null;
  let schema: z.ZodType, input: z.ZodType | undefined;
  if (parts.length === 1 && method === "GET") schema = notificationListSchema;
  else if (parts.length === 2 && method === "GET" && notificationRef.safeParse(action).success) schema = notificationItemSchema;
  else if (parts.length === 3 && method === "GET" && action === "commands" && notificationUUID.safeParse(id).success) schema = notificationCommandSchema;
  else if (parts.length === 2 && method === "POST" && action === "snapshot") { schema = notificationSnapshotSchema; input = z.object({}).strict(); }
  else if (parts.length === 2 && method === "POST" && action === "read") { schema = notificationCommandSchema; input = z.object({ ref: notificationRef }).strict(); }
  else if (parts.length === 2 && method === "POST" && action === "read-all") { schema = notificationCommandSchema; input = z.object({ id: notificationUUID, fingerprint: z.string().regex(/^[0-9a-f]{64}$/) }).strict(); }
  else return null;
  if (url.search && parts.length !== 1 || url.search.length > 4608) return null;
  for (const [key, value] of url.searchParams) {
    if (url.searchParams.getAll(key).length !== 1 || !["filter", "after", "limit"].includes(key)) return null;
    if (key === "filter" && !["all", "unread", "pending"].includes(value) || key === "after" && !/^[A-Za-z0-9_-]{1,4096}$/.test(value) || key === "limit" && (!/^[1-9][0-9]*$/.test(value) || Number(value) > 100)) return null;
    if (key === "filter" && value === "pending" && channel === "official") return null;
  }
  return { channel, schema, input, operation: action, resultId: undefined, path: (channel === "business" ? "/api/v1/workbench/notifications" : "/api/v1/notifications/" + channel) + (parts.length > 1 ? "/" + parts.slice(1).join("/") : "") };
}
export async function proxyNotifications(request: Request, token: string, userId: string, dispatch = { forwarded: false }): Promise<Response> {
  if (!token || !userId) return notificationFailure(401, "AUTHENTICATION_REQUIRED");
  if (request.headers.get("X-Expected-User-ID") !== userId) return notificationFailure(409, "IDENTITY_CONTEXT_CHANGED");
  const url = new URL(request.url), route = endpoint(url, request.method);
  if (!route || request.headers.has("content-encoding")) return notificationFailure(400, "INVALID_REQUEST");
  if (route.input && !hasTrustedSameOriginWrite(request)) return notificationFailure(403, "FORBIDDEN");
  let organization = "";
  if (route.channel === "business") {
    const cookies = (request.headers.get("cookie") ?? "").split(";").map(v => v.trim()).filter(v => v.startsWith(WORKBENCH_COOKIE_NAME + "="));
    try { if (cookies.length === 1) organization = decodeURIComponent(cookies[0].slice(WORKBENCH_COOKIE_NAME.length + 1)); } catch { /* rejected below */ }
    if (!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(organization) || organization !== request.headers.get("X-Expected-Organization-ID")) return notificationFailure(409, "ORGANIZATION_CONTEXT_CHANGED");
  }
  let origin: string;
  try { const upstream = new URL(process.env.LISTINGKIT_SERVICE_API_BASE ?? ""); if (!["http:", "https:"].includes(upstream.protocol) || upstream.username || upstream.password || upstream.search || upstream.hash || !["/api/v1", "/api/v1/"].includes(upstream.pathname)) throw new Error(); origin = upstream.origin; }
  catch { return notificationFailure(503, "NOTIFICATION_UNAVAILABLE"); }
  const controller = new AbortController(); const abort = () => controller.abort(); request.signal.addEventListener("abort", abort, { once: true }); if (request.signal.aborted) abort();
  const timer = setTimeout(abort, 12000);
  try {
    const headers = new Headers({ Accept: "application/json", Authorization: "Bearer " + token });
    if (organization) headers.set("X-Requested-Organization-ID", organization);
    let body: string | undefined;
    if (route.input) {
      const key = request.headers.get("Idempotency-Key"); if (!notificationUUID.safeParse(key).success) return notificationFailure(400, "INVALID_REQUEST");
      const bodyTimer = setTimeout(abort, 5000);
      try { const raw = await readBoundedStrictJSON(new Response(request.body, { headers: { "Content-Type": request.headers.get("Content-Type") ?? "" } }), route.operation === "publish" ? 32768 : 16384, controller.signal); body = JSON.stringify(route.input.parse(raw)); }
      finally { clearTimeout(bodyTimer); }
      headers.set("Content-Type", "application/json"); headers.set("Idempotency-Key", key!);
    } else if (!(await hasEmptyBody(request, controller.signal))) return notificationFailure(400, "INVALID_REQUEST");
    controller.signal.throwIfAborted(); dispatch.forwarded = true;
    const response = await fetch(origin + route.path + url.search, { method: request.method, headers, body, signal: controller.signal, cache: "no-store", redirect: "manual" });
    if (route.channel === "platform" && response.status === 401) return notificationFailure(401, "AUTHENTICATION_REQUIRED");
    if (route.channel === "platform" && response.status === 403) return notificationFailure(403, "FORBIDDEN");
    const payload = await readBoundedStrictJSON(response, 128 * 1024, controller.signal); controller.signal.throwIfAborted();
    if (response.status !== 200) {
      if (route.input && response.status >= 500) return notificationFailure(response.status, "OUTCOME_UNKNOWN");
      const parsed = z.object({ code: z.string().regex(/^[A-Z_]{1,80}$/) }).strict().safeParse(payload);
      if (response.status < 400 || response.status >= 600 || !parsed.success) throw new Error("invalid error response");
      return notificationFailure(response.status, parsed.data.code);
    }
    const parsed = route.schema.parse(payload);
    if (route.schema === notificationItemSchema || route.schema === notificationListSchema) {
      const items = route.schema === notificationListSchema ? (parsed as z.infer<typeof notificationListSchema>).items : [parsed as z.infer<typeof notificationItemSchema>];
      if (items.some(item => item.category !== (route.channel === "official" ? "official" : "business") || item.organizationId !== "" && item.organizationId !== organization)) throw new Error("wrong scope");
    }
    if (route.schema === notificationCommandSchema) {
      const command = parsed as z.infer<typeof notificationCommandSchema>;
      if (command.key !== (route.input ? request.headers.get("Idempotency-Key") : url.pathname.split("/").at(-1)) || route.input && command.operation !== route.operation) throw new Error("wrong receipt");
      if (route.channel === "platform" && (!command.resultId || route.resultId && command.resultId !== route.resultId)) throw new Error("wrong announcement");
    }
    return Response.json(parsed, { headers: safeHeaders });
  } catch { return notificationFailure(controller.signal.aborted ? 504 : dispatch.forwarded ? 502 : 400, dispatch.forwarded && route.input ? "OUTCOME_UNKNOWN" : dispatch.forwarded ? "NOTIFICATION_UNAVAILABLE" : "INVALID_REQUEST"); }
  finally { clearTimeout(timer); request.signal.removeEventListener("abort", abort); }
}
