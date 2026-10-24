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
const recordSchema = z.discriminatedUnion('phase', [
  z.strictObject({ ...common, phase: z.literal('pending') }),
  z.strictObject({ ...common, phase: z.literal('bound'), tabId: z.number().int().min(0).max(Number.MAX_SAFE_INTEGER) }),
]);
type ReturnRecord = z.infer<typeof recordSchema>;
type SessionArea = Pick<chrome.storage.SessionStorageArea, 'get' | 'set' | 'remove' | 'setAccessLevel'>;

// A single navigation projection, never a payload, operation identity or receipt.
export class ReturnSession {
  private record?: ReturnRecord;
  private failed = false;
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
  private async write(record: ReturnRecord) {
    this.assertReadable();
    try { await this.area!.set({ [key]: record }); this.record = record; }
    catch { this.failed = true; throw Error('RETURN_STATE_UNAVAILABLE'); }
  }
  async prepare(sourceURL: string, revision: string) {
    this.assertReadable();
    if (this.record) throw Error('ENTRY_CHANGED');
    await this.write(recordSchema.parse({ version: 1, revision, sourceURL, phase: 'pending' }));
  }
  async bind(tabId: number, revision: string) {
    this.assertReadable();
    if (!this.record || this.record.phase !== 'pending' || this.record.revision !== revision) throw Error('ENTRY_CHANGED');
    await this.write(recordSchema.parse({ ...this.record, phase: 'bound', tabId }));
  }
  async clear() {
    this.assertReadable();
    try { await this.area!.remove(key); this.record = undefined; }
    catch { this.failed = true; throw Error('RETURN_STATE_UNAVAILABLE'); }
  }
}
