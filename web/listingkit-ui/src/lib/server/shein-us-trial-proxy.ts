import { readBoundedStrictJSON } from "@/lib/api/strict-json-response";
import { expectedTrialApprovalHeader, sheinTrialError, sheinTrialReceipt, trialUUID } from "@/lib/contracts/shein-us-trial";
import { serviceOrigin } from "./account-proxy";
import { hasEmptyBody } from "./members-proxy";
import { projectFailure } from "./project-center-proxy";
import { hasTrustedSameOriginWrite } from "./same-origin-write";
import { sheinTrialTarget } from "./shein-us-trial-target";
import { WORKBENCH_COOKIE_NAME } from "./workbench-proxy";

const timeout = { view: 12_000, submit: 42_000, reconcile: 37_000 };

export function sheinTrialPath(pathname: string) {
  const match = /^\/api\/workbench\/research\/projects\/([0-9a-f-]{36})\/shein-us-trial(?:\/(submit|reconcile))?$/.exec(pathname);
  if (!match || !trialUUID.safeParse(match[1]).success) return null;
  const action = match[2] === "submit" ? "submit" : match[2] === "reconcile" ? "reconcile" : "view";
  return { project: match[1], action } as const;
}

export async function proxySheinUSTrial(request: Request, token: string, userId: string) {
  if (!token || !userId) return projectFailure(401, "AUTHENTICATION_REQUIRED");
  if (request.headers.get("X-Expected-User-ID") !== userId) return projectFailure(409, "IDENTITY_CONTEXT_CHANGED");
  const url = new URL(request.url);
  const route = sheinTrialPath(url.pathname);
  const target = sheinTrialTarget();
  if (!route || !target || route.project !== target.project) return projectFailure(404, "SHEIN_TRIAL_UNAVAILABLE");
  const write = route.action !== "view";
  const length = request.headers.get("content-length");
  if (request.method !== (write ? "POST" : "GET") || url.search || request.url.endsWith("?") ||
      (length !== null && length !== "0") || request.headers.has("transfer-encoding") ||
      request.headers.has("content-encoding") || request.headers.has("idempotency-key")) {
    void request.body?.cancel().catch(() => undefined);
    return projectFailure(400, "INVALID_REQUEST");
  }
  if (write && !hasTrustedSameOriginWrite(request)) return projectFailure(403, "FORBIDDEN");
  if (write && request.headers.get(expectedTrialApprovalHeader) !== target.approval)
    return projectFailure(409, "APPROVAL_CONTEXT_CHANGED");

  const selected = (request.headers.get("cookie") ?? "").split(";")
    .map((part) => part.trim()).filter((part) => part.startsWith(WORKBENCH_COOKIE_NAME + "="));
  let organization = "";
  try {
    if (selected.length === 1) organization = decodeURIComponent(selected[0].slice(WORKBENCH_COOKIE_NAME.length + 1));
  } catch { /* Invalid selection is rejected below. */ }
  if (!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(organization) || organization !== request.headers.get("X-Expected-Organization-ID"))
    return projectFailure(409, "ORGANIZATION_CONTEXT_CHANGED");

  const origin = serviceOrigin();
  if (!origin) return projectFailure(503, "SHEIN_TRIAL_UNAVAILABLE");
  const signal = AbortSignal.any([request.signal, AbortSignal.timeout(timeout[route.action])]);
  const suffix = write ? `/${route.action}` : "";
  const destination = `${origin}/api/v1/workbench/research/projects/${target.project}/shein-us-trials/${target.approval}${suffix}`;
  const reply = (value: unknown, status: number) => Response.json(value, {
    status, headers: { "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff" },
  });
  if (signal.aborted) return projectFailure(503, "SHEIN_TRIAL_UNAVAILABLE");
  let dispatched = false;
  try {
    if (!(await hasEmptyBody(request, signal))) return projectFailure(400, "INVALID_REQUEST");
    signal.throwIfAborted();
    dispatched = true;
    const upstream = await fetch(destination, {
      method: request.method,
      headers: { Accept: "application/json", Authorization: `Bearer ${token}`, "X-Requested-Organization-ID": organization },
      signal, cache: "no-store", redirect: "error",
    });
    if (upstream.status === 401 || upstream.status === 403) {
      void upstream.body?.cancel().catch(() => undefined);
      return projectFailure(upstream.status, upstream.status === 401 ? "AUTHENTICATION_REQUIRED" : "FORBIDDEN");
    }
    const payload = await readBoundedStrictJSON(upstream, 4096, signal);
    const error = sheinTrialError.safeParse(payload);
    if (!write && upstream.status === 404 && error.success && error.data.code === "NOT_FOUND")
      return reply({ approvalId: target.approval, state: "NOT_STARTED" }, 200);
    if (upstream.status === 409 && error.success && error.data.code === "SHEIN_TRIAL_CONFLICT")
      return projectFailure(409, "SHEIN_TRIAL_CONFLICT");
    if (upstream.status === 400 && error.success && error.data.code === "INVALID_REQUEST")
      return projectFailure(400, "INVALID_REQUEST");
    const parsed = sheinTrialReceipt.safeParse(payload);
    if (!parsed.success || parsed.data.approvalId !== target.approval ||
        upstream.status !== (write && parsed.data.state === "UNKNOWN" ? 409 : 200) ||
        (route.action !== "submit" && "publishedCount" in parsed.data))
      throw new Error("Unexpected trial response");
    return reply(parsed.data, upstream.status);
  } catch {
    return write && dispatched
      ? projectFailure(502, "OUTCOME_UNKNOWN")
      : projectFailure(503, "SHEIN_TRIAL_UNAVAILABLE");
  }
}
