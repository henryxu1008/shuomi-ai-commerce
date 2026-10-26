import { JSDOM } from 'jsdom';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { mountPageEntry } from '../src/page-entry';
import type { PageState } from '../src/page-messages';

const source = 'https://www.amazon.com/dp/B0BDLGZCTY';
const empty: PageState = { revision: crypto.randomUUID(), loading: false, captured: false, handed: false, sameProduct: false, returnState: 'none' };
let state = { ...empty }, dom: JSDOM;
const send = vi.fn<(message: unknown) => Promise<{ ok: boolean; state: typeof state }>>(async () => ({ ok: true, state: { ...state } }));
beforeEach(() => { state = { ...empty }; vi.clearAllMocks(); send.mockImplementation(async () => ({ ok: true, state: { ...state } })); vi.stubGlobal('chrome', { runtime: { sendMessage: send } }); });
afterEach(() => { dom?.window.close(); vi.unstubAllGlobals(); vi.restoreAllMocks(); });
function mount(url = source) { dom = new JSDOM('<!doctype html><html><body></body></html>', { url }); return mountPageEntry(dom.window.document, dom.window as unknown as Window)!; }
function trusted(button: HTMLButtonElement) {
  // Capture registered handlers in unit tests; the browser fixture uses real input.
  const event = { isTrusted: true, preventDefault() {}, stopPropagation() {} } as Event;
  const handlers = clicks.get(button) ?? []; for (const handler of handlers) handler(event);
}
const clicks = new Map<EventTarget, ((event: Event) => void)[]>();
function spyClicks() {
  const prototype = dom.window.EventTarget.prototype, original = prototype.addEventListener;
  vi.spyOn(prototype, 'addEventListener').mockImplementation(function (this: EventTarget, type, callback, options) {
    if (type === 'click' && typeof callback === 'function') clicks.set(this, [...(clicks.get(this) ?? []), callback as (event: Event) => void]);
    return original.call(this, type, callback, options);
  });
}
async function mounted() {
  dom = new JSDOM('<!doctype html><html><body></body></html>', { url: source }); clicks.clear(); spyClicks();
  const panel = mountPageEntry(dom.window.document, dom.window as unknown as Window)!;
  const button = (id: string) => panel.host.shadowRoot!.querySelector<HTMLButtonElement>(`#${id}`)!;
  await vi.waitFor(() => expect(button('refresh').disabled).toBe(false));
  return { ...panel, button };
}

it('keeps family opt-in off and sends its scope only for an explicit Amazon button capture',async()=>{
 const panel=await mounted();const option=panel.host.shadowRoot!.querySelector<HTMLInputElement>('#family')!;
 expect(option).not.toBeNull();expect(option.checked).toBe(false);option.checked=true;trusted(option as unknown as HTMLButtonElement);
 trusted(panel.button('primary'));await vi.waitFor(()=>expect(send).toHaveBeenCalledWith({type:'page.capture',revision:empty.revision,scope:'family'}));
});
it('returns to family progress without starting or resetting another capture',async()=>{
 state={...empty,family:{phase:'collecting',observed:1,total:171,reason:''}} as PageState;
 const panel=await mounted();expect(panel.button('primary').textContent).toBe('查看采集进度');
 expect(panel.button('fresh').disabled).toBe(true);trusted(panel.button('primary'));
 await vi.waitFor(()=>expect(send).toHaveBeenCalledWith({type:'page.return',revision:empty.revision}));
});
it('does not accept a page script setting the opt-in checkbox without a trusted input click',async()=>{
 const panel=await mounted();panel.host.shadowRoot!.querySelector<HTMLInputElement>('#family')!.checked=true;
 trusted(panel.button('primary'));await vi.waitFor(()=>expect(send).toHaveBeenCalledWith({type:'page.capture',revision:empty.revision}));
 expect(send).not.toHaveBeenCalledWith(expect.objectContaining({scope:'family'}));
});

it('offers only original-tab navigation after return-only hydration, including prior uncertainty', async () => {
  state = { ...empty, captured: true, handed: true, sameProduct: true, returnState: 'available' };
  const panel = await mounted();
  send.mockRejectedValueOnce(Error('response lost'));
  trusted(panel.button('primary'));
  await vi.waitFor(() => expect(panel.host.shadowRoot!.textContent).toContain('连接中断'));
  state = { ...empty, sameProduct: true, returnState: 'available' };
  await panel.refresh();
  expect(panel.button('primary').textContent).toBe('返回应用核实');
  expect(panel.button('primary').disabled).toBe(false);
  trusted(panel.button('primary'));
  await vi.waitFor(() => expect(send).toHaveBeenLastCalledWith({ type: 'page.return', revision: empty.revision }));
  expect(send).not.toHaveBeenCalledWith(expect.objectContaining({ type: 'page.capture' }));
});

it.each(['https://www.amazon.com/s?k=bed', 'https://www.amazon.com/dp/invalid', 'https://detail.1688.com/offer/01.html', 'http://detail.1688.com/offer/123.html', 'https://amazon.com/dp/B0BDLGZCTY', 'https://evil.example/dp/B0BDLGZCTY'])('does not mount on unsupported page %s', url => {
  expect(mount(url)).toBeUndefined(); expect(send).not.toHaveBeenCalled();
});
it.each(['https://detail.1688.com/offer/123.html', ...['com','ca','co.uk','de','co.jp'].map(site => `https://www.amazon.${site}/item/dp/B0BDLGZCTY`)])('mounts once, exposes a labelled native button and only reads status on %s', async url => {
  const panel = mount(url);
  await panel.refresh();
  expect(mountPageEntry(dom.window.document, dom.window as unknown as Window)).toBeUndefined();
  expect(panel.host.shadowRoot!.querySelector('#primary')?.textContent).toBe('采集并在应用确认');
  expect(panel.host.shadowRoot!.querySelector('[role="status"]')).not.toBeNull();
  expect(send.mock.calls.every(([message]) => (message as { type: string }).type === 'page.status')).toBe(true);
});
it('does not mount in a child frame even with a supported product URL', () => {
  dom = new JSDOM(`<iframe src="${source}"></iframe>`, { url: source });
  const child = dom.window.document.querySelector('iframe')!.contentWindow!;
  expect(mountPageEntry(child.document, child)).toBeUndefined(); expect(send).not.toHaveBeenCalled();
});
it('ignores synthetic capture/reset/confirm/return clicks and uses revision-bound trusted intent', async () => {
  const panel = await mounted();
  panel.button('primary').click(); expect(send).not.toHaveBeenCalledWith(expect.objectContaining({ type: 'page.capture' }));
  trusted(panel.button('primary'));
  await vi.waitFor(() => expect(send).toHaveBeenCalledWith({ type: 'page.capture', revision: empty.revision }));
  await vi.waitFor(() => expect(panel.button('refresh').disabled).toBe(false));
  state = { ...state, captured: true, handed: true, sameProduct: true, returnState: 'available' }; await panel.refresh();
  expect(panel.button('primary').textContent).toBe('返回应用核实');
  trusted(panel.button('primary')); await vi.waitFor(() => expect(send).toHaveBeenCalledWith({ type: 'page.return', revision: empty.revision }));
  await vi.waitFor(() => expect(panel.button('refresh').disabled).toBe(false));
  panel.button('fresh').click(); expect(panel.button('confirm').closest<HTMLElement>('#confirmation')!.hidden).toBe(true);
  trusted(panel.button('fresh')); panel.button('confirm').click();
  expect(send).not.toHaveBeenCalledWith(expect.objectContaining({ type: 'page.newCapture' }));
  trusted(panel.button('cancel')); expect(panel.button('confirm').closest<HTMLElement>('#confirmation')!.hidden).toBe(true);
  trusted(panel.button('fresh')); trusted(panel.button('confirm'));
  await vi.waitFor(() => expect(send).toHaveBeenCalledWith({ type: 'page.newCapture', revision: empty.revision }));
});
it('distinguishes another product, invalidates stale confirmation and disables busy controls', async () => {
  state = { ...state, captured: true, handed: true, returnState: 'available' }; const panel = await mounted();
  expect(panel.button('primary').textContent).toBe('返回已有采集');
  expect(panel.host.shadowRoot!.textContent).toContain('另一件商品');
  trusted(panel.button('fresh'));
  state = { ...state, revision: crypto.randomUUID(), loading: true }; await panel.refresh();
  expect(panel.button('confirm').closest<HTMLElement>('#confirmation')!.hidden).toBe(true);
  expect(panel.button('primary').disabled).toBe(true); expect(panel.button('fresh').disabled).toBe(true);
  trusted(panel.button('confirm')); expect(send).not.toHaveBeenCalledWith(expect.objectContaining({ type: 'page.newCapture' }));
});
it('does not replay after transport loss and requires explicit confirmation after known worker state is lost', async () => {
  state = { ...state, captured: true, handed: true, returnState: 'available' }; const panel = await mounted();
  send.mockRejectedValueOnce(Error('sensitive raw failure'));
  trusted(panel.button('primary'));
  await vi.waitFor(() => expect(panel.host.shadowRoot!.textContent).toContain('连接中断'));
  expect(panel.button('primary').disabled).toBe(true); expect(panel.host.shadowRoot!.textContent).not.toContain('sensitive');
  state = { ...empty, revision: crypto.randomUUID() }; await panel.refresh();
  expect(panel.button('primary').disabled).toBe(true); expect(panel.button('fresh').hidden).toBe(false);
  expect(panel.host.shadowRoot!.textContent).toContain('原应用');
});
it('removes the panel after unsupported navigation and rejects confirmation after product URL changes', async () => {
  state = { ...state, captured: true, handed: true, returnState: 'available' }; const panel = await mounted();
  trusted(panel.button('fresh'));
  dom.reconfigure({ url: 'https://www.amazon.com/dp/B012345678' });
  trusted(panel.button('confirm'));
  expect(send).not.toHaveBeenCalledWith(expect.objectContaining({ type: 'page.newCapture' }));
  dom.reconfigure({ url: 'https://www.amazon.com/s?k=bed' }); await panel.refresh();
  expect(panel.host.isConnected).toBe(false);
});
it.each(['preflight', 'capture'])('does not continue %s after same-document product navigation', async phase => {
  const panel = await mounted();
  let respond!: (value: { ok: boolean; state: typeof state }) => void;
  if (phase === 'capture') send.mockImplementationOnce(async () => ({ ok: true, state: { ...state } }));
  send.mockImplementationOnce(() => new Promise(resolve => { respond = resolve; }));
  trusted(panel.button('primary'));
  await vi.waitFor(() => expect(respond).toBeTypeOf('function'));
  dom.reconfigure({ url: 'https://www.amazon.com/dp/B012345678' });
  respond({ ok: true, state: { ...state, captured: true, handed: true, sameProduct: true, returnState: 'available' } });
  await vi.waitFor(() => expect(panel.host.shadowRoot!.textContent).toContain('页面已变化'));
  expect(panel.button('primary').disabled).toBe(true);
  expect(send.mock.calls.filter(([message]) => (message as { type: string }).type === 'page.capture')).toHaveLength(phase === 'capture' ? 1 : 0);
});
it('latches a first capture transport loss across focus and an empty restarted worker', async () => {
  const panel = await mounted();
  send.mockImplementationOnce(async () => ({ ok: true, state: { ...state } }));
  send.mockRejectedValueOnce(Error('response lost'));
  trusted(panel.button('primary'));
  await vi.waitFor(() => expect(panel.host.shadowRoot!.textContent).toContain('连接中断'));
  const callsAfterLoss = send.mock.calls.length;
  state = { ...empty, revision: crypto.randomUUID() };
  dom.window.dispatchEvent(new dom.window.Event('focus'));
  await Promise.resolve();
  expect(send).toHaveBeenCalledTimes(callsAfterLoss);
  expect(panel.button('primary').disabled).toBe(true);
  await panel.refresh();
  expect(panel.button('primary').disabled).toBe(true);
  expect(panel.button('fresh').hidden).toBe(false); expect(panel.button('fresh').disabled).toBe(false);
  trusted(panel.button('fresh')); trusted(panel.button('confirm'));
  await vi.waitFor(() => expect(send).toHaveBeenCalledWith({ type: 'page.newCapture', revision: state.revision }));
});

it.each([
  { loading: true }, { captured: true }, { handed: true },
  { returnState: 'available' as const, sameProduct: true }, { returnState: 'uncertain' as const },
])('only refreshes the initial view when preflight finds a changed entry: %j', async changed => {
  const panel = await mounted();
  const before = send.mock.calls.length;
  state = { ...empty, revision: crypto.randomUUID(), ...changed };
  trusted(panel.button('primary'));
  await vi.waitFor(() => expect(send).toHaveBeenCalledTimes(before + 1));
  expect(send).toHaveBeenLastCalledWith({ type: 'page.status' });
  expect(send.mock.calls.every(([message]) => (message as { type: string }).type === 'page.status')).toBe(true);
});

it('holds the same busy boundary through delayed preflight and sends capture only once', async () => {
  const panel = await mounted();
  const before = send.mock.calls.length;
  let finish!: (value: { ok: boolean; state: typeof state }) => void;
  send.mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
  trusted(panel.button('primary')); trusted(panel.button('primary')); trusted(panel.button('refresh'));
  await panel.refresh();
  expect(send).toHaveBeenCalledTimes(before + 1);
  expect(panel.button('primary').disabled).toBe(true); expect(panel.button('refresh').disabled).toBe(true);
  state = { ...empty, revision: crypto.randomUUID() };
  finish({ ok: true, state: { ...state } });
  await vi.waitFor(() => expect(send).toHaveBeenLastCalledWith({ type: 'page.capture', revision: state.revision }));
  expect(send.mock.calls.filter(([message]) => (message as { type: string }).type === 'page.capture')).toHaveLength(1);
});

it('does not replay capture when the initial preflight response is lost', async () => {
  const panel = await mounted();
  send.mockRejectedValueOnce(Error('response lost'));
  trusted(panel.button('primary'));
  await vi.waitFor(() => expect(panel.host.shadowRoot!.textContent).toContain('连接中断'));
  dom.window.dispatchEvent(new dom.window.Event('focus'));
  expect(send.mock.calls.every(([message]) => (message as { type: string }).type === 'page.status')).toBe(true);
  expect(panel.button('primary').disabled).toBe(true);
});

it('does not capture after a rejected preflight even when its projection is empty', async () => {
  const panel = await mounted();
  send.mockImplementationOnce(async () => ({ ok: false, code: 'PAGE_CHANGED', state: { ...state } }));
  trusted(panel.button('primary'));
  await vi.waitFor(() => expect(panel.host.shadowRoot!.textContent).toContain('页面已变化'));
  expect(send.mock.calls.every(([message]) => (message as { type: string }).type === 'page.status')).toBe(true);
});

it('clears a previous status error only after the single capture response succeeds', async () => {
  const panel = await mounted();
  send.mockImplementationOnce(async () => ({ ok: true, code: 'ENTRY_CHANGED', state: { ...state } }));
  send.mockImplementationOnce(async () => ({ ok: true, state: { ...state, captured: true, handed: true, sameProduct: true, returnState: 'available' } }));
  trusted(panel.button('primary'));
  await vi.waitFor(() => expect(panel.host.shadowRoot!.textContent).toContain('可返回原应用'));
  expect(panel.host.shadowRoot!.textContent).not.toContain('采集操作已变化');
  expect(send.mock.calls.filter(([message]) => (message as { type: string }).type === 'page.capture')).toHaveLength(1);
});

it('does not retry or change intent if the entry changes after successful preflight', async () => {
  const panel = await mounted();
  send.mockImplementationOnce(async () => ({ ok: true, state: { ...state } }));
  send.mockImplementationOnce(async () => ({ ok: false, code: 'ENTRY_CHANGED', state: { ...state, captured: true, handed: true, returnState: 'available' } }));
  trusted(panel.button('primary'));
  await vi.waitFor(() => expect(panel.host.shadowRoot!.textContent).toContain('采集操作已变化'));
  expect(send.mock.calls.filter(([message]) => (message as { type: string }).type === 'page.capture')).toHaveLength(1);
  expect(send).not.toHaveBeenCalledWith(expect.objectContaining({ type: 'page.return' }));
  expect(send).not.toHaveBeenCalledWith(expect.objectContaining({ type: 'page.newCapture' }));
});
