import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ProcurementPanel } from "./fulfillment-forms";
import type { FulfillmentDetail } from "@/lib/api/fulfillment";

afterEach(cleanup);
const detail: FulfillmentDetail = {
  name: "SO", source: { organization: "org", store: "store", source_id: "source", source_kind: "platform_supply" }, movement_kind: "owned_transfer",
  lines: [], deliveries: [], operations: [], has_more_purchases: false, has_more_deliveries: false, has_more_operations: false,
  purchases: [{ name: "PO", supplier: "Supplier", external_order: null, delivery_date: "2026-11-03", status: "To Receive and Bill", currency: "USD", receipts: [], has_more_receipts: false,
    lines: [{ line: "po-line", demand_line: "demand-line", item_code: "ITEM", uom: "Nos", qty: 6, received_qty: 4, rate: 2 }] }],
};

it.each(["record_external_order", "receive", "close_purchase"] as const)("submits %s through its actual button and form validation", async action => {
  const write=vi.fn(async () => null);
  render(<ProcurementPanel detail={detail} catalog={{ items: [], suppliers: [], currency: "USD", source_warehouse: "source", movement_kind: "owned_transfer" }} disabled={false} canPurchase canStock write={write} />);
  const user=userEvent.setup();
  if(action==="record_external_order"){
    await user.type(screen.getByRole("textbox",{name:"供应商平台订单号"}),"SYN-EXTERNAL");
    await user.click(screen.getByRole("button",{name:"登记已在外部下单"}));
    expect(write).toHaveBeenCalledExactlyOnceWith(action,{purchase:"PO",external_order:"SYN-EXTERNAL"});
  }else if(action==="receive"){
    await user.clear(screen.getByRole("spinbutton",{name:"合格收货 PO ITEM"}));
    await user.type(screen.getByRole("spinbutton",{name:"合格收货 PO ITEM"}),"2");
    await user.click(screen.getByRole("button",{name:"确认登记合格收货"}));
    expect(write).toHaveBeenCalledExactlyOnceWith(action,{purchase:"PO",lines:[{purchase_line:"po-line",qty:2,rejected_qty:0,short_qty:0}]});
  }else{
    await user.type(screen.getByRole("textbox",{name:"剩余关闭原因"}),"Synthetic supplier confirmed no further delivery");
    await user.click(screen.getByRole("checkbox",{name:"已与供应商确认剩余不再交付，本次仅关闭内部剩余采购"}));
    await user.click(screen.getByRole("button",{name:"确认关闭内部剩余采购"}));
    expect(write).toHaveBeenCalledExactlyOnceWith(action,{purchase:"PO",reason:"Synthetic supplier confirmed no further delivery"});
  }
});
