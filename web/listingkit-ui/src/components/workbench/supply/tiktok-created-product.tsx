import type {TikTokEffect} from "@/lib/contracts/tiktokshop-model";
export function TikTokCreatedProduct({value}:{value:TikTokEffect}){
 const result=value.publication;
 const status=result?.state==="submitted_for_review"?"已提交美国站审核":result?.state==="remote_draft"?"已创建美国站草稿，平台校验未通过":result?.state==="publication_failed"?"美国同步未成功，全球商品已保留":value.localSaveMode==="AS_DRAFT"?"已创建平台草稿":value.local?"已创建商品，审核状态待平台确认":value.global?"已创建全球商品，尚未确认美国刊登":"已保留图片上传回执";
 return <div className="mt-2 space-y-1 break-all text-xs text-slate-600"><p>{status}</p>{value.global?<p>全球商品：{value.global.id}</p>:null}{value.local?<><p>商品：{value.local.id}</p>{value.local.skus.map(s=><p key={s.id}>{s.seller_sku} → {s.id}</p>)}</>:null}{result?.productId?<><p>美国商品：{result.productId}</p>{result.skus.map(s=><p key={s.localSkuId}>全球 SKU {s.globalSkuId} → 美国 SKU {s.localSkuId}</p>)}</>:null}<p>已保存原结果；页面状态不代表平台审核通过或已在售。</p></div>;
}
