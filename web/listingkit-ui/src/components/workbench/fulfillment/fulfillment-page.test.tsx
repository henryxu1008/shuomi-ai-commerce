import { beforeEach, afterEach, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FulfillmentPage } from "./fulfillment-page";
import type { FulfillmentInput } from "@/lib/api/fulfillment";
import { saveFulfillmentIntent } from "@/lib/api/fulfillment-intent";
import { ConsoleNavigation } from "../console/console-navigation";
const location = vi.hoisted(() => ({ search: "", hasDemand: false, platformSuccess: false }));
vi.mock("next/navigation", async () => {
  const { useSyncExternalStore } = await import("react");
  return { useSearchParams: () => new URLSearchParams(useSyncExternalStore(
    notify => { window.addEventListener("popstate", notify); return () => window.removeEventListener("popstate", notify); },
    () => location.search,
  )) };
});
vi.mock("@/components/providers/workbench-context-provider", () => ({
  useWorkbenchContext: () => ({
    effectiveOrganization: { id: "org" },
    user: { id: "user" },
    permissions: [
      "workbench.fulfillment.read",
      "workbench.fulfillment.stock",
    ],
  }),
}));
vi.mock("@/lib/api/workbench-stores", () => ({
  listWorkbenchStores: async () => ({
    items: [{ id: "store", name: "Synthetic Store" }],
    pagination: { total: 1 },
  }),
}));
const calls: FulfillmentInput[] = [];
beforeEach(() => {
  location.search = "storeId=store&sourceId=original&kind=platform_supply";
  location.hasDemand = false; location.platformSuccess = false;
  window.history.replaceState(null, "", `/workbench/store-orders/fulfill?${location.search}`);
  const replace = window.history.replaceState.bind(window.history);
  // Next's documented native-history integration notifies useSearchParams.
  vi.spyOn(window.history, "replaceState").mockImplementation((data, unused, url) => {
    replace(data, unused, url);
    location.search = window.location.search;
    window.dispatchEvent(new PopStateEvent("popstate"));
  });
  localStorage.clear();
  calls.length = 0;
  vi.stubGlobal("navigator", {
    locks: {
      request: async (_key: string, fn: () => Promise<boolean>) => fn(),
    },
  });
  vi.stubGlobal(
    "fetch",
    vi.fn(async (_url: string, init?: RequestInit) => {
      if (!init?.body)
        return Response.json({
          available: true,
          addressConfigured: false,
          realPlatformValidation: "NOT_RUN",
        });
      const input = JSON.parse(String(init.body)) as FulfillmentInput;
      calls.push(input);
      if (input.action === "find_demand")
        return Response.json({ demand: location.hasDemand ? "SO" : null });
      if (input.action === "details") return Response.json({
        name: "SO", source: { organization: "org", store: "store", source_id: "original", source_kind: "platform_supply" }, movement_kind: "owned_transfer",
        lines: [{ line: "line", source_line: "sku", item_code: "ITEM", uom: "Nos", qty: 6, delivered_qty: 0, reserved_qty: 6, ordered_qty: 6, reserve_budget: 6, purchase_remaining: 0, reservable_qty: 0 }],
        purchases: [], deliveries: [{ name: "DN", docstatus: 0, package_id: "SYN-PACKAGE", lines: [{ demand_line: "line", qty: 6 }] }],
        operations: location.platformSuccess ? [{ operation: "11111111-1111-4111-8111-111111111111", purpose: "supply_shipping", delivery: "DN", selection: {}, stage: "SUCCEEDED", target: "SYN-DELIVERY", revision: null, result: { shipping_id: "SYN-DELIVERY" } }] : [],
        has_more_purchases: false, has_more_deliveries: false, has_more_operations: false,
      });
      if (input.action === "dispatch") return Response.json({operation: input.operation_id, stage: "SUCCEEDED", target: "DN", revision: null, result: {}});
      if (input.action === "catalog")
        return Response.json({
          items: [],
          suppliers: [],
          currency: "USD",
          source_warehouse: "source",
          movement_kind: "owned_transfer",
        });
      if (input.action === "cancel_original")
        return Response.json({
          operation: input.operation_id,
          stage: "DEFINITIVE_NO_EFFECT",
          target: null,
          revision: null,
          result: { reason: "original_command_cancelled_before_commit" },
        });
      return Response.json({
        source: {
          kind: "platform_supply",
          id: input.source_id,
          market: "unknown",
          status: 2,
          stockMode: 0,
          printStatus: 0,
          logistics: null,
          selectedLogistics: 0,
          scene: 1,
          type: 2,
          countryMarket: null,
          storageId: "warehouse",
          rules: null,
          deadline: "2026-11-03T00:00:00Z",
          lines: [],
        },
        version: "a".repeat(64),
        eligible: false,
      });
    }),
  );
});
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });
function mount(navigation = false) {
  return render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      {navigation ? <ConsoleNavigation pathname="/workbench/store-orders/fulfill" ariaLabel="主导航" fulfillmentAvailable /> : null}
      <FulfillmentPage />
    </QueryClientProvider>,
  );
}
it.each([false,true])("records exactly the original delivery only after actual-handoff confirmation and platform success=%s", async platformSuccess => {
  location.hasDemand=true; location.platformSuccess=platformSuccess;
  location.search += "&step=logistics";
  mount();
  const submit=await screen.findByRole("button",{name:"确认记录实际交接"});
  await userEvent.click(submit);
  expect(calls.some(call=>call.action==="dispatch")).toBe(false);
  await userEvent.click(screen.getByRole("checkbox",{name:"我已核对该包裹及实际交接数量"}));
  await userEvent.click(submit);
  if(platformSuccess){
    await waitFor(()=>expect(calls.filter(call=>call.action==="dispatch")).toHaveLength(1));
    expect(calls.find(call=>call.action==="dispatch")).toMatchObject({store:"store",kind:"platform_supply",source_id:"original",payload:{demand:"SO",delivery:"DN",platform_operation:"11111111-1111-4111-8111-111111111111"}});
  }else{
    expect(submit).toBeDisabled();
    expect(calls.some(call=>call.action==="dispatch")).toBe(false);
  }
});
it("keeps the sidebar selected step in sync after switching a page tab", async () => {
  location.search += "&step=purchase";
  mount(true);
  await screen.findByRole("button", { name: "采购与收货" });
  await userEvent.click(screen.getByRole("button", { name: "库存与装箱" }));
  await waitFor(() => expect(new URLSearchParams(window.location.search).get("step")).toBe("stock"));
  expect(screen.getByRole("link", { name: "装箱与发货" })).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("link", { name: "采购与收货" })).not.toHaveAttribute("aria-current");
  expect(calls.every(call => ["source", "catalog", "find_demand"].includes(call.action))).toBe(true);
});
it("carries the confirmed new original into sidebar navigation while keeping unsubmitted lookup text local", async () => {
  mount(true);
  await screen.findByText("平台原单 · original");
  const input=screen.getByRole("textbox", { name: "平台原单号" });
  await userEvent.clear(input);
  await userEvent.type(input,"next-original");
  expect(new URL(screen.getByRole("link", { name: "装箱与发货" }).getAttribute("href")!,"https://test.invalid").searchParams.get("sourceId")).toBe("original");
  await userEvent.click(screen.getByRole("button", { name: "查看原单" }));
  await screen.findByText("平台原单 · next-original");
  expect(new URLSearchParams(window.location.search).get("sourceId")).toBe("next-original");
  expect(new URL(screen.getByRole("link", { name: "装箱与发货" }).getAttribute("href")!,"https://test.invalid").searchParams.get("sourceId")).toBe("next-original");
  expect(calls.every(call => ["source", "catalog", "find_demand"].includes(call.action))).toBe(true);
});
it("opens the selected case step without executing its operations", async () => {
  location.search += "&step=purchase";
  mount();
  expect(await screen.findByRole("button", { name: "采购与收货" })).toHaveAttribute("aria-current", "step");
  expect(calls.every(call => ["source", "catalog", "find_demand"].includes(call.action))).toBe(true);
});
it("does not execute acceptance or shipping while displaying or refreshing an original source", async () => {
  mount();
  await screen.findByText("平台原单 · original");
  await userEvent.click(screen.getByRole("button", { name: "刷新原单与回执" }));
  await waitFor(() =>
    expect(
      calls.filter((call) => call.action === "source").length,
    ).toBeGreaterThan(1),
  );
  expect(
    calls.every((call) =>
      ["source", "catalog", "find_demand"].includes(call.action),
    ),
  ).toBe(true);
});
it("restores the exact original intent without resending and cancels using that same operation key", async () => {
  const input: FulfillmentInput = {
    store: "store",
    kind: "platform_supply",
    source_id: "original",
    operation_id: "a182d2e0-6f91-4e71-8b7f-5df25b75fb11",
    action: "receive",
    payload: {
      purchase: "native-po",
      lines: [{ purchase_line: "row", qty: 2 }],
    },
  };
  expect(
    await saveFulfillmentIntent(
      { organizationId: "org", userId: "user" },
      input,
      null,
    ),
  ).toBe(true);
  mount();
  await screen.findByText("有一笔原操作待核实");
  expect(
    calls.some(
      (call) => call.action === "receive" || call.action === "cancel_original",
    ),
  ).toBe(false);
  await userEvent.click(
    screen.getByRole("button", { name: "核对并取消尚未生效的原操作" }),
  );
  await waitFor(() =>
    expect(
      calls.find((call) => call.action === "cancel_original")?.operation_id,
    ).toBe(input.operation_id),
  );
  await waitFor(() =>
    expect(screen.queryByText("有一笔原操作待核实")).toBeNull(),
  );
  expect(calls.some((call) => call.action === "receive")).toBe(false);
});
