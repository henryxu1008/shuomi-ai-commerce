import { parse, type Node } from 'acorn';
import { byteLength, MAX_BYTES } from '../validation';
import { amazonPageSource } from './validation';
import type { Attribute } from '../wire';

type Syntax = Node & Record<string, unknown>;
export type FamilyMember = { sourceID: string; attributes: Attribute[] };
export type FamilyManifest = { sourceURL: string; parentASIN: string; members: FamilyMember[] };
const asin = /^[A-Z0-9]{10}$/;
const syntax = (v: unknown): v is Syntax => !!v && typeof v === 'object' && 'type' in v && 'start' in v;
const key = (n: Syntax) => n.type === 'Identifier' ? n.name : n.type === 'Literal' ? n.value : undefined;
function member(n: Syntax, name: string): Syntax | undefined {
  const entries = (n.properties as Syntax[]).filter(p => p.type === 'Property' && syntax(p.key) && key(p.key as Syntax) === name);
  if (entries.length > 1 || entries.some(p => p.computed || p.kind !== 'init' || p.method)) throw Error('FAMILY_INVALID');
  return entries[0]?.value as Syntax | undefined;
}
function literal(n: Syntax | undefined, depth = 0): unknown {
  if (!n || depth > 16) throw Error('FAMILY_INVALID');
  if (n.type === 'Literal' && (typeof n.value === 'string' || typeof n.value === 'number' || n.value === null)) return n.value;
  if (n.type === 'ArrayExpression') return (n.elements as Syntax[]).map(v => literal(v, depth + 1));
  if (n.type !== 'ObjectExpression') throw Error('FAMILY_INVALID');
  const out: Record<string, unknown> = Object.create(null);
  for (const p of n.properties as Syntax[]) {
    const name = key(p.key as Syntax);
    if (p.type !== 'Property' || p.computed || p.kind !== 'init' || p.method || typeof name !== 'string' || Object.hasOwn(out, name)) throw Error('FAMILY_INVALID');
    out[name] = literal(p.value as Syntax, depth + 1);
  }
  return out;
}
function object(v: unknown): Record<string, unknown> {
  if (!v || typeof v !== 'object' || Array.isArray(v)) throw Error('FAMILY_INVALID');
  return v as Record<string, unknown>;
}
export function discoverFamily(doc: Document, sourceURL: string): FamilyManifest {
  const source = amazonPageSource(sourceURL);
  if (amazonPageSource(doc.URL).sourceURL !== source.sourceURL || doc.querySelector<HTMLInputElement>('#ASIN')?.value !== source.sourceID) throw Error('SOURCE_CHANGED');
  let nodes = 0, bytes = 0; const started = performance.now(); const found: FamilyManifest[] = [];
  const charge = () => { if (++nodes > 65536 || performance.now() - started > 8000) throw Error('CAPTURE_TOO_LARGE'); };
  function accept(n: Syntax) {
    const current = member(n, 'currentAsin');
    if (!current || current.type !== 'Literal' || current.value !== source.sourceID) return;
    const parent = literal(member(n, 'parentAsin'));
    if (typeof parent !== 'string' || !asin.test(parent) || parent === source.sourceID) throw Error('FAMILY_INVALID');
    const dimensions = literal(member(n, 'dimensions'));
    if (!Array.isArray(dimensions) || !dimensions.length || dimensions.length > 8 || dimensions.some(v => typeof v !== 'string') || new Set(dimensions).size !== dimensions.length) throw Error('FAMILY_INVALID');
    const values = object(literal(member(n, 'variationValues'))), labels = object(literal(member(n, 'variationDisplayLabels')));
    const mapping = object(literal(member(n, 'dimensionToAsinMap')));
    const displayNode = member(n, 'dimensionValuesDisplayData'); const display = displayNode ? object(literal(displayNode)) : undefined;
    const members: FamilyMember[] = [], ids = new Set<string>(), combinations = new Set<string>();
    for (const [indices, id] of Object.entries(mapping)) {
      charge(); if (members.length >= 256 || !/^[0-9]+(?:_[0-9]+)*$/.test(indices) || typeof id !== 'string' || !asin.test(id) || ids.has(id)) throw Error('FAMILY_INVALID');
      const positions = indices.split('_').map(Number);
      if (positions.length !== dimensions.length) throw Error('FAMILY_INVALID');
      const attrs = dimensions.map((d: string, index: number) => {
        const list = values[d], name = labels[d];
        if (!Array.isArray(list) || !Number.isSafeInteger(positions[index]) || typeof name !== 'string' || !name.trim() || name.length > 200) throw Error('FAMILY_INVALID');
        const value = list[positions[index]];
        if (typeof value !== 'string' || !value.trim() || value.length > 2000) throw Error('FAMILY_INVALID');
        return { name: name.trim(), value };
      });
      if (new Set(attrs.map(a => a.name)).size !== attrs.length) throw Error('FAMILY_INVALID');
      if (display && JSON.stringify(display[id]) !== JSON.stringify(attrs.map(a => a.value))) throw Error('FAMILY_INVALID');
      const combination = JSON.stringify(attrs); if (combinations.has(combination)) throw Error('FAMILY_INVALID');
      ids.add(id); combinations.add(combination); members.push({ sourceID: id, attributes: attrs });
    }
    if (!ids.has(source.sourceID) || (display && Object.keys(display).some(id => !ids.has(id)))) throw Error('FAMILY_INVALID');
    found.push({ sourceURL: source.sourceURL, parentASIN: parent, members: [members.find(m => m.sourceID === source.sourceID)!, ...members.filter(m => m.sourceID !== source.sourceID)] });
  }
  function walk(n: Syntax, inTwister: boolean, depth: number) {
    charge(); if (depth > 64) throw Error('CAPTURE_TOO_LARGE');
    let bound = inTwister;
    if (n.type === 'CallExpression') {
      const callee = n.callee as Syntax, args = n.arguments as Syntax[];
      bound = false;
      if (callee.type === 'MemberExpression' && !callee.computed && key(callee.property as Syntax) === 'register'
        && (callee.object as Syntax).type === 'Identifier' && (callee.object as Syntax).name === 'P'
        && args[0]?.type === 'Literal' && args[0].value === 'twister-js-init-dpx-data') bound = true;
    }
    if (bound && n.type === 'ObjectExpression') accept(n);
    for (const value of Object.values(n)) {
      if (syntax(value)) walk(value, bound, depth + 1);
      else if (Array.isArray(value)) for (const child of value) if (syntax(child)) walk(child, bound, depth + 1);
    }
  }
  for (const script of doc.scripts) {
    const text = script.textContent ?? '';
    if (!text.includes('twister-js-init-dpx-data') || !text.includes('dimensionToAsinMap')) continue;
    bytes += byteLength(text); if (bytes > MAX_BYTES) throw Error('CAPTURE_TOO_LARGE');
    let tree: Syntax; try { tree = parse(text, { ecmaVersion: 'latest' }) as unknown as Syntax; } catch { throw Error('FAMILY_INVALID'); }
    walk(tree, false, 0);
  }
  if (!found.length) throw Error('FAMILY_NOT_AVAILABLE');
  if (found.some(m => JSON.stringify(m) !== JSON.stringify(found[0]))) throw Error('FAMILY_INVALID');
  return found[0];
}
