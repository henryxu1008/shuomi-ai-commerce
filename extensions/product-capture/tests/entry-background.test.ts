import { readFileSync } from 'node:fs';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { JSDOM } from 'jsdom';
import { captureAmazonDocument } from '../src/amazon/extractor';

type Reply = { ok: boolean; code?: string; state: { handed: boolean; recoveryURL: string | null; shortcut: string | null; error?: string } };
type PopupListener = (message: unknown, sender: chrome.runtime.MessageSender, reply: (value: Reply) => void) => boolean;
type CommandListener = (name: string, tab?: chrome.tabs.Tab) => void;
type ExternalListener = (message: unknown, sender: chrome.runtime.MessageSender, reply: (value: unknown) => void) => boolean;
const payload = JSON.parse(readFileSync('tests/fixtures/capture-v2.json', 'utf8'));
const source = { id: 7, url: payload.evidence.sourceURL } as chrome.tabs.Tab;
const id = 'a'.repeat(32), app = 'https://app.example.com/capture/1688';
let popup: PopupListener, command: CommandListener, external: ExternalListener;
const api = {
  runtime: { id, getURL: (path: string) => `chrome-extension://${id}/${path}`,
    onMessage: { addListener: vi.fn((fn: PopupListener) => { popup = fn; }) },
    onMessageExternal: { addListener: vi.fn((fn: ExternalListener) => { external = fn; }) } },
  tabs: { query: vi.fn(async () => [source]), create: vi.fn<(options: { url: string }) => Promise<{ id: number; windowId: number }>>().mockResolvedValue({ id: 42, windowId: 2 }),
    update: vi.fn<(id: number, options: { active: boolean }) => Promise<{ id: number; windowId: number }>>().mockResolvedValue({ id: 42, windowId: 3 }) },
  windows: { update: vi.fn<(id: number, options: { focused: boolean }) => Promise<{ id: number }>>().mockResolvedValue({ id: 3 }) },
  scripting: { executeScript: vi.fn(async (options: { files?: string[] }) => [{ frameId: 0, documentId: 'source-document', ...(options.files ? { result: { ok: true, payload } } : { result: source.url }) }]) },
  commands: { onCommand: { addListener: vi.fn((fn: CommandListener) => { command = fn; }) },
    getAll: vi.fn(async () => [{ name: 'capture-current-product', shortcut: 'Alt+Shift+P' }]) },
  action: { setBadgeText: vi.fn(async () => {}), setTitle: vi.fn(async () => {}) },
};
const sender = { id, url: `chrome-extension://${id}/popup.html` };
function send(type: string): Promise<Reply> { return new Promise((resolve, reject) => { if (!popup({ type }, sender, resolve)) reject(Error('popup refused')); }); }
function envelope() {
  const url = new URL(api.tabs.create.mock.calls[0][0].url), p = new URLSearchParams(url.hash.slice(1));
  return { version: 1, type: 'capture.read', handoffId: p.get('handoffId'), idempotencyKey: p.get('idempotencyKey') };
}
beforeEach(async () => {
  vi.resetModules(); vi.clearAllMocks();
  api.tabs.query.mockResolvedValue([source]); api.tabs.create.mockResolvedValue({ id: 42, windowId: 2 });
  api.tabs.update.mockResolvedValue({ id: 42, windowId: 3 }); api.windows.update.mockResolvedValue({ id: 3 });
  api.commands.getAll.mockResolvedValue([{ name: 'capture-current-product', shortcut: 'Alt+Shift+P' }]);
  api.scripting.executeScript.mockImplementation(async options => [{ frameId: 0, documentId: 'source-document', ...(options.files ? { result: { ok: true, payload } } : { result: source.url }) }]);
  vi.stubGlobal('chrome', api); vi.stubGlobal('APP_CAPTURE_URL', app); vi.stubGlobal('APP_AMAZON_CAPTURE_URL', 'https://app.example.com/capture/amazon');
  await import('../src/background');
});
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });

it('uses a named shortcut and its triggering tab, despite active-tab drift', async () => {
  api.tabs.query.mockResolvedValue([{ id: 99, url: 'https://www.amazon.com/dp/B012345678' } as chrome.tabs.Tab]);
  expect(command).toBeTypeOf('function');
  command('capture-current-product', source);
  await vi.waitFor(() => expect(api.tabs.create).toHaveBeenCalledOnce());
  expect(api.tabs.query).not.toHaveBeenCalled();
  expect(api.scripting.executeScript.mock.calls[0][0]).toMatchObject({ target: { tabId: 7, frameIds: [0] } });
  expect(api.scripting.executeScript.mock.calls[1][0]).toMatchObject({ target: { tabId: 7, documentIds: ['source-document'] } });
  expect(api.tabs.create.mock.calls[0][0].url).toMatch(/^https:\/\/app.example.com\/capture\/1688#/);
});

it('combines popup capture and handoff and reports the actual shortcut', async () => {
  const result = await send('popup.captureAndHandoff');
  expect(result.ok).toBe(true); expect(result.state.handed).toBe(true);
  expect(result.state.shortcut).toBe('Alt+Shift+P'); expect(api.tabs.query).toHaveBeenCalledOnce();
  expect(api.tabs.create).toHaveBeenCalledOnce();
  api.commands.getAll.mockResolvedValue([{ name: 'capture-current-product', shortcut: '' }]);
  expect((await send('popup.status')).state.shortcut).toBe('');
});

it('routes a named Amazon capture to the fixed sibling receiver using its original payload', async () => {
  const url = 'https://www.amazon.com/dp/B0BDLGZCTY';
  const dom = new JSDOM(readFileSync('tests/fixtures/amazon-inline-product.html', 'utf8'), { url });
  const amazon = await captureAmazonDocument(dom.window.document, url);
  dom.window.close();
  api.scripting.executeScript.mockImplementation(async options => [{ frameId: 0, documentId: 'amazon-document', ...(options.files ? { result: { ok: true, payload: amazon } } : { result: url }) }]);
  command('capture-current-product', { id: 17, url } as chrome.tabs.Tab);
  await vi.waitFor(() => expect(api.tabs.create).toHaveBeenCalledOnce());
  expect(api.tabs.create.mock.calls[0][0].url).toMatch(/^https:\/\/app.example.com\/capture\/amazon#/);
  expect(api.tabs.query).not.toHaveBeenCalled();
  const reply = vi.fn();
  external(envelope(), { origin: 'https://app.example.com', url: 'https://app.example.com/capture/amazon', frameId: 0, tab: { id: 42 } as chrome.tabs.Tab }, reply);
  expect(reply).toHaveBeenCalledWith(expect.objectContaining({ type: 'capture.payload', payload: amazon }));
});

it('returns to the original tab/current window without navigating, creating or extracting again', async () => {
  const initial = await send('popup.captureAndHandoff');
  const result = await send('popup.return');
  expect(result.ok).toBe(true); expect(result.state.recoveryURL).toBe(initial.state.recoveryURL);
  expect(api.tabs.update).toHaveBeenCalledExactlyOnceWith(42, { active: true });
  expect(api.windows.update).toHaveBeenCalledExactlyOnceWith(3, { focused: true });
  expect(api.tabs.create).toHaveBeenCalledOnce(); expect(api.scripting.executeScript).toHaveBeenCalledTimes(2);
});

it.each(['tab', 'window'])('preserves the original key when its %s cannot be activated', async stage => {
  const initial = await send('popup.captureAndHandoff');
  if (stage === 'tab') api.tabs.update.mockRejectedValueOnce(Error('tab unavailable'));
  else api.windows.update.mockRejectedValueOnce(Error('window unavailable'));
  const result = await send('popup.return');
  expect(result.ok).toBe(false); expect(result.code).toBe('ORIGINAL_TAB_UNAVAILABLE');
  expect(result.state.recoveryURL).toBe(initial.state.recoveryURL);
  expect(api.tabs.create).toHaveBeenCalledOnce(); expect(api.scripting.executeScript).toHaveBeenCalledTimes(2);
});

it('rejects an unsupported source before injection with an actionable failure', async () => {
  api.tabs.query.mockResolvedValue([{ id: 7, url: 'https://www.amazon.com/s?k=other' } as chrome.tabs.Tab]);
  const result = await send('popup.captureAndHandoff');
  expect(result.ok).toBe(false); expect(result.code).toBe('INVALID_PAGE');
  expect(api.scripting.executeScript).not.toHaveBeenCalled(); expect(api.tabs.create).not.toHaveBeenCalled();
});

it('treats a lost source document during the recheck as a page change without handing off', async () => {
  api.scripting.executeScript.mockResolvedValueOnce([{ frameId: 0, documentId: 'source-document', result: { ok: true, payload } }]);
  api.scripting.executeScript.mockRejectedValueOnce(Error('the source document no longer exists'));
  const result = await send('popup.captureAndHandoff');
  expect(result.ok).toBe(false); expect(result.code).toBe('PAGE_CHANGED');
  expect(api.tabs.create).not.toHaveBeenCalled();
});

it('rejects reset during original-tab activation', async () => {
  await send('popup.captureAndHandoff');
  let activate!: (value: { id: number; windowId: number }) => void;
  api.tabs.update.mockImplementationOnce(() => new Promise(resolve => { activate = resolve; }));
  const returning = send('popup.return');
  await vi.waitFor(() => expect(api.tabs.update).toHaveBeenCalledOnce());
  expect((await send('popup.new')).code).toBe('ACTION_IN_PROGRESS');
  activate({ id: 42, windowId: 3 }); await returning;
  expect(api.tabs.create).toHaveBeenCalledOnce();
});

it('accepts explicit settled reset and refuses old handoff messages afterwards', async () => {
  await send('popup.captureAndHandoff'); const old = envelope();
  expect((await send('popup.new')).ok).toBe(true);
  api.tabs.create.mockResolvedValueOnce({ id: 43, windowId: 2 });
  const next = await send('popup.captureAndHandoff');
  expect(next.ok).toBe(true); expect(api.tabs.create).toHaveBeenCalledTimes(2);
  const reply = vi.fn();
  external(old, { origin: 'https://app.example.com', url: app, frameId: 0, tab: { id: 42 } as chrome.tabs.Tab }, reply);
  expect(reply).toHaveBeenCalledWith({ version: 1, code: 'CAPTURE_UNAVAILABLE' });
});

it('rejects reset during delivery and keeps popup/shortcut on one operation', async () => {
  let finish!: (tab: { id: number; windowId: number }) => void;
  api.tabs.create.mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
  const first = send('popup.captureAndHandoff');
  await vi.waitFor(() => expect(api.tabs.create).toHaveBeenCalledOnce());
  command('capture-current-product', source);
  const reset = await send('popup.new');
  expect(reset.ok).toBe(false); expect(reset.code).toBe('ACTION_IN_PROGRESS');
  finish({ id: 42, windowId: 2 });
  const result = await first;
  await vi.waitFor(async () => expect((await send('popup.status')).state.handed).toBe(true));
  expect(result.ok).toBe(true); expect(api.tabs.create).toHaveBeenCalledOnce(); expect(api.scripting.executeScript).toHaveBeenCalledTimes(2);
});

it('fails a shortcut without a source tab without selecting another tab', async () => {
  expect(command).toBeTypeOf('function'); command('capture-current-product');
  await vi.waitFor(() => expect(api.action.setBadgeText).toHaveBeenCalledWith({ text: '!' }));
  expect(api.tabs.query).not.toHaveBeenCalled(); expect(api.scripting.executeScript).not.toHaveBeenCalled(); expect(api.tabs.create).not.toHaveBeenCalled();
  expect((await send('popup.status')).state.error).toBe('INVALID_PAGE');
});

it('retires split messages, refuses tab senders and retains the external own-tab binding', async () => {
  expect((await send('popup.capture')).ok).toBe(false); expect((await send('popup.handoff')).ok).toBe(false);
  expect(api.scripting.executeScript).not.toHaveBeenCalled();
  expect(popup({ type: 'popup.captureAndHandoff' }, { ...sender, tab: source }, vi.fn())).toBe(false);
  await send('popup.captureAndHandoff');
  const message = envelope(), reply = vi.fn();
  external(message, { origin: 'https://app.example.com', url: app, frameId: 0, tab: { id: 43 } as chrome.tabs.Tab }, reply);
  expect(reply).toHaveBeenLastCalledWith({ version: 1, code: 'CAPTURE_UNAVAILABLE' });
  external(message, { origin: 'https://app.example.com', url: app, frameId: 0, tab: { id: 42 } as chrome.tabs.Tab }, reply);
  expect(reply).toHaveBeenLastCalledWith(expect.objectContaining({ type: 'capture.payload', payload }));
});
