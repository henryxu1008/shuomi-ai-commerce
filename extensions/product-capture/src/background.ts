import { Controller, type CaptureTab } from './controller';
import { capturePageSource, validatePlatformCapture } from './platforms';
import { entryError, entryMessages, type EntryError } from './entry-errors';
import { pageMessage, pageSender, type PageState } from './page-messages';
declare const APP_CAPTURE_URL: string;
declare const APP_AMAZON_CAPTURE_URL: string;

async function extractCurrent(target?: CaptureTab) {
  const tabs: readonly CaptureTab[] = target ? [target] : await chrome.tabs.query({ active: true, currentWindow: true });
  if (tabs.length !== 1 || tabs[0].id === undefined || !tabs[0].url) throw Error('INVALID_PAGE');
  const tab = tabs[0]; const source = capturePageSource(tab.url!);
  const injectionTarget = tab.documentId ? { tabId: tab.id!, documentIds: [tab.documentId] } : { tabId: tab.id!, frameIds: [0] };
  let results: chrome.scripting.InjectionResult[];
  try { results = await chrome.scripting.executeScript({ target: injectionTarget, world: 'ISOLATED', files: ['extractor.js'] }); }
  catch (failure) { if (tab.documentId) throw Error('PAGE_CHANGED'); throw failure; }
  if (tab.documentId && results[0]?.documentId !== tab.documentId) throw Error('PAGE_CHANGED');
  const result = results[0]?.result as { ok?: boolean; payload?: unknown } | undefined;
  if (results.length !== 1 || results[0].frameId !== 0 || !results[0].documentId || !result?.ok) throw Error('CAPTURE_REJECTED');
  const payload = validatePlatformCapture(result.payload);
  // Verify the same document is still current without reading any other tab.
  try {
    const check = await chrome.scripting.executeScript({ target: { tabId: tab.id!, documentIds: [results[0].documentId] }, world: 'ISOLATED',
      func: () => document.URL });
    if (check.length !== 1 || check[0].documentId !== results[0].documentId || capturePageSource(check[0].result as string).sourceURL !== source.sourceURL
      || payload.evidence.sourceURL !== source.sourceURL) throw Error('PAGE_CHANGED');
  } catch { throw Error('PAGE_CHANGED'); }
  return payload;
}
const newController = () => new Controller(APP_CAPTURE_URL, extractCurrent, async url => {
  const tab = await chrome.tabs.create({ url });
  if (tab.id === undefined) throw Error('HANDOFF_UNAVAILABLE');
  return tab.id;
}, () => chrome.runtime.id, validatePlatformCapture,
  payload => capturePageSource(payload.evidence.sourceURL).platform === 'amazon' ? APP_AMAZON_CAPTURE_URL : APP_CAPTURE_URL,
  async id => {
    const tab = await chrome.tabs.update(id, { active: true });
    if (!tab || tab.id !== id || !Number.isInteger(tab.windowId) || tab.windowId < 0) throw Error('ORIGINAL_TAB_UNAVAILABLE');
    await chrome.windows.update(tab.windowId, { focused: true });
  });
let controller = newController();
let revision = crypto.randomUUID();
let error: EntryError | undefined;
const commandName = 'capture-current-product';
async function snapshot() {
  let shortcut: string | null = null;
  try { shortcut = (await chrome.commands.getAll()).find(command => command.name === commandName)?.shortcut ?? ''; }
  catch { /* Unknown assignment is different from an unassigned shortcut. */ }
  return { ...controller.snapshot(), shortcut, error };
}
async function report(code?: EntryError) {
  error = code;
  try {
    await chrome.action.setBadgeText({ text: code ? '!' : '' });
    await chrome.action.setTitle({ title: code ? `商品采集：${entryMessages[code]}（打开扩展查看）` : '商品采集 · 采集并在应用确认' });
  } catch { /* A badge failure cannot change a capture/handoff result. */ }
}
async function perform(type: string, tab?: CaptureTab) {
  switch (type) {
    case 'popup.captureAndHandoff':
      if (!controller.isBusy() && !controller.snapshot().captured) revision = crypto.randomUUID();
      await controller.captureAndHandoff(tab); break;
    case 'popup.return': await controller.returnToApplication(); break;
    case 'popup.new':
      reset(); break;
    case 'popup.status': return;
    default: throw Error('INVALID_MESSAGE');
  }
  await report();
}
function reset() {
  if (controller.isBusy()) throw Error('ACTION_IN_PROGRESS');
  controller = newController(); revision = crypto.randomUUID();
}
function pageSnapshot(sourceURL: string): PageState {
  const state = controller.snapshot();
  return { revision, loading: state.loading, captured: state.captured, handed: state.handed, sameProduct: state.sourceURL === sourceURL };
}

chrome.runtime.onMessage.addListener((message, sender, respond) => {
  if (pageMessage(message)) {
    const target = pageSender(sender);
    if (!target) return false;
    void (async () => {
      try {
        const check = await chrome.scripting.executeScript({ target: { tabId: target.id, documentIds: [target.documentId] }, world: 'ISOLATED', func: () => document.URL });
        if (check.length !== 1 || check[0].frameId !== 0 || check[0].documentId !== target.documentId
          || capturePageSource(check[0].result as string).sourceURL !== target.sourceURL) throw Error('PAGE_CHANGED');
      } catch { respond({ ok: false, code: 'PAGE_CHANGED', state: pageSnapshot(target.sourceURL) }); return; }
      try {
        if (message.type !== 'page.status') {
          // Compare and reserve/reset synchronously after document validation.
          if (message.revision !== revision) throw Error('ENTRY_CHANGED');
          if (message.type === 'page.capture' && (controller.isBusy() || controller.snapshot().captured)) throw Error('ENTRY_CHANGED');
          if (message.type === 'page.return' && !controller.snapshot().handed) throw Error('ENTRY_CHANGED');
          if (message.type === 'page.newCapture') reset();
          await perform(message.type === 'page.return' ? 'popup.return' : 'popup.captureAndHandoff', target);
        }
        respond({ ok: true, ...(error ? { code: error } : {}), state: pageSnapshot(target.sourceURL) });
      } catch (failure) {
        const code = entryError(failure); await report(code);
        respond({ ok: false, code, state: pageSnapshot(target.sourceURL) });
      }
    })();
    return true;
  }
  if (sender.id !== chrome.runtime.id || sender.url !== chrome.runtime.getURL('popup.html') || sender.tab
    || !message || Object.keys(message).length !== 1 || typeof message.type !== 'string') return false;
  void (async () => {
    try { await perform(message.type); respond({ ok: true, state: await snapshot() }); }
    catch (failure) {
      const code = entryError(failure); await report(code);
      respond({ ok: false, code, state: await snapshot() });
    }
  })();
  return true; // Compatible asynchronous response in Chrome and Edge.
});
chrome.commands.onCommand.addListener((command, tab) => {
  if (command !== commandName) return;
  void (async () => {
    try {
      if (!controller.snapshot().handed && (!tab || tab.id === undefined || !tab.url)) throw Error('INVALID_PAGE');
      await perform('popup.captureAndHandoff', tab ? { id: tab.id, url: tab.url } : undefined);
    } catch (failure) { await report(entryError(failure)); }
  })();
});
chrome.runtime.onMessageExternal.addListener((message, sender, respond) => {
  try { respond(controller.message(message, sender)); }
  catch { respond({ version: 1, code: 'CAPTURE_UNAVAILABLE' }); }
  return false;
});
