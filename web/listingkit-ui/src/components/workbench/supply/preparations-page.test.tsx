import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { PreparationsPage } from "./preparations-page";

const state = vi.hoisted(() => ({ context: {} as Record<string, unknown>, preparations: vi.fn(), preparation: vi.fn(), sources: vi.fn(), detail: vi.fn() }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => state.context }));
vi.mock("@/lib/api/supply-chain", () => ({ listSupplyPreparations: state.preparations, readSupplyPreparation: state.preparation, listSupplySources: state.sources, readSupplySource: state.detail }));
const preparation = "11111111-1111-4111-8111-111111111111";
const source = "22222222-2222-4222-8222-222222222222";
beforeEach(() => {
  state.context = { user: { id: "actor-a" }, effectiveOrganization: { id: "org-a" } };
  state.preparation.mockReset().mockResolvedValue({ id: preparation, name: "单件批次", count: 1 });
  state.preparations.mockReset().mockResolvedValue({ items: [{ id: preparation, name: "单件批次", count: 1 }], total: 1 });
  state.sources.mockReset().mockResolvedValue({ items: [{ id: source, preparationId: preparation, source: { productKey: "product-a", version: 1 } }], total: 1 });
  state.detail.mockReset();
});
afterEach(cleanup);

it("shows four configuration steps before selection without exposing unavailable upload actions", async () => {
  render(<PreparationsPage completionAvailable />);
  for (const name of ["1. 选择商品", "2. 选择目标店铺", "3. 配置规则与智能体", "4. 确认提交"]) {
    expect(screen.getByRole("heading", { name })).toBeInTheDocument();
  }
  expect(screen.getByRole("button", { name: "预览本次任务" })).toBeDisabled();
  expect(screen.getByLabelText("目标店铺")).toBeDisabled();
  expect(state.sources).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole("button", { name: "选择商品" }));
  expect(await screen.findByRole("dialog", { name: "选择商品批次" })).toBeInTheDocument();
  await userEvent.click(await screen.findByRole("button", { name: /单件批次/ }));
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(await screen.findByRole("button", { name: "查看资料" })).toBeEnabled();
  expect(screen.getByRole("link", { name: "创建 / 编辑待补全草稿" })).toHaveAttribute("href", `/workbench/supply/mine/missing?preparation=${preparation}&source=${source}`);
  expect(screen.getByRole("button", { name: "预览本次任务" })).toBeDisabled();
});

it("can retry the same source after its detail read fails and the user refreshes", async () => {
  state.detail.mockRejectedValueOnce(new Error("temporary read failure")).mockResolvedValueOnce({ source: { source: { productKey: "product-a", version: 1 }, collectionItemId: source }, product: { title: "恢复后的商品" } });
  render(<PreparationsPage initialPreparation={preparation} />);
  await userEvent.click(await screen.findByRole("button", { name: "查看资料" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("读取失败");
  await userEvent.click(screen.getByRole("button", { name: "刷新" }));
  await userEvent.click(await screen.findByRole("button", { name: "查看资料" }));
  await waitFor(() => expect(state.detail).toHaveBeenCalledTimes(2));
  expect(await screen.findByRole("heading", { name: "恢复后的商品" })).toBeInTheDocument();
  expect(state.detail).toHaveBeenLastCalledWith({ userId: "actor-a", organizationId: "org-a" }, source, expect.any(AbortSignal));
});

it("can change batches from a preparation deep link and opens the newly selected source", async () => {
  const other = "44444444-4444-4444-8444-444444444444";
  state.preparations.mockResolvedValue({ items: [{ id: other, name: "另一批商品", count: 1 }], total: 1 });
  state.preparation.mockImplementation((_scope, id) => Promise.resolve({ id, name: id === other ? "另一批商品" : "单件批次", count: 1 }));
  state.sources.mockImplementation((_scope, id) => Promise.resolve({ items: [{ id: source, preparationId: id, source: { productKey: id, version: 1 } }], total: 1 }));
  render(<PreparationsPage initialPreparation={preparation} completionAvailable />);
  await screen.findByRole("button", { name: "查看资料" });
  await userEvent.click(screen.getByRole("button", { name: "更换商品批次" }));
  await userEvent.click(await screen.findByRole("button", { name: /另一批商品/ }));
  await waitFor(() => expect(state.sources).toHaveBeenLastCalledWith({ userId: "actor-a", organizationId: "org-a" }, other, expect.any(Object), expect.any(AbortSignal)));
  expect(await screen.findByRole("link", { name: "创建 / 编辑待补全草稿" })).toHaveAttribute("href", `/workbench/supply/mine/missing?preparation=${other}&source=${source}`);
  expect(screen.getByLabelText("已选商品")).toHaveValue("另一批商品 · 1 件");
});
