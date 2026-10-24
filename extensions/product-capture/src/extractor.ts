import { parseTree, type Node, type ParseError } from 'jsonc-parser';
import type { CapturePayload, EvidenceV2, Price, Image } from './wire';
import { byteLength, fail, MAX_BYTES, pageSource, validateCapture, validPriceAmount } from './validation';
import { detailImages, tableFacts, putFact } from './dom-materials';

// Observed live payloads reach ~16.5k nodes, so the previous 16384 bound rejected
// real pages. The 2MB candidate bound remains the primary resource guard.
const MAX_NODES = 65536;

// EXTRACT: known 1688 productTitle/gallery/Root field semantics only. No legacy
// service, window.context access, profile, network or arbitrary script execution.
// Locates the product document inside 1688's inline scripts. The page assigns
// window.context at a statement boundary to either a JSON object literal or an IIFE
// call whose single object-literal argument carries the document (observed 2026-09).
// Only the located literal text is ever copied; page globals, cookies and unrelated
// scripts are never read. Anything ambiguous is rejected instead of guessed.
function skipSpace(text: string, index: number): number {
  while (index < text.length && /\s/.test(text[index])) index++;
  return index;
}
function balancedEnd(text: string, index: number, open: string, close: string): number {
  let depth = 0; let quote = ''; let escaped = false;
  for (; index < text.length; index++) {
    const char = text[index];
    if (quote) {
      if (escaped) escaped = false;
      else if (char === '\\') escaped = true;
      else if (char === quote) quote = '';
      continue;
    }
    if (char === '"' || char === "'") { quote = char; continue; }
    if (char === open) depth++;
    else if (char === close && --depth === 0) return index + 1;
  }
  return -1;
}
function topLevelItems(text: string): string[] {
  const items: string[] = [];
  let start = 0; let depth = 0; let quote = ''; let escaped = false;
  for (let index = 0; index < text.length; index++) {
    const char = text[index];
    if (quote) {
      if (escaped) escaped = false;
      else if (char === '\\') escaped = true;
      else if (char === quote) quote = '';
      continue;
    }
    if (char === '"' || char === "'") { quote = char; continue; }
    if ('([{'.includes(char)) depth++;
    else if (')]}'.includes(char)) depth--;
    else if (char === ',' && depth === 0) { items.push(text.slice(start, index)); start = index + 1; }
  }
  items.push(text.slice(start));
  return items;
}
// A second object argument would be ambiguous, so the call is rejected instead.
function objectArgument(call: string): string {
  const objects = topLevelItems(call).map(item => item.trim())
    .filter(item => item.startsWith('{') && balancedEnd(item, 0, '{', '}') === item.length);
  if (objects.length !== 1) fail('UNSUPPORTED_PAGE');
  return objects[0];
}
// The live page writes numeric object keys without quotes
// ({"skuWeight":{6290953586037:0.3}}), which strict JSON forbids. Only that key
// position is quoted; string contents and every other token are copied verbatim,
// so the strict parse below still rejects comments, trailing commas, duplicate
// keys and executable text.
function quoteBareNumericKeys(text: string): string {
  let out = '';
  let quote = '';
  let escaped = false;
  const containers: string[] = [];
  let expectKey = false;
  for (let index = 0; index < text.length; index++) {
    const char = text[index];
    if (quote) {
      out += char;
      if (escaped) escaped = false;
      else if (char === '\\') escaped = true;
      else if (char === quote) quote = '';
      continue;
    }
    if (char === '"') { quote = char; expectKey = false; out += char; continue; }
    if (char === '{') { containers.push('object'); expectKey = true; out += char; continue; }
    if (char === '[') { containers.push('array'); expectKey = false; out += char; continue; }
    if (char === '}' || char === ']') { containers.pop(); expectKey = false; out += char; continue; }
    if (char === ':') { expectKey = false; out += char; continue; }
    if (char === ',' && containers[containers.length - 1] === 'object') { expectKey = true; out += char; continue; }
    if (expectKey && char >= '0' && char <= '9') {
      let end = index;
      while (end < text.length && text[end] >= '0' && text[end] <= '9') end++;
      let after = end;
      while (after < text.length && /\s/.test(text[after])) after++;
      if (text[after] === ':') { out += `"${text.slice(index, end)}"`; index = end - 1; continue; }
    }
    out += char;
  }
  return out;
}
function contextPayload(text: string): string | null {
  const assignment = /(?:window\s*\.\s*)?context\s*=\s*/g;
  let payload: string | undefined;
  for (let match = assignment.exec(text); match; match = assignment.exec(text)) {
    const before = match.index > 0 ? text[match.index - 1] : '';
    if (/[A-Za-z0-9_$.]/.test(before)) continue;
    let index = match.index + match[0].length;
    let extracted: string;
    if (text[index] === '{') {
      const end = balancedEnd(text, index, '{', '}');
      if (end < 0) fail('UNSUPPORTED_PAGE');
      extracted = text.slice(index, end); index = end;
    } else if (text[index] === '(') {
      const called = balancedEnd(text, index, '(', ')');
      if (called < 0) fail('UNSUPPORTED_PAGE');
      index = skipSpace(text, called);
      if (text[index] !== '(') fail('UNSUPPORTED_PAGE');
      const args = balancedEnd(text, index, '(', ')');
      if (args < 0) fail('UNSUPPORTED_PAGE');
      extracted = objectArgument(text.slice(index + 1, args - 1)); index = args;
    } else fail('UNSUPPORTED_PAGE');
    index = skipSpace(text, index);
    if (text[index] === ';') index = skipSpace(text, index + 1);
    if (index !== text.length) fail('UNSUPPORTED_PAGE');
    if (payload !== undefined) fail('UNSUPPORTED_PAGE');
    payload = extracted;
  }
  return payload ?? null;
}

export async function captureDocument(doc: Document, source: string, now: Date): Promise<CapturePayload> {
  const started = performance.now();
  const identity = pageSource(source);
  if (pageSource(doc.URL).sourceURL !== identity.sourceURL) fail('PAGE_CHANGED');
  let raw: string | undefined;
  const scripts = doc.querySelectorAll('script:not([src])');
  if (scripts.length > 256) fail('CAPTURE_TOO_LARGE');
  for (const script of scripts) {
    // Only the located literal is copied; other script text is never retained.
    const first = script.firstChild;
    if (!first || first.nodeType !== 3) continue;
    const text = first as Text;
    if (!text.data.includes('context')) continue;
    const candidate = contextPayload(text.data);
    if (candidate === null) continue;
    if (raw !== undefined) fail('UNSUPPORTED_PAGE');
    if (byteLength(candidate) > MAX_BYTES) fail('CAPTURE_TOO_LARGE');
    raw = candidate;
  }
  if (raw === undefined) fail('UNSUPPORTED_PAGE');
  const normalized = quoteBareNumericKeys(raw);
  if (byteLength(normalized) > MAX_BYTES) fail('CAPTURE_TOO_LARGE');
  const errors: ParseError[] = [];
  const root = parseTree(normalized, errors, { disallowComments: true, allowTrailingComma: false });
  if (!root || errors.length) fail('UNSUPPORTED_PAGE');
  let nodes = 0;
  function inspect(node: Node, depth = 0) {
    if (++nodes > MAX_NODES || depth > 32 || performance.now() - started > 8000) fail('CAPTURE_TOO_LARGE');
    if (node.type === 'object') {
      const keys = node.children?.map(n => n.children?.[0].value) ?? [];
      if (new Set(keys).size !== keys.length) fail('UNSUPPORTED_PAGE');
    }
    for (const child of node.children ?? []) inspect(child, depth + 1);
  }
  inspect(root);
  function path(node: Node | undefined, ...keys: string[]): Node | undefined {
    for (const key of keys) {
      if (!node) return undefined;
      if (node.type !== 'object') fail('UNSUPPORTED_PAGE');
      node = node.children?.find(p => p.children?.[0].value === key)?.children?.[1];
    }
    return node;
  }
  function string(node: Node | undefined, numeric = false): string | null {
    if (!node || node.type === 'null') return null;
    if (node.type === 'string') return node.value as string;
    if (numeric && node.type === 'number') return normalized.slice(node.offset, node.offset + node.length);
    return fail('UNSUPPORTED_PAGE');
  }
  function array(node: Node | undefined): Node[] {
    if (!node) return [];
    if (node.type !== 'array' || (node.children?.length ?? 0) > 256) fail('UNSUPPORTED_PAGE');
    return node.children ?? [];
  }
  const data = path(root, 'result', 'data');
  const model = path(data, 'Root', 'fields', 'dataJson');
  if (string(path(model, 'tempModel', 'offerId'), true) !== identity.offerID) fail('PAGE_CHANGED');
  const title = string(path(data, 'productTitle', 'fields', 'title'));
  if (!title?.trim() || /验证码|请登录|访问受限|captcha|access denied/i.test(title)) fail('UNSUPPORTED_PAGE');
  const e: EvidenceV2 = {
    schemaVersion: 2, ...identity, title, description: null, attributes: [], variants: [], priceFacts: [], images: [],
    capturedAt: now.toISOString(), contentSHA256: '0'.repeat(64), parserVersion: '1688-browser-dom/v3', warnings: [], missingFacts: [],
  };
  const missing = (field: string) => {
    e.missingFacts.push({ field, reason: 'not_observed' });
    e.warnings.push({ code: 'MISSING_FACT', field });
  };
  e.attributes = array(path(root, 'result', 'global', 'globalData', 'model', 'offerDetail', 'featureAttributes')).map(n => ({
    name: string(path(n, 'name')) ?? fail('UNSUPPORTED_PAGE'), value: string(path(n, 'value')) ?? fail('UNSUPPORTED_PAGE'),
  }));
  const attributeFacts = new Map<string, {name:string;value:string}>(); const attributeConflicts = new Set<string>();
  const materialCharge = () => { if (++nodes > MAX_NODES) fail('CAPTURE_TOO_LARGE'); if (performance.now()-started>8000) fail('CAPTURE_TIMEOUT'); };
  for (const item of [...e.attributes, ...tableFacts(doc, '#desc-lazyload-container tr', materialCharge)]) putFact(attributeFacts,attributeConflicts,item.name,item.value);
  let unknownWeightUnit = false;
  for (const [component, label] of [['productPackInfo','来源包装重量'],['shippingServices','来源物流重量']]) {
    const weight = string(path(data, component, 'fields', 'unitWeight'), true);
    if (weight === null || !weight.trim()) continue;
    const explicitUnit = /^\s*\d+(?:\.\d+)?\s*(?:kg|g|mg|lbs?|oz|千克|公斤|克|毫克|磅|盎司)\s*$/i.test(weight);
    putFact(attributeFacts,attributeConflicts,explicitUnit ? label : `${label}（单位未提供）`,weight);
    if (!explicitUnit) unknownWeightUnit = true;
  }
  e.attributes = [...attributeFacts.values()];
  if (unknownWeightUnit) {
    e.missingFacts.push({field:'attributes.weight.unit',reason:'source weight unit not explicit; original value retained without conversion'});
    e.warnings.push({code:'MISSING_FACT',field:'attributes.weight.unit'});
  }
  if (attributeConflicts.size) {
    e.missingFacts.push({field:'attributes',reason:'conflicting product attribute values were omitted'});
    e.warnings.push({code:'MISSING_FACT',field:'attributes'});
  }
  e.images = array(path(data, 'gallery', 'fields', 'offerImgList')).map(n => ({ url: string(n) ?? fail('UNSUPPORTED_PAGE'), role: 'source' }));
  for (const url of detailImages(doc, '#desc-lazyload-container img', materialCharge)) {
    if (!e.images.some(image=>image.url===url)) {
      if (e.images.length >= 256) fail('CAPTURE_TOO_LARGE');
      e.images.push({url,role:'detail'});
    }
  }
  const price = (node: Node, field: string, reportMissing = true): Price | null => {
    const amount = string(path(node, 'price'), true);
    if (amount === null || !reportMissing && !validPriceAmount(amount)) { if (reportMissing) missing(field); return null; }
    const currency = string(path(node, 'currency'));
    if (currency === null) missing(`${field}.currency`);
    const minQuantity = string(path(node, 'beginAmount'), true);
    return { amount, currency, ...(minQuantity !== null ? { minQuantity } : {}) };
  };
  // The sourcing owner reads price ranges from every result.data key, not only
  // "price"; the same key set keeps both channels reporting identical facts.
  const dataKeys = (data?.type === 'object' ? data.children?.map(pair => pair.children?.[0].value as string) : undefined) ?? [];
  for (const key of [...dataKeys].sort()) {
    for (const item of array(path(data, key, 'fields', 'finalPriceModel', 'tradeWithoutPromotion', 'offerPriceRanges'))) {
      const value = price(item, `priceFacts[${e.priceFacts.length}]`, false);
      if (value) e.priceFacts.push(value);
    }
  }
  if (!e.priceFacts.length) {
    for (const item of array(path(model, 'orderParamModel', 'orderParam', 'skuParam', 'skuRangePrices'))) {
      const value = price(item, `priceFacts[${e.priceFacts.length}]`, false);
      if (value) e.priceFacts.push(value);
    }
  }
  if (!e.priceFacts.length) {
    for (const key of [...dataKeys].sort()) {
      for (const item of array(path(data, key, 'fields', 'priceModel', 'currentPrices'))) {
        const value = price(item, `priceFacts[${e.priceFacts.length}]`, false);
        if (value) e.priceFacts.push(value);
      }
      if (e.priceFacts.length) break;
    }
  }
  const sku = path(model, 'skuModel');
  const propNodes = array(path(sku, 'skuProps'));
  const props = propNodes.map(n => string(path(n, 'prop')) ?? fail('UNSUPPORTED_PAGE'));
  const propImages = propNodes.map(prop => {
    const values = new Map<string, string | null>();
    for (const value of array(path(prop, 'value'))) {
      const name = string(path(value, 'name')) ?? fail('UNSUPPORTED_PAGE');
      if (values.has(name)) fail('UNSUPPORTED_PAGE');
      values.set(name, string(path(value, 'imageUrl')));
    }
    return values;
  });
  const info = path(sku, 'skuInfoMap');
  if (info && (info.type !== 'object' || (info.children?.length ?? 0) > 256)) fail('UNSUPPORTED_PAGE');
  for (const pair of info?.children ?? []) {
    const key = pair.children![0].value as string; const item = pair.children![1];
    const id = string(path(item, 'skuId'), true);
    // skuId identifies the provider variant; it does not establish a seller SKU.
    missing(`variants[${e.variants.length}].sku`);
    const values = key.split('&gt;');
    if (values.length !== props.length) fail('UNSUPPORTED_PAGE');
    const images: Image[] = [];
    const addImage = (url: string | null) => {
      if (url && !images.some(image => image.url === url)) images.push({ url, role: 'source' });
    };
    // Only an image declared on this exact SKU or this exact dimension value.
    // Product gallery order never establishes a relationship to a variant.
    addImage(string(path(item, 'imageUrl')));
    values.forEach((value, i) => addImage(propImages[i].get(value) ?? null));
    e.variants.push({ sourceID: id, sku: null, title: null,
      attributes: props.map((name, i) => ({ name, value: values[i] })), price: price(item, `variants[${e.variants.length}].price`), images });
    if (!images.length) missing(`variants[${e.variants.length - 1}].images`);
  }
  // Only explicitly identified, already rendered description text. Never frames
  // or external detailUrl fetches, and never whole-body text/HTML.
  const description = doc.querySelector('#desc-lazyload-container');
  if (description) {
    const texts = doc.createTreeWalker(description, 4); const parts: string[] = []; let length = 0;
    let block: Element | null = null;
    for (let n = texts.nextNode(); n; n = texts.nextNode()) {
      if (++nodes > MAX_NODES) fail('CAPTURE_TOO_LARGE');
      const parent = n.parentElement;
      if (!parent || parent.closest('script,style,noscript,template,iframe,input,form,[hidden],[aria-hidden="true"]')) continue;
      let hidden = false;
      for (let p: Element | null = parent; p; p = p.parentElement) {
        const style = (p as HTMLElement).style;
        if (style?.display === 'none' || style?.visibility === 'hidden') { hidden = true; break; }
      }
      if (hidden) continue;
      const value = n.nodeValue ?? '';
      const nextBlock = parent.closest('p,div,li,section,tr');
      const separator = block && block !== nextBlock && parts.length ? '\n' : '';
      length += byteLength(separator) + byteLength(value);
      if (length > 8192) fail('CAPTURE_TOO_LARGE');
      parts.push(separator + value); block = nextBlock;
    }
    e.description = parts.join('').trim() || null;
  }
  for (const field of ['description', 'attributes', 'variants', 'priceFacts', 'images'] as const) {
    if (!e[field] || e[field].length === 0) missing(field);
  }
  const payload = validateCapture({ captureVersion: 2, evidence: e });
  const projection = JSON.stringify({ sourceURL: e.sourceURL, offerID: e.offerID, title: e.title, description: e.description,
    attributes: e.attributes, variants: e.variants, priceFacts: e.priceFacts, images: e.images });
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(projection));
  payload.evidence.contentSHA256 = Array.from(new Uint8Array(digest), b => b.toString(16).padStart(2, '0')).join('');
  if (performance.now() - started > 8000) fail('CAPTURE_TIMEOUT');
  if (pageSource(doc.URL).sourceURL !== identity.sourceURL) fail('PAGE_CHANGED');
  return payload;
}
