import { expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MappingForm } from "./fulfillment-forms";
import type {
  FulfillmentCatalog,
  FulfillmentSource,
} from "@/lib/api/fulfillment";
it("preserves explicit SKU and UOM selections across catalog pages without supplying a browser quantity", async () => {
  const source: FulfillmentSource = {
    source: {
      kind: "platform_supply",
      id: "order",
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
      deadline: "2026-12-01T00:00:00Z",
      lines: [
        {
          id: "a",
          sku: "sku-a",
          quantity: 2,
          delivered: 0,
          received: null,
          stored: null,
          defective: null,
        },
        {
          id: "b",
          sku: "sku-b",
          quantity: 3,
          delivered: 0,
          received: null,
          stored: null,
          defective: null,
        },
      ],
    },
    version: "a".repeat(64),
    eligible: true,
  };
  const catalog: FulfillmentCatalog = {
    items: [
      {
        name: "native-a",
        item_name: "Item A",
        stock_uom: "Nos",
        uoms: [
          { uom: "Nos", conversion_factor: 1 },
          { uom: "Box", conversion_factor: 6 },
        ],
      },
    ],
    suppliers: [],
    source_warehouse: "source",
    currency: "USD",
    movement_kind: "owned_transfer",
  };
  const write = vi.fn().mockResolvedValue(null),
    props = { source, catalog, disabled: false, write };
  const view = render(<MappingForm {...props} />);
  await userEvent.selectOptions(
    screen.getByLabelText("关联 sku-a a"),
    "native-a",
  );
  await userEvent.selectOptions(
    screen.getByLabelText("原单单位 sku-a a"),
    "Box",
  );
  view.rerender(
    <MappingForm
      {...props}
      catalog={{
        ...catalog,
        items: [
          {
            name: "native-b",
            item_name: "Item B",
            stock_uom: "Nos",
            uoms: [{ uom: "Nos", conversion_factor: 1 }],
          },
        ],
      }}
    />,
  );
  await userEvent.selectOptions(
    screen.getByLabelText("关联 sku-b b"),
    "native-b",
  );
  fireEvent.change(screen.getByLabelText("计划交付日期"), {
    target: { value: "2026-11-05" },
  });
  await userEvent.click(
    screen.getByRole("button", { name: "确认关联并准备履约" }),
  );
  expect(write).toHaveBeenCalledWith("bind_demand", {
    delivery_date: "2026-11-05",
    lines: [
      {
        source_line: "a",
        item_code: "native-a",
        uom: "Box",
        warehouse: "source",
      },
      {
        source_line: "b",
        item_code: "native-b",
        uom: "Nos",
        warehouse: "source",
      },
    ],
  });
});
