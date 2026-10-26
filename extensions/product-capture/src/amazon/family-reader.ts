import { amazonPageSource } from './validation';
import { amazonInterruptionURL } from './page-state';

export function sameAmazonProduct(input: string | undefined, target: string): boolean {
  try { return !!input && amazonPageSource(input).sourceURL === target; } catch { return false; }
}
export function sameSiteInterruption(input: string | undefined, target: string): boolean {
  try { return !!input && new URL(input).origin === new URL(target).origin && !!amazonInterruptionURL(input); } catch { return false; }
}

// Tab completion only describes loading. In particular, an old about:blank
// complete event must not authorize parsing the new child document.
export async function waitForFamilyDocument(id: number, target: string, signal: AbortSignal, navigate: boolean, firstLoad = false) {
  signal.throwIfAborted();
  const before = await chrome.tabs.get(id); signal.throwIfAborted();
  await new Promise<void>((resolve, reject) => {
    let settled = false;
    const timer = setTimeout(() => done(Error('CAPTURE_TIMEOUT')), 30000);
    const updated = (tabId: number) => { if (tabId === id) void inspect(); };
    const removed = (tabId: number) => { if (tabId === id) done(Error('FAMILY_INTERRUPTED')); };
    const aborted = () => done(Error('CAPTURE_CANCELLED'));
    function done(error?: Error) {
      if (settled) return; settled = true; clearTimeout(timer);
      signal.removeEventListener('abort', aborted);
      chrome.tabs.onUpdated.removeListener(updated); chrome.tabs.onRemoved.removeListener(removed);
      if (error) reject(error); else resolve();
    }
    async function inspect() {
      try {
        const tab = await chrome.tabs.get(id);
        if (settled) return;
        if (tab.openerTabId !== before.openerTabId || tab.windowId !== before.windowId) return done(Error('FAMILY_INTERRUPTED'));
        if (tab.pendingUrl && tab.pendingUrl !== tab.url && !sameAmazonProduct(tab.pendingUrl, target) && !sameSiteInterruption(tab.pendingUrl, target)) return done(Error('PAGE_CHANGED'));
        if (tab.status !== 'complete' || tab.pendingUrl && tab.pendingUrl !== tab.url) return;
        if (sameAmazonProduct(tab.url, target) || sameSiteInterruption(tab.url, target)) return done();
        if (firstLoad && (!tab.url || tab.url === 'about:blank')) return;
        if (navigate && (tab.url === 'about:blank' || tab.url === before.url)) return;
        done(Error('PAGE_CHANGED'));
      } catch { done(Error('FAMILY_INTERRUPTED')); }
    }
    signal.addEventListener('abort', aborted, { once: true });
    chrome.tabs.onUpdated.addListener(updated); chrome.tabs.onRemoved.addListener(removed);
    if (signal.aborted) return aborted();
    if (navigate) void chrome.tabs.update(id, { url: target, active: false }).then(inspect, () => done(Error('FAMILY_INTERRUPTED')));
    else void inspect();
  });
  signal.throwIfAborted();
}
