import { afterEach, expect, it, vi } from "vitest";
import { proxyNotifications } from "./notification-proxy";
import { WORKBENCH_COOKIE_NAME } from "./workbench-proxy";
const key = "4841d296-ef14-4c16-8d25-a7667e534feb";
const headers = { "X-Expected-User-ID": "actor", "X-Expected-Organization-ID": "org", cookie: WORKBENCH_COOKIE_NAME + "=org", Origin: "http://localhost:3000" };
afterEach(() => { vi.unstubAllEnvs(); vi.unstubAllGlobals(); });
function configure() { vi.stubEnv("LISTINGKIT_SERVICE_API_BASE", "http://127.0.0.1:9000/api/v1"); vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://localhost:3000"); }
it("rejects stale identities, duplicate query/cookies and cross-site receipt writes before dispatch", async () => {
  configure(); const fetch = vi.fn(); vi.stubGlobal("fetch", fetch);
  for (const [path, init, status] of [
    ["official?filter=unread&filter=all", {}, 400],
    ["official?org=other", {}, 400],
    ["official", { headers: { ...headers, "X-Expected-User-ID": "other" } }, 409],
    ["business", { headers: { ...headers, cookie: headers.cookie + "; " + headers.cookie } }, 409],
    ["business", { headers: { ...headers, cookie: WORKBENCH_COOKIE_NAME + "=other" } }, 409],
    ["official/read", { method: "POST", headers: { ...headers, Origin: "http://foreign.test" }, body: "{}" }, 403],
  ] as const) expect((await proxyNotifications(new Request("http://localhost:3000/api/notifications/" + path, { headers, ...init }), "server-token", "actor")).status).toBe(status);
  expect(fetch).not.toHaveBeenCalled();
});
it("official/personal are independent of organization; upstream credentials remain server-only", async () => {
  configure(); const fetch = vi.fn().mockImplementation(() => Promise.resolve(Response.json({ schemaVersion: "notification-center-v1", items: [], coverage: [], exact: true, count: 0, unread: 0, pending: 0, next: "" }))); vi.stubGlobal("fetch", fetch);
  for (const channel of ["official", "personal"]) {
    const response = await proxyNotifications(new Request("http://localhost:3000/api/notifications/" + channel, { headers: { "X-Expected-User-ID": "actor", Authorization: "Bearer browser-token" } }), "server-token", "actor");
    expect(response.status).toBe(200);
    const forwarded = new Headers(fetch.mock.calls.at(-1)![1].headers);
    expect(forwarded.get("Authorization")).toBe("Bearer server-token");
    expect(forwarded.has("X-Requested-Organization-ID")).toBe(false);
    expect(forwarded.has("cookie")).toBe(false);
  }
});
it("rejects ambiguous JSON, and retains one key on an uncertain receipt commit", async () => {
  configure(); const fetch = vi.fn().mockRejectedValue(new Error("lost response")); vi.stubGlobal("fetch", fetch);
  for (const body of ['{"id":"a","id":"b"}', '{"id":"a","subject":"other"}']) {
    expect((await proxyNotifications(new Request("http://localhost:3000/api/notifications/official/read", { method: "POST", headers: { ...headers, "Content-Type": "application/json", "Idempotency-Key": key }, body }), "server-token", "actor")).status).toBe(400);
  }
  expect(fetch).not.toHaveBeenCalled();
  const response = await proxyNotifications(new Request("http://localhost:3000/api/notifications/official/snapshot", { method: "POST", headers: { ...headers, "Content-Type": "application/json", "Idempotency-Key": key }, body: "{}" }), "server-token", "actor");
  expect((await response.json()).code).toBe("OUTCOME_UNKNOWN");
  expect(fetch).toHaveBeenCalledTimes(1);
  expect(new Headers(fetch.mock.calls[0][1].headers).get("Idempotency-Key")).toBe(key);
});

it("routes platform management to the existing publication commands and platform-only history", async () => {
  configure();
  const id = "4f5e1184-83c7-45b0-9dd9-c61242c9fb25";
  const content = { category: "SYSTEM", title: "维护", summary: "窗口安排", paragraphs: ["公告内容"], target: { kind: "none", id: "" } };
  const endpoint = "http://localhost:3000/api/notifications/platform/official";
  const fetch = vi.fn(async (url: string, init: RequestInit) => {
    if (init.method === "POST") return Response.json({ key, resultId: id, operation: url.endsWith("/withdraw") ? "withdraw" : "publish", committedAt: "2026-11-07T00:00:00Z" });
    return Response.json(url.endsWith(id) ? content : { items: [], next: "" });
  });
  vi.stubGlobal("fetch", fetch);
  const requestHeaders = { ...headers, "Content-Type": "application/json", "Idempotency-Key": key, "X-User-Roles": "platform_admin" };
  for (const [suffix, init] of [
    ["", {}],
    ["/" + id, {}],
    ["", { method: "POST", body: JSON.stringify(content) }],
    ["/" + id + "/withdraw", { method: "POST", body: JSON.stringify({ revision: 1 }) }],
  ] as const) {
    const response = await proxyNotifications(new Request(endpoint + suffix, { headers: requestHeaders, ...init }), "server-token", "actor");
    expect(response.status).toBe(200);
    const [upstream, request] = fetch.mock.calls.at(-1)!;
    expect(upstream).toBe("http://127.0.0.1:9000/api/v1/platform/notifications/official" + suffix);
    const forwarded = new Headers(request.headers);
    expect(forwarded.get("Authorization")).toBe("Bearer server-token");
    expect(forwarded.get("X-User-Roles")).toBeNull();
    expect(forwarded.get("X-Requested-Organization-ID")).toBeNull();
    if ("method" in init) expect(forwarded.get("Idempotency-Key")).toBe(key);
  }
  const count = fetch.mock.calls.length;
  const crossSite = await proxyNotifications(new Request(endpoint, { method: "POST", headers: { ...requestHeaders, Origin: "http://foreign.test" }, body: JSON.stringify(content) }), "server-token", "actor");
  expect(crossSite.status).toBe(403);
  const injectedAudience = await proxyNotifications(new Request(endpoint, { method: "POST", headers: requestHeaders, body: JSON.stringify({ ...content, audience: "other" }) }), "server-token", "actor");
  expect(injectedAudience.status).toBe(400);
  expect(fetch).toHaveBeenCalledTimes(count);
  fetch.mockImplementationOnce(async () => Response.json({ error: "listingkit_permission_denied" }, { status: 403 }));
  const forbidden = await proxyNotifications(new Request(endpoint, { headers: requestHeaders }), "server-token", "actor");
  expect(forbidden.status).toBe(403);
  expect(await forbidden.json()).toEqual({ code: "FORBIDDEN" });
});
