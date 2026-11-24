import { NextRequest, NextResponse } from "next/server";
import { afterEach, expect, it, vi } from "vitest";

const auth = vi.hoisted(() => ({
  run: vi.fn(),
  user: "",
}));

vi.mock("@/auth", () => ({
  serverAuth: (handler: (request: NextRequest) => unknown) => (request: NextRequest) => {
    auth.run();
    return handler(request);
  },
}));
vi.mock("@/lib/server/zitadel-auth", () => ({
  readZitadelIdentityFromSession: () => auth.user ? { userId: auth.user } : null,
  resolvePublicAppOrigin: () => "https://workbench.example",
}));
vi.mock("@/lib/server/zitadel-server-token", () => ({ readZitadelServerAccessToken: () => auth.user ? "local-session" : "" }));
vi.mock("@/lib/server/account-proxy", () => ({ serviceOrigin: () => "http://current-application:8080" }));

import { GET, POST } from "./route";

const shop = "fixture.myshopify.com";
const workbench = "https://workbench.example";
const store = "11111111-1111-4111-8111-111111111111";
const cookieName = "__Host-shuomi-shopify-started-install";

function request(action: string, params: ConstructorParameters<typeof NextRequest>[1] = {}) {
  return new NextRequest(workbench + "/api/shopify/" + action, params);
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
  auth.run.mockClear();
  auth.user = "";
});

it("starts Shopify OAuth before local sign-in and confirms through the existing authenticated Store route", async () => {
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", workbench);
  const fetcher = vi.fn<typeof fetch>()
    .mockResolvedValueOnce(Response.json({ authorizationUrl: "https://" + shop + "/admin/oauth/authorize?state=fixture" }))
    .mockResolvedValueOnce(Response.json({ domain: shop }))
    .mockResolvedValueOnce(Response.json({ domain: shop, storeVersion: 4, confirmed: false }))
    .mockResolvedValueOnce(Response.json({ domain: shop }));
  vi.stubGlobal("fetch", fetcher);

  const launch = await GET(request("launch?shop=" + shop + "&hmac=launch"));
  expect(launch.status).toBe(303);
  expect(launch.headers.get("location")).toContain("https://" + shop + "/admin/oauth/authorize");
  expect(launch.headers.get("referrer-policy")).toBe("no-referrer");
  if (!(launch instanceof NextResponse)) throw new Error("Expected secure initiation cookie");
  const launchCookie = launch.cookies.get(cookieName)!;
  expect(launchCookie).toMatchObject({ secure: true, httpOnly: true, sameSite: "lax", path: "/" });
  const initial = JSON.parse(Buffer.from(launchCookie.value, "base64url").toString("utf8"));
  expect(initial).toEqual({ attempt: expect.any(String), browser: expect.any(String), state: expect.any(String) });

  const callback = await GET(request("callback?shop=" + shop + "&code=fixture-code&hmac=callback",
    { headers: { cookie: cookieName + "=" + launchCookie.value } }));
  expect(callback.status).toBe(303);
  expect(callback.headers.get("location")).toBe("/workbench/stores?platform=shopify&shopify=pair");
  expect(auth.run).not.toHaveBeenCalled();
  for (const [, init] of fetcher.mock.calls) {
    expect(new Headers(init?.headers).get("authorization")).toBeNull();
  }
  if (!(callback instanceof NextResponse)) throw new Error("Expected secure callback cookie");
  const cookie = callback.cookies.get(cookieName)!.value;
  expect(JSON.parse(Buffer.from(cookie, "base64url").toString("utf8")))
    .toEqual({ attempt: initial.attempt, browser: initial.browser });

  auth.user = "actor-a";
  const scopedCookie = cookieName + "=" + cookie + "; shuomi_effective_organization=org-a";
  const pending = await GET(request("pending?store=" + store, { headers: { cookie: scopedCookie } }));
  expect(await pending.json()).toEqual({ domain: shop, attempt: initial.attempt, version: 4 });
  const headers = { cookie: scopedCookie, origin: workbench, "content-type": "application/json",
    "x-expected-organization-id": "org-a", "x-expected-user-id": "actor-a" };
  const stale = await POST(request("confirm-install", { method: "POST", headers,
    body: JSON.stringify({ store, version: 4, attempt: store }) }));
  expect(stale.status).toBe(409);
  expect(fetcher).toHaveBeenCalledTimes(3);

  const confirmed = await POST(request("confirm-install", { method: "POST", headers,
    body: JSON.stringify({ store, version: 4, attempt: initial.attempt }) }));
  expect(confirmed.status).toBe(200);
  if (!(confirmed instanceof NextResponse)) throw new Error("Expected cleared install cookie");
  expect(confirmed.cookies.get(cookieName)?.maxAge).toBe(0);
  const [url, init] = fetcher.mock.calls[3]!;
  expect(String(url)).toBe("http://current-application:8080/api/v1/workbench/stores/" + store + "/shopify-pairing");
  expect(new Headers(init?.headers).get("if-match")).toBe('"4"');
  expect(new Headers(init?.headers).get("x-requested-organization-id")).toBe("org-a");
  expect(new Headers(init?.headers).get("idempotency-key")).toBe(initial.attempt);
  expect(JSON.parse(init?.body as string)).toEqual({ action: "confirm_install", browser: initial.browser });
});

it("does not convert a failed preselected pairing into a Shopify-first authorization", async () => {
  const fetcher = vi.fn<typeof fetch>();
  vi.stubGlobal("fetch", fetcher);
  const withLegacyCookie = await GET(request("launch?shop=" + shop + "&hmac=launch",
    { headers: { cookie: "__Host-shuomi-shopify-pairing=invalid" } }));
  expect(withLegacyCookie.status).toBe(401);
  expect(auth.run).toHaveBeenCalledOnce();
  expect(fetcher).not.toHaveBeenCalled();
});
