import {parseOfflineRecordFailure} from "@/lib/api/offline-shein-record-errors";
import {offlineEndpoint,offlineCreate,offlineSaved,offlineReceipt} from "@/lib/contracts/offline-shein-records";
import {reportID} from "@/lib/contracts/report-center";
import {isOfflineSheinRecordsAvailable} from "./offline-shein-records-availability";
import { readBoundedStrictJSON } from "@/lib/api/strict-json-response";
import { hasTrustedSameOriginWrite } from "./same-origin-write";
import { WORKBENCH_COOKIE_NAME } from "./workbench-proxy";
const safe = { "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff" };
export const offlineFailure = (status: number, code: string) => Response.json({ code }, { status, headers: safe });
export async function proxyOfflineRecords(request: Request, token: string, userId: string): Promise<Response> {
  if(!isOfflineSheinRecordsAvailable())return offlineFailure(503,"DEPENDENCY_UNAVAILABLE");
  if (!token || !userId) return offlineFailure(401, "AUTHENTICATION_REQUIRED");
  if (request.headers.get("X-Expected-User-ID") !== userId) return offlineFailure(409, "IDENTITY_CONTEXT_CHANGED");
  const url = new URL(request.url), route = offlineEndpoint(url, request.method);
  if (!route || request.url.endsWith("?") || request.headers.has("content-encoding")) return offlineFailure(400, "INVALID_REQUEST");
  const write = !!route.input;
  if (write && !hasTrustedSameOriginWrite(request)) return offlineFailure(403, "FORBIDDEN");
  if (!write && (request.body || request.headers.has("transfer-encoding") || request.headers.has("content-length") && request.headers.get("content-length") !== "0")) return offlineFailure(400, "INVALID_REQUEST");
  const cookies = (request.headers.get("cookie") ?? "").split(";").map(v => v.trim()).filter(v => v.startsWith(WORKBENCH_COOKIE_NAME + "="));
  let org = "";
  try { if (cookies.length === 1) org = decodeURIComponent(cookies[0].slice(WORKBENCH_COOKIE_NAME.length + 1)); } catch {}
  if (!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(org) || org !== request.headers.get("X-Expected-Organization-ID")) return offlineFailure(409, "ORGANIZATION_CONTEXT_CHANGED");
  const controller = new AbortController(), abort = () => controller.abort();
  request.signal.addEventListener("abort", abort, { once: true }); if (request.signal.aborted) abort();
  const timer = setTimeout(abort, 12000); let dispatched = false;
  try {
    const headers = new Headers({ Accept: "application/json", Authorization: `Bearer ${token}`, "X-Requested-Organization-ID": org });
    let body: string | undefined, payload: unknown;
    if (write) {
      const key = request.headers.get("Idempotency-Key"); if (!reportID.safeParse(key).success) return offlineFailure(400, "INVALID_REQUEST");
      payload = await readBoundedStrictJSON(new Response(request.body, { headers: { "Content-Type": request.headers.get("Content-Type") ?? "" } }), 2048, controller.signal);
      const checked = route.input!.safeParse(payload); if (!checked.success) return offlineFailure(400, "INVALID_REQUEST");
      body = JSON.stringify(checked.data); headers.set("Idempotency-Key", key!); headers.set("Content-Type", "application/json");
    }
    const base = new URL(process.env.LISTINGKIT_SERVICE_API_BASE ?? "");
    if (!["http:", "https:"].includes(base.protocol) || base.username || base.password || base.search || base.hash || !["/api/v1", "/api/v1/"].includes(base.pathname)) return offlineFailure(503, "DEPENDENCY_UNAVAILABLE");
    controller.signal.throwIfAborted(); dispatched = true;
    const response = await fetch(base.origin + "/api/v1/workbench/offline-shein-records" + route.suffix + url.search, { method: request.method, headers, body, signal: controller.signal, cache: "no-store", redirect: "manual" });
    const raw = await readBoundedStrictJSON(response, (2 << 20)+4096, controller.signal); controller.signal.throwIfAborted();
    if (response.status !== (write?201:200)) {
      const failure = parseOfflineRecordFailure(raw,response.status);
      const status=response.status>=400&&response.status<600?response.status:502;
      if(failure&&!(write&&response.status>=500))return Response.json(failure,{status,headers:safe});
      return offlineFailure(status,write?"OUTCOME_UNKNOWN":"INVALID_UPSTREAM_RESPONSE");
    }
    const checked = route.output.safeParse(raw); if (!checked.success) throw new Error();
    if(write){offlineReceipt.parse(checked.data);offlineCreate.parse(payload);}
    else if(route.suffix!==""){const saved=offlineSaved.parse(checked.data);if(route.suffix.startsWith("/commands/")?saved.operationId!==route.suffix.slice(10):saved.recordId!==route.suffix.slice(1))throw new Error();}
    return Response.json(checked.data, {status:write?201:200, headers: safe });
  } catch { return offlineFailure(controller.signal.aborted ? 504 : dispatched ? 502 : 400, write && dispatched ? "OUTCOME_UNKNOWN" : dispatched ? "DEPENDENCY_UNAVAILABLE" : "INVALID_REQUEST"); }
  finally { clearTimeout(timer); request.signal.removeEventListener("abort", abort); }
}
