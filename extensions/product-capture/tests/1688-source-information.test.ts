import { describe, expect, it } from 'vitest';
import { JSDOM } from 'jsdom';
import { captureDocument } from '../src/extractor';
import { validateCapture } from '../src/validation';

const source = 'https://detail.1688.com/offer/917086279832.html';
const at = new Date('2026-10-25T10:27:12Z');
const quantityName = (id: string) => `来源可订数量（规格ID ${id}）`;

// Reconstruction from verified legacy paths and exported sample facts, not a
// complete live DOM. JSDOM never executes these scripts or fetches the source.
function model() {
  return { result: { data: {
    productTitle: { fields: { title: '聚衣堂运动短裙', unit: '件', shopInfo: {
      companyName: '浦江福衣特服饰有限公司', authCompanyName: '明确认证公司名称', sellerLoginId: 'not-a-company',
    } } },
    shippingServices: { fields: { location: '浙江省金华市', unitWeight: 0 } },
    productPackInfo: { fields: { unitWeight: 0 } },
    gallery: { fields: { offerImgList: ['https://cbu01.alicdn.com/gallery.jpg'] } },
    Root: { fields: { dataJson: { tempModel: { offerId: '917086279832' }, skuModel: {
      skuProps: [{ prop: '颜色' }, { prop: '尺码' }],
      skuInfoMap: {
        '阳光黄&gt;S': { skuId: '5989607847387', price: '27.00', canBookCount: 49846 },
        '白色&gt;M': { skuId: 'provider-white-m', price: '27.00', canBookCount: '0' },
        '柔雾粉&gt;L': { skuId: 'provider-pink-l', price: '27.00', canBookCount: '9007199254740993' },
      },
    } } } },
  }, global: { globalData: { model: { offerDetail: { featureAttributes: [{ name: '货号', value: 'JYMQ070' }] } } } } } };
}
function doc(raw = JSON.stringify(model())) {
  return new JSDOM(`<script>window.context=${raw};</script>`, { url: source }).window.document;
}
async function capture(value: unknown) {
  return (await captureDocument(doc(JSON.stringify(value)), source, at)).evidence;
}

describe('1688 source information through existing product attributes', () => {
  it('captures all four facts without turning availability into a dimension, stock or price', async () => {
    const e = await capture(model());
    expect(e.attributes).toEqual(expect.arrayContaining([
      { name: '来源计价单位', value: '件' },
      { name: '来源供应商名称', value: '浦江福衣特服饰有限公司' },
      { name: '来源发货地', value: '浙江省金华市' },
      { name: quantityName('5989607847387'), value: '49846' },
      { name: quantityName('provider-white-m'), value: '0' },
      { name: quantityName('provider-pink-l'), value: '9007199254740993' },
    ]));
    expect(e.variants.map(v => [v.sourceID, v.attributes])).toEqual([
      ['5989607847387', [{ name: '颜色', value: '阳光黄' }, { name: '尺码', value: 'S' }]],
      ['provider-white-m', [{ name: '颜色', value: '白色' }, { name: '尺码', value: 'M' }]],
      ['provider-pink-l', [{ name: '颜色', value: '柔雾粉' }, { name: '尺码', value: 'L' }]],
    ]);
    expect(e.variants.every(v => v.sku === null && v.price?.currency === null && !('stock' in v))).toBe(true);
    expect(e.attributes).toContainEqual({ name: '来源包装重量（单位未提供）', value: '0' });
    expect(e.attributes).toContainEqual({ name: '来源物流重量（单位未提供）', value: '0' });
  });

  it.each(['0', '9007199254740993', '99999999999999999999'])('keeps the exact numeric lexeme %s', async literal => {
    const raw = JSON.stringify(model()).replace('"canBookCount":49846', `"canBookCount":${literal}`);
    const e = (await captureDocument(doc(raw), source, at)).evidence;
    expect(e.attributes).toContainEqual({ name: quantityName('5989607847387'), value: literal });
  });

  it.each([undefined, null, '', '  ', false, {}, [], -1, 1.5, '-1', '1.5', '1e3', '01', '4件'].map(value => ({ value })))('does not invent zero from invalid quantity $value', async ({ value }) => {
    const input = model();
    Object.assign(input.result.data.Root.fields.dataJson.skuModel.skuInfoMap['阳光黄&gt;S'], { canBookCount: value });
    const e = await capture(input);
    expect(e.attributes.some(a => a.name === quantityName('5989607847387'))).toBe(false);
    expect(e.attributes).toContainEqual({ name: quantityName('provider-white-m'), value: '0' });
    expect(e.variants[0].sourceID).toBe('5989607847387');
    expect(e.variants[0].price?.amount).toBe('27.00');
  });

  it('rejects exponent numeric syntax as quantity without dropping known product facts', async () => {
    const e = (await captureDocument(doc(JSON.stringify(model()).replace('"canBookCount":49846', '"canBookCount":1e3')), source, at)).evidence;
    expect(e.attributes.some(a => a.name === quantityName('5989607847387'))).toBe(false);
    expect(e.title).toBe('聚衣堂运动短裙');
  });

  it.each([undefined, null, '', ' ', false, {}, []].map(value => ({ value })))('omits malformed optional text $value and uses only the explicit company fallback', async ({ value }) => {
    const input = model();
    Object.assign(input.result.data.productTitle.fields, { unit: value });
    Object.assign(input.result.data.productTitle.fields.shopInfo, { companyName: value });
    Object.assign(input.result.data.shippingServices.fields, { location: value });
    const e = await capture(input);
    expect(e.attributes.some(a => a.name === '来源计价单位' || a.name === '来源发货地')).toBe(false);
    expect(e.attributes).toContainEqual({ name: '来源供应商名称', value: '明确认证公司名称' });
    expect(e.variants).toHaveLength(3);
  });

  it.each([null, false, 'not-an-object', []].map(value => ({ value })))('tolerates an invalid optional shopInfo intermediate $value', async ({ value }) => {
    const input = model(); Object.assign(input.result.data.productTitle.fields, { shopInfo: value });
    const e = await capture(input);
    expect(e.attributes.some(a => a.name === '来源供应商名称')).toBe(false);
    expect(e.attributes).toContainEqual({ name: '来源计价单位', value: '件' });
  });

  it('does not take account names or similarly named facts from other components', async () => {
    const input = model();
    Object.assign(input.result.data.productTitle.fields, { unit: undefined });
    Object.assign(input.result.data.productTitle.fields.shopInfo, { companyName: '', authCompanyName: '' });
    Object.assign(input.result.data.shippingServices.fields, { location: undefined });
    Object.assign(input.result.data, { recommendations: { fields: { unit: '套', location: '别的地点', shopInfo: { companyName: '别的公司' }, canBookCount: 888 } } });
    const e = await capture(input);
    expect(e.attributes.some(a => ['来源计价单位', '来源供应商名称', '来源发货地'].includes(a.name))).toBe(false);
    expect(JSON.stringify(e)).not.toContain('not-a-company');
    expect(JSON.stringify(e)).not.toContain('别的公司');
  });

  it('omits unassociated quantity instead of manufacturing a provider ID', async () => {
    const input = model();
    Object.assign(input.result.data.Root.fields.dataJson.skuModel.skuInfoMap['阳光黄&gt;S'], { skuId: null });
    const e = await capture(input);
    expect(e.attributes.filter(a => a.name.startsWith('来源可订数量'))).toHaveLength(2);
    expect(e.variants[0].sourceID).toBeNull();
    expect(e.missingFacts.some(f => f.field === 'variants[0].canBookCount')).toBe(true);
  });

  it.each(['来源计价单位', '来源供应商名称', '来源发货地', quantityName('5989607847387')])('omits conflicting source attribute %s', async name => {
    const input = model(); input.result.global.globalData.model.offerDetail.featureAttributes.push({ name, value: '冲突来源值' });
    const e = await capture(input);
    expect(e.attributes.some(a => a.name === name)).toBe(false);
    expect(e.missingFacts.some(f => f.field === 'attributes' && f.reason.includes('conflicting'))).toBe(true);
  });

  it('normalizes plain source text without executing HTML or changing the offer binding', async () => {
    const input = model(); input.result.data.productTitle.fields.shopInfo.companyName = '  <img onerror=private()>  商家  ';
    const e = await capture(input);
    expect(e.attributes).toContainEqual({ name: '来源供应商名称', value: '<img onerror=private()> 商家' });
    input.result.data.Root.fields.dataJson.tempModel.offerId = 'other-offer';
    await expect(capture(input)).rejects.toThrow('PAGE_CHANGED');
  });

  it('changes the evidence digest when quantity changes while preserving variant dimensions', async () => {
    const input = model(); const first = await capture(input);
    input.result.data.Root.fields.dataJson.skuModel.skuInfoMap['阳光黄&gt;S'].canBookCount = 49845;
    const changed = await capture(input);
    expect(changed.attributes).toContainEqual({ name: quantityName('5989607847387'), value: '49845' });
    expect(changed.contentSHA256).not.toBe(first.contentSHA256);
    expect(changed.variants).toEqual(first.variants);
  });

  it('preserves the public contract for sourceID-only and duplicate-dimension variants', async () => {
    const e = await capture(model());
    const sourceOnly = { ...e, variants: e.variants.map(v => ({ ...v, attributes: [] })) };
    expect(() => validateCapture({ captureVersion: 2, evidence: sourceOnly })).not.toThrow();
    const repeated = { ...e, variants: e.variants.map(v => ({ ...v, attributes: e.variants[0].attributes })) };
    expect(() => validateCapture({ captureVersion: 2, evidence: repeated })).toThrow('INVALID_CAPTURE');
  });

  it.each([250, 251])('counts all source facts against the original 256-attribute limit: %i existing', async count => {
    const input = model();
    input.result.global.globalData.model.offerDetail.featureAttributes = Array.from({ length: count - 2 }, (_, i) => ({ name: `参数${i}`, value: '原值' }));
    // The two existing raw weight attributes plus six new source attributes.
    if (count === 250) expect((await capture(input)).attributes).toHaveLength(256);
    else await expect(capture(input)).rejects.toThrow('CAPTURE_TOO_LARGE');
  });

  it('keeps the existing per-value byte limit for source quantity', async () => {
    const input = model(); input.result.data.Root.fields.dataJson.skuModel.skuInfoMap['柔雾粉&gt;L'].canBookCount = '9'.repeat(8193);
    await expect(capture(input)).rejects.toThrow('INVALID_CAPTURE');
  });
});
