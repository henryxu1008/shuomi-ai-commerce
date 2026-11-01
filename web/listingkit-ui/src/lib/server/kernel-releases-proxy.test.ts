import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { proxyKernelReleases } from "./kernel-releases-proxy";
const id = "12345678-1234-4234-8234-123456789abc";
const payload = { version: "150.0.7871.186", os: "windows", arch: "amd64", minClientVersion: "1.8.0", maxClientVersionExclusive: "1.8.1-0", archiveFormat: "zip", size: 1, sha256: "a".repeat(64), downloadUrl: "https://assets.example.test/k.zip" };
const context = (permission = true) => ({ user: { id: "admin" }, homeOrganizationId: "org", effectiveOrganizationId: "org", selectionRequired: false, organizations: [{ id: "org", name: "Test", roles: [], permissions: permission ? ["listingkit.platform_admin"] : [] }] });
function request(body: unknown = payload, extra: Record<string, string> = {}) { return new Request(`https://console.test/api/account/kernel-releases/${id}`, { method: "PUT", headers: { "Content-Type": "application/json", Origin: "https://console.test", "X-Expected-User-ID": "admin", ...extra }, body: JSON.stringify(body) }); }
beforeEach(() => { vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "https://console.test"); vi.stubEnv("LISTINGKIT_SERVICE_API_BASE", "http://127.0.0.1:9000/api/v1"); });
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });
it("rejects missing authentication, changed identity, CSRF and invalid payload without sending", async () => {
  const fetch = vi.fn(); vi.stubGlobal("fetch", fetch);
  expect((await proxyKernelReleases(request(), "", "admin")).status).toBe(401);
  expect((await proxyKernelReleases(request(), "test-token", "other")).status).toBe(409);
  expect((await proxyKernelReleases(request(payload, { Origin: "https://evil.test" }), "test-token", "admin")).status).toBe(403);
  expect((await proxyKernelReleases(request({ ...payload, secret: "not-forwarded" }), "test-token", "admin")).status).toBe(400);
  expect(fetch).not.toHaveBeenCalled();
});
it("uses current server permission and refuses nonadministrators before publishing", async () => {
  const fetch = vi.fn().mockResolvedValue(Response.json(context(false))); vi.stubGlobal("fetch", fetch);
  expect((await proxyKernelReleases(request(), "test-token", "admin")).status).toBe(403);
  expect(fetch).toHaveBeenCalledTimes(1);
});
it("distinguishes a dependency failure before publication from an unknown write", async () => {
  const fetch = vi.fn().mockRejectedValue(new Error("context unavailable")); vi.stubGlobal("fetch", fetch);
  expect(await (await proxyKernelReleases(request(), "test-token", "admin")).json()).toEqual({ code: "NOT_SENT" });
  expect(fetch).toHaveBeenCalledTimes(1);
});
it("forwards only fixed publication metadata and server token, without enterprise identity headers", async () => {
  const fetch = vi.fn().mockResolvedValueOnce(Response.json(context())).mockResolvedValueOnce(Response.json({ releaseId: id, state: "published" })); vi.stubGlobal("fetch", fetch);
  const result = await proxyKernelReleases(request(), "test-token", "admin"); expect(result.status).toBe(200);
  expect(fetch.mock.calls[1][0]).toBe(`http://127.0.0.1:9000/api/v1/platform/browser/kernel-releases/${id}`);
  const options = fetch.mock.calls[1][1]; expect(JSON.parse(options.body)).toEqual(payload);
  const headers = new Headers(options.headers); expect(headers.get("Authorization")).toBe("Bearer test-token"); expect(headers.has("X-Requested-Organization-ID")).toBe(false); expect(headers.has("Cookie")).toBe(false);
  expect(JSON.stringify(await result.json())).not.toContain("test-token");
});
it("preserves the authoritative same-ID withdrawn PUT result", async () => {
  const fetch = vi.fn().mockResolvedValueOnce(Response.json(context())).mockResolvedValueOnce(Response.json({ releaseId: id, state: "withdrawn" })); vi.stubGlobal("fetch", fetch);
  const result = await proxyKernelReleases(request(), "test-token", "admin");
  expect(result.status).toBe(200); expect(await result.json()).toEqual({ releaseId: id, state: "withdrawn" });
  expect(fetch).toHaveBeenCalledTimes(2);
});
it("keeps publication uncertain after lost or malformed responses and does not retry", async () => {
  for (const result of [() => Promise.reject(new Error("lost")), () => Promise.resolve(Response.json({ releaseId: id, state: "published", token: "secret" })), () => Promise.resolve(Response.json({ code: "KERNEL_CATALOG_UNAVAILABLE" }, { status: 503 }))]) {
    const fetch = vi.fn().mockResolvedValueOnce(Response.json(context())).mockImplementationOnce(result); vi.stubGlobal("fetch", fetch);
    expect(await (await proxyKernelReleases(request(), "test-token", "admin")).json()).toEqual({ code: "OUTCOME_UNKNOWN" }); expect(fetch).toHaveBeenCalledTimes(2);
  }
});
it("rejects duplicate query keys and unlisted write operations", async () => {
  const fetch = vi.fn(); vi.stubGlobal("fetch", fetch);
  for (const url of ["?os=windows&os=linux&arch=amd64&clientVersion=1.8.0", `/${id}/withdraw`]) {
    const req = new Request("https://console.test/api/account/kernel-releases" + url, { headers: { "X-Expected-User-ID": "admin" } });
    expect((await proxyKernelReleases(req, "test-token", "admin")).status).toBeGreaterThanOrEqual(400);
  }
  expect(fetch).not.toHaveBeenCalled();
});
it("retains permission and identity checks for readback and bounds upstream responses", async () => {
  const req = () => new Request(`https://console.test/api/account/kernel-releases/${id}/artifact?os=windows&arch=amd64&clientVersion=1.8.0`, { headers: { "X-Expected-User-ID": "admin" } });
  const fetch = vi.fn().mockResolvedValueOnce(Response.json(context(false))); vi.stubGlobal("fetch", fetch);
  expect((await proxyKernelReleases(req(), "test-token", "admin")).status).toBe(403);
  fetch.mockReset().mockResolvedValueOnce(Response.json({ ...context(), user: { id: "other" } }));
  expect((await proxyKernelReleases(req(), "test-token", "admin")).status).toBe(409);
  fetch.mockReset().mockResolvedValueOnce(Response.json(context())).mockResolvedValueOnce(Response.json({ ...payload, releaseId: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", recommended: false, downloadOrigins: ["https://assets.example.test"] }));
  expect((await proxyKernelReleases(req(), "test-token", "admin")).status).toBe(503);
});
