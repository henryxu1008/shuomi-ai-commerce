import { createHash } from "node:crypto";
import { encode } from "@auth/core/jwt";
import { expect, test } from "@playwright/test";
import { initialListingDraft } from "../src/components/workbench/listing/draft-fields";
import { rulesSchema, sheinTargetRecordSchema, materialsRunSchema, type SupplySourceDetail } from "../src/lib/contracts/supply-chain";
import { catalogEntrySchema, materialsTemplatesPageSchema } from "../src/lib/contracts/agent-configuration";
import { readMaterialsIntent, readMaterialsMutation } from "../src/components/workbench/listing/materials-intent";
import { executionModeViewSchema } from "../src/lib/contracts/execution-mode";

// Use the regular Next development server and Auth.js with a synthetic session.
// The browser only uses the loopback server and reads a fixture organization.
const secret = "shuomi-account-popover-synthetic-ui-only";
const issuer = "https://account-popover-ui.invalid";
const client = "account-popover-fixture";
const userId = "123456789";
const displayName = "陈小明";
test.skip(process.env.AUTH_SECRET !== secret, "Requires the explicit synthetic account popover environment.");

function materialsFixture() {
  const id = "550e8400-e29b-41d4-a716-446655440000";
  const storeId = "550e8400-e29b-41d4-a716-446655440002";
  const at = "2026-11-09T00:00:00Z";
  // Reuse the existing materials unit fixture's draft, scope and candidate categories.
  const draft = initialListingDraft({ product: { title: "【合成验收】平台资料商品" } } as SupplySourceDetail);
  const merchant = { organization_id: "org", store_id: storeId, site: "shein-us", store_version: 1, connection_revision: 1, application_revision: "1", application_id: "synthetic-app", application_type: "self_operated", supplier_identity_hash: "a".repeat(64), service_expires_at: "2027-01-01T00:00:00Z" };
  const record = sheinTargetRecordSchema.parse({ id, targetId: id, revision: 1, effectiveVersion: "1", source: { id, preparationId: id, collectionItemId: id, collectionRevision: 1, source: { kind: "own", productKey: "own-materials-synthetic", publicationId: id, version: "1" } }, input: { sourceId: id, storeId, expectedRevision: 0, effectiveVersion: "1", draft }, merchant, productHash: "a".repeat(64), inventoryHash: "b".repeat(64), rulesHash: "c".repeat(64), result: { product: draft.product, images: [], ready_for_upload: false, issues: [] }, createdAt: at });
  const rules = rulesSchema.parse({ merchant, rules: { application_type: "self_operated", categories: [{ category_id: 123, product_type_id: 456, parent_category_id: 0, category_name: "【合成】候选类目A", last_category: true, children: [] }, { category_id: 124, product_type_id: 457, parent_category_id: 0, category_name: "【合成】候选类目B", last_category: true, children: [] }], brands: [], sites: [], warehouses: [], fill: { fill_in_standard_list: [], currency: "CNY", default_language: "en", default_language_title_max_length: null, language_title_max_length_list: [], picture_config_list: [], supplier_code_in_spu_dimension: false, support_duplicate_sale_attr_value: false }, attributes: { product_type_id: 0, main_attribute_status: null, attribute_infos: [] }, linked: [] } });
  const config = catalogEntrySchema.parse({ agent: { agentId: "listing.materials.agent", activation: "ENABLED", revision: "1", activationEpoch: "1", defaultTemplate: null, updatedAt: at }, name: "【合成】平台资料智能体", description: "仅用于隔离UI验证", definitionVersion: "v1", parameterSchema: "listing-materials-config-v1", canConfigure: false, canUse: true, canReadRuns: true, capabilities: [{ id: "listing.materials.inspect", support: "REQUIRED", readiness: "AVAILABLE", reason: "合成前置已就绪", observedAt: at }] });
  const templates = materialsTemplatesPageSchema.parse({ items: [1, 2].map(n => ({ templateId: `550e8400-e29b-41d4-a716-44665544000${n + 2}`, agentId: "listing.materials.agent", lifecycle: "ACTIVE", revision: String(n), version: String(n), schemaVersion: "listing-materials-config-v1", name: `【合成】平台模板${n}`, targetPlatform: "shein", createdAt: at })), nextCursor: "" });
  const mode = executionModeViewSchema.parse({ subject: { kind: "registered_agent", id: "listing.materials.agent" }, definition: { id: "listing.materials.agent", version: "v1" }, mode: "default", revision: "1", policyVersion: "agent-execution-policy-v1", canConfigure: false, updatedAt: at, updatedBy: "synthetic" });
  return { id, storeId, record, rules, config, templates, mode };
}

for (const width of [1440, 390]) {
  for (const theme of ["dark", "light"]) {
    test(`materials selections and confirmation at ${width}px in ${theme}`, async ({ page }, info) => {
      test.skip(process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED !== "true" || process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED !== "true", "Requires isolated materials features.");
      const { id, storeId, record, rules, config, templates, mode } = materialsFixture();
      const forbidden: string[] = [];
      let categoryReads = 0;
      await page.route("**/api/workbench/**", async route => {
        const request = route.request(), path = new URL(request.url()).pathname;
        if (path === "/api/workbench/supply-preparations/target-rules" && request.method() === "POST") {
          const body = request.postDataJSON();
          expect(body.sourceId).toBe(id); expect(body.storeId).toBe(storeId);
          categoryReads++;
          return route.fulfill({ json: rules });
        }
        if (request.method() !== "GET") { forbidden.push(`${request.method()} ${path}`); return route.abort(); }
        if (path === "/api/workbench/context") return route.fulfill({ json: { user: { id: userId }, homeOrganizationId: "org", effectiveOrganizationId: "org", effectiveMemberId: "member", selectionRequired: false, organizations: [{ id: "org", name: "【合成】平台资料测试企业", roles: [], permissions: ["workbench.supply.read", "workbench.agent.use", "local_agent.write"] }] } });
        if (path === `/api/workbench/supply-preparations/records/${id}`) return route.fulfill({ json: record });
        if (path === "/api/workbench/agents/listing.materials.agent") return route.fulfill({ json: config });
        if (path === "/api/workbench/agents/listing.materials.agent/templates") return route.fulfill({ json: templates });
        if (path === "/api/workbench/agents/execution-modes/registered_agent/listing.materials.agent") return route.fulfill({ json: mode });
        return route.fallback();
      });
      await page.setViewportSize({ width, height: 844 });
      await page.goto(`/workbench/listing/materials/${id}`);
      await expect(page.getByRole("heading", { name: "平台上架资料", exact: true })).toBeVisible();
      const preference = page.getByRole("combobox", { name: "显示主题", exact: true });
      await preference.selectOption(theme);
      const purpose = page.getByRole("combobox", { name: "本次用途", exact: true });
      const template = page.getByRole("combobox", { name: "本次模板", exact: true });
      await expect(page.getByRole("button", { name: "准备平台资料", exact: true })).toBeEnabled();
      await purpose.press("ArrowDown");
      await expect(purpose).toHaveValue("category-rules");
      const category = page.getByRole("combobox", { name: "候选类目", exact: true });
      await expect(category.getByRole("option", { name: "【合成】候选类目A" })).toHaveCount(1);
      await purpose.press("Tab"); await expect(category).toBeFocused();
      await category.press("ArrowDown"); await expect(category).toHaveValue("123");
      await category.press("End"); await expect(category).toHaveValue("124");
      await category.press("Tab"); await expect(template).toBeFocused();
      await template.press("ArrowDown"); await expect(template).toHaveValue(templates.items[0].templateId);
      await template.press("End"); await expect(template).toHaveValue(templates.items[1].templateId);
      await preference.selectOption(theme === "dark" ? "light" : "dark");
      await expect(purpose).toHaveValue("category-rules"); await expect(category).toHaveValue("124"); await expect(template).toHaveValue(templates.items[1].templateId);
      await preference.selectOption(theme);
      const selections = await page.getByRole("main").getByRole("combobox").evaluateAll(nodes => nodes.map(node => (node as HTMLSelectElement).value));
      expect(selections).toEqual(["category-rules", "124", templates.items[1].templateId]);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
      await page.screenshot({ path: info.outputPath(`materials-${width}-${theme}.png`) });
      for (const candidate of [true, false]) {
        if (!candidate) await purpose.selectOption("materials-edit");
        const trigger = page.getByRole("button", { name: candidate ? "查询候选类目规则" : "准备平台资料", exact: true });
        const dialog = page.getByRole("dialog", { name: candidate ? "确认查询候选类目规则" : "确认准备平台资料", exact: true });
        await trigger.press("Enter"); await expect(dialog).toBeVisible();
        const cancel = dialog.getByRole("button", { name: "取消", exact: true });
        await expect(cancel).toBeFocused(); await expect(dialog).toContainText("【合成】平台模板2 · v2");
        await expect(dialog).toHaveCSS("opacity", "1");
        await expect(dialog).toHaveCSS("background-color", theme === "dark" ? "rgb(20, 34, 55)" : "rgb(255, 255, 255)");
        if (candidate) await expect(dialog).toContainText("候选类目 【合成】候选类目B · 124");
        expect(await dialog.evaluate(node => { const r = node.getBoundingClientRect(); return r.left >= 0 && r.right <= document.documentElement.clientWidth && r.top >= 0 && r.bottom <= innerHeight; })).toBe(true);
        await page.screenshot({ path: info.outputPath(`materials-confirm-${candidate ? "rules" : "prepare"}-${width}-${theme}.png`) });
        await cancel.press("Tab");
        const confirm = dialog.getByRole("button", { name: candidate ? "确认查询并解释" : "确认生成建议", exact: true });
        await expect(confirm).toBeFocused();
        await confirm.press("Shift+Tab"); await expect(cancel).toBeFocused();
        await cancel.press("Escape"); await expect(dialog).toBeHidden(); await expect(trigger).toBeFocused();
        await trigger.press("Enter"); await cancel.press("Enter"); await expect(dialog).toBeHidden(); await expect(trigger).toBeFocused();
        await expect(template).toHaveValue(templates.items[1].templateId);
      }
      await purpose.selectOption("category-rules"); await expect(category).toHaveValue("124");
      expect(categoryReads).toBeGreaterThan(0); expect(forbidden).toEqual([]);
    });
  }
}

for (const width of [1440, 390]) {
  for (const theme of ["dark", "light"]) {
    test(`materials retained states at ${width}px in ${theme}`, async ({ page, baseURL }, info) => {
      test.skip(process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED !== "true" || process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED !== "true", "Requires isolated materials features.");
      const { id, record, config, templates, mode } = materialsFixture();
      const key = "550e8400-e29b-41d4-a716-446655440001";
      const configurationSnapshotRef = { Kind: "agent-configuration-v1", ID: key, Digest: "a".repeat(64) };
      const modeSnapshotRef = { Kind: "agent-execution-mode-v1", ID: key, Digest: "b".repeat(64) };
      const feedbackText = "【合成】保留原资料\n只补充类目核对意见";
      const intent = { recordId: id, userId, organizationId: "org", key, prepare: { requestKey: key, expectedModeRevision: "1" }, prepared: { recordId: id, requestKey: key, configurationSnapshotRef, modeSnapshotRef, modeRevision: "1", templateSelection: null } };
      const mutation = { recordId: id, userId, organizationId: "org", key, action: "resume", input: { configurationSnapshotRef, revision: "3", feedback: feedbackText } };
      expect(readMaterialsIntent(JSON.stringify(intent))).not.toBeNull();
      expect(readMaterialsMutation(JSON.stringify(mutation))).not.toBeNull();
      const common = { runId: key, requestKey: key, recordId: id, revision: "3", configurationSnapshotRef, modeSnapshotRef, executionMode: "default", modeRevision: "1", before: record.input, unresolved: ["【合成】类目仍需本人核对"], canRequestAdoption: false, steps: [], tokens: 30, estimatedCostMicros: 20, currency: "CNY", usageStatus: "observed" };
      const interrupted = materialsRunSchema.parse({ ...common, phase: "interrupted", canResume: true });
      const unknown = materialsRunSchema.parse({ ...interrupted, usageStatus: "unknown_reserved" });
      const readOnly = materialsRunSchema.parse({ ...common, phase: "human_review_required", executionMode: "read_only", canResume: false, after: { ...record.input, draft: { ...record.input.draft, product: { ...record.input.draft.product, brand_code: "synthetic-brand" } } }, proposal: { brandCode: "synthetic-brand", evidenceIds: ["synthetic-observed"] }, candidateHash: "c".repeat(64) });
      let scenario = "interrupted", unknownReadFailure = false;
      const writes: string[] = [], runReads: string[] = [];
      await page.route("**/*", async route => {
        const request = route.request(), url = new URL(request.url()), path = url.pathname;
        if (url.origin !== baseURL || request.method() !== "GET") { writes.push(`${request.method()} ${url.origin}${path}`); return route.abort(); }
        if (path === "/api/workbench/context") return route.fulfill({ json: { user: { id: userId }, homeOrganizationId: "org", effectiveOrganizationId: "org", effectiveMemberId: "member", selectionRequired: false, organizations: [{ id: "org", name: "【合成】平台资料测试企业", roles: [], permissions: ["workbench.supply.read", "workbench.agent.use", "local_agent.write"] }] } });
        if (path === `/api/workbench/supply-preparations/records/${id}`) return route.fulfill({ json: record });
        if (path.includes("/materials-agent/")) {
          expect(path).toBe(`/api/workbench/supply-preparations/records/${id}/materials-agent/runs/${key}`);
          expect(request.headers()["x-expected-user-id"]).toBe(userId);
          expect(request.headers()["x-expected-organization-id"]).toBe("org");
          runReads.push(path);
          if (scenario === "unknown" && unknownReadFailure) return route.fulfill({ status: 409, json: { code: "OUTCOME_UNKNOWN" } });
          return route.fulfill({ json: scenario === "read-only" ? readOnly : scenario === "unknown" ? unknown : interrupted });
        }
        if (path === "/api/workbench/agents/listing.materials.agent") {
          if (scenario === "permission") return route.fulfill({ status: 403, json: { code: "FORBIDDEN" } });
          const value = scenario === "dependency" ? catalogEntrySchema.parse({ ...config, capabilities: [{ ...config.capabilities[0], readiness: "NEEDS_CONFIGURATION", reason: "【合成】模型配置尚未就绪" }] }) : config;
          return route.fulfill({ json: value });
        }
        if (path === "/api/workbench/agents/listing.materials.agent/templates") return route.fulfill({ json: templates });
        if (path === "/api/workbench/agents/execution-modes/registered_agent/listing.materials.agent") return route.fulfill({ json: executionModeViewSchema.parse({ ...mode, mode: scenario === "read-only" ? "read_only" : "default" }) });
        return route.fallback();
      });
      await page.setViewportSize({ width, height: 844 });
      for (scenario of ["interrupted", "unknown", "read-only", "permission", "dependency"]) {
        const params = new URLSearchParams();
        if (["interrupted", "unknown", "read-only"].includes(scenario)) {
          params.set("agent_key", key); params.set("agent_actor", userId); params.set("agent_org", "org");
          if (scenario === "unknown") { params.set("materials_intent", JSON.stringify(intent)); params.set("materials_mutation", JSON.stringify(mutation)); }
        }
        await page.goto(`/workbench/listing/materials/${id}?${params}`);
        await expect(page.getByRole("heading", { name: "平台上架资料", exact: true })).toBeVisible();
        const preference = page.getByRole("combobox", { name: "显示主题", exact: true });
        await preference.selectOption(theme);
        const feedback = page.getByRole("textbox", { name: "补充核对意见", exact: true });
        if (scenario === "interrupted") {
          await expect(page.getByRole("heading", { name: "等待继续", exact: true })).toBeVisible();
          const resume = page.getByRole("button", { name: "按原编号继续", exact: true });
          await expect(resume).toBeDisabled();
          await page.getByRole("button", { name: "核对原运行", exact: true }).focus();
          await page.keyboard.press("Tab"); await expect(page.getByRole("button", { name: "核对原保存结果", exact: true })).toBeFocused();
          await page.keyboard.press("Tab"); await expect(feedback).toBeFocused();
          await page.keyboard.type("【合成】保留原资料"); await page.keyboard.press("Enter"); await page.keyboard.type("只补充类目核对意见");
          await expect(feedback).toHaveValue(feedbackText); await expect(resume).toBeEnabled();
          await page.keyboard.press("Tab"); await expect(resume).toBeFocused();
          await page.keyboard.press("Shift+Tab"); await expect(feedback).toBeFocused();
        } else if (scenario === "unknown") {
          await expect(feedback).toBeDisabled(); await expect(feedback).toHaveValue(feedbackText);
          await expect(page.getByText("用量待核实，原预留仍保留", { exact: false })).toBeVisible();
          // Development Strict Mode may restore twice; fail only the explicit read below.
          unknownReadFailure = true;
          await page.getByRole("button", { name: "核对原运行", exact: true }).press("Enter");
          await expect(page.getByRole("main").getByRole("alert")).toHaveText("结果尚未确认，请保留本次编号并核对原请求。");
          await expect(page.getByRole("button", { name: "重试原继续请求", exact: true })).toBeEnabled();
          await expect(page.getByRole("button", { name: "重试原确认与请求", exact: true })).toHaveCount(0);
        } else if (scenario === "read-only") {
          await expect(page.getByRole("heading", { name: "建议差异 · 尚未保存", exact: true })).toBeVisible();
          await expect(page.getByText("本次为只读模式，建议可查看，不能保存到平台草稿。", { exact: true })).toBeVisible();
          await expect(page.getByText("synthetic-brand", { exact: true })).toBeVisible();
          await expect(page.getByRole("checkbox")).toHaveCount(0); await expect(feedback).toHaveCount(0);
        } else {
          await expect(page.getByText(scenario === "permission" ? "当前身份没有操作权限。" : "【合成】模型配置尚未就绪", { exact: true })).toBeVisible();
          await expect(page.getByRole("button", { name: "准备平台资料", exact: true })).toBeDisabled();
        }
        for (const value of [theme === "dark" ? "light" : "dark", theme]) {
          await preference.selectOption(value);
          await expect(page.locator("html")).toHaveClass(new RegExp(`\\b${value}\\b`));
          if (scenario === "interrupted" || scenario === "unknown") await expect(feedback).toHaveValue(feedbackText);
        }
        if (params.has("agent_key")) {
          await expect(page.getByText(`本次编号 ${key}`, { exact: true })).toBeVisible();
          await expect(page.getByRole("button", { name: "准备平台资料", exact: true })).toHaveCount(0);
          expect(new URL(page.url()).searchParams.get("agent_key")).toBe(key);
          if (scenario === "unknown") expect(JSON.parse(new URL(page.url()).searchParams.get("materials_mutation")!)).toEqual(mutation);
        }
        await expect(page.getByRole("button", { name: "确认保存本地草稿", exact: true })).toHaveCount(0);
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
        await page.screenshot({ path: info.outputPath(`materials-state-${scenario}-${width}-${theme}.png`), fullPage: true });
      }
      expect(runReads.length).toBeGreaterThanOrEqual(4); expect(writes).toEqual([]);
    });
  }
}

for (const width of [1440, 390]) {
  for (const theme of ["light", "dark"]) {
    test(`own-product actions and unsaved return at ${width}px in ${theme}`, async ({ page }, info) => {
      test.skip(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED !== "true" || process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED !== "true" || process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED !== "true", "Requires the explicit isolated own-product UI features.");
      const itemId = "550e8400-e29b-41d4-a716-446655440002";
      const title = "【合成回归】三规格商品";
      await page.setViewportSize({ width, height: 844 });
      await page.route("**/api/workbench/context", route => route.fulfill({ json: {
        user: { id: userId }, homeOrganizationId: "org", effectiveOrganizationId: "org", effectiveMemberId: "member", selectionRequired: false,
        organizations: [{ id: "org", name: "测试企业", roles: [], permissions: ["workbench.collection.read", "workbench.collection.manage", "local_agent.write", "workbench.agent.use", "workbench.supply.read", "workbench.supply.manage"] }],
      } }));
      await page.route(`**/api/workbench/collections/items/${itemId}`, route => route.fulfill({ json: {
        item: { id: itemId, batchId: itemId, revision: 1, createdAt: "2026-11-08T00:00:00Z", source: { kind: "own", productKey: "own-responsive-fixture", publicationId: itemId, version: "1" } },
        product: { title, description: "只用于共享页面操作区布局回归。", variants: [
          { source_id: "black-m", title: "黑 M", sku: "BLK-M", attributes: [{ name: "颜色", value: "黑" }, { name: "尺码", value: "M" }], price: { currency: "CNY", amount: 29 }, stock: 0, own_stock_declared: true },
          { source_id: "black-l", title: "黑 L", sku: "BLK-L", attributes: [{ name: "颜色", value: "黑" }, { name: "尺码", value: "L" }], price: { currency: "CNY", amount: 29 } },
          { source_id: "white-m", title: "白 M", sku: "WHT-M", attributes: [{ name: "颜色", value: "白" }, { name: "尺码", value: "M" }], price: { currency: "CNY", amount: 29 } },
        ] },
      } }));
      await page.goto(`/workbench/data/mine/own/${itemId}`);
      await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible();
      await page.getByRole("combobox", { name: "显示主题", exact: true }).selectOption(theme);
      const actions = page.getByRole("group", { name: "页面操作", exact: true });
      await expect(actions.getByRole("link")).toHaveCount(5);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
      for (const action of await actions.getByRole("link").all()) {
        expect(await action.evaluate(node => {
          const rect = node.getBoundingClientRect();
          return rect.left >= 0 && rect.right <= document.documentElement.clientWidth;
        })).toBe(true);
      }
      await page.screenshot({ path: info.outputPath(`own-product-actions-${width}-${theme}.png`) });
      const nativeDialogs: string[] = [];
      const writes: string[] = [];
      page.on("dialog", async dialog => { nativeDialogs.push(dialog.message()); await dialog.dismiss(); });
      page.on("request", request => { if (request.method() !== "GET") writes.push(request.url()); });
      const readInputs = () => page.locator("main input, main select, main textarea, main [contenteditable=true]").evaluateAll(nodes => nodes.map(node => {
        const control = node as HTMLInputElement;
        return { tag: node.tagName, label: node.getAttribute("aria-label"), value: control.value ?? node.textContent, checked: control.checked ?? null };
      }));
      await page.getByRole("link", { name: "编辑商品", exact: true }).press("Enter");
      const name = page.getByRole("textbox", { name: "商品名称", exact: true });
      await expect(name).toHaveValue(title);
      await expect(page.getByRole("textbox", { name: "商品描述", exact: true })).toContainText("只用于共享页面操作区布局回归。");
      const baseline = await readInputs();
      await name.fill(`${title}未保存`);
      await page.getByRole("textbox", { name: "品牌", exact: true }).fill("未保存品牌");
      await page.getByRole("textbox", { name: "商品描述", exact: true }).fill("只在表单中保留的未保存描述");
      const dirty = await readInputs();
      const trigger = page.getByRole("button", { name: "取消并返回详情", exact: true });
      const dialog = page.getByRole("dialog", { name: "放弃未保存的修改？", exact: true });
      const keep = dialog.getByRole("button", { name: "继续编辑", exact: true });
      await trigger.press("Enter");
      await expect(dialog).toBeVisible();
      await expect(keep).toBeFocused();
      await expect(dialog).toHaveCSS("background-color", theme === "dark" ? "rgb(20, 34, 55)" : "rgb(255, 255, 255)");
      expect(await dialog.evaluate(node => { const rect = node.getBoundingClientRect(); return rect.left >= 0 && rect.right <= document.documentElement.clientWidth && rect.top >= 0 && rect.bottom <= innerHeight; })).toBe(true);
      await page.screenshot({ path: info.outputPath(`unsaved-dialog-${width}-${theme}.png`) });
      await keep.press("Enter");
      await expect(dialog).toBeHidden();
      await expect(trigger).toBeFocused();
      expect(await readInputs()).toEqual(dirty);
      await trigger.press("Enter");
      await keep.press("Escape");
      await expect(dialog).toBeHidden();
      await expect(trigger).toBeFocused();
      expect(await readInputs()).toEqual(dirty);
      await trigger.press("Enter");
      await dialog.getByRole("button", { name: "关闭", exact: true }).click();
      await expect(dialog).toBeHidden();
      await expect(trigger).toBeFocused();
      expect(await readInputs()).toEqual(dirty);
      await trigger.press("Enter");
      await keep.press("Tab");
      await expect(dialog.getByRole("button", { name: "放弃修改", exact: true })).toBeFocused();
      await dialog.getByRole("button", { name: "放弃修改", exact: true }).press("Enter");
      await expect(page).toHaveURL(new RegExp(`/workbench/data/mine/own/${itemId}$`));
      await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible();
      await page.getByRole("link", { name: "编辑商品", exact: true }).click();
      await expect(name).toHaveValue(title);
      await expect(page.getByRole("textbox", { name: "商品描述", exact: true })).toContainText("只用于共享页面操作区布局回归。");
      expect(await readInputs()).toEqual(baseline);
      await expect(page.getByRole("spinbutton", { name: "库存 1", exact: true })).toHaveValue("0");
      await expect(page.getByRole("spinbutton", { name: "库存 2", exact: true })).toHaveValue("");
      expect(nativeDialogs).toEqual([]);
      expect(writes).toEqual([]);
    });
  }
}

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

    const theme = page.getByRole("combobox", { name: "显示主题" });
    await theme.focus();
    await expect(panel).toBeHidden();
    await expect(theme).toBeFocused();
    await theme.selectOption("light");
    await expect(page.locator("html")).toHaveClass(/\blight\b/);

    await page.reload();
    await expect(theme).toHaveValue("light");
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

test("applies a saved system theme before hydration", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.addInitScript(() => localStorage.setItem("listingkit-theme", "system"));

  // The inline theme script can run while the application scripts are held.
  let releaseScripts!: () => void;
  const hydration = new Promise<void>((resolve) => { releaseScripts = resolve; });
  await page.route("**/_next/static/**", async (route) => {
    if (route.request().resourceType() === "script") await hydration;
    await route.continue();
  });
  try {
    await page.goto("/workbench", { waitUntil: "commit" });
    await expect(page.locator("html")).toHaveClass(/\blight\b/);
    await expect(page.getByRole("combobox", { name: "显示主题" })).toHaveCount(0);
  } finally {
    releaseScripts();
  }
  await expect(page.getByRole("combobox", { name: "显示主题" })).toHaveValue("system");
});

test.describe("workbench notification and store overlays", () => {
  test.skip(process.env.LISTINGKIT_NOTIFICATION_CENTER_ENABLED !== "true" || process.env.CHATWOOT_API_TOKEN !== "overlay-synthetic-only", "Requires the loopback notification and support fixture settings.");

  for (const width of [1440, 390]) {
    test(`${width}px follows system changes with the notification and store overlays open`, async ({ page }) => {
      await page.setViewportSize({ width, height: 844 });
      await page.emulateMedia({ colorScheme: "light" });
      await page.goto("/workbench/stores");

      const html = page.locator("html");
      const preference = page.getByRole("combobox", { name: "显示主题" });
      await expect(preference).toHaveValue("dark");
      await expect(html).toHaveClass(/\bdark\b/);
      await preference.selectOption("system");
      await expect(html).toHaveClass(/\blight\b/);

      const notice = page.getByRole("button", { name: "通知", exact: true });
      const panel = page.getByRole("region", { name: "通知面板", exact: true });
      const noticeClose = panel.getByRole("button", { name: "关闭通知面板" });
      await notice.press("Enter");
      await expect(panel).toHaveCSS("background-color", "rgb(255, 255, 255)");
      await page.emulateMedia({ colorScheme: "dark" });
      await expect(panel).toHaveCSS("background-color", "rgb(20, 34, 55)");
      await expect(noticeClose).toBeFocused();
      await page.keyboard.press("Escape");
      await expect(notice).toBeFocused();

      const groups = page.getByRole("button", { name: "分组管理", exact: true });
      const dialog = page.getByRole("dialog", { name: "分组管理", exact: true });
      await expect(groups).toBeEnabled();
      await groups.press("Enter");
      await expect(dialog).toHaveCSS("background-color", "rgb(20, 34, 55)");
      await page.emulateMedia({ colorScheme: "light" });
      await expect(dialog).toHaveCSS("background-color", "rgb(255, 255, 255)");
      await expect(dialog.getByRole("button", { name: "关闭" })).toBeFocused();
      await page.keyboard.press("Escape");
      await expect(groups).toBeFocused();

      await page.reload();
      await expect(preference).toHaveValue("system");
      await expect(html).toHaveClass(/\blight\b/);
      await preference.selectOption("light");
      await page.emulateMedia({ colorScheme: "dark" });
      await expect(html).toHaveClass(/\blight\b/);
      await page.reload();
      await expect(preference).toHaveValue("light");
      await expect(html).toHaveClass(/\blight\b/);
    });

    for (const theme of ["dark", "light"]) {
      test(`${width}px ${theme}: keyboard and focus restoration`, async ({ page }) => {
        await page.setViewportSize({ width, height: 844 });
        await page.emulateMedia({ colorScheme: theme === "dark" ? "light" : "dark" });
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
