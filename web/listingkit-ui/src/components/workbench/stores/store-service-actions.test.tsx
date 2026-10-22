import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { StoreServiceActions } from "./store-service-actions";
import {
  WorkbenchAPIError,
  type WorkbenchStore,
} from "@/lib/api/workbench-stores";
import { commercialResourcesFixture } from "@/test/fixtures/commercial-resources";
const state = vi.hoisted(() => ({
  renew: vi.fn(),
  balance: vi.fn(),
  members: vi.fn(),
}));
vi.mock("@/lib/api/workbench-stores", async (original) => ({
  ...(await original<typeof import("@/lib/api/workbench-stores")>()),
  renewWorkbenchStoreService: state.renew,
}));
vi.mock("@/lib/api/commercial-billing", async (original) => ({
  ...(await original<typeof import("@/lib/api/commercial-billing")>()),
  getCommercialResources: state.balance,
}));
vi.mock("@/lib/api/member-resources", async (original) => ({
  ...(await original<typeof import("@/lib/api/member-resources")>()),
  getMemberResources: state.members,
}));
const scope = { expectedUserId: "actor", expectedOrganizationId: "org-1" };
const store = {
  id: "11111111-1111-4111-8111-11111111111a",
  name: "测试店铺",
  recordStatus: "active",
  serviceStatus: "active",
  version: 2,
} as WorkbenchStore;
function tree(administrator = false, connected = true, record = store, client = new QueryClient({ defaultOptions: { queries: { retry: false } } })) {
  return (
    <QueryClientProvider
      client={client}
    >
      <StoreServiceActions
        store={record}
        scope={scope}
        administrator={administrator}
        canWrite
        connected={connected}
        onChanged={vi.fn().mockResolvedValue(null)}
      />
    </QueryClientProvider>
  );
}
beforeEach(() => {
  sessionStorage.clear();
  Object.defineProperty(HTMLDialogElement.prototype, "showModal", {
    configurable: true,
    value: function () {
      this.open = true;
    },
  });
  Object.defineProperty(HTMLDialogElement.prototype, "close", {
    configurable: true,
    value: function () {
      this.open = false;
    },
  });
  state.renew.mockReset();
  state.balance.mockReset().mockResolvedValue(commercialResourcesFixture("org-1"));
  state.members.mockReset().mockResolvedValue({
    organizationId: "org-1",
    members: [
      { memberId: "current-member", userId: "actor", periods: { free: "2" } },
    ],
  });
});
afterEach(cleanup);
it("reload restores the original renewal version and key even after the Store advances", async () => {
  state.renew.mockRejectedValue(
    new WorkbenchAPIError(503, "STORE_SERVICE_OUTCOME_UNKNOWN", "safe", "", []),
  );
  const first = render(tree());
  await userEvent.click(screen.getByRole("button", { name: "续费服务" }));
  await screen.findByText("可用 2 期");
  await userEvent.click(screen.getByRole("button", { name: "确认续费服务" }));
  await screen.findByText(/续费结果未确认/);
  const original = state.renew.mock.calls[0];
  first.unmount();
  render(tree(false, true, { ...store, version: 3 }));
  await userEvent.click(screen.getByRole("button", { name: "续费服务" }));
  await userEvent.click(
    await screen.findByRole("button", { name: "核验原操作" }),
  );
  expect(state.renew.mock.calls[1]).toEqual(original);
});
it("member renewals retry only the original key, version and captured user", async () => {
  state.renew
    .mockRejectedValueOnce(
      new WorkbenchAPIError(
        503,
        "STORE_SERVICE_OUTCOME_UNKNOWN",
        "safe",
        "",
        [],
      ),
    )
    .mockResolvedValueOnce({
      quantity: "1",
      resourceBalanceAfter: "1",
      serviceExpiresAt: "2026-10-29T00:00:00Z",
    });
  render(tree());
  await userEvent.click(screen.getByRole("button", { name: "续费服务" }));
  const dialog = screen.getByRole("dialog");
  expect(await within(dialog).findByText("可用 2 期")).toBeVisible();
  await userEvent.click(
    within(dialog).getByRole("button", { name: "确认续费服务" }),
  );
  await screen.findByText(/续费结果未确认/);
  await userEvent.click(screen.getByRole("button", { name: "核验原操作" }));
  expect(state.renew).toHaveBeenCalledTimes(2);
  expect(state.renew.mock.calls[1]).toEqual(state.renew.mock.calls[0]);
  expect(state.renew.mock.calls[0]).toEqual([
    store.id,
    1,
    2,
    expect.any(String),
    "org-1",
    "actor",
  ]);
});
it("admin renewal uses unallocated balance and closes first activation when disconnected", async () => {
  const resource = commercialResourcesFixture("org-1");
  resource.resources[0].available = "0";
  resource.resources[0].allocated = "5";
  state.balance.mockResolvedValue(resource);
  const { unmount } = render(tree(true));
  await userEvent.click(screen.getByRole("button", { name: "续费服务" }));
  await screen.findByText("可用 0 期");
  expect(screen.getByRole("button", { name: "确认续费服务" })).toBeDisabled();
  expect(state.members).not.toHaveBeenCalled();
  unmount();
  render(tree(false, false, { ...store, serviceStatus: "pending_activation" }));
  expect(screen.getByRole("button", { name: "开通服务" })).toBeDisabled();
});

it.each([true, false])("reports failed period reads and retries only the correct funding query (admin=%s)", async administrator => {
  const read = administrator ? state.balance : state.members;
  const other = administrator ? state.members : state.balance;
  let finish!: () => void;
  const result = administrator ? commercialResourcesFixture("org-1") : { organizationId: "org-1", members: [{ userId: "actor", periods: { free: "2" } }] };
  read.mockRejectedValueOnce(new Error("synthetic read failure"))
    .mockImplementationOnce(() => new Promise(resolve => { finish = () => resolve(result); }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const view = render(tree(administrator, true, store, client));
  await userEvent.click(screen.getByRole("button", { name: "续费服务" }));
  await vi.waitFor(() => expect(client.getQueryCache().getAll().some(query => query.state.status === "error")).toBe(true));
  expect(screen.queryByText("可用 正在读取 期")).not.toBeInTheDocument();
  expect(screen.getByRole("alert")).toHaveTextContent("可用期数读取失败");
  expect(screen.getByRole("button", { name: "确认续费服务" })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "重新读取期数" }));
  expect(screen.getByText("正在读取可用期数…")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "确认续费服务" })).toBeDisabled();
  finish();
  await screen.findByText(`可用 ${administrator ? "1" : "2"} 期`);
  expect(screen.getByRole("button", { name: "确认续费服务" })).toBeEnabled();
  expect(read).toHaveBeenCalledTimes(2);
  expect(other).not.toHaveBeenCalled();
  expect(state.renew).not.toHaveBeenCalled();
  view.unmount(); client.clear();
});

it.each([true, false])("keeps missing period facts unknown instead of loading or zero (admin=%s)", async administrator => {
  state.balance.mockResolvedValue({ ...commercialResourcesFixture("org-1"), resources: [] });
  state.members.mockResolvedValue({ organizationId: "org-1", members: [] });
  render(tree(administrator));
  await userEvent.click(screen.getByRole("button", { name: "续费服务" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("当前未取得可用期数");
  expect(screen.queryByText("可用 0 期")).not.toBeInTheDocument();
  expect(screen.queryByText(/正在读取/)).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "重新读取期数" })).toBeEnabled();
  expect(screen.getByRole("button", { name: "确认续费服务" })).toBeDisabled();
  expect(state.renew).not.toHaveBeenCalled();
});

it("reads period balances without changing or replaying an UNKNOWN renewal", async () => {
  state.renew.mockRejectedValue(new WorkbenchAPIError(503, "STORE_SERVICE_OUTCOME_UNKNOWN", "safe", "", []));
  const first = render(tree());
  await userEvent.click(screen.getByRole("button", { name: "续费服务" }));
  await screen.findByText("可用 2 期");
  await userEvent.click(screen.getByRole("button", { name: "确认续费服务" }));
  await screen.findByText(/续费结果未确认/);
  const original = state.renew.mock.calls[0];
  const saved = { ...sessionStorage };
  first.unmount();
  state.members.mockRejectedValueOnce(new Error("synthetic read failure"));
  render(tree(false, true, { ...store, version: 3 }));
  await userEvent.click(screen.getByRole("button", { name: "续费服务" }));
  await screen.findByText("可用期数读取失败，请重试。");
  await userEvent.click(screen.getByRole("button", { name: "重新读取期数" }));
  await screen.findByText("可用 2 期");
  expect(state.renew).toHaveBeenCalledOnce();
  expect({ ...sessionStorage }).toEqual(saved);
  await userEvent.click(screen.getByRole("button", { name: "核验原操作" }));
  await screen.findByText(/续费结果未确认/);
  expect(state.renew.mock.calls[1]).toEqual(original);
});

it("shows a member's recorded zero without treating it as a read failure", async () => {
  state.members.mockResolvedValue({ organizationId: "org-1", members: [{ userId: "actor", periods: { free: "0" } }] });
  render(tree());
  await userEvent.click(screen.getByRole("button", { name: "续费服务" }));
  await screen.findByText("可用 0 期");
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "重新读取期数" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "确认续费服务" })).toBeDisabled();
  expect(state.balance).not.toHaveBeenCalled();
});
