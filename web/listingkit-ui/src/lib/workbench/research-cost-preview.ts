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

const ZERO = BigInt(0);
const ONE = BigInt(1);
const TWO = BigInt(2);
const CENTS = BigInt(100);
const BASIS_POINTS = BigInt(10_000);
const FX_SCALE = BigInt(1_000_000);
const DENOMINATOR = BASIS_POINTS * FX_SCALE;
const MAX_CENTS = BigInt(100_000_000);

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

function validDate(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value) || value.startsWith("0000")) return false;
  const date = new Date(value + "T00:00:00.000Z");
  return Number.isFinite(date.getTime()) && date.toISOString().slice(0, 10) === value;
}

function decimal(value: string, scale: number): bigint | null {
  if (value.length > 16 || !/^(0|[1-9]\d*)(?:\.\d+)?$/.test(value)) return null;
  const [whole, fraction = ""] = value.split(".");
  return fraction.length > scale ? null : BigInt(whole + fraction.padEnd(scale, "0"));
}

function dollarString(cents: bigint): string {
  const value = cents < ZERO ? -cents : cents;
  return (cents < ZERO ? "-" : "") + (value / CENTS).toString() + "." +
    (value % CENTS).toString().padStart(2, "0");
}

function roundedCents(value: bigint, denominator: bigint): bigint {
  if (value < ZERO) return -roundedCents(-value, denominator);
  return (value + denominator / TWO) / denominator;
}

function lowestPrice(slope: bigint, fixed: bigint): string | null {
  if (slope <= ZERO) return slope >= fixed ? dollarString(ONE) : null;
  const threshold = (fixed + slope - ONE) / slope;
  const cents = threshold < ONE ? ONE : threshold;
  return cents <= MAX_CENTS ? dollarString(cents) : null;
}

type Terms = { slope: bigint; fixed: bigint; procurement: bigint };

export function evaluateCost(draft: CostDraft): CostResult {
  if (draft.rule === "unsupported") return { status: "UNSUPPORTED_COST_RULE" };

  const missing = new Set<string>();
  const invalid = new Set<string>();
  function text(field: string, value: string, max: number) {
    if (!value.trim()) missing.add(field);
    else if (value.length > max || value.includes("\u0000")) invalid.add(field);
  }
  function basis(field: string, value: CostBasis) {
    text(field + ".source", value.source, 240);
    if (!value.asOf) missing.add(field + ".asOf");
    else if (!validDate(value.asOf)) invalid.add(field + ".asOf");
  }
  function number(field: string, value: string, scale: number, min: bigint, max: bigint) {
    if (!value.trim()) {
      missing.add(field);
      return ZERO;
    }
    const n = decimal(value, scale);
    if (n === null || n < min || n > max) {
      invalid.add(field);
      return ZERO;
    }
    return n;
  }

  text("channel", draft.channel, 120);
  text("fulfillment", draft.fulfillment, 120);
  text("specification", draft.specification, 120);
  number("quantity", draft.quantity, 0, ONE, FX_SCALE);
  const price = number("price", draft.price, 2, ONE, MAX_CENTS);
  const refund = number("refundRate", draft.refundRate, 2, ZERO, BASIS_POINTS);
  basis("salesBasis", draft.salesBasis);

  let fx = FX_SCALE;
  if (draft.purchaseCurrency === "CNY") {
    fx = number("exchangeRate", draft.exchangeRate, 6, ONE, BigInt(1_000) * FX_SCALE);
    basis("exchangeBasis", draft.exchangeBasis);
  } else if (draft.purchaseCurrency !== "USD") {
    invalid.add("purchaseCurrency");
  }

  function terms(name: "completed" | "refunded", probability: bigint): Terms | null {
    if (probability === ZERO) return null;
    const { costs, basis: shared } = draft[name];
    const amounts = {} as Record<CostField, bigint>;
    for (const field of costFields) {
      const input = costs[field];
      const path = name + "." + field;
      if (input.kind === "not_applicable") {
        if (field === "procurement") invalid.add(path);
        else text(path, input.reason, 240);
        amounts[field] = ZERO;
      } else {
        const percent = field === "platformRate" || field === "taxRate";
        amounts[field] = number(path, input.value, 2, ZERO, percent ? BASIS_POINTS : MAX_CENTS);
        if (input.value.trim()) basis(input.basis ? path + ".basis" : name + ".basis", input.basis ?? shared);
      }
    }
    const procurement = amounts.procurement * fx;
    const fixed = procurement + (amounts.logistics + amounts.platformFixed + amounts.taxFixed +
      amounts.advertising + amounts.exchangeCost) * FX_SCALE;
    const revenue = name === "completed" ? BASIS_POINTS : ZERO;
    const slope = (revenue - amounts.platformRate - amounts.taxRate) * (FX_SCALE / BASIS_POINTS);
    return { slope, fixed, procurement };
  }

  const completedProbability = BASIS_POINTS - refund;
  const completed = terms("completed", completedProbability);
  const refunded = terms("refunded", refund);

  if (invalid.size) return { status: "INVALID_INPUT", fields: [...invalid] };
  if (missing.size) return { status: "INSUFFICIENT_INPUT", fields: [...missing] };

  const slope = completedProbability * (completed?.slope ?? ZERO) + refund * (refunded?.slope ?? ZERO);
  const fixed = completedProbability * (completed?.fixed ?? ZERO) + refund * (refunded?.fixed ?? ZERO);
  const contribution = price * slope - fixed;
  const gross = completedProbability * (price * FX_SCALE - (completed?.procurement ?? ZERO)) -
    refund * (refunded?.procurement ?? ZERO);

  function outcomeContribution(outcome: Terms | null): string | null {
    return outcome ? dollarString(roundedCents(price * outcome.slope - outcome.fixed, FX_SCALE)) : null;
  }

  return {
    status: "COMPUTABLE",
    gross: dollarString(roundedCents(gross, DENOMINATOR)),
    contribution: dollarString(roundedCents(contribution, DENOMINATOR)),
    contributionSign: contribution < ZERO ? "negative" : contribution > ZERO ? "positive" : "zero",
    breakEven: lowestPrice(slope, fixed),
    completedContribution: outcomeContribution(completed),
    refundedContribution: outcomeContribution(refunded),
  };
}
