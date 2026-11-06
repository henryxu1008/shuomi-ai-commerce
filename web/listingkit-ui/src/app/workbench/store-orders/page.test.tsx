import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import StoreOrdersPage from "./page";

vi.mock("server-only", () => ({}));
vi.mock("@/components/workbench/store-observations/observation-page", () => ({
  ObservationPage: () => <p>平台订单同步记录</p>,
}));
const state = vi.hoisted(() => ({ org: "org", available: true, failure: false }));
vi.mock("@/components/providers/workbench-context-provider", () => ({
  useWorkbenchContext: () => ({
    effectiveOrganization: { id: state.org }, user: { id: "user" },
    permissions: ["workbench.fulfillment.read"],
  }),
}));
vi.mock("@/lib/api/workbench-stores", () => ({
  getWorkbenchStore: vi.fn(async () => ({ id: "store", name: "合成托管店铺", connectionStatus: "connected" })),
}));
const reference = {
  synthetic: true, title: "采购备货交仓案例", organizationId: "org",
  storeId: "store", sourceId: "TRY-MANAGED-CASE-001", kind: "platform_supply",
};
const commands: string[] = [];
const detail = {
  name: "SO-CASE", source: { organization: "org", store: "store", source_id: reference.sourceId, source_kind: "platform_supply" },
  movement_kind: "owned_transfer",
  lines: [{ line: "line", source_line: "sku", item_code: "ITEM", uom: "Nos", qty: 6, delivered_qty: 0, reserved_qty: 4, ordered_qty: 6, reserve_budget: 4, purchase_remaining: 2, reservable_qty: 0 }],
  purchases: [{ name: "PO-CASE", supplier: "Synthetic Supplier", external_order: "SYN-EXTERNAL", delivery_date: "2026-11-02", status: "To Receive and Bill", currency: "USD", receipts: [{ name: "PR-CASE", date: "2026-11-01", lines: [{ purchase_line: "po-line", accepted_qty: 4, rejected_qty: 0, short_qty: 2 }] }], has_more_receipts: false, lines: [{ line: "po-line", demand_line: "line", item_code: "ITEM", uom: "Nos", qty: 6, received_qty: 4, rate: 2 }] }],
  deliveries: [], operations: [], has_more_purchases: false, has_more_deliveries: false, has_more_operations: false,
};
beforeEach(() => {
  detail.purchases[0].lines[0].received_qty = 4;
  detail.purchases[0].has_more_receipts = false;
  detail.purchases[0].receipts[0].lines[0].rejected_qty = 0;
  state.org = "org"; state.available = true; state.failure = false; commands.length = 0;
  vi.stubEnv("LISTINGKIT_ORDER_BUSINESS_CASE", JSON.stringify(reference));
  vi.stubGlobal("fetch", vi.fn(async (_url: string, init?: RequestInit) => {
    if (!init?.body) return Response.json({ available: state.available, addressConfigured: true, realPlatformValidation: "NOT_RUN" });
    const input = JSON.parse(String(init.body)); commands.push(input.action);
    if (state.failure) return Response.json({ code: "FULFILLMENT_FORBIDDEN" }, { status: 403 });
    if (input.action === "find_demand") return Response.json({ demand: "SO-CASE" });
    if (input.action === "details") return Response.json(detail);
    throw new Error("Opening a case must not mutate a business fact");
  }));
});
afterEach(() => { vi.unstubAllEnvs(); vi.unstubAllGlobals(); });
async function mount() {
  return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    {await StoreOrdersPage()}
  </QueryClientProvider>);
}

it("opens the saved case from the order menu using native facts and existing steps without executing it", async () => {
  await mount();
  expect(await screen.findByText("采购备货交仓案例")).toBeVisible();
  expect(await screen.findByText("PO-CASE")).toBeVisible();
  expect(screen.getByText("采购 6 件 · 合格收货 4 件")).toBeVisible();
  const next = new URL(screen.getByRole("link", { name: "打开采购与收货" }).getAttribute("href")!, "https://app.test");
  expect(next.pathname).toBe("/workbench/store-orders/fulfill");
  expect(Object.fromEntries(next.searchParams)).toMatchObject({ storeId: "store", sourceId: reference.sourceId, kind: "platform_supply", step: "purchase" });
  expect(screen.getByRole("link", { name: "平台订单记录" })).toHaveAttribute("href", "/workbench/store-orders/records");
  expect(commands).toEqual(["find_demand", "details"]);
  expect(screen.getByRole("heading", { name: "订单业务如何流转" })).toBeVisible();
  expect(screen.getByRole("link", { name: /查看订单与需求/ })).toHaveAttribute("href", "/workbench/store-orders/fulfill?step=source");
  expect(screen.getByRole("link", { name: /进入采购与收货/ })).toHaveAttribute("href", "/workbench/store-orders/fulfill?step=purchase");
  expect(screen.getByRole("link", { name: /进入装箱与发货/ })).toHaveAttribute("href", "/workbench/store-orders/fulfill?step=stock");
  expect(screen.getByRole("link", { name: /查看交接与物流/ })).toHaveAttribute("href", "/workbench/store-orders/fulfill?step=logistics");
});

it("does not expose another organization's case or request its facts", async () => {
  state.org = "other-org";
  await mount();
  expect(await screen.findByRole("heading", { name: "统一管理订单、采购与发货，让履约流程更清晰", level: 1 })).toBeVisible();
  expect(screen.queryByText("采购备货交仓案例")).not.toBeInTheDocument();
  expect(commands).toEqual([]);
});

it("keeps rejected arrival separate from qualified receipts", async () => {
  detail.purchases[0].lines[0].received_qty = 6;
  detail.purchases[0].receipts[0].lines[0].rejected_qty = 2;
  await mount();
  expect(await screen.findByText("PO-CASE")).toBeVisible();
  expect(screen.getByText("采购 6 件 · 合格收货 4 件")).toBeVisible();
  expect(screen.getByText("累计到货 6 件（含拒收）")).toBeVisible();
});

it("does not infer a qualified total from an incomplete receipt page", async () => {
  detail.purchases[0].has_more_receipts = true;
  await mount();
  expect(await screen.findByText("合格收货数量需翻页核对")).toBeVisible();
  expect(screen.queryByText("采购 6 件 · 合格收货 4 件")).not.toBeInTheDocument();
});

it("does not put synthetic cases into an unconfigured installation", async () => {
  vi.stubEnv("LISTINGKIT_ORDER_BUSINESS_CASE", "");
  await mount();
  expect(await screen.findByRole("heading", { name: "统一管理订单、采购与发货，让履约流程更清晰", level: 1 })).toBeVisible();
  expect(screen.queryByText("采购备货交仓案例")).not.toBeInTheDocument();
  expect(commands).toEqual([]);
});

it("keeps the four business entries disabled until the existing capability is available", async () => {
  state.available = false;
  await mount();
  expect(await screen.findByText("当前环境未开放订单履约")).toBeVisible();
  expect(screen.queryByRole("link", { name: /查看订单与需求|进入采购与收货|进入装箱与发货|查看交接与物流/ })).not.toBeInTheDocument();
  expect(screen.queryByText("采购备货交仓案例")).not.toBeInTheDocument();
  expect(commands).toEqual([]);
});

it("reports a case read failure without substituting fixture quantities", async () => {
  state.failure = true;
  await mount();
  expect(await screen.findByText("业务案例暂时无法读取")).toBeVisible();
  expect(screen.queryByText("采购 6 件 · 合格收货 4 件")).not.toBeInTheDocument();
  expect(commands).toEqual(["find_demand"]);
});
