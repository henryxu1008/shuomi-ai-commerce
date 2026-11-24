"use client";

import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import {
  amazonConfirm, amazonReview, readAmazonListing, readAmazonReview, supplyInventory,
  SupplyAPIError, type SupplyScope,
} from "@/lib/api/supply-chain";
import { amazonDetailsSchema, type AmazonListingResult, type AmazonListingReview } from "@/lib/contracts/amazon-listing";
import type { SupplySourceDetail } from "@/lib/contracts/supply-chain";

type Props = {
  scope: SupplyScope;
  source: SupplySourceDetail;
  storeId: string;
  storeName: string;
  canManage: boolean;
  canSubmit: boolean;
  resultOnly: boolean;
};

type Fields = {
  productType: string;
  title: string;
  description: string;
  brand: string;
  gtin: string;
  price: string;
  quantity: string;
  mainImageId: string;
};

const labels = {
  accepted: "Amazon 已受理",
  rejected: "Amazon 未受理",
  unknown: "提交结果待确认",
  not_sent: "尚未提交到 Amazon",
};

export function AmazonListingPanel({ scope, source, storeId, storeName, canManage, canSubmit, resultOnly }: Props) {
  const sourceId = source.source.id;
  const version = source.source.source.version;
  const [fields, setFields] = useState<Fields>(() => ({
    productType: "", title: source.product.title ?? "", description: "", brand: "",
    gtin: source.product.variants?.[0]?.barcode ?? "", price: "", quantity: "", mainImageId: "",
  }));
  const [savedReview, setReview] = useState<AmazonListingReview>();
  const [revision, setRevision] = useState(0);
  const [result, setResult] = useState<AmazonListingResult>();
  const [images, setImages] = useState<Array<{ id: string; url: string }>>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [uncertain, setUncertain] = useState(false);
  const [message, setMessage] = useState("");
  const request = useRef<AbortController>(null);
  const reviewKey = useRef<{ body: string; key: string }>(null);
  const confirmKey = useRef<{ recordId: string; key: string }>(null);

  useEffect(() => {
    const controller = new AbortController();
    const optional = <T,>(promise: Promise<T>) => promise.catch(error => {
      if (error instanceof SupplyAPIError && error.code === "NOT_FOUND") return undefined;
      throw error;
    });
    void Promise.allSettled([
      optional(readAmazonListing(scope, sourceId, storeId, false, controller.signal)),
      resultOnly || !canManage ? Promise.resolve(undefined) : optional(readAmazonReview(scope, sourceId, storeId, controller.signal)),
      resultOnly || !canManage ? Promise.resolve(undefined) : optional(supplyInventory(scope, {
        itemId: sourceId, originalPublicationId: source.source.source.publicationId,
        originalSnapshotVersion: version, effectiveCatalogVersion: version, targetPlatform: "product",
      }, controller.signal)),
    ]).then(([prior, saved, inventory]) => {
      if (controller.signal.aborted) return;
      if (prior.status === "fulfilled" && prior.value) setResult(prior.value);
      if (saved.status === "fulfilled" && saved.value) {
        const { details } = saved.value;
        setFields({ ...details, description: details.description ?? "", price: (details.priceCents / 100).toFixed(2), quantity: String(details.quantity) });
        setRevision(saved.value.revision);
        if (new Date(saved.value.validUntil).getTime() > Date.now()) setReview(saved.value);
      }
      if (inventory.status === "fulfilled" && inventory.value) setImages(inventory.value.assets);
      if ([prior, saved, inventory].some(value => value.status === "rejected")) setMessage("部分资料暂时无法读取。请检查店铺授权后重新打开。");
      setLoading(false);
    });
    return () => {
      controller.abort();
      request.current?.abort();
    };
  }, [scope, sourceId, storeId, source.source.source.publicationId, version, resultOnly, canManage]);

  const review = result?.submission === "not_sent" && result.recordId === savedReview?.recordId ? undefined : savedReview;
  const canEdit = canManage && !resultOnly && !uncertain && result?.submission !== "accepted" && result?.submission !== "unknown";

  function change(key: keyof Fields, value: string) {
    setFields(prior => ({ ...prior, [key]: value }));
    setReview(undefined);
    setMessage("");
  }

  async function run(work: (signal: AbortSignal) => Promise<void>) {
    if (request.current) return;
    const active = new AbortController();
    request.current = active;
    setMessage("");
    setBusy(true);
    try {
      await work(active.signal);
    } finally {
      if (!active.signal.aborted) {
        request.current = null;
        setBusy(false);
      }
    }
  }

  async function prepare() {
    if (!canEdit) return;
    const amount = /^(\d{1,7})(?:\.(\d{1,2}))?$/.exec(fields.price.trim());
    const priceCents = amount ? Number(amount[1]) * 100 + Number((amount[2] ?? "").padEnd(2, "0")) : 0;
    const checked = amazonDetailsSchema.safeParse({
      title: fields.title.trim(), productType: fields.productType.trim(), brand: fields.brand.trim(), gtin: fields.gtin.trim(),
      ...(fields.description.trim() ? { description: fields.description.trim() } : {}),
      priceCents, quantity: /^\d{1,6}$/.test(fields.quantity.trim()) ? Number(fields.quantity) : undefined,
      mainImageId: fields.mainImageId,
    });
    if (!checked.success) {
      setMessage("请填写商品类型、名称、品牌、有效商品条码、美元售价、可卖库存，并选择已批准主图。库存为零时填写 0。");
      return;
    }
    await run(async signal => {
      const input = { sourceId, storeId, effectiveVersion: version, expectedRevision: revision, details: checked.data };
      const body = JSON.stringify(input);
      if (reviewKey.current?.body !== body) reviewKey.current = { body, key: crypto.randomUUID() };
      try {
        const saved = await amazonReview(scope, input, reviewKey.current.key, signal);
        if (!signal.aborted) {
          setReview(saved);
          setRevision(saved.revision);
          confirmKey.current = null;
        }
      } catch (error) {
        if (!signal.aborted) setMessage(error instanceof SupplyAPIError && error.code === "REVISION_CONFLICT"
          ? "资料已在其他页面更新。请重新打开商品，读取最新资料后再核对。"
          : "核对未完成。请检查条码、商品类型要求、主图及店铺授权后再核对。");
      }
    });
  }

  async function confirm() {
    if (!canEdit || !canSubmit || !review) return;
    if (new Date(review.validUntil).getTime() <= Date.now()) {
      setReview(undefined);
      setMessage("本次核对已经过期。请重新核对资料。");
      return;
    }
    const { recordId, wireHash } = review;
    await run(async signal => {
      if (confirmKey.current?.recordId !== recordId) confirmKey.current = { recordId, key: crypto.randomUUID() };
      try {
        const value = await amazonConfirm(scope, { recordId, wireHash }, confirmKey.current.key, signal);
        if (!signal.aborted) {
          setResult(value);
          setUncertain(value.submission === "unknown");
          setReview(undefined);
        }
      } catch (error) {
        if (signal.aborted) return;
        if (error instanceof SupplyAPIError && error.status < 500 && error.code !== "OUTCOME_UNKNOWN") {
          setReview(undefined);
          setMessage("未能提交。请检查当前授权、资料版本及 Amazon 商品条码占用情况，再重新核对。");
        } else {
          setUncertain(true);
          setMessage("这次提交的结果待确认。请查询原商品结果。");
        }
      }
    });
  }

  async function check() {
    await run(async signal => {
      try {
        const value = await readAmazonListing(scope, sourceId, storeId, true, signal);
        if (!signal.aborted) {
          setResult(value);
          setUncertain(value.submission === "unknown");
        }
      } catch {
        if (!signal.aborted) setMessage("暂未查到本次提交结果。请稍后再次查询。");
      }
    });
  }

  const titleID = "amazon-listing-review-title";
  return <section className="space-y-4" aria-labelledby={titleID}>
    <h3 id={titleID} className="font-semibold">{storeName} · Amazon 美国站</h3>
    <p className="text-sm text-muted-foreground">隔离测试 · 使用合成商品和平台响应</p>
    {loading ? <p role="status">正在读取商品资料和提交结果…</p> : null}
    {message ? <p role="alert">{message}</p> : null}
    {result ? <div className="space-y-2 rounded-md border p-3" aria-label="Amazon 提交结果">
      <p role="status">{labels[result.submission]}</p>
      {result.submission === "not_sent" && !resultOnly && !review ? <p>权限恢复后，请重新核对商品资料，再确认提交。</p> : null}
      <p>卖家 SKU：{result.sku}</p>
      {result.receipt ? <p>Amazon 提交编号：{result.receipt.submissionId}</p> : null}
      {result.observation ? <><p>{result.observation.asin
        ? `ASIN：${result.observation.asin} · ${result.observation.buyable ? "可购买" : "已取得 ASIN，尚不可购买"}`
        : "Amazon 尚未提供 ASIN。"}</p><p>查询时间：{new Date(result.observation.observedAt).toLocaleString("zh-CN")}</p></>
        : result.submission === "accepted" ? <p>正在等待 Amazon 处理。</p> : null}
      {result.check === "not_found" ? <p>Amazon 暂未查到该商品。</p> : null}
      {result.check === "unavailable" ? <p>暂时无法查询最新状态，请稍后再试。</p> : null}
      {[...(result.receipt?.issues ?? []), ...(result.observation?.issues ?? [])].some(issue => issue.severity === "ERROR")
        ? <p>Amazon 提示有资料需要处理。请使用卖家 SKU 在 Seller Central 查看具体要求。</p> : null}
    </div> : resultOnly && !loading ? <p>当前没有可查看的提交结果。</p> : null}
    {uncertain || result && result.submission !== "not_sent" ? <Button type="button" variant="outline" disabled={busy} onClick={() => void check()}>查询 Amazon 结果</Button> : null}
    {!loading && canEdit && !review ? <form className="grid gap-3" onSubmit={e => { e.preventDefault(); void prepare(); }}>
      <label>Amazon 商品类型<Input required value={fields.productType} onChange={e => change("productType", e.target.value)} disabled={busy} /></label>
      <label>商品名称<Input required value={fields.title} onChange={e => change("title", e.target.value)} disabled={busy} /></label>
      <label>商品描述<Textarea value={fields.description} onChange={e => change("description", e.target.value)} disabled={busy} /></label>
      <label>品牌<Input required value={fields.brand} onChange={e => change("brand", e.target.value)} disabled={busy} /></label>
      <label>商品条码（GTIN）<Input required inputMode="numeric" value={fields.gtin} onChange={e => change("gtin", e.target.value)} disabled={busy} /></label>
      <div className="grid gap-3 sm:grid-cols-2">
        <label>售价（美元）<Input required inputMode="decimal" value={fields.price} onChange={e => change("price", e.target.value)} disabled={busy} /></label>
        <label>可卖库存<Input required inputMode="numeric" value={fields.quantity} onChange={e => change("quantity", e.target.value)} disabled={busy} /></label>
      </div>
      <label>已批准商品主图<Select required value={fields.mainImageId} onChange={e => change("mainImageId", e.target.value)} disabled={busy}>
        <option value="">选择主图</option>
        {images.map((image, i) => <option key={image.id} value={image.id}>已批准图片 {i + 1}</option>)}
      </Select></label>
      {images.length === 0 ? <p>请先在原商品的图片入口批准一张商品主图。</p> : null}
      <Button type="submit" disabled={busy || images.length === 0}>{busy ? "正在核对…" : "核对商品资料"}</Button>
    </form> : null}
    {!loading && canEdit && review ? <div className="space-y-3" aria-label="确认 Amazon 刊登资料">
      <dl className="grid gap-2 sm:grid-cols-[auto_1fr]">
        <dt>卖家 SKU</dt><dd>{review.sku}</dd>
        <dt>商品类型</dt><dd>{review.details.productType}</dd>
        <dt>商品名称</dt><dd>{review.details.title}</dd>
        {review.details.description ? <><dt>商品描述</dt><dd>{review.details.description}</dd></> : null}
        <dt>品牌</dt><dd>{review.details.brand}</dd>
        <dt>商品条码</dt><dd>{review.details.gtin}</dd>
        <dt>售价</dt><dd>USD {(review.details.priceCents / 100).toFixed(2)}</dd>
        <dt>可卖库存</dt><dd>{review.details.quantity}</dd>
        <dt>发货方式</dt><dd>卖家发货</dd>
        <dt>主图</dt><dd>{images.findIndex(image => image.id === review.details.mainImageId) >= 0 ? `已批准图片 ${images.findIndex(image => image.id === review.details.mainImageId) + 1}` : "本次已核对的主图"}</dd>
      </dl>
      <p>正式流程中，Amazon 处理完成后商品可能公开销售。本次使用合成平台响应。</p>
      <div className="flex flex-wrap gap-2">
        <Button type="button" variant="outline" disabled={busy} onClick={() => setReview(undefined)}>返回修改</Button>
        {canSubmit ? <Button type="button" disabled={busy} onClick={() => void confirm()}>{busy ? "正在提交…" : "确认提交到 Amazon"}</Button> : <p>当前成员没有商品上传权限。</p>}
      </div>
    </div> : null}
  </section>;
}
