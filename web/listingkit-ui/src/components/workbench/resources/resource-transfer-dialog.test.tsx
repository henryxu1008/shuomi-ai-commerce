import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { TransferDialog } from "./resource-transfer-dialog";
import type { MemberDataQuote } from "@/lib/api/member-resources";

const state = vi.hoisted(() => ({ quote: vi.fn(), write: vi.fn() }));
vi.mock("@/lib/api/member-resources", async original => ({
  ...await original<typeof import("@/lib/api/member-resources")>(),
  getMemberDataPrices: vi.fn().mockResolvedValue({ organizationId: "org-1", offers: [
    { offerId: "data", pricingVersion: "v1", currency: "CNY", unitPriceMinor: "10", minQuantity: "1", maxQuantity: "10000" },
    { offerId: "other", pricingVersion: "v2", currency: "CNY", unitPriceMinor: "20", minQuantity: "1", maxQuantity: "10000" },
  ] }),
  quoteMemberData: state.quote, transferMemberResource: state.write,
}));
vi.mock("@/lib/api/commercial-billing", () => ({
  getCommercialResources: vi.fn().mockResolvedValue({ organization_id: "org-1", resources: [{ resource_type: "data_row", state: "recorded", available: "2000" }] }),
}));
const position = { free: "0", reserved: "0", consumed: "0", version: "3", recorded: true };
const member = { memberId: "member-1", userId: "member-user", displayName: "成员 A", loginName: "member@example.test", state: "active", roles: [], permissions: [], storeCount: "0", periods: position, dataRows: position };
const scope = { expectedUserId: "admin", expectedOrganizationId: "org-1" };
const quote: MemberDataQuote = { organizationId: "org-1", quoteId: "100-yuan", quantity: "1000", unitPriceMinor: "10", pricingVersion: "v1", currency: "CNY", expiresAt: "2099-01-01T00:00:00Z", amountMinor: "10000", remainderMinor: "0" };
let client: QueryClient;
beforeEach(() => {
  sessionStorage.clear(); state.quote.mockReset(); state.write.mockReset().mockResolvedValue({ netCredit: "0", debtRepaid: "0" });
  HTMLDialogElement.prototype.showModal = function () { this.open = true; };
  HTMLDialogElement.prototype.close = function () { this.open = false; };
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
});
afterEach(() => { cleanup(); client.clear(); });
async function open() {
  render(<QueryClientProvider client={client}><TransferDialog member={member} action="allocate" scope={scope} onClose={vi.fn()} onChanged={vi.fn().mockResolvedValue(undefined)} /></QueryClientProvider>);
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
