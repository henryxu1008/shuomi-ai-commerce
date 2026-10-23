import { expect, it } from 'vitest';
import { JSDOM } from 'jsdom';
import { captureDocument } from '../src/extractor';
import { validateCapture } from '../src/validation';

const source = 'https://detail.1688.com/offer/981645030344.html';
const model = {result:{data:{productTitle:{fields:{title:'两个颜色'}},Root:{fields:{dataJson:{tempModel:{offerId:'981645030344'},skuModel:{
  skuProps:[{prop:'颜色',value:[{name:'蓝色',imageUrl:'https://cbu01.alicdn.com/blue.jpg'},{name:'红色',imageUrl:'https://cbu01.alicdn.com/red.jpg'}]},{prop:'尺码',value:[{name:'S'},{name:'M'}]}],
  skuInfoMap:{'蓝色&gt;S':{skuId:'blue-s'},'蓝色&gt;M':{skuId:'blue-m'},'红色&gt;S':{skuId:'red-s'}},
}}}}}}};

it('writes strict v2 and preserves explicit property-value image association across sizes', async () => {
 const doc = new JSDOM(`<script>window.context = ${JSON.stringify(model)};</script>`,{url:source}).window.document;
 const payload = await captureDocument(doc,source,new Date('2026-10-23T00:00:00Z'));
 expect(payload.captureVersion).toBe(2);
 expect(payload.evidence.schemaVersion).toBe(2);
 expect(payload.evidence.parserVersion).toBe('1688-browser-dom/v3');
 expect(payload.evidence.variants.map(v=>'images' in v ? v.images : undefined)).toEqual([
  [{url:'https://cbu01.alicdn.com/blue.jpg',role:'source'}],
  [{url:'https://cbu01.alicdn.com/blue.jpg',role:'source'}],
  [{url:'https://cbu01.alicdn.com/red.jpg',role:'source'}],
 ]);
 expect(()=>validateCapture({...payload,captureVersion:1})).toThrow();
});

it('does not fill missing variant images from the product album', async () => {
 const m = structuredClone(model);
 delete (m.result.data.Root.fields.dataJson.skuModel.skuProps[0].value[0] as {imageUrl?:string}).imageUrl;
 const doc = new JSDOM(`<script>window.context = ${JSON.stringify(m)};</script>`,{url:source}).window.document;
 const payload = await captureDocument(doc,source,new Date());
 expect(payload.evidence.variants[0]).toHaveProperty('images',[]);
 expect(payload.evidence.missingFacts).toContainEqual({field:'variants[0].images',reason:'not_observed'});
});
