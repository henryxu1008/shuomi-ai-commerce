import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";
const mockedAuthState = vi.hoisted(() => ({ session: null as Record<string, unknown> | null }));
const mockedServerToken = vi.hoisted(() => ({ accessToken: "" }));
vi.mock("@/lib/server/zitadel-server-token", () => ({
  readZitadelServerAccessToken: vi.fn(() => mockedServerToken.accessToken),
}));
import { buildVerifiedUpstreamHeaders, verifyUpstreamRequestIdentity } from "./upstream-identity";
import { verifyZitadelAccessToken } from "@/lib/server/zitadel-auth";
describe("buildVerifiedUpstreamHeaders", () => {
  it("forwards the public host and protocol to the internal API", () => {
    const request = new Request("https://pod.shuomiai.com/api/listing-kits/uploads/images", {
      headers: {
        host: "pod.shuomiai.com",
        "x-forwarded-proto": "https",
      },
    });

    const headers = buildVerifiedUpstreamHeaders(request.headers, undefined, "https");

    expect(headers.get("X-Forwarded-Host")).toBe("pod.shuomiai.com");
    expect(headers.get("X-Forwarded-Proto")).toBe("https");
  });

  it("ignores caller-supplied forwarded host and protocol", () => {
    const request = new Request("https://pod.shuomiai.com/api/listing-kits/uploads/images", {
      headers: {
        host: "pod.shuomiai.com",
        "x-forwarded-host": "attacker.example",
        "x-forwarded-proto": "http",
      },
    });

    const headers = buildVerifiedUpstreamHeaders(
      request.headers,
      undefined,
      "https",
      "pod.shuomiai.com",
    );

    expect(headers.get("X-Forwarded-Host")).toBe("pod.shuomiai.com");
    expect(headers.get("X-Forwarded-Proto")).toBe("https");
  });

  it("maps verified ZITADEL identity to verified identity headers", () => {
    const request = new Request("http://localhost/api/sds/products", {
      headers: {
        accept: "application/json",
        authorization: "Bearer user-token",
      },
    });

    const headers = buildVerifiedUpstreamHeaders(request.headers, {
      tenantId: "org-286",
      userId: "user-42",
      userType: "zitadel",
      roles: ["platform_admin", "listingkit_admin"],
    });

    expect(headers.get("Authorization")).toBe("Bearer user-token");
    expect(headers.get("tenant-id")).toBe("org-286");
    expect(headers.get("X-Tenant-ID")).toBe("org-286");
    expect(headers.get("X-User-ID")).toBe("user-42");
    expect(headers.get("X-User-Type")).toBe("zitadel");
    expect(headers.get("X-User-Roles")).toBe("platform_admin,listingkit_admin");
  });

  it("does not forward legacy gateway-only headers", () => {
    const request = new Request("http://localhost/api/sds/products", {
      headers: {
        accept: "application/json",
        "visit-tenant-id": "389",
        "login-user": encodeURIComponent(JSON.stringify({ id: 42 })),
      },
    });

    const headers = buildVerifiedUpstreamHeaders(request.headers);

    expect(headers.get("visit-tenant-id")).toBeNull();
    expect(headers.get("login-user")).toBeNull();
  });

  it("does not treat caller tenant headers as a verified identity", () => {
    const request = new Request("http://localhost/api/sds/products", {
      headers: {
        "tenant-id": "forged-tenant",
        "x-tenant-id": "forged-tenant",
      },
    });

    const headers = buildVerifiedUpstreamHeaders(request.headers);

    expect(headers.get("tenant-id")).toBeNull();
    expect(headers.get("X-Tenant-ID")).toBeNull();
  });

});

describe("verifyUpstreamRequestIdentity", () => {
  function verifyRequest(request: NextRequest) {
    return verifyUpstreamRequestIdentity(
      request,
      mockedAuthState.session as never,
    );
  }

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
    mockedAuthState.session = null;
    mockedServerToken.accessToken = "";
  });

  it("returns 503 when ZITADEL auth is required but not configured", async () => {
    const request = new NextRequest("http://localhost/api/sds/products");

    const result = await verifyRequest(request);

    expect(result.response?.status).toBe(503);
  });

  it("does not return a local debug identity when a retired auth bypass variable is set", async () => {
    vi.stubEnv("LISTINGKIT_UI_BYPASS_AUTH_GATE", "1");

    const result = await verifyRequest(
      new NextRequest("http://localhost/api/sds/products"),
    );

    expect(result.identity).toBeUndefined();
    expect(result.token).toBeUndefined();
    expect(result.response?.status).toBe(503);
  });

  it("rejects requests when no real ZITADEL session is present", async () => {
    vi.stubEnv("ZITADEL_ISSUER_URL", "https://issuer.example.com");
    vi.stubEnv("ZITADEL_CLIENT_ID", "client-1");

    const result = await verifyRequest(
      new NextRequest("http://localhost/api/sds/products"),
    );

    expect(result.identity).toBeUndefined();
    expect(result.token).toBeUndefined();
    expect(result.response?.status).toBe(401);
  });

  it("accepts an incoming bearer token without ZITADEL introspection in the proxy", async () => {
    vi.stubEnv("ZITADEL_ISSUER_URL", "https://issuer.example.com");
    vi.stubEnv("ZITADEL_CLIENT_ID", "client-1");
    const fetchMock = vi.fn<typeof fetch>();
    vi.stubGlobal("fetch", fetchMock);

    const result = await verifyRequest(
      new NextRequest("http://localhost/api/sds/products", {
        headers: { authorization: "Bearer caller-token-1" },
      }),
    );

    expect(result.response).toBeUndefined();
    expect(result.token).toBe("caller-token-1");
    expect(result.identity).toBeUndefined();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("returns the Auth.js session identity when a session is present", async () => {
    vi.stubEnv("ZITADEL_ISSUER_URL", "https://issuer.example.com");
    vi.stubEnv("ZITADEL_CLIENT_ID", "client-1");
    mockedServerToken.accessToken = "session-token-1";
    mockedAuthState.session = {
      issuerUrl: "https://issuer.example.com",
      clientId: "client-1",
      identityVersion: 3,
      identity: {
        tenantId: "org-286",
        userId: "user-42",
        username: "admin",
        userType: "zitadel",
        roles: ["listingkit_admin"],
      },
    };

    const result = await verifyRequest(
      new NextRequest("http://localhost/api/sds/products"),
    );

    expect(result.response).toBeUndefined();
    expect(result.token).toBe("session-token-1");
    expect(result.identity).toEqual({
      tenantId: "org-286",
      userId: "user-42",
      username: "admin",
      userType: "zitadel",
      roles: ["listingkit_admin"],
    });
  });

  it("forwards an Auth.js session token without revalidating the token in the proxy", async () => {
    vi.stubEnv("ZITADEL_ISSUER_URL", "https://issuer.example.com");
    vi.stubEnv("ZITADEL_CLIENT_ID", "client-1");
    mockedServerToken.accessToken = "session-token-1";
    mockedAuthState.session = {
      issuerUrl: "https://issuer.example.com",
      clientId: "client-1",
      identityVersion: 3,
      identity: {
        tenantId: "org-286",
        userId: "user-42",
        username: "admin",
        userType: "zitadel",
        roles: ["listingkit_admin"],
      },
    };
    const fetchMock = vi.fn<typeof fetch>();
    vi.stubGlobal("fetch", fetchMock);

    const result = await verifyRequest(
      new NextRequest("http://localhost/api/sds/products"),
    );

    expect(result.response).toBeUndefined();
    expect(result.token).toBe("session-token-1");
    expect(result.identity).toEqual({
      tenantId: "org-286",
      userId: "user-42",
      username: "admin",
      userType: "zitadel",
      roles: ["listingkit_admin"],
    });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("rejects unmarked legacy stored sessions without proxy-side introspection", async () => {
    vi.stubEnv("ZITADEL_ISSUER_URL", "https://issuer.example.com");
    vi.stubEnv("ZITADEL_CLIENT_ID", "client-1");
    mockedServerToken.accessToken = "session-token-1";
    mockedAuthState.session = {
      identity: {
        tenantId: "org-286",
        userId: "user-42",
        username: "admin",
        userType: "zitadel",
        roles: ["listingkit_admin"],
      },
    };
    const fetchMock = vi.fn<typeof fetch>();
    vi.stubGlobal("fetch", fetchMock);

    const result = await verifyRequest(
      new NextRequest("http://localhost/api/sds/products"),
    );

    expect(result.identity).toBeUndefined();
    expect(result.token).toBeUndefined();
    expect(result.response?.status).toBe(401);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("rejects stored sessions created for a different ZITADEL issuer or client", async () => {
    vi.stubEnv("ZITADEL_ISSUER_URL", "https://issuer.example.com");
    vi.stubEnv("ZITADEL_CLIENT_ID", "client-1");
    mockedServerToken.accessToken = "session-token-1";
    mockedAuthState.session = {
      issuerUrl: "http://localhost:8080",
      clientId: "old-client",
      identity: {
        tenantId: "org-286",
        userId: "user-42",
        username: "admin",
        userType: "zitadel",
        roles: ["listingkit_admin"],
      },
    };

    const result = await verifyRequest(
      new NextRequest("http://localhost/api/sds/products"),
    );

    expect(result.identity).toBeUndefined();
    expect(result.token).toBeUndefined();
    expect(result.response?.status).toBe(401);
  });
});

describe("verifyZitadelAccessToken", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("checks bearer token through ZITADEL introspection", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          active: true,
          sub: "user-42",
          "urn:zitadel:iam:user:resourceowner:id": "org-286",
          "urn:zitadel:iam:org:project:roles": {
            platform_admin: {},
          },
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      verifyZitadelAccessToken(
        "access-token-1",
        {
          issuerUrl: "https://issuer.example.com",
          clientId: "client-1",
          clientSecret: "secret-1",
          scopes: "openid profile",
        },
        {
          authorization_endpoint: "https://issuer.example.com/oauth/v2/authorize",
          token_endpoint: "https://issuer.example.com/oauth/v2/token",
          introspection_endpoint:
            "https://issuer.example.com/oauth/v2/introspect",
        },
      ),
    ).resolves.toEqual({
      tenantId: "org-286",
      userId: "user-42",
      userType: "zitadel",
      roles: ["platform_admin"],
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "https://issuer.example.com/oauth/v2/introspect",
      expect.objectContaining({
        body: "token=access-token-1&token_type_hint=access_token",
        method: "POST",
      }),
    );
    const init = fetchMock.mock.calls[0]?.[1];
    const headers = new Headers(init?.headers);
    expect(headers.get("Authorization")).toMatch(/^Basic /);
    expect(headers.get("Content-Type")).toBe(
      "application/x-www-form-urlencoded",
    );
  });
});
