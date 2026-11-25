"use client";

import {useEffect, useRef, useState} from "react";
import Image from "next/image";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {confirmShopifyDraftMedia, previewShopifyDraftMedia, type SupplyScope} from "@/lib/api/supply-chain";
import {SupplyAPIError} from "@/lib/api/supply-error";
import {collectionID} from "@/lib/contracts/product-collection";
import {
 shopifyMediaConfirmInput, shopifyMediaPreviewInput, shopifyMediaProductNumber,
 type ShopifyMediaConfirmStatus, type ShopifyMediaPreview,
} from "@/lib/contracts/shopify-media-preview";

const galleryStates = {present:"已在图库",missing:"图库中缺少",unverified:"需要核实"};
const variantStates = {same:"配图一致",different:"配图有变化",unverified:"需要核实"};
const imageRoles = {main:"主图",white_background:"白底图",gallery:"图库图"};

const savedOperation = z.strictObject({
 key:collectionID,
 input:shopifyMediaConfirmInput,
 lastReceipt:z.enum(["complete","expired"]).optional(),
});
const syncMessages = {
 pending:"Shopify 还在处理图片。可以继续原同步。",
 complete:"Shopify 草稿的图片已同步。",
 uncertain:"上次同步结果待核实。可以继续原同步。",
 expired:"原确认已过期。请重新预览，核对 Shopify 中已有的图片和规格配图。",
};

function readSavedOperation(storageKey:string,sourceId:string,storeId:string,product:string) {
 const saved = sessionStorage.getItem(storageKey);
 if (saved === null) return;
 const operation = savedOperation.parse(JSON.parse(saved));
 if (operation.input.sourceId !== sourceId || operation.input.storeId !== storeId || operation.input.product !== product) {
  throw new Error("Shopify draft differs from saved synchronization");
 }
 return operation;
}

export function ShopifyMediaPreviewPanel({scope, sourceId, storeId, storeName, disabled, canConfirm = false, initialProduct = ""}: {
 scope: SupplyScope;
 sourceId: string;
 storeId: string;
 storeName: string;
 disabled: boolean;
 canConfirm?: boolean;
 initialProduct?: string;
}) {
 const [product, setProduct] = useState(initialProduct);
 const [preview, setPreview] = useState<ShopifyMediaPreview>();
 const [error, setError] = useState("");
 const [loading, setLoading] = useState(false);
 const [syncing, setSyncing] = useState(false);
 const [hasOperation, setHasOperation] = useState(false);
 const [syncStatus, setSyncStatus] = useState<ShopifyMediaConfirmStatus>();
 const active = useRef<AbortController>(null);
 useEffect(() => {
  const current = active;
  return () => current.current?.abort();
 }, []);
 function storageKey(productNumber:string) {
  return "listingkit.shopify.media:" + JSON.stringify([scope.userId,scope.organizationId,sourceId,storeId,productNumber]);
 }
 async function load() {
  if (active.current || disabled) return;
  const input = shopifyMediaPreviewInput.safeParse({sourceId,storeId,product});
  setPreview(undefined);
  setHasOperation(false);
  setSyncStatus(undefined);
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
   if (controller.signal.aborted) return;
   setPreview(result);
   const productNumber = shopifyMediaProductNumber(result.adminUrl);
   if (canConfirm && productNumber) {
    try {
     const saved = readSavedOperation(storageKey(productNumber),sourceId,storeId,productNumber);
     setHasOperation(!!saved);
     setSyncStatus(saved?.lastReceipt);
    } catch {
     if (result.confirmationHash) setError("无法读取原同步记录，请检查浏览器的会话存储后重试。");
    }
   }
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
 async function confirm() {
  if (active.current || disabled || !canConfirm || !preview || syncStatus === "expired") return;
  const productNumber = shopifyMediaProductNumber(preview.adminUrl);
  if (!productNumber) return;
  const savedKey = storageKey(productNumber);
  setError("");
  let operation;
  try {
   operation = readSavedOperation(savedKey,sourceId,storeId,productNumber);
   if (operation?.lastReceipt === "expired") {
    setSyncStatus("expired");
    return;
   }
   if (!operation) {
    if (!preview.confirmationHash) return;
    operation = savedOperation.parse({key:crypto.randomUUID(),input:{
     sourceId,storeId,product:productNumber,confirmationHash:preview.confirmationHash,
    }});
    sessionStorage.setItem(savedKey,JSON.stringify(operation));
   }
  } catch {
   setError("无法保存同步记录，请检查浏览器的会话存储后重试。");
   return;
  }
  const controller = new AbortController();
  active.current = controller;
  setHasOperation(true);
  setSyncing(true);
  try {
   const result = await confirmShopifyDraftMedia(scope,operation.input,operation.key,controller.signal);
   if (controller.signal.aborted) return;
   setSyncStatus(result.status);
   if (result.status === "complete" || result.status === "expired") {
    try {
     sessionStorage.setItem(savedKey,JSON.stringify({...operation,lastReceipt:result.status}));
    } catch {
     setError("本次结果已收到，但浏览器无法保存它。");
    }
   }
  } catch (failure) {
   if (controller.signal.aborted) return;
   const code = failure instanceof SupplyAPIError ? failure.code : "";
   if (code === "PERMISSION_DENIED" || code === "FORBIDDEN") {
    setError("当前没有同步图片的权限。请核对店铺授权或联系管理员。");
   } else if (code === "REVISION_CONFLICT") {
    setError("草稿或采用图片已发生变化。请重新预览，核对当前图片。");
   } else if (code === "INVALID_REQUEST") {
    setError("Shopify 图片同步请求未被接受。请重新预览，核对当前草稿。");
   } else {
    setSyncStatus("uncertain");
   }
  } finally {
   active.current = null;
   if (!controller.signal.aborted) setSyncing(false);
  }
 }
 const same = !!preview?.images.length && preview.images.every(image => image.state === "present") &&
  preview.variants.every(variant => variant.state === "same");
 const images = new Map(preview?.images.map(image => [image.id,image]));
 return <div>
  <form className="my-3 flex flex-wrap items-end gap-2" onSubmit={event => {event.preventDefault(); void load();}}>
   <label className="min-w-52 flex-1 space-y-1 text-sm">
    <span>{storeName}的商品编号或 Shopify 后台链接</span>
    <Input value={product} maxLength={512} disabled={disabled || loading || syncing} onChange={event => {
     setProduct(event.target.value);
     setPreview(undefined);
     setHasOperation(false);
     setSyncStatus(undefined);
     setError("");
    }}/>
   </label>
   <Button type="submit" variant="outline" disabled={disabled || loading || syncing || !product.trim()}>
    {loading ? "正在核对图片…" : "预览图片"}
   </Button>
  </form>
  {error ? <p role="alert" className="text-sm">{error}</p> : null}
  {preview ? <div className="space-y-3 text-sm">
   <p className="font-medium">{preview.title} · Shopify 草稿</p>
   {syncStatus === "complete" ? <p>本次预览读取到图库 {preview.galleryCount} 项，工作台已采用 {preview.images.length} 张。
    可重新预览查看最新配图。</p> :
    <p role="status">{same ? "已采用图片与 Shopify 图库、变体配图一致。" : "已读取图库和规格配图。"}
     {" "}图库现有 {preview.galleryCount} 项，工作台已采用 {preview.images.length} 张。</p>}
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
   {syncing || syncStatus ? <p role="status">{syncing ? "正在同步 Shopify 草稿图片…" : syncStatus && syncMessages[syncStatus]}</p> : null}
   <div className="flex flex-wrap items-center gap-3">
    {canConfirm && (preview.confirmationHash || hasOperation) && syncStatus !== "expired" ?
     <Button type="button" disabled={disabled || syncing} onClick={() => void confirm()}>
      {syncing ? "正在同步…" : syncStatus === "complete" ? "再次核对原同步" : hasOperation ? "继续原同步" : "确认同步到 Shopify"}
     </Button> : null}
    <Button asChild variant="link" className="px-0"><a href={preview.adminUrl} target="_blank" rel="noopener noreferrer">在 Shopify 后台打开原草稿</a></Button>
   </div>
  </div> : null}
 </div>;
}
