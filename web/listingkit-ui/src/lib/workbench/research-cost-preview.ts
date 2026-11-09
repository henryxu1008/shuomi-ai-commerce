export type CostBasis = { source: string; asOf: string };
export type CostValue =
  | { kind: "value"; value: string; basis?: CostBasis }
  | { kind: "not_applicable"; reason: string };

export const costFields = [
  "procurement", "logistics", "platformFixed", "platformRate", "taxFixed", "taxRate",
  "advertising", "exchangeCost",
] as const;
export type CostField = (typeof costFields)[number];
export type CostOutcome = { basis: CostBasis; costs: Record<CostField, CostValue> };

export type CostDraft = {
  channel: string;
  fulfillment: string;
  specification: string;
  quantity: string;
  price: string;
  refundRate: string;
  salesBasis: CostBasis;
  rule: "fixed_percent" | "unsupported";
  purchaseCurrency: "USD" | "CNY";
  exchangeRate: string;
  exchangeBasis: CostBasis;
  completed: CostOutcome;
  refunded: CostOutcome;
};

export type CostResult =
  | { status: "INVALID_INPUT" | "INSUFFICIENT_INPUT"; fields: string[] }
  | { status: "UNSUPPORTED_COST_RULE" }
  | {
    status: "COMPUTABLE";
    gross: string;
    contribution: string;
    contributionSign: "negative" | "zero" | "positive";
    breakEven: string | null;
    completedContribution: string | null;
    refundedContribution: string | null;
  };

export const blankCostBasis = (): CostBasis => ({ source: "", asOf: "" });

export function blankCostDraft(): CostDraft {
  function outcome(): CostOutcome {
    return {
      basis: blankCostBasis(),
      costs: {
        procurement: { kind: "value", value: "" }, logistics: { kind: "value", value: "" },
        platformFixed: { kind: "value", value: "" }, platformRate: { kind: "value", value: "" },
        taxFixed: { kind: "value", value: "" }, taxRate: { kind: "value", value: "" },
        advertising: { kind: "value", value: "" }, exchangeCost: { kind: "value", value: "" },
      },
    };
  }
  return {
    channel: "", fulfillment: "", specification: "", quantity: "", price: "", refundRate: "",
    salesBasis: blankCostBasis(), rule: "fixed_percent", purchaseCurrency: "USD",
    exchangeRate: "", exchangeBasis: blankCostBasis(), completed: outcome(), refunded: outcome(),
  };
}
