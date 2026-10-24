import { capturePageSource } from './platforms';

const sourceHosts = new Set(['detail.1688.com', 'www.amazon.com', 'www.amazon.ca', 'www.amazon.co.uk', 'www.amazon.de', 'www.amazon.co.jp']);
const revisionPattern = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/;
export type PageMessage = { type: 'page.status' } | { type: 'page.capture' | 'page.return' | 'page.newCapture'; revision: string };
export type PageState = { revision: string; loading: boolean; captured: boolean; handed: boolean; sameProduct: boolean };
export function pageSource(input: string) {
  const url = new URL(input);
  if (url.protocol !== 'https:' || url.port || !sourceHosts.has(url.hostname)) throw Error('INVALID_PAGE');
  return capturePageSource(input);
}
export function pageMessage(value: unknown): value is PageMessage {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false;
  const message = value as Record<string, unknown>;
  if (message.type === 'page.status') return Object.keys(message).length === 1;
  return ['page.capture', 'page.return', 'page.newCapture'].includes(message.type as string)
    && Object.keys(message).length === 2 && typeof message.revision === 'string' && revisionPattern.test(message.revision);
}
export function pageSender(sender: chrome.runtime.MessageSender) {
  if (sender.id !== chrome.runtime.id || sender.frameId !== 0 || sender.documentLifecycle !== 'active'
    || !sender.documentId || !Number.isInteger(sender.tab?.id) || sender.tab!.id! < 0 || !sender.url || !sender.tab?.url) return;
  try {
    const source = pageSource(sender.url);
    if (sender.origin !== new URL(sender.url).origin || pageSource(sender.tab.url).sourceURL !== source.sourceURL) return;
    return { id: sender.tab.id!, url: sender.url, documentId: sender.documentId, sourceURL: source.sourceURL };
  } catch { return; }
}
export function isPageState(value: unknown): value is PageState {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false;
  const state = value as Record<string, unknown>;
  return Object.keys(state).length === 5 && typeof state.revision === 'string' && revisionPattern.test(state.revision)
    && ['loading', 'captured', 'handed', 'sameProduct'].every(key => typeof state[key] === 'boolean');
}
