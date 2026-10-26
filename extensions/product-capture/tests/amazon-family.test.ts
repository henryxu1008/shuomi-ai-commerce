import { JSDOM } from 'jsdom';
import { expect, it, vi } from 'vitest';
import { discoverFamily } from '../src/amazon/family-discovery';
import { FamilyCoordinator } from '../src/amazon/family-coordinator';
import type { AmazonCapturePayload } from '../src/amazon/wire';
import { captureAmazonDocument } from '../src/amazon/extractor';

const anchor = 'B0BDLGZCTY', sibling = 'B0CC1YM6SJ', parent = 'B0D76YXPWZ';
export const familyData = () => ({ currentAsin: anchor, parentAsin: parent, dimensions: ['color_name','size_name'], variationDisplayLabels: { color_name:'Color',size_name:'Size' }, variationValues: { color_name:['Grey','Beige'],size_name:['XL'] }, dimensionToAsinMap: { '0_0':anchor,'1_0':sibling }, dimensionValuesDisplayData: { [anchor]:['Grey','XL'],[sibling]:['Beige','XL'] } });
const document = (id=anchor, color='Grey', amount='39.98', currency='US$') => new JSDOM(`<span id="productTitle">Dog bed</span><input id="ASIN" value="${id}"><img id="landingImage" src="https://m.media-amazon.com/images/I/${id}.jpg"><div id="variation_color"><span class="a-form-label">Color:</span><span class="selection">${color}</span></div><div id="variation_size"><span class="a-form-label">Size:</span><span class="selection">XL</span></div><div id="corePrice_feature_div"><span class="a-price"><span class="a-offscreen">${currency}${amount}</span></span></div>`,{url:`https://www.amazon.com/dp/${id}`}).window.document;
const seed = async () => { const doc=document(); doc.body.insertAdjacentHTML('beforeend',`<script>P.register('twister-js-init-dpx-data',function(){return ${JSON.stringify(familyData())};});</script>`); return { manifest:discoverFamily(doc,doc.URL),anchor:await captureAmazonDocument(doc,doc.URL) }; };

it('binds parent/current/real combinations within the twister initialization object',async()=>{
  const {manifest}=await seed(); expect(manifest.parentASIN).toBe(parent);
  expect(manifest.members.map(v=>v.sourceID)).toEqual([anchor,sibling]);
  expect(manifest.members[1].attributes).toEqual([{name:'Color',value:'Beige'},{name:'Size',value:'XL'}]);
});
it.each(['absent','empty','conflicting'])('uses the same-object currentAsin when the optional ASIN input is %s',kind=>{
 const doc=document(),input=doc.querySelector<HTMLInputElement>('#ASIN')!;
 if(kind==='absent')input.remove();else input.value=kind==='empty'?'':sibling;
 doc.body.insertAdjacentHTML('beforeend',`<script>P.register('twister-js-init-dpx-data',function(){return ${JSON.stringify(familyData())};});</script>`);
 if(kind==='conflicting')expect(()=>discoverFamily(doc,doc.URL)).toThrow('SOURCE_CHANGED');else expect(discoverFamily(doc,doc.URL).members).toHaveLength(2);
});
it.each(['<div class="recommendations"></div>','<form></form>','<div hidden></div>','<div aria-hidden="true"></div>','<div style="display:none"></div>','<a href="/dp/B000000000"></a>','<div data-cel-widget="comparison"></div>'])('rejects candidate scripts with an excluded product ancestor: %s',wrapper=>{
 const doc=document(),host=doc.createElement('div');host.innerHTML=wrapper;
 const script=doc.createElement('script');script.textContent=`P.register('twister-js-init-dpx-data',function(){return ${JSON.stringify(familyData())};});`;
 host.firstElementChild!.append(script);doc.body.append(host);expect(()=>discoverFamily(doc,doc.URL)).toThrow('FAMILY_NOT_AVAILABLE');
});
it.each(['recommendation','split','conflict','missing','dynamic','cartesian'])('refuses unproven family metadata: %s',kind=>{
  const doc=document(), data=familyData(); let source=JSON.stringify(data);
  if(kind==='missing')source=JSON.stringify({...data,parentAsin:null});
  if(kind==='dynamic')source=source.replace('"B0D76YXPWZ"','getParent()');
  if(kind==='conflict')source=JSON.stringify({...data,dimensionToAsinMap:{'0_0':anchor,'1_0':anchor}});
  if(kind==='cartesian')source=JSON.stringify({...data,dimensionToAsinMap:{'0_0':anchor},dimensionValuesDisplayData:{[anchor]:['Grey','XL'],[sibling]:['Beige','XL']}});
  const code=kind==='split'?`const a={currentAsin:'${anchor}',parentAsin:'${parent}'};const b=${JSON.stringify({dimensions:data.dimensions,variationValues:data.variationValues,variationDisplayLabels:data.variationDisplayLabels,dimensionToAsinMap:data.dimensionToAsinMap})};`:`return ${source};`;
  doc.body.insertAdjacentHTML('beforeend',`<script>P.register('${kind==='recommendation'?'recommendation':'twister-js-init-dpx-data'}',function(){${code}});</script>`);
  expect(()=>discoverFamily(doc,doc.URL)).toThrow();
});
it('does not manufacture all color/size combinations or charge unrelated scripts',async()=>{
  const doc=document();const data=familyData(); data.variationValues.size_name.push('M');
  doc.body.insertAdjacentHTML('beforeend',`<script>${'x'.repeat(2100000)}</script><script>P.register('twister-js-init-dpx-data',function(){return ${JSON.stringify(data)};});</script>`);
  expect(discoverFamily(doc,doc.URL).members).toHaveLength(2);
});
it('parses a bounded 171-member source map with the observed 19-color/9-size shape',()=>{
 const data={...familyData(),variationValues:{color_name:Array.from({length:19},(_,i)=>`Color ${i}`),size_name:Array.from({length:9},(_,i)=>`Size ${i}`)},dimensionToAsinMap:{} as Record<string,string>,dimensionValuesDisplayData:{} as Record<string,string[]>};
 for(let n=0;n<171;n++){const color=Math.floor(n/9),size=n%9,id=n===0?anchor:`B${String(n).padStart(9,'0')}`;data.dimensionToAsinMap[`${color}_${size}`]=id;data.dimensionValuesDisplayData[id]=[data.variationValues.color_name[color],data.variationValues.size_name[size]];}
 const doc=document();doc.body.insertAdjacentHTML('beforeend',`<script>P.register('twister-js-init-dpx-data',function(){return ${JSON.stringify(data)};});</script>`);
 const result=discoverFamily(doc,doc.URL);expect(result.members).toHaveLength(171);expect(new Set(result.members.map(m=>m.sourceID)).size).toBe(171);
 expect(result.members[170].attributes).toEqual([{name:'Color',value:'Color 18'},{name:'Size',value:'Size 8'}]);
});
it('reads each child once and freezes one partial-family handoff with its own price and image',async()=>{
  const first=await seed(), read=vi.fn(async()=>{const doc=document(sibling,'Beige','44.98');return {capture:await captureAmazonDocument(doc,doc.URL),parentASIN:parent};});
  const handoff=vi.fn<(payload: AmazonCapturePayload) => Promise<void>>(async()=>{});const c=new FamilyCoordinator({seed:async()=>first,read,handoff,close:vi.fn(async()=>{})});
  await Promise.all([c.start(),c.start()]);expect(read).toHaveBeenCalledTimes(1);expect(handoff).toHaveBeenCalledTimes(1);
  const output=handoff.mock.calls[0][0];expect(output.evidence.observation).toEqual({scope:'partial_family',parentASIN:parent});
  expect(output.evidence.variants[1].price?.amount).toBe('44.98');expect(output.evidence.variants[1].images[0].url).toContain(sibling);
  expect(output.evidence.priceFacts[0].amount).toBe('39.98');expect(output.evidence.contentSHA256).not.toBe(first.anchor.evidence.contentSHA256);
  await c.resume();await c.useObserved();expect(handoff).toHaveBeenCalledTimes(1);
});
it('pauses on a child error, never skips or retries automatically, permits explicit partial use',async()=>{
  const handoff=vi.fn<(payload: AmazonCapturePayload) => Promise<void>>(async()=>{}),read=vi.fn(async()=>{throw Error('CAPTURE_REJECTED');});
  const c=new FamilyCoordinator({seed,read,handoff,close:vi.fn(async()=>{})});await c.start();
  expect(c.snapshot().phase).toBe('paused');expect(read).toHaveBeenCalledTimes(1);expect(handoff).not.toHaveBeenCalled();
  await c.useObserved();expect(handoff).toHaveBeenCalledTimes(1);expect(handoff.mock.calls[0][0].evidence.variants).toHaveLength(1);
  expect(handoff.mock.calls[0][0].evidence.missingFacts.some(f=>f.reason.includes(sibling))).toBe(true);
});
it('cancellation discards an in-flight child and prevents late handoff',async()=>{
  let finish!: (v: {capture: AmazonCapturePayload; parentASIN: string|null}) => void;
  const handoff=vi.fn<(payload: AmazonCapturePayload) => Promise<void>>(async()=>{});const c=new FamilyCoordinator({seed,read:()=>new Promise(resolve=>{finish=resolve;}),handoff,close:vi.fn(async()=>{})});
  const running=c.start();await vi.waitFor(()=>expect(finish).toBeTypeOf('function'));await c.cancel();
  const doc=document(sibling,'Beige');finish({capture:await captureAmazonDocument(doc,doc.URL),parentASIN:parent});await running;
  expect(c.snapshot().phase).toBe('cancelled');expect(handoff).not.toHaveBeenCalled();
});
it('locks forever after a lost handoff response, without resubmission',async()=>{
  const handoff=vi.fn(async()=>{throw Error('HANDOFF_UNAVAILABLE');});const c=new FamilyCoordinator({seed,read:async()=>{const doc=document(sibling,'Beige');return {capture:await captureAmazonDocument(doc,doc.URL),parentASIN:parent};},handoff,close:vi.fn(async()=>{})});
  await c.start();expect(c.snapshot().phase).toBe('uncertain');await c.resume();await c.useObserved();await c.cancel();expect(handoff).toHaveBeenCalledTimes(1);
});
it('aborts the reader before cancellation cleanup and never continues it afterwards',async()=>{
 let readerSignal:AbortSignal|undefined;let entered!:()=>void;const started=new Promise<void>(r=>{entered=r;});
 const c=new FamilyCoordinator({seed,read:async(_member,signal)=>{readerSignal=signal;entered();await new Promise<void>((_resolve,reject)=>signal.addEventListener('abort',()=>reject(Error('cancelled'))));throw Error('unreachable');},handoff:vi.fn(async()=>{}),close:async()=>{expect(readerSignal?.aborted).toBe(true);}});
 const running=c.start();await started;await c.cancel();await running;expect(c.snapshot().phase).toBe('cancelled');
});
it('pauses at the existing aggregate budget without truncating facts or handing off automatically',async()=>{
 const initial=await seed();initial.manifest.members=[initial.manifest.members[0],...Array.from({length:255},(_,n)=>({sourceID:`B0${String(n).padStart(8,'0')}`,attributes:[{name:'Color',value:`c${n}`},{name:'Size',value:'XL'}]}))];
 const handoff=vi.fn<(payload:AmazonCapturePayload)=>Promise<void>>(async()=>{});
 const read=vi.fn(async(m:{sourceID:string;attributes:{name:string;value:string}[]})=>{const capture=structuredClone(initial.anchor);capture.evidence.sourceID=m.sourceID;capture.evidence.sourceURL=`https://www.amazon.com/dp/${m.sourceID}`;capture.evidence.variants[0]={...capture.evidence.variants[0],sourceID:m.sourceID,attributes:m.attributes,images:[{role:'main',url:`https://m.media-amazon.com/images/I/${m.sourceID}.jpg`}]};return {capture,parentASIN:initial.manifest.parentASIN};});
 const c=new FamilyCoordinator({seed:async()=>initial,read,handoff,close:vi.fn(async()=>{})});await c.start();
 expect(c.snapshot()).toMatchObject({phase:'paused',reason:'FAMILY_LIMIT_REACHED'});expect(c.snapshot().observed).toBeLessThan(256);expect(handoff).not.toHaveBeenCalled();
 const calls=read.mock.calls.length;await c.resume();expect(read).toHaveBeenCalledTimes(calls);await c.useObserved();expect(handoff).toHaveBeenCalledOnce();
});
it.each(['currency','price'])('keeps 171 unknown %s facts attributable and foldable by the existing Go review mapper',async kind=>{
 const initial=await seed(),reason=kind==='currency'?'source currency not explicit':'source price not observed',field=kind==='currency'?'price.currency':'price';
 initial.anchor.evidence.variants[0].price=kind==='currency'?{amount:'39.98',currency:null}:null;
 initial.anchor.evidence.missingFacts.push({field,reason});
 initial.manifest.members=[initial.manifest.members[0],...Array.from({length:170},(_,n)=>({sourceID:`B${String(n).padStart(9,'0')}`,attributes:[{name:'Color',value:`c${n}`},{name:'Size',value:'XL'}]}))];
 const handoff=vi.fn<(p:AmazonCapturePayload)=>Promise<void>>(async()=>{});
 const c=new FamilyCoordinator({seed:async()=>initial,read:async m=>{const capture=structuredClone(initial.anchor);capture.evidence.sourceID=m.sourceID;capture.evidence.sourceURL=`https://www.amazon.com/dp/${m.sourceID}`;capture.evidence.variants[0].sourceID=m.sourceID;capture.evidence.variants[0].attributes=m.attributes;capture.evidence.missingFacts=[{field,reason}];return {capture,parentASIN:initial.manifest.parentASIN};},handoff,close:async()=>{}});
 await c.start();expect(c.snapshot()).toMatchObject({phase:'transferred',observed:171});
 const payload=handoff.mock.calls[0][0];expect(new Set(payload.evidence.missingFacts.map(f=>f.reason)).size).toBeLessThan(128);
 expect(payload.evidence.missingFacts).toContainEqual({field:`variants.170.${field}`,reason});
 expect(payload.evidence.variants.every(v=>kind==='currency'?v.price?.currency===null:v.price===null)).toBe(true);
});
