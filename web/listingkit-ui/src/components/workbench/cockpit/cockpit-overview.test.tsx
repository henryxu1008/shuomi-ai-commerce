import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, expect, it, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import { CockpitEntry } from "../console/cockpit-entry";
import { cockpitRequest } from "@/lib/api/operations-cockpit";
const state = vi.hoisted(() => ({ context: { user: { id: "user-a" }, effectiveOrganization: { id: "org-a" }, permissions: ["workbench.cockpit.stores.read"], operationsCockpitAvailable: true, isLoading: false, isSwitching: false, selectionRequired: false, error: null, blockingError: null } }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => state.context }));
vi.mock("@/lib/api/operations-cockpit", async original => ({ ...await original<typeof import("@/lib/api/operations-cockpit")>(), cockpitRequest: vi.fn() }));
const clients: QueryClient[] = [];
const access = { goalsRead: false, goalsCreate: false, goalsManage: false, storesRead: true, factsWrite: false, alertsRead: false, adviceRead: false };
const totals = { revenue: 120000, refunds: 0, procurement: 100000, logistics: 0, platform: 0, advertising: 0, other: 0, netRevenue: 120000, netProfit: 20000, margin: null };
const capabilities = { access, today: "2026-10-11", stores: [] };
const matrix = { rows: [], total: 0, page: 1, summary: totals, summaryComplete: false, states: { loss: 0, break_even: 0, recorded_complete: 0, data_incomplete: 0 }, capturedAt: "2026-10-11T00:00:00Z" };
function mount() { const client = new QueryClient({ defaultOptions: { queries: { retry: false } } }); clients.push(client); const tree = () => <QueryClientProvider client={client}><CockpitEntry /></QueryClientProvider>; const view = render(tree()); return { ...view, update: () => view.rerender(tree()) }; }
afterEach(() => { cleanup(); clients.splice(0).forEach(c => c.clear()); vi.mocked(cockpitRequest).mockReset(); state.context.isSwitching = false; state.context.effectiveOrganization = { id: "org-a" }; });
it("does not advertise or request oversized detail scopes and restores a valid single-store link", async () => {
  const stores=Array.from({length:400},(_,i)=>({id:`${(i+1).toString(16).padStart(8,"0")}-1111-4111-8111-111111111111`,name:`Store ${i}`,platform:"shein",region:"US"}));
  vi.mocked(cockpitRequest).mockImplementation(async (_scope,path)=>path==="capabilities"?{...capabilities,stores}:matrix);
  mount();
  await userEvent.selectOptions(await screen.findByRole("combobox",{name:"平台"}),"shein");
  expect(screen.getAllByText("范围较大，请按店铺筛选").length).toBeGreaterThan(0);
  expect(screen.queryByRole("link",{name:/查看真实经营明细/})).not.toBeInTheDocument();
  expect(vi.mocked(cockpitRequest).mock.calls.filter(call=>call[1].startsWith("stores?")).every(call=>call[1].length<16000)).toBe(true);
  await userEvent.selectOptions(screen.getByRole("combobox",{name:"店铺"}),stores[0].id);
  const href=screen.getByRole("link",{name:/查看真实经营明细/}).getAttribute("href")!;
  expect(new URL(href,"https://localhost").searchParams.getAll("storeId")).toEqual([stores[0].id]);
});
it("does not reinterpret partial manual totals as full profit, GMV or a health score", async () => {
  vi.mocked(cockpitRequest).mockImplementation(async (_scope, path) => path === "capabilities" ? capabilities : matrix);
  mount();
  await screen.findByText("待完整数据，未按部分录入推定完整利润");
  expect(screen.queryByText("¥200.00")).not.toBeInTheDocument();
  expect(screen.getByText("评分规则暂未接入")).toBeVisible();
  expect(vi.mocked(cockpitRequest).mock.calls.every(call => call[1] === "capabilities" || call[1].startsWith("stores?"))).toBe(true);
});
it("cancels the old scope and never displays its late profit after switching enterprise", async () => {
  let resolve!: (value: unknown) => void, oldSignal: AbortSignal | undefined;
  vi.mocked(cockpitRequest).mockImplementation(async (scope, path, _schema, init) => {
    if (path === "capabilities") return capabilities;
    if (scope.organizationId === "org-a") { oldSignal = init?.signal as AbortSignal; return new Promise(done => { resolve = done; }); }
    return matrix;
  });
  const view = mount();
  await waitFor(() => expect(oldSignal).toBeDefined());
  state.context.isSwitching = true; view.update();
  expect(oldSignal?.aborted).toBe(true);
  expect(screen.queryByText("净利润")).not.toBeInTheDocument();
  state.context.isSwitching = false; state.context.effectiveOrganization = { id: "org-b" }; view.update();
  await screen.findByText("待完整数据，未按部分录入推定完整利润");
  await act(async () => resolve({ ...matrix, summaryComplete: true }));
  expect(screen.queryByText("¥200.00")).not.toBeInTheDocument();
});
it.each([false, true])("preserves unavailable goal evaluation alongside actual rules (%s)", async (hasRules) => {
  const rule = { id: "rule-a", title: "真实经营预警", reason: "已保存成本", advice: "核对成本", level: "attention", actionPath: "/workbench/overview/stores", actionLabel: "查看" };
  vi.mocked(cockpitRequest).mockImplementation(async (_scope, path) => path === "capabilities" ? { ...capabilities, access: { ...access, alertsRead: true, adviceRead: true } } : path.startsWith("stores?") ? matrix : { rules: hasRules ? [rule] : [], goalUnavailable: true });
  mount();
  await waitFor(() => expect(screen.getAllByText("企业当前目标或关联店铺不可访问，目标评判暂不可用。")).toHaveLength(2));
  expect(screen.queryByText("当前范围暂无规则预警")).not.toBeInTheDocument();
  expect(screen.queryByText("当前范围暂无人工处理建议")).not.toBeInTheDocument();
  if (hasRules) expect(screen.getAllByText("真实经营预警")).toHaveLength(2);
});
