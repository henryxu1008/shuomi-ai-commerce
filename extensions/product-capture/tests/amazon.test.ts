import { describe, expect, it } from 'vitest';
import { JSDOM } from 'jsdom';
import { readFileSync } from 'node:fs';
import { captureAmazonDocument } from '../src/amazon/extractor';
import { amazonPageSource, validateAmazonCapture } from '../src/amazon/validation';

const page = (domain = 'www.amazon.com', price = 'US$19.99') => new JSDOM(`<span id="productTitle">Cotton bag</span><input id="ASIN" value="B012345678"><img id="landingImage" src="https://m.media-amazon.com/images/I/blue.jpg"><div id="variation_color"><span class="a-form-label">Color:</span><span class="selection">Blue</span></div><div id="corePrice_feature_div" data-csa-c-asin="B012345678"><span class="a-price"><span class="a-offscreen">${price}</span></span></div>`, { url: `https://${domain}/dp/B012345678` }).window.document;
const inlinePage = () => new JSDOM(readFileSync(new URL('./fixtures/amazon-inline-product.html', import.meta.url), 'utf8'), { url: 'https://www.amazon.com/dp/B0BDLGZCTY' }).window.document;
const formPage = (root = 'corePrice_feature_div', price = '$39.98') => {
  const doc = page('www.amazon.com',price);
  const form = doc.createElement('form');form.id='addToCart';
  const priceRoot = doc.querySelector('#corePrice_feature_div')!;priceRoot.id=root;
  form.append(priceRoot);
  form.insertAdjacentHTML('beforeend','<div id="availability"><span>In Stock</span></div>');
  doc.body.append(form);
  return doc;
};
describe('Amazon single product module', () => {
  it.each([
    {root:'corePrice_feature_div',price:'$39.98',currency:null},
    {root:'corePriceDisplay_desktop_feature_div',price:'US$39.98',currency:'USD'},
  ])('reads only located price and availability text inside addToCart ($root)',async ({root,price,currency})=>{
    const doc=formPage(root,price);
    const {evidence:e}=await captureAmazonDocument(doc,doc.URL);
    expect(e.priceFacts).toEqual([{amount:'39.98',currency}]);
    expect(e.variants[0].price).toEqual(e.priceFacts[0]);
    expect(e.attributes).toContainEqual({name:'Availability',value:'In Stock'});
    expect(e.missingFacts.some(f=>f.field==='price'||f.field==='availability')).toBe(false);
    expect(e.missingFacts.some(f=>f.field==='price.currency')).toBe(currency===null);
  });
  it('filters controls and excluded descendants without reading other addToCart content',async()=>{
    const doc=formPage();
    const excluded='<input value="PRIVATE_INPUT"><textarea>PRIVATE_TEXTAREA</textarea><select><option>PRIVATE_OPTION</option></select><button>PRIVATE_BUTTON</button><script>PRIVATE_SCRIPT()</script><span hidden>PRIVATE_HIDDEN</span><span aria-hidden="true">PRIVATE_ARIA</span><span style="display:none">PRIVATE_DISPLAY</span><span style="visibility:hidden">PRIVATE_VISIBILITY</span><span class="recommendations">PRIVATE_RECOMMENDATION</span><a href="/dp/B012345679">PRIVATE_OTHER_PRODUCT</a>';
    doc.querySelector('.a-offscreen')!.insertAdjacentHTML('beforeend',excluded);
    doc.querySelector('#availability')!.insertAdjacentHTML('beforeend',excluded);
    for(const target of [doc.querySelector('.a-offscreen')!,doc.querySelector('#availability')!]){
      const nested=doc.createElement('form');nested.textContent='PRIVATE_NESTED_FORM';target.append(nested);
    }
    doc.querySelector('form')!.insertAdjacentHTML('beforeend','PRIVATE_FORM_TEXT<input name="account" value="PRIVATE_ACCOUNT"><div id="productDescription">PRIVATE_DESCRIPTION</div><div id="aplus"><div class="aplus-module"><p>PRIVATE_APLUS</p><img src="https://m.media-amazon.com/PRIVATE_IMAGE.jpg"></div></div><table id="productOverview_feature_div"><tr><th>Private</th><td>PRIVATE_PARAMETER</td></tr></table>');
    const {evidence:e}=await captureAmazonDocument(doc,doc.URL);
    expect(e.priceFacts).toEqual([{amount:'39.98',currency:null}]);
    expect(e.attributes).toEqual([{name:'Availability',value:'In Stock'}]);
    expect(e.description).toBeNull();
    expect(JSON.stringify(e)).not.toContain('PRIVATE_');
  });
  it.each([
    {price:'$39.98',metadata:'USD',currency:'USD'},
    {price:'US$39.98',metadata:'EUR',currency:null},
  ])('preserves existing bound currency validation for addToCart $price / $metadata',async ({price,metadata,currency})=>{
    const doc=formPage('corePrice_feature_div',price);
    doc.querySelector('#corePrice_feature_div')!.insertAdjacentHTML('beforeend',`<meta itemprop="priceCurrency" content="${metadata}"><span hidden><meta itemprop="priceCurrency" content="CAD"></span><span class="recommendations"><meta itemprop="priceCurrency" content="GBP"></span>`);
    doc.querySelector('form')!.insertAdjacentHTML('beforeend','<meta itemprop="priceCurrency" content="JPY">');
    const {evidence:e}=await captureAmazonDocument(doc,doc.URL);
    expect(e.priceFacts).toEqual([{amount:'39.98',currency}]);
    if(currency===null)expect(e.missingFacts).toContainEqual({field:'price.currency',reason:'conflicting explicit source currencies were omitted'});
  });
  it.each(['hidden','aria-hidden="true"','style="display:none"','style="visibility:hidden"'])('keeps addToCart %s ancestors excluded',async attribute=>{
    const doc=formPage();
    const wrapper=doc.createElement('div');wrapper.innerHTML=`<div ${attribute}></div>`;
    doc.body.append(wrapper);wrapper.firstElementChild!.append(doc.querySelector('form')!);
    const {evidence:e}=await captureAmazonDocument(doc,doc.URL);
    expect(e.priceFacts).toEqual([]);
    expect(e.attributes.some(a=>a.name==='Availability')).toBe(false);
  });
  it.each(['recommendation','other-product'])('keeps addToCart fields in a %s context excluded',async kind=>{
    const doc=formPage();
    const wrapper=doc.createElement(kind==='recommendation'?'div':'a');
    if(kind==='recommendation')wrapper.id='recommendations';else wrapper.setAttribute('href','/dp/B012345679');
    doc.body.append(wrapper);wrapper.append(doc.querySelector('form')!);
    const {evidence:e}=await captureAmazonDocument(doc,doc.URL);
    expect(e.priceFacts).toEqual([]);
    expect(e.attributes.some(a=>a.name==='Availability')).toBe(false);
  });
  it.each(['price','availability'])('retains the cumulative UTF-8 budget for addToCart %s text',async field=>{
    const doc=formPage();
    const target=doc.querySelector(field==='price'?'.a-offscreen':'#availability')!;
    target.innerHTML=`<span>${'界'.repeat(1400)}</span><span>${'界'.repeat(1400)}</span>`;
    await expect(captureAmazonDocument(doc,doc.URL)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });
  it('captures rendered product facts and gallery without assigning unbound gallery images to a variant', async () => {
    const doc = page();
    doc.body.insertAdjacentHTML('beforeend', `<div id="productOverview_feature_div"><table><tr><td><span>Brand</span></td><td><span>EHEYCIGA</span></td></tr><tr hidden><td>Private</td><td>hidden</td></tr></table></div>
      <table id="productDetails_detailBullets_sections1"><tr><th>Material</th><td>Polyester</td></tr></table>
      <div id="availability"><span>In Stock</span></div><div id="altImages"><ul>
      <li><img src="https://m.media-amazon.com/images/I/blue.jpg"></li><li><img src="https://m.media-amazon.com/images/I/side.jpg"></li>
      <li class="videoThumbnail"><img src="https://m.media-amazon.com/images/I/video.jpg"></li></ul></div>
      <div id="recommendations"><img src="https://m.media-amazon.com/images/I/other.jpg"></div>`);
    const { evidence } = await captureAmazonDocument(doc, doc.URL);
    expect(evidence.attributes).toEqual([{ name: 'Brand', value: 'EHEYCIGA' }, { name: 'Material', value: 'Polyester' }, { name: 'Availability', value: 'In Stock' }]);
    expect(evidence.images).toEqual([{ url: 'https://m.media-amazon.com/images/I/blue.jpg', role: 'main' }, { url: 'https://m.media-amazon.com/images/I/side.jpg', role: 'gallery' }]);
    expect(evidence.variants[0].images).toEqual([evidence.images[0]]);
    expect(evidence.missingFacts.some(f => f.field === 'inventory')).toBe(true);
  });
  it('ignores struck reference prices and keeps conflicting explicit currencies unknown', async () => {
    const doc = page();
    doc.querySelector('#corePrice_feature_div')!.insertAdjacentHTML('afterbegin', '<span class="a-price a-text-price"><span class="a-offscreen">US$99.00</span></span>');
    doc.querySelector('#corePrice_feature_div')!.insertAdjacentHTML('beforeend', '<meta itemprop="priceCurrency" content="EUR">');
    const { evidence } = await captureAmazonDocument(doc, doc.URL);
    expect(evidence.priceFacts).toEqual([{ amount: '19.99', currency: null }]);
    expect(evidence.missingFacts).toContainEqual({field:'price.currency',reason:'conflicting explicit source currencies were omitted'});
  });
  it('ignores currencies outside the current price and detects all bound metadata conflicts',async()=>{
    const doc=page('www.amazon.com','$19.99');
    doc.head.insertAdjacentHTML('beforeend','<meta itemprop="priceCurrency" content="CAD">');
    doc.body.insertAdjacentHTML('beforeend','<div id="recommendations"><meta itemprop="priceCurrency" content="EUR"></div>');
    expect((await captureAmazonDocument(doc,doc.URL)).evidence.priceFacts[0].currency).toBeNull();
    const root=doc.querySelector('#corePrice_feature_div')!;
    root.insertAdjacentHTML('beforeend','<meta itemprop="priceCurrency" content="USD">');
    expect((await captureAmazonDocument(doc,doc.URL)).evidence.priceFacts[0].currency).toBe('USD');
    root.insertAdjacentHTML('beforeend','<meta itemprop="priceCurrency" content="EUR">');
    const {evidence}=await captureAmazonDocument(doc,doc.URL);
    expect(evidence.priceFacts[0].currency).toBeNull();
    expect(evidence.missingFacts).toContainEqual({field:'price.currency',reason:'conflicting explicit source currencies were omitted'});
  });
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
