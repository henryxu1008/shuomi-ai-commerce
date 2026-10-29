import type { FamilyManifest } from './family-discovery';
import type { AmazonCapturePayload } from './wire';
import { validateAmazonCapture } from './validation';

export const DIMENSION_CONFLICT = 'conflicting selected dimension values were omitted';
export const DIMENSION_UNOBSERVED = 'selected dimension label or value not observed';
export const DIMENSION_UNBOUND = 'real selected dimension value has no observed label';
export const RENDERED_DIMENSION_FIELD = 'variants[0].attributes.rendered';
const staticReason = 'rendered dimension selection not observed; values supplied by current-ASIN static data';

export function placeholderDimension(value: string) {
  return ['select color','select size','please select'].includes(value.trim().replace(/\s+/g,' ').toLowerCase());
}

// manifest must come from familyInfo in this capture's native document. The
// original family list is deliberately not an input to this composition.
export async function completeFamilyDimensions(capture: AmazonCapturePayload, manifest?: FamilyManifest) {
  const checked = validateAmazonCapture(capture), e = checked.evidence;
  if (e.missingFacts.some(f => f.field === 'variants[0].attributes' && (f.reason === DIMENSION_CONFLICT || f.reason === DIMENSION_UNBOUND))) throw Error('SOURCE_CHANGED');
  if (!manifest) return checked;
  if (manifest.sourceURL !== e.sourceURL || !/^[A-Z0-9]{10}$/.test(manifest.parentASIN) || manifest.parentASIN === e.sourceID
    || e.variants.length !== 1 || e.variants[0].sourceID !== e.sourceID) throw Error('SOURCE_CHANGED');
  const matches = manifest.members.filter(m => m.sourceID === e.sourceID);
  if (matches.length !== 1 || !matches[0].attributes.length) throw Error('SOURCE_CHANGED');
  const expected = matches[0].attributes, values = new Map(expected.map(a => [a.name,a.value]));
  if (values.size !== expected.length || expected.some(a => !a.name.trim() || !a.value.trim() || placeholderDimension(a.value))
    || e.variants[0].attributes.some(a => values.get(a.name) !== a.value)) throw Error('SOURCE_CHANGED');
  if (e.variants[0].attributes.length === expected.length) return checked;
  const result = structuredClone(checked);
  result.evidence.variants[0].attributes = expected.map(a => ({...a}));
  result.evidence.missingFacts = result.evidence.missingFacts.filter(f => f.field !== 'variants[0].attributes' || f.reason !== DIMENSION_UNOBSERVED);
  result.evidence.missingFacts.push({field:RENDERED_DIMENSION_FIELD,reason:staticReason});
  result.evidence.contentSHA256 = '0'.repeat(64);
  const hash = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(JSON.stringify(result.evidence)));
  result.evidence.contentSHA256 = Array.from(new Uint8Array(hash), b => b.toString(16).padStart(2,'0')).join('');
  return validateAmazonCapture(result);
}
