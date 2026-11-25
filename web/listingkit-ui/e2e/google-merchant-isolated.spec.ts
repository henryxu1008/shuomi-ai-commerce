import { createHash } from "node:crypto";
import { encode } from "@auth/core/jwt";
import { expect, test } from "@playwright/test";

const secret="isolated-google-workbench-browser-test";
const issuer="https://google-workbench-test.invalid";
const client="google-workbench-test";
const storeID = process.env.GOOGLE_TEST_STORE!;
const sourceID = process.env.GOOGLE_TEST_SOURCE!;

test.skip(process.env.GOOGLE_WORKBENCH_BROWSER !== "1" || process.env.AUTH_SECRET !== secret ||
  !storeID || !sourceID, "requires the isolated Go Google fixture");

test("authorizes Google, uploads a product and finds the saved submission", async ({ page, context, baseURL }) => {
  test.setTimeout(75_000);
  page.setDefaultTimeout(12_000);
  const origin = baseURL!;
  const namespace = createHash("sha256").update(JSON.stringify([origin, issuer, client])).digest("hex").slice(0, 32);
  const name = "shuomi." + namespace + ".session-token";
  const value = await encode({
    secret, salt: name, maxAge: 600,
    token: {
      accessToken: "fixture-google-access", expiresAt: Math.floor(Date.now() / 1000) + 600,
      identityVersion: 3, identity: { tenantId: "org-a", userId: "actor-a", userType: "zitadel", roles: [] },
    },
  });
  await context.addCookies([
    { name, value, url: origin, httpOnly: true, sameSite: "Lax" },
    { name: "shuomi_effective_organization", value: "org-a", url: origin, httpOnly: true, sameSite: "Lax" },
  ]);

  const permissions = ["workbench.store.read", "workbench.store.update", "workbench.supply.read",
    "workbench.supply.manage", "workbench.supply.submit", "workbench.collection.read"];
  await page.route("**/*", async route => {
    const url = new URL(route.request().url());
    if (url.origin === "https://accounts.google.com") {
      const callback = new URL("/workbench/stores/googlemerchant/callback", origin);
      callback.searchParams.set("code", "browser-consent-" + url.searchParams.get("code_challenge"));
      callback.searchParams.set("state", url.searchParams.get("state")!);
      return route.fulfill({ status: 302, headers: { location: callback.href } });
    }
    if (url.origin !== origin) return route.abort();
    if (url.pathname === "/api/workbench/context") {
      return route.fulfill({
        json: {
          user: { id: "actor-a" }, homeOrganizationId: "org-a", effectiveOrganizationId: "org-a",
          effectiveMemberId: "member-a", selectionRequired: false,
          organizations: [{ id: "org-a", name: "测试企业", roles: [], permissions }],
        },
      });
    }
    return route.continue();
  });

  await page.goto("/workbench/stores/" + storeID + "#authorization");
  await page.getByRole("button", { name: "连接 Google", exact: true }).click();
  await expect(page.getByRole("heading", { name: "连接 Google Merchant Center" })).toBeVisible();
  await page.getByRole("button", { name: "读取 Google 账号" }).click();
  await page.getByLabel("Merchant Center 账号").selectOption("123");
  await page.getByRole("button", { name: "查看商品数据源" }).click();
  await page.getByLabel("商品数据源").selectOption("456");
  await page.getByText("此数据源专供硕米上传商品。").click();
  await page.getByRole("button", { name: "确认连接" }).click();
  await expect(page.getByText("Google 店铺已连接。")).toBeVisible();

  const productPath = "/workbench/listing/google/" + sourceID;
  await page.goto(productPath);
  await page.getByLabel("目标店铺").selectOption(storeID);
  await page.getByLabel("商品规格").selectOption("default");
  await page.getByLabel("已审核的商品主图").selectOption("approved-main");
  await page.getByLabel("买家可购买该商品的页面").fill("https://seller.example.org/desk");
  await page.getByLabel("销售价").fill("19.95");
  await page.getByLabel("币种").fill("USD");
  await page.getByRole("button", { name: "预览上传" }).click();
  const preview = page.getByRole("region", { name: "待确认商品" });
  await expect(preview.getByRole("heading", { name: "核对即将提交的商品" })).toBeVisible();
  await expect(preview.getByText("123", { exact: true })).toBeVisible();
  await expect(preview.getByText("456", { exact: true })).toBeVisible();
  await expect(preview.getByText("en / US", { exact: true })).toBeVisible();
  const confirmation = page.getByRole("button", { name: "确认上传到 Google" });
  const refusal = page.waitForResponse(response => response.url().endsWith("/google/confirm"));
  await confirmation.click();
  const blocked = await (await refusal).json();
  expect(blocked.confirmationBlocked).toBe("retry");
  await expect(page.getByRole("alert").filter({ hasText: "暂时无法连接 Google" })).toBeVisible();
  await expect(preview.getByText("456", { exact: true })).toBeVisible();
  // The isolated OAuth fixture returned a one-second Retry-After.
  await new Promise(resolve => setTimeout(resolve, 1100));
  const accepted = page.waitForResponse(response => response.url().endsWith("/google/confirm"));
  await confirmation.click();
  const saved = await (await accepted).json();
  expect(saved.id).toBe(blocked.id);
  await expect(page.getByText("Google 已收到商品")).toBeVisible();

  await page.goto(productPath);
  const history = page.getByRole("region", { name: "最近的 Google 上传记录" });
  await history.getByRole("link", { name: /的上传记录/ }).click();
  await page.getByRole("button", { name: "查询提交结果" }).click();
  await expect(page.getByText("符合免费展示条件：US")).toBeVisible();
});
