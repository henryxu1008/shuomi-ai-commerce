import { byteLength, fail, publicImage } from './validation';
import type { Attribute } from './wire';

// Shared existing DOM admission rules; no HTML, page globals or external reads.
export function excluded(element: Element, formTextRoot?: Element): boolean {
  if (element.closest('script,style,noscript,template,iframe,input,[hidden],[aria-hidden="true"]')) return true;
  const form = element.closest('form');
  // Only a located field may ignore its enclosing form, never a nested form.
  if (form && (!formTextRoot || form === formTextRoot || !form.contains(formTextRoot) || !formTextRoot.contains(element))) return true;
  if (formTextRoot && element.closest('textarea,select,button')) return true;
  if (element.closest('#recommendations,.recommendations,.aplus-comparison-table,[data-cel-widget*="recommend" i],[data-cel-widget*="comparison" i]')) return true;
  const link = element.closest('a[href]');
  if (link) {
    try {
      const current = new URL(element.ownerDocument.URL);
      const target = new URL(link.getAttribute('href')!, current);
      const product = (url: URL) => /\/dp\/([A-Z0-9]{10})(?:\/|$)/.exec(url.pathname)?.[1]
        ?? /^\/offer\/([1-9][0-9]*)\.html$/.exec(url.pathname)?.[1];
      const ownID = product(current); const linkedID = product(target);
      if (ownID && linkedID && (ownID !== linkedID || current.hostname.replace(/^www\./,'') !== target.hostname.replace(/^www\./,''))) return true;
    } catch { /* An unreadable link does not establish another product. */ }
  }
  for (let parent: Element | null = element; parent; parent = parent.parentElement) {
    const style = (parent as HTMLElement).style;
    if (style?.display === 'none' || style?.visibility === 'hidden') return true;
  }
  return false;
}

export function productText(doc: Document, element: Element | null, charge: () => void, max = 8192, allowAncestorForm = false): string | null {
  const formTextRoot = allowAncestorForm && element ? element : undefined;
  if (!element || excluded(element, formTextRoot)) return null;
  let result = ''; let bytes = 0;
  const walker = doc.createTreeWalker(element, 4); let node: Node | null;
  while ((node = walker.nextNode())) {
    charge();
    if (node.parentElement && excluded(node.parentElement, formTextRoot)) continue;
    const value = node.nodeValue ?? '';
    if (value.length > max) fail('CAPTURE_TOO_LARGE');
    bytes += byteLength(value);
    if (bytes > max) fail('CAPTURE_TOO_LARGE');
    result += value;
  }
  return result.trim() || null;
}

export function tableFacts(doc: Document, selector: string, charge: () => void): Attribute[] {
  const rows = doc.querySelectorAll(selector);
  if (rows.length > 256) fail('CAPTURE_TOO_LARGE');
  const facts: Attribute[] = [];
  for (const row of rows) {
    charge(); if (excluded(row)) continue;
    const cells = row.querySelectorAll('th,td');
    if (cells.length !== 2) continue;
    const name = productText(doc, cells[0], charge); const value = productText(doc, cells[1], charge);
    if (name && value) facts.push({ name, value });
  }
  return facts;
}

export function putFact(facts: Map<string, Attribute>, conflicts: Set<string>, name: string | null, value: string | null) {
  name = name?.replace(/\s+/g, ' ').replace(/[:：]$/, '').trim() || null;
  value = value?.replace(/\s+/g, ' ').trim() || null;
  if (!name || !value) return;
  const key = name.toLowerCase();
  if (conflicts.has(key)) return;
  if (facts.has(key) && facts.get(key)!.value !== value) { facts.delete(key); conflicts.add(key); }
  else if (!facts.has(key)) facts.set(key, { name, value });
  if (facts.size + conflicts.size > 256) fail('CAPTURE_TOO_LARGE');
}

export function detailImages(doc: Document, selector: string, charge: () => void): string[] {
  const images = doc.querySelectorAll<HTMLImageElement>(selector);
  if (images.length > 256) fail('CAPTURE_TOO_LARGE');
  const urls = new Set<string>();
  for (const image of images) {
    charge();
    if (excluded(image) || image.closest('.videoThumbnail,.video-block,[data-video-url],[data-video-id]')) continue;
    const url = image.getAttribute('data-old-hires') || image.getAttribute('src');
    if (url && byteLength(url) > 2048) fail('CAPTURE_TOO_LARGE');
    if (url && publicImage(url)) urls.add(url);
  }
  return [...urls];
}
