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
  const excluded = (element: Element) => {
    if (element.closest('script,style,noscript,template,[hidden],[aria-hidden="true"]')) return true;
    for (let parent: Element | null = element; parent; parent = parent.parentElement) {
      const style = (parent as HTMLElement).style;
      if (style?.display === 'none' || style?.visibility === 'hidden') return true;
    }
    return false;
  };
  const text=(element:Element|null,max=8192)=>{
    if(!element || excluded(element))return null;let result='';let bytes=0;
    const walker=doc.createTreeWalker(element,4);let node:Node|null;
    while((node=walker.nextNode())){charge();if(node.parentElement && excluded(node.parentElement))continue;const value=node.nodeValue??'';if(value.length>max)fail('CAPTURE_TOO_LARGE');bytes+=byteLength(value);if(bytes>max)fail('CAPTURE_TOO_LARGE');result+=value;}
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
  if (mainURL && main && !excluded(main) && publicImage(mainURL)) images.push({ url: mainURL, role: 'main' });
  const selectedImages = [...images];
  const gallery = doc.querySelectorAll<HTMLImageElement>('#altImages li img');
  if (gallery.length > 256) fail('CAPTURE_TOO_LARGE');
  for (const image of gallery) {
    charge();
    if (excluded(image) || image.closest('.videoThumbnail,.video-block,[data-video-url],[data-video-id]')) continue;
    const url = image.getAttribute('src');
    if (url && url.length > 2048) fail('CAPTURE_TOO_LARGE');
    if (url && publicImage(url) && !images.some(item => item.url === url)) images.push({ url, role: 'gallery' });
  }
  const productFacts = new Map<string, { name: string; value: string }>();
  const factConflicts = new Set<string>();
  const fact = (name: string | null, value: string | null) => {
    name = name?.replace(/\s+/g, ' ').replace(/[:：]$/, '').trim() || null;
    value = value?.replace(/\s+/g, ' ').trim() || null;
    if (!name || !value) return;
    const key = name.toLowerCase();
    if (factConflicts.has(key)) return;
    if (productFacts.has(key) && productFacts.get(key)!.value !== value) { productFacts.delete(key); factConflicts.add(key); }
    else productFacts.set(key, { name, value });
  };
  const rows = doc.querySelectorAll('#productOverview_feature_div tr,#productDetails_detailBullets_sections1 tr,#productDetails_techSpec_section_1 tr');
  if (rows.length > 256) fail('CAPTURE_TOO_LARGE');
  for (const row of rows) {
    charge(); if (excluded(row)) continue;
    const cells = row.querySelectorAll('th,td');
    if (cells.length === 2) fact(text(cells[0]), text(cells[1]));
  }
  fact('Availability', value('#availability'));
  const dimensions = new Map<string, { name: string; value: string }>();
  const conflicts = new Set<string>();
  let incompleteDimension = false;
  const dimension = (label: Element | null, selection: Element | null) => {
    const name = text(label)?.replace(/\s+/g, ' ').replace(/[:：]$/, '').trim();
    const selected = text(selection)?.replace(/\s+/g, ' ');
    if (!name || !selected) { incompleteDimension = true; return; }
    const key = name.toLowerCase();
    if (conflicts.has(key)) return;
    const previous = dimensions.get(key);
    if (previous && previous.value !== selected) { dimensions.delete(key); conflicts.add(key); }
    else if (!previous) dimensions.set(key, { name, value: selected });
  };
  let description = value('#productDescription') ?? '';
  let descriptionBytes = byteLength(description);
  const bullets = doc.querySelector('#feature-bullets');
  let bulletCount = 0; let observedBullets = 0;
  const variationWalker=doc.createTreeWalker(doc.body,1);let current:Node|null;let variationCount=0;
  while((current=variationWalker.nextNode())) {
    charge();const node=current as Element;
    const inline = node.id.startsWith('inline-twister-expander-header-');
    if (inline || node.id.startsWith('variation_')) {
      if(++variationCount>256)fail('CAPTURE_TOO_LARGE');
      if (!excluded(node)) {
        if (inline) {
          const selected = node.querySelector('.inline-twister-dim-title-value');
          // The source label and its selected value share the same header;
          // option lists, aria-label prose and other ASINs are not consulted.
          dimension(selected?.previousElementSibling ?? null, selected);
        } else dimension(node.querySelector('.a-form-label'), node.querySelector('.selection'));
      }
    }
    if (node.tagName === 'LI' && bullets?.contains(node)) {
      if (++bulletCount > 256) fail('CAPTURE_TOO_LARGE');
      const bullet = text(node.querySelector('.a-list-item') ?? node)?.replace(/\s+/g, ' ');
      if (bullet) {
        const separator = observedBullets ? '\n' : description ? '\n\n' : '';
        descriptionBytes += byteLength(separator) + byteLength(bullet);
        if (descriptionBytes > 8192) fail('CAPTURE_TOO_LARGE');
        description += separator + bullet;
        observedBullets++;
      }
    }
  }
  const attributes = [...dimensions.values()];
  const priceNodes = doc.querySelectorAll('#corePrice_feature_div .a-price .a-offscreen,#corePriceDisplay_desktop_feature_div .a-price .a-offscreen');
  if (priceNodes.length > 256) fail('CAPTURE_TOO_LARGE');
  const currentPrices = new Set<string>();
  for (const node of priceNodes) {
    if (node.closest('.a-text-price,s,del')) continue;
    const amount = text(node); if (amount) currentPrices.add(amount);
  }
  const rawPrice = currentPrices.size === 1 ? [...currentPrices][0] : null;
  let price: { amount: string; currency: string | null } | null = null;
  let currencyConflict = false;
  // Reject ambiguous ranges/locale grouping. Decimal strings never pass through Number.
  if (rawPrice) {
    const normalized = rawPrice.replace(/\s/g, '').replace(/^(?:US\$|CA\$|CDN\$|\$|£|€|￥|¥)/, '').replace(/(?:€|円)$/, '');
    const amount = identity.site === 'de' ? normalized.replace(',', '.') : normalized;
    const explicitCurrency = doc.querySelector('meta[itemprop=priceCurrency]')?.getAttribute('content');
    const symbolCurrency = /^US\$/.test(rawPrice) ? 'USD' : /^(?:CA\$|CDN\$)/.test(rawPrice) ? 'CAD' : /£/.test(rawPrice) ? 'GBP' : /€/.test(rawPrice) ? 'EUR' : /円/.test(rawPrice) ? 'JPY' : null;
    const metadataCurrency = explicitCurrency && /^[A-Z]{3}$/.test(explicitCurrency) ? explicitCurrency : null;
    currencyConflict = Boolean(metadataCurrency && symbolCurrency && metadataCurrency !== symbolCurrency);
    const currency = currencyConflict ? null : metadataCurrency || symbolCurrency;
    if (/^(0|[1-9][0-9]{0,15})(\.[0-9]{1,8})?$/.test(amount)) price = { amount, currency };
  }
  const missingFacts = [{ field: 'variants', reason: 'only selected product observed; variant family not traversed' }, { field: 'inventory', reason: 'source inventory quantity not observed' }];
  if (!attributes.length || incompleteDimension) missingFacts.push({ field: 'variants[0].attributes', reason: 'selected dimension label or value not observed' });
  if (conflicts.size) missingFacts.push({ field: 'variants[0].attributes', reason: 'conflicting selected dimension values were omitted' });
  if (!observedBullets) missingFacts.push({ field: 'description.featureBullets', reason: 'feature bullet text not observed' });
  if (!description) missingFacts.push({ field: 'description', reason: 'product description and feature bullet text not observed' });
  if (!images.length) missingFacts.push({ field: 'images', reason: 'no admissible selected product image observed' });
  if (!productFacts.size) missingFacts.push({ field: 'attributes', reason: 'rendered product attribute rows not observed' });
  if (!productFacts.has('availability')) missingFacts.push({ field: 'availability', reason: 'source availability text not observed' });
  if (factConflicts.size) missingFacts.push({ field: 'attributes', reason: 'conflicting product attribute values were omitted' });
  if (price && !price.currency) missingFacts.push({ field: 'price.currency', reason: currencyConflict ? 'conflicting explicit source currencies were omitted' : 'source currency not explicit' });
  if (!price) missingFacts.push({ field: 'price', reason: 'exact unambiguous price not observed' });
  const payload: AmazonCapturePayload = { captureVersion: 2, source: 'amazon', evidence: {
    schemaVersion: 2, ...identity, title, description: description || null, attributes: [...productFacts.values()],
    variants: [{ sourceID: identity.sourceID, sku: null, title: null, attributes, price, images: selectedImages }],
    priceFacts: price ? [price] : [], images, observation: { scope: 'selected_product', parentASIN: null },
    capturedAt: now.toISOString(), contentSHA256: '0'.repeat(64), parserVersion: 'amazon-browser-dom/v1', warnings: [], missingFacts,
  } };
  const projection = JSON.stringify(payload.evidence);
  const hash = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(projection));
  payload.evidence.contentSHA256 = Array.from(new Uint8Array(hash), b => b.toString(16).padStart(2, '0')).join('');
  return validateAmazonCapture(payload);
}
