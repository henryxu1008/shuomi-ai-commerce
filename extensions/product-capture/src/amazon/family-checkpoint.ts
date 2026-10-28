import { z } from 'zod';
import type { AmazonCapturePayload, AmazonVariant } from './wire';
import type { FamilyManifest, FamilyMember } from './family-discovery';
import { validateAmazonCapture } from './validation';
import { RENDERED_DIMENSION_FIELD } from './dimensions';

export type FamilySeed = { manifest: FamilyManifest; anchor: AmazonCapturePayload };
export type FamilyCheckpoint = FamilySeed & { variants: AmazonVariant[]; diagnostics: { field: string; reason: string }[] };
const text = z.string().min(1).max(2000);
const manifestSchema = z.strictObject({ sourceURL: z.string().max(2048), parentASIN: z.string().regex(/^[A-Z0-9]{10}$/), members: z.array(z.strictObject({ sourceID: z.string().regex(/^[A-Z0-9]{10}$/), attributes: z.array(z.strictObject({ name: text, value: text })).min(1).max(8) })).min(1).max(256) });
const combination = (v: { attributes: { name: string; value: string }[] }) => JSON.stringify([...v.attributes].sort((a,b) => a.name.localeCompare(b.name)));
export function checkFamilyCapture(capture: AmazonCapturePayload, member: FamilyMember, seed?: FamilySeed) {
  const e = validateAmazonCapture(capture).evidence;
  if (seed && e.site !== seed.anchor.evidence.site || e.sourceID !== member.sourceID || e.variants.length !== 1 || e.variants[0].sourceID !== member.sourceID || combination(e.variants[0]) !== combination(member)) throw Error('SOURCE_CHANGED');
}
export function validateFamilySeed(seed: FamilySeed): FamilySeed {
  const manifest = manifestSchema.parse(seed.manifest), anchor = validateAmazonCapture(seed.anchor);
  if (manifest.sourceURL !== anchor.evidence.sourceURL || manifest.parentASIN === anchor.evidence.sourceID || new Set(manifest.members.map(m => m.sourceID)).size !== manifest.members.length || new Set(manifest.members.map(combination)).size !== manifest.members.length) throw Error('FAMILY_INVALID');
  checkFamilyCapture(anchor, manifest.members[0]); return { manifest, anchor };
}
export function familyDiagnostics(capture: AmazonCapturePayload, index: number) {
  const variant = capture.evidence.variants[0];
  return capture.evidence.missingFacts.filter(f => f.field === 'price' || f.field.startsWith('price.') || f.field === 'variants[0].attributes' || f.field === RENDERED_DIMENSION_FIELD || f.field === 'images' && !variant.images.length).map(f => ({ field: `variants.${index}.${f.field.replace(/^variants\[0\]\./,'')}`, reason: f.reason }));
}
export function projectFamily(checkpoint: FamilyCheckpoint, observedAt = new Date().toISOString()): AmazonCapturePayload {
  const { anchor, manifest, variants, diagnostics } = checkpoint;
  if (!variants.length) throw Error('CAPTURE_REQUIRED');
  const result = structuredClone(anchor), e = result.evidence;
  e.variants = structuredClone(variants); e.observation = { scope: 'partial_family', parentASIN: manifest.parentASIN }; e.contentSHA256 = '0'.repeat(64);
  const remaining = manifest.members.slice(variants.length).map(m => m.sourceID);
  e.missingFacts = [...e.missingFacts.filter(f => f.field !== 'variants'), ...diagnostics,
    { field: 'variants', reason: `Observed ${variants.length}/${manifest.members.length} source-listed variants; partial family, not a complete-family guarantee. Unobserved: ${remaining.join(',') || 'none'}.` },
    { field: 'observation.timeRange', reason: `Sequential observations from ${e.capturedAt} to ${observedAt}; not a simultaneous quote.` }];
  return validateAmazonCapture(result);
}
