import { beforeEach, afterEach, expect, it, vi } from "vitest";
import {
  loadFulfillmentIntent,
  saveFulfillmentIntent,
} from "./fulfillment-intent";
import type { FulfillmentInput } from "./fulfillment";
const scope = { organizationId: "org", userId: "user" };
const original: FulfillmentInput = {
  store: "store",
  kind: "platform_supply",
  source_id: "source",
  operation_id: "a182d2e0-6f91-4e71-8b7f-5df25b75fb11",
  action: "create_purchase",
  payload: { demand: "native" },
};
beforeEach(() => {
  localStorage.clear();
  vi.stubGlobal("navigator", {
    locks: {
      request: async (_name: string, callback: () => Promise<boolean>) =>
        callback(),
    },
  });
});
afterEach(() => vi.unstubAllGlobals());
it("acknowledges the exact UI-constructed original despite schema field ordering and retains mismatched payloads", async () => {
  const constructed: FulfillmentInput = {
    store: "store", kind: "platform_supply", source_id: "source",
    action: "receive", payload: { purchase: "PO", lines: [{ purchase_line: "row", qty: 2, rejected_qty: 0, short_qty: 0 }] },
    operation_id: original.operation_id,
  };
  expect(await saveFulfillmentIntent(scope, constructed, null)).toBe(true);
  expect(loadFulfillmentIntent(scope)).toEqual(constructed);
  expect(await saveFulfillmentIntent(scope, null, { ...constructed, operation_id: "ca0942bc-aef7-4ae1-a5be-e4bacf434ede" })).toBe(false);
  expect(await saveFulfillmentIntent(scope, null, { ...constructed, payload: { purchase: "other" } })).toBe(false);
  expect(loadFulfillmentIntent(scope)).toEqual(constructed);
  expect(await saveFulfillmentIntent(scope, null, constructed)).toBe(true);
  expect(loadFulfillmentIntent(scope)).toBeNull();
});
it("keeps the original key and payload through reload and refuses replacement while unresolved", async () => {
  expect(await saveFulfillmentIntent(scope, original, null)).toBe(true);
  expect(loadFulfillmentIntent(scope)).toEqual(original);
  expect(
    await saveFulfillmentIntent(
      scope,
      { ...original, operation_id: "ca0942bc-aef7-4ae1-a5be-e4bacf434ede" },
      null,
    ),
  ).toBe(false);
  expect(
    loadFulfillmentIntent({ organizationId: "other", userId: "user" }),
  ).toBeNull();
  expect(
    await saveFulfillmentIntent(scope, null, {
      ...original,
      payload: { demand: "changed" },
    }),
  ).toBe(false);
  expect(await saveFulfillmentIntent(scope, null, original)).toBe(true);
  expect(loadFulfillmentIntent(scope)).toBeNull();
});
it("fails closed before dispatch when original intent cannot be durably saved", async () => {
  vi.stubGlobal("navigator", {});
  expect(await saveFulfillmentIntent(scope, original, null)).toBe(false);
});
