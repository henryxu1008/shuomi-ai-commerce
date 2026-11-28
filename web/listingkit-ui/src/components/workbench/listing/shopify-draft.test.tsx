import {cleanup, fireEvent, render, screen, waitFor} from "@testing-library/react";
import {afterEach, expect, it, vi} from "vitest";
import {confirmShopifyDraftMedia, previewShopifyDraftMedia, readSupplyPublicationStatus, shopifySupplyRules, supplyInventory} from "@/lib/api/supply-chain";
import {SupplyAPIError} from "@/lib/api/supply-error";
import {publicationSchema, type SupplySourceDetail} from "@/lib/contracts/supply-chain";
import {ShopifyDraftEditor, ShopifyDraftResult} from "./shopify-draft";
import {ShopifyMediaPreviewPanel} from "./shopify-media-preview";
import {shopifyMediaPreviewSchema} from "@/lib/contracts/shopify-media-preview";
import type {ListingCommandState} from "./use-listing-command";

vi.mock("@/lib/api/supply-chain", () => ({
 confirmShopifyDraftMedia: vi.fn(),
 previewShopifyDraftMedia: vi.fn(),
 readSupplyPublicationStatus: vi.fn(),
 shopifySupplyRules: vi.fn(),
 supplyInventory: vi.fn(),
}));
afterEach(() => {cleanup(); sessionStorage.clear(); vi.restoreAllMocks(); vi.resetAllMocks()});

const sourceId = "11111111-1111-4111-8111-111111111111";
const storeId = "22222222-2222-4222-8222-222222222222";
const productId = "gid://shopify/Product/12345";
const receipt = {
 sourceId, storeId, receiptId: "receipt-1", observedAt: "2026-11-24T02:00:00Z",
 shopify: {id: productId, status: "DRAFT" as const},
};
const readback = {...receipt, shopifyStatus: receipt.shopify};
const adminUrl = "https://fixture-shop.myshopify.com/admin/products/12345";

it("previews the original draft and six variants with shared color covers", async () => {
 const colors = ["黑色","灰色","白色"];
 const images = colors.flatMap((color, group) => [0,1,2,3].map(position => ({
  id: "fixture-"+group+"-"+position,
  url: "https://fixture-images.test/"+group+"-"+position+".png",
  color, role: position === 0 ? "main" : "gallery", state: "present",
 })));
 const variants = colors.flatMap((_, group) => ["S","M"].map(size => ({
  sku: "FIXTURE-"+group+"-"+size, state: "same",
  currentAssetId: images[group*4]?.id, assetId: images[group*4]?.id,
 })));
 const preview = shopifyMediaPreviewSchema.parse({sourceId,storeId,title:"测试外套",adminUrl,galleryCount:27,images,variants});
 const scope = {organizationId:"org",userId:"user"};
 vi.mocked(previewShopifyDraftMedia).mockResolvedValue(preview);
 render(<ShopifyMediaPreviewPanel scope={scope} sourceId={sourceId} storeId={storeId}
  storeName="开发店" disabled={false} canConfirm initialProduct="12345"/>);
 fireEvent.click(screen.getByRole("button",{name:"预览图片"}));
 expect(await screen.findByText(/已采用图片与 Shopify 图库、变体配图一致/)).toHaveTextContent("图库现有 27 项，工作台已采用 12 张");
 expect(screen.getAllByText("配图一致")).toHaveLength(6);
 expect(screen.queryByRole("button",{name:"确认同步到 Shopify"})).toBeNull();
 expect(screen.getByRole("link",{name:"在 Shopify 后台打开原草稿"})).toHaveAttribute("href",adminUrl);
 expect(previewShopifyDraftMedia).toHaveBeenCalledExactlyOnceWith(scope,{sourceId,storeId,product:"12345"},expect.any(AbortSignal));
});

const mediaScope = {organizationId:"org",userId:"user"};
const trialPreview = shopifyMediaPreviewSchema.parse({
 sourceId,storeId,title:"同步测试草稿",adminUrl,galleryCount:4,confirmationHash:"a".repeat(64),
 images:[{id:"fixture",url:"https://fixture-images.test/cover.png",role:"main",state:"missing"}],
 variants:[{sku:"FIXTURE-S",assetId:"fixture",state:"different"}],
});
const mediaPanel = (product = "12345") => <ShopifyMediaPreviewPanel scope={mediaScope}
 sourceId={sourceId} storeId={storeId} storeName="开发店" disabled={false} canConfirm initialProduct={product}/>;

it("continues the original Shopify operation after a lost response and panel reopen", async () => {
 const submit = vi.mocked(confirmShopifyDraftMedia);
 submit.mockRejectedValueOnce(new SupplyAPIError("OUTCOME_UNKNOWN",503))
  .mockImplementation(async (_scope,_input,key) => ({operationId:key,status:"complete"}));
 vi.mocked(previewShopifyDraftMedia).mockResolvedValueOnce(trialPreview)
  .mockResolvedValue({...trialPreview,confirmationHash:undefined});
 const pane = render(mediaPanel());
 fireEvent.click(screen.getByRole("button",{name:"预览图片"}));
 fireEvent.click(await screen.findByRole("button",{name:"确认同步到 Shopify"}));
 await screen.findByText(/上次同步结果待核实/);
 expect(submit).toHaveBeenCalledOnce();
 pane.unmount();

 render(mediaPanel(adminUrl));
 fireEvent.click(screen.getByRole("button",{name:"预览图片"}));
 const resume = await screen.findByRole("button",{name:"继续原同步"});
 expect(submit).toHaveBeenCalledOnce();
 fireEvent.click(resume);
 await screen.findByText("Shopify 草稿的图片已同步。");
 expect(submit.mock.calls[1]?.slice(0,3)).toEqual(submit.mock.calls[0]?.slice(0,3));
 expect(submit.mock.calls[1]?.[1]).toEqual({sourceId,storeId,product:"12345",confirmationHash:"a".repeat(64)});
 fireEvent.click(screen.getByRole("button",{name:"再次核对原同步"}));
 await screen.findByText("Shopify 草稿的图片已同步。");
 expect(submit.mock.calls[2]?.slice(0,3)).toEqual(submit.mock.calls[0]?.slice(0,3));
});

it("does not send when it cannot save the original Shopify operation", async () => {
 vi.mocked(previewShopifyDraftMedia).mockResolvedValue(trialPreview);
 vi.spyOn(Storage.prototype,"setItem").mockImplementation(() => {throw new Error("storage full")});
 render(mediaPanel());
 fireEvent.click(screen.getByRole("button",{name:"预览图片"}));
 fireEvent.click(await screen.findByRole("button",{name:"确认同步到 Shopify"}));
 expect(await screen.findByRole("alert")).toHaveTextContent(/无法保存同步记录/);
 expect(confirmShopifyDraftMedia).not.toHaveBeenCalled();
});

it("preserves an expired operation after reopening without offering a new send", async () => {
 const submit = vi.mocked(confirmShopifyDraftMedia);
 submit.mockImplementation(async (_scope,_input,key) => ({operationId:key,status:"expired"}));
 vi.mocked(previewShopifyDraftMedia).mockResolvedValue(trialPreview);
 const pane = render(mediaPanel());
 fireEvent.click(screen.getByRole("button",{name:"预览图片"}));
 fireEvent.click(await screen.findByRole("button",{name:"确认同步到 Shopify"}));
 await screen.findByText(/原确认已过期/);
 pane.unmount();
 render(mediaPanel());
 fireEvent.click(screen.getByRole("button",{name:"预览图片"}));
 await screen.findByText(/原确认已过期/);
 expect(screen.queryByRole("button",{name:/同步/})).toBeNull();
 expect(submit).toHaveBeenCalledOnce();
});

it("shows the current permission error and keeps the original request available", async () => {
 const submit = vi.mocked(confirmShopifyDraftMedia);
 submit.mockRejectedValueOnce(new SupplyAPIError("PERMISSION_DENIED",403))
  .mockImplementation(async (_scope,_input,key) => ({operationId:key,status:"pending"}));
 vi.mocked(previewShopifyDraftMedia).mockResolvedValue(trialPreview);
 render(mediaPanel());
 fireEvent.click(screen.getByRole("button",{name:"预览图片"}));
 fireEvent.click(await screen.findByRole("button",{name:"确认同步到 Shopify"}));
 expect(await screen.findByRole("alert")).toHaveTextContent(/当前没有同步图片的权限/);
 fireEvent.click(screen.getByRole("button",{name:"继续原同步"}));
 await screen.findByText(/Shopify 还在处理图片/);
 expect(submit.mock.calls[1]?.slice(0,3)).toEqual(submit.mock.calls[0]?.slice(0,3));
});

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
