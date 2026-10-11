import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { PreparationsPage } from "./preparations-page";

const state = vi.hoisted(() => ({ context: {} as Record<string, unknown>, preparation: vi.fn(), sources: vi.fn(), detail: vi.fn() }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => state.context }));
vi.mock("@/lib/api/supply-chain", () => ({ listSupplyPreparations: vi.fn(), readSupplyPreparation: state.preparation, listSupplySources: state.sources, readSupplySource: state.detail }));
const preparation = "11111111-1111-4111-8111-111111111111";
const source = "22222222-2222-4222-8222-222222222222";
beforeEach(() => {
  state.context = { user: { id: "actor-a" }, effectiveOrganization: { id: "org-a" } };
  state.preparation.mockReset().mockResolvedValue({ id: preparation, name: "单件批次", count: 1 });
  state.sources.mockReset().mockResolvedValue({ items: [{ id: source, source: { productKey: "product-a", version: 1 } }], total: 1 });
  state.detail.mockReset();
});
afterEach(cleanup);

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
