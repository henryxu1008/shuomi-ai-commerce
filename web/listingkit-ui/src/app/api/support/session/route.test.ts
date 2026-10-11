import { createHash } from "node:crypto";
import { NextRequest } from "next/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const auth = vi.hoisted(() => ({ session: null as Record<string, unknown> | null }));
vi.mock("@/auth", () => ({ serverAuth: (handler: (request: NextRequest) => unknown) => (request: NextRequest) => handler(Object.assign(request, { auth: auth.session })) }));
import { GET } from "./route";
import { POST } from "../[...path]/route";

const subject = `platform:${createHash("sha256").update(JSON.stringify(["https://identity.example.test", "tenant-a", "user-a"])).digest("hex")}`;
const contact = { id: 7, identifier: subject };
const conversation = { id: 11, account_id: 2, inbox_id: 3, status: "open", created_at: 1, custom_attributes: { platform_support_subject: subject }, meta: { sender: contact } };
const messages = { payload: [
  { id: 19, conversation_id: 11, content: "我的问题", message_type: 0, private: false, created_at: 1, attachments: [] },
  { id: 20, conversation_id: 11, content: "人工回复", message_type: 1, private: false, created_at: 2, attachments: [] },
  { id: 21, conversation_id: 11, content: "内部笔记", message_type: 1, private: true, created_at: 3 },
] };
const get = (path: string) => GET(new NextRequest(`https://app.example.test/api/support/${path}`));
const send = (content = "请帮忙查看报错") => {
  const body = new FormData(); body.set("subject", subject); body.set("content", content); body.set("conversationId", "11");
  return POST(new NextRequest("https://app.example.test/api/support/conversations", { method: "POST", headers: { origin: "https://app.example.test", "sec-fetch-site": "same-origin" }, body }));
};

describe("Chatwoot server API support", () => {
  beforeEach(() => {
    vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "https://app.example.test"); vi.stubEnv("ZITADEL_ISSUER_URL", "https://identity.example.test"); vi.stubEnv("ZITADEL_CLIENT_ID", "app-client");
    vi.stubEnv("CHATWOOT_ORIGIN", "http://localhost:36500"); vi.stubEnv("CHATWOOT_ACCOUNT_ID", "2"); vi.stubEnv("CHATWOOT_INBOX_ID", "3"); vi.stubEnv("CHATWOOT_API_TOKEN", "private-integration-token"); vi.stubEnv("CHATWOOT_API_TOKEN_FILE", "");
    auth.session = { identityVersion: 3, issuerUrl: "https://identity.example.test", clientId: "app-client", accessToken: "private-oidc-token", identity: { tenantId: "tenant-a", userId: "user-a", roles: [] } };
    vi.stubGlobal("fetch", vi.fn().mockImplementation((input: string, init?: RequestInit) => {
      const path = new URL(input).pathname;
      if (path.endsWith("/inboxes/3")) return Promise.resolve(Response.json({ id: 3, channel_type: "Channel::Api" }));
      if (path.endsWith("/contacts/search")) return Promise.resolve(Response.json({ payload: [contact] }));
      if (path.endsWith("/contacts/7/conversations")) return Promise.resolve(Response.json({ payload: [conversation] }));
      if (path.endsWith("/conversations/11")) return Promise.resolve(Response.json(conversation));
      if (path.endsWith("/messages") && init?.method === "POST") return Promise.resolve(Response.json({ id: 22 }));
      if (path.endsWith("/messages")) return Promise.resolve(Response.json(messages));
      throw new Error(`Unexpected fixture path ${path}`);
    }));
  });
  afterEach(() => { vi.unstubAllEnvs(); vi.unstubAllGlobals(); vi.restoreAllMocks(); });
  it("returns only the current server identity, keeping provider credentials private", async () => {
    const response = await get("session");
    expect(await response.json()).toEqual({ subject }); expect(response.headers.get("cache-control")).toBe("no-store"); expect(fetch).not.toHaveBeenCalled();
  });
  it.each([null, { error: "expired" }, { issuerUrl: "https://old.test" }])("requires the current login", async value => {
    auth.session = value ? { ...auth.session, ...value } : null;
    expect((await get("conversations")).status).toBe(401); expect(fetch).not.toHaveBeenCalled();
  });
  it("lists only conversations with the current personal owner", async () => {
    const response = await get("conversations");
    expect(await response.json()).toEqual({ subject, conversations: [{ id: 11, status: "open", createdAt: 1 }] });
  });
  it("does not return a thread with another personal owner", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation((input, init) => String(input).endsWith("/conversations/11") ? Promise.resolve(Response.json({ ...conversation, custom_attributes: { platform_support_subject: "another-person" } })) : original(input, init));
    expect((await get("conversations/11/messages")).status).toBe(404);
  });
  it("returns normal transcript text without staff notes or provider metadata", async () => {
    const response = await get("conversations/11/messages");
    const result = await response.json();
    expect(result.messages.map((m: { content: string }) => m.content)).toEqual(["我的问题", "人工回复"]);
    expect(JSON.stringify(result)).not.toContain("内部笔记"); expect(JSON.stringify(result)).not.toContain("private-integration-token");
  });
  it("sends exactly one incoming message through the official API", async () => {
    expect((await send()).status).toBe(200);
    const calls = vi.mocked(fetch).mock.calls.filter(([, init]) => init?.method === "POST" && (init.body instanceof FormData));
    expect(calls).toHaveLength(1);
    const form = calls[0][1]!.body as FormData;
    expect(form.get("message_type")).toBe("incoming"); expect(form.get("private")).toBe("false"); expect(form.get("content")).toBe("请帮忙查看报错");
    expect(calls[0][1]?.headers).toMatchObject({ api_access_token: "private-integration-token" });
  });
  it("reports an unconfirmed send without retrying", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation((input, init) => init?.body instanceof FormData ? Promise.reject(new Error("offline")) : original(input, init));
    expect((await send()).status).toBe(502);
    expect(vi.mocked(fetch).mock.calls.filter(([, init]) => init?.body instanceof FormData)).toHaveLength(1);
  });
  it("leaves unavailable support unconfigured", async () => {
    vi.stubEnv("CHATWOOT_API_TOKEN", ""); expect((await get("session")).status).toBe(503);
  });
});
