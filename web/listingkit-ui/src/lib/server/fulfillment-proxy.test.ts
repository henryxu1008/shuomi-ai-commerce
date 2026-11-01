import { afterEach, expect, it, vi } from "vitest";
vi.mock("./account-proxy", () => ({
  serviceOrigin: () => "http://current-api.test",
}));
vi.mock("./same-origin-write", () => ({
  hasTrustedSameOriginWrite: () => true,
}));
vi.mock("./workbench-proxy", () => ({ WORKBENCH_COOKIE_NAME: "current-org" }));
import { proxyFulfillment } from "./fulfillment-proxy";
const input = {
  store: "store",
  kind: "platform_supply",
  source_id: "source",
  operation_id: "a182d2e0-6f91-4e71-8b7f-5df25b75fb11",
  action: "create_purchase",
  payload: { demand: "native" },
};
function request(body: unknown = input) {
  return new Request("https://app.test/api/workbench/fulfillment/commands", {
    method: "POST",
    headers: {
      "content-type": "application/json",
      cookie: "current-org=org",
      "X-Expected-User-ID": "user",
      "X-Expected-Organization-ID": "org",
    },
    body: JSON.stringify(body),
  });
}
afterEach(() => vi.unstubAllGlobals());
it("does not report another operation receipt as success", async () => {
  vi.stubGlobal(
    "fetch",
    vi
      .fn()
      .mockResolvedValue(
        Response.json({
          operation: "ca0942bc-aef7-4ae1-a5be-e4bacf434ede",
          stage: "SUCCEEDED",
          target: "native-po",
          revision: "revision",
          result: {},
        }),
      ),
  );
  const response = await proxyFulfillment(
    request(),
    "private-session-token",
    "user",
  );
  expect(response.status).toBe(409);
  expect(await response.json()).toEqual({ code: "FULFILLMENT_UNKNOWN" });
});
it("keeps an invalid upstream result unknown after the command may have executed", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(Response.json({ stage: "SUCCEEDED" })),
  );
  const response = await proxyFulfillment(
    request(),
    "private-session-token",
    "user",
  );
  expect(response.status).toBe(409);
  expect(await response.json()).toEqual({ code: "FULFILLMENT_UNKNOWN" });
});
it("never accepts browser supplied ERP origin or organization", async () => {
  const transport = vi.fn();
  vi.stubGlobal("fetch", transport);
  const response = await proxyFulfillment(
    request({ ...input, organization: "other", erp_url: "https://other.test" }),
    "private-session-token",
    "user",
  );
  expect(response.status).toBe(400);
  expect(transport).not.toHaveBeenCalled();
});
