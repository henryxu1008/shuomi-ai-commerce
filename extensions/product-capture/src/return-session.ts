import { z } from 'zod';
import { capturePageSource } from './platforms';
import { UUID } from './validation';

export type ReturnState = 'none' | 'available' | 'uncertain';
const key = 'product-capture:return-location:v1';
const source = z.string().max(2048).refine(value => {
  try { return capturePageSource(value).sourceURL === value && new URL(value).protocol === 'https:'; }
  catch { return false; }
});
const common = { version: z.literal(1), revision: z.string().regex(UUID), sourceURL: source };
const handoffSchema = z.strictObject({
  draftId: z.string().regex(UUID),
  key: z.string().regex(UUID),
  nonce: z.string().regex(UUID),
  expires: z.number().int().positive(),
  documentId: z.string().regex(/^[A-Za-z0-9_-]{1,128}$/),
});
export type BoundHandoff = z.infer<typeof handoffSchema> & { tabId: number };
const recordSchema = z.discriminatedUnion('phase', [
  z.strictObject({ ...common, phase: z.literal('pending') }),
  z.strictObject({
    ...common, phase: z.literal('bound'), tabId: z.number().int().min(0).max(Number.MAX_SAFE_INTEGER),
    nonce: z.string().regex(UUID).optional(), handoff: handoffSchema.optional(),
  }).refine(value => !value.handoff || value.handoff.nonce === value.nonce),
]);
type ReturnRecord = z.infer<typeof recordSchema>;
type SessionArea = Pick<chrome.storage.SessionStorageArea, 'get' | 'set' | 'remove' | 'setAccessLevel'>;

// Browser-session navigation and document binding. DraftStore owns payload and submission state.
export class ReturnSession {
  private record?: ReturnRecord;
  private failed = false;
  private queue: Promise<unknown> = Promise.resolve();
  readonly ready: Promise<void>;
  constructor(private readonly area: SessionArea | undefined) { this.ready = this.load(); }
  private async load() {
    try {
      if (!this.area) throw Error('Storage unavailable');
      await this.area.setAccessLevel({ accessLevel: 'TRUSTED_CONTEXTS' });
      const values = await this.area.get(key);
      if (!values || typeof values !== 'object' || Array.isArray(values)) throw Error('Invalid storage');
      if (Object.hasOwn(values, key)) this.record = recordSchema.parse(values[key]);
    } catch { this.failed = true; }
  }
  assertReadable() { if (this.failed) throw Error('RETURN_STATE_UNAVAILABLE'); }
  get unavailable() { return this.failed; }
  get revision() { return this.record?.revision; }
  get sourceURL() { return this.record?.sourceURL; }
  get state(): ReturnState { return this.failed ? 'uncertain' : !this.record ? 'none' : this.record.phase === 'bound' ? 'available' : 'uncertain'; }
  tabID() { return !this.failed && this.record?.phase === 'bound' ? this.record.tabId : undefined; }
  get handoff() {
    const record = this.record;
    return !this.failed && record?.phase === 'bound' && record.handoff
      ? { ...record.handoff, tabId: record.tabId, sourceURL: record.sourceURL } : undefined;
  }
  // Receiver operations run on the same queue as navigation changes and revocation.
  run<T>(operation: () => Promise<T>): Promise<T> {
    const next = this.queue.then(operation);
    this.queue = next.catch(() => {});
    return next;
  }
  private async write(record: ReturnRecord) {
    this.assertReadable();
    try { await this.area!.set({ [key]: record }); this.record = record; }
    catch { this.failed = true; throw Error('RETURN_STATE_UNAVAILABLE'); }
  }
  prepare(sourceURL: string, revision: string) {
    return this.run(async () => {
      this.assertReadable();
      if (this.record) throw Error('ENTRY_CHANGED');
      await this.write(recordSchema.parse({ version: 1, revision, sourceURL, phase: 'pending' }));
    });
  }
  bind(tabId: number, revision: string, nonce?: string) {
    return this.run(async () => {
      this.assertReadable();
      if (!this.record || this.record.phase !== 'pending' || this.record.revision !== revision) throw Error('ENTRY_CHANGED');
      await this.write(recordSchema.parse({ ...this.record, phase: 'bound', tabId, ...(nonce ? { nonce } : {}) }));
    });
  }
  assertHandoff(tabId: number, nonce: string) {
    this.assertReadable();
    const record = this.record;
    if (record?.phase !== 'bound' || record.tabId !== tabId || record.nonce !== nonce) throw Error('CAPTURE_EXPIRED');
    return record;
  }
  // Called only during a receiver operation already holding the queue.
  async rememberHandoff({ tabId, ...handoff }: BoundHandoff) {
    const record = this.assertHandoff(tabId, handoff.nonce);
    if (record.handoff) throw Error('ENTRY_CHANGED');
    await this.write(recordSchema.parse({ ...record, handoff }));
  }
  clear() {
    return this.run(async () => {
      this.assertReadable();
      try { await this.area!.remove(key); this.record = undefined; }
      catch { this.failed = true; throw Error('RETURN_STATE_UNAVAILABLE'); }
    });
  }
}
