// @vitest-environment node
import { afterEach, expect, it, vi } from "vitest";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";
import { pathToFileURL } from "node:url";
import { CookieJar } from "jsdom";
import { Auth } from "@auth/core";
import { encode } from "@auth/core/jwt";

vi.mock("@/lib/server/zitadel-acceptance-token", () => ({
  persistZitadelAcceptanceToken: vi.fn(async () => false),
}));
import { buildAuthConfig, buildServerAuthConfig } from "@/auth.config";
import { ZITADEL_IDENTITY_VERSION } from "@/lib/server/zitadel-identity";

// Test-only access to the pinned SDK's actual PKCE and chunk implementation.
// Production uses only the supported Auth.js cookie configuration API.
const require = createRequire(import.meta.url);
const core = dirname(require.resolve("@auth/core"));
const sdk = (file) => import(pathToFileURL(join(core, file)).href);
const { init } = await sdk("lib/init.js");
const { defaultCookies, SessionStore } = await sdk("lib/utils/cookie.js");
const { serialize } = await sdk("lib/vendored/cookie.js");
const { pkce } = await sdk("lib/actions/callback/oauth/checks.js");
const logger = { debug() {}, warn() {}, error() {} };
const originA = "https://localhost:35444";
const originB = "https://localhost:31444";
const secretA = "synthetic-fixture-secret-A-".repeat(3);
const secretB = "synthetic-fixture-secret-B-".repeat(3);

afterEach(() => { vi.unstubAllEnvs(); vi.unstubAllGlobals(); });

function configFor(origin, secret = secretA, client = "fixture-client") {
  vi.stubEnv("AUTH_URL", origin);
  vi.stubEnv("AUTH_SECRET", secret);
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", origin);
  vi.stubEnv("ZITADEL_ISSUER_URL", "https://issuer.example.invalid");
  vi.stubEnv("ZITADEL_CLIENT_ID", client);
  vi.stubGlobal("fetch", vi.fn(() => { throw new Error("Unexpected network request"); }));
  return { ...buildAuthConfig(), logger };
}

async function nativeOptions(config, origin) {
  return (await init({ authOptions: config, providerId: "zitadel", action: "callback",
    url: new URL(`${origin}/api/auth`), cookies: {} })).options;
}
const read = (jar, origin) => Object.fromEntries(jar.getCookiesSync(origin).map(c => [c.key, c.value]));
const write = (jar, origin, cookie) => jar.setCookieSync(serialize(cookie.name, cookie.value, cookie.options), origin);

async function issueSession(jar, origin, options, subject, padding = "") {
  const jwt = await encode({ ...options.jwt, salt: options.cookies.sessionToken.name, token: {
    sub: subject, accessToken: "synthetic-access-token", expiresAt: Math.floor(Date.now() / 1000) + 3600,
    identity: { tenantId: `org-${subject}`, userId: subject, username: subject, userType: "zitadel", roles: [] },
    identityVersion: ZITADEL_IDENTITY_VERSION, padding,
  } });
  const chunks = new SessionStore(options.cookies.sessionToken, {}, logger).chunk(jwt, {});
  chunks.forEach(cookie => write(jar, origin, cookie));
  return chunks;
}

async function session(jar, origin, config) {
  const errors = [];
  const response = await Auth(new Request(`${origin}/api/auth/session`, {
    headers: { cookie: jar.getCookieStringSync(origin) },
  }), { ...config, basePath: "/api/auth", logger: { ...logger, error(error) { errors.push(error.name); } } });
  const setCookies = response.headers.getSetCookie();
  setCookies.forEach(cookie => jar.setCookieSync(cookie, origin));
  return { body: await response.json(), errors, setCookies };
}

it("keeps native chunked sessions and one-use PKCE independent across interleaved localhost instances", async () => {
  const jar = new CookieJar();
  const a = configFor(originA);
  const b = configFor(originB, secretB);
  const oa = await nativeOptions(a, originA);
  const ob = await nativeOptions(b, originB);
  expect((await issueSession(jar, originA, oa, "A", "x".repeat(7000))).length).toBeGreaterThan(1);
  const challengeA = await pkce.create(oa);
  write(jar, originA, challengeA.cookie);
  await issueSession(jar, originB, ob, "B");
  const challengeB = await pkce.create(ob);
  write(jar, originB, challengeB.cookie);

  const sa = await session(jar, originA, a);
  expect(sa.errors).toEqual([]);
  expect(sa.body?.identity?.userId).toBe("A");
  const clearA = [];
  expect(typeof await pkce.use(read(jar, originA), clearA, oa)).toBe("string");
  clearA.forEach(cookie => write(jar, originA, cookie));
  await expect(pkce.use(read(jar, originA), [], oa)).rejects.toMatchObject({ name: "InvalidCheck" });
  expect(typeof await pkce.use(read(jar, originB), [], ob)).toBe("string");
  const sb = await session(jar, originB, b);
  expect(sb.errors).toEqual([]);
  expect(sb.body?.identity?.userId).toBe("B");
});

it("ignores and preserves foreign default cookies before the first sign-in and through a valid session", async () => {
  const jar = new CookieJar();
  const a = configFor(originA);
  const oa = await nativeOptions(a, originA);
  const legacy = defaultCookies(true);
  // A foreign application owns these exact global slots. Do not clear/copy them.
  const foreignOptions = { ...oa, cookies: legacy, jwt: { ...oa.jwt, secret: secretB } };
  await issueSession(jar, originB, foreignOptions, "foreign", "y".repeat(7000));
  write(jar, originB, (await pkce.create(foreignOptions)).cookie);
  for (const cookie of [legacy.state, legacy.csrfToken]) {
    write(jar, originB, { ...cookie, value: "foreign-application-value" });
  }
  const foreign = () => jar.getCookiesSync(originA).filter(c => c.key.includes("authjs.")).map(c => [c.key, c.value]);
  const before = foreign();
  const first = await session(jar, originA, a);
  expect(first.errors).toEqual([]);
  expect(first.body).toBeNull();
  expect(foreign()).toEqual(before);
  await issueSession(jar, originA, oa, "A");
  expect((await session(jar, originA, a)).body?.identity?.userId).toBe("A");
  expect(foreign()).toEqual(before);
});

it.each(["https://localhost:35444", "http://localhost:3000"])("retains all seven native cookie attributes and stable public/server names for %s", async origin => {
  const a = configFor(origin);
  const server = buildServerAuthConfig();
  expect(server.cookies).toEqual(a.cookies);
  const oa = await nativeOptions(a, origin);
  const defaults = defaultCookies(origin.startsWith("https:"));
  const keys = Object.keys(defaults);
  expect(Object.keys(a.cookies ?? {})).toEqual(keys);
  for (const key of keys) {
    expect(oa.cookies[key].options).toEqual(defaults[key].options);
    expect(oa.cookies[key].name).not.toBe(defaults[key].name);
    expect(oa.cookies[key].name.startsWith(origin.startsWith("https:") ? (key === "csrfToken" ? "__Host-" : "__Secure-") : "shuomi.")).toBe(true);
  }
  expect(configFor(`${origin}/api/auth`).cookies).toEqual(a.cookies);
  expect(configFor(origin, secretB).cookies).toEqual(a.cookies); // A secret drift cannot create a hidden new slot.
  expect(configFor(origin, secretA, "another-client").cookies).not.toEqual(a.cookies);
  const anotherPort = configFor(origin.replace(/:\d+$/, ":39888"));
  for (const key of keys) expect(anotherPort.cookies[key].name).not.toBe(a.cookies[key].name);
});

it("continues to reject a wrong secret/check under its own namespace without a legacy fallback", async () => {
  const jar = new CookieJar();
  const a = configFor(originA);
  const oa = await nativeOptions(a, originA);
  const wrong = await nativeOptions(configFor(originA, secretB), originA);
  await issueSession(jar, originA, wrong, "intruder");
  const bad = await pkce.create(wrong);
  write(jar, originA, bad.cookie);
  const result = await session(jar, originA, a);
  expect(result.body).toBeNull();
  expect(result.errors).toContain("JWTSessionError");
  await expect(pkce.use(read(jar, originA), [], oa)).rejects.toMatchObject({ name: "InvalidCheck" });
});

it("honors the SDK's existing NEXTAUTH_URL authority without a secure-cookie downgrade", () => {
  const canonical = configFor(originA);
  vi.stubEnv("AUTH_URL", undefined);
  vi.stubEnv("NEXTAUTH_URL", originA);
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://localhost:3000");
  const alias = buildAuthConfig();
  expect(alias.useSecureCookies).toBe(true);
  expect(alias.cookies).toEqual(canonical.cookies);
});

it.each(["not-a-url", "file:///tmp/app", "https://user:password@localhost:35444"])("rejects invalid configured cookie authority %s", origin => {
  expect(() => configFor(origin)).toThrow(/origin/i);
});
