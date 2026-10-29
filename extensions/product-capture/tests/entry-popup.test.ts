import { readFileSync } from 'node:fs';
import { JSDOM } from 'jsdom';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';

const html = readFileSync('src/popup.html', 'utf8');
const state = { loading: false, captured: false, handed: false, title: null, sourceURL: null,
  revision: crypto.randomUUID(), returnState: 'none' as 'none' | 'available' | 'uncertain', outcome: 'empty', recoveryURL: null as string | null, missingFacts: [], shortcut: 'Alt+Shift+P' as string | null, error: undefined as string | undefined };
let dom: JSDOM;
let current = { ...state };
const send = vi.fn<(message: { type: string; revision?: string }) => Promise<{ ok: boolean; state: typeof current }>>(async () => ({ ok: true, state: current }));
beforeEach(async () => {
  vi.resetModules(); vi.clearAllMocks(); current = { ...state };
  dom = new JSDOM(html, { url: 'https://popup.example.com/' }); vi.stubGlobal('document', dom.window.document);
  vi.stubGlobal('chrome', { runtime: { sendMessage: send } });
  send.mockImplementation(async () => ({ ok: true, state: current }));
  await import('../src/popup');
  await vi.waitFor(() => expect(send).toHaveBeenCalledWith({ type: 'popup.status' }));
});
afterEach(() => { dom.window.close(); vi.unstubAllGlobals(); });

it('offers original-tab return without a payload or private recovery URL', async () => {
  current = { ...state, returnState: 'available' };
  (dom.window.document.querySelector('#refresh') as HTMLButtonElement).click();
  await vi.waitFor(() => expect((dom.window.document.querySelector('#recovery') as HTMLButtonElement).hidden).toBe(false));
  expect((dom.window.document.querySelector('#capture') as HTMLButtonElement).hidden).toBe(true);
  expect(dom.window.document.querySelector('#status')?.textContent).toContain('原应用');
});

it('has one combined capture action and no independent handoff or new-tab recovery link', async () => {
  expect(dom.window.document.querySelector('#handoff')).toBeNull();
  expect(dom.window.document.querySelector('a#recovery')).toBeNull();
  expect(dom.window.document.querySelector('#capture')?.textContent).toContain('采集并在应用确认');
  (dom.window.document.querySelector('#capture') as HTMLButtonElement).click();
  await vi.waitFor(() => expect(send).toHaveBeenCalledWith({ type: 'popup.captureAndHandoff' }));
  expect(dom.window.document.querySelector('#shortcut')?.textContent).toContain('Alt+Shift+P');
});

it('returns by a message to the original tab without opening a recovery URL', async () => {
  current = { ...state, captured: true, handed: true, returnState: 'available', recoveryURL: 'https://app.example.com/capture/amazon#operationKey=original' };
  (dom.window.document.querySelector('#refresh') as HTMLButtonElement).click();
  await vi.waitFor(() => expect((dom.window.document.querySelector('#recovery') as HTMLButtonElement).hidden).toBe(false));
  (dom.window.document.querySelector('#recovery') as HTMLButtonElement).click();
  await vi.waitFor(() => expect(send).toHaveBeenCalledWith({ type: 'popup.return' }));
  expect(dom.window.document.querySelector('#recovery')?.hasAttribute('href')).toBe(false);
});

it('keeps explicit new-capture confirmation and shows an unassigned shortcut accurately', async () => {
  current = { ...state, captured: true, handed: true, returnState: 'available', shortcut: '' };
  (dom.window.document.querySelector('#refresh') as HTMLButtonElement).click();
  await vi.waitFor(() => expect(dom.window.document.querySelector('#shortcut')?.textContent).toContain('未分配'));
  (dom.window.document.querySelector('#fresh') as HTMLButtonElement).click();
  expect(send).not.toHaveBeenCalledWith(expect.objectContaining({ type: 'popup.new' }));
  (dom.window.document.querySelector('#new-confirm-button') as HTMLButtonElement).click();
  await vi.waitFor(() => expect(send).toHaveBeenCalledWith({ type: 'popup.new', revision: state.revision }));
});

it('shows a specific safe failure rather than a raw browser exception', async () => {
  current = { ...state, error: 'ORIGINAL_TAB_UNAVAILABLE' };
  (dom.window.document.querySelector('#refresh') as HTMLButtonElement).click();
  await vi.waitFor(() => expect(dom.window.document.querySelector('#status')?.textContent).toContain('检查并保留原页面'));
});

it('keeps a popup confirmation bound to the displayed entry rather than a later refresh', async () => {
  current = { ...state, returnState: 'available' };
  (dom.window.document.querySelector('#refresh') as HTMLButtonElement).click();
  await vi.waitFor(() => expect((dom.window.document.querySelector('#fresh') as HTMLButtonElement).hidden).toBe(false));
  (dom.window.document.querySelector('#fresh') as HTMLButtonElement).click();
  current = { ...current, revision: crypto.randomUUID() };
  (dom.window.document.querySelector('#refresh') as HTMLButtonElement).click();
  await vi.waitFor(() => expect(send).toHaveBeenCalledWith({ type: 'popup.status' }));
  (dom.window.document.querySelector('#new-confirm-button') as HTMLButtonElement).click();
  await vi.waitFor(() => expect(send).toHaveBeenCalledWith({ type: 'popup.new', revision: state.revision }));
});

it('does not enable capture after transport loss or an uncertain session projection', async () => {
  send.mockRejectedValueOnce(Error('connection lost'));
  (dom.window.document.querySelector('#refresh') as HTMLButtonElement).click();
  await vi.waitFor(() => expect(dom.window.document.querySelector('#status')?.textContent).toContain('状态不可读'));
  expect((dom.window.document.querySelector('#capture') as HTMLButtonElement).disabled).toBe(true);
  current = { ...state, returnState: 'uncertain' };
  (dom.window.document.querySelector('#refresh') as HTMLButtonElement).click();
  await vi.waitFor(() => expect(dom.window.document.querySelector('#status')?.textContent).toContain('定位待核实'));
  expect((dom.window.document.querySelector('#capture') as HTMLButtonElement).disabled).toBe(true);
});
