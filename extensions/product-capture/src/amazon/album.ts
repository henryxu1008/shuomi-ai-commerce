import { parse, type Node as SyntaxNode } from 'acorn';
import { parseTree, type Node as JSONNode, type ParseError } from 'jsonc-parser';
import { byteLength, fail, MAX_BYTES, publicImage } from '../validation';

type Syntax = SyntaxNode & Record<string, unknown>;
const syntax = (value: unknown): value is Syntax => !!value && typeof value === 'object' && 'type' in value && 'start' in value;
const key = (node: Syntax) => node.type === 'Identifier' ? node.name : node.type === 'Literal' ? node.value : undefined;
const properties = (node: Syntax, name: string): Syntax[] => node.type === 'ObjectExpression'
  ? (node.properties as Syntax[]).filter(p => p.type === 'Property' && !p.computed && p.kind === 'init' && key(p.key as Syntax) === name) : [];
const child = (node: JSONNode, name: string) => node.children?.find(p => p.children?.[0].value === name)?.children?.[1];
type ImageRecord = { thumb?: string; hiRes?: string; large?: string; variant?: string };

function records(text: string, value: Syntax, charge: () => void): ImageRecord[] | null {
  let raw: string;
  if (value.type === 'ArrayExpression') raw = text.slice(value.start, value.end);
  else if (value.type === 'CallExpression') {
    const call = value.callee as Syntax; const owner = call.object as Syntax | undefined;
    const args = value.arguments as Syntax[];
    if (call.type !== 'MemberExpression' || call.computed || key(call.property as Syntax) !== 'parseJSON'
      || owner?.type !== 'MemberExpression' || owner.computed || key(owner.property as Syntax) !== '$'
      || (owner.object as Syntax)?.type !== 'Identifier' || (owner.object as Syntax).name !== 'A'
      || args.length !== 1 || args[0].type !== 'Literal' || typeof args[0].value !== 'string') return null;
    raw = args[0].value;
  } else return null;
  if (byteLength(raw) > MAX_BYTES) fail('CAPTURE_TOO_LARGE');
  const errors: ParseError[] = [];
  const root = parseTree(raw, errors, { disallowComments: true, allowTrailingComma: false });
  if (!root || errors.length || root.type !== 'array') return null;
  const inspect = (node: JSONNode, depth: number): boolean => {
    charge(); if (depth > 32) fail('CAPTURE_TOO_LARGE');
    if (node.type === 'object') {
      const keys = node.children?.map(p => p.children?.[0].value) ?? [];
      if (new Set(keys).size !== keys.length) return false;
    }
    return (node.children ?? []).every(n => inspect(n, depth + 1));
  };
  if (!inspect(root, 0)) return null;
  if ((root.children?.length ?? 0) > 256) fail('CAPTURE_TOO_LARGE');
  const out: ImageRecord[] = [];
  for (const node of root.children ?? []) {
    if (node.type !== 'object') return null;
    const record: ImageRecord = {};
    for (const field of ['thumb', 'hiRes', 'large', 'variant'] as const) {
      const member = child(node, field);
      if (member && member.type !== 'null') {
        if (member.type !== 'string') return null;
        if (byteLength(member.value as string) > 2048) fail('CAPTURE_TOO_LARGE');
        record[field] = member.value as string;
      }
    }
    out.push(record);
  }
  return out;
}

// EXTRACT: explicit hiRes -> original large, bound to this observed MAIN/thumb
// set. Static parsing never invokes A.$.parseJSON or evaluates the source page.
export function albumURLs(doc: Document, mainURL: string | undefined, thumbs: string[], charge: () => void) {
  const urls = new Map<string, string>(); let reason: string | undefined;
  if (!thumbs.length) return { urls, reason };
  reason = 'current-product high-resolution metadata not observed';
  const scripts = doc.querySelectorAll('script:not([src])');
  if (scripts.length > 256) fail('CAPTURE_TOO_LARGE');
  let bytes = 0; let matched = false; let conflict = false;
  for (const script of scripts) {
    charge();
    const first = script.firstChild;
    if (!first || first.nodeType !== 3) continue;
    const text = (first as Text).data;
    if (!text.includes('colorImages')) continue;
    bytes += byteLength(text); if (bytes > MAX_BYTES) fail('CAPTURE_TOO_LARGE');
    let root: Syntax;
    try {
      const parsed: unknown = parse(text, { ecmaVersion: 2024 });
      if (!syntax(parsed)) continue;
      root = parsed;
    } catch (error) { if (error instanceof RangeError) fail('CAPTURE_TOO_LARGE'); continue; }
    const pending: { node: Syntax; depth: number }[] = [{ node: root, depth: 0 }];
    while (pending.length) {
      const { node, depth } = pending.pop()!;
      charge(); if (depth > 32) fail('CAPTURE_TOO_LARGE');
      for (const property of properties(node, 'colorImages')) {
        const initial = properties(property.value as Syntax, 'initial');
        if (initial.length !== 1) continue;
        const images = records(text, initial[0].value as Syntax, charge);
        if (!mainURL || !images?.some(i => i.variant === 'MAIN' && (i.hiRes === mainURL || i.large === mainURL))) continue;
        matched = true;
        for (const image of images) {
          if (!image.thumb || !thumbs.includes(image.thumb)) continue;
          const url = image.hiRes && publicImage(image.hiRes) ? image.hiRes : image.large && publicImage(image.large) ? image.large : undefined;
          if (!url) continue;
          if (urls.has(image.thumb) && urls.get(image.thumb) !== url) conflict = true;
          urls.set(image.thumb, url);
          if (url !== image.hiRes) reason = 'some album records provide large but no admissible hiRes';
        }
      }
      for (const value of Object.values(node)) {
        for (const item of Array.isArray(value) ? value : [value]) if (syntax(item)) pending.push({ node: item, depth: depth + 1 });
      }
    }
  }
  if (conflict) return { urls: new Map<string, string>(), reason: 'conflicting current-product image metadata' };
  if (matched && thumbs.every(t => urls.has(t)) && reason === 'current-product high-resolution metadata not observed') reason = undefined;
  return { urls, reason };
}
