import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { parseResourceEvents } from "@/lib/api/resource-events";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {useState,type ComponentProps} from "react";
import { afterEach, expect, it, vi } from "vitest";
import { commercialOverviewFixture } from "@/test/fixtures/commercial-overview";
import {
  CommercialOverviewView,
  UsageDetailsView,
} from "./commercial-module-views";
import { OrdersView as ControlledOrdersView, WalletView } from "./commercial-billing-views";
function OrdersView(props:Omit<ComponentProps<typeof ControlledOrdersView>,"form"|"onFormChange">){
 const [form,setForm]=useState({query:"",kind:"",status:"",from:"",until:""});
 return <ControlledOrdersView {...props} form={form} onFormChange={setForm}/>;
}
vi.mock("./wallet-topup", () => ({
  WalletTopUpEntry: () => <button disabled>充值钱包</button>,
}));

const scope = { userId: "reader", organizationId: "org-B" };
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

it("renders unified base and real service observations without a subscription gate", () => {
  render(<CommercialOverviewView data={commercialOverviewFixture()} />);
  expect(screen.getByText("基础方案")).toBeVisible();
  expect(screen.getByText("2 家生效中")).toBeVisible();
  expect(screen.getByRole("link", { name: "购买资源" })).toHaveAttribute(
    "href",
    "/workbench/plans/options",
  );
  expect(screen.queryByText(/¥168|未提供|专业版/)).not.toBeInTheDocument();
});
it("filters actual resource ledger rows on the server and preserves exact resource units", async () => {
  const onFilter = vi.fn();
  render(
    <UsageDetailsView
      scope={scope}
      page={{ organization_id: "org-B", items: [], next_cursor: null }}
      onFilter={onFilter}
      onNext={() => undefined}
    />,
  );
  expect(screen.getByText("暂无符合条件的用量记录。")).toBeVisible();
  await userEvent.click(screen.getByRole("button", { name: "AI 点数" }));
  expect(onFilter).toHaveBeenCalledWith({ resourceType: "ai_point" });
});

it("renders only real wallet data and delegates recharge to the configured payment entry", () => {
  render(
    <WalletView
      userId="reader"
      permissions={["listingkit_admin"]}
      wallet={{
        organization_id: "org-A",
        currency: "CNY",
        available_minor: "12000",
        reserved_minor: "3000",
        debt_minor: "0",
        lifetime_topup_minor: "30000",
        lifetime_spend_minor: "18000",
        version: "4",
        observed_at: "2026-09-23T10:00:00Z",
      }}
      entries={{
        organization_id: "org-A",
        items: [
          {
            entry_id: "entry-1",
            currency: "CNY",
            entry_type: "PURCHASE_COMMIT",
            available_delta_minor: "0",
            reserved_delta_minor: "-3000",
            debt_delta_minor: "0",
            available_after_minor: "12000",
            reserved_after_minor: "0",
            debt_after_minor: "0",
            order_id: "order-1",
            source_id: "commercial-order:order-1",
            occurred_at: "2026-09-23T10:00:00Z",
          },
        ],
        next_cursor: "",
      }}
      onNext={() => undefined}
    />,
  );
  expect(screen.getByText("¥120.00")).toBeVisible();
  expect(screen.getByText("¥30.00")).toBeVisible();
  expect(screen.getByRole("button", { name: "充值钱包" })).toBeDisabled();
  expect(
    screen.getByText(/有欠款时先偿债；渠道手续费由平台承担/),
  ).toBeVisible();
  expect(screen.getByRole("link", { name: "order-1" })).toHaveAttribute(
    "href",
    "/workbench/plans/orders/order-1",
  );
  expect(screen.queryByText("¥5,000.00")).not.toBeInTheDocument();
});

it("renders owner-backed orders and submits bounded search and date/type/status filters", async () => {
  const onFilter = vi.fn();
  render(
    <OrdersView
      summary={{
        organization_id: "org-A",
        currency: "CNY",
        from: "2026-08-24T10:00:00Z",
        until: "2026-09-23T10:00:00Z",
        spend_minor: "3000",
        store_renewal_spend_minor: "0",
        ai_point_spend_minor: "3000",
        data_row_spend_minor: "0",
        other_spend_minor: "0",
        observed_at: "2026-09-23T10:00:00Z",
      }}
      page={{
        organization_id: "org-A",
        items: [
          {
            order_id: "order-1",
            organization_id: "org-A",
            kind: "RESOURCE_PURCHASE",
            description: "AI 点数 × 3",
            quote_id: "quote-1",
            currency: "CNY",
            total_minor: "3000",
            status: "FULFILLED",
            product_kind: "AI_POINT",
            items: [
              {
                order_item_id: "item-1",
                product_kind: "AI_POINT",
                resource_type: "ai_point",
                resource_quantity: "3",
                amount_minor: "3000",
              },
            ],
            created_at: "2026-09-23T10:00:00Z",
            updated_at: "2026-09-23T10:00:00Z",
          },
        ],
        next_cursor: "next",
      }}
      onFilter={onFilter}
      onNext={() => undefined}
    />,
  );
  expect(screen.getAllByText("¥30.00")).toHaveLength(3);
  expect(screen.getByText("AI 点数 × 3")).toBeVisible();
  await userEvent.selectOptions(screen.getByLabelText("订单类型"), "STORAGE_CAPACITY_PURCHASE");
  await userEvent.click(screen.getByRole("button", { name: "筛选" }));
  expect(onFilter).toHaveBeenLastCalledWith({ query: "", kind: "STORAGE_CAPACITY_PURCHASE", status: "", from: "", until: "" });
  expect(screen.getByRole("link", { name: "详情" })).toHaveAttribute(
    "href",
    "/workbench/plans/orders/order-1",
  );
  await userEvent.type(screen.getByLabelText("搜索订单"), " renewal ");
  await userEvent.selectOptions(
    screen.getByLabelText("订单类型"),
    "RESOURCE_PURCHASE",
  );
  await userEvent.selectOptions(screen.getByLabelText("订单状态"), "FULFILLED");
  fireEvent.change(screen.getByLabelText("开始日期"), {
    target: { value: "2026-09-01" },
  });
  fireEvent.change(screen.getByLabelText("结束日期"), {
    target: { value: "2026-09-23" },
  });
  await userEvent.click(screen.getByRole("button", { name: "筛选" }));
  expect(onFilter).toHaveBeenCalledWith({
    query: "renewal",
    kind: "RESOURCE_PURCHASE",
    status: "FULFILLED",
    from: "2026-08-31T16:00:00.000Z",
    until: "2026-09-23T16:00:00.000Z",
  });
  expect(
    screen.getByText(/不提供购买、退款、发票开具或导出写操作/),
  ).toBeVisible();
});

it("restores applied ledger filters when a query remounts the view", () => {
  render(
    <UsageDetailsView
      scope={scope}
      page={{ organization_id: "org-B", items: [], next_cursor: null }}
      filters={{
        resourceType: "ai_point",
        from: "2026-09-01T00:00:00.000Z",
        until: "2026-10-01T00:00:00.000Z",
      }}
      onFilter={() => undefined}
      onNext={() => undefined}
    />,
  );
  expect(screen.getByRole("button", { name: "AI 点数" })).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByLabelText("开始日期（UTC）")).toHaveValue("2026-09-01");
  expect(screen.getByLabelText("结束日期（UTC）")).toHaveValue("2026-09-30");
});

it("preserves fractional AI points, resource units and the recorded balance on the row and in its details", async () => {
  Object.defineProperties(HTMLDialogElement.prototype, {
    showModal: { configurable: true, value(this: HTMLDialogElement) { this.open = true; } },
    close: { configurable: true, value(this: HTMLDialogElement) { this.open = false; } },
  });
  const pointEvent = {
    event_id: "event-ai",
    operation_id: "operation-ai",
    resource_type: "ai_point",
    unit_version: "ai-micropoint-v1",
    quantity: "1200.000012",
    available_delta: "-0.000001",
    allocated_delta: "0.000000",
    reserved_delta: "0.000001",
    consumed_delta: "1200.000012",
    available_after: "100.000001",
    allocated_after: "0",
    reserved_after: "0.000001",
    consumed_after: "1200.000012",
    reason: "consumed",
    source_type: "model_invocation_v1",
    source_identity: "invocation-1",
    occurred_at: "2026-11-16T00:00:00Z",
  };
  const rowEvent = {
    ...pointEvent,
    event_id: "event-data",
    resource_type: "data_row",
    unit_version: undefined,
    quantity: "2500",
    available_delta: "2500",
    allocated_delta: "0",
    reserved_delta: "0",
    consumed_delta: "0",
    available_after: "2500",
    reserved_after: "0",
    consumed_after: "0",
    reason: "purchase",
    source_type: "purchase",
    source_identity: "purchase-1",
  };
  const page = parseResourceEvents({
    organization_id: "org-B",
    items: [pointEvent, rowEvent],
    next_cursor: null,
  });
  expect(page).not.toBeNull();
  render(<UsageDetailsView scope={scope} page={page!} onFilter={vi.fn()} onNext={vi.fn()} />);

  const rows = screen.getAllByRole("row");
  expect(rows[1]).toHaveTextContent("AI 模型调用");
  expect(rows[1]).toHaveTextContent("可用 -0.000001 点");
  expect(rows[1]).toHaveTextContent("已使用 +1,200.000012 点");
  expect(rows[2]).toHaveTextContent("可用 +2,500 条");
  await userEvent.click(within(rows[1]).getByRole("button", { name: "详情" }));
  const modal = within(await screen.findByRole("dialog", { name: "用量记录详情" }));
  expect(modal.getByText("invocation-1")).toBeVisible();
  expect(modal.getByRole("row", { name: "可用 -0.000001 点 100.000001 点" })).toBeVisible();
  expect(modal.getByRole("row", { name: "已分配 0 点 0 点" })).toBeVisible();
  expect(modal.getByRole("row", { name: "预留 +0.000001 点 0.000001 点" })).toBeVisible();
  expect(modal.getByRole("row", { name: "已使用 +1,200.000012 点 1,200.000012 点" })).toBeVisible();
});

it.each(["available", "denied", "mismatched"] as const)(
  "shows the 1688 source and only displays a product from the matching authorized lookup (%s)",
  async (result) => {
    Object.defineProperties(HTMLDialogElement.prototype, {
      showModal: { configurable: true, value(this: HTMLDialogElement) { this.open = true; } },
      close: { configurable: true, value(this: HTMLDialogElement) { this.open = false; } },
    });
    const operationId = "55555555-5555-4555-8555-555555555555";
    const returnedId = result === "mismatched" ? "66666666-6666-4666-8666-666666666666" : operationId;
    const responses = result === "denied"
      ? Response.json({ code: "PERMISSION_DENIED" }, { status: 403 })
      : Response.json({
        schemaVersion: 1, operationId: returnedId, productKey: "crawler:1688:1234",
        publicationId: `source-run:acquisition:${returnedId}`, catalogVersion: "1",
        title: "已保存的商品", sources: [], images: [], specifications: [], warnings: [], missingFacts: [],
      });
    const fetchMock = vi.fn().mockResolvedValue(responses);
    vi.stubGlobal("fetch", fetchMock);
    const page = parseResourceEvents({
      organization_id: scope.organizationId, next_cursor: null,
      items: [{
        event_id: "acquisition-event", operation_id: "charge-event", resource_type: "data_row",
        quantity: "1", available_delta: "0", allocated_delta: "0", reserved_delta: "-1", consumed_delta: "1",
        available_after: "10", allocated_after: "0", reserved_after: "0", consumed_after: "1",
        reason: "commit", source_type: "product_acquisition_v1", source_identity: operationId,
        occurred_at: "2026-11-17T04:12:39Z",
      }],
    });
    expect(page).not.toBeNull();
    const client = new QueryClient();
    const view = render(
      <QueryClientProvider client={client}>
        <UsageDetailsView scope={scope} page={page!} onFilter={vi.fn()} />
      </QueryClientProvider>,
    );
    try {
      expect(screen.getByRole("row", { name: /1688 商品采集/ })).toBeVisible();
      expect(fetchMock).not.toHaveBeenCalled();
      await userEvent.click(screen.getByRole("button", { name: "详情" }));
      const dialog = within(await screen.findByRole("dialog", { name: "用量记录详情" }));
      if (result === "available") {
        expect(await dialog.findByRole("link", { name: "已保存的商品" })).toHaveAttribute(
          "href", `/workbench/supply/acquisition/operation/${operationId}`,
        );
      } else {
        expect(await dialog.findByText("未能查看关联商品")).toBeVisible();
        expect(dialog.queryByRole("link")).not.toBeInTheDocument();
      }
      expect(dialog.getByText(operationId)).toBeVisible();
      expect(fetchMock).toHaveBeenCalledTimes(1);
      const [path, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(path).toBe(`/api/workbench/sourcing/1688/acquisitions/${operationId}/product`);
      expect(init.method).toBe("GET");
      expect(new Headers(init.headers).get("X-Expected-Organization-ID")).toBe(scope.organizationId);
      expect(new Headers(init.headers).get("X-Expected-User-ID")).toBe(scope.userId);
    } finally {
      view.unmount();
      client.clear();
    }
  },
);
