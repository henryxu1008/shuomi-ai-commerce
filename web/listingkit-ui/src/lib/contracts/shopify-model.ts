import { z } from "zod";

const text = (max: number) => z.string().refine(
  value => new TextEncoder().encode(value).length <= max && !value.includes("\0"),
);

export const shopifyCurrency = z.string().regex(/^[A-Z]{3}$/);
export const shopifyPrice = z.string().regex(/^(?:0|[1-9][0-9]{0,11})(?:\.[0-9]{1,4})?$/);
export const shopifyDraft = z.strictObject({
  title: text(1024).min(1),
  descriptionHtml: text(65536).optional(),
  currency: shopifyCurrency,
  options: z.array(z.strictObject({
    name: text(1024),
    values: z.array(text(1024)).max(30),
  })).max(3).optional(),
  variants: z.array(z.strictObject({
    sku: text(1024).optional(),
    price: shopifyPrice,
    imageIndex: z.number().int().min(1).max(10).optional(),
    values: z.array(text(1024)).max(3).optional(),
  })).min(1).max(30),
  images: z.array(z.strictObject({
    url: text(2048).min(1),
    alt: text(1024).optional(),
  })).max(10).optional(),
});

export const shopifyUploadConfirmation = shopifyDraft.omit({ descriptionHtml: true }).extend({
  shopDomain: z.string().regex(/^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.myshopify\.com$/),
  description: text(32768),
  images: shopifyDraft.shape.images.refine(
    images => !images || images.every(image => URL.canParse(image.url) && new URL(image.url).protocol === "https:"),
  ),
});
