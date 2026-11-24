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
const installCookie = "__Host-shuomi-shopify-started-install";
const cookieOptions = { path: "/", httpOnly: true, sameSite: "lax", secure: true } as const;
const domain = z.string().regex(/^[a-z0-9][a-z0-9-]{0,62}\.myshopify\.com$/);
const identifier = z.string().regex(/^[a-zA-Z0-9][a-zA-Z0-9_.:-]{0,199}$/);
const nonce = z.string().regex(/^[a-zA-Z0-9_-]{43}$/);
const pairing = z.strictObject({
  store: z.uuid(), attempt: z.uuid(), organization: identifier, actor: identifier,
  browser: nonce, state: nonce, domain: domain.optional(), launched: z.boolean().optional(),
});
type Pairing = z.infer<typeof pairing>;
const installProof = z.strictObject({
  attempt: z.uuid(), browser: nonce, state: nonce.optional(),
});
type InstallProof = z.infer<typeof installProof>;
const startInput = z.strictObject({ store: z.uuid(), version: z.number().int().min(1).max(Number.MAX_SAFE_INTEGER) });
const startReply = z.object({ authorizationUrl: z.url(), attemptId: z.uuid() });
const nextReply = z.object({ authorizationUrl: z.url() });
const shopReply = z.object({ domain });
const confirmInstallInput = startInput.extend({ attempt: z.uuid() });
const installSelection = shopReply.extend({
  storeVersion: z.number().int().min(1).max(Number.MAX_SAFE_INTEGER),
  confirmed: z.boolean(),
});

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

function readInstallProof(request: NextRequest) {
  const values = request.cookies.getAll(installCookie);
  if (values.length !== 1 || !values[0] || values[0].value.length > 384) return null;
  try {
    return installProof.parse(JSON.parse(Buffer.from(values[0].value, "base64url").toString("utf8")));
  } catch {
    return null;
  }
}

function saveInstallProof(response: NextResponse, proof: InstallProof | null) {
  response.cookies.set(installCookie, proof ? Buffer.from(JSON.stringify(proof)).toString("base64url") : "",
    { ...cookieOptions, maxAge: proof ? 600 : 0 });
  return response;
}

async function dispatch(token: string, pairing: Pick<Pairing, "organization" | "store" | "attempt">, body: unknown, request: NextRequest, version?: number) {
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

async function shopifyStarted(request: NextRequest, action: "launch" | "callback") {
  const query = request.nextUrl.search.slice(1);
  const origin = serviceOrigin();
  if (!origin || !query || query.length > 8192) return redirect("/workbench/stores?shopify=restart");
  const proof = action === "launch" ? {
    attempt: randomUUID(), browser: randomBytes(32).toString("base64url"),
    state: randomBytes(32).toString("base64url"),
  } : readInstallProof(request);
  if (!proof?.state) return redirect("/workbench/stores?shopify=restart");

  const signal = AbortSignal.any([AbortSignal.timeout(18000), request.signal]);
  const response = await fetch(new URL("/api/v1/platform/shopify/installation", origin), {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify({ action: action === "launch" ? "launch" : "stage",
      attemptId: proof.attempt, browser: proof.browser, state: proof.state, query }),
    signal, cache: "no-store", redirect: "error",
  });
  const result = await readBoundedStrictJSON(response, 8192, signal);
  if (action === "launch" && response.ok) {
    const next = nextReply.safeParse(result);
    const url = next.success ? shopifyURL(next.data.authorizationUrl, "authorize") : null;
    if (url) return saveInstallProof(redirect(url), proof);
  }
  if (action === "callback" && response.ok && shopReply.safeParse(result).success) {
    return saveInstallProof(redirect("/workbench/stores?platform=shopify&shopify=pair"),
      { attempt: proof.attempt, browser: proof.browser });
  }
  const error = z.object({ code: z.string() }).safeParse(result);
  if (action === "callback" && response.status === 409 && error.success && error.data.code === "SHOPIFY_INSTALL_CHECKING") {
    return new NextResponse("正在核对 Shopify 授权。请稍后刷新此页。", {
      status: 409, headers: { "Content-Type": "text/plain; charset=utf-8", "Cache-Control": "no-store", "Referrer-Policy": "no-referrer" },
    });
  }
  const reason = error.success && error.data.code === "SHOPIFY_OWNER_REQUIRED" ? "owner" : "restart";
  return saveInstallProof(redirect("/workbench/stores?shopify=" + reason), null);
}

const authenticated = serverAuth(async (request: NextRequest & { auth?: unknown }) => {
  const token = readZitadelServerAccessToken(request.auth as never);
  const actor = identifier.safeParse(readZitadelIdentityFromSession(request.auth as never)?.userId);
  const organizations = request.cookies.getAll(WORKBENCH_COOKIE_NAME);
  const organization = organizations.length === 1 ? organizations[0]?.value : undefined;
  const action = request.nextUrl.pathname.split("/").pop();
  if (!token || !actor.success || !organization || !identifier.safeParse(organization).success) {
    if (action === "launch" || action === "callback") {
      return savePairing(redirect("/workbench/stores?shopify=restart"), null);
    }
    return json({ code: "AUTHENTICATION_REQUIRED" }, 401);
  }
  const mutation = action === "begin" || action === "confirm" || action === "confirm-install";
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
  const startedInstall = readInstallProof(request);
  if (startedInstall && !startedInstall.state && action === "pending" && request.nextUrl.searchParams.get("flow") !== "selected") {
    const store = z.uuid().safeParse(request.nextUrl.searchParams.get("store"));
    if (!store.success) return json({});
    const reply = await dispatch(token, { organization, store: store.data, attempt: startedInstall.attempt },
      { action: "inspect_install", browser: startedInstall.browser, attemptId: startedInstall.attempt }, request);
    const result = installSelection.safeParse(reply?.body);
    if (!reply?.response.ok || !result.success) return json({});
    if (result.data.confirmed) return saveInstallProof(json({}), null);
    return json({ domain: result.data.domain, attempt: startedInstall.attempt, version: result.data.storeVersion });
  }
  if (action === "confirm-install") {
    const choice = confirmInstallInput.safeParse(await readBoundedStrictJSON(new Response(request.body, { headers: request.headers }), 4096, request.signal));
    if (!choice.success) return json({ code: "INVALID_REQUEST" }, 400);
    if (!startedInstall || startedInstall.state || startedInstall.attempt !== choice.data.attempt) {
      return json({ code: "SHOPIFY_RESTART_REQUIRED" }, 409);
    }
    const reply = await dispatch(token, { organization, store: choice.data.store, attempt: startedInstall.attempt },
      { action: "confirm_install", browser: startedInstall.browser }, request, choice.data.version);
    const result = shopReply.safeParse(reply?.body);
    if (!reply?.response.ok || !result.success) return json({ code: "SHOPIFY_RESTART_REQUIRED" }, 409);
    return saveInstallProof(json({ connected: true }), null);
  }
  const flow = readPairing(request);
  if (!flow || flow.actor !== actor.data || flow.organization !== organization) {
    return action === "launch" || action === "callback"
      ? savePairing(redirect("/workbench/stores?shopify=restart"), null) : json({ code: "SHOPIFY_RESTART_REQUIRED" }, 409);
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
  const selectedStore = returning + "?shopify=selected#authorization";
  if (action === "callback" && flow.domain) return redirect(selectedStore);
  const query = request.nextUrl.search.slice(1);
  if (!query || query.length > 8192) return redirect(`${returning}?shopify=restart#authorization`);
  const output = await dispatch(token, flow, {
    action: action === "launch" ? "launch" : "stage", browser: flow.browser, state: flow.state, attemptId: flow.attempt, query,
  }, request);
  if (action === "launch") {
    const result = nextReply.safeParse(output?.body);
    const url = result.success ? shopifyURL(result.data.authorizationUrl, "authorize") : null;
    return output?.response.ok && url ? savePairing(redirect(url), { ...flow, launched: true }) : savePairing(redirect(`${returning}?shopify=restart#authorization`), null);
  }
  const staged = shopReply.safeParse(output?.body);
  if (output?.response.ok && staged.success) return savePairing(redirect(selectedStore), { ...flow, domain: staged.data.domain });
  const error = z.object({ code: z.string() }).safeParse(output?.body);
  const status = error.success && error.data.code === "SHOPIFY_OWNER_REQUIRED" ? "owner" : "restart";
  return savePairing(redirect(`${returning}?shopify=${status}#authorization`), null);
});

async function handle(request: NextRequest) {
  try {
    const action = request.nextUrl.pathname.split("/").pop();
    if (request.method === "GET" && action === "launch") {
      const selected = readPairing(request);
      if (selected?.launched || selected?.domain || !request.cookies.has(cookie)) {
        return await shopifyStarted(request, action);
      }
    }
    if (request.method === "GET" && action === "callback") {
      // The query only selects a browser flow. Each backend still verifies the
      // original Shopify HMAC and atomically claims its own stored state.
      const states = request.nextUrl.searchParams.getAll("state");
      const state = states.length === 1 ? states[0] : null;
      if (!state) return redirect("/workbench/stores?shopify=restart");
      const selected = readPairing(request);
      const fromShopify = readInstallProof(request);
      const selectedMatch = state === selected?.state;
      const shopifyMatch = state === fromShopify?.state;
      if (selectedMatch === shopifyMatch) return redirect("/workbench/stores?shopify=restart");
      if (shopifyMatch) return await shopifyStarted(request, action);
    }
    return (await authenticated(request, { params: Promise.resolve({}) })) ?? json({ code: "AUTHENTICATION_REQUIRED" }, 401);
  } catch {
    return json({ code: "SHOPIFY_CONNECTION_UNAVAILABLE" }, 503);
  }
}

export const GET = handle;
export const POST = handle;
