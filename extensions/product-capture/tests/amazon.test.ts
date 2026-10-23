import { describe, expect, it } from 'vitest';
import { JSDOM } from 'jsdom';
import { readFileSync } from 'node:fs';
import { captureAmazonDocument } from '../src/amazon/extractor';
import { amazonPageSource, validateAmazonCapture } from '../src/amazon/validation';

const page = (domain = 'www.amazon.com', price = 'US$19.99') => new JSDOM(`<span id="productTitle">Cotton bag</span><input id="ASIN" value="B012345678"><img id="landingImage" src="https://m.media-amazon.com/images/I/blue.jpg"><div id="variation_color"><span class="a-form-label">Color:</span><span class="selection">Blue</span></div><div id="corePrice_feature_div"><span class="a-price"><span class="a-offscreen">${price}</span></span></div>`, { url: `https://${domain}/dp/B012345678` }).window.document;
const inlinePage = () => new JSDOM(readFileSync(new URL('./fixtures/amazon-inline-product.html', import.meta.url), 'utf8'), { url: 'https://www.amazon.com/dp/B0BDLGZCTY' }).window.document;
describe('Amazon single product module', () => {
  it.each(['amazon.com', 'amazon.ca', 'amazon.co.uk', 'amazon.de', 'amazon.co.jp'])('admits only current five sites %s', domain => expect(amazonPageSource(`https://${domain}/dp/B012345678`).sourceID).toBe('B012345678'));
  it('keeps explicit variant images and partial observation', async () => {
    const payload = await captureAmazonDocument(page(), 'https://www.amazon.com/dp/B012345678', new Date('2026-10-23T00:00:00Z'));
    expect(payload.evidence.variants[0].images[0].url).toContain('blue.jpg');
    expect(payload.evidence.variants[0].price).toEqual({ amount: '19.99', currency: 'USD' });
    expect(payload.evidence.observation).toEqual({ scope: 'selected_product', parentASIN: null });
    expect(payload.evidence.missingFacts.some(v => v.field === 'variants')).toBe(true);
    expect(() => validateAmazonCapture({ ...payload, producer: 'amazon_data' })).toThrow();
  });
  it('does not guess a currency from an ambiguous dollar sign', async () => {
    const payload = await captureAmazonDocument(page('www.amazon.ca', '$19.99'), 'https://www.amazon.ca/dp/B012345678');
    expect(payload.evidence.variants[0].price?.currency).toBeNull();
  });
  it('rejects a changed document before extraction', async () => {
    await expect(captureAmazonDocument(page(), 'https://www.amazon.com/dp/B012345679')).rejects.toThrow('SOURCE_CHANGED');
    expect(() => amazonPageSource('https://www.amazon.fr/dp/B012345678')).toThrow();
  });
  it('rejects product text outside the current saved-detail bounds', async () => {
    const payload = await captureAmazonDocument(page(), 'https://www.amazon.com/dp/B012345678');
    expect(() => validateAmazonCapture({ ...payload, evidence: { ...payload.evidence, title: 'a'.repeat(2001) } })).toThrow();
  });
  it('bounds node and text reads before materializing product text', async () => {
    const doc=page();const description=doc.createElement('div');description.id='productDescription';description.append(doc.createTextNode('Observed description'));doc.body.append(description);
    Object.defineProperty(description,'textContent',{get(){throw new Error('UNBOUNDED_TEXT_READ');}});
    const captured=await captureAmazonDocument(doc,doc.URL);expect(captured.evidence.description).toBe('Observed description');
    for(let i=0;i<257;i++){const variation=doc.createElement('div');variation.id=`variation_${i}`;variation.innerHTML='<span class="a-form-label">Color:</span><span class="selection">Blue</span>';doc.body.append(variation);}
    await expect(captureAmazonDocument(doc,doc.URL)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });
  it('captures only the inline twister current color and size', async () => {
    const doc = inlinePage();
    const payload = await captureAmazonDocument(doc, doc.URL);
    expect(payload.evidence.variants).toHaveLength(1);
    expect(payload.evidence.variants[0].sourceID).toBe('B0BDLGZCTY');
    expect(payload.evidence.variants[0].attributes).toEqual([
      { name: 'Color', value: 'Grey' },
      { name: 'Size', value: '44.0"l X 32.0"w X 6.5"th' },
    ]);
    expect(payload.evidence.observation.scope).toBe('selected_product');
    expect(payload.evidence.missingFacts.some(v => v.field === 'variants[0].attributes')).toBe(false);
  });
  it('captures feature bullets in source order as bounded plain description text', async () => {
    const doc = inlinePage();
    const payload = await captureAmazonDocument(doc, doc.URL);
    expect(payload.evidence.description).toBe('Supportive foam & washable cover.\nNon-slip base.');
    expect(payload.evidence.missingFacts.some(v => v.field === 'description' || v.field === 'description.featureBullets')).toBe(false);
  });
  it('retains old selected attributes and description when adding feature bullets', async () => {
    const doc = page();
    doc.body.insertAdjacentHTML('beforeend', '<div id="productDescription">Original product description.</div><div id="feature-bullets"><ul><li><span class="a-list-item">First benefit.</span></li><li><span class="a-list-item">Second benefit.</span></li></ul></div>');
    const payload = await captureAmazonDocument(doc, doc.URL);
    expect(payload.evidence.variants[0].attributes).toEqual([{ name: 'Color', value: 'Blue' }]);
    expect(payload.evidence.description).toBe('Original product description.\n\nFirst benefit.\nSecond benefit.');
  });
  it('deduplicates agreeing layouts and omits conflicting selected values with a diagnostic', async () => {
    const doc = inlinePage();
    doc.body.insertAdjacentHTML('beforeend', '<div id="variation_color_name"><span class="a-form-label">Color:</span><span class="selection">Grey</span></div>');
    expect((await captureAmazonDocument(doc, doc.URL)).evidence.variants[0].attributes).toHaveLength(2);
    doc.querySelector('#variation_color_name .selection')!.textContent = 'Beige';
    const conflict = await captureAmazonDocument(doc, doc.URL);
    expect(conflict.evidence.variants[0].attributes).toEqual([{ name: 'Size', value: '44.0"l X 32.0"w X 6.5"th' }]);
    expect(conflict.evidence.missingFacts).toContainEqual({ field: 'variants[0].attributes', reason: 'conflicting selected dimension values were omitted' });
  });
  it('does not read explicitly hidden alternative dimensions', async () => {
    const doc = inlinePage();
    doc.body.insertAdjacentHTML('afterbegin', '<div hidden><div id="variation_color"><span class="a-form-label">Color:</span><span class="selection">Black</span></div></div>');
    expect((await captureAmazonDocument(doc, doc.URL)).evidence.variants[0].attributes[0]).toEqual({ name: 'Color', value: 'Grey' });
  });
  it('reports missing selected dimensions, description and feature bullets without inventing values', async () => {
    const doc = inlinePage();
    doc.getElementById('inline-twister-expanded-dimension-text-color_name')!.remove();
    doc.getElementById('inline-twister-expanded-dimension-text-size_name')!.remove();
    doc.getElementById('feature-bullets')!.remove();
    const payload = await captureAmazonDocument(doc, doc.URL);
    expect(payload.evidence.variants[0].attributes).toEqual([]);
    expect(payload.evidence.description).toBeNull();
    expect(payload.evidence.missingFacts.map(v => v.field)).toEqual(expect.arrayContaining(['variants[0].attributes', 'description', 'description.featureBullets']));
  });
  it('keeps the combined description within the existing UTF-8 byte budget', async () => {
    const doc = inlinePage();
    doc.getElementById('feature-bullets')!.innerHTML = '<ul><li><span class="a-list-item">' + '界'.repeat(1400) + '</span></li></ul>';
    doc.body.insertAdjacentHTML('beforeend', '<div id="productDescription">' + '界'.repeat(1400) + '</div>');
    await expect(captureAmazonDocument(doc, doc.URL)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });
  it('bounds feature bullet count before accumulating the description', async () => {
    const doc = inlinePage();
    doc.getElementById('feature-bullets')!.innerHTML = '<ul>' + '<li><span class="a-list-item">x</span></li>'.repeat(257) + '</ul>';
    await expect(captureAmazonDocument(doc, doc.URL)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });
});
