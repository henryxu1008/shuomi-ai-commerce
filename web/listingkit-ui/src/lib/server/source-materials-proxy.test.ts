import { beforeEach, expect, it, vi } from "vitest";
import { buildWorkbenchUpstreamRequest, buildWorkbenchBrowserResponse } from "./workbench-proxy";

const id = "550e8400-e29b-41d4-a716-446655440000";
const other = "550e8400-e29b-41d4-a716-446655440001";
const path = ["supply-preparations", "sources", id, "product-inputs"];
const headers = { cookie: "shuomi_effective_organization=org-a", "X-Expected-Organization-ID": "org-a", "X-Expected-User-ID": "actor-a" };
const url = `http://localhost/api/workbench/${path.join("/")}`;
export const inputsFixture = () => ({ sourceId: id, inputs: {
  scope: "product.inputs", status: "blocked", product_key: "product-a", product_version: "1", publication_id: id, target_platform: "product",
  input_rule_version: "listing.readiness.ProductInputs/tool-v1.0.0", reasons: ["approved_assets_not_ready"], missing_facts: ["approved_assets"],
  asset_binding: { product_version: "1", status: "missing", count: 0 }, marketplace: { status: "not_evaluated", reasons: ["marketplace_rules_out_of_scope"] },
  diagnostics: { needs_review: false, review_reasons: [], warnings: [] },
} });

beforeEach(() => vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://localhost"));
it("admits an ordinary source read with no query, body or mutation authority", async () => {
  const result = await buildWorkbenchUpstreamRequest(new Request(url, { headers }), path, "synthetic-token", "actor-a");
  expect(result).not.toBeInstanceOf(Response);
  if (result instanceof Response) return;
  expect(result.sourceMutation).toBe(false);
  expect(result.expectedStoreId).toBe(id);
  for (const query of ["?targetPlatform=temu", "?version=2", "?org=other", "?after=" + id]) {
    expect(await buildWorkbenchUpstreamRequest(new Request(url + query, { headers }), path, "synthetic-token", "actor-a")).toBeInstanceOf(Response);
  }
  const body = await buildWorkbenchUpstreamRequest(new Request(url, { method: "POST", headers, body: "{}" }), path, "synthetic-token", "actor-a");
  expect(body).toBeInstanceOf(Response);
});
it("rejects wrong source, non-original target and fabricated ready responses", async () => {
  const response = (payload: unknown) => buildWorkbenchBrowserResponse(Response.json(payload), "supply-product-inputs", id);
  expect((await response(inputsFixture())).status).toBe(200);
  expect((await response({ ...inputsFixture(), sourceId: other })).status).toBe(502);
  const wrong = inputsFixture(); wrong.inputs.target_platform = "shein";
  expect((await response(wrong)).status).toBe(502);
  const ready = inputsFixture(); ready.inputs.status = "ready";
  expect((await response(ready)).status).toBe(502);
  const failed = inputsFixture(); failed.inputs.marketplace.status = "ready";
  expect((await response(failed)).status).toBe(502);
});
