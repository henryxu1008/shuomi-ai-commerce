import { byteLength, fail, publicImage } from '../validation';
import { amazonPageSource, validateAmazonCapture } from './validation';
import type { AmazonCapturePayload, AmazonImage } from './wire';
import { excluded, productText, tableFacts, putFact, detailImages } from '../dom-materials';
import { albumURLs } from './album';
import { amazonListFacts, amazonVisibleTitle } from './product-details';

// Reads product DOM and statically bound album literals. No script execution,
// page-global state, network, credentials or variant traversal.
export async function captureAmazonDocument(doc: Document, source: string, now = new Date()): Promise<AmazonCapturePayload> {
  const identity = amazonPageSource(source);
  if (amazonPageSource(doc.URL).sourceURL !== identity.sourceURL) fail('SOURCE_CHANGED');
  let examined=0; const started = performance.now();
  const charge=()=>{if(++examined>65536)fail('CAPTURE_TOO_LARGE');if(performance.now()-started>8000)fail('CAPTURE_TIMEOUT');};
  const text=(element:Element|null,max=8192)=>productText(doc,element,charge,max);
  const fieldText=(element:Element|null)=>productText(doc,element,charge,8192,true);
  const value = (selector: string,max=8192) => text(doc.querySelector(selector),max);
  const titleElement = doc.querySelector('#productTitle');
  const primaryTitle = text(titleElement,2000); if (!primaryTitle) fail('UNSUPPORTED_PAGE');
  const observedASIN = doc.querySelector<HTMLInputElement>('input#ASIN')?.value;
  if (observedASIN && observedASIN !== identity.sourceID) fail('SOURCE_CHANGED');
  const { title, reason: subtitleReason } = amazonVisibleTitle(doc, titleElement!, primaryTitle, charge);
  const images: AmazonImage[] = [];
  const main = doc.querySelector<HTMLImageElement>('#landingImage, #imgBlkFront');
  const mainURL = main?.getAttribute('data-old-hires') || main?.getAttribute('src');
  if(mainURL && mainURL.length>2048)fail('CAPTURE_TOO_LARGE');
  if (mainURL && main && !excluded(main) && publicImage(mainURL)) images.push({ url: mainURL, role: 'main' });
  const selectedImages = [...images];
  const gallery = doc.querySelectorAll<HTMLImageElement>('#altImages li img');
  if (gallery.length > 256) fail('CAPTURE_TOO_LARGE');
  const thumbs: string[] = [];
  for (const image of gallery) {
    charge();
    if (excluded(image) || image.closest('.videoThumbnail,.video-block,[data-video-url],[data-video-id]')) continue;
    const url = image.getAttribute('src');
    if (url && byteLength(url) > 2048) fail('CAPTURE_TOO_LARGE');
    if (url && publicImage(url) && !thumbs.includes(url)) thumbs.push(url);
  }
  const album = albumURLs(doc, images[0]?.url, thumbs, charge);
  for (const thumb of thumbs) {
    const url = album.urls.get(thumb) ?? thumb;
    if (!images.some(item => item.url === url)) {
      if (images.length >= 256) fail('CAPTURE_TOO_LARGE');
      images.push({ url, role: 'gallery' });
    }
  }
  for (const url of detailImages(doc, '#productDescription img,#aplus img,#aplus_feature_div img,.aplus-v2 .aplus-module-content img', charge, true)) {
    if (!images.some(item => item.url === url)) {
      if (images.length >= 256) fail('CAPTURE_TOO_LARGE');
      images.push({ url, role: 'detail' });
    }
  }
  const productFacts = new Map<string, { name: string; value: string }>();
  const factConflicts = new Set<string>();
  const fact = (name: string | null, value: string | null) => {
    putFact(productFacts, factConflicts, name, value);
  };
  for (const item of tableFacts(doc, '#productOverview_feature_div tr,#productDetails_detailBullets_sections1 tr,#productDetails_techSpec_section_1 tr,#technicalSpecifications_section_1 tr', charge)) fact(item.name,item.value);
  for (const item of amazonListFacts(doc, charge)) fact(item.name,item.value);
  fact('Availability', fieldText(doc.querySelector('#availability')));
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
  const paragraphs = doc.querySelectorAll('#aplus .aplus-module p,#aplus_feature_div .celwidget p,.aplus-v2 .aplus-module-content p');
  if (paragraphs.length > 256) fail('CAPTURE_TOO_LARGE');
  const knownParagraphs = new Set(description.split(/\n+/).map(p=>p.replace(/\s+/g,' ').trim()));
  let addedParagraphs = 0;
  for (const paragraph of paragraphs) {
    charge();
    const value = text(paragraph)?.replace(/\s+/g,' ');
    if (!value || knownParagraphs.has(value)) continue;
    const separator = description ? addedParagraphs ? '\n' : '\n\n' : '';
    descriptionBytes += byteLength(separator) + byteLength(value);
    if (descriptionBytes > 8192) fail('CAPTURE_TOO_LARGE');
    description += separator + value; knownParagraphs.add(value); addedParagraphs++;
  }
  const attributes = [...dimensions.values()];
  const priceNodes = doc.querySelectorAll('#corePrice_feature_div .a-price .a-offscreen,#corePriceDisplay_desktop_feature_div .a-price .a-offscreen');
  if (priceNodes.length > 256) fail('CAPTURE_TOO_LARGE');
  const currentPrices = new Set<string>();
  const currencies = new Set<string>();
  const priceRoots = new Set<Element>();
  for (const node of priceNodes) {
    if (node.closest('.a-text-price,s,del')) continue;
    const amount = fieldText(node); if (!amount) continue;
    currentPrices.add(amount);
    const root = node.closest('#corePrice_feature_div,#corePriceDisplay_desktop_feature_div');
    if (!root || priceRoots.has(root)) continue;
    priceRoots.add(root);
    const metadata = root.querySelectorAll('meta[itemprop=priceCurrency]');
    if (metadata.length > 256) fail('CAPTURE_TOO_LARGE');
    for (const meta of metadata) {
      charge(); if (excluded(meta,root)) continue;
      const currency = meta.getAttribute('content');
      if (currency && /^[A-Z]{3}$/.test(currency)) currencies.add(currency);
    }
  }
  const rawPrice = currentPrices.size === 1 ? [...currentPrices][0] : null;
  let price: { amount: string; currency: string | null } | null = null;
  let currencyConflict = false;
  // Reject ambiguous ranges/locale grouping. Decimal strings never pass through Number.
  if (rawPrice) {
    const normalized = rawPrice.replace(/\s/g, '').replace(/^(?:US\$|CA\$|CDN\$|\$|£|€|￥|¥)/, '').replace(/(?:€|円)$/, '');
    const amount = identity.site === 'de' ? normalized.replace(',', '.') : normalized;
    const symbolCurrency = /^US\$/.test(rawPrice) ? 'USD' : /^(?:CA\$|CDN\$)/.test(rawPrice) ? 'CAD' : /£/.test(rawPrice) ? 'GBP' : /€/.test(rawPrice) ? 'EUR' : /円/.test(rawPrice) ? 'JPY' : null;
    if (symbolCurrency) currencies.add(symbolCurrency);
    currencyConflict = currencies.size > 1;
    const currency = currencies.size === 1 ? [...currencies][0] : null;
    if (/^(0|[1-9][0-9]{0,15})(\.[0-9]{1,8})?$/.test(amount)) price = { amount, currency };
  }
  const missingFacts = [{ field: 'variants', reason: 'only selected product observed; variant family not traversed' }, { field: 'inventory', reason: 'source inventory quantity not observed' }];
  if (subtitleReason) missingFacts.push({ field: 'title.subtitle', reason: subtitleReason });
  if (album.reason) missingFacts.push({ field: 'images.hiRes', reason: album.reason });
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
