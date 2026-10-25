import { excluded, productText, tableFacts } from '../dom-materials';
import { byteLength, fail } from '../validation';
import type { Attribute } from '../wire';

const detailsRoots = '#detailBulletsWrapper_feature_div,#detailBullets_feature_div';
const controls = 'button,input,textarea,select,[role="button"]';
const metric = /^(?:best sellers rank|customer reviews|customer ratings|rating|ratings|review count)$/i;
const visibleText = (value: string) => value.replace(/[\u200e\u200f\u202a-\u202e\u2066-\u2069]/g, '').replace(/\s+/g, ' ').trim();

function containsTitlePhrase(title: string, value: string): boolean {
  const literal = value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const left = /^[\p{L}\p{N}]/u.test(value) ? '(^|[^\\p{L}\\p{N}])' : '';
  const right = /[\p{L}\p{N}]$/u.test(value) ? '(?=$|[^\\p{L}\\p{N}])' : '';
  return new RegExp(`${left}${literal}${right}`, 'u').test(title);
}

function excludedTitle(node: Element) {
  return excluded(node) || !!node.closest(`${controls},a,[role="link"],#bylineInfo,#averageCustomerReviews,#acrCustomerReviewText,.a-offscreen,.aok-offscreen`);
}

// Only the current product's bound title section can supply a subtitle. This
// does not inspect the document title, image alt text, metadata or other ASINs.
export function amazonVisibleTitle(doc: Document, primary: Element, title: string, charge: () => void): { title: string; reason?: string } {
  const roots = doc.querySelectorAll('#titleSection');
  if (roots.length > 256) fail('CAPTURE_TOO_LARGE');
  if (!roots.length) return { title };
  const ambiguous = { title, reason: 'current-product subtitle region or value is ambiguous' };
  if (roots.length !== 1 || !roots[0].contains(primary)) return ambiguous;
  const root = roots[0];
  if (excludedTitle(root)) return { title };
  const heading = primary.closest('h1');
  const primaryBranch = heading && root.contains(heading) ? heading : primary;
  const pending: Node[] = Array.from(root.childNodes).reverse();
  if (pending.length > 256) fail('CAPTURE_TOO_LARGE');
  const candidates = new Set<string>();
  while (pending.length) {
    charge();
    const node = pending.pop()!;
    if (node === primaryBranch) continue;
    let raw: string | null = null;
    if (node.nodeType === 1) {
      const element = node as Element;
      if (excludedTitle(element)) continue;
      if (element.contains(primary)) {
        if (pending.length + element.childNodes.length > 256) fail('CAPTURE_TOO_LARGE');
        pending.push(...Array.from(element.childNodes).reverse());
        continue;
      }
      raw = productText(doc, element, charge, 2000, false, excludedTitle);
    } else if (node.nodeType === 3) {
      raw = node.nodeValue;
      if (raw && byteLength(raw) > 2000) fail('CAPTURE_TOO_LARGE');
    }
    const value = raw && visibleText(raw);
    if (!value || containsTitlePhrase(visibleText(title), value)) continue;
    candidates.add(value);
    if (candidates.size > 256) fail('CAPTURE_TOO_LARGE');
  }
  if (!candidates.size) return { title };
  if (candidates.size !== 1) return ambiguous;
  const complete = `${title} | ${[...candidates][0]}`;
  if (byteLength(complete) > 2000) fail('CAPTURE_TOO_LARGE');
  return { title: complete };
}

// Keep the source region in the attribute name: the overview dimensions and a
// details row containing dimensions plus weight are distinct original facts.
export function amazonListFacts(doc: Document, charge: () => void): Attribute[] {
  const rows = doc.querySelectorAll('#detailBulletsWrapper_feature_div li,#detailBullets_feature_div li');
  if (rows.length > 256) fail('CAPTURE_TOO_LARGE');
  const facts: Attribute[] = [];
  for (const row of rows) {
    charge();
    if (excluded(row) || row.parentElement?.closest('li')?.closest(detailsRoots)) continue;
    if (row.querySelector('#detailBullets_averageCustomerReviews,#acrCustomerReviewLink,a[href*="/gp/bestsellers/"]')) continue;
    const reject = (node: Element) => {
      const nested = node.closest('ul,ol');
      return excluded(node) || !!node.closest(controls) || !!(nested && row.contains(nested));
    };
    const raw = productText(doc, row, charge, 8192, false, reject);
    if (!raw) continue;
    const text = visibleText(raw);
    const separator = text.search(/[:：]/);
    if (separator < 1) continue;
    const name = text.slice(0, separator).trim();
    const value = text.slice(separator + 1).trim();
    if (!name || !value || metric.test(name)) continue;
    const scopedName = `Product details · ${name}`;
    if (byteLength(scopedName) > 8192) fail('CAPTURE_TOO_LARGE');
    facts.push({ name: scopedName, value });
  }
  return facts;
}

// Public product facts can be present in a closed UI expander. Bind this exact
// region to its own ASIN before admitting the narrowly scoped display collapse.
export function amazonTableFacts(doc: Document, sourceID: string, charge: () => void): { facts: Attribute[]; reason?: string } {
  const roots = doc.querySelectorAll('#productDetails_expanderSectionTables');
  if (roots.length > 256) fail('CAPTURE_TOO_LARGE');
  if (!roots.length) return { facts: [] };
  const omitted = { facts: [], reason: 'current-product details region, expansion structure or ASIN is not established' };
  const root = roots[0];
  if (roots.length !== 1 || !root.closest('#productDetails_feature_div #prodDetails') || excluded(root)) return omitted;
  const sections = '#productDetails_expanderTables_depthLeftSections,#productDetails_expanderTables_depthRightSections';
  for (const selector of sections.split(',')) if (root.querySelectorAll(selector).length > 1) return omitted;
  const containers = root.querySelectorAll('.a-expander-container');
  if (containers.length > 256) fail('CAPTURE_TOO_LARGE');
  const contents = new Map<Element, Element | undefined>();
  const tables = new Set<Element>();
  for (const container of containers) {
    charge();
    const section = container.closest(sections);
    if (!section || !root.contains(section)) continue;
    const headers = container.querySelectorAll(':scope > .a-declarative > a.a-expander-header[role="button"][data-action="a-expander-toggle"][aria-expanded]');
    const bodies = container.querySelectorAll(':scope > .a-expander-content.a-expander-extend-content');
    if (headers.length !== 1 || bodies.length !== 1 || excluded(headers[0])) continue;
    const expanded = headers[0].getAttribute('aria-expanded');
    if (expanded !== 'true' && expanded !== 'false') continue;
    const content = bodies[0];
    const collapsed = expanded === 'false' && (content as HTMLElement).style.display === 'none' ? content : undefined;
    if (excluded(content, undefined, collapsed)) continue;
    contents.set(content, collapsed);
    const candidates = content.querySelectorAll('table.a-keyvalue.prodDetTable');
    if (candidates.length > 256) fail('CAPTURE_TOO_LARGE');
    for (const table of candidates) {
      charge();
      const outerTable = table.parentElement?.closest('table');
      if (!outerTable || !content.contains(outerTable)) tables.add(table);
    }
  }
  const reject = (node: Element) => {
    const content = node.closest('.a-expander-content');
    const table = node.closest('table');
    return !content || !contents.has(content) || !table || !tables.has(table)
      || node.closest('.a-expander-container') !== content.parentElement
      || excluded(node, undefined, contents.get(content)) || !!node.closest(controls);
  };
  const rows = tableFacts(root, 'table.a-keyvalue.prodDetTable tr', charge, reject);
  const asins = new Set(rows.filter(row => visibleText(row.name).toLowerCase() === 'asin').map(row => visibleText(row.value)));
  if (asins.size !== 1 || !asins.has(sourceID)) return omitted;
  const facts: Attribute[] = [];
  for (const row of rows) {
    charge(); const name = visibleText(row.name); const value = visibleText(row.value);
    if (!name || !value || metric.test(name)) continue;
    const scopedName = `Product details · ${name}`;
    if (byteLength(scopedName) > 8192) fail('CAPTURE_TOO_LARGE');
    facts.push({ name: scopedName, value });
  }
  return { facts };
}
