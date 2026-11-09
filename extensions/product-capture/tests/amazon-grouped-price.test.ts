import { afterEach, describe, expect, it } from 'vitest';
import { JSDOM } from 'jsdom';
import { amazonPrice } from '../src/amazon/price';
import { captureAmazonDocument } from '../src/amazon/extractor';

const asin = 'B0BDLGZCTY';
const sites = [
  { site: 'us', host: 'www.amazon.com', currency: 'USD', plain: 'US$1299.00', grouped: 'US$1,299.00', amount: '1299.00' },
  { site: 'ca', host: 'www.amazon.ca', currency: 'CAD', plain: 'CA$1299.00', grouped: 'CA$1,299.00', amount: '1299.00' },
  { site: 'uk', host: 'www.amazon.co.uk', currency: 'GBP', plain: '£1299.00', grouped: '£1,299.00', amount: '1299.00' },
  { site: 'de', host: 'www.amazon.de', currency: 'EUR', plain: '1299,00 €', grouped: '1.299,00 €', amount: '1299.00' },
  { site: 'jp', host: 'www.amazon.co.jp', currency: 'JPY', plain: '1299円', grouped: '1,299円', amount: '1299' },
] as const;
type Site = typeof sites[number]['site'];
const windows: JSDOM[] = [];
afterEach(() => { for (const dom of windows.splice(0)) dom.window.close(); });
function page(site: Site, raw: string, currency: string | null = null) {
  const host = sites.find(s => s.site === site)!.host;
  const dom = new JSDOM(`<span id="productTitle">Example product</span>
    <img id="landingImage" src="https://m.media-amazon.com/images/I/example.jpg">
    <div id="corePriceDisplay_desktop_feature_div" data-csa-c-asin="${asin}">
      <span class="a-price"><span class="a-offscreen">${raw}</span></span>
      ${currency ? `<meta itemprop="priceCurrency" content="${currency}">` : ''}
    </div>`, { url: `https://${host}/dp/${asin}` });
  windows.push(dom);
  return dom.window.document;
}
const quote = (doc: Document, site: Site) => amazonPrice(doc, site, asin, () => {});

describe('Amazon grouped source amounts', () => {
  it.each(sites)('$site preserves the same exact amount for plain and grouped prices', ({ site, plain, grouped, currency, amount }) => {
    for (const raw of [plain, grouped]) {
      expect(quote(page(site, raw, currency), site).price).toEqual({ amount, currency });
    }
  });
  it.each(sites)('$site supplies both extractor price slots without a false missing-price fact', async ({ site, grouped, currency, amount }) => {
    const doc = page(site, grouped, currency);
    const { evidence } = await captureAmazonDocument(doc, doc.URL);
    expect(evidence.priceFacts).toEqual([{ amount, currency }]);
    expect(evidence.variants[0].price).toEqual(evidence.priceFacts[0]);
    expect(evidence.missingFacts.some(f => f.field === 'price')).toBe(false);
  });
  it.each([
    ['us', '9,999,999,999,999,999.12345678', '9999999999999999.12345678'],
    ['de', '9.999.999.999.999.999,12345678', '9999999999999999.12345678'],
    ['de', '1.299', '1299'],
    ['us', '  US$\u00a01,299.00  ', '1299.00'],
    ['de', '  1.299,00\u00a0€  ', '1299.00'],
    ['us', '0.00', '0.00'],
    ['de', '0,00', '0.00'],
  ] as const)('preserves decimal spelling and bounds: %s %s', (site, raw, amount) => {
    expect(quote(page(site, raw), site).price?.amount).toBe(amount);
  });
  it.each([
    ['us', '12,99.00'], ['us', '1,29,900.00'], ['us', '1,,299.00'],
    ['us', '1,299,'], ['us', ',1299'], ['us', '0,129.00'], ['us', '01,299.00'],
    ['us', '1.299,00'], ['us', '1,299.00-1,499.00'], ['us', '1 299.00'],
    ['us', '10,000,000,000,000,000'], ['us', '1,299.123456789'],
    ['de', '12.99,00'], ['de', '1.29.900,00'], ['de', '1..299,00'],
    ['de', '1.299.'], ['de', '.1299'], ['de', '0.129,00'], ['de', '01.299,00'],
    ['de', '1,299.00'], ['de', '1.299,00–1.499,00'], ['de', '1 299,00'],
    ['de', '10.000.000.000.000.000'], ['de', '1.299,123456789'],
    ['jp', '1,29,900円'], ['ca', 'CA$12,99.00'], ['uk', '£12,99.00'],
  ] as const)('rejects malformed, ambiguous or out-of-bounds amounts: %s %s', (site, raw) => {
    expect(quote(page(site, raw, 'USD'), site).price).toBeNull();
  });
  it('keeps currency unknown instead of deriving it from a site or bare dollar symbol', () => {
    expect(quote(page('us', '$1,299.00'), 'us').price).toEqual({ amount: '1299.00', currency: null });
  });
  it('keeps a currency conflict explicit', () => {
    expect(quote(page('us', 'US$1,299.00', 'EUR'), 'us')).toMatchObject({
      price: { amount: '1299.00', currency: null }, currencyConflict: true,
    });
  });
  it('requires the same ASIN binding for grouped prices', () => {
    const doc = page('us', 'US$1,299.00');
    doc.querySelector('[data-csa-c-asin]')!.setAttribute('data-csa-c-asin', 'B012345678');
    expect(quote(doc, 'us').price).toBeNull();
  });
  it('normalizes the accessibility price but keeps the reference price out of current quotes', () => {
    const doc = page('us', 'US$1,299.00');
    doc.querySelector('.a-price')!.remove();
    doc.querySelector('[data-csa-c-asin]')!.insertAdjacentHTML('beforeend', `
      <span id="apex-pricetopay-accessibility-label" data-pricetopay-label="{priceToPay}">US$1,299.00</span>
      <span class="apex-basisprice-offscreen-label" data-basisprice-label="{label} {price}">List Price: US$1,499.00</span>`);
    expect(quote(doc, 'us')).toMatchObject({
      price: { amount: '1299.00', currency: 'USD' },
      attributes: [{ name: 'Source reference price', value: 'List Price: US$1,499.00' }],
    });
  });
});
