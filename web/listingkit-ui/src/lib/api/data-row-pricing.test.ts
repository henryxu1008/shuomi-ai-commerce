import { expect, it } from "vitest";
import { dataRowPriceInput } from "./data-row-pricing";

const input = { expected_version: "original-price", reason: "isolated validation" };

it.each(["", " ", "1.5", "-", "abc", "-1", "+1", "1e3", "01", " 1", "1 ", "9223372036854775808", "9".repeat(1000)])(
  "rejects an invalid price without throwing: %s",
  unit_price_minor => {
    expect(dataRowPriceInput.safeParse({ ...input, unit_price_minor }).success).toBe(false);
  },
);

it.each(["0", "1", "7", "9223372036854775807"])("preserves the canonical integer price %s", unit_price_minor => {
  expect(dataRowPriceInput.parse({ ...input, unit_price_minor })).toEqual({ ...input, unit_price_minor });
});
