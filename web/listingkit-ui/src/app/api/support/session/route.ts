import { createHmac } from "node:crypto";
import { NextRequest } from "next/server";
import { serverAuth } from "@/auth";
import { readBoundedStrictJSON } from "@/lib/api/strict-json-response";
import { readChatwootConfig, supportSubject } from "@/lib/server/chatwoot-support";
import { hasTrustedSameOriginWrite } from "@/lib/server/same-origin-write";
import { authorizeZitadelIdentity, getZitadelAuthOptions, readZitadelIdentityFromSession, readZitadelSessionClientID, readZitadelSessionError, readZitadelSessionIssuerURL } from "@/lib/server/zitadel-auth";
import { readZitadelServerAccessToken } from "@/lib/server/zitadel-server-token";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

const json = (body: unknown, status = 200) => Response.json(body, { status, headers: { "Cache-Control": "no-store" } });
const failure = (status: number, code: string) => json({ code }, status);

const authenticated = serverAuth(async request => {
  if (request.signal.aborted) return failure(504, "SUPPORT_TIMEOUT");
  const config = readChatwootConfig();
  const options = getZitadelAuthOptions();
  if (!config || !options) return failure(503, "SUPPORT_NOT_CONFIGURED");
  const identity = readZitadelIdentityFromSession(request.auth);
  if (!identity || !readZitadelServerAccessToken(request.auth) || readZitadelSessionError(request.auth) || readZitadelSessionIssuerURL(request.auth) !== options.issuerUrl || readZitadelSessionClientID(request.auth) !== options.clientId) return failure(401, "AUTHENTICATION_REQUIRED");
  if (!authorizeZitadelIdentity(identity).authorized) return failure(403, "PERMISSION_DENIED");
  const subject = supportSubject(options.issuerUrl, identity);
  if (request.method === "GET") return json({ origin: config.origin, websiteToken: config.websiteToken, subject });

  let input: unknown;
  try {
    input = await readBoundedStrictJSON(new Response(request.body, { headers: { "Content-Type": "application/json" } }), 4096, request.signal);
  } catch { return failure(request.signal.aborted ? 504 : 400, request.signal.aborted ? "SUPPORT_TIMEOUT" : "INVALID_REQUEST"); }
  if (!input || typeof input !== "object" || Array.isArray(input)) return failure(400, "INVALID_REQUEST");
  const body = input as Record<string, unknown>;
  if (Object.keys(body).length !== 2 || typeof body.subject !== "string" || typeof body.widgetAuthToken !== "string" || !/^[A-Za-z0-9._-]{1,2048}$/.test(body.widgetAuthToken)) return failure(400, "INVALID_REQUEST");
  if (body.subject !== subject) return failure(409, "IDENTITY_CONTEXT_CHANGED");
  if (request.signal.aborted) return failure(504, "SUPPORT_TIMEOUT");
  try {
    const url = new URL("/api/v1/widget/contact/set_user", config.origin);
    url.searchParams.set("website_token", config.websiteToken);
    const upstream = await fetch(url.toString(), {
      method: "PATCH", cache: "no-store", redirect: "manual", credentials: "omit", signal: request.signal,
      headers: { Accept: "application/json", "Content-Type": "application/json", "X-Auth-Token": body.widgetAuthToken },
      body: JSON.stringify({ identifier: subject, name: `平台用户 ${subject.slice(-8)}`, identifier_hash: createHmac("sha256", config.secret).update(subject).digest("hex") }),
    });
    if (!upstream.ok) { void upstream.body?.cancel().catch(() => undefined); return failure(502, "SUPPORT_UNAVAILABLE"); }
    const result = await readBoundedStrictJSON(upstream, 32768, request.signal) as Record<string, unknown> | null;
    if (!result || !Number.isSafeInteger(result.id) || Number(result.id) <= 0 || result.widget_auth_token !== undefined && result.widget_auth_token !== body.widgetAuthToken) return failure(502, "SUPPORT_UNAVAILABLE");
    return json({ ok: true });
  } catch { return failure(request.signal.aborted ? 504 : 502, request.signal.aborted ? "SUPPORT_TIMEOUT" : "SUPPORT_UNAVAILABLE"); }
});

async function handle(request: NextRequest) {
  const reject = (status: number, code: string) => { void request.body?.cancel().catch(() => undefined); return failure(status, code); };
  if (new URL(request.url).search) return reject(400, "INVALID_REQUEST");
  if (request.method === "POST") {
    if (!hasTrustedSameOriginWrite(request)) return reject(403, "PERMISSION_DENIED");
    if (!/^application\/json(?:\s*;|$)/i.test(request.headers.get("content-type") ?? "")) return reject(400, "INVALID_REQUEST");
    const length = request.headers.get("content-length");
    if (length !== null && (!/^\d+$/.test(length) || Number(length) > 4096)) return reject(413, "INVALID_REQUEST");
  }
  const controller = new AbortController();
  const abort = () => controller.abort();
  let finish = () => {};
  const deadline = new Promise<Response>(resolve => { finish = () => resolve(failure(504, "SUPPORT_TIMEOUT")); controller.signal.addEventListener("abort", finish, { once: true }); });
  request.signal.addEventListener("abort", abort, { once: true });
  if (request.signal.aborted) abort();
  const timer = setTimeout(abort, 15_000);
  try {
    const result = await Promise.race([authenticated(new NextRequest(request, { signal: controller.signal }), { params: Promise.resolve({}) }), deadline]);
    return controller.signal.aborted ? failure(504, "SUPPORT_TIMEOUT") : result ?? failure(503, "SUPPORT_UNAVAILABLE");
  } catch { return failure(controller.signal.aborted ? 504 : 503, controller.signal.aborted ? "SUPPORT_TIMEOUT" : "SUPPORT_UNAVAILABLE"); }
  finally { clearTimeout(timer); request.signal.removeEventListener("abort", abort); controller.signal.removeEventListener("abort", finish); }
}

export const GET = handle;
export const POST = handle;
