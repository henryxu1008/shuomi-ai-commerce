import { afterEach, expect, it, vi } from "vitest";
import { skillAdmissionSchema, skillPageSchema } from "@/lib/contracts/skill";
import { proxySkill } from "@/lib/server/skill-proxy";
import { WORKBENCH_COOKIE_NAME } from "@/lib/server/workbench-proxy";
import { skillRequest } from "./skill";

const scope = { userId: "alice", organizationId: "org-a" };
const page = { items: [], nextCursor: "" };

afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });

function connectProxy(selectedOrganization = "org-a", payload: unknown = page) {
  vi.stubEnv("LISTINGKIT_SERVICE_API_BASE", "http://service.test/api/v1");
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "http://app.test");
  const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const url = String(input);
    if (!url.startsWith("/api/workbench/skills/")) return Response.json(payload);
    const headers = new Headers(init?.headers);
    headers.set("Cookie", `${WORKBENCH_COOKIE_NAME}=${selectedOrganization}`);
    return proxySkill(new Request(`http://app.test${url}`, { ...init, headers }), "token", "alice");
  });
  vi.stubGlobal("fetch", fetch);
  return fetch;
}

it.each(["organization", "organization?limit=20", "organization?limit=20&cursor=next_page"])(
  "reads the organization skill list through the strict proxy: %s",
  async path => {
    const fetch = connectProxy();
    await expect(skillRequest(scope, path, skillPageSchema)).resolves.toEqual(page);
    expect(fetch).toHaveBeenCalledTimes(2);
    expect(new Headers(fetch.mock.calls[0][1]?.headers).get("X-Expected-Organization-ID")).toBe("org-a");
    expect(fetch.mock.calls[1][0]).toBe(`http://service.test/api/v1/workbench/skills/${path}`);
    expect(new Headers(fetch.mock.calls[1][1]?.headers).get("X-Requested-Organization-ID")).toBe("org-a");
  },
);

it.each([scope, { userId: "alice" }])("rejects a changed or missing organization before upstream dispatch: %j", async expectedScope => {
  const fetch = connectProxy("org-b");
  await expect(skillRequest(expectedScope, "organization?limit=20", skillPageSchema)).rejects.toMatchObject({ code: "ORGANIZATION_CONTEXT_CHANGED", unknown: false });
  expect(fetch).toHaveBeenCalledTimes(1);
});

it.each(["personal?limit=20", "official?limit=20"])("keeps %s independent of the selected organization", async path => {
  const fetch = connectProxy();
  await expect(skillRequest(scope, path, skillPageSchema)).resolves.toEqual(page);
  expect(new Headers(fetch.mock.calls[0][1]?.headers).has("X-Expected-Organization-ID")).toBe(false);
  expect(new Headers(fetch.mock.calls[1][1]?.headers).has("X-Requested-Organization-ID")).toBe(false);
});

it("preserves the original admission command identity and revision through the strict proxy", async () => {
  const admission = { policy: { allowPersonal: true, revision: "1" }, canConfigure: true };
  const fetch = connectProxy("org-a", admission);
  const key = "11111111-1111-4111-8111-111111111111";
  await expect(skillRequest(scope, "organization/admission", skillAdmissionSchema, {
    method: "PUT",
    headers: { Origin: "http://app.test", "Content-Type": "application/json", "Idempotency-Key": key, "If-Match": '"0"' },
    body: JSON.stringify({ allowPersonal: true }),
  })).resolves.toEqual(admission);
  expect(fetch).toHaveBeenCalledTimes(2);
  const forwarded = fetch.mock.calls[1][1];
  const headers = new Headers(forwarded?.headers);
  expect(headers.get("X-Requested-Organization-ID")).toBe("org-a");
  expect(headers.get("Idempotency-Key")).toBe(key);
  expect(headers.get("If-Match")).toBe('"0"');
  expect(forwarded).toMatchObject({ method: "PUT", body: JSON.stringify({ allowPersonal: true }) });
});
