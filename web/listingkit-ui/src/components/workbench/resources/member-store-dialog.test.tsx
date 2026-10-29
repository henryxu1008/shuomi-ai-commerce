import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { MemberStores } from "./member-store-dialog";
import {
  MemberResourceError,
  type MemberResourceEntry,
} from "@/lib/api/member-resources";
const state = vi.hoisted(() => ({
  read: vi.fn(), write: vi.fn(), available: vi.fn(), grant: vi.fn(),
}));
vi.mock("@/lib/api/member-resources", async (original) => ({
  ...(await original<typeof import("@/lib/api/member-resources")>()),
  getMemberStores: state.read,
  getMemberStoreGrant: state.grant,
  setMemberStoreGrant: state.write,
}));
vi.mock("@/lib/api/workbench-stores", () => ({
  listWorkbenchStores: state.available,
}));
const member = {
  memberId: "member",
  displayName: "保留成员",
  roles: [], permissions: [],
  state: "departed",
} as unknown as MemberResourceEntry;
const store = {
  id: "11111111-1111-4111-8111-11111111111a",
  name: "原授权店铺",
  grantVersion: "1",
  serviceExpiresAt: null,
};
const activeMember = { ...member, state: "active" } as MemberResourceEntry;
const availableStores = {
  items: [{ id: store.id, name: "可选店铺" }],
  pagination: { page: 1, pageSize: 20, total: 1 },
};
function tree(org = "org-1", currentMember = member) {
  return (
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <MemberStores
        member={currentMember}
        scope={{ expectedUserId: "actor", expectedOrganizationId: org }}
        canManage
        onClose={vi.fn()}
        onChanged={vi.fn().mockResolvedValue(undefined)}
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
  vi.resetAllMocks();
  state.read.mockResolvedValue({ items: [store], total: "1" });
  state.available.mockResolvedValue(availableStores);
  state.grant.mockResolvedValue({ active: false, version: "3" });
  state.write.mockResolvedValue(undefined);
});
afterEach(cleanup);
it("restores a revoked grant command across reload without adopting a new grant version", async () => {
  state.write.mockRejectedValue(
    new MemberResourceError(502, "RESULT_UNVERIFIED", "unknown"),
  );
  const first = render(tree());
  await userEvent.click(
    await screen.findByRole("button", { name: "撤销 原授权店铺 授权" }),
  );
  await screen.findByText(/操作结果未确认/);
  const original = state.write.mock.calls[0];
  first.unmount();
  state.read.mockResolvedValue({
    items: [{ ...store, grantVersion: "2" }],
    total: "1",
  });
  render(tree());
  await userEvent.click(
    await screen.findByRole("button", { name: "核验原操作" }),
  );
  expect(state.write.mock.calls[1]).toEqual(original);
});
it("does not restore a pending grant into another organization", async () => {
  state.write.mockRejectedValue(
    new MemberResourceError(502, "RESULT_UNVERIFIED", "unknown"),
  );
  const first = render(tree());
  await userEvent.click(
    await screen.findByRole("button", { name: "撤销 原授权店铺 授权" }),
  );
  await screen.findByText(/操作结果未确认/);
  first.unmount();
  render(tree("org-2"));
  await screen.findByRole("button", { name: "撤销 原授权店铺 授权" });
  expect(
    screen.queryByRole("button", { name: "核验原操作" }),
  ).not.toBeInTheDocument();
  expect(state.write).toHaveBeenCalledTimes(1);
});

it("explains available-store read failure and recovers with the existing read-only refresh", async () => {
  state.available.mockRejectedValueOnce(new Error("unavailable"));
  render(tree("org-1", activeMember));
  expect(await screen.findByRole("alert")).toHaveTextContent(
    "可选店铺读取失败，请刷新店铺授权后重试。",
  );
  expect(screen.getAllByRole("option")).toHaveLength(1);
  expect(screen.getByRole("button", { name: "确认授权" })).toBeDisabled();
  expect(state.grant).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole("button", { name: "刷新店铺授权" }));
  await screen.findByRole("option", { name: "可选店铺" });
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  expect(state.available).toHaveBeenLastCalledWith(
    { page: 1, pageSize: 20 }, "org-1", expect.any(AbortSignal),
  );
  expect(state.write).not.toHaveBeenCalled();
});

it("explains selected-grant read failure and retains selection when refresh recovers", async () => {
  state.grant.mockRejectedValueOnce(new Error("unavailable"));
  render(tree("org-1", activeMember));
  await userEvent.selectOptions(await screen.findByRole("combobox"),
    (await screen.findByRole("option", { name: "可选店铺" })).getAttribute("value")!,
  );
  expect(await screen.findByRole("alert")).toHaveTextContent(
    "所选店铺授权状态读取失败，请刷新店铺授权后重试。",
  );
  expect(screen.getByRole("button", { name: "确认授权" })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "刷新店铺授权" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "确认授权" })).toBeEnabled());
  expect(screen.getByRole("combobox")).toHaveValue(store.id);
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  expect(state.grant).toHaveBeenLastCalledWith(
    { expectedUserId: "actor", expectedOrganizationId: "org-1" },
    "member", store.id, expect.any(AbortSignal),
  );
  expect(state.write).not.toHaveBeenCalled();
});

it("shows actual loading states without inventing a grant read before selection", async () => {
  let finishAvailable!: (value: typeof availableStores) => void;
  let finishGrant!: (value: { active: boolean; version: string }) => void;
  state.available.mockImplementationOnce(() => new Promise((resolve) => { finishAvailable = resolve; }));
  state.grant.mockImplementationOnce(() => new Promise((resolve) => { finishGrant = resolve; }));
  render(tree("org-1", activeMember));
  expect(await screen.findByText("正在读取可选店铺…")).toHaveAttribute("role", "status");
  expect(screen.queryByText("正在读取所选店铺授权状态…")).not.toBeInTheDocument();
  expect(state.grant).not.toHaveBeenCalled();
  finishAvailable(availableStores);
  await userEvent.selectOptions(screen.getByRole("combobox"),
    (await screen.findByRole("option", { name: "可选店铺" })).getAttribute("value")!,
  );
  expect(await screen.findByText("正在读取所选店铺授权状态…")).toHaveAttribute("role", "status");
  expect(screen.getByRole("button", { name: "确认授权" })).toBeDisabled();
  finishGrant({ active: false, version: "3" });
  await waitFor(() => expect(screen.getByRole("button", { name: "确认授权" })).toBeEnabled());
  expect(screen.queryByRole("status")).not.toBeInTheDocument();
  expect(state.write).not.toHaveBeenCalled();
});

it("keeps a successful empty directory distinct from a read failure", async () => {
  state.available.mockResolvedValue({ items: [], pagination: { page: 1, pageSize: 20, total: 0 } });
  render(tree("org-1", activeMember));
  await waitFor(() => expect(screen.getByRole("combobox")).toBeEnabled());
  expect(screen.getAllByRole("option")).toHaveLength(1);
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  expect(screen.queryByText("正在读取所选店铺授权状态…")).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "确认授权" })).toBeDisabled();
  expect(state.write).not.toHaveBeenCalled();
});

it("does not present a cached active grant as confirmed after its refresh fails", async () => {
  state.grant.mockResolvedValueOnce({ active: true, version: "3" })
    .mockRejectedValueOnce(new Error("unavailable"));
  render(tree("org-1", activeMember));
  await userEvent.selectOptions(screen.getByRole("combobox"),
    (await screen.findByRole("option", { name: "可选店铺" })).getAttribute("value")!,
  );
  expect(await screen.findByRole("button", { name: "已授权" })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "刷新店铺授权" }));
  await screen.findByRole("alert");
  expect(screen.queryByRole("button", { name: "已授权" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "确认授权" })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "刷新店铺授权" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "确认授权" })).toBeEnabled());
  expect(screen.getByRole("combobox")).toHaveValue(store.id);
  expect(state.write).not.toHaveBeenCalled();
});

it("preserves an UNKNOWN grant command through read failures and read-only refresh", async () => {
  state.write.mockRejectedValue(new MemberResourceError(502, "RESULT_UNVERIFIED", "unknown"));
  render(tree("org-1", activeMember));
  await userEvent.selectOptions(screen.getByRole("combobox"),
    (await screen.findByRole("option", { name: "可选店铺" })).getAttribute("value")!,
  );
  const confirm = screen.getByRole("button", { name: "确认授权" });
  await waitFor(() => expect(confirm).toBeEnabled());
  state.available.mockRejectedValue(new Error("unavailable"));
  state.grant.mockRejectedValue(new Error("unavailable"));
  await userEvent.click(confirm);
  await screen.findByText(/操作结果未确认/);
  const original = state.write.mock.calls[0];
  expect(original).toEqual([
    { expectedUserId: "actor", expectedOrganizationId: "org-1" },
    "member", store.id, { active: true, expectedVersion: "3" }, expect.any(String),
  ]);
  const saved = { ...sessionStorage };
  await userEvent.click(screen.getByRole("button", { name: "刷新店铺授权" }));
  await screen.findByText("所选店铺授权状态读取失败，请刷新店铺授权后重试。");
  expect({ ...sessionStorage }).toEqual(saved);
  expect(state.write).toHaveBeenCalledTimes(1);
  expect(screen.getByRole("button", { name: "确认授权" })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "核验原操作" }));
  expect(state.write.mock.calls[1]).toEqual(original);
});

it("keeps assigned-list error feedback and the existing successful grant command", async () => {
  state.read.mockRejectedValueOnce(new Error("unavailable"));
  render(tree("org-1", activeMember));
  expect(await screen.findByRole("alert")).toHaveTextContent("本次未取得店铺授权，请刷新后重试。");
  await userEvent.click(screen.getByRole("button", { name: "刷新店铺授权" }));
  await screen.findByRole("button", { name: "撤销 原授权店铺 授权" });
  await userEvent.selectOptions(screen.getByRole("combobox"), store.id);
  const confirm = screen.getByRole("button", { name: "确认授权" });
  await waitFor(() => expect(confirm).toBeEnabled());
  expect(state.write).not.toHaveBeenCalled();
  await userEvent.click(confirm);
  await screen.findByText("店铺授权已保存。");
  expect(state.write).toHaveBeenCalledExactlyOnceWith(
    { expectedUserId: "actor", expectedOrganizationId: "org-1" },
    "member", store.id, { active: true, expectedVersion: "3" }, expect.any(String),
  );
  expect(sessionStorage.length).toBe(0);
});
