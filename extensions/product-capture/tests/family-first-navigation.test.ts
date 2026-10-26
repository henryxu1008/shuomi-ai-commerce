import { readFileSync } from 'node:fs';
import { JSDOM } from 'jsdom';
import { afterEach, expect, it, vi } from 'vitest';
import { mountFamilyPage } from '../src/amazon/family-page';
import { FamilyBridge } from '../src/amazon/family-bridge';
import { captureAmazonDocument } from '../src/amazon/extractor';
import { amazonInterruptionURL } from '../src/amazon/page-state';
import { Controller } from '../src/controller';
import { ReturnSession } from '../src/return-session';
import { validatePlatformCapture } from '../src/platforms';
import { returnStorage } from './fixtures/return-storage';
import type { AmazonCapturePayload } from '../src/amazon/wire';

const id='a'.repeat(32),source='https://www.amazon.com/dp/B0BDLGZCTY',target='https://www.amazon.com/dp/B0CC1YM6SJ';
const checkpoint='https://www.amazon.com/checkpoint/continue?token=DO_NOT_FORWARD';
let dom:JSDOM;
afterEach(()=>{dom?.window.dispatchEvent(new dom.window.Event('pagehide'));dom?.window.close();vi.unstubAllGlobals();});

async function setup(options:{first?:'ready'|'omitted'|'empty'|'pending'|'verification'|'signin';loseReply?:boolean;creation?:'reject'|'no-id';native?:Partial<chrome.tabs.Tab>}={}) {
 const original=new JSDOM(readFileSync('tests/fixtures/amazon-inline-product.html','utf8'),{url:source});
 const anchor=await captureAmazonDocument(original.window.document,source);original.window.close();
 const other=structuredClone(anchor);other.evidence.sourceID='B0CC1YM6SJ';other.evidence.sourceURL=target;other.evidence.variants[0].sourceID='B0CC1YM6SJ';
 other.evidence.variants[0].attributes=anchor.evidence.variants[0].attributes.map(a=>a.name==='Color'?{...a,value:'Beige'}:a);
 other.evidence.variants[0].price={amount:'44.98',currency:'USD'};other.evidence.variants[0].images=[{role:'main',url:'https://m.media-amazon.com/images/I/child.jpg'}];
 const manifest={sourceURL:source,parentASIN:'B0D76YXPWZ',members:[anchor,other].map(p=>({sourceID:p.evidence.sourceID,attributes:p.evidence.variants[0].attributes}))};
 const revision=crypto.randomUUID(),progress=`chrome-extension://${id}/popup.html#amazon-family=${revision}`;
 dom=new JSDOM('<body/>',{url:progress});
 const storage=returnStorage(),app=new ReturnSession(storage.session as unknown as chrome.storage.SessionStorageArea);await app.ready;
 const receiver=vi.fn(async()=>42),activate=vi.fn(async()=>{});
 const controller=new Controller('https://app.example.com/capture/amazon',async()=>{throw Error('must not extract through ordinary entry');},receiver,()=>id,validatePlatformCapture,()=> 'https://app.example.com/capture/amazon',activate,{prepare:s=>app.prepare(s,revision),bind:n=>app.bind(n,revision),tabID:()=>app.tabID()});
 const contexts=[{contextType:'TAB',documentId:'progress-doc',documentUrl:progress,tabId:41,frameId:0,windowId:1}];
 const updated=new Set<(tabId:number)=>void>(),removed=new Set<(tabId:number)=>void>();
 let live:Partial<chrome.tabs.Tab>={id:50,windowId:1,openerTabId:41,status:'complete'},bridge:FamilyBridge,lost=false,exists=false;
 const emit=()=>updated.forEach(fn=>fn(50));
 const create=vi.fn(async(o:chrome.tabs.CreateProperties)=>{
  if(o.url===progress)return {id:41,url:progress,windowId:1};
  exists=true;
  if(options.creation==='reject')throw Error('native creation result unavailable');
  if(options.creation==='no-id')return {windowId:1};
  live={...live,url:o.url,openerTabId:o.openerTabId,windowId:o.windowId,...options.native};
  if(options.first==='omitted')delete live.url;
  if(options.first==='empty')live.url='';
  if(options.first==='pending'){delete live.url;live.pendingUrl=target;live.status='loading';}
  if(options.first==='verification')live.url=checkpoint;
  if(options.first==='signin')live.url='https://www.amazon.com/ap/signin?private=DO_NOT_FORWARD';
  return {id:50,windowId:1}; // create responses need not expose URL metadata either.
 });
 const get=vi.fn(async(n:number)=>{if(n===41)return {id:41,url:progress,windowId:1};if(n!==50||!exists)throw Error('closed');return {...live};});
 const update=vi.fn(async(n:number,o:chrome.tabs.UpdateProperties)=>{if(o.url){live.url=o.url;live.pendingUrl=undefined;live.status='complete';queueMicrotask(emit);}return {id:n,...live};});
 const remove=vi.fn(async(n:number)=>{if(n===50){exists=false;removed.forEach(fn=>fn(n));}});
 const execute=vi.fn(async(o:{target:{tabId:number;documentIds?:string[]};files?:string[];func?:()=>unknown})=>{
  const root=o.target.tabId===7,p=root?anchor:other,documentId=root?'anchor-doc':'child-doc';
  const code=!root&&live.url?amazonInterruptionURL(live.url):undefined;
  const result=o.files?(code?{ok:false,code}:{ok:true,payload:p}):String(o.func).includes('CaptureEntry')?{ok:true,manifest:{...manifest,sourceURL:p.evidence.sourceURL}}:String(o.func).includes('=>true')||String(o.func).includes('=> true')?true:root?source:live.url;
  return [{frameId:0,documentId,result}];
 });
 const send=vi.fn(async(m:{type:string;revision:string;payload?:AmazonCapturePayload})=>{
  try{const result=await bridge.handle(m,{id,url:progress,documentId:'progress-doc'});
   if(m.type==='family.reader'&&options.loseReply&&!lost){lost=true;return {ok:false,code:'FAMILY_INTERRUPTED'};}
   return {ok:true,result};
  }catch(e){return {ok:false,code:e instanceof Error?e.message:'CAPTURE_REJECTED'};}
 });
 vi.stubGlobal('document',dom.window.document);vi.stubGlobal('location',dom.window.location);vi.stubGlobal('addEventListener',dom.window.addEventListener.bind(dom.window));
 vi.stubGlobal('chrome',{storage:{session:storage.session},runtime:{id,getURL:(p:string)=>`chrome-extension://${id}/${p}`,getContexts:vi.fn(async()=>contexts),sendMessage:send,onMessage:{addListener:vi.fn(),removeListener:vi.fn()}},
  tabs:{create,get,update,remove,onUpdated:{addListener:(f:(n:number)=>void)=>updated.add(f),removeListener:(f:(n:number)=>void)=>updated.delete(f)},onRemoved:{addListener:(f:(n:number)=>void)=>removed.add(f),removeListener:(f:(n:number)=>void)=>removed.delete(f)}},scripting:{executeScript:execute},windows:{update:vi.fn(async()=>({}))}});
 bridge=new FamilyBridge(()=>controller,app);await bridge.ready;await bridge.begin({id:7,url:source,documentId:'anchor-doc'},revision);
 mountFamilyPage(revision);
 const button=(text:string)=>[...dom.window.document.querySelectorAll('button')].find(b=>b.textContent===text)!;
 const status=()=>dom.window.document.querySelector('[role="status"]')!.textContent!;
 return {create,get,update,remove,execute,send,receiver,activate,storage,app,button,status,updated,removed,
  bridge:()=>bridge,live:()=>live,change:(value:Partial<chrome.tabs.Tab>)=>{Object.assign(live,value);emit();},
  rebuild:async()=>{bridge=new FamilyBridge(()=>controller,app);await bridge.ready;},
  readerCreates:()=>create.mock.calls.filter(([o])=>o.url!==progress),
  navigations:()=>update.mock.calls.filter(([,o])=>o.url),
  childReads:()=>execute.mock.calls.filter(([o])=>o.target.tabId===50&&o.files),
  payload:()=>send.mock.calls.find(([m])=>m.type==='family.finish')?.[0].payload};
}

it('opens the validated first child directly, hands off once and returns to the same receiver',async()=>{
 const s=await setup();await vi.waitFor(()=>expect(s.receiver).toHaveBeenCalledOnce());
 expect(s.readerCreates()).toEqual([[{url:target,active:false,openerTabId:41,windowId:1}]]);expect(s.navigations()).toHaveLength(0);
 expect(s.payload()?.evidence.variants).toHaveLength(2);expect(s.payload()!.evidence.variants[1].price?.amount).toBe('44.98');
 expect(s.payload()!.evidence.variants[1].images[0].url).toContain('child.jpg');
 s.button('返回应用核实').click();await vi.waitFor(()=>expect(s.activate).toHaveBeenCalledWith(42));expect(s.receiver).toHaveBeenCalledOnce();
});
it.each(['omitted','empty','pending'] as const)('waits for initially %s URL metadata instead of navigating a blank reader',async first=>{
 const s=await setup({first});await vi.waitFor(()=>expect(s.readerCreates()).toHaveLength(1));
 await new Promise(r=>setTimeout(r,25));expect(s.status()).not.toContain('已暂停');expect(s.childReads()).toHaveLength(0);expect(s.navigations()).toHaveLength(0);
 s.change({url:target,status:'complete',pendingUrl:undefined});await vi.waitFor(()=>expect(s.receiver).toHaveBeenCalledOnce());expect(s.readerCreates()[0][0].url).toBe(target);
 expect(s.navigations()).toHaveLength(0);expect(s.updated.size).toBe(0);expect(s.removed.size).toBe(0);
});
it.each(['verification','signin'] as const)('preserves first-page %s and only continues the same manually restored child',async first=>{
 const s=await setup({first});await vi.waitFor(()=>expect(s.status()).toContain('已暂停'));
 expect(s.status()).toContain(first==='verification'?'真人校验':'要求登录');expect(s.status()).not.toContain('DO_NOT_FORWARD');
 expect(JSON.stringify(s.storage.contents())).not.toContain('DO_NOT_FORWARD');expect(s.bridge().location.current?.phase==='bound'&&s.bridge().location.current?.reader?.sourceURL).toBe(target);
 s.button('继续采集').click();await vi.waitFor(()=>expect(s.childReads()).toHaveLength(2));expect(s.navigations()).toHaveLength(0);expect(s.receiver).not.toHaveBeenCalled();
 s.change({url:target+'?ref=manual'});s.button('继续采集').click();await vi.waitFor(()=>expect(s.receiver).toHaveBeenCalledOnce());
 expect(s.readerCreates()).toHaveLength(1);expect(s.navigations()).toHaveLength(0);
});
it('reconfirms a committed first checkpoint locator after a lost reply and worker rebuild',async()=>{
 const s=await setup({first:'verification',loseReply:true});await vi.waitFor(()=>expect(s.status()).toContain('已暂停'));expect(s.childReads()).toHaveLength(0);
 await s.rebuild();s.button('继续采集').click();await vi.waitFor(()=>expect(s.status()).toContain('真人校验'));
 expect(s.readerCreates()).toHaveLength(1);expect(s.navigations()).toHaveLength(0);
 s.change({url:target});s.button('继续采集').click();await vi.waitFor(()=>expect(s.receiver).toHaveBeenCalledOnce());
 expect(s.readerCreates()).toHaveLength(1);expect(s.navigations()).toHaveLength(0);
});
it.each(['reject','no-id'] as const)('does not create another reader after creation result is %s',async creation=>{
 const s=await setup({creation});await vi.waitFor(()=>expect(s.status()).toContain('已暂停'));
 s.button('继续采集').click();await new Promise(r=>setTimeout(r,25));expect(s.readerCreates()).toHaveLength(1);expect(s.receiver).not.toHaveBeenCalled();
 s.button('取消本次采集').click();await vi.waitFor(()=>expect(s.status()).toContain('已取消'));expect(s.remove).not.toHaveBeenCalled();
});
it.each([{openerTabId:7},{windowId:2},{url:'https://www.amazon.com/dp/B000000003'},{pendingUrl:'https://example.com/'}])('refuses changed native reader ownership or target %j',async native=>{
 const s=await setup({native});await vi.waitFor(()=>expect(s.status()).toContain('已暂停'));
 expect(s.childReads()).toHaveLength(0);expect(s.navigations()).toHaveLength(0);expect(s.receiver).not.toHaveBeenCalled();
});
it('cancels before first metadata arrives without parsing, guessed cleanup or a late handoff',async()=>{
 const s=await setup({first:'omitted'});await vi.waitFor(()=>expect(s.readerCreates()).toHaveLength(1));
 s.button('取消本次采集').click();await vi.waitFor(()=>expect(s.status()).toContain('已取消'));
 s.change({url:target});await new Promise(r=>setTimeout(r,25));expect(s.childReads()).toHaveLength(0);expect(s.receiver).not.toHaveBeenCalled();
 expect(s.remove).not.toHaveBeenCalled();expect(s.updated.size).toBe(0);expect(s.removed.size).toBe(0);
});
