import { fail, publicImage } from '../validation';
import { amazonPageSource, validateAmazonCapture } from './validation';
import type { AmazonCapturePayload, AmazonImage } from './wire';

// Reads only visible product text and selected-product image nodes. No script
// execution, page-global state, network, credentials or variant traversal.
export async function captureAmazonDocument(doc: Document, source: string, now = new Date()): Promise<AmazonCapturePayload> {
  const identity = amazonPageSource(source);
  if (amazonPageSource(doc.URL).sourceURL !== identity.sourceURL) fail('SOURCE_CHANGED');
  const value = (selector: string) => doc.querySelector(selector)?.textContent?.trim() || null;
  const title = value('#productTitle'); if (!title) fail('UNSUPPORTED_PAGE');
  const observedASIN = doc.querySelector<HTMLInputElement>('input#ASIN')?.value;
  if (observedASIN && observedASIN !== identity.sourceID) fail('SOURCE_CHANGED');
  const images: AmazonImage[] = [];
  const main = doc.querySelector<HTMLImageElement>('#landingImage, #imgBlkFront');
  const mainURL = main?.getAttribute('data-old-hires') || main?.getAttribute('src');
  if (mainURL && publicImage(mainURL)) images.push({ url: mainURL, role: 'main' });
  const attributes: { name: string; value: string }[] = [];
  for (const node of doc.querySelectorAll('[id^="variation_"]')) {
    const name = node.querySelector('.a-form-label')?.textContent?.trim().replace(/:$/, '');
    const selected = node.querySelector('.selection')?.textContent?.trim();
    if (name && selected && !attributes.some(v => v.name === name)) attributes.push({ name, value: selected });
  }
  const rawPrice = value('#corePrice_feature_div .a-price .a-offscreen') || value('#corePriceDisplay_desktop_feature_div .a-price .a-offscreen');
  let price: { amount: string; currency: string | null } | null = null;
  // Reject ambiguous ranges/locale grouping. Decimal strings never pass through Number.
  if (rawPrice) {
    const normalized = rawPrice.replace(/\s/g, '').replace(/^(?:US\$|CA\$|CDN\$|\$|£|€|￥|¥)/, '').replace(/(?:€|円)$/, '');
    const amount = identity.site === 'de' ? normalized.replace(',', '.') : normalized;
    const explicitCurrency = doc.querySelector('meta[itemprop=priceCurrency]')?.getAttribute('content');
    const currency = explicitCurrency && /^[A-Z]{3}$/.test(explicitCurrency) ? explicitCurrency : /^US\$/.test(rawPrice) ? 'USD' : /^(?:CA\$|CDN\$)/.test(rawPrice) ? 'CAD' : /£/.test(rawPrice) ? 'GBP' : /€/.test(rawPrice) ? 'EUR' : /[￥¥円]/.test(rawPrice) ? 'JPY' : null;
    if (/^(0|[1-9][0-9]{0,15})(\.[0-9]{1,8})?$/.test(amount)) price = { amount, currency };
  }
  const missingFacts = [{ field: 'variants', reason: 'only selected product observed; variant family not traversed' }, { field: 'inventory', reason: 'source inventory quantity not observed' }];
  if (!images.length) missingFacts.push({ field: 'images', reason: 'no admissible selected product image observed' });
  if (price && !price.currency) missingFacts.push({ field: 'price.currency', reason: 'source currency not explicit' });
  if (!price) missingFacts.push({ field: 'price', reason: 'exact unambiguous price not observed' });
  const payload: AmazonCapturePayload = { captureVersion: 2, source: 'amazon', evidence: {
    schemaVersion: 2, ...identity, title, description: value('#productDescription'), attributes: [],
    variants: [{ sourceID: identity.sourceID, sku: null, title: null, attributes, price, images }],
    priceFacts: price ? [price] : [], images, observation: { scope: 'selected_product', parentASIN: null },
    capturedAt: now.toISOString(), contentSHA256: '0'.repeat(64), parserVersion: 'amazon-browser-dom/v1', warnings: [], missingFacts,
  } };
  const projection = JSON.stringify(payload.evidence);
  const hash = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(projection));
  payload.evidence.contentSHA256 = Array.from(new Uint8Array(hash), b => b.toString(16).padStart(2, '0')).join('');
  return validateAmazonCapture(payload);
}
