import { excludedDetail } from './dom-materials';
import { fail } from './validation';

type DetailRoot = { root: Element | null; reason?: string };
const ambiguous = 'current detail root is ambiguous';
const unready = 'current detail root is not ready';

// Two observed current-page shapes, one selected fact root. No frame/network
// reads, recursive shadow discovery, page mutation or waiting for image bytes.
export function current1688Detail(doc: Document, charge: () => void): DetailRoot {
  const candidates = doc.querySelectorAll('#description[data-module="od_product_description"] .module-od-product-description .html-description');
  if (candidates.length > 256) fail('CAPTURE_TOO_LARGE');
  const hosts = [...candidates].filter(host => { charge(); return !excludedDetail(host); });
  if (hosts.length > 1) return { root: null, reason: ambiguous };
  if (hosts.length === 1) {
    const shadow = hosts[0].shadowRoot;
    if (!shadow || shadow.mode !== 'open') return { root: null, reason: unready };
    const roots = shadow.querySelectorAll('#detail');
    if (roots.length > 256) fail('CAPTURE_TOO_LARGE');
    roots.forEach(() => charge());
    if (roots.length !== 1) return { root: null, reason: roots.length ? ambiguous : unready };
    if (excludedDetail(roots[0])) return { root: null, reason: 'current detail root is excluded' };
    if (!roots[0].hasChildNodes()) return { root: null, reason: unready };
    return { root: roots[0] };
  }
  const original = doc.querySelectorAll('#desc-lazyload-container');
  if (original.length > 256) fail('CAPTURE_TOO_LARGE');
  const roots = [...original].filter(root => { charge(); return !excludedDetail(root); });
  if (roots.length > 1) return { root: null, reason: ambiguous };
  return { root: roots[0] ?? null };
}
