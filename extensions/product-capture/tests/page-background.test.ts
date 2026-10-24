import { readFileSync } from 'node:fs';
import { JSDOM } from 'jsdom';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { captureAmazonDocument } from '../src/amazon/extractor';
import { mountPageEntry } from '../src/page-entry';
import { returnStorage } from './fixtures/return-storage';

type State = { revision: string; loading: boolean; captured: boolean; handed: boolean; sameProduct: boolean; returnState: 'none' | 'available' | 'uncertain' };
type Reply = { ok: boolean; code?: string; state: State };
type Listener = (message: unknown, sender: chrome.runtime.MessageSender, respond: (value: Reply) => void) => boolean;
const payload = JSON.parse(readFileSync('tests/fixtures/capture-v2.json', 'utf8'));
const url = payload.evidence.sourceURL, id = 'a'.repeat(32);
const tab = { id: 7, url } as chrome.tabs.Tab;
const sender: chrome.runtime.MessageSender = { id, url, origin: new URL(url).origin, tab, frameId: 0, documentId: 'doc-1', documentLifecycle: 'active' };
let listener: Listener, command: (name: string, tab?: chrome.tabs.Tab) => void;
const storage = returnStorage();
const api = {
  storage,
  runtime: { id, getURL: (path: string) => `chrome-extension://${id}/${path}`,
    onMessage: { addListener: vi.fn((fn: Listener) => { listener = fn; }) }, onMessageExternal: { addListener: vi.fn() } },
  tabs: { query: vi.fn(async () => [tab]), create: vi.fn<(options: { url: string }) => Promise<{ id: number; windowId: number }>>().mockResolvedValue({ id: 42, windowId: 2 }),
    update: vi.fn(async () => ({ id: 42, windowId: 2 })) }, windows: { update: vi.fn(async () => ({})) },
  scripting: { executeScript: vi.fn(async (options: { files?: string[] }) => [{ frameId: 0, documentId: 'doc-1', result: options.files ? { ok: true, payload } : url }]) },
  commands: { getAll: vi.fn(async () => []), onCommand: { addListener: vi.fn((fn: typeof command) => { command = fn; }) } },
  action: { setBadgeText: vi.fn(async () => {}), setTitle: vi.fn(async () => {}) },
};
const send = (message: unknown, from = sender) => new Promise<Reply>((resolve, reject) => {
  if (!listener(message, from, resolve)) reject(Error('refused'));
});
const status = () => send({ type: 'page.status' });
const mutate = (type: string, revision: string) => send({ type, revision });
beforeEach(async () => {
  vi.resetModules(); vi.clearAllMocks();
  storage.reset();
  api.scripting.executeScript.mockImplementation(async options => [{ frameId: 0, documentId: 'doc-1', result: options.files ? { ok: true, payload } : url }]);
  api.tabs.create.mockResolvedValue({ id: 42, windowId: 2 }); api.tabs.update.mockResolvedValue({ id: 42, windowId: 2 });
  vi.stubGlobal('chrome', api); vi.stubGlobal('APP_CAPTURE_URL', 'https://app.example.com/capture/1688');
  vi.stubGlobal('APP_AMAZON_CAPTURE_URL', 'https://app.example.com/capture/amazon');
  await import('../src/background');
});
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });

it.each(['1688', 'amazon'])('captures %s once on the first page click after an empty background is recreated', async platform => {
  const targetURL = platform === 'amazon' ? 'https://www.amazon.com/dp/B0BDLGZCTY' : url;
  const html = readFileSync(platform === 'amazon' ? 'tests/fixtures/amazon-inline-product.html' : 'tests/fixtures/product.html', 'utf8');
  const dom = new JSDOM(html, { url: targetURL });
  try {
    const targetPayload = platform === 'amazon' ? await captureAmazonDocument(dom.window.document, targetURL) : payload;
    const targetSender = { ...sender, url: targetURL, origin: new URL(targetURL).origin, tab: { ...tab, url: targetURL } };
    const sendPage = vi.fn((message: unknown) => send(message, targetSender));
    vi.stubGlobal('chrome', { ...api, runtime: { ...api.runtime, sendMessage: sendPage } });
    api.scripting.executeScript.mockImplementation(async options => [{ frameId: 0, documentId: 'doc-1', result: options.files ? { ok: true, payload: targetPayload } : targetURL }]);
    let primaryClick!: (event: Event) => void;
    const prototype = dom.window.EventTarget.prototype, add = prototype.addEventListener;
    vi.spyOn(prototype, 'addEventListener').mockImplementation(function (this: EventTarget, type, callback, options) {
      if (this instanceof dom.window.HTMLButtonElement && this.id === 'primary' && type === 'click' && typeof callback === 'function') primaryClick = callback as (event: Event) => void;
      return add.call(this, type, callback, options);
    });
    const panel = mountPageEntry(dom.window.document, dom.window as unknown as Window)!;
    const primary = panel.host.shadowRoot!.querySelector<HTMLButtonElement>('#primary')!;
    await vi.waitFor(() => expect(primary.disabled).toBe(false));
    const initial = await sendPage.mock.results[0].value;
    expect(storage.contents()).toEqual({});
    vi.resetModules(); await import('../src/background');
    const recreated = await send({ type: 'page.status' }, targetSender);
    expect(recreated.state.revision).not.toBe(initial.state.revision);
    expect(recreated.state).toMatchObject({ captured: false, handed: false, returnState: 'none' });
    primaryClick({ isTrusted: true, preventDefault() {}, stopPropagation() {} } as Event);
    await vi.waitFor(() => expect(primary.disabled).toBe(false));
    expect((await sendPage.mock.results.at(-1)!.value).code).toBeUndefined();
    expect(api.tabs.create).toHaveBeenCalledOnce();
    await vi.waitFor(() => expect(primary.textContent).toBe('返回应用核实'));
    const captures = sendPage.mock.calls.filter(([message]) => (message as { type: string }).type === 'page.capture');
    expect(captures).toEqual([[{ type: 'page.capture', revision: recreated.state.revision }]]);
    expect(api.scripting.executeScript.mock.calls.filter(([options]) => options.files)).toHaveLength(1);
    expect(api.tabs.create.mock.calls[0][0]).toMatchObject({ url: expect.stringContaining(`/capture/${platform}#`) });
  } finally { dom.window.close(); }
});

it('still rejects a stale initial revision directly after recreation without capturing', async () => {
  const initial = await status();
  vi.resetModules(); await import('../src/background');
  expect((await mutate('page.capture', initial.state.revision)).code).toBe('ENTRY_CHANGED');
  expect(api.tabs.create).not.toHaveBeenCalled();
  expect(api.scripting.executeScript.mock.calls.every(([options]) => !options.files)).toBe(true);
  expect(storage.contents()).toEqual({});
});

it('restores return-only location and the same revision after background recreation', async () => {
  const initial = await status();
  const handed = await mutate('page.capture', initial.state.revision);
  expect(handed.ok).toBe(true);
  vi.resetModules(); await import('../src/background');
  const restored = await status();
  expect(restored.state).toMatchObject({ revision: handed.state.revision, returnState: 'available', captured: false, handed: false, sameProduct: true });
  expect((await mutate('page.return', handed.state.revision)).ok).toBe(true);
  expect(api.tabs.update).toHaveBeenCalledExactlyOnceWith(42, { active: true });
  expect(api.tabs.create).toHaveBeenCalledOnce();
  expect(api.scripting.executeScript.mock.calls.filter(([options]) => options.files)).toHaveLength(1);
});

it('loads a narrow read-only projection and captures only the browser sender document', async () => {
  const initial = await status();
  expect(initial.ok).toBe(true);
  expect(Object.keys(initial.state).sort()).toEqual(['captured', 'handed', 'loading', 'returnState', 'revision', 'sameProduct']);
  expect(api.scripting.executeScript.mock.calls.every(([options]) => !options.files)).toBe(true);
  expect(api.tabs.create).not.toHaveBeenCalled();
  const result = await mutate('page.capture', initial.state.revision);
  expect(result.ok).toBe(true); expect(result.state.handed).toBe(true); expect(result.state.sameProduct).toBe(true);
  expect(result.state.revision).not.toBe(initial.state.revision);
  expect(api.scripting.executeScript.mock.calls.find(([options]) => options.files)?.[0]).toMatchObject({ target: { tabId: 7, documentIds: ['doc-1'] } });
  expect(api.tabs.query).not.toHaveBeenCalled(); expect(api.tabs.create).toHaveBeenCalledOnce();
  expect(JSON.stringify(result)).not.toMatch(/payload|handoffId|idempotencyKey|recoveryURL|app\.example/);
});

it.each([
  { id: 'b'.repeat(32) }, { frameId: 1 }, { documentId: undefined }, { documentLifecycle: undefined },
  { documentLifecycle: 'prerender' }, { origin: 'https://evil.example' }, { tab: { ...tab, url: 'https://www.amazon.com/dp/B012345678' } },
  { url: 'https://detail.1688.com/offer/001.html' }, { url: 'http://detail.1688.com/offer/123.html', origin: 'http://detail.1688.com' },
])('rejects non-source or non-active main-document sender: %j', async patch => {
  await expect(send({ type: 'page.status' }, { ...sender, ...patch } as chrome.runtime.MessageSender)).rejects.toThrow('refused');
  expect(api.scripting.executeScript).not.toHaveBeenCalled(); expect(api.tabs.create).not.toHaveBeenCalled();
});

it.each([{ type: 'page.status', url }, { type: 'page.capture' }, { type: 'page.capture', revision: 'x'.repeat(10000) },
  { type: 'page.return', revision: crypto.randomUUID(), tabId: 1 }, { type: 'page.newCapture', revision: crypto.randomUUID(), payload },
])('rejects unknown/oversized message fields case %#', async message => {
  await expect(send(message)).rejects.toThrow('refused'); expect(api.tabs.create).not.toHaveBeenCalled();
});

it('refuses replaced documents and same-document product changes before capture', async () => {
  const initial = await status();
  api.scripting.executeScript.mockResolvedValueOnce([{ frameId: 0, documentId: 'doc-2', result: url }]);
  expect((await mutate('page.capture', initial.state.revision)).code).toBe('PAGE_CHANGED');
  api.scripting.executeScript.mockResolvedValueOnce([{ frameId: 0, documentId: 'doc-1', result: 'https://detail.1688.com/offer/111.html' }]);
  expect((await mutate('page.capture', initial.state.revision)).code).toBe('PAGE_CHANGED');
  expect(api.tabs.create).not.toHaveBeenCalled();
});

it('does not turn stale capture into return; explicit return preserves the own tab without navigation', async () => {
  const initial = await status(); await mutate('page.capture', initial.state.revision);
  expect((await mutate('page.capture', initial.state.revision)).code).toBe('ENTRY_CHANGED');
  const current = await status();
  expect((await mutate('page.capture', current.state.revision)).code).toBe('ENTRY_CHANGED');
  expect(api.tabs.update).not.toHaveBeenCalled();
  expect((await mutate('page.return', current.state.revision)).ok).toBe(true);
  expect(api.tabs.update).toHaveBeenCalledExactlyOnceWith(42, { active: true });
  expect(api.windows.update).toHaveBeenCalledExactlyOnceWith(2, { focused: true });
  expect(api.tabs.create).toHaveBeenCalledOnce();
});

it('reports another product without exposing its evidence and requires explicit revision-bound new capture', async () => {
  const initial = await status(); await mutate('page.capture', initial.state.revision);
  const otherURL = 'https://www.amazon.com/dp/B0BDLGZCTY';
  const other = { ...sender, url: otherURL, origin: new URL(otherURL).origin, tab: { ...tab, url: otherURL } };
  api.scripting.executeScript.mockResolvedValueOnce([{ frameId: 0, documentId: 'doc-1', result: otherURL }]);
  const projection = await send({ type: 'page.status' }, other);
  expect(projection.state.sameProduct).toBe(false); expect(projection.state.handed).toBe(true);
  expect(JSON.stringify(projection)).not.toContain(url);
  expect((await mutate('page.newCapture', initial.state.revision)).code).toBe('ENTRY_CHANGED');
  expect(api.tabs.create).toHaveBeenCalledOnce();
  const next = await mutate('page.newCapture', projection.state.revision);
  expect(next.ok).toBe(true); expect(api.tabs.create).toHaveBeenCalledTimes(2);
});

it('rejects stale new confirmations after shortcut/popup reserve an empty entry and refuses busy reset', async () => {
  const initial = await status();
  let finish!: (value: { id: number; windowId: number }) => void;
  api.tabs.create.mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
  command('capture-current-product', tab);
  await vi.waitFor(() => expect(api.tabs.create).toHaveBeenCalledOnce());
  expect((await mutate('page.newCapture', initial.state.revision)).code).toBe('ENTRY_CHANGED');
  const busy = await status();
  expect((await mutate('page.newCapture', busy.state.revision)).code).toBe('ACTION_IN_PROGRESS');
  finish({ id: 42, windowId: 2 });
  await vi.waitFor(async () => expect((await status()).state.loading).toBe(false));
  expect(api.tabs.create).toHaveBeenCalledOnce();
});

it('keeps unknown/original-tab errors narrow and never opens a replacement', async () => {
  const initial = await status(); const handed = await mutate('page.capture', initial.state.revision);
  api.tabs.update.mockRejectedValueOnce(Error('secret raw browser error'));
  const failed = await mutate('page.return', handed.state.revision);
  expect(failed.code).toBe('ORIGINAL_TAB_UNAVAILABLE');
  expect(JSON.stringify(failed)).not.toMatch(/secret|payload|recoveryURL|operationKey/);
  expect(api.tabs.create).toHaveBeenCalledOnce();
  vi.resetModules(); await import('../src/background');
  expect((await mutate('page.capture', handed.state.revision)).code).toBe('ENTRY_CHANGED');
  expect(api.tabs.create).toHaveBeenCalledOnce();
});
