import { beforeEach, expect, it, vi } from "vitest";
import { buildWorkbenchBrowserResponse, buildWorkbenchUpstreamRequest } from "./workbench-proxy";

const store = "550e8400-e29b-41d4-a716-446655440000";
const second = "550e8400-e29b-41d4-a716-446655440001";
const query = `?startDate=2026-10-01&endDate=2026-10-07&storeId=${store}`;
const scope = { cookie: "shuomi_effective_organization=org-a", "X-Expected-Organization-ID": "org-a", "X-Expected-User-ID": "user-a" };
const capturedAt = "2026-10-08T00:00:00Z";

beforeEach(() => vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://localhost"));

it.each([
  ["supply-preparations", "supply-cockpit-metrics", { pending: 3, review: 1 }],
  ["tasks", "ai-task-cockpit-metrics", { RUNNING: 1, WAITING_CONFIRMATION: 0, COMPLETED: 0, ERROR: 2, PAUSED: 0, UNKNOWN: 0 }],
] as const)("forwards only the scoped %s aggregate GET", async (owner, contract, counts) => {
  const path = [owner, "cockpit-metrics"];
  const url = `http://localhost/api/workbench/${path.join("/")}`;
  const request = new Request(url + query + `&storeId=${second}`, { headers: scope });
  const mapped = await buildWorkbenchUpstreamRequest(request, path, "server-token", "user-a");
  expect(mapped).not.toBeInstanceOf(Response);
  if (mapped instanceof Response) return;
  expect(mapped.url).toContain(query + `&storeId=${second}`);
  expect(mapped.responseContract).toBe(contract);
  expect(mapped.sourceMutation).toBe(false);
  for (const suffix of ["", query + `&storeId=${store}`, query + "&actor=another", query + "&startDate=2026-10-01"]) {
    const invalid = await buildWorkbenchUpstreamRequest(new Request(url + suffix, { headers: scope }), path, "server-token", "user-a");
    expect(invalid).toBeInstanceOf(Response);
    if (invalid instanceof Response) expect(invalid.status).toBe(400);
  }
  for (const headers of [{ ...scope, "X-Expected-User-ID": "user-b" }, { ...scope, "X-Expected-Organization-ID": "org-b" }]) {
    const invalid = await buildWorkbenchUpstreamRequest(new Request(url + query, { headers }), path, "server-token", "user-a");
    expect(invalid).toBeInstanceOf(Response);
    if (invalid instanceof Response) expect(invalid.status).toBe(409);
  }
  const complete = { status: "complete", capturedAt, counts };
  expect((await buildWorkbenchBrowserResponse(Response.json(complete), contract)).status).toBe(200);
  expect((await buildWorkbenchBrowserResponse(Response.json({ status: "limited", capturedAt }), contract)).status).toBe(200);
  expect((await buildWorkbenchBrowserResponse(Response.json({ ...complete, rawTask: "private" }), contract)).status).toBe(502);
  expect((await buildWorkbenchBrowserResponse(Response.json({ ...complete, status: "limited" }), contract)).status).toBe(502);
});

it("admits one store and only bounded counts for the Walmart Feed metric", async () => {
  const path = ["supply-preparations", "walmart-feed-metrics"];
  const url = "http://localhost/api/workbench/" + path.join("/");
  const mapped = await buildWorkbenchUpstreamRequest(new Request(url + query, { headers: scope }), path, "server-token", "user-a");
  expect(mapped).not.toBeInstanceOf(Response);
  if (mapped instanceof Response) return;
  expect(mapped.responseContract).toBe("supply-walmart-feed-metrics");
  expect(mapped.sourceMutation).toBe(false);
  const multiple = await buildWorkbenchUpstreamRequest(new Request(url + query + "&storeId=" + second, { headers: scope }), path, "server-token", "user-a");
  expect(multiple).toBeInstanceOf(Response);
  if (multiple instanceof Response) expect(multiple.status).toBe(400);
  const complete = { status: "complete", capturedAt, counts: { accepted: 0, rejected: 0, unknown: 1 } };
  expect((await buildWorkbenchBrowserResponse(Response.json(complete), mapped.responseContract)).status).toBe(200);
  expect((await buildWorkbenchBrowserResponse(Response.json({ ...complete, counts: { accepted: 250, rejected: 1, unknown: 0 } }), mapped.responseContract)).status).toBe(502);
  expect((await buildWorkbenchBrowserResponse(Response.json({ ...complete, attemptId: "private" }), mapped.responseContract)).status).toBe(502);
});
it("allows a bounded multi-store overview while keeping identity guards and incomplete totals", async () => {
 const path=["supply-preparations","submission-overview-metrics"];
 const url="http://localhost/api/workbench/"+path.join("/");
 const mapped=await buildWorkbenchUpstreamRequest(new Request(url+query+"&storeId="+second,{headers:scope}),path,"server-token","user-a");
 expect(mapped).not.toBeInstanceOf(Response);
 if(mapped instanceof Response)return;
 expect(mapped.responseContract).toBe("supply-submission-overview-metrics");
 expect(mapped.sourceMutation).toBe(false);
 for(const suffix of [query+"&actor=other",query+"&storeId="+store]){
  const invalid=await buildWorkbenchUpstreamRequest(new Request(url+suffix,{headers:scope}),path,"server-token","user-a");
  expect(invalid).toBeInstanceOf(Response);if(invalid instanceof Response)expect(invalid.status).toBe(400);
 }
 const incomplete={status:"incomplete",capturedAt,excludedStores:0,rows:[
  {platform:"walmart",status:"empty",stores:0},{platform:"tiktokshop",status:"empty",stores:0},
  {platform:"shein",status:"empty",stores:0},{platform:"temu",status:"limited",stores:2},
 ]};
 expect((await buildWorkbenchBrowserResponse(Response.json(incomplete),mapped.responseContract)).status).toBe(200);
 expect((await buildWorkbenchBrowserResponse(Response.json({...incomplete,unknownTotal:0}),mapped.responseContract)).status).toBe(502);
});
