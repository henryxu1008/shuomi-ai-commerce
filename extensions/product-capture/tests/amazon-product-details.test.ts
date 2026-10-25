import { readFileSync } from 'node:fs';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { JSDOM } from 'jsdom';
import { captureAmazonDocument } from '../src/amazon/extractor';

const source = 'https://www.amazon.com/dp/B0BDLGZCTY';
const main = 'EHEYCIGA Orthopedic Extra Large Dog Bed, Waterproof Washable XL Pet Bed 44"';
const subtitle = 'Non-Slip Bottom and Egg-Crate Polyurethane Foam Big Dog Couch Bed with Washable Removable Cover, Grey';
const fixture = readFileSync(new URL('./fixtures/amazon-product-details.html', import.meta.url), 'utf8');
function page() { return new JSDOM(fixture, { url: source }).window.document; }
function list(doc: Document, html: string) { doc.querySelector('#detailBulletsWrapper_feature_div')!.innerHTML = `<ul>${html}</ul>`; }
function titlePart(doc: Document) { return doc.querySelector<HTMLElement>('[data-test-subtitle]')!; }
afterEach(() => vi.restoreAllMocks());

describe('Amazon visible product details through existing capture', () => {
  it('captures the visible subtitle and list parameters without choosing a dimension or weight meaning', async () => {
    const { evidence: e } = await captureAmazonDocument(page(), source);
    expect(e.title).toBe(`${main} | ${subtitle}`);
    expect(e.attributes).toEqual(expect.arrayContaining([
      { name: 'Product Dimensions', value: '44"L x 32"W x 6.5"Th' },
      { name: 'Product details · Product Dimensions', value: '44 x 32 x 6 inches; 7.96 pounds' },
      { name: 'Product details · Item model number', value: 'XL' },
      { name: 'Product details · Manufacturer', value: 'EHEYCIGA' },
      { name: 'Product details · Date First Available', value: 'September 8, 2022' },
      { name: 'Product details · ASIN', value: 'B0BDLGZCTY' },
    ]));
    expect(e.attributes.some(a => /rank|review|net weight|gross weight/i.test(a.name))).toBe(false);
    expect(e.observation).toEqual({ scope: 'selected_product', parentASIN: null });
    expect(e.variants).toHaveLength(1);
    expect(e.variants[0].sourceID).toBe('B0BDLGZCTY');
    expect(e.variants[0].attributes.some(a => a.name.startsWith('Product details'))).toBe(false);
    expect(e.variants[0].images).toEqual([{ url: 'https://m.media-amazon.com/images/I/current._AC_SL1500_.jpg', role: 'main' }]);
  });

  it('keeps the original title when the optional section/subtitle is absent', async () => {
    const doc = page(); titlePart(doc).remove();
    expect((await captureAmazonDocument(doc, source)).evidence.title).toBe(main);
    doc.querySelector('#titleSection')!.replaceWith(doc.querySelector('#productTitle')!);
    expect((await captureAmazonDocument(doc, source)).evidence.title).toBe(main);
  });

  it('finds one subtitle when its wrapper also contains the main heading', async () => {
    const doc = page(); const section = doc.querySelector('#titleSection')!;
    const wrapper = doc.createElement('div');
    wrapper.append(...Array.from(section.childNodes)); section.append(wrapper);
    expect((await captureAmazonDocument(doc, source)).evidence.title).toBe(`${main} | ${subtitle}`);
  });

  it('deduplicates repeated subtitle/main branches instead of repeating title text', async () => {
    const doc = page(); const root = doc.querySelector('#titleSection')!;
    root.insertAdjacentHTML('beforeend', `<div>${subtitle}</div><div>${main}</div>`);
    expect((await captureAmazonDocument(doc, source)).evidence.title).toBe(`${main} | ${subtitle}`);
  });

  it.each(['different subtitle', 'another title section', 'unbound primary'])('keeps the primary title and reports %s ambiguity', async kind => {
    const doc = page(); const root = doc.querySelector('#titleSection')!;
    if (kind === 'different subtitle') root.insertAdjacentHTML('beforeend', '<div>Different subtitle</div>');
    else if (kind === 'another title section') doc.body.insertAdjacentHTML('beforeend', '<div id="titleSection">Different product</div>');
    else doc.body.prepend(doc.querySelector('#productTitle')!);
    const e = (await captureAmazonDocument(doc, source)).evidence;
    expect(e.title).toBe(main);
    expect(e.missingFacts.some(f => f.field === 'title.subtitle')).toBe(true);
  });

  it.each([
    '<span hidden>UNRELATED</span>', '<span aria-hidden="true">UNRELATED</span>',
    '<span style="display:none">UNRELATED</span>', '<span style="visibility:hidden">UNRELATED</span>',
    '<form><span>UNRELATED</span></form>', '<a href="/stores/brand">UNRELATED</a>',
    '<button>UNRELATED</button>', '<select><option>UNRELATED</option></select>',
    '<div class="recommendations">UNRELATED</div>', '<div id="bylineInfo">UNRELATED</div>',
    '<div id="averageCustomerReviews">UNRELATED</div>', '<script>UNRELATED</script>',
  ])('excludes title auxiliary content %s', async html => {
    const doc = page(); titlePart(doc).remove(); doc.querySelector('#titleSection')!.insertAdjacentHTML('beforeend', html);
    const e = (await captureAmazonDocument(doc, source)).evidence;
    expect(e.title).toBe(main);
    expect(JSON.stringify(e)).not.toContain('UNRELATED');
  });

  it.each(['detailBulletsWrapper_feature_div', 'detailBullets_feature_div'])('reads bounded top-level parameters in %s', async id => {
    const doc = page(); doc.querySelector('#detailBulletsWrapper_feature_div')!.id = id;
    const e = (await captureAmazonDocument(doc, source)).evidence;
    expect(e.attributes).toContainEqual({ name: 'Product details · Item model number', value: 'XL' });
  });

  it('keeps valid linked parameter values and only splits the first visible colon', async () => {
    const doc = page(); list(doc, '<li>Manufacturer： <a href="/stores/EHEYCIGA">EHEYCIGA: US</a></li>');
    expect((await captureAmazonDocument(doc, source)).evidence.attributes).toContainEqual({ name: 'Product details · Manufacturer', value: 'EHEYCIGA: US' });
  });

  it('does not read unrelated wrappers, nested rows, metrics, other products or controls', async () => {
    const doc = page(); list(doc, `<li>Manufacturer: EHEYCIGA<ul><li>Nested: SECRET</li></ul></li>
      <li>Other: <a href="/dp/B012345678">SECRET</a></li><li>Input: <input value="SECRET"></li>
      <li>Button: <button>SECRET</button></li><li>Choice: <select><option>SECRET</option></select></li>
      <li>Review: <div id="detailBullets_averageCustomerReviews">SECRET</div></li>
      <li>Best Sellers Rank: SECRET</li>`);
    doc.body.insertAdjacentHTML('beforeend', '<aside><ul><li>Unrelated: SECRET</li></ul></aside>');
    const e = (await captureAmazonDocument(doc, source)).evidence;
    expect(e.attributes.filter(a => a.name.startsWith('Product details'))).toEqual([{ name: 'Product details · Manufacturer', value: 'EHEYCIGA' }]);
    expect(JSON.stringify(e)).not.toContain('SECRET');
  });

  it.each([
    '<li hidden>Fake: SECRET</li>', '<li aria-hidden="true">Fake: SECRET</li>',
    '<li style="display:none">Fake: SECRET</li>', '<form><li>Fake: SECRET</li></form>',
    '<div class="recommendations"><li>Fake: SECRET</li></div>', '<a href="/dp/B012345678"><li>Fake: SECRET</li></a>',
  ])('keeps existing exclusions for source list %s', async html => {
    const doc = page(); list(doc, `<li>Manufacturer: EHEYCIGA</li>${html}`);
    const e = (await captureAmazonDocument(doc, source)).evidence;
    expect(e.attributes).toContainEqual({ name: 'Product details · Manufacturer', value: 'EHEYCIGA' });
    expect(JSON.stringify(e)).not.toContain('SECRET');
  });

  it('deduplicates equal source rows and omits conflicting source rows without changing overview', async () => {
    const doc = page(); list(doc, '<li>Manufacturer: EHEYCIGA</li><li>Manufacturer: EHEYCIGA</li><li>Model: A</li><li>Model: B</li><li>Brand: Other source value</li>');
    const e = (await captureAmazonDocument(doc, source)).evidence;
    expect(e.attributes.filter(a => a.name === 'Product details · Manufacturer')).toHaveLength(1);
    expect(e.attributes.some(a => a.name === 'Product details · Model')).toBe(false);
    expect(e.attributes).toContainEqual({ name: 'Brand', value: 'EHEYCIGA' });
    expect(e.attributes).toContainEqual({ name: 'Product details · Brand', value: 'Other source value' });
    expect(e.missingFacts.some(f => f.field === 'attributes' && f.reason.includes('conflicting'))).toBe(true);
  });

  it('ignores empty or unlabeled rows without inventing facts', async () => {
    const doc = page(); list(doc, '<li>Unlabeled text</li><li>: value</li><li>Manufacturer: </li><li>Weight: 0.000 lb</li>');
    expect((await captureAmazonDocument(doc, source)).evidence.attributes.filter(a => a.name.startsWith('Product details'))).toEqual([{ name: 'Product details · Weight', value: '0.000 lb' }]);
  });

  it('counts overlapping wrapper nodes only once', async () => {
    const doc = page(); doc.querySelector('#detailBulletsWrapper_feature_div > ul')!.id = 'detailBullets_feature_div';
    expect((await captureAmazonDocument(doc, source)).evidence.attributes.filter(a => a.name === 'Product details · Manufacturer')).toHaveLength(1);
  });

  it('accepts the existing 256-row collection boundary without inventing duplicate facts', async () => {
    const doc = page(); list(doc, '<li>Manufacturer: EHEYCIGA</li>'.repeat(256));
    expect((await captureAmazonDocument(doc, source)).evidence.attributes.filter(a => a.name.startsWith('Product details'))).toHaveLength(1);
  });

  it.each(['title bytes', 'parameter bytes', 'parameter rows', 'title branches'])('retains the %s resource boundary', async kind => {
    const doc = page();
    if (kind === 'title bytes') titlePart(doc).textContent = '界'.repeat(700);
    else if (kind === 'parameter bytes') list(doc, `<li>Manufacturer: ${'界'.repeat(2800)}</li>`);
    else if (kind === 'parameter rows') list(doc, '<li>Manufacturer: EHEYCIGA</li>'.repeat(257));
    else doc.querySelector('#titleSection')!.append(...Array.from({ length: 257 }, () => doc.createTextNode('')));
    await expect(captureAmazonDocument(doc, source)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });

  it('enforces the complete title byte limit including the primary title and separator', async () => {
    const doc = page(); const maxSubtitle = 2000 - new TextEncoder().encode(main).length - 3;
    titlePart(doc).textContent = 'a'.repeat(maxSubtitle);
    expect(new TextEncoder().encode((await captureAmazonDocument(doc, source)).evidence.title!).length).toBe(2000);
    titlePart(doc).textContent += 'a';
    await expect(captureAmazonDocument(doc, source)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });

  it.each(['subtitle', 'list row'])('charges the original total node budget in %s reads', async kind => {
    const doc = page(); const target = kind === 'subtitle' ? titlePart(doc) : doc.querySelector('li')!;
    const original = doc.createTreeWalker.bind(doc); const empty = doc.createTextNode(''); let remaining = 65537;
    vi.spyOn(doc, 'createTreeWalker').mockImplementation((root, what, filter) => root === target && what === 4
      ? { nextNode: () => --remaining >= 0 ? empty : null } as unknown as TreeWalker
      : original(root, what, filter));
    await expect(captureAmazonDocument(doc, source)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });

  it('charges the original time deadline while reading a subtitle', async () => {
    const doc = page(); const target = titlePart(doc); const original = doc.createTreeWalker.bind(doc); let expired = false;
    vi.spyOn(performance, 'now').mockImplementation(() => expired ? 9001 : 0);
    vi.spyOn(doc, 'createTreeWalker').mockImplementation((root, what, filter) => {
      const walker = original(root, what, filter);
      if (root !== target || what !== 4) return walker;
      return { nextNode: () => { expired = true; return walker.nextNode(); } } as unknown as TreeWalker;
    });
    await expect(captureAmazonDocument(doc, source)).rejects.toThrow('CAPTURE_TIMEOUT');
  });
});
