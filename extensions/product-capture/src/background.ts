import { Controller, type CaptureTab } from './controller';
import { capturePageSource, validatePlatformCapture } from './platforms';
import { entryError, entryMessages, type EntryError } from './entry-errors';
import { pageMessage, pageSender, type PageState } from './page-messages';
import { ReturnSession } from './return-session';
import { UUID } from './validation';
import { extractTab } from './capture-tab';
import { FamilyBridge, isFamilyCommand } from './amazon/family-bridge';
declare const APP_CAPTURE_URL: string;
declare const APP_AMAZON_CAPTURE_URL: string;

async function extractCurrent(target?: CaptureTab) {
  const tabs: readonly CaptureTab[] = target ? [target] : await chrome.tabs.query({ active: true, currentWindow: true });
  if (tabs.length !== 1) throw Error('INVALID_PAGE');
  return (await extractTab(tabs[0])).payload;
}
const location = new ReturnSession(chrome.storage?.session);
let revision: string = crypto.randomUUID();
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
  }, { prepare: source => location.prepare(source, revision), bind: id => location.bind(id, revision), tabID: () => location.tabID() });
let controller = newController();
const family = new FamilyBridge(() => controller, location);
let resetting: Promise<void> | undefined;
const ready = Promise.all([location.ready, family.ready]).then(() => { if (location.revision) revision = location.revision; else if(family.revision) revision = family.revision; });
let error: EntryError | undefined;
const commandName = 'capture-current-product';
const isBusy = () => Boolean(resetting) || controller.isBusy() || family.busy;
function entrySnapshot() {
  return { ...controller.snapshot(), revision, loading: isBusy(), sourceURL: controller.snapshot().sourceURL ?? location.sourceURL ?? null,
    returnState: location.state, error: location.unavailable ? 'RETURN_STATE_UNAVAILABLE' as const : error };
}
async function snapshot() {
  let shortcut: string | null = null;
  try { shortcut = (await chrome.commands.getAll()).find(command => command.name === commandName)?.shortcut ?? ''; }
  catch { /* Unknown assignment is different from an unassigned shortcut. */ }
  return { ...entrySnapshot(), shortcut, ...(location.state!=='available' && family.active ? {family:await family.status()} : {}) };
}
async function report(code?: EntryError) {
  error = code;
  try {
    await chrome.action.setBadgeText({ text: code ? '!' : '' });
    await chrome.action.setTitle({ title: code ? `商品采集：${entryMessages[code]}（打开扩展查看）` : '商品采集 · 采集并在应用确认' });
  } catch { /* A badge failure cannot change a capture/handoff result. */ }
}
async function perform(type: string, tab?: CaptureTab) {
  if (type !== 'popup.status') {
    if (resetting || family.busy) throw Error('ACTION_IN_PROGRESS');
    location.assertReadable();
    family.location.assertReadable();
    if(family.active && location.state!=='available' && type!=='popup.new'){await family.focus();return;}
  }
  switch (type) {
    case 'popup.captureAndHandoff':
      if (location.state === 'available') { await controller.returnToApplication(); break; }
      if (location.state === 'uncertain' && !controller.isBusy()) throw Error('HANDOFF_UNAVAILABLE');
      if (!controller.isBusy() && !controller.snapshot().captured) revision = crypto.randomUUID();
      await controller.captureAndHandoff(tab); break;
    case 'popup.return': await controller.returnToApplication(); break;
    case 'popup.new':
      await reset(); break;
    case 'popup.status': return;
    default: throw Error('INVALID_MESSAGE');
  }
  await report();
}
function reset(captureTab?: CaptureTab, scope?: 'family') {
  if (isBusy()) throw Error('ACTION_IN_PROGRESS');
  location.assertReadable();
  resetting = Promise.resolve().then(async () => {
    if(family.active)await family.abandonInterrupted();
    await location.clear();
    controller = newController(); revision = crypto.randomUUID();
    // Reserve the new entry before releasing reset's async busy boundary.
    if (captureTab) { if(scope==='family')await family.begin(captureTab,revision);else await controller.captureAndHandoff(captureTab); }
  }).finally(() => { resetting = undefined; });
  return resetting;
}
async function pageSnapshot(sourceURL: string): Promise<PageState> {
  const state = entrySnapshot();
  return { revision, loading: state.loading, captured: state.captured, handed: state.handed, sameProduct: state.sourceURL === sourceURL, returnState: state.returnState, ...(location.state!=='available' && family.active ? {family:await family.status()} : {}) };
}

chrome.runtime.onMessage.addListener((message, sender, respond) => {
  if(isFamilyCommand(message)) {
    void (async()=>{await ready;try{respond({ok:true,result:await family.handle(message,sender)});}catch(e){respond({ok:false,code:entryError(e)});}})();
    return true;
  }
  if (pageMessage(message)) {
    const target = pageSender(sender);
    if (!target) return false;
    void (async () => {
      await ready;
      try {
        const check = await chrome.scripting.executeScript({ target: { tabId: target.id, documentIds: [target.documentId] }, world: 'ISOLATED', func: () => document.URL });
        if (check.length !== 1 || check[0].frameId !== 0 || check[0].documentId !== target.documentId
          || capturePageSource(check[0].result as string).sourceURL !== target.sourceURL) throw Error('PAGE_CHANGED');
      } catch { respond({ ok: false, code: 'PAGE_CHANGED', state: await pageSnapshot(target.sourceURL) }); return; }
      try {
        if (message.type !== 'page.status') {
          // Compare and reserve/reset synchronously after document validation.
          if (message.revision !== revision) throw Error('ENTRY_CHANGED');
          if (message.type === 'page.capture' && !family.active && (isBusy() || controller.snapshot().captured || location.state !== 'none')) throw Error('ENTRY_CHANGED');
          if (message.type === 'page.return' && !family.active && location.state !== 'available') throw Error('ORIGINAL_TAB_UNAVAILABLE');
          if (message.type === 'page.newCapture') { await reset(target, message.scope); await report(); }
          else if(family.active && location.state!=='available'){await family.focus();}
          else if(message.type==='page.capture' && message.scope==='family'){await family.begin(target,revision);await report();}
          else await perform(message.type === 'page.return' ? 'popup.return' : 'popup.captureAndHandoff', target);
        }
        const code = entrySnapshot().error;
        respond({ ok: !location.unavailable, ...(code ? { code } : {}), state: await pageSnapshot(target.sourceURL) });
      } catch (failure) {
        const code = entryError(failure); await report(code);
        respond({ ok: false, code, state: await pageSnapshot(target.sourceURL) });
      }
    })();
    return true;
  }
  if (sender.id !== chrome.runtime.id || sender.url !== chrome.runtime.getURL('popup.html') || sender.tab
    || !message || typeof message.type !== 'string'
    || (message.type === 'popup.new' ? Object.keys(message).length !== 2 || typeof message.revision !== 'string' || !UUID.test(message.revision)
      : Object.keys(message).length !== 1)) return false;
  void (async () => {
    await ready;
    try {
      if (message.type === 'popup.new' && message.revision !== revision) throw Error('ENTRY_CHANGED');
      await perform(message.type); respond({ ok: !location.unavailable, state: await snapshot() });
    }
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
    await ready;
    try {
      if (location.state === 'none' && !controller.snapshot().handed && (!tab || tab.id === undefined || !tab.url)) throw Error('INVALID_PAGE');
      await perform('popup.captureAndHandoff', tab ? { id: tab.id, url: tab.url } : undefined);
    } catch (failure) { await report(entryError(failure)); }
  })();
});
chrome.runtime.onMessageExternal.addListener((message, sender, respond) => {
  try { respond(controller.message(message, sender)); }
  catch { respond({ version: 1, code: 'CAPTURE_UNAVAILABLE' }); }
  return false;
});

chrome.tabs.onRemoved?.addListener(id=>{void ready.then(()=>family.closed(id)).catch(()=>report('FAMILY_INTERRUPTED'));});
