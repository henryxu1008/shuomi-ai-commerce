import { openDB, type DBSchema, type IDBPDatabase } from 'idb';
import { z } from 'zod';
import { validatePlatformCapture, capturePageSource } from './platforms';
import { byteLength, MAX_BYTES, UUID } from './validation';
import type { PlatformCapturePayload } from './wire';
import type { FamilyManifest } from './amazon/family-discovery';
import type { AmazonCapturePayload, AmazonVariant } from './amazon/wire';
import { amazonCaptureBody } from '../../../web/listingkit-ui/src/lib/contracts/amazon-capture-body';
import { checkFamilyCapture, familyDiagnostics, projectFamily, sameFamilyManifest, validateFamilySeed, type FamilyCheckpoint, type FamilySeed } from './amazon/family-checkpoint';

export type DraftScope = { userId: string; organizationId: string; memberId: string };
export const sameDraftScope = (a: DraftScope, b: DraftScope) => a.userId === b.userId && a.organizationId === b.organizationId && a.memberId === b.memberId;
export type DraftPhase = 'capturing' | 'paused' | 'ready' | 'bound' | 'sending' | 'unknown' | 'saved_reported' | 'rejected';
export type Draft = {
  schemaVersion: 1; id: string; key: string; sourceURL: string; mode: 'single' | 'family'; app: string;
  revision: number; generation: string; createdAt: number; updatedAt: number; phase: DraftPhase; reason: string;
  payload?: PlatformCapturePayload; family?: { anchor: AmazonCapturePayload; manifest: FamilyManifest; digest: string; count: number };
  scope?: DraftScope; intent?: { body: string; digest: string; bytes: number }; attemptId?: string; operationId?: string;
};
export type Observation = { draftId: string; sourceID: string; variant: AmazonVariant; diagnostics: { field: string; reason: string }[]; observedAt: string; digest?: string };
type Envelope = { id: string; revision: number; json: string; digest: string };
interface CaptureDB extends DBSchema {
  drafts: { key: string; value: Envelope };
  observations: { key: [string, string]; value: Observation; indexes: { draftId: string } };
}
const scopeSchema = z.strictObject({ userId: z.string().min(1).max(200), organizationId: z.string().min(1).max(200), memberId: z.string().min(1).max(200) });
export function draftScope(value: unknown) { return scopeSchema.parse(value); }
export async function sha256(value: string) {
  return Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(value))), b => b.toString(16).padStart(2, '0')).join('');
}
export function draftBody(payload: PlatformCapturePayload) {
  const value = validatePlatformCapture(payload);
  return capturePageSource(value.evidence.sourceURL).platform === 'amazon' ? amazonCaptureBody(value as AmazonCapturePayload) : JSON.stringify(value);
}
async function envelope(draft: Draft): Promise<Envelope> {
  const json = JSON.stringify(draft); return { id: draft.id, revision: draft.revision, json, digest: await sha256(json) };
}
async function decode(value: Envelope): Promise<Draft> {
  if (!value || typeof value.json !== 'string' || await sha256(value.json) !== value.digest) throw Error('DRAFT_CORRUPT');
  const draft = JSON.parse(value.json) as Draft;
  if (draft.schemaVersion !== 1 || draft.id !== value.id || !UUID.test(draft.id) || !UUID.test(draft.key) || !UUID.test(draft.generation) || draft.revision !== value.revision
    || !['capturing','paused','ready','bound','sending','unknown','saved_reported','rejected'].includes(draft.phase)) throw Error('DRAFT_CORRUPT');
  if (draft.payload) validatePlatformCapture(draft.payload);
  return draft;
}

// Service-worker owned; pages receive projections only. IndexedDB is the sole
// durable owner of unpublished facts, not another business publication store.
export class DraftStore {
  private readonly db: Promise<IDBPDatabase<CaptureDB>>;
  readonly ready: Promise<void>;
  constructor(name = 'product-capture-drafts-v1', private readonly budget = 8 * 1024 * 1024) {
    this.db = openDB<CaptureDB>(name, 1, { upgrade(db) {
      db.createObjectStore('drafts', { keyPath: 'id' });
      db.createObjectStore('observations', { keyPath: ['draftId','sourceID'] }).createIndex('draftId', 'draftId');
    } });
    this.ready = this.recover();
  }
  private async recover() {
    const db = await this.db;
    // The old worker may have committed a checkpoint while hashes were being
    // checked. Fence only the exact scanned envelope, then re-read conflicts.
    for (;;) {
      const values=await db.getAll('drafts'),changes:{before:Envelope;after:Envelope}[]=[];
      for(const value of values){
        let d:Draft;try{d=await decode(value);}catch{continue;}
        if(d.phase==='capturing'||d.phase==='sending'){
          d.phase=d.phase==='sending'?'unknown':'paused';d.reason='DRAFT_INTERRUPTED';d.revision++;d.generation=crypto.randomUUID();d.updatedAt=Date.now();
          changes.push({before:value,after:await envelope(d)});
        }
      }
      const tx=db.transaction('drafts','readwrite',{durability:'strict'});
      const current=await tx.store.getAll();let changed=current.some(row=>!values.some(v=>v.id===row.id&&v.revision===row.revision&&v.json===row.json&&v.digest===row.digest));
      for(const {before,after} of changes){
        const row=current.find(v=>v.id===before.id);
        if(row?.revision!==before.revision||row.json!==before.json||row.digest!==before.digest){changed=true;continue;}
        await tx.store.put(after);
      }
      await tx.done;if(!changed)return;
    }
  }

  async get(id: string): Promise<Draft | undefined> {
    await this.ready; const value = await (await this.db).get('drafts', id); return value ? decode(value) : undefined;
  }
  async list(): Promise<Draft[]> {
    await this.ready; const rows=await (await this.db).getAll('drafts');
    const decoded=await Promise.all(rows.map(async row=>{try{return await decode(row);}catch{return undefined;}}));
    return decoded.filter((d):d is Draft=>!!d);
  }
  async summaries() {
    await this.ready;
    return Promise.all((await (await this.db).getAll('drafts')).map(async value => {
      try { const d = await decode(value); return { id:d.id, revision:d.revision, sourceURL:d.sourceURL, phase:d.phase as string, observed:d.family?.count??0, total:d.family?.manifest.members.length??0, createdAt:d.createdAt, restorable:!!d.payload||!!d.family }; }
      catch { return { id:value.id, revision:value.revision, sourceURL:'', phase:'corrupt', observed:0, total:0, createdAt:0, restorable:false }; }
    }));
  }
  async latest(sourceURL?: string) { return (await this.list()).filter(d => !sourceURL || d.sourceURL === sourceURL).sort((a,b) => b.createdAt - a.createdAt)[0]; }
  async create(sourceURL: string, mode: 'single' | 'family', app: string, id: string = crypto.randomUUID()) {
    await this.ready; const source = capturePageSource(sourceURL);
    const destination = new URL(app);
    if (destination.pathname !== `/capture/${source.platform}` || destination.hash || destination.search) throw Error('INVALID_APP_URL');
    if (!UUID.test(id)) throw Error('INVALID_MESSAGE');
    const createdAt=Math.max(Date.now(),...(await this.list()).map(d=>d.createdAt+1));
    const d: Draft = { schemaVersion: 1, id, key: crypto.randomUUID(), sourceURL: source.sourceURL, mode, app,
      revision: 1, generation: crypto.randomUUID(), createdAt, updatedAt: createdAt, phase: 'capturing', reason: '' };
    return this.commit(d, undefined);
  }
  private async commit(d: Draft, previous: Draft | undefined, observation?: Observation, compact = false) {
    if (observation) observation = { ...observation, digest: await sha256(JSON.stringify(observation)) };
    const value = await envelope(d), expected=previous?await envelope(previous):undefined,db = await this.db;
    const activeSnapshot=d.phase==='capturing'?await db.getAll('drafts'):[];
    const phases=new Map<string,DraftPhase>();
    // A corrupt envelope cannot pass get() or the exact previous-envelope CAS
    // below. Preserve it without giving it write authority over healthy drafts.
    for(const row of activeSnapshot){try{phases.set(row.id,(await decode(row)).phase);}catch{/* Unwritable; retain the original bytes. */}}
    const tx = db.transaction(['drafts','observations'], 'readwrite', { durability: 'strict' });
    try {
      const old = await tx.objectStore('drafts').get(d.id);
      if (expected ? !old || old.revision!==expected.revision || old.json!==expected.json || old.digest!==expected.digest : !!old) throw Error('DRAFT_CHANGED');
      const drafts = await tx.objectStore('drafts').getAll();
      // A single active capture; creating a new operation never evicts another.
      if(d.phase==='capturing'){
        if(drafts.some(row=>!activeSnapshot.some(v=>v.id===row.id&&v.revision===row.revision&&v.json===row.json&&v.digest===row.digest)))throw Error('DRAFT_CHANGED');
        if(drafts.some(v=>v.id!==d.id&&phases.get(v.id)==='capturing'))throw Error('ACTION_IN_PROGRESS');
      }
      let observations = await tx.objectStore('observations').getAll();
      if (compact) {
        for (const item of observations.filter(o => o.draftId === d.id)) await tx.objectStore('observations').delete([item.draftId,item.sourceID]);
        observations = observations.filter(o => o.draftId !== d.id);
      }
      if (observation) {
        if (await tx.objectStore('observations').get([observation.draftId, observation.sourceID])) throw Error('DRAFT_CHANGED');
        observations.push(observation); await tx.objectStore('observations').add(observation);
      }
      const size = byteLength(JSON.stringify([...drafts.filter(v => v.id !== d.id), value])) + byteLength(JSON.stringify(observations));
      if (size > this.budget) throw Error('DRAFT_STORAGE_FULL');
      await tx.objectStore('drafts').put(value); await tx.done; return structuredClone(d);
    } catch (e) {
      try { tx.abort(); } catch { /* Already aborted. */ }
      await tx.done.catch(() => {});
      if (e instanceof DOMException && e.name === 'QuotaExceededError') throw Error('DRAFT_STORAGE_FULL');
      throw e;
    }
  }
  private async update(id: string, revision: number, generation: string, change: (d: Draft) => void, observation?: Observation, compact = false) {
    const before = await this.get(id);
    if (!before || before.revision !== revision || before.generation !== generation) throw Error('DRAFT_CHANGED');
    const after = structuredClone(before); change(after); after.revision++; after.updatedAt = Date.now();
    return this.commit(after, before, observation, compact);
  }
  complete(id: string, revision: number, generation: string, payload: unknown) {
    const validated = validatePlatformCapture(payload);
    return this.update(id, revision, generation, d => {
      if (!['capturing','paused'].includes(d.phase) || validated.evidence.sourceURL !== d.sourceURL) throw Error('DRAFT_CHANGED');
      d.payload = validated; d.family = undefined; d.phase = 'ready'; d.reason = '';
    }, undefined, true);
  }
  async seed(id: string, revision: number, generation: string, input: FamilySeed) {
    const seed = validateFamilySeed(input), digest = await sha256(JSON.stringify(seed.manifest));
    return this.update(id, revision, generation, d => {
      if (d.mode !== 'family' || d.phase !== 'capturing' || d.family || d.sourceURL !== seed.manifest.sourceURL) throw Error('DRAFT_CHANGED');
      d.family = { ...seed, digest, count: 1 };
    }, { draftId: id, sourceID: seed.manifest.members[0].sourceID, variant: seed.anchor.evidence.variants[0], diagnostics: [], observedAt: seed.anchor.evidence.capturedAt });
  }
  async checkpoint(id: string): Promise<FamilyCheckpoint | undefined> {
    const d = await this.get(id); if (!d?.family) return;
    const f = d.family;
    if (await sha256(JSON.stringify(f.manifest)) !== f.digest) throw Error('DRAFT_CORRUPT');
    validateFamilySeed(f);
    const observations = await (await this.db).getAllFromIndex('observations', 'draftId', id);
    for (const {digest,...observation} of observations) if (await sha256(JSON.stringify(observation)) !== digest) throw Error('DRAFT_CORRUPT');
    const ordered = f.manifest.members.slice(0, f.count).map(member => observations.find(o => o.sourceID === member.sourceID));
    if (observations.length !== f.count || ordered.some(o => !o)) throw Error('DRAFT_CORRUPT');
    const result = { anchor: f.anchor, manifest: f.manifest, variants: ordered.map(o => o!.variant), diagnostics: ordered.flatMap(o => o!.diagnostics) };
    projectFamily(result); return result;
  }
  async observe(id: string, revision: number, generation: string, result: { capture: AmazonCapturePayload; parentASIN: string | null; manifest?: FamilyManifest }) {
    const d = await this.get(id), checkpoint = await this.checkpoint(id);
    if (!d?.family || !checkpoint || d.phase !== 'capturing' || d.revision !== revision || d.generation !== generation) throw Error('DRAFT_CHANGED');
    const member = checkpoint.manifest.members[checkpoint.variants.length]; if (!member) throw Error('DRAFT_CHANGED');
    if (result.parentASIN && result.parentASIN !== checkpoint.manifest.parentASIN) throw Error('SOURCE_CHANGED');
    if (result.manifest && !sameFamilyManifest(result.manifest, checkpoint.manifest)) throw Error('SOURCE_CHANGED');
    checkFamilyCapture(result.capture, member, checkpoint);
    const observation: Observation = { draftId: id, sourceID: member.sourceID, variant: result.capture.evidence.variants[0], diagnostics: familyDiagnostics(result.capture, checkpoint.variants.length), observedAt: result.capture.evidence.capturedAt };
    try { projectFamily({ ...checkpoint, variants: [...checkpoint.variants, observation.variant], diagnostics: [...checkpoint.diagnostics, ...observation.diagnostics] }); } catch { throw Error('FAMILY_LIMIT_REACHED'); }
    return this.update(id, revision, generation, value => { value.family!.count++; }, observation);
  }
  async finishFamily(id: string, revision: number, generation: string) {
    const checkpoint = await this.checkpoint(id); if (!checkpoint) throw Error('CAPTURE_REQUIRED');
    const result = projectFamily(checkpoint); result.evidence.contentSHA256 = await sha256(JSON.stringify(result.evidence));
    return this.complete(id, revision, generation, result);
  }
  pause(id: string, revision: number, generation: string, reason = '') {
    return this.update(id, revision, generation, d => { if (d.phase !== 'capturing') throw Error('DRAFT_CHANGED'); d.phase = 'paused'; d.reason = reason; d.generation = crypto.randomUUID(); });
  }
  resume(id: string, revision: number, generation: string) {
    return this.update(id, revision, generation, d => { if (d.phase !== 'paused') throw Error('DRAFT_CHANGED'); d.phase = 'capturing'; d.reason = ''; d.generation = crypto.randomUUID(); });
  }
  async bind(id: string, revision: number, generation: string, scope: DraftScope, body: string) {
    scope = draftScope(scope); const d = await this.get(id);
    if (!d?.payload || byteLength(body) > MAX_BYTES || body !== draftBody(d.payload)) throw Error('DRAFT_BODY_MISMATCH');
    const intent = { body, digest: await sha256(body), bytes: byteLength(body) };
    return this.update(id, revision, generation, value => {
      if (value.scope && !sameDraftScope(value.scope, scope)) throw Error('DRAFT_SCOPE_CHANGED');
      if (value.intent && value.intent.body !== body || !['ready','bound','unknown','rejected','saved_reported'].includes(value.phase)) throw Error('DRAFT_CHANGED');
      value.scope = scope; value.intent = intent; if (value.phase === 'ready') value.phase = 'bound';
    });
  }
  async beginSend(id: string, revision: number, generation: string, scope: DraftScope) {
    return this.update(id, revision, generation, d => {
      if (!d.scope || !sameDraftScope(d.scope, draftScope(scope)) || !d.intent || d.phase !== 'bound' && !(d.phase === 'rejected' && ['ACQUISITION_CAPACITY','DEADLINE_EXCEEDED'].includes(d.reason))) throw Error('DRAFT_CHANGED');
      d.phase = 'sending'; d.attemptId = crypto.randomUUID();
    });
  }
  retry(id: string, revision: number, generation: string, scope: DraftScope) {
    return this.update(id, revision, generation, d => {
      if (!d.scope || !sameDraftScope(d.scope, draftScope(scope)) || !d.intent || d.phase !== 'unknown' || capturePageSource(d.sourceURL).platform !== 'amazon') throw Error('DRAFT_CHANGED');
      d.phase = 'bound';
    });
  }
  async report(id: string, revision: number, generation: string, scope: DraftScope, phase: 'unknown' | 'rejected' | 'saved_reported', operationId?: string, code?: string) {
    return this.update(id, revision, generation, d => {
      if (!d.scope || !sameDraftScope(d.scope, draftScope(scope)) || !d.intent || !['sending','unknown','rejected','saved_reported'].includes(d.phase)) throw Error('DRAFT_CHANGED');
      if (operationId && !UUID.test(operationId)) throw Error('INVALID_MESSAGE');
      if (d.phase === 'saved_reported' && phase !== 'saved_reported') throw Error('DRAFT_CHANGED');
      d.phase = phase; d.operationId = operationId ?? d.operationId; d.reason = phase === 'rejected' && ['ACQUISITION_CAPACITY','DEADLINE_EXCEEDED','INVALID_ACQUISITION','SOURCE_TOO_LARGE'].includes(code??'') ? code! : '';
    });
  }
  interrupt(id: string, revision: number, generation: string) {
    return this.update(id, revision, generation, d => { if(d.phase!=='sending')throw Error('DRAFT_CHANGED');d.phase='unknown'; });
  }
  async remove(id: string, revision: number) {
    await this.ready;
    const raw = await (await this.db).get('drafts',id);if(!raw||raw.revision!==revision)throw Error('DRAFT_CHANGED');
    let d:Draft|undefined;
    try{d=await decode(raw);}catch{/* Explicit local cleanup is also the only way to discard a corrupt draft. */}
    if(d?.phase==='sending'||d?.phase==='capturing')throw Error('DRAFT_CHANGED');
    const db = await this.db, tx = db.transaction(['drafts','observations'], 'readwrite', { durability: 'strict' });
    const current = await tx.objectStore('drafts').get(id);
    if (current?.revision !== revision) { tx.abort(); await tx.done.catch(() => {}); throw Error('DRAFT_CHANGED'); }
    await tx.objectStore('drafts').delete(id);
    for (const o of await tx.objectStore('observations').index('draftId').getAll(id)) await tx.objectStore('observations').delete([id,o.sourceID]);
    await tx.done;
  }
}
