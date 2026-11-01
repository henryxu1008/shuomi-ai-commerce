import { z } from "zod";
import { kernelArtifact, kernelCompatible, kernelID, kernelPage, kernelPublish, kernelQuery, kernelResult } from "../contracts/kernel-releases";
import { parseWorkbenchContextPayload } from "../api/workbench-context";
import { readBoundedStrictJSON } from "../api/strict-json-response";
import { configuredAgentConfigurationOrigin } from "./agent-configuration-origin";
import { hasTrustedSameOriginWrite } from "./same-origin-write";

export const kernelFailure = (status: number, code: string) => Response.json({ code }, { status, headers: { "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff" } });
export async function proxyKernelReleases(request: Request, token: string, userId: string): Promise<Response> {
  if (!token || !userId) return kernelFailure(401, "AUTHENTICATION_REQUIRED");
  if (request.headers.get("X-Expected-User-ID") !== userId) return kernelFailure(409, "IDENTITY_CONTEXT_CHANGED");
  const url = new URL(request.url), prefix = "/api/account/kernel-releases";
  if (url.pathname !== prefix && !url.pathname.startsWith(prefix + "/")) return kernelFailure(400, "INVALID_REQUEST");
  const path = url.pathname.slice(prefix.length);
  const publish = request.method === "PUT";
  const parts = path.slice(1).split("/");
  const id = parts[0];
  const list = !path && request.method === "GET";
  const artifact = parts.length === 2 && kernelID.safeParse(id).success && parts[1] === "artifact" && request.method === "GET";
  if (!list && !artifact && !(publish && parts.length === 1 && kernelID.safeParse(id).success)) return kernelFailure(405, "INVALID_REQUEST");
  if (url.search.length > 512 || request.url.endsWith("?") || request.headers.has("content-encoding")) return kernelFailure(400, "INVALID_REQUEST");
  if (publish && (url.search || !hasTrustedSameOriginWrite(request))) return kernelFailure(url.search ? 400 : 403, "INVALID_REQUEST");
  for (const [name, value] of url.searchParams) {
    if (url.searchParams.getAll(name).length !== 1 || !["os", "arch", "clientVersion", ...(list ? ["cursor", "limit"] : [])].includes(name)) return kernelFailure(400, "INVALID_REQUEST");
    if (name === "cursor" && !kernelID.safeParse(value).success || name === "limit" && (!/^[1-9][0-9]?$|^100$/.test(value))) return kernelFailure(400, "INVALID_REQUEST");
  }
  if (!publish && !kernelQuery.safeParse({ os: url.searchParams.get("os"), arch: url.searchParams.get("arch"), clientVersion: url.searchParams.get("clientVersion") }).success) return kernelFailure(400, "INVALID_REQUEST");
  if (!publish && (request.body || request.headers.has("transfer-encoding") || request.headers.has("content-length") && request.headers.get("content-length") !== "0")) return kernelFailure(400, "INVALID_REQUEST");
  const origin = configuredAgentConfigurationOrigin();
  const unavailable = (status = 503) => kernelFailure(status, publish ? "NOT_SENT" : "UNAVAILABLE");
  if (!origin) return unavailable();
  const controller = new AbortController(), abort = () => controller.abort();
  request.signal.addEventListener("abort", abort, { once: true }); if (request.signal.aborted) abort();
  const timer = setTimeout(abort, 15000); let dispatched = false, validated = false;
  try {
    let body: string | undefined;
    if (publish) {
      const raw = await readBoundedStrictJSON(request as unknown as Response, 16 * 1024, controller.signal);
      const parsed = kernelPublish.safeParse(raw); if (!parsed.success) return kernelFailure(400, "INVALID_REQUEST");
      body = JSON.stringify(parsed.data);
    }
    validated = true;
    const headers = new Headers({ Accept: "application/json", Authorization: "Bearer " + token });
    // Consume the current server authorizer's permission, never a renderer role
    // or an enterprise-admin claim. The Publish handler checks it again.
    const contextResponse = await fetch(origin + "/api/v1/workbench/context", { headers, cache: "no-store", redirect: "manual", signal: controller.signal });
    if (contextResponse.status !== 200) { void contextResponse.body?.cancel(); return contextResponse.status === 401 ? kernelFailure(401, "AUTHENTICATION_REQUIRED") : unavailable(); }
    const context = parseWorkbenchContextPayload(await readBoundedStrictJSON(contextResponse, 1024 * 1024, controller.signal));
    if (!context.success) return unavailable(502);
    if (context.data.user.id !== userId) return kernelFailure(409, "IDENTITY_CONTEXT_CHANGED");
    if (!context.data.organizations.some(o => o.permissions.includes("listingkit.platform_admin"))) return kernelFailure(403, "FORBIDDEN");
    if (publish) headers.set("Content-Type", "application/json");
    controller.signal.throwIfAborted(); dispatched = publish;
    const upstream = origin + "/api/v1/" + (publish ? "platform/" : "") + "browser/kernel-releases" + path + url.search;
    const response = await fetch(upstream, { method: request.method, headers, body, cache: "no-store", redirect: "manual", signal: controller.signal });
    const raw = await readBoundedStrictJSON(response, 256 * 1024, controller.signal); controller.signal.throwIfAborted();
    if (response.status !== 200) {
      const error = z.object({ code: z.string().regex(/^[A-Z_]{1,80}$/) }).safeParse(raw);
      return kernelFailure(response.status >= 400 && response.status < 600 ? response.status : 502, publish && response.status >= 500 ? "OUTCOME_UNKNOWN" : error.success ? error.data.code : publish ? "OUTCOME_UNKNOWN" : "UNAVAILABLE");
    }
    const data = list ? kernelPage.parse(raw) : artifact ? kernelArtifact.parse(raw) : kernelResult.parse(raw);
    if (!list && "releaseId" in data && data.releaseId !== id) throw new Error("Mismatched release");
    if (artifact && "os" in data && (data.os !== url.searchParams.get("os") || data.arch !== url.searchParams.get("arch"))) throw new Error("Mismatched platform");
    const rows = "releases" in data ? data.releases : "minClientVersion" in data ? [data] : [];
    if (rows.some(row => row.os !== url.searchParams.get("os") || row.arch !== url.searchParams.get("arch") || !kernelCompatible(url.searchParams.get("clientVersion")!, row))) throw new Error("Incompatible response");
    return Response.json(data, { headers: { "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff" } });
  } catch { return kernelFailure(controller.signal.aborted ? 504 : dispatched ? 502 : validated ? 503 : 400, dispatched ? "OUTCOME_UNKNOWN" : validated || controller.signal.aborted ? publish ? "NOT_SENT" : "UNAVAILABLE" : "INVALID_REQUEST"); }
  finally { clearTimeout(timer); request.signal.removeEventListener("abort", abort); }
}
