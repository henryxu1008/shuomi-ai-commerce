import { z } from "zod";
import { readBoundedStrictJSON } from "./strict-json-response";
export const fulfillmentID = z
  .string()
  .regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/);
const text = z.string().max(800);
const qty = z.number().int().min(0).max(100000000);
export const fulfillmentInput = z
  .object({
    store: fulfillmentID,
    kind: z.enum(["consumer_order", "platform_supply"]),
    source_id: fulfillmentID,
    operation_id: z.uuid(),
    action: z.enum([
      "source",
      "options",
      "shipment",
      "catalog",
      "bind_demand",
      "details",
      "find_demand",
      "create_purchase",
      "record_external_order",
      "close_purchase",
      "receive",
      "reserve",
      "pack",
      "dispatch",
      "prepare",
      "confirm",
      "ensure",
      "cancel",
      "cancel_original",
      "read",
      "read_private",
    ]),
    payload: z.record(z.string(), z.unknown()),
  })
  .strict();
export type FulfillmentInput = z.infer<typeof fulfillmentInput>;
export type FulfillmentScope = { organizationId: string; userId: string };
export const fulfillmentCapabilities = z
  .object({
    available: z.boolean(),
    addressConfigured: z.boolean(),
    realPlatformValidation: z.literal("NOT_RUN"),
  })
  .strict();
const sourceLine = z
  .object({
    id: fulfillmentID,
    sku: text,
    quantity: qty,
    delivered: qty,
    received: qty.nullable(),
    stored: qty.nullable(),
    defective: qty.nullable(),
    goodsStatus: z.number().int().optional(),
    exchangeTag: z.number().int().optional(),
  })
  .strict();
export const fulfillmentSource = z
  .object({
    source: z
      .object({
        kind: z.enum(["consumer_order", "platform_supply"]),
        id: fulfillmentID,
        market: text,
        status: z.number().int(),
        stockMode: z.number().int(),
        printStatus: z.number().int(),
        logistics: z.array(z.number().int()).nullable(),
        selectedLogistics: z.number().int(),
        scene: z.number().int(),
        type: z.number().int(),
        countryMarket: z.number().int().nullable(),
        storageId: text,
        rules: z.unknown(),
        deadline: z.string(),
        lines: z.array(sourceLine).max(20),
      })
      .strict(),
    version: z.string().length(64),
    eligible: z.boolean(),
  })
  .strict();
export type FulfillmentSource = z.infer<typeof fulfillmentSource>;
export const fulfillmentReceipt = z
  .object({
    operation: z.uuid(),
    stage: z.enum([
      "PREPARED",
      "CONFIRMED",
      "CLAIMED",
      "DISPATCHING",
      "SUCCEEDED",
      "UNKNOWN",
      "DEFINITIVE_NO_EFFECT",
      "CANCELLED",
      "PROCESSING",
    ]),
    target: text.nullable(),
    revision: text.nullable(),
    result: z
      .object({
        reason: text.optional(),
        trace_id: text.optional(),
        shipping_id: text.optional(),
        package_ids: z.array(text).optional(),
        accepted_goods_ids: z.array(text).optional(),
      })
      .strict(),
  })
  .strict();
export type FulfillmentReceipt = z.infer<typeof fulfillmentReceipt>;
export const fulfillmentCatalog = z
  .object({
    items: z
      .array(
        z
          .object({
            name: text,
            item_name: text,
            stock_uom: text,
            uoms: z
              .array(
                z
                  .object({
                    uom: text,
                    conversion_factor: z.number().positive().finite(),
                  })
                  .strict(),
              )
              .min(1)
              .max(20),
          })
          .strict(),
      )
      .max(20),
    suppliers: z
      .array(z.object({ name: text, supplier_name: text }).strict())
      .max(20),
    source_warehouse: text,
    currency: z.string().length(3),
    movement_kind: z.enum(["owned_transfer", "commercial_sale"]),
  })
  .strict();
export type FulfillmentCatalog = z.infer<typeof fulfillmentCatalog>;
const demandLine = z
  .object({
    line: text,
    source_line: text,
    item_code: text,
    uom: text,
    qty: qty,
    delivered_qty: qty,
    reserved_qty: z.number().min(0),
    ordered_qty: qty,
    reserve_budget: qty,
    purchase_remaining: qty,
    reservable_qty: qty,
  })
  .strict();
const purchase = z
  .object({
    name: text,
    supplier: text,
    external_order: text.nullable(),
    delivery_date: text,
    status: text,
    currency: text,
    receipts: z
      .array(
        z
          .object({
            name: text,
            date: text,
            lines: z
              .array(
                z
                  .object({
                    purchase_line: text,
                    accepted_qty: qty,
                    rejected_qty: qty,
                    short_qty: qty,
                  })
                  .strict(),
              )
              .max(20),
          })
          .strict(),
      )
      .max(20),
    has_more_receipts: z.boolean(),
    lines: z
      .array(
        z
          .object({
            line: text,
            demand_line: text,
            item_code: text,
            uom: text,
            qty: qty,
            received_qty: qty,
            rate: z.number().min(0),
          })
          .strict(),
      )
      .max(20),
  })
  .strict();
const delivery = z
  .object({
    name: text,
    docstatus: z.number().int().min(0).max(1),
    package_id: text,
    lines: z.array(z.object({ demand_line: text, qty: qty }).strict()).max(20),
  })
  .strict();
export const fulfillmentDetail = z
  .object({
    name: text,
    source: z
      .object({
        source_kind: z.enum(["consumer_order", "platform_supply"]),
        store: fulfillmentID,
        source_id: fulfillmentID,
        organization: fulfillmentID,
      })
      .strict(),
    movement_kind: z.enum(["owned_transfer", "commercial_sale"]),
    lines: z.array(demandLine).max(20),
    purchases: z.array(purchase).max(20),
    deliveries: z.array(delivery).max(20),
    has_more_purchases: z.boolean(),
    has_more_deliveries: z.boolean(),
    operations: z
      .array(
        fulfillmentReceipt.extend({
          purpose: z.enum([
            "accept_address",
            "read_address",
            "consumer_tracking",
            "supply_shipping",
          ]),
          delivery: text.nullable(),
          selection: z.record(z.string(), z.union([text, z.number()])),
        }),
      )
      .max(20),
    has_more_operations: z.boolean(),
  })
  .strict();
export type FulfillmentDetail = z.infer<typeof fulfillmentDetail>;
export const fulfillmentOptions = z
  .object({
    channels: z
      .array(
        z
          .object({ site: text, expressIdCode: text, expressChannelCode: text })
          .strict(),
      )
      .optional(),
    addressList: z
      .array(
        z.object({ addressId: z.number().int(), addressName: text }).strict(),
      )
      .optional(),
    deliveryTypeList: z
      .array(
        z
          .object({
            deliveryTypeName: text,
            deliveryTypeValue: z.number().int(),
          })
          .strict(),
      )
      .optional(),
    supplierWarehouseList: z
      .array(
        z
          .object({
            supplierWarehouseId: z.number().int(),
            warehouseName: text,
          })
          .strict(),
      )
      .optional(),
    motorcadeList: z
      .array(z.object({ motorcadeId: text, motorcadeName: text }).strict())
      .optional(),
    packageType: z.number().int().optional(),
    shippingRouteList: text.optional(),
  })
  .strict();
export type FulfillmentOptions = z.infer<typeof fulfillmentOptions>;
export const fulfillmentShipment = z
  .object({
    deliveryCode: text,
    deliveryType: z.number().int(),
    supplierWarehouseId: z.number().int(),
    sendPackage: qty.nullable(),
    expressCode: text,
    expressCompanyName: text,
    addTime: text.nullable(),
    takeParcelTime: text.nullable(),
    preReceiptTime: text.nullable(),
    receiptTime: text.nullable(),
    deliveryOrderDataList: z
      .array(
        z
          .object({
            orderNo: text,
            skuCode: text,
            deliveryQuantity: qty.nullable(),
          })
          .strict(),
      )
      .max(20),
  })
  .strict();
export const fulfillmentPrivate = z
  .object({ private: z.record(z.string(), text), expires: text })
  .strict();
export function fulfillmentResponseSchema(action: FulfillmentInput["action"]) {
  if (action === "shipment") return fulfillmentShipment;
  if (action === "find_demand")
    return z.object({ demand: text.nullable() }).strict();
  if (action === "source") return fulfillmentSource;
  if (action === "details") return fulfillmentDetail;
  if (action === "catalog") return fulfillmentCatalog;
  if (action === "options") return fulfillmentOptions;
  if (action === "read_private") return fulfillmentPrivate;
  return fulfillmentReceipt;
}
export class FulfillmentError extends Error {
  constructor(
    public code: string,
    public status: number,
  ) {
    super(code);
  }
}
export async function fulfillmentRequest<T>(
  scope: FulfillmentScope,
  action: FulfillmentInput | "capabilities",
  schema: z.ZodType<T>,
  signal?: AbortSignal,
): Promise<T> {
  const response = await fetch(
    `/api/workbench/fulfillment/${action === "capabilities" ? "capabilities" : "commands"}`,
    {
      method: action === "capabilities" ? "GET" : "POST",
      headers: {
        "X-Expected-Organization-ID": scope.organizationId,
        "X-Expected-User-ID": scope.userId,
        ...(action !== "capabilities"
          ? { "Content-Type": "application/json" }
          : {}),
      },
      body:
        action === "capabilities"
          ? undefined
          : JSON.stringify(fulfillmentInput.parse(action)),
      credentials: "same-origin",
      cache: "no-store",
      signal,
    },
  );
  const value = await readBoundedStrictJSON(response, 2 * 1024 * 1024, signal);
  if (!response.ok) {
    const error = z.object({ code: z.string() }).safeParse(value);
    throw new FulfillmentError(
      error.success ? error.data.code : "FULFILLMENT_UNKNOWN",
      response.status,
    );
  }
  return schema.parse(value);
}
