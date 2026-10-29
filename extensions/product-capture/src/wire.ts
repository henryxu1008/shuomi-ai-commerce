// Wire consumer of src2b-acquisition-v1 and SRC-2B2 R1, not Product authority.
export type Attribute = { name: string; value: string };
export type Price = { amount: string; currency: string | null; minQuantity?: string | null };
export type Image = { url: string; role: string };
export type Variant = { sourceID: string | null; sku: string | null; title: string | null; attributes: Attribute[]; price: Price | null };
export type VariantV2 = Variant & { images: Image[] };
export type Evidence = {
  schemaVersion: 1; sourceURL: string; offerID: string; title: string | null; description: string | null;
  attributes: Attribute[]; variants: Variant[]; priceFacts: Price[]; images: { url: string; role: string }[];
  capturedAt: string; contentSHA256: string; parserVersion: string;
  warnings: { code: string; field: string }[]; missingFacts: { field: string; reason: string }[];
};
export type EvidenceV2 = Omit<Evidence, 'schemaVersion' | 'variants'> & { schemaVersion: 2; variants: VariantV2[] };
export type CapturePayload = { captureVersion: 1; evidence: Evidence } | { captureVersion: 2; evidence: EvidenceV2 };
// Common transport view only. Each platform owns a strict validator for the
// complete payload; this view never strips its platform-specific fields.
export type PlatformCapturePayload = { captureVersion: 1 | 2; evidence: {
  sourceURL: string; title: string | null; missingFacts: { field: string; reason: string }[];
} };
