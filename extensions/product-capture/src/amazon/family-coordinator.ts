import type { AmazonCapturePayload, AmazonVariant } from './wire';
import type { FamilyManifest, FamilyMember } from './family-discovery';
import { validateAmazonCapture } from './validation';
import { captureException } from '../entry-errors';
import { RENDERED_DIMENSION_FIELD } from './dimensions';

export type FamilyPhase = 'initializing' | 'collecting' | 'paused' | 'transferring' | 'transferred' | 'uncertain' | 'cancelled';
export type FamilySummary = { phase: FamilyPhase; observed: number; total: number; reason: string };
export type FamilyPorts = {
  seed(signal:AbortSignal): Promise<{ manifest: FamilyManifest; anchor: AmazonCapturePayload }>;
  read(member: FamilyMember, signal:AbortSignal): Promise<{ capture: AmazonCapturePayload; parentASIN: string | null; manifest?: FamilyManifest }>;
  handoff(payload: AmazonCapturePayload): Promise<void>;
  close(): Promise<void>;
};
const combination = (v: { attributes: { name: string; value: string }[] }) => JSON.stringify([...v.attributes].sort((a,b) => a.name.localeCompare(b.name)));
export class FamilyCoordinator {
  private state: FamilySummary = { phase: 'initializing', observed: 0, total: 0, reason: '' };
  private seedValue?: { manifest: FamilyManifest; anchor: AmazonCapturePayload };
  private variants: AmazonVariant[] = [];
  private diagnostics: { field: string; reason: string }[] = [];
  private running?: Promise<void>;
  private abort?:AbortController;
  constructor(private readonly ports: FamilyPorts) {}
  snapshot(): FamilySummary { return { ...this.state }; }
  start() {
    if (this.running) return this.running;
    if (!['initializing','paused'].includes(this.state.phase)) return Promise.resolve();
    this.state.phase = 'collecting'; this.state.reason = '';
    this.abort=new AbortController();
    this.running = this.collect(this.abort.signal).catch(e => {
      if (this.state.phase === 'collecting') { this.state.phase = 'paused'; this.state.reason = captureException(e); }
    }).finally(() => { this.running = undefined; });
    return this.running;
  }
  private async collect(signal:AbortSignal) {
    if (!this.seedValue) {
      const seed = await this.ports.seed(signal); if (this.state.phase !== 'collecting') return;
      this.check(seed.anchor, seed.manifest.members[0]);
      this.seedValue = seed; this.variants = [structuredClone(seed.anchor.evidence.variants[0])];
      this.state.total = seed.manifest.members.length; this.state.observed = 1;
    }
    while (this.variants.length < this.seedValue.manifest.members.length && this.state.phase === 'collecting') {
      const expected = this.seedValue.manifest.members[this.variants.length];
      const result = await this.ports.read(expected,signal); if (this.state.phase !== 'collecting') return;
      if (result.parentASIN && result.parentASIN !== this.seedValue.manifest.parentASIN) throw Error('SOURCE_CHANGED');
      if (result.manifest) {
        const normalize = (m: FamilyManifest) => JSON.stringify([...m.members].sort((a,b) => a.sourceID.localeCompare(b.sourceID)));
        if (normalize(result.manifest) !== normalize(this.seedValue.manifest)) throw Error('SOURCE_CHANGED');
      }
      this.check(result.capture, expected);
      const variant = structuredClone(result.capture.evidence.variants[0]);
      const missing = result.capture.evidence.missingFacts.filter(f => f.field==='price' || f.field==='price.currency' || f.field==='variants[0].attributes' || f.field===RENDERED_DIMENSION_FIELD || f.field==='images' && !variant.images.length).map(f => ({field:`variants.${this.variants.length}.${f.field.replace(/^variants\[0\]\./,'')}`,reason:f.reason}));
      const diagnostics = [...this.diagnostics, ...missing];
      // Project only one variant/diagnostic set; never retain every child product's album/body.
      try { this.project([...this.variants, variant], diagnostics); } catch { throw Error('FAMILY_LIMIT_REACHED'); }
      this.variants.push(variant); this.diagnostics = diagnostics; this.state.observed = this.variants.length;
    }
    if (this.state.phase === 'collecting') await this.deliver();
  }
  private check(capture: AmazonCapturePayload, member: FamilyMember) {
    const e = validateAmazonCapture(capture).evidence;
    if (this.seedValue && e.site!==this.seedValue.anchor.evidence.site)throw Error('SOURCE_CHANGED');
    if (e.sourceID !== member.sourceID || e.variants.length !== 1 || e.variants[0].sourceID !== member.sourceID || combination(e.variants[0]) !== combination(member)) throw Error('SOURCE_CHANGED');
  }
  private project(variants = this.variants, diagnostics = this.diagnostics): AmazonCapturePayload {
    if (!this.seedValue || !variants.length) throw Error('CAPTURE_REQUIRED');
    const result = structuredClone(this.seedValue.anchor), e = result.evidence;
    e.variants = variants; e.observation = {scope:'partial_family',parentASIN:this.seedValue.manifest.parentASIN}; e.contentSHA256 = '0'.repeat(64);
    const remaining = this.seedValue.manifest.members.slice(variants.length).map(m => m.sourceID);
    e.missingFacts = [...e.missingFacts.filter(f => f.field !== 'variants'), ...diagnostics,
      {field:'variants',reason:`Observed ${variants.length}/${this.seedValue.manifest.members.length} source-listed variants; partial family, not a complete-family guarantee. Unobserved: ${remaining.join(',') || 'none'}.`},
      {field:'observation.timeRange',reason:`Sequential observations from ${e.capturedAt} to ${new Date().toISOString()}; not a simultaneous quote.`}];
    return validateAmazonCapture(result);
  }
  private async deliver() {
    if (!['collecting','paused'].includes(this.state.phase)) return;
    const result = this.project(); this.state.phase = 'transferring';
    try {
      await this.ports.close();
      const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(JSON.stringify(result.evidence)));
      result.evidence.contentSHA256 = Array.from(new Uint8Array(digest), b => b.toString(16).padStart(2,'0')).join('');
      await this.ports.handoff(validateAmazonCapture(result)); this.state.phase = 'transferred';
    } catch { this.state.phase = 'uncertain'; this.state.reason = 'HANDOFF_UNAVAILABLE'; }
  }
  resume() { return this.state.phase === 'paused' && this.state.reason !== 'FAMILY_LIMIT_REACHED' ? this.start() : Promise.resolve(); }
  async useObserved() { if (this.state.phase === 'paused' && this.variants.length) await this.deliver(); }
  async cancel() {
    if (!['initializing','collecting','paused'].includes(this.state.phase)) return;
    this.state.phase = 'cancelled'; this.abort?.abort(); this.variants = []; this.diagnostics = []; this.seedValue = undefined;
    await this.ports.close();
  }
}
