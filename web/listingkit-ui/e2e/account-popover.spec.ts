import { createHash } from "node:crypto";
import { encode } from "@auth/core/jwt";
import { expect, test } from "@playwright/test";

// Use the regular Next development server and Auth.js with a synthetic session.
// The browser only uses the loopback server and reads a fixture organization.
const secret = "shuomi-account-popover-synthetic-ui-only";
const issuer = "https://account-popover-ui.invalid";
const client = "account-popover-fixture";
test.skip(process.env.AUTH_SECRET !== secret, "Requires the explicit synthetic account popover environment.");

test.beforeEach(async ({ context, page, baseURL }) => {
  if (baseURL !== "http://127.0.0.1:3210") throw new Error("Loopback fixture required");
  const namespace = createHash("sha256").update(JSON.stringify([baseURL, issuer, client])).digest("hex").slice(0, 32);
  const name = `shuomi.${namespace}.session-token`;
  const value = await encode({ secret, salt: name, maxAge: 600, token: {
    accessToken: "synthetic-access", expiresAt: Math.floor(Date.now() / 1000) + 600, identityVersion: 3,
    identity: { tenantId: "org", userId: "operator", userType: "zitadel", roles: [] },
  } });
  await context.addCookies([{ name, value, url: baseURL, httpOnly: true, sameSite: "Lax" }]);
  await page.route("**/*", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (url.origin !== baseURL || request.method() !== "GET") return route.abort();
    if (url.pathname === "/api/workbench/context") return route.fulfill({ json: {
      user: { id: "operator" }, homeOrganizationId: "org", effectiveOrganizationId: "org", effectiveMemberId: "member", selectionRequired: false,
      organizations: [{ id: "org", name: "测试企业", roles: [], permissions: [] }],
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
