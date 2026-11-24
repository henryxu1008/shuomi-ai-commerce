import { randomBytes, randomUUID } from "node:crypto";
import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { serverAuth } from "@/auth";
import { readBoundedStrictJSON } from "@/lib/api/strict-json-response";
import { serviceOrigin } from "@/lib/server/account-proxy";
import { hasTrustedSameOriginWrite } from "@/lib/server/same-origin-write";
import { WORKBENCH_COOKIE_NAME } from "@/lib/server/workbench-proxy";
import { readZitadelIdentityFromSession } from "@/lib/server/zitadel-auth";
import { readZitadelServerAccessToken } from "@/lib/server/zitadel-server-token";

const cookie = "__Host-shuomi-shopify-pairing";
const cookieOptions = { path: "/", httpOnly: true, sameSite: "lax", secure: true } as const;
const domain = z.string().regex(/^[a-z0-9][a-z0-9-]{0,62}\.myshopify\.com$/);
const identifier = z.string().regex(/^[a-zA-Z0-9][a-zA-Z0-9_.:-]{0,199}$/);
const nonce = z.string().regex(/^[a-zA-Z0-9_-]{43}$/);
const pairing = z.strictObject({
  store: z.uuid(), attempt: z.uuid(), organization: identifier, actor: identifier,
  browser: nonce, state: nonce, domain: domain.optional(),
});
type Pairing = z.infer<typeof pairing>;
const startInput = z.strictObject({ store: z.uuid(), version: z.number().int().min(1).max(Number.MAX_SAFE_INTEGER) });
const startReply = z.object({ authorizationUrl: z.url(), attemptId: z.uuid() });
const nextReply = z.object({ authorizationUrl: z.url() });
const shopReply = z.object({ domain });

function json(value: unknown, status = 200) {
  return NextResponse.json(value, { status, headers: { "Cache-Control": "no-store", "Referrer-Policy": "no-referrer" } });
}

function redirect(location: string) {
  return new NextResponse(null, { status: 303, headers: { Location: location, "Cache-Control": "no-store", "Referrer-Policy": "no-referrer" } });
}

function shopifyURL(value: string, kind: "install" | "authorize") {
  const parsed = new URL(value);
  if (parsed.protocol !== "https:" || parsed.username || parsed.password || parsed.port || parsed.hash) return null;
  if (kind === "install") return ["apps.shopify.com", "admin.shopify.com"].includes(parsed.hostname) ? parsed.href : null;
  return domain.safeParse(parsed.hostname).success && parsed.pathname === "/admin/oauth/authorize" ? parsed.href : null;
}

function readPairing(request: NextRequest) {
  const cookies = request.cookies.getAll(cookie);
  if (cookies.length !== 1) return null;
  try {
    return pairing.parse(JSON.parse(Buffer.from(cookies[0]!.value, "base64url").toString("utf8")));
  } catch {
    return null;
  }
}

function savePairing(response: NextResponse, data: Pairing | null) {
  response.cookies.set(cookie, data ? Buffer.from(JSON.stringify(data)).toString("base64url") : "",
    { ...cookieOptions, maxAge: data ? 600 : 0 });
  return response;
}

async function dispatch(token: string, pairing: Pairing, body: unknown, request: NextRequest, version?: number) {
  const origin = serviceOrigin();
  if (!origin) return null;
  const headers = new Headers({
    Authorization: `Bearer ${token}`, "Content-Type": "application/json", Accept: "application/json",
    "X-Requested-Organization-ID": pairing.organization,
  });
  if (version !== undefined) {
    headers.set("If-Match", `"${version}"`);
    headers.set("Idempotency-Key", pairing.attempt);
  }
  const signal = AbortSignal.any([AbortSignal.timeout(18000), request.signal]);
  const response = await fetch(new URL(`/api/v1/workbench/stores/${pairing.store}/shopify-pairing`, origin), {
    method: "POST", headers, body: JSON.stringify(body), signal, cache: "no-store", redirect: "error",
  });
  return { response, body: await readBoundedStrictJSON(response, 8192, signal) };
}

const authenticated = serverAuth(async (request: NextRequest & { auth?: unknown }) => {
  const token = readZitadelServerAccessToken(request.auth as never);
  const actor = identifier.safeParse(readZitadelIdentityFromSession(request.auth as never)?.userId);
  const organizations = request.cookies.getAll(WORKBENCH_COOKIE_NAME);
  const organization = organizations.length === 1 ? organizations[0]?.value : undefined;
  if (!token || !actor.success || !organization || !identifier.safeParse(organization).success) {
    return json({ code: "AUTHENTICATION_REQUIRED" }, 401);
  }
  const action = request.nextUrl.pathname.split("/").pop();
  const mutation = action === "begin" || action === "confirm";
  if (request.method !== (mutation ? "POST" : "GET")) return json({ code: "INVALID_REQUEST" }, 405);
  if (mutation && (!hasTrustedSameOriginWrite(request) || request.headers.get("X-Expected-Organization-ID") !== organization ||
    request.headers.get("X-Expected-User-ID") !== actor.data)) return json({ code: "ORGANIZATION_CONTEXT_CHANGED" }, 409);
  if (action === "begin") {
    const data = startInput.safeParse(await readBoundedStrictJSON(new Response(request.body, { headers: request.headers }), 4096, request.signal));
    if (!data.success) return json({ code: "INVALID_REQUEST" }, 400);
    const flow: Pairing = { store: data.data.store, attempt: randomUUID(), actor: actor.data, organization,
      state: randomBytes(32).toString("base64url"), browser: randomBytes(32).toString("base64url") };
    const started = await dispatch(token, flow, { action: "begin", browser: flow.browser, state: flow.state }, request, data.data.version);
    const result = startReply.safeParse(started?.body);
    const url = result.success ? shopifyURL(result.data.authorizationUrl, "install") : null;
    if (!started?.response.ok || !result.success || result.data.attemptId !== flow.attempt || !url) {
      return json({ code: "SHOPIFY_CONNECTION_UNAVAILABLE" }, 503);
    }
    return savePairing(json({ authorizationUrl: url }), flow);
  }
  const flow = readPairing(request);
  if (!flow || flow.actor !== actor.data || flow.organization !== organization) {
    return action === "launch" || action === "callback"
      ? redirect("/workbench/stores?shopify=restart") : json({ code: "SHOPIFY_RESTART_REQUIRED" }, 409);
  }
  const returning = `/workbench/stores/${flow.store}`;
  if (action === "pending") {
    return json(request.nextUrl.searchParams.get("store") === flow.store && flow.domain ? { domain: flow.domain } : {});
  }
  if (action === "confirm") {
    const data = z.strictObject({ store: z.uuid() }).safeParse(await readBoundedStrictJSON(new Response(request.body, { headers: request.headers }), 4096, request.signal));
    if (!data.success || data.data.store !== flow.store || !flow.domain) return json({ code: "SHOPIFY_RESTART_REQUIRED" }, 409);
    const reply = await dispatch(token, flow, { action: "confirm", browser: flow.browser, attemptId: flow.attempt }, request);
    const result = shopReply.safeParse(reply?.body);
    if (!reply?.response.ok || !result.success || result.data.domain !== flow.domain) return json({ code: "SHOPIFY_RESTART_REQUIRED" }, 409);
    return savePairing(json(result.data), null);
  }
  if (action !== "launch" && action !== "callback") return json({ code: "INVALID_REQUEST" }, 404);
  const query = request.nextUrl.search.slice(1);
  if (!query || query.length > 8192) return redirect(`${returning}?shopify=restart#authorization`);
  const output = await dispatch(token, flow, {
    action: action === "launch" ? "launch" : "stage", browser: flow.browser, state: flow.state, attemptId: flow.attempt, query,
  }, request);
  if (action === "launch") {
    const result = nextReply.safeParse(output?.body);
    const url = result.success ? shopifyURL(result.data.authorizationUrl, "authorize") : null;
    return output?.response.ok && url ? redirect(url) : redirect(`${returning}?shopify=restart#authorization`);
  }
  const staged = shopReply.safeParse(output?.body);
  if (output?.response.ok && staged.success) return savePairing(redirect(`${returning}#authorization`), { ...flow, domain: staged.data.domain });
  const error = z.object({ code: z.string() }).safeParse(output?.body);
  const status = error.success && error.data.code === "SHOPIFY_OWNER_REQUIRED" ? "owner" : "restart";
  return savePairing(redirect(`${returning}?shopify=${status}#authorization`), null);
});

async function handle(request: NextRequest) {
  try {
    return (await authenticated(request, { params: Promise.resolve({}) })) ?? json({ code: "AUTHENTICATION_REQUIRED" }, 401);
  } catch {
    return json({ code: "SHOPIFY_CONNECTION_UNAVAILABLE" }, 503);
  }
}

export const GET = handle;
export const POST = handle;
