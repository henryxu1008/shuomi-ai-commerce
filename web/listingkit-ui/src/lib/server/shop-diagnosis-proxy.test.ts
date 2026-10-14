import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { proxyPOD } from "./pod-proxy";
const id = "11111111-1111-4111-8111-111111111111",
  other = "22222222-2222-4222-8222-222222222222";
function request(path: string, body: unknown) {
  return new Request(
    "https://console.example/api/workbench/shop-diagnosis/" + path,
    {
      method: "POST",
      headers: {
        "X-Expected-User-ID": "user-a",
        "X-Expected-Organization-ID": "org-a",
        cookie: "shuomi_effective_organization=org-a",
        Origin: "https://console.example",
        "Content-Type": "application/json",
        "Idempotency-Key": id,
      },
      body: JSON.stringify(body),
    },
  );
}
beforeEach(() => {
  vi.stubEnv("LISTINGKIT_SERVICE_API_BASE", "https://api.example/api/v1");
  vi.stubEnv("LISTINGKIT_SDS_POD_ENABLED", "false");
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "https://console.example");
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});
it("reads comparison facts without borrowing the SDS feature or forwarding arbitrary fields", async () => {
  const facts = {
    ref: {
      StoreID: other,
      StartDate: "2026-10-01",
      EndDate: "2026-10-07",
      EvidenceDigest: "a".repeat(64),
    },
    facts: [],
    missing: ["financial.amounts"],
  };
  const fetcher = vi.fn<
    (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>
  >(async () => Response.json(facts));
  vi.stubGlobal("fetch", fetcher);
  const body = {
    contextId: id,
    storeId: other,
    startDate: "2026-10-01",
    endDate: "2026-10-07",
  };
  const response = await proxyPOD(
    request("inspect", body),
    "private-token",
    "user-a",
  );
  expect(response.status).toBe(200);
  expect(fetcher).toHaveBeenCalledOnce();
  expect(fetcher.mock.calls[0][0]).toBe(
    "https://api.example/api/v1/workbench/shop-diagnosis/inspect",
  );
  const unsafe = await proxyPOD(
    request("inspect", { ...body, tenantId: "other" }),
    "private-token",
    "user-a",
  );
  expect(unsafe.status).toBe(400);
  expect(fetcher).toHaveBeenCalledOnce();
});
it("retains the original key when an analysis response is uncertain", async () => {
  const fetcher = vi.fn(async () => {
    throw new Error("lost response");
  });
  vi.stubGlobal("fetch", fetcher);
  const response = await proxyPOD(
    request("contexts/" + id + "/runs", {
      configurationSnapshotRef: {
        Kind: "agent-configuration-v1",
        ID: id,
        Digest: "a".repeat(64),
      },
    }),
    "private-token",
    "user-a",
  );
  expect(await response.json()).toEqual({ code: "OUTCOME_UNKNOWN" });
  expect(fetcher).toHaveBeenCalledOnce();
});
