import { expect, it } from "vitest";
import { catalogEntrySchema, configReceiptSchema, templateSchema } from "./agent-configuration";

const id = "8fc227bb-b572-4138-8e2a-5f1a0be98617";
const content = { schema: "content-config-v1", selection: { targetLanguage: "en-US", title: true, description: true, sellingPoints: true, seoKeywords: true } };
const template = { templateId: id, agentId: "product.content.agent", lifecycle: "ACTIVE", revision: "1", version: "1", schemaVersion: "content-config-v1", name: "English content", targetPlatform: "product", content, createdAt: "2026-10-12T00:00:00Z" };

it("reads only the registered content agent's typed language and field preferences", () => {
  expect(templateSchema.safeParse(template).success).toBe(true);
  for (const value of [
    { ...template, agentId: "product.title.agent" },
    { ...template, schemaVersion: "title-config-v1" },
    { ...template, content: undefined },
    { ...template, content: { ...content, selection: { targetLanguage: "en-US" } } },
    { ...template, content: { ...content, selection: { targetLanguage: "en-us", title: true } } },
    { ...template, content: { ...content, selection: { targetLanguage: "und", title: true } } },
    { ...template, content: { ...content, selection: { ...content.selection, price: true } } },
  ]) expect(templateSchema.safeParse(value).success).toBe(false);
});

it("accepts content catalog and receipts without borrowing title readiness or parameter schema", () => {
  const entry = { agent: { agentId: template.agentId, activation: "NOT_ENABLED", revision: "", activationEpoch: "", defaultTemplate: null, updatedAt: template.createdAt }, name: "商品内容完善与本地化", description: "文本建议", definitionVersion: "v1.0.0", parameterSchema: "content-config-v1", canConfigure: true, canUse: false, canReadRuns: false, capabilities: [] };
  expect(catalogEntrySchema.safeParse(entry).success).toBe(true);
  expect(catalogEntrySchema.safeParse({ ...entry, parameterSchema: "title-config-v1" }).success).toBe(false);
  expect(catalogEntrySchema.safeParse({ ...entry, agent: { ...entry.agent, agentId: "product.title.agent" } }).success).toBe(false);
  expect(configReceiptSchema.safeParse({ commandId: id, operation: "enable", agentId: template.agentId, revision: "1", noop: false, committedAt: template.createdAt }).success).toBe(true);
});
