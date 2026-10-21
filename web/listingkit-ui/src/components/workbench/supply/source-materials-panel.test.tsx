import { createContext, useContext, useState } from "react";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { SupplyAPIError, type SupplyIntent } from "@/lib/api/supply-chain";
import { SourceMaterialsPanel } from "./source-materials-panel";

const state = vi.hoisted(() => ({ source: vi.fn(), inventory: vi.fn(), inputs: vi.fn(), write: vi.fn(), verify: vi.fn() }));
const FixtureContext = createContext<Record<string,unknown>>({});
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => useContext(FixtureContext) }));
vi.mock("@/lib/api/supply-chain", async original => ({ ...await original<object>(), readSupplySource: state.source, supplyInventory: state.inventory, readSourceProductInputs: state.inputs, supplyCommand: state.write, readSupplyCommand: state.verify }));
const id = "11111111-1111-4111-8111-111111111111", image = "22222222-2222-4222-8222-222222222222", second = "33333333-3333-4333-8333-333333333333";
const scope = { userId: "actor-a", organizationId: "org-a" };
const source = { source: { id, preparationId: id, collectionItemId: id, collectionRevision: 1, source: { productKey: "product-a", publicationId: id, version: "1", kind: "acquisition" } },
  product: { title: "原始来源商品", sources: [{ platform: "1688" }] }, images: [{ id: image, url: "https://example.org/source.jpg", width: 900, height: 900 }, { id: second, url: "https://example.org/other.jpg", width: 900, height: 900 }] };
function assets(saved = false) { return { scope: { tenant_id: "org-a", product_key: "product-a", target_platform: "product", source_snapshot_version: "1" }, assets: saved ? [{ id: "saved", source_asset_id: image, role: "main", url: "https://example.org/source.jpg" }] : [] }; }
function inputs(saved = false) { return { sourceId: id, inputs: { status: "blocked", product_version: "1", publication_id: id, product_key: "product-a", asset_binding: { status: saved ? "exact" : "missing", count: saved ? 1 : 0 },
  reasons: saved ? ["source_review_required"] : ["approved_assets_not_ready", "source_review_required"], marketplace: { status: "not_evaluated" }, diagnostics: { needs_review: true, review_reasons: ["价格缺失"], warnings: [{ code: "source_missing", message: "来源价格尚未提供" }] } } }; }
function Harness({ organization = "org-a", permissions = ["workbench.supply.manage"] }: { organization?: string; permissions?: string[] }) {
  const [pending, setPending] = useState<SupplyIntent | null>(null);
  const [open, setOpen] = useState(true);
  const context = { user: { id: "actor-a" }, effectiveOrganization: { id: organization }, permissions, pendingSupplyIntent: pending, supplyIntentReady: true,
    setPendingSupplyIntent: (value: SupplyIntent | null) => { setPending(value); return true; }, registerOrganizationSwitchGuard: () => () => {} };
  return <FixtureContext.Provider value={context}>{open ? <SourceMaterialsPanel scope={scope} sourceId={id} onClose={() => setOpen(false)} /> : <button onClick={() => setOpen(true)}>打开来源素材</button>}</FixtureContext.Provider>;
}
beforeEach(() => {
  state.source.mockReset().mockResolvedValue(source);
  state.inventory.mockReset().mockResolvedValue(assets()); state.inputs.mockReset().mockResolvedValue(inputs());
  state.write.mockReset(); state.verify.mockReset();
});
afterEach(cleanup);

it("requires an explicit full image selection and keeps source warnings after confirmation", async () => {
  state.write.mockImplementation(async (intent: SupplyIntent) => { state.inventory.mockResolvedValue(assets(true)); state.inputs.mockResolvedValue(inputs(true)); return { action_id: intent.key, asset_ids: ["saved"] }; });
  render(<Harness />);
  expect(await screen.findByText("原始来源商品")).toBeInTheDocument();
  expect(state.write).not.toHaveBeenCalled();
  expect(screen.getByRole("button", { name: "确认采用所选图片" })).toBeDisabled();
  await userEvent.click(screen.getByRole("checkbox", { name: "使用来源图 1" }));
  await userEvent.selectOptions(screen.getByRole("combobox", { name: "来源图 1 用途" }), "main");
  await userEvent.click(screen.getByRole("button", { name: "确认采用所选图片" }));
  expect(await screen.findByText("图片已确认，资料仍待处理")).toBeInTheDocument();
  expect(screen.getByText("来源价格尚未提供")).toBeInTheDocument();
  expect(screen.queryByText("本版本资料就绪")).not.toBeInTheDocument();
  expect(state.write).toHaveBeenCalledOnce();
  expect(state.write.mock.calls[0][0].command).toEqual({ selection: { itemId: id, originalPublicationId: id, originalSnapshotVersion: "1", effectiveCatalogVersion: "1", targetPlatform: "product" }, images: [{ id: image, role: "main" }], approved: [] });
});

it("retains the original key and selection across close/reopen and only verifies that intent", async () => {
  state.write.mockRejectedValue(new SupplyAPIError("OUTCOME_UNKNOWN", 503));
  state.verify.mockImplementation(async (intent: SupplyIntent) => { state.inventory.mockResolvedValue(assets(true)); state.inputs.mockResolvedValue(inputs(true)); return { action_id: intent.key, asset_ids: ["saved"] }; });
  render(<Harness />);
  await screen.findByText("原始来源商品");
  await userEvent.click(screen.getByRole("checkbox", { name: "使用来源图 1" }));
  await userEvent.click(screen.getByRole("button", { name: "确认采用所选图片" }));
  expect(await screen.findByText("结果待核实")).toBeInTheDocument();
  const original = state.write.mock.calls[0][0];
  await userEvent.click(screen.getByRole("button", { name: "关闭弹窗" }));
  // A current inventory is not an immutable receipt for the pending action.
  state.inventory.mockResolvedValue(assets(true)); state.inputs.mockResolvedValue(inputs(true));
  await userEvent.click(screen.getByRole("button", { name: "打开来源素材" }));
  await screen.findByText("原始来源商品");
  expect(screen.getByRole("checkbox", { name: "使用来源图 1" })).toBeChecked();
  expect(screen.getByRole("checkbox", { name: "使用来源图 1" })).toBeDisabled();
  expect(screen.queryByText("图片已确认，资料仍待处理")).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "核实原确认" }));
  await waitFor(() => expect(state.verify).toHaveBeenCalledOnce());
  expect(state.verify.mock.calls[0][0]).toEqual(original);
  expect(state.write).toHaveBeenCalledOnce();
  expect(await screen.findByText("图片已确认，资料仍待处理")).toBeInTheDocument();
});

it("does not turn a denied or failed read into an empty or ready inventory", async () => {
  state.inventory.mockRejectedValue(new SupplyAPIError("PERMISSION_DENIED", 403));
  render(<Harness />);
  expect(await screen.findByRole("alert")).toHaveTextContent("当前身份没有来源素材权限");
  expect(screen.queryByText("资料暂不就绪")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "确认采用所选图片" })).not.toBeInTheDocument();
  expect(state.write).not.toHaveBeenCalled();
});

it("aborts old reads and removes old identity data on an organization change", async () => {
  let resolve!: (value: unknown) => void;
  state.source.mockReturnValue(new Promise(ok => { resolve = ok; }));
  const view = render(<Harness />);
  await waitFor(() => expect(state.source).toHaveBeenCalledOnce());
  const signal = state.source.mock.calls[0][2] as AbortSignal;
  view.rerender(<Harness organization="org-b" />);
  expect(signal.aborted).toBe(true);
  await act(async () => resolve(source));
  expect(screen.queryByText("原始来源商品")).not.toBeInTheDocument();
  expect(state.inventory).not.toHaveBeenCalled();
});

it("does not read or approve when the current permission is absent", async () => {
  render(<Harness permissions={[]} />);
  expect(await screen.findByRole("alert")).toHaveTextContent("没有来源素材权限");
  expect(state.source).not.toHaveBeenCalled(); expect(state.write).not.toHaveBeenCalled();
});
