import { createHash } from "node:crypto";
import { encode } from "@auth/core/jwt";
import { expect, test } from "@playwright/test";

const secret = "codex-chat-compose-synthetic-ui-only";
test.skip(process.env.CHAT_COMPOSE_UI_FIXTURE !== "1" || process.env.AUTH_SECRET !== secret, "requires the isolated synthetic UI fixture");

test("keyboard search in Chat, return from configuration, and original history without persisted results", async ({ page, context, baseURL }) => {
  test.setTimeout(60_000);
  const origin = baseURL!, id = "11111111-1111-4111-8111-111111111111";
  const namespace = createHash("sha256").update(JSON.stringify([origin, "https://chat-compose.invalid", "chat-compose-fixture"])).digest("hex").slice(0, 32);
  const name = "shuomi." + namespace + ".session-token";
  const token = await encode({ secret, salt: name, maxAge: 600, token: { accessToken: "synthetic-chat-search", expiresAt: Math.floor(Date.now() / 1000) + 600, identityVersion: 3, identity: { tenantId: "org", userId: "user", userType: "zitadel", roles: [] } } });
  await context.addCookies([{ name, value: token, url: origin, httpOnly: true, sameSite: "Lax" }]);

  const chat = "/workbench/ai/chat/" + id;
  const endpoint = "/api/model-route-pricing/search";
  const owner = { organizationId: "org", actorId: "user", memberId: "member" };
  const date = "2026-11-19T00:00:00Z";
  const preview = "a".repeat(64);
  const option = { name: "Brave 试用", available: true, source: "platform", preview, limit: 10 };
  let configured = false, writes = 0, keyWrites = 0, keyReceiptReads = 0, keyCommand = "", receipt: { key: string; query: string; status: string; source: string; startedAt: string; usage: number } | undefined;
  await page.route("**/*", async route => {
    const request = route.request(), url = new URL(request.url()), path = url.pathname;
    if (url.origin !== origin) return route.abort();
    if (!path.startsWith("/api/")) return route.continue();
    const json = (data: unknown, status = 200) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify(data) });
    if (path === "/api/workbench/context") return json({
      user: { id: "user" }, homeOrganizationId: "org", effectiveOrganizationId: "org", effectiveMemberId: "member",
      selectionRequired: false, aiWorkbenchAvailable: true, aiWorkbenchChatTurnsReady: true, aiWorkbenchPlanningReadiness: "AVAILABLE",
      organizations: [{ id: "org", name: "合成企业", roles: ["listingkit_admin"], permissions: ["workbench.chat.use", "workbench.chat.read", "workbench.agent.configure", "workbench.ai.channels.manage", "workbench.ai.credentials.manage"], capabilities: { "workbench.source_account.manage": false, "workbench.chat.use": true } }],
    });
    if (path === "/api/workbench/chat/conversations/" + id) return json({
      conversation: { ID: id, Scope: { OrganizationID: "org", ActorID: "user" }, Title: "公开资料核对", Favorite: false, Archived: false,
        MetadataRevision: 1, NextSequence: 1, CreatedAt: date, UpdatedAt: date, businessScope: { version: "workbench-scope-v1", kind: "general" } },
      messages: [], proposals: [], before: "",
    });
    if (path === endpoint + "/owned-key") {
      if (request.method() === "POST") {
        expect(request.postDataJSON()).toEqual({ revision: 0, apiKey: "browser-fixture-key", enabled: true, expectedMemberId: "member" });
        keyWrites++;
        keyCommand = request.headers()["idempotency-key"];
        return json({ code: "OUTCOME_UNKNOWN" }, 503);
      }
      return json({ ...owner, requestKey: "", result: { revision: keyWrites, enabled: Boolean(keyWrites), keyPresent: Boolean(keyWrites), updatedAt: keyWrites ? date : null } });
    }
    if (keyCommand && path === endpoint + "/owned-key/receipts/" + keyCommand) {
      expect(request.method()).toBe("GET");
      keyReceiptReads++;
      return json({ ...owner, requestKey: keyCommand, result: { revision: 1, enabled: true, keyPresent: true, updatedAt: date } });
    }
    if (path === endpoint + "/enterprise") {
      if (request.method() === "GET") return json({ ...owner, requestKey: "", result: { selection: { revision: configured ? 1 : 0, source: configured ? "platform" : "" }, options: [option] } });
      expect(request.postDataJSON()).toEqual({ source: "platform", revision: 0, expectedMemberId: "member" });
      configured = true;
      return json({ ...owner, requestKey: request.headers()["idempotency-key"], result: { revision: 1, source: "platform" } });
    }
    if (path === endpoint + "/chat/" + id) {
      if (request.method() === "GET") return json({ ...owner, conversationId: id, option: configured ? option : { name: "", available: false, source: "", limit: 0 }, result: receipt ? [receipt] : [] });
      const query = "public rule";
      expect(request.postDataJSON()).toEqual({ query, preview, expectedMemberId: "member" });
      writes++;
      receipt = { key: request.headers()["idempotency-key"], query, status: "complete", source: "platform", startedAt: date, usage: 1 };
      return json({ ...owner, conversationId: id, result: { receipt, results: [{ title: "<b>官方公开规则</b>", url: "https://example.org/rules", snippet: "只在当前页面显示的模拟摘要" }] } });
    }
    return json({ code: "NOT_FOUND" }, 404);
  });

  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(chat);
  await page.getByRole("button", { name: "查公开资料", exact: true }).click();
  await page.getByRole("link", { name: "查看模型与服务", exact: true }).last().click();
  await page.getByRole("combobox", { name: "选择搜索服务" }).selectOption("platform");
  await page.getByRole("button", { name: "保存选择", exact: true }).click();
  await expect(page.getByText("当前选用：Brave 试用")).toBeVisible();
  await page.getByLabel("Brave API Key").fill("browser-fixture-key");
  await page.getByRole("button", { name: "保存 Key", exact: true }).click();
  await expect(page.getByText("操作结果待核实，请查看原操作。")).toBeVisible();
  await expect(page.getByLabel("Brave API Key")).toHaveValue("");
  await page.reload();
  await expect(page.getByLabel("Brave API Key")).toHaveValue("");
  await page.getByRole("button", { name: "核对原操作", exact: true }).click();
  await expect(page.getByText("已保存，当前配置已重新读取。")).toBeVisible();
  await expect(page.getByText("Key 已保存，调用权限未验证")).toBeVisible();
  await expect(page.getByText(/^更新时间 /)).toBeVisible();
  await expect(page.getByRole("button", { name: "更换 Key", exact: true })).toBeVisible();
  expect(keyWrites).toBe(1);
  expect(keyReceiptReads).toBe(1);
  await page.getByRole("link", { name: "返回原业务页面", exact: true }).click();
  await expect(page).toHaveURL(new RegExp(chat + "$"));
  const open = page.getByRole("button", { name: "查公开资料", exact: true });
  await open.focus();
  await page.keyboard.press("Enter");
  const input = page.getByRole("textbox", { name: "公开搜索词" });
  await expect(input).toBeFocused();
  await input.fill("public rule");
  await input.press("Enter");
  await expect(page.getByRole("link", { name: "<b>官方公开规则</b>", exact: true })).toHaveAttribute("href", "https://example.org/rules");
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  expect(writes).toBe(1);
  await page.reload();
  await page.getByText("搜索记录", { exact: true }).click();
  await expect(page.getByText(/public rule · 已完成/)).toBeVisible();
  await expect(page.getByText("只在当前页面显示的模拟摘要")).toHaveCount(0);
  expect(writes).toBe(1);
});
