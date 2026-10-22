import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { MemberResources } from "./member-resources";
import { MemberResourceError } from "@/lib/api/member-resources";
import { commercialResourcesFixture } from "@/test/fixtures/commercial-resources";
const state = vi.hoisted(() => ({
  read: vi.fn(),
  write: vi.fn(),
  balance: vi.fn(),
  prices: vi.fn(),
}));
vi.mock("@/lib/api/member-resources", async (original) => ({
  ...(await original<typeof import("@/lib/api/member-resources")>()),
  getMemberResources: state.read,
  transferMemberResource: state.write,
  getMemberDataPrices: state.prices,
}));
vi.mock("@/lib/api/commercial-billing", async (original) => ({
  ...(await original<typeof import("@/lib/api/commercial-billing")>()),
  getCommercialResources: state.balance,
}));
const position = {
  free: "2",
  reserved: "1",
  consumed: "3",
  version: "1",
  recorded: true,
};
const member = {
  memberId: "member-1",
  userId: "user-1",
  displayName: "张琳",
  loginName: "member@example.test",
  state: "active",
  roles: ["listingkit_operator"], permissions: ["product_sourcing.write","local_agent.write","listingkit.image_agent.read","listingkit.image_agent.write","workbench.agent.read","workbench.agent.use","workbench.chat.read","workbench.chat.use","workbench.task.read","workbench.knowledge.read","workbench.store.read","workbench.store.create","workbench.store.update","workbench.store.lifecycle","workbench.source_account.read","workbench.source_account.manage","workbench.organization_member.read","workbench.commercial.read"],
  storeCount: "1",
  periods: position,
  dataRows: position,
};
function tree(client = new QueryClient({ defaultOptions: { queries: { retry: false } } })) {
  return (
    <QueryClientProvider
      client={client}
    >
      <MemberResources
        userId="actor"
        organizationId="org-1"
        canManage
        sequence={0}
      />
    </QueryClientProvider>
  );
}
beforeEach(() => {
  sessionStorage.clear();
  state.read.mockReset();
  state.balance.mockReset();
  state.prices.mockReset();
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
  state.read.mockResolvedValue({
    schemaVersion: "member-resource-directory-v1",
    organizationId: "org-1",
    observedAt: "2026-09-29T00:00:00Z",
    members: [member],
  });
  state.balance.mockResolvedValue(commercialResourcesFixture("org-1"));
  state.prices.mockResolvedValue({ organizationId: "org-1", offers: [] });
  state.write.mockReset();
});
afterEach(cleanup);
it.each(["bad", "1.5", "1e3", "-1", "9223372036854775808"])(
  "rejects invalid allocation quantity %s without crashing or dispatching",
  async (value) => {
    render(tree());
    await screen.findByText("张琳");
    await userEvent.click(screen.getByRole("button", { name: "分配资源" }));
    const dialog = screen.getByRole("dialog", { name: "分配资源" });
    await userEvent.type(
      within(dialog).getByRole("textbox", { name: /本次期数/ }),
      value,
    );
    expect(
      within(dialog).getByRole("button", { name: "确认分配" }),
    ).toBeDisabled();
    expect(state.write).not.toHaveBeenCalled();
  },
);
it("retains the exact allocation command after response loss and verifies only that operation", async () => {
  state.write
    .mockRejectedValueOnce(
      new MemberResourceError(502, "RESULT_UNVERIFIED", "unknown"),
    )
    .mockResolvedValueOnce({ netCredit: "0", debtRepaid: "0" });
  render(tree());
  await screen.findByText("张琳");
  await userEvent.click(screen.getByRole("button", { name: "分配资源" }));
  const dialog = screen.getByRole("dialog", { name: "分配资源" });
  await userEvent.type(
    within(dialog).getByRole("textbox", { name: /本次期数/ }),
    "1",
  );
  await userEvent.click(
    within(dialog).getByRole("button", { name: "确认分配" }),
  );
  await screen.findByText(/操作结果未确认/);
  await userEvent.click(screen.getByRole("button", { name: "核验原操作" }));
  expect(state.write).toHaveBeenCalledTimes(2);
  expect(state.write.mock.calls[1]).toEqual(state.write.mock.calls[0]);
});
it("restores an allocation after a reload and retries only its original command", async () => {
  state.write.mockRejectedValue(
    new MemberResourceError(502, "RESULT_UNVERIFIED", "unknown"),
  );
  const first = render(tree());
  await screen.findByText("张琳");
  await userEvent.click(screen.getByRole("button", { name: "分配资源" }));
  await userEvent.type(screen.getByRole("textbox", { name: /本次期数/ }), "1");
  await userEvent.click(screen.getByRole("button", { name: "确认分配" }));
  await screen.findByText(/操作结果未确认/);
  const original = state.write.mock.calls[0];
  first.unmount();
  render(tree());
  await screen.findByText("张琳");
  await userEvent.click(screen.getByRole("button", { name: "分配资源" }));
  await userEvent.click(
    await screen.findByRole("button", { name: "核验原操作" }),
  );
  expect(state.write.mock.calls[1]).toEqual(original);
  expect(state.write).toHaveBeenCalledTimes(2);
});
it("does not dispatch an allocation when recovery storage cannot be saved", async () => {
  const save = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
    throw new Error("disabled");
  });
  try {
    render(tree());
    await screen.findByText("张琳");
    await userEvent.click(screen.getByRole("button", { name: "分配资源" }));
    await userEvent.type(
      screen.getByRole("textbox", { name: /本次期数/ }),
      "1",
    );
    await userEvent.click(screen.getByRole("button", { name: "确认分配" }));
    expect(state.write).not.toHaveBeenCalled();
  } finally {
    save.mockRestore();
  }
});
it("keeps departed member holdings visible and allows only reclaim", async () => {
  state.read.mockResolvedValue({
    schemaVersion: "member-resource-directory-v1",
    organizationId: "org-1",
    observedAt: "2026-09-29T00:00:00Z",
    members: [{ ...member, state: "departed" }],
  });
  render(tree());
  await screen.findByText("张琳");
  expect(screen.getByRole("button", { name: "分配资源" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "回收资源" })).toBeEnabled();
  expect(screen.queryByText("未提供")).not.toBeInTheDocument();
});

const directoryKey = ["workbench", "actor", "org-1", "member-resources"];
function directory(entry = member) {
  return {
    schemaVersion: "member-resource-directory-v1",
    organizationId: "org-1",
    observedAt: "2026-09-29T00:00:00Z",
    members: [entry],
  };
}
async function openTransfer(action: "分配资源" | "回收资源", client?: QueryClient) {
  render(tree(client));
  await screen.findByText("张琳");
  await userEvent.click(screen.getByRole("button", { name: action }));
  return within(screen.getByRole("dialog", { name: action }));
}
it.each(["store_renewal_period", "data_row"])(
  "uses the refreshed %s holding for a new reclaim after debt repayment",
  async (resourceType) => {
    const updated = {
      ...member,
      periods: { ...position, free: "1", version: "2" },
      dataRows: { ...position, free: "1", version: "2" },
    };
    state.write.mockImplementationOnce(async () => {
      state.read.mockResolvedValue(directory(updated));
      return { debtRepaid: "1", netCredit: "0" };
    }).mockResolvedValue({ debtRepaid: "0", netCredit: "1" });
    const dialog = await openTransfer("回收资源");
    await userEvent.selectOptions(dialog.getByRole("combobox", { name: "资源类型" }), resourceType);
    const input = dialog.getByRole("textbox", { name: /本次期数|本次条数/ });
    await userEvent.type(input, "1");
    await userEvent.click(dialog.getByRole("button", { name: "确认回收" }));
    await dialog.findByText("已回收，偿还企业欠额 1，转入未分配余额 0。");
    expect(screen.getByRole("table")).toHaveTextContent("1 条");
    await userEvent.clear(input);
    await userEvent.type(input, "2");
    expect(dialog.getByRole("button", { name: "确认回收" })).toBeDisabled();
    await userEvent.clear(input);
    await userEvent.type(input, "1");
    await userEvent.click(dialog.getByRole("button", { name: "确认回收" }));
    expect(state.write).toHaveBeenCalledTimes(2);
    expect(state.write.mock.calls[1][2]).toEqual({
      resourceType, action: "reclaim", quantity: "1", expectedVersion: "2",
    });
    expect(state.write.mock.calls[1][3]).not.toBe(state.write.mock.calls[0][3]);
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  },
);

it.each(["pending", "error", "missing"])(
  "does not submit from the old selection when the directory is %s",
  async (result) => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const dialog = await openTransfer("回收资源", client);
    await userEvent.type(dialog.getByRole("textbox", { name: /本次期数/ }), "1");
    await waitFor(() => expect(dialog.getByRole("button", { name: "确认回收" })).toBeEnabled());
    if (result === "pending") state.read.mockImplementation(() => new Promise(() => {}));
    else if (result === "error") state.read.mockRejectedValue(new Error("directory unavailable"));
    else state.read.mockResolvedValue({ ...directory(), members: [] });
    await act(async () => {
      const refresh = client.invalidateQueries({ queryKey: directoryKey });
      if (result !== "pending") await refresh;
    });
    await waitFor(() => expect(dialog.getByRole("button", { name: "确认回收" })).toBeDisabled());
    await userEvent.click(dialog.getByRole("button", { name: "确认回收" }));
    expect(state.write).not.toHaveBeenCalled();
    client.clear();
  },
);

it("uses the current member version for an allocation after a directory refresh", async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const dialog = await openTransfer("分配资源", client);
  await userEvent.type(dialog.getByRole("textbox", { name: /本次期数/ }), "1");
  state.read.mockResolvedValue(directory({ ...member, periods: { ...position, version: "2" } }));
  await act(() => client.invalidateQueries({ queryKey: directoryKey }));
  await waitFor(() => expect(dialog.getByRole("button", { name: "确认分配" })).toBeEnabled());
  state.write.mockResolvedValue({ debtRepaid: "0", netCredit: "0" });
  await userEvent.click(dialog.getByRole("button", { name: "确认分配" }));
  expect(state.write.mock.calls[0][2].expectedVersion).toBe("2");
});

it.each(["分配资源", "回收资源"] as const)(
  "keeps the original UNKNOWN %s command frozen across directory refresh",
  async (action) => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    state.write.mockRejectedValueOnce(new MemberResourceError(502, "RESULT_UNVERIFIED", "unknown"))
      .mockResolvedValue({ debtRepaid: "0", netCredit: "0" });
    const dialog = await openTransfer(action, client);
    await userEvent.type(dialog.getByRole("textbox", { name: /本次期数/ }), "1");
    await waitFor(() => expect(dialog.getByRole("button", { name: /确认/ })).toBeEnabled());
    await userEvent.click(dialog.getByRole("button", { name: /确认/ }));
    await dialog.findByText(/操作结果未确认/);
    const original = state.write.mock.calls[0];
    state.read.mockResolvedValue(directory({ ...member, periods: { ...position, free: "1", version: "2" } }));
    await act(() => client.invalidateQueries({ queryKey: directoryKey }));
    await userEvent.click(dialog.getByRole("button", { name: "核验原操作" }));
    expect(state.write).toHaveBeenCalledTimes(2);
    expect(state.write.mock.calls[1]).toEqual(original);
    expect(original[2].expectedVersion).toBe("1");
  },
);

it.each(["store_renewal_period", "data_row"])(
  "shows an allocation %s read failure and retries only the scoped balance read",
  async (resourceType) => {
    let resolve!: (value: ReturnType<typeof commercialResourcesFixture>) => void;
    state.balance.mockRejectedValueOnce(new Error("balance unavailable"))
      .mockImplementationOnce(() => new Promise<ReturnType<typeof commercialResourcesFixture>>(ok => { resolve = ok; }));
    const dialog = await openTransfer("分配资源");
    await userEvent.selectOptions(dialog.getByRole("combobox", { name: "资源类型" }), resourceType);
    if (resourceType === "store_renewal_period") {
      await userEvent.type(dialog.getByRole("textbox", { name: /本次期数/ }), "1");
    }
    await dialog.findByText("管理员可用余额读取失败，请重试。");
    expect(dialog.queryByText(/管理员可用 正在读取/)).not.toBeInTheDocument();
    expect(dialog.getByRole("button", { name: "确认分配" })).toBeDisabled();
    const reads = state.balance.mock.calls.length;
    await userEvent.click(dialog.getByRole("button", { name: "重新读取管理员余额" }));
    expect(state.balance).toHaveBeenCalledTimes(reads + 1);
    expect(state.balance.mock.calls.at(-1)).toEqual(["actor", "org-1", expect.any(AbortSignal)]);
    expect(dialog.getByRole("status")).toHaveTextContent("正在读取管理员可用余额");
    expect(dialog.queryByRole("button", { name: "重新读取管理员余额" })).not.toBeInTheDocument();
    expect(dialog.getByRole("button", { name: "确认分配" })).toBeDisabled();
    await act(async () => resolve(commercialResourcesFixture("org-1")));
    await waitFor(() => expect(dialog.queryByRole("status")).not.toBeInTheDocument());
    if (resourceType === "store_renewal_period") {
      expect(dialog.getByRole("textbox", { name: /本次期数/ })).toHaveValue("1");
      await waitFor(() => expect(dialog.getByRole("button", { name: "确认分配" })).toBeEnabled());
    }
    expect(state.write).not.toHaveBeenCalled();
  },
);

it.each(["store_renewal_period", "data_row"])(
  "shows a missing %s allocation balance as unknown with a read retry",
  async (resourceType) => {
    state.balance.mockResolvedValue({ ...commercialResourcesFixture("org-1"), resources: [] });
    const dialog = await openTransfer("分配资源");
    await userEvent.selectOptions(dialog.getByRole("combobox", { name: "资源类型" }), resourceType);
    await dialog.findByText("当前未取得管理员可用余额，请重新读取。");
    expect(dialog.queryByText(/管理员可用 0|管理员可用 正在读取/)).not.toBeInTheDocument();
    expect(dialog.getByRole("button", { name: "重新读取管理员余额" })).toBeEnabled();
    expect(dialog.getByRole("button", { name: "确认分配" })).toBeDisabled();
    expect(state.write).not.toHaveBeenCalled();
  },
);

it.each(["store_renewal_period", "data_row"].flatMap(resourceType =>
  ["recorded", "not_recorded"].map(resourceState => ({ resourceType, resourceState })),
))(
  "keeps $resourceType $resourceState zero distinct from allocation read errors",
  async ({ resourceType, resourceState }) => {
    const resources = commercialResourcesFixture("org-1");
    state.balance.mockResolvedValue({
      ...resources,
      resources: resources.resources.map(entry => entry.resource_type !== resourceType ? entry : {
        ...entry, state: resourceState,
        ...Object.fromEntries(["available", "allocated", "reserved", "consumed", "debt"].map(key =>
          [key, resourceState === "recorded" ? "0" : null],
        )),
        updated_at: resourceState === "recorded" ? resources.observed_at : null,
      }),
    });
    const dialog = await openTransfer("分配资源");
    await userEvent.selectOptions(dialog.getByRole("combobox", { name: "资源类型" }), resourceType);
    await dialog.findByText(/管理员可用 0/);
    expect(dialog.queryByRole("button", { name: "重新读取管理员余额" })).not.toBeInTheDocument();
    expect(dialog.getByRole("button", { name: "确认分配" })).toBeDisabled();
    expect(state.write).not.toHaveBeenCalled();
  },
);

it("allows reclaim from the member holding when the administrator balance read fails", async () => {
  state.balance.mockRejectedValue(new Error("balance unavailable"));
  state.write.mockResolvedValue({ debtRepaid: "0", netCredit: "1" });
  const dialog = await openTransfer("回收资源");
  await userEvent.type(dialog.getByRole("textbox", { name: /本次期数/ }), "1");
  await waitFor(() => expect(dialog.getByRole("button", { name: "确认回收" })).toBeEnabled());
  expect(dialog.queryByText(/管理员可用余额读取失败/)).not.toBeInTheDocument();
  await userEvent.click(dialog.getByRole("button", { name: "确认回收" }));
  expect(state.write).toHaveBeenCalledTimes(1);
  expect(state.write.mock.calls[0][2]).toEqual({
    resourceType: "store_renewal_period", action: "reclaim", quantity: "1", expectedVersion: "1",
  });
});

it("does not change or replay an UNKNOWN allocation when retrying the balance read", async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  state.write.mockRejectedValueOnce(new MemberResourceError(502, "RESULT_UNVERIFIED", "unknown"))
    .mockResolvedValue({ debtRepaid: "0", netCredit: "0" });
  const dialog = await openTransfer("分配资源", client);
  await userEvent.type(dialog.getByRole("textbox", { name: /本次期数/ }), "1");
  await waitFor(() => expect(dialog.getByRole("button", { name: "确认分配" })).toBeEnabled());
  await userEvent.click(dialog.getByRole("button", { name: "确认分配" }));
  await dialog.findByText(/操作结果未确认/);
  const original = state.write.mock.calls[0];
  const pending = { ...sessionStorage };
  state.balance.mockRejectedValue(new Error("balance unavailable"));
  await act(() => client.invalidateQueries({ queryKey: ["workbench", "commercial-resources", "actor", "org-1"] }));
  await dialog.findByText("管理员可用余额读取失败，请重试。");
  state.balance.mockResolvedValue(commercialResourcesFixture("org-1"));
  await userEvent.click(dialog.getByRole("button", { name: "重新读取管理员余额" }));
  await waitFor(() => expect(dialog.queryByText("管理员可用余额读取失败，请重试。")).not.toBeInTheDocument());
  expect({ ...sessionStorage }).toEqual(pending);
  expect(state.write).toHaveBeenCalledTimes(1);
  await userEvent.click(dialog.getByRole("button", { name: "核验原操作" }));
  expect(state.write.mock.calls[1]).toEqual(original);
});
