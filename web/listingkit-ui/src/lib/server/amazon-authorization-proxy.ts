import { NextResponse } from "next/server";

import { AMAZON_AUTHORIZATION_PATH, AMAZON_AUTHORIZATION_PRIVACY_HEADERS as privacyHeaders } from "@/lib/contracts/amazon-store-authorization";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
import { resolvePublicAppOrigin } from "@/lib/server/public-app-origin";
import {
  getWorkbenchServiceRoot,
  readBodyWithinLimit,
  readStrictWorkbenchOrganization,
  workbenchProtocolError,
} from "@/lib/server/workbench-proxy";

const flowCookie = "__Secure-shuomi-amazon-flow";
const flowValue = /^[A-Za-z0-9_-]{42}[AEIMQUYcgkosw048]$/;
const appId = /^amzn1\.sellerapps\.app\.[a-f0-9-]{36}$/;
const callbackParameters = {
  login: { amazon_callback_uri: "callbackUri", amazon_state: "amazonState", selling_partner_id: "sellerId", version: "version" },
  redirect: { state: "state", selling_partner_id: "sellerId", spapi_oauth_code: "code" },
} as const;

type WorkbenchCredentials = { actor: string; accessToken: string };

export async function proxyAmazonAuthorization(
  request: Request,
  phase: string,
  credentials: WorkbenchCredentials,
  fetchImpl: typeof fetch = fetch,
): Promise<Response> {
  if (phase !== "begin" && phase !== "login" && phase !== "redirect") {
    return amazonFailure(phase, 404, "INVALID_REQUEST");
  }
  const url = new URL(request.url);
  if (url.pathname !== `${AMAZON_AUTHORIZATION_PATH}/${phase}` ||
      request.method !== (phase === "begin" ? "POST" : "GET")) {
    return amazonFailure(phase, 404, "INVALID_REQUEST");
  }
  if (!credentials.actor || !credentials.accessToken) {
    return amazonFailure(phase, 401, "AUTHENTICATION_REQUIRED");
  }
  const organization = readStrictWorkbenchOrganization(request);
  if (typeof organization !== "string" || !organization) {
    return amazonFailure(phase, 409, "ORGANIZATION_CONTEXT_CHANGED");
  }
  const headers = new Headers({
    Accept: "application/json",
    Authorization: `Bearer ${credentials.accessToken}`,
    "X-Requested-Organization-ID": organization,
  });
  if (phase === "begin") {
    return beginAmazonAuthorization(request, url, credentials.actor, organization, headers, fetchImpl);
  }
  const cookie = readFlowCookie(request.headers);
  const values = readCallbackParameters(url, phase);
  if (!cookie || !values) return amazonFailure(phase, 400, "INVALID_REQUEST");
  headers.set("Cookie", `${flowCookie}=${cookie}`);
  headers.set("Content-Type", "application/json");
  const response = await amazonUpstream(`store-amazon-authorization/${phase}`, headers, fetchImpl, request.signal, JSON.stringify(values));
  if (!response?.ok) return amazonFailure(phase, 503, "STORE_AUTHORIZATION_UNAVAILABLE");
  const result = await readAmazonResponse(response, request.signal);
  if (!result) return amazonFailure(phase, 502, "STORE_AUTHORIZATION_UNAVAILABLE");
  if (phase === "login") {
    const provider = readAmazonRedirect(result.authorizationUrl, phase);
    return provider ? redirectFromCallback(provider) : amazonFailure(phase, 502, "STORE_AUTHORIZATION_UNAVAILABLE");
  }
  const storeId = typeof result.storeId === "string" ? result.storeId : "";
  return isAcquisitionUUID(storeId) ? redirectFromCallback(`/workbench/stores/${storeId}`) :
    amazonFailure(phase, 502, "STORE_AUTHORIZATION_UNAVAILABLE");
}

async function beginAmazonAuthorization(
  request: Request, url: URL, actor: string, organization: string, headers: Headers, fetchImpl: typeof fetch,
) {
  if (request.headers.get("origin") !== resolvePublicAppOrigin() || request.headers.get("x-expected-user-id") !== actor) {
    return amazonFailure("begin", 403, "PERMISSION_DENIED");
  }
  if (request.headers.get("x-expected-organization-id") !== organization) {
    return amazonFailure("begin", 409, "ORGANIZATION_CONTEXT_CHANGED");
  }
  const id = request.headers.get("x-store-id") ?? "";
  const version = request.headers.get("if-match") ?? "";
  if (url.search || request.body !== null || !isAcquisitionUUID(id) || !/^"[1-9][0-9]{0,18}"$/.test(version)) {
    return amazonFailure("begin", 400, "INVALID_REQUEST");
  }
  headers.set("If-Match", version);
  const response = await amazonUpstream(`stores/${id}/amazon-authorization/begin`, headers, fetchImpl, request.signal);
  if (!response?.ok) {
    const status = response && [400, 401, 403, 404, 409].includes(response.status) ? response.status : 503;
    return amazonFailure("begin", status, "STORE_AUTHORIZATION_UNAVAILABLE");
  }
  const result = await readAmazonResponse(response, request.signal);
  const target = readAmazonRedirect(result?.authorizationUrl, "begin");
  const cookie = readUpstreamFlowCookie(response.headers);
  if (!target || !cookie) return amazonFailure("begin", 502, "STORE_AUTHORIZATION_UNAVAILABLE");
  const outgoing = NextResponse.json({ authorizationUrl: target }, { headers: privacyHeaders });
  outgoing.cookies.set(flowCookie, cookie.value, {
    path: AMAZON_AUTHORIZATION_PATH, maxAge: cookie.maxAge, httpOnly: true, secure: true, sameSite: "lax",
  });
  return outgoing;
}

function readCallbackParameters(url: URL, phase: "login" | "redirect") {
  if (!url.search || url.search.length > 12_288) return null;
  const mapping: Readonly<Record<string, string>> = callbackParameters[phase];
  const result: Record<string, string> = {};
  for (const [key, value] of url.searchParams) {
    if (!Object.hasOwn(mapping, key) || Object.hasOwn(result, mapping[key]) || !/^[\x21-\x7e]+$/.test(value)) return null;
    result[mapping[key]] = value;
  }
  const required = Object.entries(mapping).filter(([key]) => key !== "version");
  return required.every(([, field]) => Object.hasOwn(result, field)) ? result : null;
}

function readFlowCookie(headers: Headers) {
  const parts = (headers.get("cookie") ?? "").split(";").map((item) => item.trim())
    .filter((item) => item.startsWith(`${flowCookie}=`));
  if (parts.length !== 1) return "";
  const value = parts[0].slice(flowCookie.length + 1);
  return flowValue.test(value) ? value : "";
}

function readUpstreamFlowCookie(headers: Headers) {
  const values = headers.getSetCookie();
  if (values.length !== 1) return null;
  const [pair, ...attributes] = values[0].split(";").map((part) => part.trim());
  const value = pair.startsWith(`${flowCookie}=`) ? pair.slice(flowCookie.length + 1) : "";
  const ages = attributes.filter((attribute) => /^max-age=/i.test(attribute));
  const rawAge = ages.length === 1 ? ages[0].slice(8) : "";
  const maxAge = /^[1-9][0-9]{0,2}$/.test(rawAge) ? Number(rawAge) : 0;
  return flowValue.test(value) && maxAge > 0 && maxAge <= 540 ? { value, maxAge } : null;
}

function readAmazonRedirect(value: unknown, phase: "begin" | "login") {
  if (typeof value !== "string" || value.length > 11_000) return "";
  try {
    const url = new URL(value);
    if (url.protocol !== "https:" || url.port || url.username || url.password || url.hash) return "";
    if (phase === "begin") {
      return url.hostname === "sellercentral.amazon.com" && url.pathname === "/apps/authorize/consent" &&
        appId.test(url.searchParams.get("application_id") ?? "") ? url.href : "";
    }
    const prefix = "/apps/authorize/confirm/";
    return (url.hostname === "amazon.com" || url.hostname === "sellercentral.amazon.com") &&
      url.pathname.startsWith(prefix) && appId.test(url.pathname.slice(prefix.length)) ? url.href : "";
  } catch {
    return "";
  }
}

async function amazonUpstream(path: string, headers: Headers, fetchImpl: typeof fetch, signal: AbortSignal, body?: string) {
  try {
    return await fetchImpl(new URL(`${getWorkbenchServiceRoot()}/workbench/${path}`), {
      method: "POST", headers, body, cache: "no-store", redirect: "manual",
      signal: AbortSignal.any([signal, AbortSignal.timeout(36_000)]),
    });
  } catch {
    return null;
  }
}

async function readAmazonResponse(response: Response, signal: AbortSignal): Promise<Record<string, unknown> | null> {
  try {
    const bytes = await readBodyWithinLimit(response.body, 16_384, 3_000, signal);
    const value: unknown = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
    return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : null;
  } catch {
    return null;
  }
}

function redirectFromCallback(location: string) {
  return new Response(null, { status: 303, headers: { ...privacyHeaders, Location: location } });
}

function amazonFailure(phase: string, status: number, code: string) {
  if (phase === "login" || phase === "redirect") {
    return redirectFromCallback(`${AMAZON_AUTHORIZATION_PATH}/result?reason=${status === 401 ? "signin" : "restart"}`);
  }
  const response = workbenchProtocolError(status, code, "Amazon store authorization unavailable");
  response.headers.set("Cache-Control", privacyHeaders["Cache-Control"]);
  response.headers.set("Referrer-Policy", privacyHeaders["Referrer-Policy"]);
  return response;
}
