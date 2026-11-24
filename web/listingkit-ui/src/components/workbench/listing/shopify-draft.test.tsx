import {cleanup, fireEvent, render, screen, waitFor} from "@testing-library/react";
import {afterEach, expect, it, vi} from "vitest";
import {readSupplyPublicationStatus, shopifySupplyRules, supplyInventory} from "@/lib/api/supply-chain";
import {SupplyAPIError} from "@/lib/api/supply-error";
import {publicationSchema, type SupplySourceDetail} from "@/lib/contracts/supply-chain";
import {ShopifyDraftEditor, ShopifyDraftResult} from "./shopify-draft";
import type {ListingCommandState} from "./use-listing-command";

vi.mock("@/lib/api/supply-chain", () => ({
 readSupplyPublicationStatus: vi.fn(),
 shopifySupplyRules: vi.fn(),
 supplyInventory: vi.fn(),
}));
afterEach(() => {cleanup(); vi.resetAllMocks()});

const sourceId = "11111111-1111-4111-8111-111111111111";
const storeId = "22222222-2222-4222-8222-222222222222";
const productId = "gid://shopify/Product/12345";
const receipt = {
 sourceId, storeId, receiptId: "receipt-1", observedAt: "2026-11-24T02:00:00Z",
 shopify: {id: productId, status: "DRAFT" as const},
};
const readback = {...receipt, shopifyStatus: receipt.shopify};
const adminUrl = "https://fixture-shop.myshopify.com/admin/products/12345";

it.each([undefined, 4])("submits confirmed Shopify prices with the visible target revision %s", async priorRevision => {
 vi.mocked(shopifySupplyRules).mockImplementation(async () => ({
  merchant: {store_id: storeId}, shopify: {currency: "USD", priorRevision},
 }) as Awaited<ReturnType<typeof shopifySupplyRules>>);
 vi.mocked(supplyInventory).mockResolvedValue({
  scope: {tenant_id: "org", product_key: "fixture", target_platform: "product", source_snapshot_version: "1"},
  assets: [],
 });
 const source: SupplySourceDetail = {
  source: {
   id: sourceId, preparationId: sourceId, collectionItemId: sourceId, collectionRevision: 1,
   source: {productKey: "fixture", publicationId: sourceId, version: "1", kind: "own"},
  },
  product: {title: "Test travel pouch", variants: [{sku: "SAMPLE"}]},
  images: [],
 };
 const execute = vi.fn<ListingCommandState["execute"]>();
 const command: ListingCommandState = {execute, busy: false, error: null, foreign: false, ready: true,
  pending: null, verify: vi.fn(), retry: vi.fn()};
 render(<ShopifyDraftEditor scope={{organizationId: "org", userId: "user"}} source={source}
  storeId={storeId} storeName="Fixture shop" command={command} disabled={false}/>);
 fireEvent.change(await screen.findByRole("textbox", {name: "SAMPLE"}), {target: {value: "12.50"}});
 fireEvent.click(screen.getByRole("button", {name: "保存待上传资料"}));
 expect(execute).toHaveBeenCalledExactlyOnceWith("save-target", {
  sourceId, storeId, expectedRevision: priorRevision ?? 0, effectiveVersion: "1",
  shopify: {currency: "USD", prices: ["12.50"], imageIds: []},
 });
});

it("shows the Shopify admin link after verifying the product", async () => {
 vi.mocked(readSupplyPublicationStatus).mockResolvedValue(publicationSchema.parse({...readback, shopifyAdminUrl: adminUrl}));
 render(<ShopifyDraftResult scope={{organizationId: "org", userId: "user"}} sourceId={sourceId} storeId={storeId} publication={receipt}/>);
 expect(screen.queryByRole("link", {name: "在 Shopify 后台打开商品"})).toBeNull();
 fireEvent.click(screen.getByRole("button", {name: "查看 Shopify 结果"}));
 await waitFor(() => expect(screen.getByRole("link", {name: "在 Shopify 后台打开商品"})).toHaveAttribute("href", adminUrl));
 expect(screen.getByText("商品编号 12345")).toBeTruthy();
});

it("clears a previously shown Shopify receipt when the server says it is no longer visible", async () => {
 vi.mocked(readSupplyPublicationStatus).mockRejectedValue(new SupplyAPIError("NOT_FOUND", 404));
 render(<ShopifyDraftResult scope={{organizationId: "org", userId: "user"}} sourceId={sourceId}
  storeId={storeId} publication={receipt}/>);
 expect(screen.getByText("商品编号 12345")).toBeTruthy();
 fireEvent.click(screen.getByRole("button", {name: "查看 Shopify 结果"}));
 expect(await screen.findByText("暂时没有可查看的 Shopify 结果。")).toBeTruthy();
 expect(screen.queryByText("商品编号 12345")).toBeNull();
});

it("rejects an admin link from an unverified readback or for a different product", () => {
 expect(publicationSchema.safeParse({...receipt, shopifyAdminUrl: adminUrl}).success).toBe(false);
 for (const href of [
  "https://fixture-shop.myshopify.com/admin/products/123456",
  "https://fixture-shop.myshopify.com.evil.example/admin/products/12345",
  "https://fixture-shop.myshopify.com@evil.example/admin/products/12345",
  "https://fixture-shop.myshopify.com/admin/products/12345?next=elsewhere",
 ]) {
  expect(publicationSchema.safeParse({...readback, shopifyAdminUrl: href}).success).toBe(false);
 }
});


it("removes the old admin link after a failed recheck and keeps the receipt", async () => {
 const read = vi.mocked(readSupplyPublicationStatus);
 read.mockResolvedValueOnce(publicationSchema.parse({...readback, shopifyAdminUrl: adminUrl}));
 read.mockRejectedValueOnce(new Error("authorization changed"));
 render(<ShopifyDraftResult scope={{organizationId: "org", userId: "user"}} sourceId={sourceId} storeId={storeId} publication={receipt}/>);
 fireEvent.click(screen.getByRole("button", {name: "查看 Shopify 结果"}));
 await screen.findByRole("link", {name: "在 Shopify 后台打开商品"});
 fireEvent.click(screen.getByRole("button", {name: "查看 Shopify 结果"}));
 await screen.findByRole("status");
 expect(screen.queryByRole("link", {name: "在 Shopify 后台打开商品"})).toBeNull();
 expect(screen.getByText("商品编号 12345")).toBeTruthy();
});
