import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { AgentPage } from "./agent-page";

vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => ({ user: { id: "actor" }, effectiveOrganization: { id: "org-a", name: "企业A" }, permissions: [], isLoading: false, isSwitching: false }) }));
vi.mock("next/link", () => ({ default: ({ children, ...props }: React.ComponentProps<"a">) => <a {...props}>{children}</a> }));
vi.mock("./execution-mode-settings", () => ({ ExecutionModeSettings: ({ subject }: { subject: { id: string } }) => <p>模式归属：{subject.id}</p> }));
const agentId = "product.content.agent";
const entry = { agent: { agentId, activation: "ENABLED", revision: "1", activationEpoch: "1", defaultTemplate: null, updatedAt: "2026-10-12T00:00:00Z" }, name: "商品内容完善与本地化智能体", description: "标题、描述、卖点和SEO建议", definitionVersion: "v1.0.0", parameterSchema: "content-config-v1", canConfigure: true, canUse: false, canReadRuns: false, capabilities: [{ id: "text.generate", support: "REQUIRED", readiness: "NEEDS_CONFIGURATION", reason: "内容模型尚未配置", observedAt: "2026-10-12T00:00:00Z" }] };
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

it("reads the content detail and saves explicit locale and fields to its own template endpoint", async () => {
  Object.defineProperty(HTMLDialogElement.prototype, "showModal", { configurable: true, value: function (this: HTMLDialogElement) { this.open = true; } });
  Object.defineProperty(HTMLDialogElement.prototype, "close", { configurable: true, value: function (this: HTMLDialogElement) { this.open = false; } });
  const fetch = vi.fn(async (url: unknown, init?: RequestInit) => {
    if (init?.method === "POST") return Response.json({ commandId: "8fc227bb-b572-4138-8e2a-5f1a0be98617", operation: "create-template", agentId, templateId: "45a227bb-b572-4138-8e2a-5f1a0be98617", revision: "1", version: "1", noop: false, committedAt: entry.agent.updatedAt });
    if (String(url) === `/api/workbench/agents/${agentId}`) return Response.json(entry);
    if (String(url).includes("/templates")) return Response.json({ items: [], nextCursor: "" });
    return Response.json({ code: "DEPENDENCY_UNAVAILABLE" }, { status: 503 });
  });
  vi.stubGlobal("fetch", fetch);
  render(<AgentPage mode="mine" id={agentId} />);
  expect(await screen.findByText(entry.name)).toBeInTheDocument();
  expect(screen.getByText("模式归属：product.content.agent")).toBeInTheDocument();
  expect(screen.getByText("内容模型尚未配置")).toBeInTheDocument();
  expect(screen.queryByText("选择商品并进入使用 →")).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "新建模板" }));
  fireEvent.change(screen.getByLabelText("模板名称"), { target: { value: "日语内容" } });
  expect(screen.getByRole("button", { name: "保存模板" })).toBeDisabled();
  fireEvent.change(screen.getByLabelText("目标语言"), { target: { value: "ja" } });
  fireEvent.click(screen.getByLabelText("标题"));
  fireEvent.click(screen.getByLabelText("SEO关键词"));
  fireEvent.click(screen.getByRole("button", { name: "保存模板" }));
  await waitFor(() => expect(fetch.mock.calls.filter(([, init]) => init?.method === "POST")).toHaveLength(1));
  const [url, init] = fetch.mock.calls.find(([, init]) => init?.method === "POST")!;
  expect(url).toBe(`/api/workbench/agents/${agentId}/templates`);
  expect(JSON.parse(init!.body as string)).toEqual({ name: "日语内容", targetPlatform: "product", defaultKnowledgeBaseId: null, content: { schema: "content-config-v1", selection: { targetLanguage: "ja", description: true, sellingPoints: true } } });
  expect(fetch.mock.calls.some(([url]) => String(url).includes("product.title.agent"))).toBe(false);
});
