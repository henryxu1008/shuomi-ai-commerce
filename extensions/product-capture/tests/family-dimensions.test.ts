import { JSDOM } from 'jsdom';
import { afterEach, expect, it, vi } from 'vitest';
import { captureAmazonDocument } from '../src/amazon/extractor';
import { discoverFamily, type FamilyManifest } from '../src/amazon/family-discovery';
import { extractFamilyTab } from '../src/capture-tab';
import type { AmazonCapturePayload } from '../src/amazon/wire';

const anchor='B0BDLGZCTY',child='B0D4DGJ47V',parent='B0D76YXPWZ';
const source=`https://www.amazon.com/dp/${child}`;
const expected=[{name:'Color',value:'Graige'},{name:'Size',value:'52.0"l X 41.0"w X 8.0"th'}];
const windows:JSDOM[]=[];
afterEach(()=>{windows.splice(0).forEach(d=>d.window.close());vi.restoreAllMocks();vi.unstubAllGlobals();});
function page(color='Select Color',size='Select Size',extra='') {
 const data={currentAsin:child,parentAsin:parent,dimensions:['color_name','size_name'],variationDisplayLabels:{color_name:'Color',size_name:'Size'},variationValues:{color_name:['Grey','Graige'],size_name:['XL',expected[1].value]},dimensionToAsinMap:{'0_0':anchor,'1_1':child},dimensionValuesDisplayData:{[anchor]:['Grey','XL'],[child]:expected.map(a=>a.value)},deletedLandingAsinInfo:{asin:child,dimValues:['Please Select','Please Select']}};
 const dom=new JSDOM(`<body><span id="productTitle">Current child dog bed</span><input id="ASIN" value="${child}"><img id="landingImage" src="https://m.media-amazon.com/images/I/child-graige.jpg"><div id="inline-twister-expander-header-color_name"><span>Color:</span><span class="inline-twister-dim-title-value">${color}</span></div><div id="inline-twister-expander-header-size_name"><span>Size:</span><span class="inline-twister-dim-title-value">${size}</span></div><div id="corePrice_feature_div"><span class="a-price"><span class="a-offscreen">US$81.89</span></span></div>${extra}<script>P.register('twister-js-init-dpx-data',function(){return ${JSON.stringify(data)};});</script></body>`,{url:source});
 windows.push(dom);return dom.window.document;
}
function reader(doc:Document,modify?:(m:FamilyManifest)=>FamilyManifest|null) {
 let nativeDocument='child-doc',raw:AmazonCapturePayload|undefined;
 const execute=vi.fn(async(o:{target:{tabId:number;documentIds?:string[]};files?:string[];func?:()=>unknown})=>{
  let result:unknown;
  if(o.files){raw=await captureAmazonDocument(doc,doc.URL);result={ok:true,payload:raw};}
  else if(String(o.func).includes('CaptureEntry')){
   const own=discoverFamily(doc,doc.URL),manifest=modify?modify(own):own;
   result=manifest?{ok:true,manifest}:{ok:false,code:'FAMILY_NOT_AVAILABLE'};
  }else result=doc.URL;
  return [{frameId:0,documentId:nativeDocument,result}];
 });
 vi.stubGlobal('chrome',{scripting:{executeScript:execute}});
 return {read:(signal?:AbortSignal)=>extractFamilyTab({id:50,url:source},false,signal),raw:()=>raw!,replaceDocument:()=>{nativeDocument='replacement-doc';},execute};
}
it.each(['Select Color',' select  color ','Please Select'])('omits the exact rendered placeholder %j in default capture without discovering a family',async placeholder=>{
 const doc=page(placeholder);doc.querySelector('script')!.textContent='invalid family script';
 const p=await captureAmazonDocument(doc,source);expect(p.evidence.variants[0].attributes).toEqual([]);
 expect(p.evidence.missingFacts.some(f=>f.field==='variants[0].attributes')).toBe(true);
});
it('keeps a genuine dimension value that only resembles a placeholder',async()=>{
 const p=await captureAmazonDocument(page('Select Colorway','XL'),source);
 expect(p.evidence.variants[0].attributes).toContainEqual({name:'Color',value:'Select Colorway'});
});
it('fills placeholder dimensions only from the current child own bound data, retaining its quote, image and accurate digest',async()=>{
 const r=reader(page()),result=await r.read();
 expect(result.capture.evidence.variants[0].attributes).toEqual(expected);
 expect(result.capture.evidence.variants[0].price).toEqual({amount:'81.89',currency:'USD'});
 expect(result.capture.evidence.variants[0].images[0].url).toContain('child-graige');
 expect(result.capture.evidence.missingFacts).toContainEqual({field:'variants[0].attributes.rendered',reason:expect.stringMatching(/current.ASIN.*static|static.*current.ASIN/i)});
 const evidence=structuredClone(result.capture.evidence);evidence.contentSHA256='0'.repeat(64);
 const hash=await crypto.subtle.digest('SHA-256',new TextEncoder().encode(JSON.stringify(evidence)));
 expect(result.capture.evidence.contentSHA256).toBe(Array.from(new Uint8Array(hash),b=>b.toString(16).padStart(2,'0')).join(''));
 expect(result.capture.evidence.contentSHA256).not.toBe(r.raw().evidence.contentSHA256);
 expect(r.raw().evidence.variants[0].attributes).toEqual([]);
});
it('retains a real matching value and fills only the missing dimension',async()=>{
 expect((await reader(page('Graige')).read()).capture.evidence.variants[0].attributes).toEqual(expected);
});
it.each(['value','omitted-conflict','extra'])('refuses a real rendered %s conflict instead of repairing it from metadata',async kind=>{
 const extra=kind==='omitted-conflict'?'<div id="variation_color"><span class="a-form-label">Color:</span><span class="selection">Beige</span></div>':kind==='extra'?'<div id="variation_style"><span class="a-form-label">Style:</span><span class="selection">Other</span></div>':'';
 const doc=page(kind==='value'?'Beige':'Graige',expected[1].value,extra);
 await expect(reader(doc).read()).rejects.toThrow('SOURCE_CHANGED');
});
it.each(['root','missing-current','duplicate-current','placeholder-value'])('refuses %s metadata as evidence for this child',async kind=>{
 const r=reader(page(),m=>{
  if(kind==='root')m.sourceURL=`https://www.amazon.com/dp/${anchor}`;
  if(kind==='missing-current')m.members=m.members.filter(v=>v.sourceID!==child);
  if(kind==='duplicate-current')m.members.push(structuredClone(m.members.find(v=>v.sourceID===child)!));
  if(kind==='placeholder-value')m.members.find(v=>v.sourceID===child)!.attributes[0].value='Select Color';
  return m;
 });await expect(r.read()).rejects.toThrow('SOURCE_CHANGED');
});
it('leaves missing dimensions unfilled when no current-document family exists',async()=>{
 const result=await reader(page(),()=>null).read();expect(result.manifest).toBeUndefined();
 expect(result.capture.evidence.variants[0].attributes).toEqual([]);
});
it.each(['document','cancel'])('rejects %s drift after asynchronous dimension digest calculation',async kind=>{
 const r=reader(page()),abort=new AbortController(),digest=crypto.subtle.digest.bind(crypto.subtle);let calls=0;
 vi.spyOn(crypto.subtle,'digest').mockImplementation(async(algorithm,data)=>{const value=await digest(algorithm,data);if(++calls===2){if(kind==='document')r.replaceDocument();else abort.abort();}return value;});
 if(kind==='document')await expect(r.read(abort.signal)).rejects.toThrow('PAGE_CHANGED');
 else await expect(r.read(abort.signal)).rejects.toThrow(/abort/i);
});
