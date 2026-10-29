import { familyJournal } from './fixtures/family-journal';
import { readFileSync } from 'node:fs';
import { JSDOM } from 'jsdom';
import { afterEach,expect,it,vi } from 'vitest';
import { mountFamilyPage } from '../src/amazon/family-page';
import { captureAmazonDocument } from '../src/amazon/extractor';
import type { AmazonCapturePayload } from '../src/amazon/wire';
const id='a'.repeat(32),revision=crypto.randomUUID(), source='https://www.amazon.com/dp/B0BDLGZCTY', sibling='B0CC1YM6SJ';
let dom:JSDOM;
afterEach(()=>{dom?.window.dispatchEvent(new dom.window.Event('pagehide'));dom?.window.close();vi.unstubAllGlobals();});
it('wires the ordinary progress document through two observed variants and one final handoff',async()=>{
 const sourceDoc=new JSDOM(readFileSync('tests/fixtures/amazon-inline-product.html','utf8'),{url:source}).window.document;
 const anchor=await captureAmazonDocument(sourceDoc,source);const other=structuredClone(anchor);other.evidence.sourceID=sibling;other.evidence.sourceURL=`https://www.amazon.com/dp/${sibling}`;other.evidence.variants[0].sourceID=sibling;
 other.evidence.variants[0].attributes=anchor.evidence.variants[0].attributes.map(a=>a.name==='Color'?{...a,value:'Beige'}:a);
 other.evidence.variants[0].price={amount:'44.98',currency:'USD'};other.evidence.variants[0].images=[{role:'main',url:`https://m.media-amazon.com/images/I/${sibling}.jpg`}];
 const manifest={sourceURL:source,parentASIN:'B0D76YXPWZ',members:[anchor,other].map(c=>({sourceID:c.evidence.sourceID,attributes:c.evidence.variants[0].attributes}))};
 const docId=crypto.randomUUID(),readerDoc=crypto.randomUUID();
 dom=new JSDOM('<body></body>',{url:`chrome-extension://${id}/popup.html#amazon-family=${revision}`});
 const updated=new Set<(id:number,info:{status:string})=>void>(),removed=new Set<(id:number)=>void>();
 let liveURL='about:blank';let handed:AmazonCapturePayload|undefined;
 let reader:{tabId:number;sourceURL:string}|undefined;
 const journal=await familyJournal(revision,source);
 const send=vi.fn(async(m:{type:string;payload?:AmazonCapturePayload;tabId?:number;sourceURL?:string;[key:string]:unknown})=>{
  if(m.type==='family.init')return {ok:true,result:{source:{id:7,url:source,documentId:docId},coordinatorTabId:41,windowId:1,reader,...await journal.current()}};
  if(m.type==='family.reader')reader={tabId:m.tabId!,sourceURL:m.sourceURL!};
  if(['family.resume','family.seed','family.observe','family.pause','family.finish'].includes(m.type)){
   const result=await journal.handle(m);if(m.type==='family.finish')handed=result.payload as AmazonCapturePayload;return {ok:true,result};
  }
  return {ok:true,result:{}};
 });
 vi.stubGlobal('document',dom.window.document);vi.stubGlobal('location',dom.window.location);vi.stubGlobal('addEventListener',dom.window.addEventListener.bind(dom.window));
 vi.stubGlobal('chrome',{runtime:{id,getURL:(p:string)=>`chrome-extension://${id}/${p}`,sendMessage:send,onMessage:{addListener:vi.fn(),removeListener:vi.fn()}},
  tabs:{create:vi.fn(async(o:{url:string})=>{liveURL=o.url;return {id:50};}),get:vi.fn(async()=>({id:50,url:liveURL,status:'complete',openerTabId:41,windowId:1})),remove:vi.fn(async()=>{}),
   update:vi.fn(async(n:number,o:{url?:string})=>{if(o.url)liveURL=o.url;queueMicrotask(()=>updated.forEach(fn=>fn(n,{status:'complete'})));return {id:n};}),
   onUpdated:{addListener:(fn:(id:number,info:{status:string})=>void)=>updated.add(fn),removeListener:(fn:(id:number,info:{status:string})=>void)=>updated.delete(fn)},
   onRemoved:{addListener:(fn:(id:number)=>void)=>removed.add(fn),removeListener:(fn:(id:number)=>void)=>removed.delete(fn)}},
  scripting:{executeScript:vi.fn(async(o:{target:{tabId:number};files?:string[];func?:()=>unknown})=>{
    const a=o.target.tabId===7,p=a?anchor:other,d=a?docId:readerDoc;
    return [{frameId:0,documentId:d,result:o.files?{ok:true,payload:p}:String(o.func).includes('CaptureEntry')?{ok:true,manifest:{...manifest,sourceURL:p.evidence.sourceURL}}:p.evidence.sourceURL}];
  })}});
 mountFamilyPage(revision);
 await vi.waitFor(()=>expect(handed?.evidence.variants).toHaveLength(2));
 expect(send.mock.calls.filter(([m])=>m.type==='family.finish')).toHaveLength(1);
 expect(handed!.evidence.variants[1].price?.amount).toBe('44.98');expect(handed!.evidence.variants[1].images[0].url).toContain(sibling);
 expect(dom.window.document.body.textContent).toContain('确认保存');
 send.mockRejectedValueOnce(Error('ORIGINAL_TAB_UNAVAILABLE'));
 [...dom.window.document.querySelectorAll('button')].find(b=>b.textContent==='返回应用核实')!.click();
 await vi.waitFor(()=>expect(dom.window.document.querySelector('[role="status"]')!.textContent).toContain('操作未完成'));
});
