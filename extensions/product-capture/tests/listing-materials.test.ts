import { describe, expect, it } from 'vitest';
import { JSDOM } from 'jsdom';
import { captureAmazonDocument } from '../src/amazon/extractor';
import { captureDocument } from '../src/extractor';

const amazonURL = 'https://www.amazon.com/dp/B0BDLGZCTY';
const cdn = 'https://m.media-amazon.com/images/I/';
const main = `${cdn}61Main._AC_SL1500_.jpg`;
const thumb = `${cdn}41Thumb._AC_US40_.jpg`;
const side = `${cdn}71Side._AC_SL1500_.jpg`;
const sideThumb = `${cdn}51SideThumb._AC_US40_.jpg`;
const initial = [
  { large: `${cdn}41Thumb._AC_.jpg`, variant: 'MAIN', thumb, hiRes: main },
  { variant: 'PT01', thumb: sideThumb, large: `${cdn}51SideThumb._AC_.jpg`, hiRes: side },
  { variant: 'PT02', thumb: `${cdn}unobserved.jpg`, hiRes: `${cdn}unobserved-hd.jpg` },
];
function amazon(extra = '', metadata: string | null = `A.$.parseJSON('${JSON.stringify(initial)}')`) {
  return new JSDOM(`<span id="productTitle">Current product</span><input id="ASIN" value="B0BDLGZCTY">
    <img id="landingImage" src="${main}" data-old-hires="${main}">
    <div id="altImages"><ul><li><img src="${thumb}"></li><li><img src="${sideThumb}"></li>
    <li class="videoThumbnail"><img src="${cdn}video.jpg"></li></ul></div>
    ${metadata === null ? '' : `<script>var data = {colorImages:{initial:${metadata}}};</script>`}${extra}`,
    { url: amazonURL }).window.document;
}
const source1688 = 'https://detail.1688.com/offer/981645030344.html';
function alibaba(extra = '', weight = '"0.3500"') {
  const data = {result:{data:{
    productTitle:{fields:{title:'Current 1688 product'}},
    gallery:{fields:{offerImgList:['https://cbu01.alicdn.com/album.jpg']}},
    productPackInfo:{fields:{unitWeight:'__WEIGHT__'}}, shippingServices:{fields:{unitWeight:'0.4'}},
    Root:{fields:{dataJson:{tempModel:{offerId:'981645030344'},skuModel:{skuProps:[],skuInfoMap:{}}}}},
  },global:{globalData:{model:{offerDetail:{featureAttributes:[{name:'材质',value:'棉'}]}}}}}};
  return new JSDOM(`<script>window.context=${JSON.stringify(data).replace('"__WEIGHT__"',weight)};</script>${extra}`,
    {url:source1688}).window.document;
}

describe('listing materials through existing v2 capture', () => {
  it.each(['wrapped','array'])('uses explicit %s image data, deduplicates main and preserves selected variant image', async kind => {
    const doc = amazon('',kind === 'array' ? JSON.stringify(initial) : undefined);
    const {evidence:e} = await captureAmazonDocument(doc,amazonURL);
    expect(e.images).toEqual([{url:main,role:'main'},{url:side,role:'gallery'}]);
    expect(e.variants[0].images).toEqual([{url:main,role:'main'}]);
    expect(e.observation.scope).toBe('selected_product');
    expect(e.contentSHA256).toMatch(/^[a-f0-9]{64}$/);
  });
  it('uses the original large URL when hiRes is null without manufacturing an address',async()=>{
    const records=structuredClone(initial);records[1].hiRes=null as unknown as string;
    const {evidence:e}=await captureAmazonDocument(amazon('',JSON.stringify(records)),amazonURL);
    expect(e.images[1].url).toBe(`${cdn}51SideThumb._AC_.jpg`);
    expect(e.missingFacts.some(f=>f.field==='images.hiRes')).toBe(true);
  });
  it('ignores another product group and rejects conflicting current-product mappings',async()=>{
    const unrelated=structuredClone(initial);unrelated[0].hiRes=`${cdn}other-main.jpg`;unrelated[1].hiRes=`${cdn}other-side.jpg`;
    const doc=amazon(`<script>const other={colorImages:{initial:${JSON.stringify(unrelated)}}};</script>`);
    expect((await captureAmazonDocument(doc,amazonURL)).evidence.images[1].url).toBe(side);
    const conflict=structuredClone(initial);conflict[1].hiRes=`${cdn}conflict.jpg`;
    doc.body.insertAdjacentHTML('beforeend',`<script>const alternate={colorImages:{initial:${JSON.stringify(conflict)}}};</script>`);
    const {evidence:e}=await captureAmazonDocument(doc,amazonURL);
    expect(e.images.map(i=>i.url)).toEqual([main,thumb,sideThumb]);
    expect(e.missingFacts).toContainEqual({field:'images.hiRes',reason:'conflicting current-product image metadata'});
  });
  it.each(['missing','executable','duplicate'])('keeps known images for %s metadata without executing or guessing',async kind=>{
    const data=kind==='missing'?null:kind==='executable'?'window.executeImageData()':`A.$.parseJSON('[{"variant":"MAIN","thumb":"${thumb}","hiRes":"${main}","hiRes":"${side}"}]')`;
    const doc=amazon('',data);const {evidence:e}=await captureAmazonDocument(doc,amazonURL);
    expect(e.images.map(i=>i.url)).toEqual([main,thumb,sideThumb]);
    expect(e.missingFacts.some(f=>f.field==='images.hiRes')).toBe(true);
    expect(doc.defaultView).not.toHaveProperty('executeImageData');
  });
  it('reads technical parameters, bounded A+ paragraphs and detail images without changing variant ownership',async()=>{
    const doc=amazon(`<div id="productDescription"><p>Original description.</p><img src="${cdn}detail.jpg"></div>
      <table id="technicalSpecifications_section_1"><tr><th>Item Weight</th><td>1.50 kg</td></tr><tr><th>Country of Origin</th><td>China</td></tr></table>
      <div id="aplus"><div class="aplus-module"><p>Actual product care.</p><p>Actual product care.</p><p hidden>hidden text</p>
      <p>Safe <script>secret()</script>text</p><img src="${cdn}aplus.jpg"><form><img src="${cdn}private.jpg"></form>
      <div class="videoThumbnail"><img src="${cdn}video-cover.jpg"></div></div></div><aside><p>Unrelated prose</p><img src="${cdn}recommendation.jpg"></aside>`);
    const {evidence:e}=await captureAmazonDocument(doc,amazonURL);
    expect(e.attributes).toEqual(expect.arrayContaining([{name:'Item Weight',value:'1.50 kg'},{name:'Country of Origin',value:'China'}]));
    expect(e.description).toBe('Original description.\n\nActual product care.\nSafe text');
    expect(e.images.filter(i=>i.role==='detail')).toEqual([{url:`${cdn}detail.jpg`,role:'detail'},{url:`${cdn}aplus.jpg`,role:'detail'}]);
    expect(e.variants[0].images).toHaveLength(1);
  });
  it('retains a description and reports conflicting table parameters instead of choosing one',async()=>{
    const doc=amazon('<table id="technicalSpecifications_section_1"><tr><th>Material</th><td>Cotton</td></tr></table><table id="productDetails_techSpec_section_1"><tr><th>Material</th><td>Polyester</td></tr></table>');
    const {evidence:e}=await captureAmazonDocument(doc,amazonURL);
    expect(e.attributes.some(a=>a.name==='Material')).toBe(false);
    expect(e.missingFacts.some(f=>f.field==='attributes'&&f.reason.includes('conflicting'))).toBe(true);
  });
  it('bounds candidate script bytes before parsing',async()=>{
    const doc=amazon('',`[]`);doc.querySelector('script')!.textContent='/*colorImages'+ ' '.repeat(2*1024*1024)+'*/';
    await expect(captureAmazonDocument(doc,amazonURL)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });
  it.each(['description','images'])('bounds new Amazon %s before claiming complete material',async kind=>{
    const html=kind==='description'?`<p>${'界'.repeat(2800)}</p>`:Array.from({length:257},(_,i)=>`<img src="${cdn}detail-${i}.jpg">`).join('');
    await expect(captureAmazonDocument(amazon(`<div id="aplus"><div class="aplus-module">${html}</div></div>`),amazonURL)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });
  it('keeps 1688 original weights and exact numeric lexemes with unknown units explicit',async()=>{
    const {evidence:e}=await captureDocument(alibaba('','0.123456789012345678901'),source1688,new Date());
    expect(e.attributes).toEqual(expect.arrayContaining([
      {name:'来源包装重量（单位未提供）',value:'0.123456789012345678901'},
      {name:'来源物流重量（单位未提供）',value:'0.4'},
    ]));
    expect(e.missingFacts.some(f=>f.field==='attributes.weight.unit')).toBe(true);
  });
  it('merges 1688 description parameters and detail images while preserving original album/SKU facts',async()=>{
    const doc=alibaba('<div id="desc-lazyload-container"><p>Actual loaded description</p><table><tr><th>材质</th><td>棉</td></tr><tr><th>净重</th><td>350 g</td></tr></table><img src="https://cbu01.alicdn.com/detail.jpg"><img src="https://cbu01.alicdn.com/album.jpg"><span hidden><img src="https://cbu01.alicdn.com/hidden.jpg"></span><iframe src="https://example.org/private"></iframe></div>');
    const {evidence:e}=await captureDocument(doc,source1688,new Date());
    expect(e.attributes.filter(a=>a.name==='材质')).toEqual([{name:'材质',value:'棉'}]);
    expect(e.attributes).toContainEqual({name:'净重',value:'350 g'});
    expect(e.images).toEqual([{url:'https://cbu01.alicdn.com/album.jpg',role:'source'},{url:'https://cbu01.alicdn.com/detail.jpg',role:'detail'}]);
    expect(e.description).toContain('Actual loaded description');
    expect(e.variants).toEqual([]);
  });
  it('omits conflicting 1688 parameters and never replaces unknown weight with a default',async()=>{
    const doc=alibaba('<div id="desc-lazyload-container"><table><tr><td>材质</td><td>涤纶</td></tr></table></div>','null');
    const {evidence:e}=await captureDocument(doc,source1688,new Date());
    expect(e.attributes.some(a=>a.name==='材质'||a.name.includes('包装重量'))).toBe(false);
    expect(e.missingFacts.some(f=>f.field==='attributes'&&f.reason.includes('conflicting'))).toBe(true);
  });
});
