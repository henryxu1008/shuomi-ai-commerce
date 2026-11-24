import {cleanup, fireEvent, render, screen, waitFor} from "@testing-library/react";
import {afterEach, expect, it, vi} from "vitest";
import {readSupplyPublicationStatus} from "@/lib/api/supply-chain";
import {publicationSchema} from "@/lib/contracts/supply-chain";
import {ShopifyDraftResult} from "./shopify-draft";

vi.mock("@/lib/api/supply-chain", () => ({readSupplyPublicationStatus: vi.fn()}));
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

it("shows the Shopify admin link after verifying the product", async () => {
 vi.mocked(readSupplyPublicationStatus).mockResolvedValue(publicationSchema.parse({...readback, shopifyAdminUrl: adminUrl}));
 render(<ShopifyDraftResult scope={{organizationId: "org", userId: "user"}} sourceId={sourceId} storeId={storeId} publication={receipt}/>);
 expect(screen.queryByRole("link", {name: "在 Shopify 后台打开商品"})).toBeNull();
 fireEvent.click(screen.getByRole("button", {name: "查看 Shopify 结果"}));
 await waitFor(() => expect(screen.getByRole("link", {name: "在 Shopify 后台打开商品"})).toHaveAttribute("href", adminUrl));
 expect(screen.getByText("商品编号 12345")).toBeTruthy();
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
