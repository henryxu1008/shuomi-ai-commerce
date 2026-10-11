import { expect, it } from "vitest";
import { cockpitRules, moneyCents, moneyText, ratioText } from "./operations-cockpit";
it("keeps cents and derived rational display exact without float rounding", () => {
 expect(moneyCents("100.25")).toBe(10025);expect(moneyCents("0")).toBe(0);expect(moneyCents("1.005")).toBeNull();expect(moneyCents("10000000000.01")).toBeNull();
 expect(moneyText(9007199254740991)).toBe("¥90,071,992,547,409.91");
 expect(moneyText(-101)).toBe("-¥1.01");expect(ratioText({ numerator: "-1", denominator: "3" })).toBe("-33.3%");
});

it("requires the same full-result summary at the BFF and rejects mismatched counts", () => {
 const result = { rules: [], total: 61, page: 3, capturedAt: "2026-10-10T01:00:00Z", goalUnavailable: false, observations: { state: "unavailable", exceptional: 0, complete: false, sources: [], latest: [], actionPath: "/workbench/store-orders" } };
 expect(cockpitRules.safeParse(result).success).toBe(false);
 expect(cockpitRules.safeParse({ ...result, summary: { total: 61, critical: 1, attention: 25, data: 35 } }).success).toBe(true);
 expect(cockpitRules.safeParse({ ...result, summary: { total: 60, critical: 1, attention: 25, data: 35 } }).success).toBe(false);
 expect(cockpitRules.safeParse({ ...result, summary: { total: 61, critical: 1, attention: 24, data: 35 } }).success).toBe(false);
});
