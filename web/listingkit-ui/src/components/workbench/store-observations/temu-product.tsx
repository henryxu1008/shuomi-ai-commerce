import type {TemuObservedProduct} from "@/lib/contracts/temu-observations";

export function TemuProductFacts({product}:{product:TemuObservedProduct}) {
 const modes={local:"本土",fully_managed:"全托管",semi_managed:"半托管"};
 return <section className="space-y-3"><h3 className="font-semibold">TEMU · {modes[product.mode]} · {product.title||"平台未提供标题"}</h3>
  <p>商品 {product.productId}{product.entity==="supplier_skc"?` · SKC ${product.id}`:""}</p>
  <p>平台原始销售状态：{product.rawStatus||"未提供"}</p>
  {product.mode==="fully_managed"?<p>消费者市场由平台分配。</p>:product.mode==="local"?<p>本土列表覆盖 Active 与 InActive 商品；创建回执不代表审核通过。</p>:<p>已绑定站点：{product.sites.map(s=>`${s.name} (${s.id})`).join("、")||"平台未提供"}</p>}
  {product.publishStatus?<p>审核代码：{product.publishStatus.status??"未提供"} · 子状态：{product.publishStatus.subStatus??"未提供"}</p>:null}
  {product.selectionStatus!==undefined?<p>平台选品状态代码：{product.selectionStatus}</p>:null}
  <p>此列表接口未提供售价或供货价。</p>
  {product.skus.map(sku=><div key={sku.id} className="space-y-1 rounded-md border p-3"><p className="font-medium">SKU {sku.id}</p>
   <p>外部编号：{sku.externalCode||"平台未提供"} · 平台编码：{sku.serialNumber||"平台未提供"}</p>
   <p>{sku.stockKind==="virtualStock"?"虚拟库存":"库存"}：{sku.stock??"平台未提供"}</p>
   <p>规格：{sku.specifications.map(s=>`${s.parentName||s.parentId}：${s.name||s.id}`).join("、")||"平台未提供"}</p>
   {sku.applyJitStatus!==undefined?<p>JIT 申请状态代码：{sku.applyJitStatus}</p>:null}
  </div>)}
 </section>;
}
