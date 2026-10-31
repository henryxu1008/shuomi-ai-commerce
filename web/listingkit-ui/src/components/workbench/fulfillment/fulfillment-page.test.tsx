import { beforeEach, afterEach, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FulfillmentPage } from "./fulfillment-page";
import type { FulfillmentInput } from "@/lib/api/fulfillment";
import { saveFulfillmentIntent } from "@/lib/api/fulfillment-intent";
vi.mock("next/navigation", () => ({
  useSearchParams: () =>
    new URLSearchParams("storeId=store&sourceId=original&kind=platform_supply"),
}));
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
        return Response.json({ demand: null });
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
          id: "original",
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
afterEach(() => vi.unstubAllGlobals());
function mount() {
  return render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <FulfillmentPage />
    </QueryClientProvider>,
  );
}
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
