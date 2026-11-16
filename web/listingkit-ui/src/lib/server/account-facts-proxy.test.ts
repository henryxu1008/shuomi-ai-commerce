import { afterEach, it, expect, vi } from "vitest";
import { proxyAccountFacts } from "./account-facts-proxy";
afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});
it("saves only self preferences without an enterprise and refuses duplicate fields", async () => {
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://localhost:3000");
  vi.stubEnv("LISTINGKIT_SERVICE_API_BASE", "http://127.0.0.1:9000/api/v1");
  const fetch = vi.fn((_url: RequestInfo | URL, _init?: RequestInit) =>
    Promise.resolve(
      Response.json({
        schemaVersion: "account-preferences-v1",
        userId: "self",
        country: "China",
        province: "",
        city: "杭州",
        updatedAt: "2026-09-28T00:00:00Z",
        readAt: "2026-09-28T00:00:00Z",
        source: "account_profile",
      }),
    ),
  );
  vi.stubGlobal("fetch", fetch);
  const make = (body: string, origin = "http://localhost:3000") =>
    new Request("http://localhost:3000/api/account/preferences", {
      method: "PUT",
      headers: {
        Origin: origin,
        "X-Expected-User-ID": "self",
        "Content-Type": "application/json",
      },
      body,
    });
  expect(
    (
      await proxyAccountFacts(
        make('{"catalogVersion":"gn-0123456789abcdef","country":"CN","province":"1","city":"3"}'),
        "token",
        "self",
      )
    ).status,
  ).toBe(200);
  expect(
    new Headers(fetch.mock.calls[0][1]?.headers).has(
      "X-Requested-Organization-ID",
    ),
  ).toBe(false);
  for (const body of [
    '{"catalogVersion":"gn-0123456789abcdef","country":"CN","country":"SG","province":"","city":""}',
    '{"country":"China","province":"","city":"杭州"}',
    '{"country":"China","province":"","city":"","userId":"foreign"}',
  ])
    expect((await proxyAccountFacts(make(body), "token", "self")).status).toBe(
      400,
    );
  expect(
    (
      await proxyAccountFacts(
        make("{}", "http://foreign.test"),
        "token",
        "self",
      )
    ).status,
  ).toBe(403);
  expect(fetch).toHaveBeenCalledTimes(1);
});

it("forwards only an authenticated country catalog GET and verifies its country", async () => {
  vi.stubEnv("LISTINGKIT_SERVICE_API_BASE", "http://127.0.0.1:9000/api/v1");
  const fetch = vi.fn().mockResolvedValue(Response.json({
    version: "gn-0123456789abcdef",
    country: { id: "SG", name: "Singapore", provinces: [], cities: [{ id: "1880252", name: "Singapore" }] },
  }));
  vi.stubGlobal("fetch", fetch);
  const request = (query: string, identity = "self") => new Request("http://localhost:3000/api/account/region-catalog" + query,
    { headers: { "X-Expected-User-ID": identity } });
  expect((await proxyAccountFacts(request("?country=SG", "other"), "token", "self")).status).toBe(409);
  expect((await proxyAccountFacts(request("?country=SG&country=CN"), "token", "self")).status).toBe(400);
  expect((await proxyAccountFacts(request("?country=SG&view=shipping"), "token", "self")).status).toBe(400);
  expect((await proxyAccountFacts(request("?country=CN&view=shipping&view=shipping"), "token", "self")).status).toBe(400);
  expect((await proxyAccountFacts(request("?country=SG"), "token", "self")).status).toBe(200);
  expect(fetch).toHaveBeenCalledWith("http://127.0.0.1:9000/api/v1/account/region-catalog?country=SG", expect.objectContaining({ method: "GET" }));
  fetch.mockResolvedValueOnce(Response.json({
    version: "gn-0123456789abcdef",
    country: { id: "CN", name: "China", provinces: [{ id: "330000", name: "浙江省", cities: [{ id: "330100", name: "杭州市" }] }], cities: [] },
  }));
  const shipping = await proxyAccountFacts(request("?view=shipping&country=CN"), "token", "self");
  expect(shipping.status).toBe(200);
  expect((await shipping.json()).country.provinces[0].name).toBe("浙江省");
  expect(fetch).toHaveBeenLastCalledWith("http://127.0.0.1:9000/api/v1/account/region-catalog?country=CN&view=shipping", expect.objectContaining({ method: "GET" }));
});
