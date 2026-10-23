import { byteLength, fail, publicImage } from '../validation';
import { amazonPageSource, validateAmazonCapture } from './validation';
import type { AmazonCapturePayload, AmazonImage } from './wire';

// Reads only visible product text and selected-product image nodes. No script
// execution, page-global state, network, credentials or variant traversal.
export async function captureAmazonDocument(doc: Document, source: string, now = new Date()): Promise<AmazonCapturePayload> {
  const identity = amazonPageSource(source);
  if (amazonPageSource(doc.URL).sourceURL !== identity.sourceURL) fail('SOURCE_CHANGED');
  let examined=0;
  const charge=()=>{if(++examined>65536)fail('CAPTURE_TOO_LARGE');};
  const text=(element:Element|null,max=8192)=>{
    if(!element)return null;let result='';let bytes=0;
    const walker=doc.createTreeWalker(element,4);let node:Node|null;
    while((node=walker.nextNode())){charge();const value=node.nodeValue??'';if(value.length>max)fail('CAPTURE_TOO_LARGE');bytes+=byteLength(value);if(bytes>max)fail('CAPTURE_TOO_LARGE');result+=value;}
    return result.trim()||null;
  };
  const value = (selector: string,max=8192) => text(doc.querySelector(selector),max);
  const title = value('#productTitle',2000); if (!title) fail('UNSUPPORTED_PAGE');
  const observedASIN = doc.querySelector<HTMLInputElement>('input#ASIN')?.value;
  if (observedASIN && observedASIN !== identity.sourceID) fail('SOURCE_CHANGED');
  const images: AmazonImage[] = [];
  const main = doc.querySelector<HTMLImageElement>('#landingImage, #imgBlkFront');
  const mainURL = main?.getAttribute('data-old-hires') || main?.getAttribute('src');
  if(mainURL && mainURL.length>2048)fail('CAPTURE_TOO_LARGE');
  if (mainURL && publicImage(mainURL)) images.push({ url: mainURL, role: 'main' });
  const attributes: { name: string; value: string }[] = [];
  const variationWalker=doc.createTreeWalker(doc.body,1);let current:Node|null;let variationCount=0;
  while((current=variationWalker.nextNode())) {
    charge();const node=current as Element;if(!node.id.startsWith('variation_'))continue;
    if(++variationCount>256)fail('CAPTURE_TOO_LARGE');
    const name = text(node.querySelector('.a-form-label'))?.replace(/:$/, '');
    const selected = text(node.querySelector('.selection'));
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
