import 'fake-indexeddb/auto';
import { IDBFactory } from 'fake-indexeddb';
import { readFileSync } from 'node:fs';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { Handoff } from '../src/handoff';
import { DraftStore, draftBody } from '../src/draft-store';
import { Controller } from '../src/controller';
import type { PlatformCapturePayload } from '../src/wire';
const app = 'https://app.example.com/capture/1688';
const payload = JSON.parse(readFileSync('tests/fixtures/capture-v2.json','utf8'));
const nonce = crypto.randomUUID();
const scope = { userId: 'user', organizationId: 'org', memberId: 'member' };
const sender = { origin: 'https://app.example.com', url: app, frameId: 0, tab: { id: 42 }, documentId: 'doc1' };
beforeEach(() => { globalThis.indexedDB = new IDBFactory(); });
afterEach(() => vi.restoreAllMocks());
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
it.each([false,true])('requires a new original-key check after retry-send expiry (worker rebuilt=%s)',async rebuilt=>{
  let now=Date.now();vi.spyOn(Date,'now').mockImplementation(()=>now);
  const value=JSON.parse(readFileSync('../../internal/product/sourcing/testdata/amazon-capture-v2.json','utf8')) as PlatformCapturePayload;
  const destination='https://app.example.com/capture/amazon',name=crypto.randomUUID(),store=new DraftStore(name);await store.ready;
  let draft=await store.create(value.evidence.sourceURL,'single',destination);draft=await store.complete(draft.id,draft.revision,draft.generation,value);
  const extract=vi.fn(async()=>value),create=vi.fn<(url:string)=>Promise<number>>(async()=>42);
  const control=(drafts:DraftStore)=>new Controller(drafts,async()=>({url:value.evidence.sourceURL}),destination,extract,create,()=> 'a'.repeat(32));
  let owner=control(store);await owner.restore(draft.id);await owner.handoff();
  const first=new URL(create.mock.calls[0][0]),from={...sender,url:destination} as chrome.runtime.MessageSender;
  const message=(url:URL,type:string,extra={})=>owner.message({version:2,type,handoffId:new URLSearchParams(url.hash.slice(1)).get('handoffId'),idempotencyKey:draft.key,scope,...extra},from);
  await message(first,'capture.meta');await message(first,'capture.bind',{body:draftBody(value)});await message(first,'capture.send');await message(first,'capture.status',{phase:'unknown'});
  const original=await store.get(draft.id);
  await message(first,'capture.armRetry');await message(first,'capture.retry');now+=300001;
  await expect(message(first,'capture.send')).rejects.toThrow('CAPTURE_EXPIRED');
  const restoredStore=rebuilt?new DraftStore(name):store;await restoredStore.ready;
  if(rebuilt)owner=control(restoredStore);
  await owner.restore(draft.id);await owner.handoff();
  const fresh=new URL(create.mock.calls[1][0]);await message(fresh,'capture.meta');
  expect(await message(fresh,'capture.read')).toMatchObject({phase:'unknown',canSubmit:false,intent:original!.intent});
  expect(await restoredStore.get(draft.id)).toMatchObject({key:original!.key,attemptId:original!.attemptId,intent:original!.intent,scope:original!.scope});
  await expect(message(fresh,'capture.send')).rejects.toThrow('DRAFT_CHANGED');
  await expect(message(fresh,'capture.retry')).rejects.toThrow('DRAFT_CHANGED');
  await message(fresh,'capture.armRetry');await message(fresh,'capture.retry');await message(fresh,'capture.send');
  expect((await restoredStore.get(draft.id))?.attemptId).not.toBe(original!.attemptId);expect(extract).not.toHaveBeenCalled();
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
it.each([
  {platform:'amazon',bound:false},{platform:'amazon',bound:true},
  {platform:'1688',bound:false},{platform:'1688',bound:true},
] as const)('explicitly restores the same $platform draft (bound=$bound) after expiry without extraction or submission',async ({platform,bound}) => {
  const value = (platform === 'amazon' ? JSON.parse(readFileSync('../../internal/product/sourcing/testdata/amazon-capture-v2.json','utf8')) : payload) as PlatformCapturePayload;
    let now=Date.now();vi.spyOn(Date,'now').mockImplementation(()=>now);
    const destination=`https://app.example.com/capture/${platform}`,store=new DraftStore(crypto.randomUUID());await store.ready;
    let draft=await store.create(value.evidence.sourceURL,'single',destination);
    draft=await store.complete(draft.id,draft.revision,draft.generation,value);
    const extract=vi.fn(async()=>value),create=vi.fn<(url:string)=>Promise<number>>(async()=>42);
    const control=new Controller(store,async()=>({url:value.evidence.sourceURL}),destination,extract,create,()=> 'a'.repeat(32));
    await control.restore(draft.id);await control.handoff();
    const first=new URL(create.mock.calls[0][0]);
    const from={...sender,url:destination} as chrome.runtime.MessageSender;
    const message=(url:URL,type:string,extra={})=>control.message({version:2,type,handoffId:new URLSearchParams(url.hash.slice(1)).get('handoffId'),idempotencyKey:draft.key,scope,...extra},from);
    await message(first,'capture.meta');
    if(bound)await message(first,'capture.bind',{body:draftBody(value)});
    const before=await store.get(draft.id);
    now+=300001;
    await expect(message(first,'capture.bind',{body:draftBody(value)})).rejects.toThrow('CAPTURE_EXPIRED');
    expect(await store.get(draft.id)).toEqual(before);
    await control.restore(draft.id);await control.handoff();
    const restored=new URL(create.mock.calls[1][0]),params=new URLSearchParams(restored.hash.slice(1));
    expect(params.get('idempotencyKey')).toBe(draft.key);expect(params.get('handoffId')).not.toBe(new URLSearchParams(first.hash.slice(1)).get('handoffId'));
    if(bound)await expect(message(restored,'capture.meta',{scope:{...scope,memberId:'other'}})).rejects.toThrow('DRAFT_SCOPE_CHANGED');
    await message(restored,'capture.meta');
    const read=await message(restored,'capture.read');
    expect(read).toMatchObject({payload:before!.payload,phase:bound?'bound':'ready',canSubmit:true});
    if(bound)expect(read).toHaveProperty('intent',before!.intent);
    expect(await store.get(draft.id)).toEqual(before);expect(await store.list()).toHaveLength(1);expect(extract).not.toHaveBeenCalled();
});
