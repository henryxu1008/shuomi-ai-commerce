import { useEffect } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ProductAgentPanel } from "./product-agent-panel";
import { ProductAgentError } from "@/lib/api/product-agent";

const f = vi.hoisted(() => ({ request: vi.fn(), params: new URLSearchParams(), context: { user: { id: "actor" }, effectiveOrganization: { id: "org", name: "企业" }, roles: [], permissions: [], registerOrganizationSwitchGuard: () => () => {} } }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => f.context }));
vi.mock("next/navigation", () => ({ useSearchParams: () => f.params }));
vi.mock("@/lib/api/product-agent", async original => ({ ...await original<typeof import("@/lib/api/product-agent")>(), requestProductAgent: f.request }));
vi.mock("../agents/agent-configuration-hooks", () => ({ useConfigurationRead: (_scope: unknown, path: string) => ({ data: { subject: { kind: "registered_agent", id: path.split("/").at(-1) }, definition: { id: path.split("/").at(-1), version: "v1.0.0" }, mode: "default", revision: "0", policyVersion: "agent-execution-policy-v1", canConfigure: false, updatedAt: "2026-10-12T00:00:00Z", updatedBy: "" } }) }));
vi.mock("./title-agent-templates", () => ({ TitleAgentTemplates: ({ onReady, onChoose }: { onReady: (ready: boolean) => void; onChoose: (value: null) => void }) => { useEffect(() => { onReady(true); onChoose(null); }, [onReady]); return null; } }));
const props = { enabled: true, operationId: "11111111-1111-4111-8111-111111111111", productKey: "product", catalogVersion: "1" };
beforeEach(() => {
  f.request.mockReset(); f.params = new URLSearchParams(); window.history.replaceState(null, "", "/");
  Object.defineProperty(HTMLDialogElement.prototype, "showModal", { configurable: true, value: function (this: HTMLDialogElement) { this.open = true; } });
  Object.defineProperty(HTMLDialogElement.prototype, "close", { configurable: true, value: function (this: HTMLDialogElement) { this.open = false; } });
});
afterEach(cleanup);

it("confirms exact content fields and locale, and retains the same namespace and intent after an unknown outcome", async () => {
  f.request.mockRejectedValue(new ProductAgentError("OUTCOME_UNKNOWN"));
  render(<ProductAgentPanel {...props} />);
  fireEvent.change(screen.getByLabelText("商品智能体"), { target: { value: "content" } });
  expect(screen.getByText("生成内容建议")).toBeDisabled();
  fireEvent.change(screen.getByLabelText("目标语言"), { target: { value: "en-US" } });
  fireEvent.click(screen.getByLabelText("SEO关键词"));
  fireEvent.click(screen.getByText("生成内容建议"));
  expect(screen.getByRole("dialog", { name: "确认内容完善与本地化" })).toHaveTextContent("en-US · 标题、描述、卖点");
  fireEvent.click(screen.getByText("确认生成"));
  await screen.findByRole("alert");
  const first = f.request.mock.calls[0];
  expect(first[7]).toBe("product");
  expect(first[11]).toEqual({ targetLanguage: "en-US", title: true, description: true, sellingPoints: true });
  expect(first[12]).toBe("content");
  expect(screen.getByLabelText("商品智能体")).toBeDisabled();
  const saved = new URLSearchParams(window.location.search);
  expect(saved.get("agent_kind")).toBe("content");
  cleanup(); f.params = saved;
  render(<ProductAgentPanel {...props} />);
  expect(f.request).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByText("使用原请求核实启动"));
  await waitFor(() => expect(f.request).toHaveBeenCalledTimes(2));
  expect(f.request.mock.calls[1].slice(0, 3)).toEqual(first.slice(0, 3));
  expect(f.request.mock.calls[1][11]).toEqual(first[11]);
  expect(f.request.mock.calls[1][12]).toBe("content");
});
