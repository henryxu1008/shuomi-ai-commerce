import { describe, expect, it, vi } from "vitest";

import { proxyAmazonAuthorization } from "./amazon-authorization-proxy";
import { WORKBENCH_COOKIE_NAME } from "./workbench-proxy";

const storeId = "afafb20d-132d-44c2-bff8-03eb949c7410";
const flow = "a".repeat(42) + "E";
const route = "https://app.example.org/workbench/stores/amazon/";
const credentials = { actor: "member-a", accessToken: "verified-server-token" };
const appId = "amzn1.sellerapps.app.2eca283f-9f5a-4d13-b16c-474000000057";

function callback(path: string, cookie = `${WORKBENCH_COOKIE_NAME}=org-a; __Secure-shuomi-amazon-flow=${flow}`) {
  return new Request(route + path, { headers: { cookie } });
}

describe("Amazon workbench authorization proxy", () => {
  it("keeps the flow cookie HttpOnly and validates the same-origin start", async () => {
    vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "https://app.example.org");
    try {
      const native = vi.fn(async () => Response.json(
        { authorizationUrl: `https://sellercentral.amazon.com/apps/authorize/consent?application_id=${appId}` },
        { headers: { "Set-Cookie": `__Secure-shuomi-amazon-flow=${flow}; Path=/workbench/stores/amazon; Max-Age=540; HttpOnly; Secure; SameSite=Lax` } },
      ));
      const start = (origin: string) => new Request(route + "begin", { method: "POST", headers: {
        origin, cookie: `${WORKBENCH_COOKIE_NAME}=org-a`, "X-Expected-User-ID": "member-a",
        "X-Expected-Organization-ID": "org-a", "X-Store-ID": storeId, "If-Match": '"3"',
      } });
      expect((await proxyAmazonAuthorization(start("https://wrong.example.org"), "begin", credentials, native)).status).toBe(403);
      expect(native).not.toHaveBeenCalled();
      const response = await proxyAmazonAuthorization(start("https://app.example.org"), "begin", credentials, native);
      expect(response.status).toBe(200);
      expect(response.headers.get("set-cookie")).toContain("HttpOnly");
      expect(response.headers.get("set-cookie")).toContain("Secure");
      expect(response.headers.get("set-cookie")).toContain("SameSite=lax");
      expect(await response.text()).not.toContain(flow);
    } finally {
      vi.unstubAllEnvs();
    }
  });

  it("removes callback parameters before redirecting and rejects duplicate state", async () => {
    const native = vi.fn(async (upstream: URL, init: RequestInit) => {
      expect(upstream.search).toBe("");
      expect(upstream.pathname).toContain("/store-amazon-authorization/redirect");
      expect(init.method).toBe("POST");
      const headers = new Headers(init.headers);
      expect(headers.get("authorization")).toBe("Bearer verified-server-token");
      expect(headers.get("x-requested-organization-id")).toBe("org-a");
      expect(headers.get("cookie")).toBe(`__Secure-shuomi-amazon-flow=${flow}`);
      expect(JSON.parse(String(init.body))).toEqual({ state: "state-a", sellerId: "SELLERA", code: "secret-code" });
      return Response.json({ storeId, generation: 1 });
    });
    const invalid = await proxyAmazonAuthorization(callback("redirect?state=one&state=two&selling_partner_id=SELLERA&spapi_oauth_code=secret-code"), "redirect", credentials, native as typeof fetch);
    expect(invalid.headers.get("location")).toBe("/workbench/stores/amazon/result?reason=restart");
    expect(native).not.toHaveBeenCalled();
    const returned = await proxyAmazonAuthorization(callback("redirect?state=state-a&selling_partner_id=SELLERA&spapi_oauth_code=secret-code"), "redirect", credentials, native as typeof fetch);
    expect(returned.status).toBe(303);
    expect(returned.headers.get("location")).toBe(`/workbench/stores/${storeId}`);
    expect(returned.headers.get("referrer-policy")).toBe("no-referrer");
    expect(returned.headers.get("cache-control")).toBe("no-store");
    expect(await returned.text()).not.toContain("secret-code");
  });

  it("accepts only the exact Amazon login callback destination after the Store claim", async () => {
    const callbackUri = encodeURIComponent(`https://amazon.com/apps/authorize/confirm/${appId}`);
    const login = callback(`login?amazon_callback_uri=${callbackUri}&amazon_state=amazon-state&selling_partner_id=SELLERA`);
    const bad = vi.fn(async () => Response.json({ authorizationUrl: `https://example.org/apps/authorize/confirm/${appId}` }));
    expect((await proxyAmazonAuthorization(login, "login", credentials, bad)).headers.get("location")).toBe("/workbench/stores/amazon/result?reason=restart");
    const valid = vi.fn(async () => Response.json({ authorizationUrl: `https://amazon.com/apps/authorize/confirm/${appId}?state=next` }));
    expect((await proxyAmazonAuthorization(login, "login", credentials, valid)).headers.get("location")).toBe(`https://amazon.com/apps/authorize/confirm/${appId}?state=next`);
  });

  it("redirects a signed-out callback to a safe page without sending the code upstream", async () => {
    const native = vi.fn(async () => Response.json({ storeId }));
    const result = await proxyAmazonAuthorization(
      callback("redirect?state=private-state&selling_partner_id=SELLERA&spapi_oauth_code=private-code"),
      "redirect", { actor: "", accessToken: "" }, native,
    );
    expect(result.status).toBe(303);
    expect(result.headers.get("location")).toBe("/workbench/stores/amazon/result?reason=signin");
    expect(result.headers.get("cache-control")).toBe("no-store");
    expect(result.headers.get("referrer-policy")).toBe("no-referrer");
    expect(native).not.toHaveBeenCalled();
  });
});
