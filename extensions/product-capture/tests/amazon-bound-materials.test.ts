import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { JSDOM } from 'jsdom';
import { captureAmazonDocument } from '../src/amazon/extractor';

const source = 'https://www.amazon.com/dp/B0BDLGZCTY';
const fixture = readFileSync(new URL('./fixtures/amazon-bound-materials.html', import.meta.url), 'utf8');
const page = () => new JSDOM(fixture, { url: source }).window.document;
const capture = (doc: Document) => captureAmazonDocument(doc, source);
const detailRoot = '#productDetails_expanderSectionTables';
const row = (doc: Document, name: string) => doc.querySelector<HTMLElement>(`tr[data-name="${name}"]`)!;
const weight = (doc: Document) => row(doc, 'Item Weight');
const priceRoot = (doc: Document) => doc.querySelector<HTMLElement>('#corePriceDisplay_desktop_feature_div')!;
const priceLabel = (doc: Document) => doc.querySelector<HTMLElement>('#apex-pricetopay-accessibility-label')!;
const reference = (doc: Document) => doc.querySelector<HTMLElement>('.apex-basisprice-offscreen-label')!;
const params = (e: Awaited<ReturnType<typeof capture>>['evidence']) => e.attributes.filter(a => a.name.startsWith('Product details · '));
type Record = { hiRes: string | null; large: string; thumb: string; variant: string };
function records(doc: Document): Record[] {
  const text = doc.querySelector('[data-test-album]')!.textContent!;
  return JSON.parse(text.match(/parseJSON\('(.+)'\)/)![1]);
}
function album(doc: Document, values: Record[]) {
  doc.querySelector('[data-test-album]')!.textContent = `const album={colorImages:{initial:${JSON.stringify(values)}}};`;
}

describe('bound Amazon collapsed parameters', () => {
  it('reads all 26 non-metric facts without an input ASIN and preserves weight meanings', async () => {
    const doc = page(); expect(doc.querySelector('input#ASIN')).toBeNull();
    const e = (await capture(doc)).evidence;
    expect(params(e)).toHaveLength(26);
    expect(e.attributes).toEqual(expect.arrayContaining([
      { name: 'Product details · Item Weight', value: '7.94 pounds' },
      { name: 'Product details · Maximum Weight Recommendation', value: '45.36 kg' },
      { name: 'Product details · Manufacturer Part Number', value: 'ECUS22MPT8S13XL' },
      { name: 'Product details · Fill Material', value: 'Polyester Fiber, Polyurethane Foam' },
      { name: 'Product details · Product Care Instructions', value: 'Machine Wash' },
    ]));
    expect(e.variants).toHaveLength(1);
    expect(e.variants[0].sku).toBeNull();
    expect(e.variants[0].attributes.some(a => /Weight|Part Number/.test(a.name))).toBe(false);
    expect(params(e).some(a => /rank|review|net weight|gross weight/i.test(a.name))).toBe(false);
    expect(e.observation).toEqual({ scope: 'selected_product', parentASIN: null });
  });
  it('also reads the same explicitly expanded structure', async () => {
    const doc = page();
    for (const content of doc.querySelectorAll<HTMLElement>('.a-expander-content')) content.style.display = 'block';
    for (const header of doc.querySelectorAll('.a-expander-header')) header.setAttribute('aria-expanded', 'true');
    expect(params((await capture(doc)).evidence)).toHaveLength(26);
  });
  it.each(['missing', 'different', 'conflicting', 'hidden', 'duplicate root'])('omits unbound details for %s ASIN evidence', async kind => {
    const doc = page(); const asin = row(doc, 'ASIN');
    if (kind === 'missing') asin.remove();
    if (kind === 'different') asin.querySelector('td')!.textContent = 'B012345678';
    if (kind === 'conflicting') asin.insertAdjacentHTML('afterend', '<tr><th>ASIN</th><td>B012345678</td></tr>');
    if (kind === 'hidden') asin.hidden = true;
    if (kind === 'duplicate root') doc.body.append(doc.querySelector(detailRoot)!.cloneNode(true));
    const e = (await capture(doc)).evidence;
    expect(params(e)).toEqual([]);
    expect(e.attributes).toContainEqual({ name: 'Material', value: 'Polyester' });
    expect(e.missingFacts.some(f => f.field === 'attributes.details')).toBe(true);
  });
  it.each(['ancestor display', 'content hidden', 'content aria', 'content visibility', 'row display', 'row aria', 'form'])('does not exempt %s along with the collapsed node', async kind => {
    const doc = page(); const target = weight(doc); const content = target.closest<HTMLElement>('.a-expander-content')!;
    if (kind === 'ancestor display') doc.querySelector<HTMLElement>('#prodDetails')!.style.display = 'none';
    if (kind === 'content hidden') content.hidden = true;
    if (kind === 'content aria') content.setAttribute('aria-hidden', 'true');
    if (kind === 'content visibility') content.style.visibility = 'hidden';
    if (kind === 'row display') target.style.display = 'none';
    if (kind === 'row aria') target.setAttribute('aria-hidden', 'true');
    if (kind === 'form') { const form = doc.createElement('form'); content.replaceWith(form); form.append(content); }
    expect((await capture(doc)).evidence.attributes.some(a => a.name === 'Product details · Item Weight')).toBe(false);
  });
  it.each(['button', 'hidden', 'script', 'other product', 'recommendation', 'nested table'])('excludes %s text inside parameter values', async kind => {
    const doc = page(); const cell = weight(doc).querySelector('td')!;
    const snippets: { [key: string]: string } = {
      button: '<button>SECRET</button>', hidden: '<span style="display:none">SECRET</span>', script: '<script>SECRET</script>',
      'other product': '<a href="/dp/B012345678">SECRET</a>', recommendation: '<span class="recommendations">SECRET</span>',
      'nested table': '<table><tr><th>Other</th><td>SECRET</td></tr></table>',
    };
    cell.insertAdjacentHTML('beforeend', snippets[kind]);
    const e = (await capture(doc)).evidence;
    expect(JSON.stringify(e)).not.toContain('SECRET');
    if (kind !== 'nested table') expect(e.attributes).toContainEqual({ name: 'Product details · Item Weight', value: '7.94 pounds' });
  });
  it('requires an admissible, unique expander header and a direct content child', async () => {
    const doc = page(); const container = weight(doc).closest('.a-expander-container')!;
    container.querySelector('.a-expander-header')!.setAttribute('aria-hidden', 'true');
    expect((await capture(doc)).evidence.attributes.some(a => a.name === 'Product details · Item Weight')).toBe(false);
    container.querySelector('.a-expander-header')!.removeAttribute('aria-hidden');
    container.querySelector('.a-expander-header')!.insertAdjacentHTML('afterend', '<a class="a-expander-header" role="button" data-action="a-expander-toggle" aria-expanded="false">Other</a>');
    expect((await capture(doc)).evidence.attributes.some(a => a.name === 'Product details · Item Weight')).toBe(false);
  });
  it('does not admit a collapsed body nested inside an unverified wrapper', async () => {
    const doc = page(); const content = weight(doc).closest('.a-expander-content')!;
    const wrapper = doc.createElement('div'); content.replaceWith(wrapper); wrapper.append(content);
    expect((await capture(doc)).evidence.attributes.some(a => a.name === 'Product details · Item Weight')).toBe(false);
  });
  it('keeps generic table visibility rules unchanged', async () => {
    const doc = page(); doc.body.insertAdjacentHTML('beforeend', '<table id="technicalSpecifications_section_1" style="display:none"><tr><th>SECRET</th><td>SECRET</td></tr></table>');
    const e = (await capture(doc)).evidence;
    expect(JSON.stringify(e)).not.toContain('SECRET'); expect(params(e)).toHaveLength(26);
  });
  it('retains same-source conflict omission', async () => {
    const doc = page(); weight(doc).insertAdjacentHTML('afterend', '<tr><th>Item Weight</th><td>9 kg</td></tr>');
    const e = (await capture(doc)).evidence;
    expect(e.attributes.some(a => a.name === 'Product details · Item Weight')).toBe(false);
    expect(e.missingFacts.some(f => /conflicting product attribute/.test(f.reason))).toBe(true);
  });
  it.each(['row count', 'UTF-8 bytes'])('retains %s limits in the new table region', async kind => {
    const doc = page();
    if (kind === 'row count') weight(doc).parentElement!.insertAdjacentHTML('beforeend', '<tr><th>Extra</th><td>x</td></tr>'.repeat(257));
    else weight(doc).querySelector('td')!.textContent = '中'.repeat(3000);
    await expect(capture(doc)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });
});

describe('complete current album through the existing static parser', () => {
  it.each([false, true])('reads all nine bound records, no visible thumbnails=%s', async noThumbs => {
    const doc = page(); const values = records(doc); if (noThumbs) doc.querySelector('#altImages')!.remove();
    const e = (await capture(doc)).evidence;
    expect(e.images.filter(i => i.role !== 'detail').map(i => i.url)).toEqual(values.map(i => i.hiRes));
    expect(e.images.filter(i => i.role === 'detail')).toHaveLength(1);
    expect(e.variants[0].images).toEqual([{ url: values[0].hiRes, role: 'main' }]);
  });
  it('detects conflicts confined to records without visible thumbnails', async () => {
    const doc = page(); const values = records(doc); values[8].hiRes = 'https://m.media-amazon.com/images/I/conflict.jpg';
    doc.body.insertAdjacentHTML('beforeend', `<script>const other={colorImages:{initial:${JSON.stringify(values)}}};</script>`);
    const e = (await capture(doc)).evidence;
    expect(e.images.some(i => i.url === values[8].hiRes)).toBe(false);
    expect(e.missingFacts).toContainEqual({ field: 'images.hiRes', reason: 'conflicting current-product image metadata' });
  });
  it.each(['duplicate MAIN', 'different MAIN', 'no MAIN'])('does not trust a group with %s', async kind => {
    const doc = page(); const values = records(doc);
    if (kind === 'duplicate MAIN') values.push({ ...values[0] });
    if (kind === 'different MAIN') values[0].hiRes = 'https://m.media-amazon.com/images/I/other.jpg';
    if (kind === 'no MAIN') values[0].variant = 'PT00';
    album(doc, values);
    const e = (await capture(doc)).evidence;
    expect(e.images.some(i => i.url === values[8].hiRes)).toBe(false);
    expect(e.missingFacts.some(f => f.field === 'images.hiRes')).toBe(true);
  });
  it('excludes VID records and preserves an explicitly declared large address', async () => {
    const doc = page(); const values = records(doc); values[7].variant = 'VID'; values[8].hiRes = null; album(doc, values);
    const e = (await capture(doc)).evidence;
    expect(e.images.some(i => i.url === values[7].hiRes)).toBe(false);
    expect(e.images).toContainEqual({ url: values[8].large, role: 'gallery' });
    expect(e.images.some(i => i.url.includes('41p-y5+eGmL._AC_SL1500_'))).toBe(false);
  });
  it('also excludes an explicitly declared VID thumbnail without a DOM video class', async () => {
    const doc = page(); const values = records(doc); values[8].variant = 'VID'; album(doc, values);
    doc.querySelector('#altImages ul')!.insertAdjacentHTML('beforeend', `<li><img src="${values[8].thumb}"></li>`);
    const e = (await capture(doc)).evidence;
    expect(e.images.some(i => i.url === values[8].thumb || i.url === values[8].hiRes)).toBe(false);
  });
});

describe('bound current price and separate reference label', () => {
  it('keeps current price, unknown currency and source reference in their original owners', async () => {
    const e = (await capture(page())).evidence;
    expect(e.priceFacts).toEqual([{ amount: '39.98', currency: null }]);
    expect(e.variants[0].price).toEqual({ amount: '39.98', currency: null });
    expect(e.attributes).toContainEqual({ name: 'Source reference price', value: 'List Price: $46.99' });
    expect(e.priceFacts.some(p => p.amount === '46.99' || p.amount === '9999.00')).toBe(false);
  });
  it('does not substitute the list price when the current label is absent', async () => {
    const doc = page(); priceLabel(doc).remove(); const e = (await capture(doc)).evidence;
    expect(e.priceFacts).toEqual([]); expect(e.variants[0].price).toBeNull();
    expect(e.attributes).toContainEqual({ name: 'Source reference price', value: 'List Price: $46.99' });
  });
  it('rejects disagreement between original and accessibility current-price candidates', async () => {
    const doc = page(); priceRoot(doc).querySelector('.priceToPay .a-offscreen')!.textContent = '$40.98';
    expect((await capture(doc)).evidence.priceFacts).toEqual([]);
  });
  it.each(['wrong template', 'extra prose', 'range'])('does not loosely strip %s into a price', async kind => {
    const doc = page();
    if (kind === 'wrong template') priceLabel(doc).setAttribute('data-pricetopay-savings-label', '{priceToPay} other {savings}');
    if (kind === 'extra prose') priceLabel(doc).textContent += ' buy another product';
    if (kind === 'range') priceLabel(doc).textContent = '$39.98 - $59.98 with 15 percent savings';
    expect((await capture(doc)).evidence.priceFacts).toEqual([]);
  });
  it.each(['USD', 'conflict', 'outside'])('preserves the same-root currency rule for %s metadata', async kind => {
    const doc = page(); const parent = kind === 'outside' ? doc.body : priceRoot(doc);
    parent.insertAdjacentHTML('beforeend', '<meta itemprop="priceCurrency" content="USD">');
    if (kind === 'conflict') parent.insertAdjacentHTML('beforeend', '<meta itemprop="priceCurrency" content="EUR">');
    expect((await capture(doc)).evidence.priceFacts).toEqual([{ amount: '39.98', currency: kind === 'USD' ? 'USD' : null }]);
  });
  it('does not read a price root declaring another ASIN', async () => {
    const doc = page(); priceRoot(doc).setAttribute('data-csa-c-asin', 'B012345678');
    priceRoot(doc).querySelector('.priceToPay .a-offscreen')!.textContent = 'US$39.98';
    const e = (await capture(doc)).evidence;
    expect(e.priceFacts).toEqual([]); expect(e.attributes.some(a => a.name === 'Source reference price')).toBe(false);
  });
  it('does not inherit fields from a nested price root declaring another ASIN', async () => {
    const doc = page(); const inner = priceRoot(doc); inner.setAttribute('data-csa-c-asin', 'B012345678');
    const outer = doc.createElement('div'); outer.id = 'corePrice_feature_div'; inner.replaceWith(outer); outer.append(inner);
    const e = (await capture(doc)).evidence;
    expect(e.priceFacts).toEqual([]); expect(e.attributes.some(a => a.name === 'Source reference price')).toBe(false);
  });
  it('does not mix unit-price amounts or currency metadata into the current quote', async () => {
    const doc = page(); priceRoot(doc).insertAdjacentHTML('beforeend', '<div class="pricePerUnit"><span class="a-price"><span class="a-offscreen">US$1.23</span></span><meta itemprop="priceCurrency" content="USD"></div>');
    expect((await capture(doc)).evidence.priceFacts).toEqual([{ amount: '39.98', currency: null }]);
  });
  it('retains narrowly located price text inside an ancestor form without reading controls', async () => {
    const doc = page(); const form = doc.createElement('form'); const root = priceRoot(doc); root.replaceWith(form); form.append(root);
    root.insertAdjacentHTML('beforeend', '<input value="SECRET"><button>SECRET</button>');
    const e = (await capture(doc)).evidence;
    expect(e.priceFacts).toEqual([{ amount: '39.98', currency: null }]);
    expect(e.attributes).toContainEqual({ name: 'Source reference price', value: 'List Price: $46.99' });
    expect(JSON.stringify(e)).not.toContain('SECRET');
  });
  it.each(['hidden current', 'hidden reference', 'outside reference', 'invalid reference'])('preserves %s exclusions', async kind => {
    const doc = page();
    if (kind === 'hidden current') priceLabel(doc).setAttribute('aria-hidden', 'true');
    if (kind === 'hidden reference') reference(doc).setAttribute('aria-hidden', 'true');
    if (kind === 'outside reference') doc.body.append(reference(doc));
    if (kind === 'invalid reference') reference(doc).textContent = 'Other offer $46.99';
    const e = (await capture(doc)).evidence;
    if (kind === 'hidden current') expect(e.priceFacts).toEqual([]);
    else expect(e.attributes.some(a => a.name === 'Source reference price')).toBe(false);
  });
  it('omits conflicting reference labels without changing the current price', async () => {
    const doc = page(); const duplicate = reference(doc).cloneNode(true) as HTMLElement; duplicate.textContent = 'List Price: $49.99'; priceRoot(doc).append(duplicate);
    const e = (await capture(doc)).evidence;
    expect(e.priceFacts).toEqual([{ amount: '39.98', currency: null }]);
    expect(e.attributes.some(a => a.name === 'Source reference price')).toBe(false);
    expect(e.missingFacts.some(f => /conflicting product attribute/.test(f.reason))).toBe(true);
  });
});
