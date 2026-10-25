import { excluded, productText } from '../dom-materials';
import { byteLength, fail, validPriceAmount } from '../validation';
import type { Attribute, Price } from '../wire';
import type { AmazonCapturePayload } from './wire';

type Site = AmazonCapturePayload['evidence']['site'];
const savingsTemplate = '{priceToPay} with {savings} percent savings';
const priceRoots = '#corePrice_feature_div,#corePriceDisplay_desktop_feature_div';
const unitPrice = '.pricePerUnit,.a-price-per-unit';
const referencePrice = '.a-text-price,s,del';

function amountText(raw: string, site: Site): string | null {
  const normalized = raw.replace(/\s/g, '').replace(/^(?:US\$|CA\$|CDN\$|\$|£|€|￥|¥)/, '').replace(/(?:€|円)$/, '');
  const amount = site === 'de' ? normalized.replace(',', '.') : normalized;
  return /^(0|[1-9][0-9]{0,15})(\.[0-9]{1,8})?$/.test(amount) ? amount : null;
}

// An explicit product tuple is separate from the general text reader: only
// these named product inputs may contribute currency, never other form data.
function productFormCurrency(doc: Document, sourceID: string, amount: string, charge: () => void): string | null {
  const forms = doc.querySelectorAll<HTMLFormElement>('form#addToCart');
  if (forms.length > 256) fail('CAPTURE_TOO_LARGE');
  if (forms.length !== 1) return null;
  const form = forms[0]; charge();
  const blockedSelf = (node: HTMLElement) => node.matches('[hidden],[aria-hidden="true"],.recommendations,.aplus-comparison-table,[data-cel-widget*="recommend" i],[data-cel-widget*="comparison" i]')
    || node.style.display === 'none' || node.style.visibility === 'hidden';
  if (!form.parentElement || excluded(form.parentElement) || blockedSelf(form)) return null;
  const read = (name: string, optional = false): string | null | undefined => {
    const nodes = form.querySelectorAll<HTMLInputElement>(`input[name="${name}"]`); charge();
    if (nodes.length > 256) fail('CAPTURE_TOO_LARGE');
    if (optional && !nodes.length) return undefined;
    if (nodes.length !== 1) return null;
    const node = nodes[0]; const parent = node.parentElement;
    if (node.type !== 'hidden' || node.form !== form || !parent || blockedSelf(node)
      || parent !== form && excluded(parent, parent)) return null;
    const value = node.value;
    if (byteLength(value) > 8192) fail('CAPTURE_TOO_LARGE');
    return value;
  };
  const currency = read('items[0.base][customerVisiblePrice][currencyCode]');
  const declaredAmount = read('items[0.base][customerVisiblePrice][amount]');
  const asin = read('items[0.base][asin]'); const extraASIN = read('ASIN', true);
  if (!currency || !/^[A-Z]{3}$/.test(currency) || !declaredAmount || !validPriceAmount(declaredAmount)
    || asin !== sourceID || extraASIN !== undefined && extraASIN !== sourceID) return null;
  const decimalKey = (value: string) => {
    const [whole, fraction = ''] = value.split('.');
    return `${whole}.${fraction.replace(/0+$/, '')}`;
  };
  return decimalKey(declaredAmount) === decimalKey(amount) ? currency : null;
}

// Current and reference prices share only their bounded DOM reader. A reference
// label remains an ordinary source attribute, never a current quote candidate.
export function amazonPrice(doc: Document, site: Site, sourceID: string, charge: () => void): { price: Price | null; currencyConflict: boolean; attributes: Attribute[] } {
  const roots = doc.querySelectorAll(priceRoots);
  if (roots.length > 256) fail('CAPTURE_TOO_LARGE');
  const current = new Set<string>(); const currencies = new Set<string>(); const attributes: Attribute[] = [];
  let fields = 0; let metadataCount = 0; let boundCurrent = false;
  const text = (node: Element) => productText(doc, node, charge, 8192, true);
  const candidates = (root: Element, selector: string) => {
    const nodes = root.querySelectorAll(selector); fields += nodes.length;
    if (fields > 256) fail('CAPTURE_TOO_LARGE');
    return nodes;
  };
  for (const root of roots) {
    charge();
    const declared = root.getAttribute('data-csa-c-asin');
    if (declared && declared !== sourceID || excluded(root, root)) continue;
    const belongs = (node: Element) => node.closest(priceRoots) === root && !node.closest(unitPrice);
    let observedCurrent = false;
    for (const node of candidates(root, '.a-price .a-offscreen')) {
      charge(); if (!belongs(node) || node.closest(referencePrice)) continue;
      const raw = text(node); if (!raw) continue;
      current.add(raw); observedCurrent = true;
    }
    for (const node of candidates(root, '#apex-pricetopay-accessibility-label')) {
      charge(); if (!belongs(node) || node.closest(referencePrice) || node.getAttribute('data-pricetopay-label') !== '{priceToPay}') continue;
      let raw = text(node)?.replace(/\s+/g, ' ').trim(); if (!raw) continue;
      if (node.getAttribute('data-pricetopay-savings-label') === savingsTemplate) {
        raw = raw.replace(/ with [0-9]+(?:\.[0-9]+)? percent savings$/, '');
      }
      if (amountText(raw, site) === null) continue;
      current.add(raw); observedCurrent = true;
    }
    if (observedCurrent) {
      if (declared === sourceID) boundCurrent = true;
      const metadata = root.querySelectorAll('meta[itemprop=priceCurrency]'); metadataCount += metadata.length;
      if (metadataCount > 256) fail('CAPTURE_TOO_LARGE');
      for (const node of metadata) {
        charge(); if (!belongs(node) || node.closest(referencePrice) || excluded(node, root)) continue;
        const currency = node.getAttribute('content');
        if (currency && /^[A-Z]{3}$/.test(currency)) currencies.add(currency);
      }
    }
    for (const node of candidates(root, '.apex-basisprice-offscreen-label[data-basisprice-label="{label} {price}"]')) {
      charge(); if (!belongs(node)) continue;
      const raw = text(node)?.replace(/\s+/g, ' ').trim();
      const reference = raw?.match(/^List Price:\s*(.+)$/);
      if (reference && amountText(reference[1], site) !== null) attributes.push({ name: 'Source reference price', value: raw! });
    }
  }
  const raw = current.size === 1 ? [...current][0] : null;
  let price: Price | null = null; let currencyConflict = false;
  if (raw) {
    const amount = amountText(raw, site);
    if (amount !== null && boundCurrent) {
      const currency = productFormCurrency(doc, sourceID, amount, charge);
      if (currency) currencies.add(currency);
    }
    const symbolCurrency = /^US\$/.test(raw) ? 'USD' : /^(?:CA\$|CDN\$)/.test(raw) ? 'CAD' : /£/.test(raw) ? 'GBP' : /€/.test(raw) ? 'EUR' : /円/.test(raw) ? 'JPY' : null;
    if (symbolCurrency) currencies.add(symbolCurrency);
    currencyConflict = currencies.size > 1;
    if (amount !== null) price = { amount, currency: currencies.size === 1 ? [...currencies][0] : null };
  }
  return { price, currencyConflict, attributes };
}
