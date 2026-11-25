"use client";

import {useEffect, useRef, useState} from "react";
import Image from "next/image";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {previewShopifyDraftMedia, type SupplyScope} from "@/lib/api/supply-chain";
import {SupplyAPIError} from "@/lib/api/supply-error";
import {shopifyMediaPreviewInput, type ShopifyMediaPreview} from "@/lib/contracts/shopify-media-preview";

const galleryStates = {present:"已在图库",missing:"图库中缺少",unverified:"需要核实"};
const variantStates = {same:"配图一致",different:"配图有变化",unverified:"需要核实"};
const imageRoles = {main:"主图",white_background:"白底图",gallery:"图库图"};

export function ShopifyMediaPreviewPanel({scope, sourceId, storeId, storeName, disabled, initialProduct = ""}: {
 scope: SupplyScope;
 sourceId: string;
 storeId: string;
 storeName: string;
 disabled: boolean;
 initialProduct?: string;
}) {
 const [product, setProduct] = useState(initialProduct);
 const [preview, setPreview] = useState<ShopifyMediaPreview>();
 const [error, setError] = useState("");
 const [loading, setLoading] = useState(false);
 const active = useRef<AbortController>(null);
 useEffect(() => {
  const current = active;
  return () => current.current?.abort();
 }, []);
 async function load() {
  if (active.current || disabled) return;
  const input = shopifyMediaPreviewInput.safeParse({sourceId,storeId,product});
  setPreview(undefined);
  setError("");
  if (!input.success) {
   setError("请输入 Shopify 商品编号或该店的后台商品链接。");
   return;
  }
  const controller = new AbortController();
  active.current = controller;
  setLoading(true);
  try {
   const result = await previewShopifyDraftMedia(scope,input.data,controller.signal);
   if (!controller.signal.aborted) setPreview(result);
  } catch (error) {
   if (controller.signal.aborted) return;
   if (error instanceof SupplyAPIError && error.code === "REVISION_CONFLICT") {
    setError("无法核对这份草稿。请确认它仍是草稿，商品规格与工作台一致，再试一次。");
   } else if (error instanceof SupplyAPIError && error.code === "INVALID_REQUEST") {
    setError("请核对商品编号，或改用当前店铺的 Shopify 后台链接。");
   } else {
    setError("暂时无法读取 Shopify 草稿。请核对店铺授权，稍后重试。");
   }
  } finally {
   active.current = null;
   if (!controller.signal.aborted) setLoading(false);
  }
 }
 const same = !!preview?.images.length && preview.images.every(image => image.state === "present") &&
  preview.variants.every(variant => variant.state === "same");
 const images = new Map(preview?.images.map(image => [image.id,image]));
 return <div>
  <form className="my-3 flex flex-wrap items-end gap-2" onSubmit={event => {event.preventDefault(); void load();}}>
   <label className="min-w-52 flex-1 space-y-1 text-sm">
    <span>{storeName}的商品编号或 Shopify 后台链接</span>
    <Input value={product} maxLength={512} disabled={disabled || loading} onChange={event => {
     setProduct(event.target.value);
     setPreview(undefined);
     setError("");
    }}/>
   </label>
   <Button type="submit" variant="outline" disabled={disabled || loading || !product.trim()}>
    {loading ? "正在核对图片…" : "预览图片"}
   </Button>
  </form>
  {error ? <p role="alert" className="text-sm">{error}</p> : null}
  {preview ? <div className="space-y-3 text-sm">
   <p className="font-medium">{preview.title} · Shopify 草稿</p>
   <p role="status">{same ? "已采用图片与 Shopify 图库、变体配图一致。" : "已读取图库和规格配图。"}
    {" "}图库现有 {preview.galleryCount} 项，工作台已采用 {preview.images.length} 张。</p>
   <ul className="grid grid-cols-2 gap-2 sm:grid-cols-3">
    {preview.images.map(image => <li key={image.id} className="flex items-center gap-2">
     <Image src={image.url} alt="" width={48} height={48} unoptimized className="size-12 rounded object-cover"/>
     <span>{image.color ? image.color+" · " : ""}{imageRoles[image.role]}<br/>{galleryStates[image.state]}</span>
    </li>)}
   </ul>
   <div className="overflow-x-auto">
    <table className="w-full text-left">
     <thead><tr><th scope="col">规格 SKU</th><th scope="col">当前配图</th><th scope="col">采用主图</th><th scope="col">核对结果</th></tr></thead>
     <tbody>{preview.variants.map(variant => <tr key={variant.sku} className="border-t">
      <th scope="row" className="py-2 pr-2 font-medium">{variant.sku}</th>
      {[variant.currentAssetId,variant.assetId].map((id,index) => {
       const image = id ? images.get(id) : undefined;
       return <td key={index} className="py-2 pr-2">{image ?
        <Image src={image.url} alt={image.color ? image.color+"主图" : "商品图片"} width={36} height={36} unoptimized className="size-9 rounded object-cover"/> :
        index === 0 ? "待核实" : "未确定"}</td>;
      })}
      <td className="py-2">{variantStates[variant.state]}</td>
     </tr>)}</tbody>
    </table>
   </div>
   <Button asChild variant="link" className="px-0"><a href={preview.adminUrl} target="_blank" rel="noopener noreferrer">在 Shopify 后台打开原草稿</a></Button>
  </div> : null}
 </div>;
}
