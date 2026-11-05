// @vitest-environment node
import { afterEach, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { createHash } from "node:crypto";
import { CookieJar } from "jsdom";

vi.mock("@/lib/server/zitadel-acceptance-token", () => ({
  persistZitadelAcceptanceToken: vi.fn(async () => false),
}));

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.resetModules();
});

const origin = "https://127.0.0.37:47644";

async function setup(env = "AUTH_URL", secrets = {}) {
  vi.stubEnv("AUTH_URL", undefined);
  vi.stubEnv("NEXTAUTH_URL", undefined);
  vi.stubEnv(env, origin);
  for (const [name, value] of Object.entries({
    AUTH_SECRET: "synthetic-test-secret-".repeat(3),
    NEXTAUTH_SECRET: undefined,
    AUTH_SECRET_1: undefined,
    AUTH_SECRET_2: undefined,
    AUTH_SECRET_3: undefined,
    ZITADEL_CLIENT_SECRET: "synthetic-client-secret",
    ...secrets,
  })) {
    vi.stubEnv(name, value);
  }
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", origin);
  vi.stubEnv("ZITADEL_ISSUER_URL", "https://issuer.example.invalid");
  vi.stubEnv("ZITADEL_CLIENT_ID", "fixture-client");
  return (await import("@/auth")).handlers;
}

it.each([
  ["OIDC client ID alone", { AUTH_SECRET: undefined, ZITADEL_CLIENT_SECRET: undefined }],
  ["OIDC client secret", { AUTH_SECRET: undefined }],
])("rejects session requests with %s", async (_label, secrets) => {
  vi.spyOn(console, "error").mockImplementation(() => {});
  const handlers = await setup("AUTH_URL", secrets);
  const response = await handlers.GET(new NextRequest(`${origin}/api/auth/session`));
  expect(response.status).toBe(500);
  expect(response.headers.getSetCookie()).toEqual([]);
});

it.each(["AUTH_SECRET", "NEXTAUTH_SECRET"])(
  "rejects whitespace-only %s before Auth.js reads the environment",
  async (name) => {
    await expect(setup("AUTH_URL", { AUTH_SECRET: undefined, [name]: "   " }))
      .rejects.toThrow("Configure a nonblank AUTH_SECRET or NEXTAUTH_SECRET");
  },
);

it("accepts the dedicated NextAuth secret alias", async () => {
  const handlers = await setup("AUTH_URL", {
    AUTH_SECRET: undefined,
    NEXTAUTH_SECRET: "synthetic-dedicated-alias-secret-".repeat(3),
  });
  const response = await handlers.GET(new NextRequest(`${origin}/api/auth/providers`));
  expect(response.status).toBe(200);
  expect(await response.json()).toHaveProperty("zitadel");
});

it.each(["AUTH_URL", "NEXTAUTH_URL"])(
  "preserves the configured loopback origin through Auth.js public handlers using %s",
  async (env) => {
    const handlers = await setup(env);
    const request = new NextRequest(`${origin}/api/auth/providers`, {
      headers: { host: "untrusted.example", "x-forwarded-host": "untrusted.example" },
    });
    // Pinned Next.js normalizes 127/8 to localhost. Auth.js must receive the
    // configured public URL, matching signIn and the registered callback URI.
    expect(request.url).toBe("https://localhost:47644/api/auth/providers");
    const response = await handlers.GET(request);
    expect(response.status).toBe(200);
    expect(await response.json()).toMatchObject({
      zitadel: {
        signinUrl: `${origin}/api/auth/signin/zitadel`,
        callbackUrl: `${origin}/api/auth/callback/zitadel`,
      },
    });
  },
);

it("uses the authorization callback URI for the native code grant and retains PKCE and CSRF checks", async () => {
  const grants = [];
  vi.stubGlobal("fetch", vi.fn(async (input, init) => {
    const url = String(input);
    if (url === "https://issuer.example.invalid/.well-known/openid-configuration") {
      return Response.json({
        issuer: "https://issuer.example.invalid",
        authorization_endpoint: "https://issuer.example.invalid/authorize",
        token_endpoint: "https://issuer.example.invalid/token",
        jwks_uri: "https://issuer.example.invalid/keys",
        userinfo_endpoint: "https://issuer.example.invalid/userinfo",
        code_challenge_methods_supported: ["S256"],
      });
    }
    if (url === "https://issuer.example.invalid/token") {
      grants.push(new URLSearchParams(init?.body));
      // Stop at the identity authority boundary after inspecting the actual
      // native grant. No fabricated identity/session is returned by this test.
      return Response.json({ error: "invalid_grant" }, { status: 400 });
    }
    throw new Error(`Unexpected fixture endpoint: ${url}`);
  }));
  vi.spyOn(console, "error").mockImplementation(() => {});
  const handlers = await setup();
  const jar = new CookieJar();
  const request = (path, init = {}) => new NextRequest(`${origin}/api/auth/${path}`, {
    ...init,
    headers: { ...init.headers, cookie: jar.getCookieStringSync(origin) },
  });
  const saveCookies = (response) => {
    response.headers.getSetCookie().forEach(cookie => jar.setCookieSync(cookie, origin));
  };
  const csrf = await handlers.GET(request("csrf"));
  saveCookies(csrf);
  const { csrfToken } = await csrf.json();
  const signin = await handlers.POST(request("signin/zitadel", {
    method: "POST",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({ csrfToken, callbackUrl: `${origin}/workbench/orders` }),
  }));
  saveCookies(signin);
  const authorization = new URL(signin.headers.get("location"));
  expect(authorization.origin).toBe("https://issuer.example.invalid");
  const redirectURI = authorization.searchParams.get("redirect_uri");
  expect(redirectURI).toBe(`${origin}/api/auth/callback/zitadel`);
  expect(authorization.searchParams.get("code_challenge_method")).toBe("S256");
  await handlers.GET(request("callback/zitadel?code=synthetic-code"));
  expect(grants).toHaveLength(1);
  expect(grants[0].get("redirect_uri")).toBe(redirectURI);
  expect(grants[0].get("code")).toBe("synthetic-code");
  expect(createHash("sha256").update(grants[0].get("code_verifier")).digest("base64url"))
    .toBe(authorization.searchParams.get("code_challenge"));

  jar.removeAllCookiesSync();
  await handlers.GET(request("callback/zitadel?code=synthetic-code-without-check"));
  expect(grants).toHaveLength(1);
  const denied = await handlers.POST(request("signin/zitadel", {
    method: "POST",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({ csrfToken: "wrong-csrf-token" }),
  }));
  expect(new URL(denied.headers.get("location")).searchParams.get("error")).toBe("MissingCSRF");
  expect(grants).toHaveLength(1);
});
