import { excluded, productText } from '../dom-materials';
import { byteLength, fail, validPriceAmount } from '../validation';
import type { Attribute } from '../wire';
import type { AmazonCapturePayload, AmazonOffer, AmazonPrice } from './wire';

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
function productFormCurrency(scope: Document | Element, sourceID: string, amount: string, charge: () => void): { currency: string | null; conflict: boolean; currencyAmbiguous: boolean } {
  const unknown = { currency: null, conflict: false, currencyAmbiguous:false };
  const forms = scope.querySelectorAll<HTMLFormElement>('form#addToCart');
  if (forms.length > 256) fail('CAPTURE_TOO_LARGE');
  if (forms.length > 1) return {currency:null,conflict:true,currencyAmbiguous:false};
  if (!forms.length) return unknown;
  const form = forms[0]; charge();
  const blockedSelf = (node: HTMLElement) => node.matches('[hidden],[aria-hidden="true"],.recommendations,.aplus-comparison-table,[data-cel-widget*="recommend" i],[data-cel-widget*="comparison" i]')
    || node.style.display === 'none' || node.style.visibility === 'hidden';
  if (!form.parentElement || excluded(form.parentElement) || blockedSelf(form)) return unknown;
  let ambiguousIdentity = false;
  let currencyAmbiguous = false;
  const read = (name: string, optional = false): string | null | undefined => {
    const nodes = form.querySelectorAll<HTMLInputElement>(`input[name="${name}"]`); charge();
    if (nodes.length > 256) fail('CAPTURE_TOO_LARGE');
    if (optional && !nodes.length) return undefined;
    if (nodes.length !== 1) { if (nodes.length > 1) { if (name === 'items[0.base][customerVisiblePrice][currencyCode]') currencyAmbiguous = true; else ambiguousIdentity = true; } return null; }
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
  const decimalKey = (value: string) => {
    const [whole, fraction = ''] = value.split('.');
    return `${whole}.${fraction.replace(/0+$/, '')}`;
  };
  const conflict = ambiguousIdentity || !!asin && asin !== sourceID || !!extraASIN && extraASIN !== sourceID || !!declaredAmount && validPriceAmount(declaredAmount) && decimalKey(declaredAmount) !== decimalKey(amount);
  if (conflict) return {currency:null,conflict:true,currencyAmbiguous};
  if (currencyAmbiguous) return {currency:null,conflict:false,currencyAmbiguous:true};
  if (!currency || !/^[A-Z]{3}$/.test(currency) || !declaredAmount || !validPriceAmount(declaredAmount) || asin !== sourceID || extraASIN !== undefined && extraASIN !== sourceID) return unknown;
  return {currency,conflict:false,currencyAmbiguous:false};
}

// Current and reference prices share only their bounded DOM reader. A reference
// label remains an ordinary source attribute, never a current quote candidate.
export function amazonPrice(doc: Document, site: Site, sourceID: string, charge: () => void): { price: AmazonPrice | null; currencyConflict: boolean; attributes: Attribute[]; diagnostics: {field:string;reason:string}[] } {
  const roots = doc.querySelectorAll(priceRoots);
  if (roots.length > 256) fail('CAPTURE_TOO_LARGE');
  type Group = { current: Set<string>; currencies: Set<string>; bound: boolean; scopes: Set<Element> };
  const groups: Record<'new' | 'used' | 'unknown', Group> = Object.fromEntries(['new','used','unknown'].map(k => [k,{current:new Set<string>(),currencies:new Set<string>(),bound:false,scopes:new Set<Element>()}])) as Record<'new' | 'used' | 'unknown', Group>;
  const attributes: Attribute[] = [];
  let fields = 0; let metadataCount = 0;
  const text = (node: Element) => productText(doc, node, charge, 8192, true);
  const candidates = (root: Element, selector: string) => {
    const nodes = root.querySelectorAll(selector); fields += nodes.length;
    if (fields > 256) fail('CAPTURE_TOO_LARGE');
    return nodes;
  };
  for (const root of roots) {
    charge();
    const declared = root.getAttribute('data-csa-c-asin');
    if (declared !== sourceID || excluded(root, root)) continue;
    const scope = root.closest('#newAccordionRow_0,#usedAccordionRow,#apex_desktop_newAccordionRow,#apex_desktop_usedAccordionRow');
    const condition = scope?.id === 'newAccordionRow_0' || scope?.id === 'apex_desktop_newAccordionRow' ? 'new' : scope ? 'used' : 'unknown';
    const group = groups[condition];
    const belongs = (node: Element) => node.closest(priceRoots) === root && !node.closest(unitPrice);
    let observedCurrent = false;
    for (const node of candidates(root, '.a-price .a-offscreen')) {
      charge(); if (!belongs(node) || node.closest(referencePrice)) continue;
      const raw = text(node); if (!raw) continue;
      group.current.add(raw); observedCurrent = true;
    }
    for (const node of candidates(root, '#apex-pricetopay-accessibility-label')) {
      charge(); if (!belongs(node) || node.closest(referencePrice) || node.getAttribute('data-pricetopay-label') !== '{priceToPay}') continue;
      let raw = text(node)?.replace(/\s+/g, ' ').trim(); if (!raw) continue;
      if (node.getAttribute('data-pricetopay-savings-label') === savingsTemplate) {
        raw = raw.replace(/ with [0-9]+(?:\.[0-9]+)? percent savings$/, '');
      }
      if (amountText(raw, site) === null) continue;
      group.current.add(raw); observedCurrent = true;
    }
    if (observedCurrent) {
      if (declared === sourceID) group.bound = true;
      if (scope) group.scopes.add(scope);
      const metadata = root.querySelectorAll('meta[itemprop=priceCurrency]'); metadataCount += metadata.length;
      if (metadataCount > 256) fail('CAPTURE_TOO_LARGE');
      for (const node of metadata) {
        charge(); if (!belongs(node) || node.closest(referencePrice) || excluded(node, root)) continue;
        const currency = node.getAttribute('content');
        if (currency && /^[A-Z]{3}$/.test(currency)) group.currencies.add(currency);
      }
    }
    for (const node of candidates(root, '.apex-basisprice-offscreen-label[data-basisprice-label="{label} {price}"]')) {
      charge(); if (!belongs(node)) continue;
      const raw = text(node)?.replace(/\s+/g, ' ').trim();
      const reference = raw?.match(/^List Price:\s*(.+)$/);
      if (reference && amountText(reference[1], site) !== null) attributes.push({ name: 'Source reference price', value: raw! });
    }
  }
  const diagnostics: {field:string;reason:string}[] = [];
  const currencyConflicts = {new:false,used:false,unknown:false};
  const offer = (condition: 'new' | 'used' | 'unknown'): AmazonOffer | null => {
    const {current, currencies, bound, scopes} = groups[condition];
    const raw = current.size === 1 ? [...current][0] : null;
    if (!raw || !bound) {
      if (condition !== 'unknown' && current.size) diagnostics.push({field:`price.${condition}`,reason:'conflicting or unbound source offer'});
      return null;
    }
    const amount = amountText(raw, site);
    if (amount === null) return null;
    let currencyAmbiguous = false;
    if (bound) {
      for (const scope of scopes.size ? scopes : [doc]) {
        const result = productFormCurrency(scope, sourceID, amount, charge);
        if (result.conflict) { if (condition !== 'unknown') diagnostics.push({field:`price.${condition}`,reason:'conflicting or unbound source offer'}); return null; }
        if (result.currencyAmbiguous) currencyAmbiguous = true;
        if (result.currency) currencies.add(result.currency);
      }
    }
    const symbolCurrency = /^US\$/.test(raw) ? 'USD' : /^(?:CA\$|CDN\$)/.test(raw) ? 'CAD' : /£/.test(raw) ? 'GBP' : /€/.test(raw) ? 'EUR' : /円/.test(raw) ? 'JPY' : null;
    if (symbolCurrency) currencies.add(symbolCurrency);
    currencyConflicts[condition] = currencyAmbiguous || currencies.size > 1;
    if (condition !== 'unknown' && (currencyAmbiguous || currencies.size !== 1)) diagnostics.push({field:`price.${condition}.currency`,reason:currencyConflicts[condition] ? 'conflicting explicit source currencies were omitted' : 'source currency not explicit'});
    const notes = new Set<string>();
    if (condition === 'used') for (const scope of scopes) {
      const nodes = scope.querySelectorAll('span');
      if (nodes.length > 256) fail('CAPTURE_TOO_LARGE');
      for (const node of nodes) {
        charge(); if (node.children.length) continue;
        const match = text(node)?.replace(/\s+/g, ' ').match(/^Used\s*[-–:]\s*(Like New|Very Good|Good|Acceptable)$/i);
        if (match) notes.add(match[1]);
      }
    }
    return { amount, currency: !currencyAmbiguous && currencies.size === 1 ? [...currencies][0] : null, ...(condition === 'unknown' ? {} : {condition}), ...(notes.size === 1 ? {conditionNote:[...notes][0]} : {}) };
  };
  const primary = offer('new'), used = offer('used');
  const price: AmazonPrice | null = primary ? {...primary, ...(used ? {alternative:used} : {})} : used ?? (groups.new.current.size || groups.used.current.size ? null : offer('unknown'));
  return { price, currencyConflict: currencyConflicts[price?.condition ?? 'unknown'], attributes, diagnostics };
}
