import { describe, expect, it } from 'vitest';
import { JSDOM } from 'jsdom';
import { captureAmazonDocument } from '../src/amazon/extractor';
import { amazonPrice } from '../src/amazon/price';

const asin = 'B0BDLGZCTY';
const names = {
  currency: 'items[0.base][customerVisiblePrice][currencyCode]',
  amount: 'items[0.base][customerVisiblePrice][amount]',
  asin: 'items[0.base][asin]',
  extraASIN: 'ASIN',
};
// Reconstructed from the read-only structure and user-confirmed product values,
// not an export of the complete source HTML or any unrelated form fields.
function page(amount = '39.98') {
  const doc = new JSDOM(`<span id="productTitle">Dog bed</span>
    <img id="landingImage" src="https://m.media-amazon.com/images/I/bed.jpg">
    <div id="corePriceDisplay_desktop_feature_div" data-csa-c-asin="${asin}">
      <span class="a-price"><span class="a-offscreen">$${amount}</span></span>
    </div><form id="addToCart"></form>`, { url: `https://www.amazon.com/dp/${asin}` }).window.document;
  const form = doc.querySelector('form')!;
  for (const [key, value] of Object.entries({ currency: 'USD', amount, asin, extraASIN: asin })) {
    const input = doc.createElement('input'); input.type = 'hidden';
    input.name = names[key as keyof typeof names]; input.value = value;
    form.append(input);
  }
  return doc;
}
const input = (doc: Document, key: keyof typeof names) => Array.from(doc.querySelectorAll('form input')).find(n => n.getAttribute('name') === names[key]) as HTMLInputElement;
const quote = (doc: Document) => amazonPrice(doc, 'us', asin, () => {});

describe('Amazon explicitly bound product currency', () => {
  it('uses the confirmed current-product currency for both existing price slots', async () => {
    const { evidence } = await captureAmazonDocument(page(), `https://www.amazon.com/dp/${asin}`);
    expect(evidence.priceFacts).toEqual([{ amount: '39.98', currency: 'USD' }]);
    expect(evidence.variants[0].price).toEqual(evidence.priceFacts[0]);
    expect(evidence.missingFacts.some(f => f.field === 'price.currency')).toBe(false);
  });
  it('reads only the live value of the located product inputs', () => {
    const doc = page();
    for (const key of Object.keys(names) as (keyof typeof names)[]) {
      const node = input(doc, key); const value = node.value;
      node.removeAttribute('value'); Object.defineProperty(node, 'value', { get: () => value });
    }
    const unrelated = doc.createElement('input'); unrelated.name = 'session-token';
    Object.defineProperty(unrelated, 'value', { get: () => { throw Error('unrelated input read'); } });
    doc.querySelector('form')!.append(unrelated);
    expect(quote(doc).price).toEqual({ amount: '39.98', currency: 'USD' });
  });
  it('does not require the optional extra ASIN when the required tuple is complete', () => {
    const doc = page(); input(doc, 'extraASIN').remove();
    expect(quote(doc).price?.currency).toBe('USD');
  });
  it('compares exact decimals and preserves the visible price spelling', () => {
    const doc = page(); input(doc, 'amount').value = '39.98000000';
    expect(quote(doc).price).toEqual({ amount: '39.98', currency: 'USD' });
    const precise = page('9999999999999999.12345678');
    expect(quote(precise).price?.currency).toBe('USD');
    input(precise, 'amount').value = '9999999999999999.12345679';
    expect(quote(precise).price).toEqual({ amount: '9999999999999999.12345678', currency: null });
  });
  it.each(['currency', 'amount', 'asin'] as const)('retains unknown currency when required %s is absent', key => {
    const doc = page(); input(doc, key).remove();
    expect(quote(doc).price).toEqual({ amount: '39.98', currency: null });
  });
  it.each([
    ['currency', ''], ['currency', '$'], ['currency', 'usd'],
    ['amount', ''], ['amount', '46.99'], ['amount', 'NaN'], ['amount', '039.98'],
    ['asin', ''], ['asin', 'B012345678'], ['extraASIN', ''], ['extraASIN', 'B012345678'],
  ] as const)('does not use invalid/stale %s=%s', (key, value) => {
    const doc = page(); input(doc, key).value = value;
    expect(quote(doc).price).toEqual({ amount: '39.98', currency: null });
  });
  it.each(Object.keys(names) as (keyof typeof names)[])('rejects duplicate %s fields even when equal', key => {
    const doc = page(); input(doc, key).after(input(doc, key).cloneNode());
    expect(quote(doc).price?.currency).toBeNull();
  });
  it('does not choose between duplicate addToCart forms', () => {
    const doc = page(); doc.body.append(doc.querySelector('form')!.cloneNode(true));
    expect(quote(doc).price?.currency).toBeNull();
  });
  it.each(['missing-asin', 'other-asin', 'no-current-price', 'reference-only', 'unit-only'])('requires a current-price binding: %s', kind => {
    const doc = page(); const root = doc.querySelector('#corePriceDisplay_desktop_feature_div')!;
    if (kind === 'missing-asin') root.removeAttribute('data-csa-c-asin');
    if (kind === 'other-asin') root.setAttribute('data-csa-c-asin', 'B012345678');
    if (kind === 'no-current-price') root.replaceChildren();
    if (kind === 'reference-only') root.querySelector('.a-price')!.classList.add('a-text-price');
    if (kind === 'unit-only') root.querySelector('.a-price')!.classList.add('pricePerUnit');
    expect(quote(doc).price?.currency ?? null).toBeNull();
  });
  it.each(['form-hidden', 'input-hidden', 'recommendation', 'other-product', 'script', 'template', 'wrong-type'])('excludes unsafe tuple context: %s', kind => {
    const doc = page(); const form = doc.querySelector('form')!;
    if (kind === 'form-hidden') form.hidden = true;
    else if (kind === 'input-hidden') input(doc, 'currency').setAttribute('aria-hidden', 'true');
    else if (kind === 'wrong-type') input(doc, 'currency').type = 'text';
    else {
      const wrapper = doc.createElement(kind === 'other-product' ? 'a' : kind === 'script' ? 'script' : kind === 'template' ? 'template' : 'div');
      if (kind === 'recommendation') wrapper.className = 'recommendations';
      if (kind === 'other-product') wrapper.setAttribute('href', '/dp/B012345678');
      doc.body.append(wrapper); wrapper.append(form);
    }
    expect(quote(doc).price?.currency).toBeNull();
  });
  it.each(['CAD', 'EUR'])('retains explicit currency conflicts with %s metadata', other => {
    const doc = page(); doc.querySelector('#corePriceDisplay_desktop_feature_div')!.insertAdjacentHTML('beforeend', `<meta itemprop="priceCurrency" content="${other}">`);
    const result = quote(doc);
    expect(result.price).toEqual({ amount: '39.98', currency: null });
    expect(result.currencyConflict).toBe(true);
  });
  it('retains conflicts with an unambiguous price symbol', () => {
    const doc = page(); doc.querySelector('.a-offscreen')!.textContent = 'CA$39.98';
    expect(quote(doc)).toMatchObject({ price: { amount: '39.98', currency: null }, currencyConflict: true });
  });
  it('preserves independently observed currency when the tuple is unusable', () => {
    const doc = page(); input(doc, 'amount').value = '46.99';
    doc.querySelector('.a-offscreen')!.textContent = 'US$39.98';
    expect(quote(doc).price).toEqual({ amount: '39.98', currency: 'USD' });
  });
  it('does not read malformed unbounded values without the original resource failure', () => {
    const doc = page(); input(doc, 'currency').value = '界'.repeat(3000);
    expect(() => quote(doc)).toThrow('CAPTURE_TOO_LARGE');
  });
});
