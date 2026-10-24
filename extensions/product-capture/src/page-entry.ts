import { entryMessages, type EntryError } from './entry-errors';
import { isPageState, pageSource, type PageMessage, type PageState } from './page-messages';

export function mountPageEntry(doc: Document, view: Window) {
  const hostID = 'shuomi-product-capture';
  if (view.top !== view || doc.getElementById(hostID)) return;
  try { pageSource(doc.URL); } catch { return; }
  const host = doc.createElement('aside'); host.id = hostID;
  host.setAttribute('aria-label', '硕米商品采集');
  const shadow = host.attachShadow({ mode: 'open' });
  // Static markup only. Product evidence, identities and transport keys never enter this DOM.
  shadow.innerHTML = `<style>
    :host{all:initial;position:fixed;right:16px;bottom:20px;z-index:2147483647;display:block;width:min(256px,calc(100vw - 32px));font:14px/1.5 system-ui,sans-serif;color:#203733}
    *{box-sizing:border-box}section{background:#fff;border:1px solid #9bb3ac;border-radius:12px;padding:12px;box-shadow:0 4px 20px #0002}
    strong{display:block;font-size:13px}p{font-size:12px;margin:8px 0;overflow-wrap:anywhere}button{font:inherit;cursor:pointer;border:1px solid #78988e;border-radius:6px;padding:9px 10px;background:#fff;color:#284c40;min-height:40px;width:100%;margin-top:6px}
    #primary,#confirm{background:#315e50;color:#fff;border-color:#315e50}button:focus-visible{outline:3px solid #036ed2;outline-offset:2px}button:disabled{cursor:wait;opacity:.65}[hidden]{display:none!important}#refresh{font-size:12px;min-height:32px;padding:4px}#confirmation{border-top:1px solid #ccd8d3;margin-top:8px;padding-top:4px}
  </style><section><strong>硕米 · 商品采集</strong><p id="status" role="status" aria-live="polite">正在读取插件状态…</p>
  <button id="primary" type="button" disabled>采集并在应用确认</button><button id="fresh" type="button" hidden>采集当前商品（新操作）</button>
  <div id="confirmation" hidden><p>开始新的采集？此前可能提交的操作请先在原应用页核实，原记录和页面会保留。</p><button id="confirm" type="button">确认采集当前商品</button><button id="cancel" type="button">取消</button></div>
  <button id="refresh" type="button">刷新状态</button></section>`;
  const element = <T extends HTMLElement>(id: string) => shadow.getElementById(id) as T;
  const primary = element<HTMLButtonElement>('primary'), fresh = element<HTMLButtonElement>('fresh');
  const confirm = element<HTMLButtonElement>('confirm'), cancel = element<HTMLButtonElement>('cancel');
  const confirmation = element('confirmation'), status = element('status'), refreshButton = element<HTMLButtonElement>('refresh');
  let state: PageState | undefined, pending = false, disconnected = false, lostKnownOperation = false;
  let confirmationRevision: string | undefined, confirmationSource: string | undefined;
  function source() {
    try { return pageSource(doc.URL).sourceURL; }
    catch { host.remove(); return; }
  }
  function closeConfirmation() { confirmation.hidden = true; confirmationRevision = undefined; confirmationSource = undefined; }
  function render(code?: EntryError) {
    primary.textContent = state?.handed ? state.sameProduct ? '返回应用核实' : '返回已有采集' : '采集并在应用确认';
    const busy = pending || Boolean(state?.loading);
    primary.disabled = busy || disconnected || !state || lostKnownOperation;
    fresh.hidden = !state?.captured && !lostKnownOperation;
    fresh.disabled = busy || disconnected; confirm.disabled = busy || disconnected; cancel.disabled = busy;
    refreshButton.disabled = pending;
    status.textContent = disconnected ? '插件连接中断，请先在原应用页核实；可刷新状态，不要重复保存。'
      : busy ? '正在处理当前采集…'
        : code ? entryMessages[code]
          : lostKnownOperation ? '插件状态已变化，请先在原应用页核实此前操作。新采集需要再次确认。'
            : state?.handed ? state.sameProduct ? '已交给应用，保存与结果请在原应用页确认。' : '已有另一件商品的采集，请先返回应用核实，或明确开始新操作。'
              : '点击采集后到应用确认保存。如曾交给应用，请先回原标签核实。';
  }
  async function action(type: PageMessage['type'], expectedRevision?: string) {
    const requestSource = source();
    if (pending || !requestSource) return;
    pending = true; render();
    let code: EntryError | undefined;
    try {
      const message: PageMessage = type === 'page.status' ? { type } : { type, revision: expectedRevision! };
      const response = await chrome.runtime.sendMessage(message);
      if (!source()) return;
      if (source() !== requestSource) {
        lostKnownOperation = lostKnownOperation || Boolean(state?.captured) || type !== 'page.status';
        state = undefined; closeConfirmation(); code = 'PAGE_CHANGED'; return;
      }
      if (!response || typeof response.ok !== 'boolean' || !isPageState(response.state)
        || (!response.ok && (typeof response.code !== 'string' || !Object.hasOwn(entryMessages, response.code)))) throw Error('UNAVAILABLE');
      const next = response.state;
      if (state?.revision !== next.revision) {
        if (state?.captured && !next.captured) lostKnownOperation = true;
        closeConfirmation();
      }
      state = next; disconnected = false;
      if (next.captured) lostKnownOperation = false;
      if (typeof response.code === 'string' && Object.hasOwn(entryMessages, response.code)) code = response.code as EntryError;
    } catch { disconnected = true; closeConfirmation(); }
    finally { pending = false; render(code); }
  }
  function onClick(button: HTMLButtonElement, callback: () => void) {
    button.addEventListener('click', event => {
      if (!event.isTrusted || button.disabled || !source()) return;
      event.preventDefault(); event.stopPropagation(); callback();
    });
  }
  onClick(primary, () => { if (state) void action(state.handed ? 'page.return' : 'page.capture', state.revision); });
  onClick(fresh, () => {
    if (!state) return;
    confirmationRevision = state.revision; confirmationSource = source(); confirmation.hidden = false; confirm.focus();
  });
  onClick(cancel, () => { closeConfirmation(); fresh.focus(); });
  onClick(confirm, () => {
    if (confirmation.hidden || !confirmationRevision) return;
    const expected = confirmationRevision, sameSource = confirmationSource === source(); closeConfirmation();
    if (!sameSource) { void action('page.status'); return; }
    void action('page.newCapture', expected);
  });
  onClick(refreshButton, () => { void action('page.status'); });
  const refresh = () => action('page.status');
  for (const event of ['focus', 'pageshow', 'popstate', 'hashchange']) view.addEventListener(event, () => { void refresh(); });
  doc.documentElement.append(host); void refresh();
  return { host, refresh };
}
