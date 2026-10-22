import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { TransferDialog } from "./resource-transfer-dialog";
import { MemberResourceError, type MemberDataQuote } from "@/lib/api/member-resources";
import { QueryProvider } from "@/components/providers/query-provider";
import { useEffect } from "react";

const state = vi.hoisted(() => ({ prices: vi.fn(), quote: vi.fn(), write: vi.fn(), balance: vi.fn() }));
const offers = [
  { offerId: "data", pricingVersion: "v1", currency: "CNY", unitPriceMinor: "10", minQuantity: "1", maxQuantity: "10000" },
  { offerId: "other", pricingVersion: "v2", currency: "CNY", unitPriceMinor: "20", minQuantity: "1", maxQuantity: "10000" },
];
vi.mock("@/lib/api/member-resources", async original => ({
  ...await original<typeof import("@/lib/api/member-resources")>(),
  getMemberDataPrices: state.prices,
  quoteMemberData: state.quote, transferMemberResource: state.write,
}));
vi.mock("@/lib/api/commercial-billing", () => ({
  getCommercialResources: state.balance,
}));
const position = { free: "0", reserved: "0", consumed: "0", version: "3", recorded: true };
const member = { memberId: "member-1", userId: "member-user", displayName: "成员 A", loginName: "member@example.test", state: "active", roles: [], permissions: [], storeCount: "0", periods: position, dataRows: position };
const scope = { expectedUserId: "admin", expectedOrganizationId: "org-1" };
const quote: MemberDataQuote = { organizationId: "org-1", quoteId: "100-yuan", quantity: "1000", unitPriceMinor: "10", pricingVersion: "v1", currency: "CNY", expiresAt: "2099-01-01T00:00:00Z", amountMinor: "10000", remainderMinor: "0" };
let client: QueryClient;
beforeEach(() => {
  sessionStorage.clear(); state.quote.mockReset(); state.write.mockReset().mockResolvedValue({ netCredit: "0", debtRepaid: "0" });
  state.prices.mockReset().mockResolvedValue({ organizationId: "org-1", offers });
  state.balance.mockReset().mockResolvedValue({ organization_id: "org-1", resources: [{ resource_type: "data_row", state: "recorded", available: "2000" }] });
  HTMLDialogElement.prototype.showModal = function () { this.open = true; };
  HTMLDialogElement.prototype.close = function () { this.open = false; };
});
afterEach(() => { cleanup(); client.clear(); });
function CaptureClient() {
  const current = useQueryClient();
  useEffect(() => { client = current; }, [current]);
  return null;
}
function mount() {
  render(<QueryProvider><CaptureClient /><TransferDialog member={member} memberAvailable action="allocate" scope={scope} onClose={vi.fn()} onChanged={vi.fn().mockResolvedValue(undefined)} /></QueryProvider>);
}
async function open() {
  mount();
  await userEvent.selectOptions(screen.getByRole("combobox", { name: "资源类型" }), "data_row");
  await screen.findByRole("option", { name: "data · 10 分/条" });
  await userEvent.type(screen.getByRole("textbox", { name: "分配金额（元）" }), "100");
}

it("locks amount, price and resource type until the matching conversion completes", async () => {
  let resolve!: (value: MemberDataQuote) => void;
  state.quote.mockImplementation(() => new Promise<MemberDataQuote>(ok => { resolve = ok; }));
  await open();
  await userEvent.click(screen.getByRole("button", { name: "换算数据条数" }));
  const amount = screen.getByRole("textbox", { name: "分配金额（元）" });
  expect(amount).toBeDisabled();
  expect(screen.getByRole("combobox", { name: "数据价格" })).toBeDisabled();
  expect(screen.getByRole("combobox", { name: "资源类型" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "确认分配" })).toBeDisabled();
  expect(state.quote).toHaveBeenCalledWith(scope, "data", "10000");
  await act(async () => resolve(quote));
  expect(amount).toBeEnabled(); expect(amount).toHaveValue("100");
  await waitFor(() => expect(screen.getByRole("button", { name: "确认分配" })).toBeEnabled());
  await userEvent.click(screen.getByRole("button", { name: "确认分配" }));
  expect(state.write).toHaveBeenCalledOnce();
  expect(state.write).toHaveBeenCalledWith(scope, member.memberId, { resourceType: "data_row", action: "allocate", quantity: "1000", quoteId: "100-yuan", expectedVersion: "3" }, expect.any(String));
});

it.each(["amount", "price", "type"])("invalidates a completed quote when %s changes", async field => {
  state.quote.mockResolvedValue(quote); await open();
  await userEvent.click(screen.getByRole("button", { name: "换算数据条数" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "确认分配" })).toBeEnabled());
  if (field === "amount") {
    const amount = screen.getByRole("textbox", { name: "分配金额（元）" });
    await userEvent.clear(amount); await userEvent.type(amount, "10");
  } else if (field === "price") await userEvent.selectOptions(screen.getByRole("combobox", { name: "数据价格" }), "other");
  else {
    await userEvent.selectOptions(screen.getByRole("combobox", { name: "资源类型" }), "store_renewal_period");
    await userEvent.selectOptions(screen.getByRole("combobox", { name: "资源类型" }), "data_row");
  }
  expect(screen.getByRole("button", { name: "确认分配" })).toBeDisabled();
  expect(state.write).not.toHaveBeenCalled();
});

it("preserves a completed data quote when retrying only the administrator balance read", async () => {
  state.quote.mockResolvedValue(quote);
  await open();
  await userEvent.click(screen.getByRole("button", { name: "换算数据条数" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "确认分配" })).toBeEnabled());
  state.balance.mockRejectedValue(new Error("balance unavailable"));
  await act(() => client.invalidateQueries({ queryKey: ["workbench", "commercial-resources", "admin", "org-1"] }));
  await screen.findByText("管理员可用余额读取失败，请重试。");
  expect(screen.getByRole("button", { name: "确认分配" })).toBeDisabled();
  state.balance.mockResolvedValue({ organization_id: "org-1", resources: [{ resource_type: "data_row", state: "recorded", available: "2000" }] });
  await userEvent.click(screen.getByRole("button", { name: "重新读取管理员余额" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "确认分配" })).toBeEnabled());
  expect(screen.getByRole("textbox", { name: "分配金额（元）" })).toHaveValue("100");
  expect(screen.getByText(/预计换算：1000 条/)).toBeInTheDocument();
  expect(state.quote).toHaveBeenCalledTimes(1);
  expect(state.write).not.toHaveBeenCalled();
});

const currentOffer = { ...offers[0], offerId: "current", pricingVersion: "v3", unitPriceMinor: "30" };
async function rereadPrices(nextOffers: typeof offers) {
  await userEvent.selectOptions(screen.getByRole("combobox", { name: "资源类型" }), "store_renewal_period");
  state.prices.mockResolvedValue({ organizationId: "org-1", offers: nextOffers });
  await userEvent.selectOptions(screen.getByRole("combobox", { name: "资源类型" }), "data_row");
  await screen.findByRole("option", { name: "current · 30 分/条" });
}

it("requires a current price selection before quoting when rereading removes the selected offer", async () => {
  state.quote.mockResolvedValue({ ...quote, quantity: "333", unitPriceMinor: "30", pricingVersion: "v3", remainderMinor: "10" });
  await open();
  expect(client.getDefaultOptions().queries?.refetchOnWindowFocus).toBe(false);
  await userEvent.selectOptions(screen.getByRole("combobox", { name: "数据价格" }), "other");
  await rereadPrices([currentOffer]);
  expect(screen.queryByRole("option", { name: "other · 20 分/条" })).not.toBeInTheDocument();
  expect(screen.getByRole("combobox", { name: "数据价格" })).toHaveValue("");
  expect(screen.getByRole("button", { name: "换算数据条数" })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "换算数据条数" }));
  expect(state.quote).not.toHaveBeenCalled();
  await userEvent.selectOptions(screen.getByRole("combobox", { name: "数据价格" }), "current");
  await userEvent.click(screen.getByRole("button", { name: "换算数据条数" }));
  expect(state.quote).toHaveBeenCalledExactlyOnceWith(scope, "current", "10000");
  expect(await screen.findByText(/预计换算：333 条/)).toBeInTheDocument();
  expect(state.write).not.toHaveBeenCalled();
});

it("retains an explicitly selected offer that is still present after the directory is reordered", async () => {
  state.quote.mockResolvedValue(quote);
  await open();
  await userEvent.selectOptions(screen.getByRole("combobox", { name: "数据价格" }), "other");
  await rereadPrices([currentOffer, offers[1]]);
  expect(screen.getByRole("combobox", { name: "数据价格" })).toHaveValue("other");
  await userEvent.click(screen.getByRole("button", { name: "换算数据条数" }));
  expect(state.quote).toHaveBeenCalledExactlyOnceWith(scope, "other", "10000");
  expect(state.write).not.toHaveBeenCalled();
});

it("distinguishes failed price reads and retries in place without quoting or changing the amount", async () => {
  state.prices.mockRejectedValueOnce(new Error("prices unavailable"));
  state.quote.mockResolvedValue(quote);
  mount();
  await userEvent.selectOptions(screen.getByRole("combobox", { name: "资源类型" }), "data_row");
  await screen.findByText("数据价格读取失败，请重试。");
  expect(screen.queryByText("当前未配置可用的数据价格，不能开始新换算。")).not.toBeInTheDocument();
  expect(screen.getByRole("combobox", { name: "数据价格" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "换算数据条数" })).toBeDisabled();
  await userEvent.type(screen.getByRole("textbox", { name: "分配金额（元）" }), "100");
  let resolve!: (value: { organizationId: string; offers: typeof offers }) => void;
  state.prices.mockImplementationOnce(() => new Promise(ok => { resolve = ok; }));
  await userEvent.click(screen.getByRole("button", { name: "重新读取数据价格" }));
  expect(screen.getByText("正在读取数据价格…")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "重新读取数据价格" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "换算数据条数" })).toBeDisabled();
  expect(state.prices).toHaveBeenCalledTimes(2);
  await act(async () => resolve({ organizationId: "org-1", offers }));
  await waitFor(() => expect(screen.getByRole("button", { name: "换算数据条数" })).toBeEnabled());
  expect(screen.queryByText("数据价格读取失败，请重试。")).not.toBeInTheDocument();
  expect(screen.getByRole("textbox", { name: "分配金额（元）" })).toHaveValue("100");
  expect(state.quote).not.toHaveBeenCalled();
  expect(state.write).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole("button", { name: "换算数据条数" }));
  expect(state.quote).toHaveBeenCalledExactlyOnceWith(scope, "data", "10000");
});

it("keeps a successfully empty price directory distinct from a read failure", async () => {
  state.prices.mockResolvedValue({ organizationId: "org-1", offers: [] });
  mount();
  await userEvent.selectOptions(screen.getByRole("combobox", { name: "资源类型" }), "data_row");
  await screen.findByText("当前未配置可用的数据价格，不能开始新换算。");
  expect(screen.queryByText("数据价格读取失败，请重试。")).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "换算数据条数" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "确认分配" })).toBeDisabled();
  expect(state.quote).not.toHaveBeenCalled();
  expect(state.write).not.toHaveBeenCalled();
});

const pricesKey = ["workbench", "admin", "org-1", "member-data-prices"];
it("blocks stale cached prices during failed rereads while preserving a valid immutable quote", async () => {
  state.quote.mockResolvedValue(quote);
  await open();
  await userEvent.click(screen.getByRole("button", { name: "换算数据条数" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "确认分配" })).toBeEnabled());
  state.prices.mockRejectedValueOnce(new Error("reread failed"));
  await act(() => client.invalidateQueries({ queryKey: pricesKey }));
  await screen.findByText("数据价格读取失败，请重试。");
  expect(screen.getByRole("button", { name: "换算数据条数" })).toBeDisabled();
  expect(screen.getByRole("combobox", { name: "数据价格" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "确认分配" })).toBeEnabled();
  let resolve!: (value: { organizationId: string; offers: typeof offers }) => void;
  state.prices.mockImplementationOnce(() => new Promise(ok => { resolve = ok; }));
  await userEvent.click(screen.getByRole("button", { name: "重新读取数据价格" }));
  expect(screen.getByRole("button", { name: "换算数据条数" })).toBeDisabled();
  await act(async () => resolve({ organizationId: "org-1", offers: [currentOffer] }));
  await screen.findByRole("option", { name: "current · 30 分/条" });
  expect(screen.getByText(/预计换算：1000 条/)).toHaveTextContent("价格版本 v1");
  expect(screen.getByRole("button", { name: "确认分配" })).toBeEnabled();
  expect(screen.getByRole("textbox", { name: "分配金额（元）" })).toHaveValue("100");
  expect(state.quote).toHaveBeenCalledTimes(1);
  expect(state.write).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole("button", { name: "确认分配" }));
  expect(state.write).toHaveBeenCalledWith(scope, member.memberId, { resourceType: "data_row", action: "allocate", quantity: "1000", quoteId: "100-yuan", expectedVersion: "3" }, expect.any(String));
});

it("keeps an UNKNOWN command frozen and disables price retry until the original command is recovered", async () => {
  state.quote.mockResolvedValue(quote);
  state.write.mockRejectedValueOnce(new MemberResourceError(502, "RESULT_UNVERIFIED", "unknown"));
  await open();
  await userEvent.click(screen.getByRole("button", { name: "换算数据条数" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "确认分配" })).toBeEnabled());
  await userEvent.click(screen.getByRole("button", { name: "确认分配" }));
  await screen.findByRole("button", { name: "核验原操作" });
  const original = state.write.mock.calls[0];
  const pending = { ...sessionStorage };
  state.prices.mockRejectedValueOnce(new Error("prices unavailable"));
  await act(() => client.invalidateQueries({ queryKey: pricesKey }));
  const reads = state.prices.mock.calls.length;
  const retry = await screen.findByRole("button", { name: "重新读取数据价格" });
  expect(retry).toBeDisabled();
  await userEvent.click(retry);
  expect(state.prices).toHaveBeenCalledTimes(reads);
  expect(state.quote).toHaveBeenCalledTimes(1);
  expect(state.write).toHaveBeenCalledTimes(1);
  expect({ ...sessionStorage }).toEqual(pending);
  await userEvent.click(screen.getByRole("button", { name: "核验原操作" }));
  expect(state.write).toHaveBeenCalledTimes(2);
  expect(state.write.mock.calls[1]).toEqual(original);
});
