import { z } from 'zod';
import { byteLength, fail, MAX_BYTES, publicImage } from '../validation';
import type { AmazonCapturePayload } from './wire';

const domains = { us: 'www.amazon.com', ca: 'www.amazon.ca', uk: 'www.amazon.co.uk', de: 'www.amazon.de', jp: 'www.amazon.co.jp' } as const;
export function amazonPageSource(input: string) {
  if (input.length>2048 || byteLength(input) > 2048 || /[\\\u0000-\u0020]/.test(input)) fail('INVALID_PAGE');
  let url: URL; try { url = new URL(input); } catch { return fail('INVALID_PAGE'); }
  const site = (Object.keys(domains) as (keyof typeof domains)[]).find(key => url.hostname === domains[key] || url.hostname === domains[key].slice(4));
  // Bounded current product URL only; tracking query/fragment never become identity.
  const match = /^\/(?:[^/]+\/)?dp\/([A-Z0-9]{10})(?:\/)?$/.exec(url.pathname);
  if (!site || !match || url.protocol !== 'https:' || url.username || url.password || url.port) fail('INVALID_PAGE');
  return { site, sourceID: match[1], sourceURL: `https://${domains[site]}/dp/${match[1]}` };
}
const text = z.string().refine(v => byteLength(v) <= 8192 && !/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f\ud800-\udfff]/u.test(v));
const sensitive = /(?:cookie|authorization|password|passwd|token|session|profile|localstorage|sessionstorage)/i;
const attr = z.strictObject({ name: text.min(1).refine(v => !sensitive.test(v) && !v.startsWith('source.')), value: text });
const attrs = z.array(attr).max(256).refine(a => new Set(a.map(v => v.name.trim())).size === a.length);
const price = z.strictObject({ amount: text.regex(/^(0|[1-9][0-9]{0,15})(\.[0-9]{1,8})?$/), currency: text.regex(/^[A-Z]{3}$/).nullable(), minQuantity: text.regex(/^[1-9][0-9]{0,19}$/).nullable().optional() });
const image = z.strictObject({ url: text.refine(v => byteLength(v) <= 2048 && publicImage(v)), role: z.enum(['main', 'detail', 'variant', 'gallery']) });
const images = z.array(image).max(256);
const variant = z.strictObject({ sourceID: text.regex(/^[A-Z0-9]{10}$/).nullable(), sku: text.refine(v=>byteLength(v)<=200).nullable(), title: text.refine(v=>byteLength(v)<=2000).nullable(), attributes: attrs, price: price.nullable(), images });
const schema = z.strictObject({ captureVersion: z.literal(2), source: z.literal('amazon'), evidence: z.strictObject({
  schemaVersion: z.literal(2), site: z.enum(['us', 'ca', 'uk', 'de', 'jp']), sourceID: text.regex(/^[A-Z0-9]{10}$/), sourceURL: text,
  title: text.min(1).refine(v=>byteLength(v)<=2000), description: text.nullable(), attributes: attrs, variants: z.array(variant).max(256), priceFacts: z.array(price).max(256), images,
  observation: z.strictObject({ scope: z.enum(['selected_product', 'partial_family']), parentASIN: text.regex(/^[A-Z0-9]{10}$/).nullable() }),
  capturedAt: text.refine(v => /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,9})?Z$/.test(v) && Number.isFinite(Date.parse(v))),
  contentSHA256: text.regex(/^[a-f0-9]{64}$/), parserVersion: z.literal('amazon-browser-dom/v1'),
  warnings: z.array(z.strictObject({ code: text, field: text })).max(256), missingFacts: z.array(z.strictObject({ field: text, reason: text })).max(256),
}) });
export function validateAmazonCapture(value: unknown): AmazonCapturePayload {
  const parsed = schema.safeParse(value); if (!parsed.success) fail('INVALID_CAPTURE');
  const result = parsed.data; const e = result.evidence; const source = amazonPageSource(e.sourceURL);
  if (!e.title.trim() || e.site !== source.site || e.sourceID !== source.sourceID || e.sourceURL !== source.sourceURL) fail('INVALID_CAPTURE');
  if (e.observation.scope === 'selected_product' ? e.observation.parentASIN !== null || e.variants.length > 1 || e.variants.some(v => v.sourceID !== null && v.sourceID !== e.sourceID) : !e.observation.parentASIN || e.observation.parentASIN === e.sourceID) fail('INVALID_CAPTURE');
  let items = e.attributes.length + e.variants.length + e.priceFacts.length + e.images.length + e.warnings.length + e.missingFacts.length;
  const ids = new Set<string>(); const combinations = new Set<string>(); const skus = new Set<string>();
  for (const v of e.variants) {
    items += v.attributes.length + v.images.length;
    const combo = JSON.stringify([...v.attributes].sort((a, b) => a.name.localeCompare(b.name)));
    if (v.sourceID && ids.has(v.sourceID) || v.sku && skus.has(v.sku) || e.variants.length > 1 && (!v.attributes.length || combinations.has(combo)) || !v.sourceID && !v.attributes.length) fail('INVALID_CAPTURE');
    if (v.sourceID) ids.add(v.sourceID); if (v.sku) skus.add(v.sku); combinations.add(combo);
  }
  if (items > 1024 || byteLength(JSON.stringify(result)) > MAX_BYTES) fail('CAPTURE_TOO_LARGE');
  return result;
}
