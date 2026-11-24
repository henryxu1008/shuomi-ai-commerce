import { afterEach, expect, it, vi } from "vitest";
import { buildWorkbenchBrowserResponse, buildWorkbenchUpstreamRequest, WORKBENCH_COOKIE_NAME } from "./workbench-proxy";

const id = "11111111-1111-4111-8111-111111111111";
const other = "22222222-2222-4222-8222-222222222222";
const path = ["supply-preparations", "amazon-listings", "confirm"];
const body = { recordId: id, wireHash: "a".repeat(64) };
const request = (actor: string) => new Request("http://localhost/api/workbench/" + path.join("/"), {
  method: "POST", headers: {
    cookie: WORKBENCH_COOKIE_NAME + "=org", "X-Expected-Organization-ID": "org", "X-Expected-User-ID": actor,
    Origin: "http://localhost", "Content-Type": "application/json", "Idempotency-Key": id,
  }, body: JSON.stringify(body),
});
afterEach(() => vi.unstubAllEnvs());

it("keeps the endpoint closed by default and binds an isolated confirmation to the original actor and record", async () => {
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://localhost");
  vi.stubEnv("LISTINGKIT_ISOLATED_AMAZON_LISTING_ENABLED", "");
  expect(await buildWorkbenchUpstreamRequest(request("actor"), path, "server-token", "actor")).toMatchObject({ status:404 });
  vi.stubEnv("LISTINGKIT_ISOLATED_AMAZON_LISTING_ENABLED", "true");
  expect(await buildWorkbenchUpstreamRequest(request("foreign"), path, "server-token", "actor")).toMatchObject({ status:409 });
  const upstream = await buildWorkbenchUpstreamRequest(request("actor"), path, "server-token", "actor");
  expect(upstream).not.toBeInstanceOf(Response);
  if (upstream instanceof Response) return;
  expect(JSON.parse(upstream.expectedStoreId!)).toEqual(body);
  const result = { sourceId: id, storeId: other, recordId: id, attemptId: id, sku: "shuomi-" + "a".repeat(32), submission: "unknown" };
  expect(await buildWorkbenchBrowserResponse(Response.json(result), "supply-amazon-confirm", upstream.expectedStoreId)).toMatchObject({ status:200 });
  expect(await buildWorkbenchBrowserResponse(Response.json({ ...result, recordId: other }), "supply-amazon-confirm", upstream.expectedStoreId)).toMatchObject({ status:502 });
});
