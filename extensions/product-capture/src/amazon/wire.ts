import type { Attribute, Price } from '../wire';

export type AmazonImage = { url: string; role: 'main' | 'detail' | 'variant' | 'gallery' };
export type AmazonOffer = Price & { condition?: 'new' | 'used'; conditionNote?: string };
export type AmazonPrice = AmazonOffer & { alternative?: AmazonOffer };
export type AmazonVariant = { sourceID: string | null; sku: string | null; title: string | null; attributes: Attribute[]; price: AmazonPrice | null; images: AmazonImage[] };
export type AmazonCapturePayload = {
  captureVersion: 2; source: 'amazon'; evidence: {
    schemaVersion: 2; site: 'us' | 'ca' | 'uk' | 'de' | 'jp'; sourceID: string; sourceURL: string;
    title: string | null; description: string | null; attributes: Attribute[]; variants: AmazonVariant[];
    priceFacts: AmazonPrice[]; images: AmazonImage[]; observation: { scope: 'selected_product' | 'partial_family'; parentASIN: string | null };
    capturedAt: string; contentSHA256: string; parserVersion: 'amazon-browser-dom/v1';
    warnings: { code: string; field: string }[]; missingFacts: { field: string; reason: string }[];
  };
};
