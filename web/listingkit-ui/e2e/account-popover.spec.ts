import { createHash } from "node:crypto";
import { encode } from "@auth/core/jwt";
import { expect, test } from "@playwright/test";

// Use the regular Next development server and Auth.js with a synthetic session.
// The browser only uses the loopback server and reads a fixture organization.
const secret = "shuomi-account-popover-synthetic-ui-only";
const issuer = "https://account-popover-ui.invalid";
const client = "account-popover-fixture";
const userId = "123456789";
const displayName = "陈小明";
test.skip(process.env.AUTH_SECRET !== secret, "Requires the explicit synthetic account popover environment.");

test.beforeEach(async ({ context, page, baseURL }) => {
  if (baseURL !== "http://127.0.0.1:3210") throw new Error("Loopback fixture required");
  const namespace = createHash("sha256").update(JSON.stringify([baseURL, issuer, client])).digest("hex").slice(0, 32);
  const name = `shuomi.${namespace}.session-token`;
  const value = await encode({ secret, salt: name, maxAge: 600, token: {
    accessToken: "synthetic-access", expiresAt: Math.floor(Date.now() / 1000) + 600, identityVersion: 3,
    identity: { tenantId: "org", userId, userType: "zitadel", roles: [] },
  } });
  await context.addCookies([{ name, value, url: baseURL, httpOnly: true, sameSite: "Lax" }]);
  await page.route("**/*", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (url.origin !== baseURL || request.method() !== "GET") return route.abort();
    if (url.pathname === "/api/workbench/context") return route.fulfill({ json: {
      user: { id: userId }, homeOrganizationId: "org", effectiveOrganizationId: "org", effectiveMemberId: "member", selectionRequired: false,
      organizations: [{ id: "org", name: "测试企业", roles: [], permissions: ["workbench.store.read"] }],
    } });
    if (url.pathname === "/api/account/profile" && request.headers()["x-expected-user-id"] === userId) return route.fulfill({ json: {
      schemaVersion: "account-v1", userId, homeOrganizationId: "org", displayName,
      email: null, emailVerified: null, phoneNumber: null, phoneNumberVerified: null,
      readAt: "2026-11-07T00:00:00Z", source: "zitadel_userinfo",
    } });
    if (url.pathname === "/api/notifications/identity") return route.fulfill({ json: { userId } });
    if (/^\/api\/notifications\/(official|business)$/.test(url.pathname)) return route.fulfill({ json: {
      schemaVersion: "notification-center-v1", items: [], coverage: [], exact: true, count: 0, unread: 0, pending: 0, next: "",
    } });
    if (url.pathname === "/api/workbench/store-center/catalog") return route.fulfill({ json: {
      schemaVersion: "store-center-v1", organizationId: "org", userId, data: {
        items: [], pagination: { page: 1, pageSize: 20, total: 0 },
        metrics: { total: 0, normal: 0, attention: 0, syncHealthy: 0, unknown: 0, complete: true, observedAt: "2026-11-07T00:00:00Z" },
        facets: { regions: [], applicationTypes: [], groups: [], canManageGroups: true },
      },
    } });
    if (url.pathname === "/api/workbench/store-center/groups") return route.fulfill({ json: {
      schemaVersion: "store-center-v1", organizationId: "org", userId, data: { items: [] },
    } });
    if (url.pathname.startsWith("/api/")) return route.fulfill({ status: 404, json: { code: "NOT_FOUND" } });
    return route.continue();
  });
});

for (const width of [1440, 390]) {
  test(`account popover closes with Escape and follows the workbench theme at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 844 });
    await page.goto("/workbench");

    const trigger = page.getByRole("button", { name: "我的账户", exact: true });
    const panel = page.getByRole("dialog", { name: "账户快捷操作" });
    const profile = page.getByRole("link", { name: "账户资料", exact: true });
    const mobileNav = page.getByRole("navigation", { name: "移动工作台导航" });
    if (width === 390) await page.getByRole("button", { name: "打开工作台导航" }).click();

    await trigger.press("Enter");
    await expect(panel).toBeFocused();
    await expect(panel.getByText(displayName, { exact: true })).toBeVisible();
    await expect(panel.getByText(userId, { exact: true })).toHaveCount(0);
    await page.keyboard.press("Tab");
    await expect(profile).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(profile).toBeHidden();
    await expect(trigger).toBeFocused();
    if (width === 390) {
      await expect(mobileNav).toBeVisible();
      await page.getByRole("button", { name: "关闭工作台导航" }).click();
    }

    await trigger.press("Enter");
    await expect(panel).toBeVisible();
    await expect(profile).toHaveAttribute("href", "/workbench/account/profile");
    await expect(panel.getByRole("link", { name: "退出登录" })).toHaveAttribute("href", "/api/zitadel-auth/logout");
    const darkSurface = await panel.evaluate((node) => getComputedStyle(node).backgroundColor);

    const theme = page.getByRole("switch", { name: "浅色模式" });
    await theme.click();
    await expect(panel).toBeHidden();
    await expect(theme).toBeFocused();
    await expect(theme).toBeChecked();

    await page.reload();
    await expect(theme).toBeChecked();
    await trigger.press("Enter");
    await expect(panel).toBeVisible();
    expect(await panel.evaluate((node) => getComputedStyle(node).backgroundColor)).not.toBe(darkSurface);
    await expect.poll(async () => panel.evaluate((node) => {
      const { left, right, top, bottom } = node.getBoundingClientRect();
      return left >= 0 && right <= innerWidth && top >= 0 && bottom <= innerHeight;
    })).toBe(true);

    await profile.click();
    await expect(page).toHaveURL(/\/workbench\/account\/profile$/);
    await expect(panel).toBeHidden();
  });
}

test.describe("workbench notification and store overlays", () => {
  test.skip(process.env.LISTINGKIT_NOTIFICATION_CENTER_ENABLED !== "true" || process.env.CHATWOOT_API_TOKEN !== "overlay-synthetic-only", "Requires the loopback notification and support fixture settings.");

  for (const width of [1440, 390]) {
    for (const theme of ["dark", "light"]) {
      test(`${width}px ${theme}: keyboard and focus restoration`, async ({ page }) => {
        await page.setViewportSize({ width, height: 844 });
        await page.addInitScript((value) => localStorage.setItem("listingkit-theme", value), theme);
        await page.goto("/workbench/stores");
        await expect(page.locator("html")).toHaveClass(new RegExp(`\\b${theme}\\b`));
        await expect(page.getByRole("button", { name: "联系客服", exact: true })).toBeEnabled();

        const notice = page.getByRole("button", { name: "通知", exact: true });
        const panel = page.getByRole("region", { name: "通知面板", exact: true });
        const close = panel.getByRole("button", { name: "关闭通知面板" });
        const official = panel.getByRole("tab", { name: "硕米官方通知" });
        const business = panel.getByRole("tab", { name: "商家经营触发通知" });
        await notice.press("Enter");
        await expect(official).toBeVisible();
        await expect(close).toBeFocused();
        const surface = theme === "dark" ? "rgb(20, 34, 55)" : "rgb(255, 255, 255)";
        await expect(panel).toHaveCSS("background-color", surface);
        await page.keyboard.press("Tab");
        await expect(official).toBeFocused();
        await page.keyboard.press("ArrowRight");
        await expect.soft(business).toBeFocused({ timeout: 1500 });
        await expect.soft(business).toHaveAttribute("aria-selected", "true", { timeout: 1500 });
        await expect(panel.getByRole("tabpanel", { name: "商家经营触发通知" })).toBeVisible();
        await expect.poll(async () => panel.evaluate((node) => {
          const { left, right } = node.getBoundingClientRect();
          return left >= 0 && right <= document.documentElement.clientWidth + 1;
        })).toBe(true);
        await page.keyboard.press("ArrowLeft");
        await expect(official).toBeFocused();
        await page.keyboard.press("Escape");
        await expect(panel).toBeHidden();
        await expect.soft(notice).toBeFocused({ timeout: 1500 });

        const groups = page.getByRole("button", { name: "分组管理", exact: true });
        const dialog = page.getByRole("dialog", { name: "分组管理", exact: true });
        await expect(groups).toBeEnabled();
        await groups.press("Enter");
        await expect(dialog).toHaveCSS("background-color", surface);
        await expect(dialog.getByRole("button", { name: "关闭" })).toBeFocused();
        await page.keyboard.press("Tab");
        await expect(dialog.getByRole("textbox", { name: "新建分组" })).toBeFocused();
        await page.keyboard.press("Escape");
        await expect(dialog).toBeHidden();
        await expect(groups).toBeFocused();
        await groups.press("Enter");
        await dialog.getByRole("button", { name: "关闭" }).click();
        await expect(dialog).toBeHidden();
        await expect(groups).toBeFocused();
      });
    }
  }
});
