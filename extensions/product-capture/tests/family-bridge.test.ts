import { testDrafts } from './fixtures/drafts';
import { beforeEach,afterEach,expect,it,vi } from 'vitest';
import { FamilyBridge } from '../src/amazon/family-bridge';
import { familyURL } from '../src/amazon/family-location';
import { ReturnSession } from '../src/return-session';
import { Controller } from '../src/controller';
import { returnStorage } from './fixtures/return-storage';
import { captureAmazonDocument } from '../src/amazon/extractor';
import { JSDOM } from 'jsdom';
import { readFileSync } from 'node:fs';
const revision=crypto.randomUUID(), source={id:7,url:'https://www.amazon.com/dp/B0BDLGZCTY',documentId:crypto.randomUUID()},documentId=crypto.randomUUID(),id='a'.repeat(32);
const storage=returnStorage();
let contexts:chrome.runtime.ExtensionContext[]=[];
let tabs:Record<number,Partial<chrome.tabs.Tab>>;
const api={storage:{session:storage.session},runtime:{id,getURL:(p:string)=>`chrome-extension://${id}/${p}`,getContexts:vi.fn(async()=>contexts),sendMessage:vi.fn(async()=>({phase:'collecting',observed:1,total:171,reason:''}))},
 tabs:{create:vi.fn(async(o:{url:string})=>{tabs[41]={id:41,url:o.url,windowId:1};return tabs[41];}),get:vi.fn(async(n:number)=>{if(!tabs[n])throw Error('gone');return tabs[n];}),update:vi.fn(async(n:number)=>tabs[n]),reload:vi.fn<(n:number)=>Promise<void>>(async()=>{}),remove:vi.fn(async(n:number)=>{delete tabs[n];})},windows:{update:vi.fn(async()=>({}))}};
let app:ReturnSession,controller:Controller,bridge:FamilyBridge,drafts:ReturnType<typeof testDrafts>;
const sender=()=>({id,url:familyURL(revision),documentId} as chrome.runtime.MessageSender);
const context=()=>({contextType:'TAB',documentId,documentUrl:familyURL(revision),tabId:41,frameId:0,windowId:1} as chrome.runtime.ExtensionContext);
beforeEach(async()=>{storage.reset();vi.clearAllMocks();tabs={};contexts=[];vi.stubGlobal('chrome',api);
 drafts=testDrafts();
 app=new ReturnSession(storage.session as unknown as chrome.storage.SessionStorageArea);await app.ready;
 // Use the platform validator, just as production assembly does.
 const {validatePlatformCapture}=await import('../src/platforms');
 controller=new Controller(drafts,async()=>source,'https://app.example.com/capture/amazon',async()=>{throw Error('must not extract');},async()=>42,()=>id,validatePlatformCapture,()=> 'https://app.example.com/capture/amazon',async n=>{await api.tabs.update(n);},{prepare:s=>app.prepare(s,revision),bind:n=>app.bind(n,revision),tabID:()=>app.tabID()});
 bridge=new FamilyBridge(()=>controller,app,drafts,'https://app.example.com/capture/amazon');await bridge.ready;
});
afterEach(()=>vi.unstubAllGlobals());
async function open(){await bridge.begin(source,revision);contexts=[context()];await bridge.handle({type:'family.init',revision},sender());}
async function capture(){const doc=new JSDOM(readFileSync('tests/fixtures/amazon-inline-product.html','utf8'),{url:source.url}).window.document;const p=await captureAmazonDocument(doc,doc.URL);p.evidence.observation={scope:'partial_family',parentASIN:'B0D76YXPWZ'};return p;}
it('persists pending before create and only binds a native progress document',async()=>{
 api.tabs.create.mockImplementationOnce(async o=>{expect(Object.values(storage.contents()).some(v=>(v as {phase:string}).phase==='pending')).toBe(true);tabs[41]={id:41,url:o.url,windowId:1};return tabs[41];});
 await bridge.begin(source,revision);expect(bridge.location.current?.phase).toBe('pending');contexts=[context()];
 const init=await bridge.handle({type:'family.init',revision},sender());expect(init).toMatchObject({source});expect(bridge.location.current?.phase).toBe('bound');
 await bridge.handle({type:'family.init',revision},sender());expect(api.tabs.create).toHaveBeenCalledOnce();
});
it.each(['popup','source','document','tab'])('refuses a forged %s finalization without handoff',async kind=>{
 await open();const bad={...sender()};if(kind==='popup')bad.url=api.runtime.getURL('popup.html');if(kind==='source')bad.url=source.url;if(kind==='document')bad.documentId=crypto.randomUUID();if(kind==='tab')bad.tab={id:999} as chrome.tabs.Tab;
 await expect(bridge.handle({type:'family.finish',revision,payload:await capture()},bad)).rejects.toThrow();expect(app.state).toBe('none');
});
it('recovers the same live progress after a worker rebuild without creating or reading another source',async()=>{
 await open();const recreated=new FamilyBridge(()=>controller,app,drafts,'https://app.example.com/capture/amazon');await recreated.ready;expect(await recreated.status()).toMatchObject({phase:'collecting',total:171});await recreated.focus();
 expect(api.tabs.create).toHaveBeenCalledOnce();expect(api.tabs.update).toHaveBeenCalledWith(41,{active:true});
});
it('fences the old document and restores the same draft paused after a progress page refresh',async()=>{
 await open();contexts=[{...context(),documentId:crypto.randomUUID()}];expect(await bridge.status()).toMatchObject({phase:'interrupted'});
 const restored=await bridge.handle({type:'family.init',revision},{...sender(),documentId:contexts[0].documentId});expect(restored).toMatchObject({draft:{phase:'paused'}});
 await expect(bridge.handle({type:'family.init',revision},sender())).rejects.toThrow();
 await bridge.abandonInterrupted();expect(bridge.active).toBe(false);
});
it('reinitializes a browser-restored progress document only after explicit restore when its session locator is gone',async()=>{
 await open();const current=(await drafts.get(revision))!;await drafts.pause(current.id,current.revision,current.generation);
 storage.reset();const restored=new FamilyBridge(()=>controller,app,drafts,'https://app.example.com/capture/amazon');await restored.ready;
 // The browser can restore this document before the user restores its locator.
 await expect(restored.handle({type:'family.init',revision},sender())).rejects.toThrow();
 expect(api.tabs.reload).not.toHaveBeenCalled();
 await restored.focus();
 expect(api.tabs.reload).toHaveBeenCalledWith(41);expect(api.tabs.create).toHaveBeenCalledOnce();
 contexts=[{...context(),documentId:crypto.randomUUID()}];
 const result=await restored.handle({type:'family.init',revision},{...sender(),documentId:contexts[0].documentId});
 expect(result).toMatchObject({draft:{phase:'paused'}});expect(restored.location.current?.phase).toBe('bound');
});
it('one finalization preserves original return, and a rebuilt worker only navigates to that app',async()=>{
 await open();const anchor=await capture();anchor.evidence.observation={scope:'selected_product',parentASIN:null};const manifest={sourceURL:source.url,parentASIN:'B0D76YXPWZ',members:[{sourceID:anchor.evidence.sourceID,attributes:anchor.evidence.variants[0].attributes}]};let d=(await drafts.get(revision))!;d=await drafts.seed(d.id,d.revision,d.generation,{anchor,manifest});await bridge.handle({type:'family.finish',revision,expectedRevision:d.revision,generation:d.generation},sender());expect(app.tabID()).toBe(42);expect(bridge.active).toBe(false);
 const rebuilt=new FamilyBridge(()=>controller,app,drafts,'https://app.example.com/capture/amazon');await rebuilt.ready;await rebuilt.handle({type:'family.return',revision},sender());expect(api.tabs.update).toHaveBeenLastCalledWith(42);
});
it('rejects a reader not created by this progress page and never closes a user-navigated tab',async()=>{
 await open();tabs[50]={id:50,url:source.url,openerTabId:7,windowId:1};await expect(bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender())).rejects.toThrow();
 tabs[50].openerTabId=41;await bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender());tabs[50].url='https://example.com/';
 await drafts.get(revision).then(d=>bridge.handle({type:'family.cancel',revision,expectedRevision:d!.revision,generation:d!.generation},sender()));expect(api.tabs.remove).not.toHaveBeenCalled();expect(bridge.active).toBe(true);expect((await drafts.get(revision))?.phase).toBe('paused');
});
it('does not treat a closed pending progress tab as still initializing',async()=>{
 await bridge.begin(source,revision);delete tabs[41];expect(await bridge.status()).toMatchObject({phase:'paused'});
});
it('retries the same second-child navigation after its locator response was lost and the worker rebuilt',async()=>{
 await open();const first='https://www.amazon.com/dp/B0CC1YM6SJ',second='https://www.amazon.com/dp/B000000002';tabs[50]={id:50,url:first,openerTabId:41,windowId:1};
 await bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:first},sender());tabs[50].url=first;
 await bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:second},sender());
 // The page did not receive that reply, so it has not navigated away from first.
 const recreated=new FamilyBridge(()=>controller,app,drafts,'https://app.example.com/capture/amazon');await recreated.ready;
 await expect(recreated.handle({type:'family.reader',revision,tabId:50,sourceURL:second},sender())).resolves.toEqual({});
 tabs[50].url='https://www.amazon.com/dp/B000000003';
 await expect(recreated.handle({type:'family.reader',revision,tabId:50,sourceURL:second},sender())).rejects.toThrow('FAMILY_INTERRUPTED');
 tabs[50].url=first;tabs[50].pendingUrl='https://example.com/';
 await expect(recreated.handle({type:'family.reader',revision,tabId:50,sourceURL:second},sender())).rejects.toThrow('FAMILY_INTERRUPTED');
 expect(api.tabs.create).toHaveBeenCalledOnce();
});
it('serializes cancellation behind an in-flight reader locator write without reviving the cancelled record',async()=>{
 await open();tabs[50]={id:50,url:source.url,openerTabId:41,windowId:1};
 let release!:()=>void;storage.session.set.mockImplementationOnce(async()=>new Promise<void>(r=>{release=r;}));
 const reader=bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender());
 await vi.waitFor(()=>expect(release).toBeTypeOf('function'));
 const cancel=drafts.get(revision).then(d=>bridge.handle({type:'family.cancel',revision,expectedRevision:d!.revision,generation:d!.generation},sender()));release();await reader;await cancel;
 expect(bridge.active).toBe(true);expect((await drafts.get(revision))?.phase).toBe('paused');expect(api.tabs.remove).toHaveBeenCalledExactlyOnceWith(50);
 const reconstructed=new FamilyBridge(()=>controller,app,drafts,'https://app.example.com/capture/amazon');await reconstructed.ready;expect(reconstructed.active).toBe(true);
});
it('allows tracking queries only when retrying the same registered child after manual handling',async()=>{
 await open();const target='https://www.amazon.com/dp/B0CC1YM6SJ';tabs[50]={id:50,url:target,openerTabId:41,windowId:1};
 await bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:target},sender());
 tabs[50].url=target+'?ref=manual';
 await expect(bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:target},sender())).resolves.toEqual({});
 for(const url of ['https://www.amazon.com/dp/B000000003','https://example.com/']){
  tabs[50].url=url;await expect(bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:target},sender())).rejects.toThrow('FAMILY_INTERRUPTED');
 }
 tabs[50].url=target+'?ref=manual';tabs[50].pendingUrl='https://example.com/';
 await expect(bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:target},sender())).rejects.toThrow('FAMILY_INTERRUPTED');
});
it('advances from an already registered child with tracking query without accepting a different current ASIN',async()=>{
 await open();const first='https://www.amazon.com/dp/B0CC1YM6SJ',next='https://www.amazon.com/dp/B000000002';
 tabs[50]={id:50,url:first,openerTabId:41,windowId:1};await bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:first},sender());
 tabs[50].url=first+'?ref=manual';
 await expect(bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:next},sender())).resolves.toEqual({});
 expect(bridge.location.current?.phase==='bound' && bridge.location.current.reader?.sourceURL).toBe(next);
});
it.each([undefined,'','about:blank'])('never takes first %j URL metadata alone as ownership proof',async url=>{
 await open();tabs[50]={id:50,url,openerTabId:41,windowId:1};
 await expect(bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender())).rejects.toThrow('FAMILY_INTERRUPTED');
 expect(bridge.location.current?.phase==='bound'&&bridge.location.current.reader).toBeUndefined();
});
it('accepts a first matching pending product but refuses foreign current/pending pages and windows',async()=>{
 await open();tabs[50]={id:50,openerTabId:41,windowId:2,pendingUrl:source.url};
 await expect(bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender())).rejects.toThrow('FAMILY_INTERRUPTED');
 tabs[50].windowId=1;tabs[50].url='https://www.amazon.com/dp/B000000003';
 await expect(bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender())).rejects.toThrow('FAMILY_INTERRUPTED');
 delete tabs[50].url;tabs[50].pendingUrl='https://example.com/';
 await expect(bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender())).rejects.toThrow('FAMILY_INTERRUPTED');
 tabs[50].pendingUrl=source.url;await expect(bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender())).resolves.toEqual({});
});
it('keeps first interrupted registration idempotent without allowing a different reader or next target',async()=>{
 await open();tabs[50]={id:50,url:'https://www.amazon.com/ap/signin?token=PRIVATE',openerTabId:41,windowId:1};
 await bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender());
 const recreated=new FamilyBridge(()=>controller,app,drafts,'https://app.example.com/capture/amazon');await recreated.ready;
 await expect(recreated.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender())).resolves.toEqual({});
 expect(JSON.stringify(storage.contents())).not.toContain('PRIVATE');
 await expect(recreated.handle({type:'family.reader',revision,tabId:50,sourceURL:'https://www.amazon.com/dp/B000000003'},sender())).rejects.toThrow('FAMILY_INTERRUPTED');
 tabs[51]={...tabs[50],id:51};
 await expect(recreated.handle({type:'family.reader',revision,tabId:51,sourceURL:source.url},sender())).rejects.toThrow('FAMILY_INTERRUPTED');
 tabs[50].pendingUrl='https://example.com/';
 await expect(recreated.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender())).rejects.toThrow('FAMILY_INTERRUPTED');
});
