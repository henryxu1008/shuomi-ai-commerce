"use client";

import {useEffect, useState} from "react";
import Image from "next/image";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {readSupplyPublicationStatus, shopifySupplyRules, supplyInventory, type SupplyScope} from "@/lib/api/supply-chain";
import {inventorySchema, shopifyTargetInputSchema, type ShopifySupplyTarget, type SupplyPublication, type SupplySourceDetail} from "@/lib/contracts/supply-chain";
import type {z} from "zod";
import type {ListingCommandState} from "./use-listing-command";

type ApprovedImage = z.infer<typeof inventorySchema>["assets"][number];

export function ShopifyDraftEditor({scope, source, storeId, storeName, existing, command, disabled}: {
 scope: SupplyScope;
 source: SupplySourceDetail;
 storeId: string;
 storeName: string;
 existing?: ShopifySupplyTarget;
 command: ListingCommandState;
 disabled: boolean;
}) {
 const original = source.source.source;
 const variants = source.product.variants?.length ? source.product.variants : [{sku: "", attributes: []}];
 const [currency, setCurrency] = useState("");
 const [prices, setPrices] = useState(existing?.input.shopify.prices ?? variants.map(() => ""));
 const [images, setImages] = useState<ApprovedImage[]>();
 const [selectedImages, setSelectedImages] = useState(existing?.input.shopify.imageIds ?? []);
 const [currencyError, setCurrencyError] = useState(false);
 const [imageError, setImageError] = useState(false);
 const sourceId = source.source.id;
 const revision = existing?.revision ?? 0;
 useEffect(() => {
  const controller = new AbortController();
  void shopifySupplyRules(scope, {
   sourceId, storeId, expectedRevision: revision, effectiveVersion: original.version, shopify: {currency: "", prices: []},
  }, controller.signal).then(value => {
   if (controller.signal.aborted) return;
   if (value.merchant.store_id === storeId) setCurrency(value.shopify.currency);
   else setCurrencyError(true);
  }).catch(() => {
   if (!controller.signal.aborted) setCurrencyError(true);
  });
  void supplyInventory(scope, {
   itemId: sourceId, originalPublicationId: original.publicationId, originalSnapshotVersion: original.version,
   effectiveCatalogVersion: original.version, targetPlatform: "product",
  }, controller.signal).then(value => {
   if (!controller.signal.aborted) setImages(value.assets.filter(image => image.role !== "design" && image.url.startsWith("https://")));
  }).catch(() => {
   if (!controller.signal.aborted) setImageError(true);
  });
  return () => controller.abort();
 }, [scope, sourceId, storeId, revision, original.version, original.publicationId]);
 const imageIds = selectedImages.filter(id => images?.some(image => image.id === id));
 const input = shopifyTargetInputSchema.safeParse({
  sourceId, storeId, expectedRevision: revision, effectiveVersion: original.version, shopify: {currency, prices, imageIds},
 });
 const canSave = !!source.product.title?.trim() && input.success && !disabled && (images !== undefined || imageError);
 return <form className="space-y-4" onSubmit={event => {
  event.preventDefault();
  if (canSave && input.success) void command.execute("save-target", input.data);
 }}>
  <div>
   <h3 className="font-semibold">{source.product.title || "未命名商品"}</h3>
   <p className="text-sm text-muted-foreground">{storeName} · Shopify 草稿</p>
   <p className="text-sm text-muted-foreground">保存资料后，在上传计划中核对并确认发送。草稿可在 Shopify 后台查看。</p>
  </div>
  {source.product.description ? <details className="text-sm"><summary>查看商品描述</summary>
   <p className="max-h-36 overflow-auto whitespace-pre-wrap">{source.product.description}</p>
  </details> : null}
  {currencyError ? <p role="alert">无法读取店铺币种。请检查店铺授权和服务状态后重试。</p> : null}
  {!currency && !currencyError ? <p role="status">正在读取店铺币种…</p> : null}
  {currency ? <fieldset className="space-y-2" disabled={disabled || variants.length > 30}>
   <legend className="font-medium">确认售价（{currency}）</legend>
   {variants.length > 30 ? <p>此商品有 {variants.length} 个规格，当前最多支持 30 个。</p> : variants.map((variant, index) => {
    const specs = variant.attributes?.map(attribute => `${attribute.name} ${attribute.value}`).join(" · ");
    return <label key={index} className="grid grid-cols-[minmax(0,1fr)_minmax(7ch,10ch)] items-center gap-3 text-sm">
     <span className="break-words">{variant.sku || `规格 ${index + 1}`}{specs ? ` · ${specs}` : ""}</span>
     <Input value={prices[index] ?? ""} inputMode="decimal" maxLength={17}
      onChange={event => setPrices(values => values.map((price, position) => position === index ? event.target.value : price))}/>
    </label>;
   })}
  </fieldset> : null}
  <fieldset className="space-y-2" disabled={disabled}>
   <legend className="font-medium">随商品发送的图片（可选）</legend>
   {imageError ? <p>暂时无法读取已批准图片。</p> : !images ? <p role="status">正在读取已批准图片…</p> :
    images.length === 0 ? <p>暂无可上传的已批准图片。</p> : <div className="flex flex-wrap gap-3">
     {images.map((image, index) => <label key={image.id} className="grid items-center justify-items-center gap-2 rounded border p-2">
      <Image src={image.url} alt="" width={64} height={64} unoptimized className="size-16 rounded object-cover"/>
      <input type="checkbox" aria-label={`选择图片 ${index + 1}`} checked={imageIds.includes(image.id)}
       disabled={imageIds.length >= 10 && !imageIds.includes(image.id)}
       onChange={event => setSelectedImages(event.target.checked ? [...imageIds, image.id] : imageIds.filter(id => id !== image.id))}/>
     </label>)}
    </div>}
  </fieldset>
  <Button disabled={!canSave}>保存待上传资料</Button>
 </form>;
}

const statusLabels = {DRAFT: "草稿", ACTIVE: "已启用", ARCHIVED: "已归档", UNLISTED: "未列出"};

export function ShopifyDraftResult({scope, sourceId, storeId, publication}: {
 scope: SupplyScope; sourceId: string; storeId: string; publication?: SupplyPublication;
}) {
 const [latest, setLatest] = useState<SupplyPublication>();
 const [message, setMessage] = useState("");
 const [checking, setChecking] = useState(false);
 const result = latest?.shopify ?? publication?.shopify;
 const current = latest?.shopifyStatus;
 const media = current?.media ?? result?.media ?? [];
 async function check() {
  if (checking) return;
  setChecking(true);
  setMessage("");
  try {
   const observed = await readSupplyPublicationStatus(scope, sourceId, storeId);
   setLatest(observed);
   if (observed.shopifyAttemptStatus === "not_started") setMessage("此商品还没有发往 Shopify 的记录。");
   if (observed.shopifyAttemptStatus === "outcome_unknown") setMessage("Shopify 结果仍待核实。系统会保留原操作。");
   if (observed.shopifyAttemptStatus === "claimed") setMessage("上传正在处理，请稍后查看。");
   if (observed.shopifyAttemptStatus === "not_sent") setMessage("原操作尚未发送。请核对店铺授权和资料，再确认上传。");
  } catch {
   setMessage("暂时无法核对 Shopify 结果。请检查店铺授权，或稍后再查。");
  } finally {
   setChecking(false);
  }
 }
 return <div className="mt-2 space-y-1 text-xs">
  {result ? <>
   <p className="font-medium">已创建 Shopify 草稿</p>
   <p className="break-all">商品编号 {result.id.split("/").at(-1)}</p>
   {current ? <p>当前状态 {statusLabels[current.status]}</p> : null}
   {media.some(state => state === "FAILED") ? <p>部分图片处理失败，请到 Shopify 后台查看。</p> : null}
   {media.some(state => state === "PROCESSING" || state === "UPLOADED") ? <p>图片正在处理。</p> : null}
  </> : null}
  <Button variant="outline" size="sm" onClick={() => void check()} disabled={checking}>
   {checking ? "正在查看…" : "查看 Shopify 结果"}
  </Button>
  {message ? <p role="status">{message}</p> : null}
 </div>;
}
