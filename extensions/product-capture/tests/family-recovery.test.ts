import { readFileSync } from 'node:fs';
import { JSDOM } from 'jsdom';
import { afterEach, expect, it, vi } from 'vitest';
import { mountFamilyPage } from '../src/amazon/family-page';
import { captureAmazonDocument } from '../src/amazon/extractor';
import type { AmazonCapturePayload } from '../src/amazon/wire';

const id='a'.repeat(32), source='https://www.amazon.com/dp/B0BDLGZCTY', sibling='B0CC1YM6SJ', target=`https://www.amazon.com/dp/${sibling}`;
let dom:JSDOM;
afterEach(()=>{dom?.window.dispatchEvent(new dom.window.Event('pagehide'));dom?.window.close();vi.unstubAllGlobals();});
async function setup(staleComplete=false) {
 const original=new JSDOM(readFileSync('tests/fixtures/amazon-inline-product.html','utf8'),{url:source});
 const anchor=await captureAmazonDocument(original.window.document,source);original.window.close();
 const other=structuredClone(anchor);other.evidence.sourceID=sibling;other.evidence.sourceURL=target;other.evidence.variants[0].sourceID=sibling;
 other.evidence.variants[0].attributes=anchor.evidence.variants[0].attributes.map(a=>a.name==='Color'?{...a,value:'Beige'}:a);
 other.evidence.variants[0].price={amount:'44.98',currency:'USD'};other.evidence.variants[0].images=[{role:'main',url:`https://m.media-amazon.com/images/I/${sibling}.jpg`}];
 const manifest={sourceURL:source,parentASIN:'B0D76YXPWZ',members:[anchor,other].map(p=>({sourceID:p.evidence.sourceID,attributes:p.evidence.variants[0].attributes}))};
 const revision=crypto.randomUUID();dom=new JSDOM('<body/>',{url:`chrome-extension://${id}/popup.html#amazon-family=${revision}`});
 const updated=new Set<(id:number,info:{status:string})=>void>(), removed=new Set<(id:number)=>void>();
 const live={url:'about:blank',status:'complete',openerTabId:41,windowId:1,pendingUrl:undefined as string|undefined};
 let handed:AmazonCapturePayload|undefined,childParses=0;
 const send=vi.fn(async(m:{type:string;payload?:AmazonCapturePayload})=>{
  if(m.type==='family.init')return {ok:true,result:{source:{id:7,url:source,documentId:'anchor'},coordinatorTabId:41,windowId:1}};
  if(m.type==='family.finish')handed=m.payload;
  if(m.type==='family.reader' && !live.url.startsWith(target) && !live.url.includes('/checkpoint/'))return {ok:false,code:'FAMILY_INTERRUPTED'};
  return {ok:true,result:{}};
 });
 const emit=()=>updated.forEach(fn=>fn(50,{status:'complete'}));
 const update=vi.fn(async(n:number,o:{url?:string})=>{
  if(o.url){if(staleComplete){emit();await new Promise<void>(r=>setTimeout(r,0));live.status='loading';live.pendingUrl=target;}
   else {live.url='https://www.amazon.com/checkpoint/continue?token=DO_NOT_FORWARD';live.status='complete';queueMicrotask(emit);}}
  return {id:n,...live};
 });
 const execute=vi.fn(async(o:{target:{tabId:number;documentIds?:string[]};files?:string[];func?:()=>unknown})=>{
  const a=o.target.tabId===7,p=a?anchor:other,d=a?'anchor':'child';
  if(o.files){if(!a)childParses++;return [{frameId:0,documentId:d,result:!a&&live.url.includes('/checkpoint/')?{ok:false,code:'AMAZON_VERIFICATION_REQUIRED'}:{ok:true,payload:p}}];}
  return [{frameId:0,documentId:d,result:String(o.func).includes('CaptureEntry')?{ok:true,manifest:{...manifest,sourceURL:p.evidence.sourceURL}}:a?source:live.url}];
 });
 vi.stubGlobal('document',dom.window.document);vi.stubGlobal('location',dom.window.location);vi.stubGlobal('addEventListener',dom.window.addEventListener.bind(dom.window));
 vi.stubGlobal('chrome',{runtime:{id,getURL:(p:string)=>`chrome-extension://${id}/${p}`,sendMessage:send,onMessage:{addListener:vi.fn(),removeListener:vi.fn()}},
  tabs:{create:vi.fn(async()=>{if(staleComplete){live.status='loading';live.pendingUrl=target;}else live.url='https://www.amazon.com/checkpoint/continue?token=DO_NOT_FORWARD';return {id:50};}),get:vi.fn(async()=>({id:50,...live})),update,remove:vi.fn(async()=>{}),
   onUpdated:{addListener:(fn:(id:number,info:{status:string})=>void)=>updated.add(fn),removeListener:(fn:(id:number,info:{status:string})=>void)=>updated.delete(fn)},
   onRemoved:{addListener:(fn:(id:number)=>void)=>removed.add(fn),removeListener:(fn:(id:number)=>void)=>removed.delete(fn)}},scripting:{executeScript:execute}});
 mountFamilyPage(revision);
 const button=(text:string)=>[...dom.window.document.querySelectorAll('button')].find(b=>b.textContent===text)!;
 const status=()=>dom.window.document.querySelector('[role="status"]')!.textContent!;
 return {live,emit,update,send,execute,button,status,handed:()=>handed,childParses:()=>childParses,updated,removed};
}
it('pauses on verification and only explicitly resumes the same ready child without navigating again',async()=>{
 const s=await setup();await vi.waitFor(()=>expect(s.status()).toContain('已暂停'));
 expect(s.status()).toContain('真人校验');expect(s.handed()).toBeUndefined();expect(s.childParses()).toBe(1);
 s.button('继续采集').click();await vi.waitFor(()=>expect(s.childParses()).toBe(2));
 await vi.waitFor(()=>expect(s.status()).toContain('已暂停'));expect(s.update.mock.calls.filter(([,o])=>o.url)).toHaveLength(0);
 s.live.url=target+'?ref=manual';s.button('继续采集').click();
 await vi.waitFor(()=>expect(s.handed()?.evidence.variants).toHaveLength(2));
 expect(s.update.mock.calls.filter(([,o])=>o.url)).toHaveLength(0);
 expect(s.send.mock.calls.filter(([m])=>m.type==='family.finish')).toHaveLength(1);
 expect(s.handed()!.evidence.variants[1].price?.amount).toBe('44.98');expect(s.handed()!.evidence.variants[1].images[0].url).toContain(sibling);
 expect(s.updated.size).toBe(0);expect(s.removed.size).toBe(0);
});
it('ignores an old blank complete event, waits for the actual target and preserves cancellation',async()=>{
 const s=await setup(true);await vi.waitFor(()=>expect(s.live.status).toBe('loading'));
 expect(s.childParses()).toBe(0);expect(s.handed()).toBeUndefined();
 s.button('取消本次采集').click();await vi.waitFor(()=>expect(s.status()).toContain('已取消'));
 s.live.url=target;s.live.status='complete';s.live.pendingUrl=undefined;s.emit();
 expect(s.childParses()).toBe(0);expect(s.handed()).toBeUndefined();expect(s.updated.size).toBe(0);expect(s.removed.size).toBe(0);
});
it('waits for a manually restored target still loading without a second navigation',async()=>{
 const s=await setup();await vi.waitFor(()=>expect(s.status()).toContain('已暂停'));
 s.live.url=target;s.live.status='loading';s.button('继续采集').click();
 await new Promise(r=>setTimeout(r,30));expect(s.childParses()).toBe(1);expect(s.update.mock.calls.filter(([,o])=>o.url)).toHaveLength(0);
 s.live.status='complete';s.emit();await vi.waitFor(()=>expect(s.handed()?.evidence.variants).toHaveLength(2));
});
