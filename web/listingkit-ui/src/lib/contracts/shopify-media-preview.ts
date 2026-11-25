import {z} from "zod";
import {collectionID} from "./product-collection";

const shopifyProductPath = /^\/admin\/products\/([1-9][0-9]{0,19})$/;
const shopifyStorePath = /^\/store\/[a-z0-9-]+\/products\/([1-9][0-9]{0,19})$/;
const shopifyHost = /^[a-z0-9][a-z0-9-]*\.myshopify\.com$/;

function productNumber(value: string): string | undefined {
 const text = value.trim();
 if (/^[1-9][0-9]{0,19}$/.test(text)) return text;
 try {
  const url = new URL(text);
  if (url.protocol !== "https:" || url.username || url.password) return;
  if (shopifyHost.test(url.host)) return shopifyProductPath.exec(url.pathname)?.[1];
  if (url.host === "admin.shopify.com") return shopifyStorePath.exec(url.pathname)?.[1];
 } catch {
  return;
 }
}

export const shopifyMediaPreviewInput = z.strictObject({
 sourceId:collectionID,
 storeId:collectionID,
 product:z.string().trim().max(512).refine(value => !!productNumber(value)),
});

const assetId = z.string().min(1).max(128);
export const shopifyMediaPreviewSchema = z.strictObject({
 sourceId:collectionID,
 storeId:collectionID,
 title:z.string().min(1).max(512),
 adminUrl:z.string().url().refine(value => {
  const url = new URL(value);
  return url.protocol === "https:" && !url.username && !url.password && !url.search && !url.hash &&
   shopifyHost.test(url.host) && shopifyProductPath.test(url.pathname);
 }),
 galleryCount:z.number().int().min(0).max(60),
 images:z.array(z.strictObject({
  id:assetId,
  url:z.string().url().max(2048).refine(value => value.startsWith("https://")),
  color:z.string().min(1).max(255).optional(),
  role:z.enum(["main","white_background","gallery"]),
  state:z.enum(["present","missing","unverified"]),
 })).max(30),
 variants:z.array(z.strictObject({
  sku:z.string().min(1).max(128),
  assetId:assetId.optional(),
  currentAssetId:assetId.optional(),
  state:z.enum(["same","different","unverified"]),
 })).min(1).max(30),
}).superRefine((value, ctx) => {
 const ids = new Set(value.images.map(image => image.id));
 if (ids.size !== value.images.length || new Set(value.variants.map(variant => variant.sku)).size !== value.variants.length ||
     value.variants.some(variant => [variant.assetId, variant.currentAssetId].some(id => id !== undefined && !ids.has(id)))) {
  ctx.addIssue({code:"custom",message:"Image and variant references must belong to the preview"});
 }
});

export type ShopifyMediaPreview = z.infer<typeof shopifyMediaPreviewSchema>;
export type ShopifyMediaPreviewInput = z.infer<typeof shopifyMediaPreviewInput>;

export function matchesShopifyMediaPreview(result: ShopifyMediaPreview, request: ShopifyMediaPreviewInput): boolean {
 return result.sourceId === request.sourceId && result.storeId === request.storeId &&
  productNumber(result.adminUrl) === productNumber(request.product);
}
