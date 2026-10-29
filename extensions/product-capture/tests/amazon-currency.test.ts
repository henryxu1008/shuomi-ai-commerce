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

it('separates explicitly labelled new and used offers for the current ASIN', () => {
  const doc = new JSDOM(`<div id="newAccordionRow_0" class="a-accordion-active"><div id="corePrice_feature_div" data-csa-c-asin="${asin}"><span class="a-price"><span class="a-offscreen">$35.98</span></span></div></div>
    <div id="usedAccordionRow"><div id="corePrice_feature_div" data-csa-c-asin="${asin}"><span class="a-price"><span class="a-offscreen">$31.99</span></span></div></div>`, { url: `https://www.amazon.com/dp/${asin}` }).window.document;
  expect(quote(doc).price).toEqual({amount:'35.98',currency:null,condition:'new',alternative:{amount:'31.99',currency:null,condition:'used'}});
});
it('preserves an explicitly labelled used condition from the same offer', () => {
  const doc = new JSDOM(`<div id="usedAccordionRow"><span>Used - Very Good</span><div id="corePrice_feature_div" data-csa-c-asin="${asin}"><span class="a-price"><span class="a-offscreen">$31.99</span></span></div></div>`, {url:`https://www.amazon.com/dp/${asin}`}).window.document;
  expect(quote(doc).price).toEqual({amount:'31.99',currency:null,condition:'used',conditionNote:'Very Good'});
});
it.each(['new','used'] as const)('rejects %s offer when its own form contradicts the product or amount', condition=>{
 const row=condition==='new'?'newAccordionRow_0':'usedAccordionRow';
 const doc=page('35.98');const root=doc.querySelector('#corePriceDisplay_desktop_feature_div')!;const form=doc.querySelector('form')!;
 const wrapper=doc.createElement('div');wrapper.id=row;doc.body.append(wrapper);wrapper.append(root,form);
 input(doc,'asin').value='B012345678';expect(quote(doc).price).toBeNull();
 input(doc,'asin').value=asin;input(doc,'amount').value='30.00';expect(quote(doc).price).toBeNull();
 form.remove();expect(quote(doc).price).toMatchObject({amount:'35.98',currency:null,condition});
});
it('rejects unbound amounts but keeps a bound price whose condition is unknown',()=>{
 const doc=page();doc.querySelector('#corePriceDisplay_desktop_feature_div')!.removeAttribute('data-csa-c-asin');
 expect(quote(doc).price).toBeNull();
 doc.querySelector('#corePriceDisplay_desktop_feature_div')!.setAttribute('data-csa-c-asin',asin);
 expect(quote(doc).price).toEqual({amount:'39.98',currency:'USD'});
});
it('does not mix metadata from an unbound sibling price root',()=>{
 const doc=page();input(doc,'currency').remove();
 doc.body.insertAdjacentHTML('beforeend','<div id="corePrice_feature_div"><span class="a-price"><span class="a-offscreen">$39.98</span></span><meta itemprop="priceCurrency" content="EUR"></div>');
 expect(quote(doc).price).toEqual({amount:'39.98',currency:null});
});
it('does not relabel a rejected explicit offer as an unknown-status price',()=>{
 const doc=page();const root=doc.querySelector('#corePriceDisplay_desktop_feature_div')!;const form=doc.querySelector('form')!;
 const row=doc.createElement('div');row.id='newAccordionRow_0';doc.body.append(row);row.append(root,form);input(doc,'asin').value='B012345678';
 doc.body.insertAdjacentHTML('beforeend',`<div id="corePrice_feature_div" data-csa-c-asin="${asin}"><span class="a-price"><span class="a-offscreen">$39.98</span></span></div>`);
 doc.body.insertAdjacentHTML('beforeend','<form id="addToCart"></form>');
 expect(quote(doc).price).toBeNull();
});
it.each(['asin','amount'] as const)('rejects contradictory duplicate %s fields in an otherwise bound offer',key=>{
 const doc=page();const inputNode=input(doc,key);const duplicate=inputNode.cloneNode() as HTMLInputElement;duplicate.value=key==='asin'?'B012345678':'44.00';inputNode.after(duplicate);
 expect(quote(doc).price).toBeNull();
});
it('keeps currency unknown when duplicate currency fields conflict with an explicit symbol',()=>{
 const doc=page();doc.querySelector('.a-offscreen')!.textContent='US$39.98';const node=input(doc,'currency');const duplicate=node.cloneNode() as HTMLInputElement;duplicate.value='EUR';node.after(duplicate);
 expect(quote(doc).price).toEqual({amount:'39.98',currency:null});
});
it('reports the conflicting offer without dropping the other condition',()=>{
 const doc=new JSDOM(`<div id="newAccordionRow_0"><div id="corePrice_feature_div" data-csa-c-asin="${asin}"><span class="a-price"><span class="a-offscreen">$35.98</span></span><span class="a-price"><span class="a-offscreen">$36.98</span></span></div></div><div id="usedAccordionRow"><div id="corePrice_feature_div" data-csa-c-asin="${asin}"><span class="a-price"><span class="a-offscreen">$31.99</span></span></div></div>`,{url:`https://www.amazon.com/dp/${asin}`}).window.document;
 const result=quote(doc);expect(result.price).toMatchObject({amount:'31.99',condition:'used'});
 expect(result.diagnostics).toContainEqual({field:'price.new',reason:'conflicting or unbound source offer'});
});

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
    expect(quote(precise).price).toBeNull();
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
    if (key === 'amount' && value === '46.99' || (key === 'asin' || key === 'extraASIN') && value === 'B012345678') expect(quote(doc).price).toBeNull();
    else expect(quote(doc).price).toEqual({ amount: '39.98', currency: null });
  });
  it.each(Object.keys(names) as (keyof typeof names)[])('rejects duplicate %s fields even when equal', key => {
    const doc = page(); input(doc, key).after(input(doc, key).cloneNode());
    if(key==='currency') expect(quote(doc).price?.currency).toBeNull();
    else expect(quote(doc).price).toBeNull();
  });
  it('does not choose between duplicate addToCart forms', () => {
    const doc = page(); doc.body.append(doc.querySelector('form')!.cloneNode(true));
    expect(quote(doc).price).toBeNull();
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
  it('rejects a conflicting tuple even with independently observed currency', () => {
    const doc = page(); input(doc, 'amount').value = '46.99';
    doc.querySelector('.a-offscreen')!.textContent = 'US$39.98';
    expect(quote(doc).price).toBeNull();
  });
  it('does not read malformed unbounded values without the original resource failure', () => {
    const doc = page(); input(doc, 'currency').value = '界'.repeat(3000);
    expect(() => quote(doc)).toThrow('CAPTURE_TOO_LARGE');
  });
});
