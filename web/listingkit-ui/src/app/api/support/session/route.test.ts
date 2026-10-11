import { createHash, createHmac } from "node:crypto";
import { NextRequest } from "next/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const auth = vi.hoisted(() => ({ session: null as Record<string, unknown> | null, stalled: false }));
vi.mock("@/auth", () => ({ serverAuth: (handler: (request: NextRequest) => unknown) =>
  (request: NextRequest) => auth.stalled ? new Promise(() => {}) : handler(Object.assign(request, { auth: auth.session })) }));
import { GET, POST } from "./route";

const origin = "https://support.example.test";
const secret = "a".repeat(64);
const subject = (tenant = "tenant-a", user = "user-a", issuer = "https://identity.example.test") =>
  `platform:${createHash("sha256").update(JSON.stringify([issuer, tenant, user])).digest("hex")}`;
const request = (body: unknown = { subject: subject(), widgetAuthToken: "visitor-token" }, headers: Record<string, string> = {}) =>
  new NextRequest("https://app.example.test/api/support/session", { method: "POST", headers: { origin: "https://app.example.test", "sec-fetch-site": "same-origin", "content-type": "application/json", ...headers }, body: JSON.stringify(body) });
const session = () => ({ identityVersion: 3, issuerUrl: "https://identity.example.test", clientId: "app-client", accessToken: "private-oidc-access-token", identity: { tenantId: "tenant-a", userId: "user-a", username: "private-person", roles: [] } });

describe("platform support session", () => {
  beforeEach(() => {
    vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "https://app.example.test");
    vi.stubEnv("ZITADEL_ISSUER_URL", "https://identity.example.test");
    vi.stubEnv("ZITADEL_CLIENT_ID", "app-client");
    vi.stubEnv("CHATWOOT_ORIGIN", origin);
    vi.stubEnv("CHATWOOT_WEBSITE_TOKEN", "public-website-token");
    vi.stubEnv("CHATWOOT_IDENTITY_SECRET", secret);
    vi.stubEnv("CHATWOOT_IDENTITY_SECRET_FILE", "");
    auth.session = session(); auth.stalled = false;
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json({ id: 42, has_name: true })));
  });
  afterEach(() => { vi.useRealTimers(); vi.unstubAllEnvs(); vi.unstubAllGlobals(); vi.restoreAllMocks(); });

  it("returns only public config and a namespaced subject, with no provider mutation", async () => {
    const response = await GET(new NextRequest("https://app.example.test/api/support/session"));
    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({ origin, websiteToken: "public-website-token", subject: subject() });
    expect(response.headers.get("cache-control")).toBe("no-store");
    expect(fetch).not.toHaveBeenCalled();
  });
  it("keeps identical user IDs in different canonical tenants separate", async () => {
    auth.session = { ...session(), identity: { tenantId: "tenant-b", userId: "user-a" } };
    expect(await (await GET(new NextRequest("https://app.example.test/api/support/session"))).json()).toMatchObject({ subject: subject("tenant-b") });
    expect(subject("tenant-b")).not.toBe(subject());
  });
  it.each([null, { ...session(), error: "expired" }, { ...session(), accessToken: undefined }, { ...session(), issuerUrl: "https://old-realm.test" }, { ...session(), clientId: "old-client" }, { ...session(), identityVersion: 2 }])("rejects invalid/old sessions on both operations", async value => {
    auth.session = value;
    expect((await GET(new NextRequest("https://app.example.test/api/support/session"))).status).toBe(401);
    expect((await POST(request())).status).toBe(401);
    expect(fetch).not.toHaveBeenCalled();
  });
  it("applies existing app authorization", async () => {
    vi.stubEnv("LISTINGKIT_ZITADEL_ALLOWED_USER_IDS", "another-user");
    expect((await POST(request())).status).toBe(403);
    expect(fetch).not.toHaveBeenCalled();
  });
  it.each(["https://app.example.test", "http://support.example.test", "https://support.example.test/path", "https://user:pass@support.example.test", "https://support.example.test?x=y"]) ("fails closed for unsafe configuration %s", async configured => {
    vi.stubEnv("CHATWOOT_ORIGIN", configured);
    expect((await GET(new NextRequest("https://app.example.test/api/support/session"))).status).toBe(503);
    expect(fetch).not.toHaveBeenCalled();
  });
  it("fails closed for a missing/invalid secret file", async () => {
    vi.stubEnv("CHATWOOT_IDENTITY_SECRET", ""); vi.stubEnv("CHATWOOT_IDENTITY_SECRET_FILE", "C:/missing/support-secret");
    expect((await POST(request())).status).toBe(503);
    expect(fetch).not.toHaveBeenCalled();
  });
  it("accepts Chatwoot's native 24-character inbox identity secret", async () => {
    vi.stubEnv("CHATWOOT_IDENTITY_SECRET", "a".repeat(24));
    expect((await POST(request())).status).toBe(200);
  });
  it.each<Record<string, string>>([{ origin: "https://evil.test" }, { "sec-fetch-site": "cross-site" }, { origin: "" }])("rejects cross-site mutation before touching provider", async headers => {
    expect((await POST(request(undefined, headers))).status).toBe(403);
    expect(fetch).not.toHaveBeenCalled();
  });
  it.each([{ subject: subject("tenant-b"), widgetAuthToken: "visitor-token" }, { subject: subject(), widgetAuthToken: "visitor-token", userId: "victim" }])("rejects substituted or extra identity input", async body => {
    expect([400, 409]).toContain((await POST(request(body))).status);
    expect(fetch).not.toHaveBeenCalled();
  });
  it("rejects actual oversized bodies", async () => {
    expect((await POST(request({ subject: subject(), widgetAuthToken: "x".repeat(5000) }))).status).toBe(400);
    expect(fetch).not.toHaveBeenCalled();
  });
  it("signs server identity and only forwards the visitor credential to the fixed provider", async () => {
    const response = await POST(request());
    expect(response.status).toBe(200);
    const [url, init] = vi.mocked(fetch).mock.calls[0];
    expect(url).toBe(`${origin}/api/v1/widget/contact/set_user?website_token=public-website-token`);
    expect(init).toMatchObject({ method: "PATCH", redirect: "manual", cache: "no-store", credentials: "omit" });
    expect(new Headers(init?.headers).get("X-Auth-Token")).toBe("visitor-token");
    expect(init?.body).toBe(JSON.stringify({ identifier: subject(), name: `平台用户 ${subject().slice(-8)}`, identifier_hash: createHmac("sha256", secret).update(subject()).digest("hex") }));
    expect(JSON.stringify(init)).not.toContain("private-oidc-access-token");
    expect(await response.json()).toEqual({ ok: true });
  });
  it.each([Response.json({ error: "invalid hmac" }, { status: 401 }), Response.json({ id: 42, widget_auth_token: "different-token" }), Response.json({ unexpected: true }), new Response("redirect", { status: 302 }), Response.json({ id: 42, junk: "x".repeat(33000) })])("does not report a successful handshake for provider errors/rebinding", async upstream => {
    vi.mocked(fetch).mockResolvedValue(upstream);
    expect((await POST(request())).status).toBe(502);
  });
  it("bounds stalled auth and prevents late provider dispatch", async () => {
    vi.useFakeTimers(); auth.stalled = true;
    const pending = POST(request());
    await vi.advanceTimersByTimeAsync(15_000);
    expect((await pending).status).toBe(504);
    expect(fetch).not.toHaveBeenCalled();
  });
  it("bounds provider transport and streamed response bodies", async () => {
    vi.useFakeTimers();
    vi.mocked(fetch).mockResolvedValue(new Response(new ReadableStream({ start() {} }), { headers: { "content-type": "application/json" } }));
    const pending = POST(request());
    await vi.advanceTimersByTimeAsync(15_000);
    expect((await pending).status).toBe(504);
  });
});
