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
 tabs:{create:vi.fn(async(o:{url:string})=>{tabs[41]={id:41,url:o.url,windowId:1};return tabs[41];}),get:vi.fn(async(n:number)=>{if(!tabs[n])throw Error('gone');return tabs[n];}),update:vi.fn(async(n:number)=>tabs[n]),remove:vi.fn(async(n:number)=>{delete tabs[n];})},windows:{update:vi.fn(async()=>({}))}};
let app:ReturnSession,controller:Controller,bridge:FamilyBridge;
const sender=()=>({id,url:familyURL(revision),documentId} as chrome.runtime.MessageSender);
const context=()=>({contextType:'TAB',documentId,documentUrl:familyURL(revision),tabId:41,frameId:0,windowId:1} as chrome.runtime.ExtensionContext);
beforeEach(async()=>{storage.reset();vi.clearAllMocks();tabs={};contexts=[];vi.stubGlobal('chrome',api);
 app=new ReturnSession(storage.session as unknown as chrome.storage.SessionStorageArea);await app.ready;
 // Use the platform validator, just as production assembly does.
 const {validatePlatformCapture}=await import('../src/platforms');
 controller=new Controller('https://app.example.com/capture/amazon',async()=>{throw Error('must not extract');},async()=>42,()=>id,validatePlatformCapture,()=> 'https://app.example.com/capture/amazon',async n=>{await api.tabs.update(n);},{prepare:s=>app.prepare(s,revision),bind:n=>app.bind(n,revision),tabID:()=>app.tabID()});
 bridge=new FamilyBridge(()=>controller,app);await bridge.ready;
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
 await open();const recreated=new FamilyBridge(()=>controller,app);await recreated.ready;expect(await recreated.status()).toMatchObject({phase:'collecting',total:171});await recreated.focus();
 expect(api.tabs.create).toHaveBeenCalledOnce();expect(api.tabs.update).toHaveBeenCalledWith(41,{active:true});
});
it('rejects a same-url replacement progress document and allows only explicit interrupted reset',async()=>{
 await open();contexts=[{...context(),documentId:crypto.randomUUID()}];expect(await bridge.status()).toMatchObject({phase:'interrupted'});
 await expect(bridge.handle({type:'family.init',revision},{...sender(),documentId:contexts[0].documentId})).rejects.toThrow();
 await bridge.abandonInterrupted();expect(bridge.active).toBe(false);
});
it('one finalization preserves original return, and a rebuilt worker only navigates to that app',async()=>{
 await open();await bridge.handle({type:'family.finish',revision,payload:await capture()},sender());expect(app.tabID()).toBe(42);expect(bridge.active).toBe(false);
 const rebuilt=new FamilyBridge(()=>controller,app);await rebuilt.ready;await rebuilt.handle({type:'family.return',revision},sender());expect(api.tabs.update).toHaveBeenLastCalledWith(42);
});
it('rejects a reader not created by this progress page and never closes a user-navigated tab',async()=>{
 await open();tabs[50]={id:50,url:'about:blank',openerTabId:7};await expect(bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender())).rejects.toThrow();
 tabs[50].openerTabId=41;await bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender());tabs[50].url='https://example.com/';
 await bridge.handle({type:'family.cancel',revision},sender());expect(api.tabs.remove).not.toHaveBeenCalled();expect(bridge.active).toBe(false);
});
it('does not treat a closed pending progress tab as still initializing',async()=>{
 await bridge.begin(source,revision);delete tabs[41];expect(await bridge.status()).toMatchObject({phase:'interrupted'});
});
it('serializes cancellation behind an in-flight reader locator write without reviving the cancelled record',async()=>{
 await open();tabs[50]={id:50,url:'about:blank',openerTabId:41};
 let release!:()=>void;storage.session.set.mockImplementationOnce(async()=>new Promise<void>(r=>{release=r;}));
 const reader=bridge.handle({type:'family.reader',revision,tabId:50,sourceURL:source.url},sender());
 await vi.waitFor(()=>expect(release).toBeTypeOf('function'));
 const cancel=bridge.handle({type:'family.cancel',revision},sender());release();await reader;await cancel;
 expect(bridge.active).toBe(false);expect(bridge.location.current).toBeUndefined();expect(api.tabs.remove).toHaveBeenCalledExactlyOnceWith(50);
 const reconstructed=new FamilyBridge(()=>controller,app);await reconstructed.ready;expect(reconstructed.active).toBe(false);
});
