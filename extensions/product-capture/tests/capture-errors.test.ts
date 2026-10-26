import { afterEach, expect, it, vi } from 'vitest';
import { JSDOM } from 'jsdom';
import { runCapture, familyInfo } from '../src/injected';
import { extractTab, extractFamilyTab } from '../src/capture-tab';
import * as amazon from '../src/amazon/extractor';
import * as discovery from '../src/amazon/family-discovery';
import { entryMessages } from '../src/entry-errors';

const source = 'https://www.amazon.com/dp/B0BDLGZCTY';
const documents: JSDOM[] = [];
function page(url = source, html = '<h1 id="productTitle">Product</h1>', ready = 'complete') {
  const dom = new JSDOM(html, { url }); documents.push(dom);
  Object.defineProperty(dom.window.document, 'readyState', { value: ready });
  vi.stubGlobal('document', dom.window.document);
  return dom.window.document;
}
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); documents.splice(0).forEach(d => d.window.close()); });

it.each(['/checkpoint/continue?token=DO_NOT_FORWARD', '/errors/validateCaptcha'])('recognizes %s before product parsing', async path => {
  page(`https://www.amazon.com${path}`, '<h1>Check the box to continue.</h1><input type="checkbox" id="continueCheck">');
  const parser = vi.spyOn(amazon, 'captureAmazonDocument');
  expect(await runCapture()).toEqual({ ok: false, code: 'AMAZON_VERIFICATION_REQUIRED' });
  expect(parser).not.toHaveBeenCalled();
});
it.each([
  [source, '<input type="checkbox" id="continueCheck"><h1 id="productTitle">Recommendation</h1>', 'AMAZON_VERIFICATION_REQUIRED'],
  ['https://www.amazon.com/ap/signin', '<form name="signIn"><input type="password"></form>', 'AMAZON_SIGN_IN_REQUIRED'],
  [source, '<form name="signIn"><input type="password"></form>', 'AMAZON_SIGN_IN_REQUIRED'],
  [source, '<div>Product loading</div>', 'PAGE_NOT_READY'],
  ['https://evil.test/checkpoint/continue', '<input id="continueCheck">', 'INVALID_PAGE'],
  ['https://www.amazon.com/s?k=product', '<h1 id="productTitle">Recommendation</h1>', 'INVALID_PAGE'],
])('classifies only explicit supported documents: %s', async (url, html, code) => {
  page(url, html); expect(await runCapture()).toEqual({ ok: false, code });
});
it('does not parse a document still loading', async () => {
  page(source, '<h1 id="productTitle">Product</h1>', 'loading');
  const parser = vi.spyOn(amazon, 'captureAmazonDocument');
  expect(await runCapture()).toEqual({ ok: false, code: 'PAGE_NOT_READY' }); expect(parser).not.toHaveBeenCalled();
});
it.each(['CAPTURE_TOO_LARGE', 'CAPTURE_TIMEOUT', 'SOURCE_CHANGED', 'UNSUPPORTED_PAGE', 'INVALID_CAPTURE'])('preserves safe parser code %s', async code => {
  page(); vi.spyOn(amazon, 'captureAmazonDocument').mockRejectedValueOnce(Error(code));
  expect(await runCapture()).toEqual({ ok: false, code }); expect(Object.hasOwn(entryMessages, code)).toBe(true);
});
it.each([Error('token=secret'), Error('UNKNOWN_INTERNAL_ERROR'), { message: 'CAPTURE_TIMEOUT' }, 'CAPTURE_TIMEOUT'])('redacts unknown parser failures', async error => {
  page(); vi.spyOn(amazon, 'captureAmazonDocument').mockRejectedValueOnce(error);
  expect(await runCapture()).toEqual({ ok: false, code: 'CAPTURE_REJECTED' });
});
it('redacts family discovery exceptions too', () => {
  page(); vi.spyOn(discovery, 'discoverFamily').mockImplementationOnce(() => { throw Error('token=secret'); });
  expect(familyInfo()).toEqual({ ok: false, code: 'CAPTURE_REJECTED' });
});

function injection(code: unknown, options: { frame?: number; document?: string; replaced?: boolean } = {}) {
  const executeScript = vi.fn(async (o: { files?: string[] }) => o.files
    ? [{ frameId: options.frame ?? 0, documentId: options.document ?? 'native-child', result: { ok: false, code } }]
    : [{ frameId: 0, documentId: options.replaced ? 'replacement' : 'native-child', result: true }]);
  vi.stubGlobal('chrome', { scripting: { executeScript } }); return executeScript;
}
it.each(['AMAZON_VERIFICATION_REQUIRED', 'AMAZON_SIGN_IN_REQUIRED', 'PAGE_NOT_READY', 'CAPTURE_TOO_LARGE', 'CAPTURE_TIMEOUT'])('propagates %s only after native document verification', async code => {
  const execute = injection(code);
  await expect(extractFamilyTab({ id: 50, url: source }, false)).rejects.toThrow(code);
  expect(execute).toHaveBeenCalledTimes(2);
  expect(execute.mock.calls[1][0]).toMatchObject({ target: { tabId: 50, documentIds: ['native-child'] } });
});
it.each(['token=secret', 'UNKNOWN_INTERNAL_ERROR', { code: 'CAPTURE_TIMEOUT' }])('redacts unrecognized injected error %s', async code => {
  injection(code); await expect(extractTab({ id: 50, url: source })).rejects.toThrow('CAPTURE_REJECTED');
});
it('rejects a replaced error document rather than forwarding its reason', async () => {
  injection('AMAZON_VERIFICATION_REQUIRED', { replaced: true });
  await expect(extractTab({ id: 50, url: source })).rejects.toThrow('PAGE_CHANGED');
});
it.each([{ frame: 1 }, { document: '' }])('rejects unbound native error results', async options => {
  const execute = injection('AMAZON_VERIFICATION_REQUIRED', options);
  await expect(extractTab({ id: 50, url: source })).rejects.toThrow('CAPTURE_REJECTED'); expect(execute).toHaveBeenCalledOnce();
});
it('checks a caller-pinned document before propagating an error', async () => {
  injection('AMAZON_VERIFICATION_REQUIRED');
  await expect(extractTab({ id: 50, url: source, documentId: 'original' })).rejects.toThrow('PAGE_CHANGED');
});
