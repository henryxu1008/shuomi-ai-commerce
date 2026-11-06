import { describe, expect, it } from "vitest";
import { blankCostDraft, evaluateCost, type CostDraft } from "./research-cost-preview";

const basis = { source: "合成成本资料", asOf: "2026-11-05" };
const amount = (value: string) => ({ kind: "value" as const, value });
const na = { kind: "not_applicable" as const, reason: "合成情景无此项" };

function fixture(): CostDraft {
  const draft = blankCostDraft();
  const common = {
    procurement: amount("8"), platformFixed: amount("0"), taxFixed: na,
    taxRate: na, advertising: amount("3"), exchangeCost: na,
  };
  return {
    ...draft, channel: "美国站", fulfillment: "商家直发", specification: "蓝色单件",
    quantity: "20", price: "30", refundRate: "5", salesBasis: basis,
    completed: { basis, costs: { ...common, logistics: amount("6"), platformRate: amount("10") } },
    refunded: { basis, costs: { ...common, logistics: amount("8"), platformRate: amount("0") } },
  };
}

describe("manual cost estimate", () => {
  it("calculates the original full refund fixture and lowest cent price", () => {
    const draft = fixture();
    expect(evaluateCost(draft)).toMatchObject({
      status: "COMPUTABLE", contribution: "8.55", gross: "20.50",
      completedContribution: "10.00", refundedContribution: "-19.00",
      breakEven: "20.00", contributionSign: "positive",
    });
    expect(evaluateCost({ ...draft, price: "19.99" })).toMatchObject({
      status: "COMPUTABLE", contribution: "-0.01", contributionSign: "negative", breakEven: "20.00",
    });
  });

  it("separates missing inputs, explicit zero, missing provenance and unsupported rules", () => {
    const draft = fixture();
    draft.completed.costs.procurement = amount("");
    expect(evaluateCost(draft)).toEqual({ status: "INSUFFICIENT_INPUT", fields: ["completed.procurement"] });
    draft.completed.costs.procurement = amount("0");
    expect(evaluateCost(draft).status).toBe("COMPUTABLE");
    draft.completed.basis = { source: "", asOf: "" };
    expect(evaluateCost(draft)).toMatchObject({ status: "INSUFFICIENT_INPUT", fields: ["completed.basis.source", "completed.basis.asOf"] });
    expect(evaluateCost({ ...draft, rule: "unsupported" })).toEqual({ status: "UNSUPPORTED_COST_RULE" });
  });

  it("uses explicit directional FX and rejects wrong units, invalid amounts and dates", () => {
    const draft = fixture();
    draft.purchaseCurrency = "CNY";
    draft.completed.costs.procurement = amount("40");
    draft.refunded.costs.procurement = amount("40");
    expect(evaluateCost(draft)).toMatchObject({ status: "INSUFFICIENT_INPUT", fields: expect.arrayContaining(["exchangeRate"]) });
    draft.exchangeRate = "0.200000";
    draft.exchangeBasis = basis;
    expect(evaluateCost(draft)).toMatchObject({ status: "COMPUTABLE", contribution: "8.55", breakEven: "20.00" });
    for (const input of [
      { ...draft, exchangeRate: "0.2000001" },
      { ...draft, purchaseCurrency: "EUR" as CostDraft["purchaseCurrency"] },
      { ...draft, price: "3e1" },
      { ...draft, quantity: "1.5" },
      { ...draft, salesBasis: { ...basis, asOf: "2026-02-29" } },
      { ...draft, refundRate: "100.01" },
    ]) expect(evaluateCost(input).status).toBe("INVALID_INPUT");
  });

  it("finds the next price grid point and preserves the sign of sub-cent losses", () => {
    const draft = fixture();
    draft.refundRate = "0";
    draft.price = "0.01";
    draft.completed.costs = {
      procurement: amount("0.01"), logistics: na, platformFixed: na,
      platformRate: amount("0.01"), taxFixed: na, taxRate: na, advertising: na, exchangeCost: na,
    };
    expect(evaluateCost(draft)).toMatchObject({
      status: "COMPUTABLE", contribution: "0.00", contributionSign: "negative",
      breakEven: "0.02", refundedContribution: null,
    });
    draft.completed.costs.platformRate = amount("100");
    expect(evaluateCost(draft)).toMatchObject({ status: "COMPUTABLE", breakEven: null });
    draft.completed.costs.procurement = amount("0");
    expect(evaluateCost(draft)).toMatchObject({ status: "COMPUTABLE", breakEven: "0.01", contributionSign: "zero" });
    draft.completed.costs.taxRate = amount("1");
    expect(evaluateCost(draft)).toMatchObject({ status: "COMPUTABLE", breakEven: null, contributionSign: "negative" });
  });
});


it("rounds positive and negative half cents away from zero", () => {
  const draft = fixture();
  draft.refundRate = "0";
  draft.price = "0.01";
  draft.completed.costs = {
    procurement: amount("0"), logistics: na, platformFixed: na,
    platformRate: amount("50"), taxFixed: na, taxRate: na, advertising: na, exchangeCost: na,
  };
  expect(evaluateCost(draft)).toMatchObject({ status: "COMPUTABLE", contribution: "0.01" });
  draft.completed.costs.procurement = amount("0.01");
  expect(evaluateCost(draft)).toMatchObject({ status: "COMPUTABLE", contribution: "-0.01" });
});
