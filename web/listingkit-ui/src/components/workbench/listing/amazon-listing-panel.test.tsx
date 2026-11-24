import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { AmazonListingPanel } from "./amazon-listing-panel";
import { SupplyAPIError } from "@/lib/api/supply-chain";
import type { SupplySourceDetail } from "@/lib/contracts/supply-chain";
import type { AmazonListingReview, AmazonListingResult } from "@/lib/contracts/amazon-listing";

const calls = vi.hoisted(() => ({ review: vi.fn(), confirm: vi.fn(), details: vi.fn(), listing: vi.fn(), inventory: vi.fn() }));
vi.mock("@/lib/api/supply-chain", async original => ({
  ...await original<object>(), amazonReview: calls.review, amazonConfirm: calls.confirm,
  readAmazonReview: calls.details, readAmazonListing: calls.listing, supplyInventory: calls.inventory,
}));
const id = "11111111-1111-4111-8111-111111111111";
const storeId = "22222222-2222-4222-8222-222222222222";
const scope = { userId: "actor", organizationId: "org" };
const source: SupplySourceDetail = {
  source: { id, preparationId: id, collectionItemId: id, collectionRevision: 1, source: { productKey: "synthetic", publicationId: id, version: "1", kind: "own" } },
  product: { title: "Synthetic pouch", variants: [{ barcode: "036000291452", stock: 42, price: { amount: 99 } }] },
  images: [],
};
const details = { productType: "LUGGAGE", title: "Synthetic pouch", brand: "Fixture", gtin: "036000291452", priceCents: 1230, quantity: 0, mainImageId: "main" };
const review = (): AmazonListingReview => ({
  sourceId: id, storeId, recordId: id, revision: 1, sku: "shuomi-" + "a".repeat(32),
  wireHash: "a".repeat(64), details, validUntil: new Date(Date.now() + 60_000).toISOString(),
});
const accepted: AmazonListingResult = {
  sourceId: id, storeId, recordId: id, attemptId: id, sku: "shuomi-" + "a".repeat(32),
  submission: "accepted", receipt: { accepted: true, submissionId: "SYNTHETIC-RETURN" },
};
beforeEach(() => {
  vi.resetAllMocks();
  calls.details.mockRejectedValue(new SupplyAPIError("NOT_FOUND", 404));
  calls.listing.mockRejectedValue(new SupplyAPIError("NOT_FOUND", 404));
  calls.inventory.mockResolvedValue({ assets: [{ id: "main", url: "https://images.example.org/product.png" }] });
  calls.review.mockImplementation(async () => review());
});
afterEach(cleanup);

it("reviews a single product and shows a receipt separately from the later ASIN", async () => {
  const user = userEvent.setup();
  calls.confirm.mockResolvedValue(accepted);
  render(<AmazonListingPanel scope={scope} source={source} storeId={storeId} storeName="合成店铺" canManage canSubmit resultOnly={false} />);
  await user.type(await screen.findByLabelText("Amazon 商品类型"), "LUGGAGE");
  await user.type(screen.getByLabelText("品牌"), "Fixture");
  expect(screen.getByLabelText("售价（美元）")).toHaveValue("");
  expect(screen.getByLabelText("可卖库存")).toHaveValue("");
  await user.type(screen.getByLabelText("售价（美元）"), "12.30");
  await user.type(screen.getByLabelText("可卖库存"), "0");
  await user.selectOptions(screen.getByLabelText("已批准商品主图"), "main");
  await user.click(screen.getByRole("button", { name: "核对商品资料" }));
  const summary = await screen.findByLabelText("确认 Amazon 刊登资料");
  expect(within(summary).getByText("0")).toBeVisible();
  expect(calls.review).toHaveBeenCalledWith(scope, { sourceId: id, storeId, effectiveVersion: "1", expectedRevision: 0, details }, expect.any(String), expect.any(AbortSignal));
  expect(calls.confirm).not.toHaveBeenCalled();
  await user.click(within(summary).getByRole("button", { name: "确认提交到 Amazon" }));
  await screen.findByText("Amazon 已受理");
  expect(screen.getByText("Amazon 提交编号：SYNTHETIC-RETURN")).toBeVisible();
  expect(screen.queryByText(/ASIN：/)).toBeNull();
  calls.listing.mockResolvedValue({ ...accepted, observation: { asin: "B000TE0001", buyable: true, discoverable: true, observedAt: new Date().toISOString() } });
  await user.click(screen.getByRole("button", { name: "查询 Amazon 结果" }));
  expect(await screen.findByText(/ASIN：B000TE0001 · 可购买/)).toBeVisible();
  expect(calls.confirm).toHaveBeenCalledOnce();
});

it("uses result lookup after an uncertain send and keeps the form read-only while it is unresolved", async () => {
  const user = userEvent.setup();
  calls.details.mockImplementation(async () => review());
  calls.confirm.mockRejectedValue(new SupplyAPIError("OUTCOME_UNKNOWN", 503));
  render(<AmazonListingPanel scope={scope} source={source} storeId={storeId} storeName="合成店铺" canManage canSubmit resultOnly={false} />);
  await user.click(await screen.findByRole("button", { name: "确认提交到 Amazon" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("提交的结果待确认");
  calls.listing.mockResolvedValue({ ...accepted, submission: "unknown", receipt: undefined });
  await user.click(screen.getByRole("button", { name: "查询 Amazon 结果" }));
  expect(await screen.findByText("提交结果待确认")).toBeVisible();
  expect(screen.queryByRole("button", { name: "确认提交到 Amazon" })).toBeNull();
  expect(screen.queryByRole("button", { name: "核对商品资料" })).toBeNull();
  expect(calls.confirm).toHaveBeenCalledOnce();
  expect(calls.listing).toHaveBeenLastCalledWith(scope, id, storeId, true, expect.any(AbortSignal));
});

it("requires a new review after a locally blocked send, including when reopening the product", async () => {
  const user = userEvent.setup();
  const ready = review();
  const blocked: AmazonListingResult = { ...accepted, submission: "not_sent", receipt: undefined };
  const nextReview = { ...ready, recordId: "33333333-3333-4333-8333-333333333333", revision: 2 };
  const panel = <AmazonListingPanel scope={scope} source={source} storeId={storeId} storeName="合成店铺" canManage canSubmit resultOnly={false} />;
  calls.details.mockResolvedValue(ready);
  calls.confirm.mockResolvedValueOnce(blocked).mockResolvedValueOnce({ ...accepted, recordId: nextReview.recordId });
  calls.review.mockResolvedValue(nextReview);
  const first = render(panel);
  await user.click(await screen.findByRole("button", { name: "确认提交到 Amazon" }));
  expect(await screen.findByText("尚未提交到 Amazon")).toBeVisible();
  expect(screen.queryByRole("button", { name: "确认提交到 Amazon" })).toBeNull();
  first.unmount();

  calls.listing.mockResolvedValue(blocked);
  render(panel);
  expect(await screen.findByText("权限恢复后，请重新核对商品资料，再确认提交。")).toBeVisible();
  expect(screen.queryByRole("button", { name: "确认提交到 Amazon" })).toBeNull();
  expect(screen.queryByRole("button", { name: "查询 Amazon 结果" })).toBeNull();
  await user.click(await screen.findByRole("button", { name: "核对商品资料" }));
  expect(calls.review).toHaveBeenCalledWith(scope, {
    sourceId: id, storeId, effectiveVersion: "1", expectedRevision: 1, details,
  }, expect.any(String), expect.any(AbortSignal));
  await user.click(await screen.findByRole("button", { name: "确认提交到 Amazon" }));
  expect(await screen.findByText("Amazon 已受理")).toBeVisible();
  expect(calls.confirm).toHaveBeenLastCalledWith(scope, {
    recordId: nextReview.recordId, wireHash: nextReview.wireHash,
  }, expect.any(String), expect.any(AbortSignal));
  expect(calls.confirm.mock.calls[0]?.[2]).not.toBe(calls.confirm.mock.calls[1]?.[2]);
});

it("requires a new review when checking a lost browser response proves the request was blocked", async () => {
  const user = userEvent.setup();
  calls.details.mockImplementation(async () => review());
  calls.confirm.mockRejectedValue(new SupplyAPIError("OUTCOME_UNKNOWN", 503));
  render(<AmazonListingPanel scope={scope} source={source} storeId={storeId} storeName="合成店铺" canManage canSubmit resultOnly={false} />);
  await user.click(await screen.findByRole("button", { name: "确认提交到 Amazon" }));
  await screen.findByText("这次提交的结果待确认。请查询原商品结果。");
  calls.listing.mockResolvedValue({ ...accepted, submission: "not_sent", receipt: undefined });
  await user.click(screen.getByRole("button", { name: "查询 Amazon 结果" }));
  expect(await screen.findByText("尚未提交到 Amazon")).toBeVisible();
  expect(screen.getByRole("button", { name: "核对商品资料" })).toBeEnabled();
  expect(screen.queryByRole("button", { name: "确认提交到 Amazon" })).toBeNull();
});
