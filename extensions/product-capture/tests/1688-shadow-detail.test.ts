import { afterEach, describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { JSDOM } from 'jsdom';
import { captureDocument } from '../src/extractor';
import { publicImage, validateCapture } from '../src/validation';
import { captureAmazonDocument } from '../src/amazon/extractor';
import { validatePlatformCapture } from '../src/platforms';

const source = 'https://detail.1688.com/offer/981645030344.html';
const fixture = readFileSync('tests/fixtures/product.html', 'utf8');
// User-provided offline detail fragment: no scripts, browser or network calls.
const detailHTML = readFileSync('tests/fixtures/1688-shadow-detail.html', 'utf8');
const gallery = 'https://cbu01.alicdn.com/img/ibank/fixture.jpg';
const detailURL = 'https://cbu01.alicdn.com/img/ibank/detail.jpg?__r__=1774849509105';
const hostHTML = '<div id="description" data-module="od_product_description"><div class="module-od-product-description"><v-detail-p class="html-description"></v-detail-p></div></div>';
const now = new Date('2026-10-25T04:00:00Z');

function document(keepOriginal = false) {
  const doc = new JSDOM(fixture, { url: source }).window.document;
  if (!keepOriginal) doc.querySelector('#desc-lazyload-container')!.remove();
  return doc;
}
function addShadow(doc: Document, html = detailHTML, mode: ShadowRootMode = 'open') {
  const wrapper = doc.createElement('section');
  wrapper.innerHTML = hostHTML;
  doc.body.append(wrapper);
  const host = wrapper.querySelector('.html-description')!;
  const shadow = host.attachShadow({ mode });
  shadow.innerHTML = html;
  return { wrapper, host, shadow };
}
function payload(url = detailURL, role = 'detail') {
  const p = JSON.parse(readFileSync('tests/fixtures/capture-v2.json', 'utf8'));
  p.evidence.images = [{ url, role }];
  return p;
}
afterEach(() => vi.restoreAllMocks());

describe('1688 explicitly scoped open-shadow detail', () => {
  it('captures all 34 declared src URLs including 30 original queries without making text or SKU images', async () => {
    const doc = document();
    const baseline = (await captureDocument(doc, source, now)).evidence;
    const { shadow } = addShadow(doc);
    const expected = [...shadow.querySelectorAll('img')].map(i => i.getAttribute('src')).filter(Boolean);
    expect(expected).toHaveLength(34);
    expect(doc.querySelectorAll('#description img')).toHaveLength(0);
    const { evidence: e } = await captureDocument(doc, source, now);
    expect(e.images).toEqual([{ url: gallery, role: 'source' }, ...expected.map(url => ({ url, role: 'detail' }))]);
    expect(e.images.filter(i => i.url.includes('?__r__='))).toHaveLength(30);
    expect(e.description).toBeNull();
    expect(e.missingFacts).toContainEqual({ field: 'description', reason: 'source detail contains images but no text nodes' });
    expect(e.variants).toEqual(baseline.variants);
    expect(e.priceFacts).toEqual(baseline.priceFacts);
    expect(e.attributes).toEqual(baseline.attributes);
  });
  it('shares the selected root for text, parameters and images and keeps gallery URL ownership', async () => {
    const doc = document(true);
    addShadow(doc, `<div id="detail"><p>Current detail</p><table><tr><th>净重</th><td>350 g</td></tr></table><img src="${gallery}"><img src="${detailURL}"></div>`);
    const e = (await captureDocument(doc, source, now)).evidence;
    expect(e.description).toContain('Current detail');
    expect(e.description).not.toContain('可重复使用');
    expect(e.attributes).toContainEqual({ name: '净重', value: '350 g' });
    expect(e.images).toEqual([{ url: gallery, role: 'source' }, { url: detailURL, role: 'detail' }]);
  });
  it.each([
    '<div hidden>HOST</div>', '<div aria-hidden="true">HOST</div>', '<div style="display:none">HOST</div>',
    '<form>HOST</form>', '<div id="recommendations">HOST</div>',
    '<a href="https://detail.1688.com/offer/123456789.html">HOST</a>',
    '<div class="video-block">HOST</div>', '<div class="videoThumbnail">HOST</div>',
    '<div data-video-url="x">HOST</div>', '<div data-video-id="x">HOST</div>',
  ])('excludes the host through its light-DOM ancestors: %s', async html => {
    const doc = document();
    const { wrapper } = addShadow(doc);
    const ancestor = doc.createElement('div');
    ancestor.innerHTML = html.replace('HOST', '<span id="mount"></span>');
    doc.body.append(ancestor);
    ancestor.querySelector('#mount')!.replaceWith(wrapper);
    const e = (await captureDocument(doc, source, now)).evidence;
    expect(e.images).toEqual([{ url: gallery, role: 'source' }]);
    expect(e.description).toBeNull();
  });
  it('preserves exclusions inside the shadow root for image, text and parameter descendants', async () => {
    const doc = document();
    const unwanted = `<p>Discarded</p><img src="${detailURL}"><table><tr><th>Discarded field</th><td>Discarded value</td></tr></table>`;
    const blocked = ['<span hidden>CONTENT</span>', '<form>CONTENT</form>', '<div class="recommendations">CONTENT</div>',
      '<a href="https://detail.1688.com/offer/123456789.html">CONTENT</a>', '<div class="video-block">CONTENT</div>']
      .map(s => s.replace('CONTENT', unwanted)).join('');
    addShadow(doc, `<div id="detail"><p>Allowed</p><img src="https://cbu01.alicdn.com/allowed.jpg">${blocked}</div>`);
    const e = (await captureDocument(doc, source, now)).evidence;
    expect(e.description).toBe('Allowed');
    expect(e.images).toEqual([{ url: gallery, role: 'source' }, { url: 'https://cbu01.alicdn.com/allowed.jpg', role: 'detail' }]);
    expect(e.attributes.some(a => a.name === 'Discarded field')).toBe(false);
  });
  it.each(['absent', 'closed', 'empty'])('reports a %s modern root without switching to a different detail owner', async kind => {
    const doc = document(true);
    if (kind === 'absent') doc.body.insertAdjacentHTML('beforeend', hostHTML);
    else addShadow(doc, kind === 'empty' ? '<div id="detail"></div>' : detailHTML, kind === 'closed' ? 'closed' : 'open');
    const e = (await captureDocument(doc, source, now)).evidence;
    expect(e.description).toBeNull();
    expect(e.images).toEqual([{ url: gallery, role: 'source' }]);
    expect(e.missingFacts).toContainEqual({ field: 'images.detail', reason: 'current detail root is not ready' });
  });
  it.each(['hosts', 'roots', 'original roots'])('does not arbitrarily select among multiple %s', async kind => {
    const doc = document(true);
    if (kind === 'hosts') { addShadow(doc); addShadow(doc); }
    else if (kind === 'roots') addShadow(doc, '<div id="detail">First</div><div id="detail">Second</div>');
    else doc.body.insertAdjacentHTML('beforeend', '<div id="desc-lazyload-container">Second</div>');
    const e = (await captureDocument(doc, source, now)).evidence;
    expect(e.description).toBeNull();
    expect(e.images).toEqual([{ url: gallery, role: 'source' }]);
    expect(e.missingFacts).toContainEqual({ field: 'images.detail', reason: 'current detail root is ambiguous' });
  });
  it('retains an existing admitted page shape when unrelated modern hosts are filtered out', async () => {
    const doc = document(true);
    const { wrapper } = addShadow(doc);
    wrapper.hidden = true;
    expect((await captureDocument(doc, source, now)).evidence.description).toBe('可重复使用的棉质收纳袋。');
  });
  it.each(['hosts', 'images', 'text'])('keeps existing resource bounds for new shadow %s', async kind => {
    const doc = document();
    if (kind === 'hosts') doc.body.insertAdjacentHTML('beforeend', hostHTML.repeat(257));
    else addShadow(doc, `<div id="detail">${kind === 'images'
      ? '<img src="https://cbu01.alicdn.com/repeated.jpg">'.repeat(257) : `<p>${'界'.repeat(2800)}</p>`}</div>`);
    await expect(captureDocument(doc, source, now)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });
  it('charges shadow text traversal to the existing deadline', async () => {
    const doc = document();
    addShadow(doc, '<div id="detail"><p>Text</p></div>');
    let enteringDetail = false;
    const original = doc.createTreeWalker.bind(doc);
    vi.spyOn(doc, 'createTreeWalker').mockImplementation((...args) => {
      if ((args[0] as Element).id === 'detail') enteringDetail = true;
      return original(...args);
    });
    vi.spyOn(performance, 'now').mockImplementation(() => enteringDetail ? 9001 : 0);
    await expect(captureDocument(doc, source, now)).rejects.toThrow('CAPTURE_TIMEOUT');
  });
  it('counts shadow text nodes against the existing total node budget even when text is empty', async () => {
    const doc = document();
    addShadow(doc, '<div id="detail"><p>Text</p></div>');
    const empty = doc.createTextNode('');
    let visited = 0;
    const original = doc.createTreeWalker.bind(doc);
    vi.spyOn(doc, 'createTreeWalker').mockImplementation((...args) => {
      const walker = original(...args);
      // Exercise the existing cursor seam without constructing a huge DOM.
      if ((args[0] as Element).id === 'detail') vi.spyOn(walker, 'nextNode').mockImplementation(() => ++visited <= 65536 ? empty : null);
      return walker;
    });
    await expect(captureDocument(doc, source, now)).rejects.toThrow('CAPTURE_TOO_LARGE');
    expect(visited).toBeGreaterThan(64000);
  });
});

describe('1688 v2 top-level detail query admission only', () => {
  it('preserves the exact known query at the new payload position while common validation remains strict', () => {
    const p = payload();
    expect(validateCapture(p)).toEqual(p);
    expect(publicImage(detailURL)).toBe(false);
  });
  it.each(['source', 'gallery', 'variant', 'other'])('does not admit the query in top-level %s images', role => {
    expect(() => validateCapture(payload(detailURL, role))).toThrow('INVALID_CAPTURE');
  });
  it('does not admit the query in SKU images even if the caller labels it detail, or in v1', () => {
    const p = payload(gallery, 'source');
    p.evidence.variants[0].images = [{ url: detailURL, role: 'detail' }];
    expect(() => validateCapture(p)).toThrow('INVALID_CAPTURE');
    const v1 = JSON.parse(readFileSync('tests/fixtures/capture-v1.json', 'utf8'));
    v1.evidence.images = [{ url: detailURL, role: 'detail' }];
    expect(() => validateCapture(v1)).toThrow('INVALID_CAPTURE');
  });
  it('does not expand Amazon admission through the shared validator', async () => {
    const url = 'https://www.amazon.com/dp/B0BDLGZCTY';
    const doc = new JSDOM('<span id="productTitle">Selected</span><input id="ASIN" value="B0BDLGZCTY"><img id="landingImage" src="https://m.media-amazon.com/images/I/main.jpg">', { url }).window.document;
    const p = await captureAmazonDocument(doc, url, now);
    p.evidence.images = [{ url: detailURL, role: 'detail' }];
    expect(() => validatePlatformCapture(p)).toThrow('INVALID_CAPTURE');
  });
  it.each([
    '?__r__=1774849509105&token=x', '?__r__=1774849509105&__r__=1774849509105',
    '?%5F%5Fr%5F%5F=1774849509105', '?__r__=%31774849509105', '?__r__=177484950910',
    '?__r__=17748495091055', '?__r__=177484950910x', '?__r__=1774849509105#x',
    '?other=1774849509105', '?__r__=1774849509105&',
    '?__r__=1774849509105#', '?__r__=1774849509105\n', '?__r__=1774849509105\r', '?__r__=1774849509105\u2028',
  ])('rejects non-exact or additional query material: %s', query => {
    expect(() => validateCapture(payload(`https://cbu01.alicdn.com/img/ibank/detail.jpg${query}`))).toThrow('INVALID_CAPTURE');
  });
  it.each([
    'http://cbu01.alicdn.com/img/ibank/detail.jpg', 'https://cbu02.alicdn.com/img/ibank/detail.jpg',
    'https://cbu01.alicdn.com.evil.test/img/ibank/detail.jpg', 'https://cbu01.alicdn.com/other/detail.jpg',
    'https://user:pw@cbu01.alicdn.com/img/ibank/detail.jpg', 'https://cbu01.alicdn.com:8443/img/ibank/detail.jpg',
    'https://cbu01.alicdn.com/img/ibank/token.jpg', 'https://127.0.0.1/img/ibank/detail.jpg',
  ])('keeps original origin/path/credential defenses: %s', base => {
    expect(() => validateCapture(payload(`${base}?__r__=1774849509105`))).toThrow('INVALID_CAPTURE');
  });
  it('enforces 2048 UTF-8 URL bytes on direct detail payloads, including multibyte paths', () => {
    const suffix = '.jpg?__r__=1774849509105';
    const prefix = 'https://cbu01.alicdn.com/img/ibank/';
    const exact = prefix + 'a'.repeat(2048 - prefix.length - suffix.length) + suffix;
    expect(validateCapture(payload(exact)).evidence.images[0].url).toBe(exact);
    expect(() => validateCapture(payload(`${prefix}${'界'.repeat(700)}${suffix}`))).toThrow('INVALID_CAPTURE');
    expect(() => validateCapture(payload(`${prefix}${'a'.repeat(2050)}.jpg`))).toThrow('INVALID_CAPTURE');
  });
});
