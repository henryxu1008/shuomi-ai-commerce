import {describe, expect, it} from "vitest";
import {priceInput, routeOptions} from "./model-route-pricing";

const rates = {inputText: 10000, cachedInputText: 0, inputImage: 0, cachedInputImage: 0, outputText: 20000, outputImage: 0};
const price = {unitVersion: "ai-micropoint-v1", priceVersion: "price-v1", fxVersion: "fx-v1", currency: "CNY",
  cnyMicrosPerCurrencyUnit: 1000000, model: "model", wireModel: "vendor/model", channel: "configured-channel",
  usageVersion: "reference-text-totals-v1", rates};

describe("qualified multi-provider model contracts", () => {
  it("reads an explicitly bound channel model without replacing the manufacturer price model", () => {
    const dto = {organizationId: "org", channels: ["configured-channel"], models: [{
      category: "text", reference: {id: "configured-model", version: "v1"},
      model: "vendor/model", channel: "configured-channel", ready: true, price,
    }]};
    expect(routeOptions.safeParse(dto).success).toBe(true);
    expect(routeOptions.safeParse({...dto, models: [{...dto.models[0], model: "other/model"}]}).success).toBe(false);
    expect(routeOptions.safeParse({...dto, models: [{...dto.models[0], price: {...price, wireModel: undefined}}]}).success).toBe(false);
  });

  it("allows the existing platform price entry to publish an Anthropic reference, in the same unit", () => {
    const entry = {id: "claude", provider: "anthropic", region: "global", serviceType: "standard-sync",
      channel: "configured-channel", model: "model", currency: "CNY", usageVersion: "reference-text-totals-v1", rates,
      sourceUrl: "https://www.anthropic.com/pricing", verifiedAt: "2026-11-04T00:00:00Z",
      effectiveFrom: "2026-11-04T00:00:00Z", effectiveUntil: "2026-11-05T00:00:00Z"};
    expect(priceInput.safeParse({expectedRevision: "NONE", prices: [entry], fx: []}).success).toBe(true);
    expect(priceInput.safeParse({expectedRevision: "NONE", prices: [{...entry, provider: "unqualified-vendor"}], fx: []}).success).toBe(false);
  });
});
