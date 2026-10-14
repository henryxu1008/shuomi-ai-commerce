import type {TemuCreatedGoods} from "@/lib/contracts/temu-model";

export function TemuCreatedProduct({value}:{value:TemuCreatedGoods}) {
 return <div className="mt-1 space-y-2 text-xs text-emerald-700">
  <p className="break-all">TEMU 商品：{value.local?.goodsId??value.supplier?.productId}</p>
  <p>已确认创建；审核与在售状态请查看店铺商品同步结果。</p>
  {value.local?.warnings?.map((warning,i)=><p key={i} className="text-amber-700">{warning.message} · 请前往 Seller Center 确认。</p>)}
  <details><summary className="cursor-pointer">平台 SKU / SKC 编号</summary>
   {value.local?.skuInfoList.map(sku=><p key={sku.skuId} className="break-all">{sku.outSkuSn} → SKU {sku.skuId}</p>)}
   {value.supplier?.productSkuList.map(sku=><p key={sku.productSkuId} className="break-all">{sku.extCode} → SKU {sku.productSkuId} · SKC {sku.productSkcId}</p>)}
  </details>
 </div>;
}
