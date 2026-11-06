import { describe, it, expect, vi, afterEach } from "vitest";
import { buildDataRequest, dataBrowserResponse } from "./data-services-proxy";
import { dataRoute } from "@/lib/contracts/data-services";
const id = "a233d58b-1fd3-40d7-a983-d35bbec45313";
afterEach(() => vi.unstubAllEnvs());
function request(body: string) { return new Request("https://app.example.test/api/workbench/data-services/keys", { method: "POST", headers: { Origin: "https://app.example.test", "Content-Type": "application/json", "Sec-Fetch-Site": "same-origin", Cookie: "shuomi_effective_organization=org", "X-Expected-User-ID": "user", "X-Expected-Organization-ID": "org", "Idempotency-Key": id }, body }); }
describe("data BFF boundaries", () => {
    const protocolError = (code: string) => ({ code, message: "native authorization response", requestId: "request", fieldErrors: [] });
    it.each(["options", "amazon/jobs", "custom"])("preserves native permission denial on %s instead of reporting a gateway failure", async path => {
        const response = await dataBrowserResponse(Response.json(protocolError("PERMISSION_DENIED"), { status: 403 }), dataRoute("GET", path.split("/"))!, false);
        expect(response.status).toBe(403);
        expect(await response.json()).toEqual({ error: { code: "PERMISSION_DENIED", message: "PERMISSION_DENIED" } });
    });
    it.each([
        [401, "AUTHENTICATION_REQUIRED"], [403, "ORGANIZATION_ACCESS_DENIED"],
        [403, "ORGANIZATION_ACCESS_REVOKED"], [403, "ORGANIZATION_SUSPENDED"],
        [409, "ORGANIZATION_SELECTION_REQUIRED"], [503, "DEPENDENCY_UNAVAILABLE"],
        [504, "DEADLINE_EXCEEDED"], [502, "INVALID_UPSTREAM_RESPONSE"],
    ] as const)("preserves qualified native read failure %s %s", async (status, code) => {
        const response = await dataBrowserResponse(Response.json(protocolError(code), { status }), dataRoute("GET", ["options"])!, false);
        expect(response.status).toBe(status);
        expect((await response.json()).error.code).toBe(code);
    });
    it("keeps a native permission denial definitive but a native server failure ambiguous for writes", async () => {
        const route = dataRoute("POST", ["amazon", "jobs"])!;
        const denied = await dataBrowserResponse(Response.json(protocolError("PERMISSION_DENIED"), { status: 403 }), route, true);
        expect(denied.status).toBe(403);
        expect((await denied.json()).error.code).toBe("PERMISSION_DENIED");
        const unknown = await dataBrowserResponse(Response.json(protocolError("DEPENDENCY_UNAVAILABLE"), { status: 503 }), route, true);
        expect(unknown.status).toBe(503);
        expect((await unknown.json()).error.code).toBe("DATA_UNKNOWN");
    });
    it.each([
        [401, protocolError("PERMISSION_DENIED")],
        [403, protocolError("UNREGISTERED_FAILURE")],
        [403, { ...protocolError("PERMISSION_DENIED"), scope: "untrusted" }],
    ])("rejects contradictory or malformed native envelopes without treating a write as safely absent", async (status, payload) => {
        const read = await dataBrowserResponse(Response.json(payload, { status }), dataRoute("GET", ["options"])!, false);
        expect(read.status).toBe(502);
        const write = await dataBrowserResponse(Response.json(payload, { status }), dataRoute("POST", ["keys"])!, true);
        expect(write.status).toBe(503);
        expect((await write.json()).error.code).toBe("DATA_UNKNOWN");
    });
    it("retains an ambiguous mutation when upstream status contradicts its error code", async () => {
        const response = await dataBrowserResponse(Response.json({ error: { code: "DATA_UNKNOWN" } }, { status: 409 }), dataRoute("POST", ["keys"])!, true);
        expect(response.status).toBe(503);
        expect((await response.json()).error.code).toBe("DATA_UNKNOWN");
    });
    it("rejects ambiguous JSON and stale enterprise before dispatch", async () => { vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "https://app.example.test"); const req = request('{"name":"a","name":"b"}'); expect((await buildDataRequest(req, ["keys"], "server-token", "user") as Response).status).toBe(400); const stale = request('{}'); stale.headers.set("X-Expected-Organization-ID", "other"); expect((await buildDataRequest(stale, ["keys"], "server-token", "user") as Response).status).toBe(409); });
    it("uses server bearer and fixed route, stripping browser credentials", async () => {
        vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "https://app.example.test");
        const req = request(JSON.stringify({ name: "key", expiresAt: "2026-10-20T00:00:00Z", dailyRows: 1, monthlyCostFen: 5, permissions: ["amazon.acquire"] }));
        req.headers.set("Authorization", "DataKey never-forward");
        const mapped = await buildDataRequest(req, ["keys"], "server-token", "user");
        expect(mapped).not.toBeInstanceOf(Response);
        if (mapped instanceof Response)
            return;
        expect(mapped.init.headers.get("Authorization")).toBe("Bearer server-token");
        expect(mapped.init.headers.get("Cookie")).toBeNull();
        expect(mapped.url.pathname).toBe("/api/v1/workbench/data-services/keys");
    });
    it("preserves unknown when a dispatched mutation response is malformed", async () => { const route = dataRoute("POST", ["keys"])!; const response = await dataBrowserResponse(new Response('{"key":{},"secret":"first","secret":"second"}', { headers: { "Content-Type": "application/json" } }), route, true); expect(response.status).toBe(503); expect((await response.json()).error.code).toBe("DATA_UNKNOWN"); });
});
