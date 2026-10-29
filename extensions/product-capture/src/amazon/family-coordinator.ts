import type { AmazonCapturePayload } from './wire';
import type { FamilyManifest, FamilyMember } from './family-discovery';
import { validateAmazonCapture } from './validation';
import { captureException } from '../entry-errors';
import { checkFamilyCapture, familyDiagnostics, projectFamily, sameFamilyManifest, validateFamilySeed, type FamilyCheckpoint, type FamilySeed } from './family-checkpoint';

export type FamilyPhase = 'initializing' | 'collecting' | 'paused' | 'transferring' | 'transferred' | 'uncertain';
export type FamilySummary = { phase: FamilyPhase; observed: number; total: number; reason: string };
export type FamilyPorts = {
  seed(signal:AbortSignal): Promise<FamilySeed>;
  read(member: FamilyMember, signal:AbortSignal): Promise<{ capture: AmazonCapturePayload; parentASIN: string | null; manifest?: FamilyManifest }>;
  handoff(payload: AmazonCapturePayload): Promise<void>;
  close(): Promise<void>;
};
export type FamilyJournal = {
  resume(): Promise<void>;
  seed(seed: FamilySeed): Promise<void>;
  observe(result: { capture: AmazonCapturePayload; parentASIN: string | null; manifest?: FamilyManifest }): Promise<void>;
  pause(reason: string): Promise<void>;
};
export class FamilyCoordinator {
  private state: FamilySummary = { phase: 'initializing', observed: 0, total: 0, reason: '' };
  private checkpoint?: FamilyCheckpoint;
  private running?: Promise<void>;
  private abort?: AbortController;
  constructor(private readonly ports: FamilyPorts, private readonly journal: FamilyJournal) {}
  snapshot(): FamilySummary { return { ...this.state }; }
  restore(checkpoint: FamilyCheckpoint | undefined, phase = 'paused') {
    if (this.running) throw Error('ACTION_IN_PROGRESS');
    this.checkpoint = checkpoint ? structuredClone(checkpoint) : undefined;
    this.state = { phase: phase === 'ready' || phase === 'bound' || phase === 'sending' || phase === 'unknown' || phase === 'saved_reported' ? 'uncertain' : 'paused',
      observed: checkpoint?.variants.length ?? 0, total: checkpoint?.manifest.members.length ?? 0, reason: 'DRAFT_INTERRUPTED' };
  }
  start() {
    if (this.running) return this.running;
    if (!['initializing','paused'].includes(this.state.phase)) return Promise.resolve();
    this.state.phase = 'collecting'; this.state.reason = ''; this.abort = new AbortController();
    this.running = this.collect(this.abort.signal).catch(async e => {
      if (this.state.phase === 'collecting') {
        const reason = captureException(e);
        try { await this.journal.pause(reason); } catch { /* The last committed checkpoint is still authoritative. */ }
        this.state.phase = 'paused'; this.state.reason = reason;
      }
    }).finally(() => { this.running = undefined; });
    return this.running;
  }
  private async collect(signal: AbortSignal) {
    await this.journal.resume(); signal.throwIfAborted();
    if (!this.checkpoint) {
      const seed = validateFamilySeed(await this.ports.seed(signal)); if (this.state.phase !== 'collecting') return;
      await this.journal.seed(seed); signal.throwIfAborted();
      this.checkpoint = { ...seed, variants: [structuredClone(seed.anchor.evidence.variants[0])], diagnostics: [] };
      this.state.total = seed.manifest.members.length; this.state.observed = 1;
    }
    while (this.checkpoint && this.checkpoint.variants.length < this.checkpoint.manifest.members.length && this.state.phase === 'collecting') {
      const checkpoint:FamilyCheckpoint = this.checkpoint, expected = checkpoint.manifest.members[checkpoint.variants.length];
      const result = await this.ports.read(expected,signal); if (this.state.phase !== 'collecting') return;
      if (result.parentASIN && result.parentASIN !== checkpoint.manifest.parentASIN) throw Error('SOURCE_CHANGED');
      if (result.manifest && !sameFamilyManifest(result.manifest, checkpoint.manifest)) throw Error('SOURCE_CHANGED');
      checkFamilyCapture(result.capture, expected, checkpoint);
      const variant = structuredClone(result.capture.evidence.variants[0]);
      const diagnostics = [...checkpoint.diagnostics, ...familyDiagnostics(result.capture, checkpoint.variants.length)];
      const next:FamilyCheckpoint = { ...checkpoint, variants: [...checkpoint.variants, variant], diagnostics };
      try { projectFamily(next); } catch { throw Error('FAMILY_LIMIT_REACHED'); }
      await this.journal.observe(result);
      this.checkpoint = next; this.state.observed = next.variants.length;
      signal.throwIfAborted();
    }
    if (this.state.phase === 'collecting') await this.deliver();
  }
  private async deliver() {
    if (!['collecting','paused'].includes(this.state.phase) || !this.checkpoint) return;
    const result = projectFamily(this.checkpoint); this.state.phase = 'transferring';
    try {
      await this.ports.close();
      const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(JSON.stringify(result.evidence)));
      result.evidence.contentSHA256 = Array.from(new Uint8Array(digest), b => b.toString(16).padStart(2,'0')).join('');
      await this.ports.handoff(validateAmazonCapture(result)); this.state.phase = 'transferred';
    } catch (e) {
      const reason=captureException(e);
      if(reason==='DRAFT_STORAGE_FULL'){
        try{await this.journal.pause(reason);}catch{/* The prior committed checkpoint remains recoverable even if this state write also cannot fit. */}
        this.state.phase='paused';this.state.reason=reason;
      }else{this.state.phase='uncertain';this.state.reason=reason;}
    }
  }
  resume() { return this.state.phase === 'paused' && this.state.reason !== 'FAMILY_LIMIT_REACHED' ? this.start() : Promise.resolve(); }
  async useObserved() { if (this.state.phase === 'paused' && this.checkpoint?.variants.length) await this.deliver(); }
  async cancel() {
    if (!['initializing','collecting','paused'].includes(this.state.phase)) return;
    this.state.phase = 'paused'; this.abort?.abort();
    await this.journal.pause('');
    await this.ports.close();
  }
}
