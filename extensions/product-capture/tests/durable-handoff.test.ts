import 'fake-indexeddb/auto';
import { IDBFactory } from 'fake-indexeddb';
import { readFileSync } from 'node:fs';
import { beforeEach, expect, it } from 'vitest';
import { Handoff } from '../src/handoff';
import { DraftStore, draftBody } from '../src/draft-store';
const app = 'https://app.example.com/capture/1688';
const payload = JSON.parse(readFileSync('tests/fixtures/capture-v2.json','utf8'));
const nonce = crypto.randomUUID();
const scope = { userId: 'user', organizationId: 'org', memberId: 'member' };
const sender = { origin: 'https://app.example.com', url: app, frameId: 0, tab: { id: 42 }, documentId: 'doc1' };
beforeEach(() => { globalThis.indexedDB = new IDBFactory(); });
async function setup() {
  const store = new DraftStore(); await store.ready;
  let draft = await store.create(payload.evidence.sourceURL,'single',app);
  draft = await store.complete(draft.id,draft.revision,draft.generation,payload);
  const flow = new Handoff(app,store); flow.prepare(draft,nonce); flow.bindTab(42);
  const message = (type: string, extra = {}, from:Parameters<Handoff['message']>[1] = sender) => flow.message({ version: 2, type, handoffId: nonce, idempotencyKey: draft.key, scope, ...extra },from);
  return { store,draft,flow,message };
}
it('requires metadata and verified scope before revealing any product body',async () => {
  const { message } = await setup();
  await expect(message('capture.read')).rejects.toThrow('INVALID_SENDER');
  expect(await message('capture.meta')).not.toHaveProperty('payload');
  expect(await message('capture.read')).toMatchObject({ payload });
});
it.each([
  { ...sender, origin: 'https://evil.test' }, { ...sender, url: `${app}/other` },
  { ...sender, url: 'https://app.example.com:8443/capture/1688' },
  { ...sender, frameId: 1 }, { ...sender, tab: { id: 43 } },
  { ...sender, origin: undefined }, { ...sender, id: 'another-extension' },
])('rejects a mismatched native sender',async bad => {
  const { message } = await setup(); await expect(message('capture.meta',{},bad)).rejects.toThrow('INVALID_SENDER');
});
it('binds all bytes, isolates changed identity, and never deletes on a page receipt',async () => {
  const { message,store,draft } = await setup(); await message('capture.meta');
  await expect(message('capture.bind',{ body: JSON.stringify({...payload,evidence:{...payload.evidence,title:'wrong'}}) })).rejects.toThrow('DRAFT_BODY_MISMATCH');
  await message('capture.bind',{ body: draftBody(payload) }); await message('capture.send');
  await message('capture.status',{ phase:'saved_reported',operationId:crypto.randomUUID() });
  expect(await store.get(draft.id)).toBeDefined();
  await expect(message('capture.read',{ scope:{...scope,organizationId:'other'} })).rejects.toThrow('DRAFT_SCOPE_CHANGED');
  await expect(message('capture.read',{}, {...sender,documentId:'new-doc'})).rejects.toThrow('INVALID_SENDER');
});
it('keeps original data after capability expiry; restoring uses the same key',async () => {
  const { store,draft } = await setup(); let now = 1000;
  const flow = new Handoff(app,store,()=>now); flow.prepare(draft,nonce); flow.bindTab(42); now += 300001;
  await expect(flow.message({version:2,type:'capture.meta',handoffId:nonce,idempotencyKey:draft.key,scope},sender)).rejects.toThrow('CAPTURE_EXPIRED');
  expect((await store.get(draft.id))?.payload).toEqual(payload);
  const fresh = new Handoff(app,store); fresh.prepare((await store.get(draft.id))!,crypto.randomUUID());
  expect(new URL(fresh.handoffURL('a'.repeat(32))).hash).toContain(draft.key);
});
