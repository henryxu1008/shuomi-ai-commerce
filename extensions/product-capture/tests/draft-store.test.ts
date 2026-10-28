import 'fake-indexeddb/auto';
import { IDBFactory } from 'fake-indexeddb';
import { beforeEach, expect, it,vi } from 'vitest';
import { openDB } from 'idb';
import { JSDOM } from 'jsdom';
import { captureAmazonDocument } from '../src/amazon/extractor';
import type { AmazonCapturePayload } from '../src/amazon/wire';
import { browserCaptureSchema } from '../../../web/listingkit-ui/src/lib/contracts/browser-capture';
import { draftBody } from '../src/draft-store';
import { readFileSync } from 'node:fs';
import { DraftStore } from '../src/draft-store';

const sourceURL = 'https://detail.1688.com/offer/9.html';
const app = 'https://app.example.com/capture/1688';
const payload = JSON.parse(readFileSync(new URL('./fixtures/capture-v2.json', import.meta.url), 'utf8'));
beforeEach(() => { globalThis.indexedDB = new IDBFactory(); });

it('retains the full validated payload and original operation key across a new worker', async () => {
  const store = new DraftStore(); await store.ready;
  const draft = await store.create(payload.evidence.sourceURL, 'single', app);
  const saved = await store.complete(draft.id, draft.revision, draft.generation, payload);
  const restarted = new DraftStore(); await restarted.ready;
  const restored = await restarted.get(draft.id);
  expect(restored?.payload).toEqual(payload);
  expect(restored?.key).toBe(saved.key);
  expect(restored?.phase).toBe('ready');
});

it('fences a late old writer before accepting commands after worker reconstruction', async () => {
  const store = new DraftStore(); await store.ready;
  const draft = await store.create(sourceURL, 'single', app);
  const restarted = new DraftStore(); await restarted.ready;
  expect((await restarted.get(draft.id))?.phase).toBe('paused');
  await expect(store.complete(draft.id, draft.revision, draft.generation, payload)).rejects.toThrow('DRAFT_CHANGED');
});

it('rejects capacity overflow atomically and preserves the previous checkpoint', async () => {
  const store = new DraftStore('limited', 1800); await store.ready;
  const draft = await store.create(payload.evidence.sourceURL, 'single', app);
  await expect(store.complete(draft.id, draft.revision, draft.generation, payload)).rejects.toThrow('DRAFT_STORAGE_FULL');
  expect((await store.get(draft.id))?.revision).toBe(draft.revision);
  expect((await store.get(draft.id))?.payload).toBeUndefined();
});

it('keeps stopped drafts when allocating a separate new operation', async () => {
  const store = new DraftStore(); await store.ready;
  const original = await store.create(sourceURL, 'single', app);
  await store.pause(original.id, original.revision, original.generation);
  const next = await store.create(sourceURL, 'single', app);
  expect(next.id).not.toBe(original.id); expect(next.key).not.toBe(original.key);
  expect(await store.get(original.id)).toBeDefined();
});
it('restores the newest operation when the clock has not advanced', async () => {
  const now=vi.spyOn(Date,'now').mockReturnValue(1000);
  try {
    const store=new DraftStore('same-clock');
    const first=await store.create(sourceURL,'single',app,'00000000-0000-4000-8000-000000000000');
    await store.pause(first.id,first.revision,first.generation);
    const second=await store.create(sourceURL,'single',app,'ffffffff-ffff-4fff-8fff-ffffffffffff');
    const restarted=new DraftStore('same-clock');
    expect((await restarted.latest(sourceURL))?.id).toBe(second.id);
  } finally { now.mockRestore(); }
});

async function family(name:string,budget?:number){
  const dom=new JSDOM(readFileSync('tests/fixtures/amazon-inline-product.html','utf8'),{url:'https://www.amazon.com/dp/B0BDLGZCTY'});
  const anchor=await captureAmazonDocument(dom.window.document,dom.window.document.URL);dom.window.close();
  const child=structuredClone(anchor);child.evidence.sourceID='B000000002';child.evidence.sourceURL='https://www.amazon.com/dp/B000000002';child.evidence.variants[0].sourceID='B000000002';child.evidence.variants[0].attributes=anchor.evidence.variants[0].attributes.map(a=>a.name==='Color'?{...a,value:'Beige'}:a);
  const manifest={sourceURL:anchor.evidence.sourceURL,parentASIN:'B0D76YXPWZ',members:[anchor,child].map(p=>({sourceID:p.evidence.sourceID,attributes:p.evidence.variants[0].attributes}))};
  const store=new DraftStore(name,budget);let d=await store.create(anchor.evidence.sourceURL,'family','https://app.example.com/capture/amazon');d=await store.seed(d.id,d.revision,d.generation,{anchor,manifest});
  return {store,d,anchor,child,manifest};
}
it('rebuilds the full root and ordered manifest after restart without counting an uncommitted child',async()=>{
 const {store,d,anchor,child,manifest}=await family('family');
 const restarted=new DraftStore('family');await restarted.ready;
 const checkpoint=await restarted.checkpoint(d.id);expect(checkpoint?.anchor).toEqual(anchor);expect(checkpoint?.manifest).toEqual(manifest);expect(checkpoint?.variants).toHaveLength(1);
 await expect(store.observe(d.id,d.revision,d.generation,{capture:child,parentASIN:manifest.parentASIN})).rejects.toThrow('DRAFT_CHANGED');
 let current=(await restarted.get(d.id))!;current=await restarted.resume(current.id,current.revision,current.generation);current=await restarted.observe(current.id,current.revision,current.generation,{capture:child,parentASIN:manifest.parentASIN});
 const finished=await restarted.finishFamily(current.id,current.revision,current.generation);expect((finished.payload as AmazonCapturePayload).evidence.variants).toHaveLength(2);expect(await restarted.checkpoint(d.id)).toBeUndefined();
 const db=await openDB('family');expect(await db.count('observations')).toBe(0);db.close();
});
it('an actual failed IndexedDB write leaves the last checkpoint and revision intact',async()=>{
 const {store,d,child,manifest}=await family('abort');
 const put=vi.spyOn(IDBObjectStore.prototype,'put').mockImplementationOnce(()=>{throw new DOMException('disk quota','QuotaExceededError');});
 try{await expect(store.observe(d.id,d.revision,d.generation,{capture:child,parentASIN:manifest.parentASIN})).rejects.toThrow('DRAFT_STORAGE_FULL');}finally{put.mockRestore();}
 expect((await store.get(d.id))?.revision).toBe(d.revision);expect((await store.checkpoint(d.id))?.variants).toHaveLength(1);
});
it('detects a valid-looking corrupted observation and never reassembles it into a payload',async()=>{
 const {store,d}=await family('corrupt');const db=await openDB('corrupt');const row=(await db.getAll('observations'))[0];row.variant.price.amount='999.99';await db.put('observations',row);db.close();
 await expect(store.checkpoint(d.id)).rejects.toThrow('DRAFT_CORRUPT');expect((await store.get(d.id))?.payload).toBeUndefined();
});
it('preserves a frozen sending intent across restart as unknown and rejects another identity',async()=>{
 const store=new DraftStore('intent');let d=await store.create(payload.evidence.sourceURL,'single',app);d=await store.complete(d.id,d.revision,d.generation,payload);
 const scope={userId:'actor',organizationId:'org',memberId:'member'},body=JSON.stringify(payload);
 d=await store.bind(d.id,d.revision,d.generation,scope,body);d=await store.beginSend(d.id,d.revision,d.generation,scope);
 const restarted=new DraftStore('intent');await restarted.ready;const restored=(await restarted.get(d.id))!;expect(restored.phase).toBe('unknown');expect(restored.intent?.body).toBe(body);expect(restored.key).toBe(d.key);
 await expect(restarted.bind(restored.id,restored.revision,restored.generation,{...scope,memberId:'recreated'},body)).rejects.toThrow('DRAFT_SCOPE_CHANGED');
 await expect(restarted.bind(restored.id,restored.revision,restored.generation,scope,body+' ')).rejects.toThrow('DRAFT_BODY_MISMATCH');
});
it('uses exactly the current 1688 application serialization, including optional source fields',()=>{
 expect(JSON.stringify(browserCaptureSchema.parse(payload))).toBe(draftBody(payload));
});
it('leaves the original family checkpoint or ready payload intact when finalization or binding has no headroom',async()=>{
 const {store,d}=await family('headroom');
 const limited=new DraftStore('headroom',1);await limited.ready;let current=(await limited.get(d.id))!;
 await expect(limited.finishFamily(current.id,current.revision,current.generation)).rejects.toThrow('DRAFT_STORAGE_FULL');expect((await limited.checkpoint(d.id))?.variants).toHaveLength(1);
 current=await store.finishFamily(current.id,current.revision,current.generation);
 const body=draftBody(current.payload!);await expect(limited.bind(current.id,current.revision,current.generation,{userId:'actor',organizationId:'org',memberId:'member'},body)).rejects.toThrow('DRAFT_STORAGE_FULL');
 expect((await store.get(d.id))?.payload).toEqual(current.payload);expect((await store.get(d.id))?.intent).toBeUndefined();
});
it('recovers a healthy draft while preserving a corrupt sibling verbatim, including when resuming a healthy family',async()=>{
 const {store,d}=await family('mixed');let paused=await store.pause(d.id,d.revision,d.generation);
 const bad=await store.create(payload.evidence.sourceURL,'single',app);await store.pause(bad.id,bad.revision,bad.generation);
 const db=await openDB('mixed');const raw=await db.get('drafts',bad.id);raw.json='{damaged';await db.put('drafts',raw);
 const restarted=new DraftStore('mixed');await restarted.ready;expect((await restarted.latest())?.id).toBe(d.id);expect((await restarted.summaries()).find(x=>x.id===bad.id)?.phase).toBe('corrupt');
 paused=await restarted.resume(paused.id,paused.revision,paused.generation);expect((await restarted.checkpoint(d.id))?.variants).toHaveLength(1);
 expect(await db.get('drafts',bad.id)).toEqual(raw);expect(paused.phase).toBe('capturing');db.close();
});
it('does not overwrite a checkpoint that committed after the recovery scan',async()=>{
 const store=new DraftStore('fence-race');const d=await store.create(payload.evidence.sourceURL,'single',app);
 const original=crypto.subtle.digest.bind(crypto.subtle);let release!:()=>void;
 const spy=vi.spyOn(crypto.subtle,'digest').mockImplementationOnce(async(...args)=>{const hash=await original(...args);await new Promise<void>(r=>{release=r;});return hash;});
 try{
  const restarted=new DraftStore('fence-race');await vi.waitFor(()=>expect(release).toBeTypeOf('function'));
  const finished=await store.complete(d.id,d.revision,d.generation,payload);release();await restarted.ready;
  expect(await restarted.get(d.id)).toEqual(finished);
 }finally{spy.mockRestore();}
});
it('preserves an outer revision corrupted after the recovery scan verbatim',async()=>{
 const store=new DraftStore('outer-corrupt');const d=await store.create(payload.evidence.sourceURL,'single',app);
 const original=crypto.subtle.digest.bind(crypto.subtle);let release!:()=>void;
 const spy=vi.spyOn(crypto.subtle,'digest').mockImplementationOnce(async(...args)=>{const hash=await original(...args);await new Promise<void>(r=>{release=r;});return hash;});
 try{
  const restarted=new DraftStore('outer-corrupt');await vi.waitFor(()=>expect(release).toBeTypeOf('function'));
  const db=await openDB('outer-corrupt');const raw=await db.get('drafts',d.id);raw.revision++;await db.put('drafts',raw);release();await restarted.ready;
  expect(await db.get('drafts',d.id)).toEqual(raw);await expect(restarted.get(d.id)).rejects.toThrow('DRAFT_CORRUPT');db.close();
 }finally{spy.mockRestore();}
});
it('never repairs or overwrites a record corrupted while an old writer was preparing its commit',async()=>{
 const store=new DraftStore('corrupt-race');const d=await store.create(payload.evidence.sourceURL,'single',app);
 const original=crypto.subtle.digest.bind(crypto.subtle);let release!:()=>void;
 const spy=vi.spyOn(crypto.subtle,'digest').mockImplementationOnce(original).mockImplementationOnce(async(...args)=>{const hash=await original(...args);await new Promise<void>(r=>{release=r;});return hash;});
 try{
  const writing=store.complete(d.id,d.revision,d.generation,payload);await vi.waitFor(()=>expect(release).toBeTypeOf('function'));
  const db=await openDB('corrupt-race');const raw=await db.get('drafts',d.id);raw.json='{damaged';await db.put('drafts',raw);release();
  await expect(writing).rejects.toThrow('DRAFT_CHANGED');expect(await db.get('drafts',d.id)).toEqual(raw);db.close();
 }finally{spy.mockRestore();}
});
