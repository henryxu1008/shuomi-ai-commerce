import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { proxyMarketResearch } from "./market-research-proxy";
import { WORKBENCH_COOKIE_NAME } from "./workbench-proxy";

const project = "11111111-1111-4111-8111-111111111111";
const approval = "22222222-2222-4222-8222-222222222222";
const other = "33333333-3333-4333-8333-333333333333";
const base = `https://app.example/api/workbench/research/projects/${project}/shein-us-trial`;
const upstream = `http://localhost:8080/api/v1/workbench/research/projects/${project}/shein-us-trials/${approval}`;
const fetcher = vi.fn();
const pending = { approvalId: approval, state: "UNKNOWN", requestId: "request-1", credits: null, reservedCredits: 25 };

beforeEach(() => {
  fetcher.mockReset();
  vi.stubGlobal("fetch", fetcher);
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "https://app.example");
  vi.stubEnv("LISTINGKIT_SERVICE_API_BASE", "http://localhost:8080/api/v1");
  vi.stubEnv("LISTINGKIT_SHEIN_US_TRIAL_TARGET", `${project}/${approval}`);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

function request(suffix = "", init: RequestInit = {}) {
  const headers = new Headers({
    Cookie: `${WORKBENCH_COOKIE_NAME}=org-a`,
    "X-Expected-Organization-ID": "org-a",
    "X-Expected-User-ID": "user-a",
    Origin: "https://app.example",
    "Sec-Fetch-Site": "same-origin",
    ...(suffix.startsWith("/") ? { "X-Expected-SHEIN-Trial-Approval-ID": approval } : {}),
  });
  for (const [name, value] of new Headers(init.headers)) headers.set(name, value);
  return new Request(base + suffix, { method: suffix.startsWith("/") ? "POST" : "GET", ...init, headers });
}

const proxy = (r: Request, token = "synthetic-session-token") => proxyMarketResearch(r, token, "user-a");

it("allows an authorized empty original but rejects a missing backend route", async () => {
  fetcher.mockResolvedValueOnce(Response.json({ code: "NOT_FOUND" }, { status: 404 }));
  const ready = await proxy(request());
  expect(ready.status).toBe(200);
  expect(ready.headers.get("cache-control")).toBe("private, no-store");
  expect(await ready.json()).toEqual({ approvalId: approval, state: "NOT_STARTED" });
  expect(fetcher.mock.calls[0][0]).toBe(upstream);

  fetcher.mockResolvedValueOnce(new Response("404 page not found", { status: 404 }));
  expect((await proxy(request())).status).toBe(503);
  fetcher.mockResolvedValueOnce(Response.json({ code: "FORBIDDEN" }, { status: 403 }));
  expect(await (await proxy(request())).json()).toEqual({ code: "FORBIDDEN" });
});

it("denies unauthenticated, cross-context, cross-origin and tampered requests before dispatch", async () => {
  expect((await proxy(request(), "")).status).toBe(401);
  expect((await proxy(request("", { headers: { "X-Expected-User-ID": "user-b" } }))).status).toBe(409);
  expect((await proxy(request("", { headers: { "X-Expected-Organization-ID": "org-b" } }))).status).toBe(409);
  expect((await proxy(request("/submit", { headers: { Origin: "https://outside.example" } }))).status).toBe(403);
  expect((await proxy(request("/submit", { headers: { "X-Expected-SHEIN-Trial-Approval-ID": other } }))).status).toBe(409);
  expect((await proxy(request("/submit", { body: '{"source":"other"}' }))).status).toBe(400);
  expect((await proxy(request("/submit?next=other"))).status).toBe(400);
  vi.stubEnv("LISTINGKIT_SHEIN_US_TRIAL_TARGET", `${other}/${approval}`);
  expect((await proxy(request())).status).toBe(404);
  vi.stubEnv("LISTINGKIT_SHEIN_US_TRIAL_TARGET", "");
  expect((await proxy(request())).status).toBe(404);
  expect(fetcher).not.toHaveBeenCalled();
});

it("passes only the fixed actions and preserves an unknown receipt until the original is reconciled", async () => {
  const confirmed = { ...pending, state: "CONFIRMED", credits: 0, reservedCredits: 0 };
  fetcher.mockResolvedValueOnce(Response.json(pending, { status: 409 }));
  fetcher.mockResolvedValueOnce(Response.json(confirmed));
  // The server may represent an empty POST as a zero-byte stream.
  const result = await proxy(request("/submit", { body: "", headers: { Authorization: "untrusted", "X-Requested-Organization-ID": "untrusted" } }));
  expect(result.status).toBe(409);
  expect(await result.json()).toEqual(pending);
  const [destination, options] = fetcher.mock.calls[0];
  expect(destination).toBe(upstream + "/submit");
  expect(options.body).toBeUndefined();
  const headers = new Headers(options.headers);
  expect(headers.get("Authorization")).toBe("Bearer synthetic-session-token");
  expect(headers.get("X-Requested-Organization-ID")).toBe("org-a");
  expect(headers.has("X-Expected-SHEIN-Trial-Approval-ID")).toBe(false);
  expect(await (await proxy(request("/reconcile"))).json()).toEqual(confirmed);
  expect(fetcher.mock.calls[1][0]).toBe(upstream + "/reconcile");
});

it("keeps a failed submit ambiguous and rejects receipts for another approval", async () => {
  fetcher.mockRejectedValueOnce(new Error("network error"));
  fetcher.mockResolvedValueOnce(Response.json({ ...pending, approvalId: other }));
  expect(await (await proxy(request("/submit"))).json()).toEqual({ code: "OUTCOME_UNKNOWN" });
  expect((await proxy(request())).status).toBe(503);
  expect(fetcher).toHaveBeenCalledTimes(2);
});
