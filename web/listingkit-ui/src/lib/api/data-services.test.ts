import { describe, it, expect } from "vitest";
import { loadDataIntent, saveDataIntent, validDataIntent } from "./data-services";
import { dataRoute, jobSchema, optionsSchema, overviewSchema } from "@/lib/contracts/data-services";
describe("data command recovery", () => {
    it("retains the original command and scope without retaining key secrets", () => {
        const scope = { userId: "user", organizationId: "org" };
        const intent = { ...scope, key: "a233d58b-1fd3-40d7-a983-d35bbec45313", path: "keys", body: { name: "key", expiresAt: "2026-10-20T00:00:00Z", dailyRows: 20, monthlyCostFen: 100, permissions: ["amazon.acquire"] }, specialist: false };
        expect(saveDataIntent(intent)).toBe(true);
        expect(loadDataIntent(scope, false)).toEqual(intent);
        expect(loadDataIntent({ userId: "another", organizationId: "org" }, false)).toBeNull();
        expect(saveDataIntent({ ...intent, body: { ...intent.body, secret: "never-store" } })).toBe(false);
    });
});

const id = "a233d58b-1fd3-40d7-a983-d35bbec45313";
const query = { site: "us", mode: "asin", asins: ["B000123456"], limit: 1 };
const pricingFixtures = (amount: string) => {
    const price = { offerId: "offer", pricingVersion: "original-price", currency: "CNY", unitPriceFen: amount };
    const options = { sites: [], customSites: [], fields: [], price, maximumRows: 200, formats: [], acquisitionReady: false };
    const job = { id, commandKey: id, query, priceStatus:"FROZEN", price, state: "ADMITTED", discovered: false, canceled: false, saved: 0, failed: 0, pending: 0, confirmedFen: amount, pendingFen: amount, createdAt: "2026-11-08T00:00:00Z", deadline: "2026-11-08T00:30:00Z" };
    const overview = { options, keys: [], jobs: [job], keyQuotas: [{ keyId: id, dayConsumedRows: 0, dayReservedRows: 0, monthConsumedFen: amount, monthReservedFen: amount }], usage: { dayRows: 0, monthConfirmedFen: amount, monthPendingFen: amount, finishedJobs: 0, succeededJobs: 0, successRate: null, window: "UTC month" } };
    return { options, job, overview, request: { query, maximumRows: 1, maximumCostFen: amount, pricingVersion: price.pricingVersion } };
};

describe("frozen data price and budget decimal validation", () => {
    const request = dataRoute("POST", ["amazon", "jobs"])!.request!;
    it.each(["", " ", "1.5", "-", "abc", "-1", "+1", "1e3", "01", " 1", "1 ", "9223372036854775808", "9".repeat(1000)])("rejects malformed request/response amounts without throwing: %s", amount => {
        const fixture = pricingFixtures(amount);
        expect(request.safeParse(fixture.request).success).toBe(false);
        expect(optionsSchema.safeParse(fixture.options).success).toBe(false);
        expect(jobSchema.safeParse(fixture.job).success).toBe(false);
        expect(overviewSchema.safeParse(fixture.overview).success).toBe(false);
    });
    it.each(["0", "1", "7", "9223372036854775807"])("keeps the exact canonical amount %s in requests, snapshots and usage", amount => {
        const fixture = pricingFixtures(amount);
        expect(request.parse(fixture.request)).toEqual(fixture.request);
        expect(optionsSchema.parse(fixture.options).price?.unitPriceFen).toBe(amount);
        expect(jobSchema.parse(fixture.job).confirmedFen).toBe(amount);
        expect(overviewSchema.parse(fixture.overview).keyQuotas[0].monthReservedFen).toBe(amount);
    });
    it("refuses to recover a browser command whose stored budget is malformed", () => {
        const scope = { userId: "user", organizationId: "org" };
        const fixture = pricingFixtures("abc");
        const intent = { ...scope, key: id, path: "amazon/jobs", body: fixture.request, specialist: false };
        expect(validDataIntent(intent)).toBe(false);
        expect(saveDataIntent(intent)).toBe(false);
        const raw = JSON.stringify(intent);
        const key = `data-services-command-v1:${scope.userId}:${scope.organizationId}`;
        sessionStorage.setItem(key, raw);
        expect(loadDataIntent(scope, false)).toBeNull();
        expect(sessionStorage.getItem(key)).toBe(raw);
        sessionStorage.removeItem(key);
    });
});
