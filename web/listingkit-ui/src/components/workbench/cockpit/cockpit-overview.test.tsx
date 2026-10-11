import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, expect, it, vi } from "vitest";
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
