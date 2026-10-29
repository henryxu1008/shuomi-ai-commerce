import { z } from 'zod';
import { UUID } from './validation';
import { DraftStore, draftScope, sameDraftScope, type Draft, type DraftScope } from './draft-store';

type Sender = { origin?: string; url?: string; frameId?: number; tab?: { id?: number }; id?: string; documentId?: string };
type Capability = { draftId: string; key: string; nonce: string; expires: number; tabId?: number; documentId?: string };
const common = { version: z.literal(2), handoffId: z.string().regex(UUID), idempotencyKey: z.string().regex(UUID), scope: z.unknown() };
const schema = z.discriminatedUnion('type', [
  z.strictObject({ ...common, type: z.literal('capture.meta') }),
  z.strictObject({ ...common, type: z.literal('capture.read') }),
  z.strictObject({ ...common, type: z.literal('capture.bind'), body: z.string().max(2 * 1024 * 1024) }),
  z.strictObject({ ...common, type: z.literal('capture.send') }),
  z.strictObject({ ...common, type: z.literal('capture.armRetry') }),
  z.strictObject({ ...common, type: z.literal('capture.retry') }),
  z.strictObject({ ...common, type: z.literal('capture.status'), phase: z.enum(['unknown','rejected','saved_reported']), operationId: z.string().regex(UUID).optional(),code:z.enum(['ACQUISITION_CAPACITY','DEADLINE_EXCEEDED','INVALID_ACQUISITION','SOURCE_TOO_LARGE']).optional() }),
]);
// Short-lived receiver capability only. Full payload, bytes and original key are
// owned by DraftStore and survive expiry/reconstruction of this transport.
export class Handoff {
  private capability?: Capability;
  private last?: Draft;
  private retryPermit?: { revision: number; scope: DraftScope; until: number };
  private invalidated=false;
  private queue: Promise<unknown> = Promise.resolve();
  constructor(private readonly app: string, private readonly drafts: DraftStore, private readonly clock: () => number = Date.now) {}
  prepare(draft: Draft, nonce: string) {
    if (this.capability) throw Error('HANDOFF_ACTIVE');
    if (!draft.payload || draft.app !== this.app || !UUID.test(nonce)) throw Error('INVALID_MESSAGE');
    this.last = draft;
    this.capability = { draftId: draft.id, key: draft.key, nonce, expires: this.clock() + 300000 };
  }
  bindTab(id: number) {
    if (!this.capability || this.capability.tabId !== undefined || !Number.isInteger(id) || id < 0) throw Error('INVALID_SENDER');
    this.capability.tabId = id;
  }
  private current() {
    if (!this.capability || this.clock() >= this.capability.expires) throw Error('CAPTURE_EXPIRED');
    return this.capability;
  }
  message(value: unknown, sender: Sender): Promise<unknown> {
    const next = this.queue.then(() => this.handle(value, sender)); this.queue = next.catch(() => {}); return next;
  }
  private async handle(value: unknown, sender: Sender) {
    const p = this.current(), app = new URL(this.app);
    let url: URL; try { url = new URL(sender.url ?? ''); } catch { throw Error('INVALID_SENDER'); }
    if (sender.id !== undefined || p.tabId === undefined || sender.tab?.id !== p.tabId || sender.frameId !== 0 || !sender.documentId
      || sender.origin !== app.origin || url.origin !== app.origin || url.pathname !== app.pathname || url.username || url.password || url.search) throw Error('INVALID_SENDER');
    const m = schema.parse(value), scope = draftScope(m.scope);
    if (m.handoffId !== p.nonce || m.idempotencyKey !== p.key) throw Error('INVALID_MESSAGE');
    if (p.documentId ? p.documentId !== sender.documentId : m.type !== 'capture.meta') {if(p.documentId)this.invalidated=true;throw Error('INVALID_SENDER');}
    const d = await this.drafts.get(p.draftId);
    if (!d || d.key !== p.key || d.app !== this.app) throw Error('DRAFT_CHANGED');
    if (d.scope && !sameDraftScope(d.scope, scope)) throw Error('DRAFT_SCOPE_CHANGED');
    p.documentId ??= sender.documentId;
    let result = d;
    switch (m.type) {
      case 'capture.meta': break;
      case 'capture.read':
        this.last = d;
        return { version: 2, type: 'capture.draft', idempotencyKey: d.key, phase: d.phase, canSubmit:['ready','bound'].includes(d.phase)||d.phase==='rejected'&&['ACQUISITION_CAPACITY','DEADLINE_EXCEEDED'].includes(d.reason),payload: d.payload, ...(d.intent ? { intent: d.intent } : {}) };
      case 'capture.bind': result = await this.drafts.bind(d.id, d.revision, d.generation, scope, m.body); break;
      case 'capture.send': result = await this.drafts.beginSend(d.id, d.revision, d.generation, scope); this.retryPermit = undefined; break;
      case 'capture.status': result = await this.drafts.report(d.id, d.revision, d.generation, scope, m.phase, m.operationId,m.code); break;
      case 'capture.armRetry':
        if (d.phase !== 'unknown' || !d.intent) throw Error('DRAFT_CHANGED');
        this.retryPermit = { revision: d.revision, scope, until: this.clock() + 30000 }; break;
      case 'capture.retry': {
        const permit = this.retryPermit; this.retryPermit = undefined;
        if (!permit || permit.until <= this.clock() || permit.revision !== d.revision || !sameDraftScope(permit.scope, scope)) throw Error('DRAFT_CHANGED');
        result = await this.drafts.retry(d.id, d.revision, d.generation, scope); break;
      }
    }
    this.last = result;
    return { version: 2, type: 'capture.metadata', phase: result.phase, idempotencyKey: result.key };
  }
  snapshot() {
    const d = this.last;
    const outcomes: Record<string,string> = { ready:'awaiting_confirmation',bound:'awaiting_confirmation',sending:'processing',unknown:'outcome_unknown',saved_reported:'published',rejected:'failed' };
    return d ? { outcome: outcomes[d.phase] ?? 'empty', idempotencyKey: d.key, operationId: d.operationId } : { outcome: 'empty' };
  }
  handoffURL(extensionId: string) {
    const p = this.current(), url = new URL(this.app);
    url.hash = new URLSearchParams({ extensionId, handoffId: p.nonce, idempotencyKey: p.key }).toString(); return url.href;
  }
  recoveryURL(): string | null { return this.last ? this.app + '#operationKey=' + this.last.key : null; }
  applicationTabID() { return this.capability?.tabId; }
  canReturn() { return !this.invalidated&&this.capability?.tabId!==undefined&&this.clock()<this.capability.expires; }
}
