import { describe, expect, it } from "vitest";
import { jobSchema } from "./data-services";

const missing = {
  id: "8b980c6a-1ed7-46bf-9fbc-ce2b9a8cefd5",
  commandKey: "8b980c6a-1ed7-46bf-9fbc-ce2b9a8cefd5",
  query: { site: "us", mode: "asin", asins: ["B000123456"], limit: 1 },
  state: "FAILED", priceStatus: "UNAVAILABLE_TERMINAL", price: null,
  confirmedFen: null, pendingFen: null,
  discovered: true, canceled: false, saved: 0, failed: 1, pending: 0,
  createdAt: "2026-11-07T12:46:44Z", deadline: "2026-11-07T13:16:44Z",
};

describe("current acquisition price availability", () => {
  it("accepts explicitly unavailable failed history without turning null into zero", () => {
    const job = jobSchema.parse(missing);
    expect(job.price).toBeNull();
    expect(job.confirmedFen).toBeNull();
  });
  it.each([{ state: "RUNNING" }, { saved: 1 }, { pending: 1 }, { confirmedFen: "0" }, { priceStatus: "FROZEN" }])("refuses an invalid missing-price projection %o", patch => {
    expect(jobSchema.safeParse({ ...missing, ...patch }).success).toBe(false);
  });
  it.each(["0", "7"])("preserves an explicit frozen price %s", amount => {
    expect(jobSchema.safeParse({ ...missing, priceStatus: "FROZEN", price: { offerId: "offer", pricingVersion: "v1", currency: "CNY", unitPriceFen: amount }, confirmedFen: "0", pendingFen: "0" }).success).toBe(true);
  });
});
