import { describe, expect, it } from 'vitest';
import { JSDOM } from 'jsdom';
import { captureAmazonDocument } from '../src/amazon/extractor';
import { amazonPageSource, validateAmazonCapture } from '../src/amazon/validation';

const page = (domain = 'www.amazon.com', price = 'US$19.99') => new JSDOM(`<span id="productTitle">Cotton bag</span><input id="ASIN" value="B012345678"><img id="landingImage" src="https://m.media-amazon.com/images/I/blue.jpg"><div id="variation_color"><span class="a-form-label">Color:</span><span class="selection">Blue</span></div><div id="corePrice_feature_div"><span class="a-price"><span class="a-offscreen">${price}</span></span></div>`, { url: `https://${domain}/dp/B012345678` }).window.document;
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
});
