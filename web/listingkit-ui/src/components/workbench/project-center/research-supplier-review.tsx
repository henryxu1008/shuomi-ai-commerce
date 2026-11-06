"use client";

import {useEffect, useRef, useState} from "react";
import type {z} from "zod";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {ResearchError, researchRequest} from "@/lib/api/market-research";
import type {ProjectScope} from "@/lib/api/project-center";
import {researchReceipt, selectionOf, supplierReviewPage, supplierReviewPending, supplierReviewPreview, type ResearchCommand, type ResearchSelection} from "@/lib/contracts/market-research";
import {CollectionPicker} from "../pod/collection-picker";
import {useResourcePending} from "../resources/resource-pending";

type Preview = z.infer<typeof supplierReviewPreview>;
type Page = z.infer<typeof supplierReviewPage>;
type SupplierCommand = Extract<ResearchCommand, {action:"record_supplier_review"}>;
type Comparison = SupplierCommand["supplier"]["comparisons"][number];
type Intent = {key:string; body:SupplierCommand};
const initialComparison: Comparison = {marketKey:"", supplierKey:"", supplierLevel:"sku", result:"PENDING"};
const comparisonLabels = {SAME:"一致", DIFFERENT:"不同", PENDING:"待核对"} as const;

function statusMessage(error:unknown) {
  if (error instanceof ResearchError) {
    if (error.code === "REVISION_CONFLICT") return "商品资料或候选选择已变化，请重新核对。";
    if (error.code === "FORBIDDEN" || error.code === "NOT_FOUND") return "原资料当前不可查看。";
    if (error.code === "INVALID_REQUEST") return "请核对 SKU、数量、规格差异和理由。";
    if (error.code === "SOURCE_NOT_SUPPORTED") return "这条已保存商品没有可核对的 1688 原版。";
  }
  return "货源核对暂时无法读取，请稍后刷新。";
}

export function ResearchSupplierReview({scope, projectId, candidateId, canWrite, onSaved}:{
  scope:ProjectScope; projectId:string; candidateId:string; canWrite:boolean; onSaved:()=>void;
}) {
  const pending = useResourcePending(
    {expectedUserId:scope.userId, expectedOrganizationId:scope.organizationId},
    ["research-supplier-review", projectId, candidateId], supplierReviewPending,
  );
  const [page, setPage] = useState<Page|null>(null);
  const [pageNumber, setPageNumber] = useState(0);
  const [picker, setPicker] = useState(false);
  const [selection, setSelection] = useState<ResearchSelection|null>(null);
  const [preview, setPreview] = useState<Preview|null>(null);
  const [skuId, setSkuId] = useState("");
  const [quantity, setQuantity] = useState("");
  const [reason, setReason] = useState("");
  const [comparisons, setComparisons] = useState<Comparison[]>([initialComparison]);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const lifetime = useRef<AbortController|null>(null);
  const reading = useRef<AbortController|null>(null);
  const [operation, setOperation] = useState<Intent|null>(null);
  const writing = useRef(false);
  const selectedSku = preview?.offer.variants.find(v => v.id === skuId);

  function clearDraft() {
    setSelection(null);
    setPreview(null);
    setSkuId("");
    setQuantity("");
    setReason("");
    setComparisons([initialComparison]);
    setPicker(false);
  }

  async function readHistory(before = 0) {
    if (!lifetime.current || lifetime.current.signal.aborted || writing.current) return;
    reading.current?.abort();
    const read = new AbortController();
    reading.current = read;
    setPage(null);
    clearDraft();
    setMessage("");
    try {
      const result = await researchRequest(scope, projectId, supplierReviewPage, read.signal, {supplier:{candidate:candidateId, before}});
      if (!read.signal.aborted) {
        setPage(result);
        setPageNumber(before);
        if (result.items.some(row => row.current && !row.detail)) setOperation(null);
      }
    } catch (error) {
      if (!read.signal.aborted) {
        setOperation(null);
        setMessage(statusMessage(error));
      }
    }
  }

  useEffect(() => {
    const controller = new AbortController();
    lifetime.current = controller;
    void Promise.resolve().then(() => { if (!controller.signal.aborted) void readHistory(); });
    return () => { controller.abort(); reading.current?.abort(); };
    // The parent keys this component on scope, candidate, revision and access.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function selectOffer(next:ResearchSelection) {
    if (!lifetime.current || lifetime.current.signal.aborted || !canWrite || writing.current) return;
    reading.current?.abort();
    const read = new AbortController();
    reading.current = read;
    clearDraft();
    setMessage("");
    try {
      const result = await researchRequest(scope, projectId, supplierReviewPreview, read.signal, {supplier:{candidate:candidateId, selection:next}});
      if (read.signal.aborted) return;
      setSelection(next);
      setPreview(result);
      if (result.offer.variants.length === 1) setSkuId(result.offer.variants[0].id);
    } catch (error) {
      if (!read.signal.aborted) setMessage(statusMessage(error));
    }
  }

  async function submit(intent:Intent) {
    const controller = lifetime.current;
    if (!controller || controller.signal.aborted || writing.current || !canWrite) return;
    const pointer = {key:intent.key};
    try {
      pending.persist(pointer);
    } catch {
      setMessage("无法保存原请求编号，请检查浏览器设置。");
      return;
    }
    setOperation(intent);
    reading.current?.abort();
    writing.current = true;
    setBusy(true);
    setMessage("");
    try {
      await researchRequest(scope, projectId, researchReceipt, controller.signal, {intent});
      if (controller.signal.aborted) return;
      pending.clear(pointer);
      setOperation(null);
      clearDraft();
      setMessage("已保存货源核对。");
      onSaved();
    } catch (error) {
      if (controller.signal.aborted) return;
      clearDraft();
      setPage(null);
      if (error instanceof ResearchError && ["INVALID_REQUEST","REVISION_CONFLICT","PROJECT_ARCHIVED","SOURCE_NOT_SUPPORTED","FORBIDDEN","NOT_FOUND"].includes(error.code)) {
        pending.clear(pointer);
        setOperation(null);
        setMessage(statusMessage(error));
      } else {
        setMessage("保存结果暂未确认，请查询原请求。");
      }
    } finally {
      writing.current = false;
      if (!controller.signal.aborted) setBusy(false);
    }
  }

  async function lookup() {
    const pointer = pending.command;
    const controller = lifetime.current;
    if (!pointer || !controller || controller.signal.aborted || writing.current) return;
    reading.current?.abort();
    writing.current = true;
    setBusy(true);
    setMessage("");
    try {
      await researchRequest(scope, projectId, researchReceipt, controller.signal, {lookup:pointer.key});
      if (controller.signal.aborted) return;
      pending.clear(pointer);
      setOperation(null);
      onSaved();
    } catch (error) {
      if (controller.signal.aborted) return;
      setMessage(error instanceof ResearchError && error.code === "NOT_FOUND" ? "暂未查到原请求的结果。" : statusMessage(error));
      if (error instanceof ResearchError && error.code === "FORBIDDEN") { setOperation(null); clearDraft(); setPage(null); }
    } finally {
      writing.current = false;
      if (!controller.signal.aborted) setBusy(false);
    }
  }

  function changeComparison(index:number, patch:Partial<Comparison>) {
    setComparisons(rows => rows.map((row,i) => i === index ? {...row,...patch} : row));
  }

  const canSave = !!(preview && selection && selectedSku && /^[1-9][0-9]{0,6}$/.test(quantity) &&
    Number(quantity) <= 1_000_000 && reason.trim() && new TextEncoder().encode(reason).length <= 2048 &&
    comparisons.every(row => {
      const supplier = row.supplierLevel === "sku" ? selectedSku.attributes : preview.offer.attributes;
      const marketKnown = Object.hasOwn(preview.market.attributes,row.marketKey);
      const supplierKnown = Object.hasOwn(supplier,row.supplierKey);
      return row.marketKey.trim() && row.supplierKey.trim() &&
        (row.result === "PENDING" ? marketKnown || supplierKnown : marketKnown && supplierKnown);
    }) && pending.ready && !pending.command && !busy);
  const awaitingSpecs = comparisons.some(row => row.result === "PENDING");

  return <section aria-label="已保存货源核对" className="grid min-w-0 gap-3">
    <div className="flex flex-wrap gap-2">
      {canWrite && !pending.command && <Button type="button" variant="secondary" disabled={busy || !pending.ready} onClick={() => {clearDraft();setPicker(true);}}>选择已收藏的 1688 货源</Button>}
      <Button type="button" variant="outline" disabled={busy} onClick={() => void readHistory()}>刷新核对记录</Button>
    </div>
    {pending.error && <p role="alert">原请求编号暂时无法读取，请检查浏览器设置后刷新。</p>}
    {pending.command && <div className="flex flex-wrap gap-2">
      <Button type="button" variant="outline" disabled={busy} onClick={() => void lookup()}>查询货源核对原请求</Button>
      {canWrite && operation?.key === pending.command.key &&
        <Button type="button" variant="outline" disabled={busy} onClick={() => {if (operation) void submit(operation);}}>按原请求重试</Button>}
    </div>}
    {message && <p role="status">{message}</p>}
    {picker && canWrite && <CollectionPicker scope={scope} kind="product" isEligible={item => item.source.kind === "acquisition"}
      authorizationKey={projectId + ":" + candidateId} emptyMessage="这页没有可选的已收藏货源。" onSelect={item => void selectOffer(selectionOf(item))}/>}
    {preview && selection && canWrite && <div className="grid min-w-0 gap-3 rounded-md border p-3">
      <p>Amazon 美国站原版 · {preview.market.title} · {preview.market.asin}</p>
      <p>1688 已收藏原版 · {preview.offer.title}</p>
      {preview.offer.variants.length === 0 ? <p role="status">原版没有可确认的 SKU，规格待核对。</p> : <>
        <label className="grid gap-1">1688 SKU
          <select value={skuId} onChange={e => {setSkuId(e.target.value);setComparisons([initialComparison]);}} className="rounded border bg-background p-2">
            <option value="">请选择 SKU</option>
            {preview.offer.variants.map(sku => <option key={sku.id} value={sku.id}>{sku.label} · {sku.id}</option>)}
          </select>
        </label>
        <label className="grid gap-1">采购意向数量（件）<Input type="text" inputMode="numeric" maxLength={7} value={quantity} onChange={e => setQuantity(e.target.value)}/></label>
        {selectedSku && comparisons.map((row, index) => {
          const supplierAttrs = row.supplierLevel === "sku" ? selectedSku.attributes : preview.offer.attributes;
          const both = !!(Object.hasOwn(preview.market.attributes,row.marketKey) && Object.hasOwn(supplierAttrs,row.supplierKey));
          return <fieldset key={index} className="grid gap-2 rounded border p-2">
            <legend>核对规格 {index + 1}</legend>
            <label className="grid gap-1">Amazon 规格字段
              <Input list={candidateId + "-market-attrs"} maxLength={128} value={row.marketKey} onChange={e => changeComparison(index,{marketKey:e.target.value,result:"PENDING"})}/>
            </label>
            <p>{Object.hasOwn(preview.market.attributes,row.marketKey) ? preview.market.attributes[row.marketKey] : "原版未见该规格"}</p>
            <label className="grid gap-1">1688 规格出处
              <select value={row.supplierLevel} onChange={e => changeComparison(index,{supplierLevel:e.target.value === "product" ? "product" : "sku",supplierKey:"",result:"PENDING"})} className="rounded border bg-background p-2">
                <option value="sku">所选 SKU</option><option value="product">商品级</option>
              </select>
            </label>
            <label className="grid gap-1">1688 规格字段
              <Input list={candidateId + "-" + row.supplierLevel + "-attrs"} maxLength={128} value={row.supplierKey} onChange={e => changeComparison(index,{supplierKey:e.target.value,result:"PENDING"})}/>
            </label>
            <p>{Object.hasOwn(supplierAttrs,row.supplierKey) ? supplierAttrs[row.supplierKey] : "原版未见该规格"}</p>
            <label className="grid gap-1">人工核对结果
              <select value={row.result} onChange={e => changeComparison(index,{result:e.target.value === "SAME" ? "SAME" : e.target.value === "DIFFERENT" ? "DIFFERENT" : "PENDING"})} className="rounded border bg-background p-2">
                <option value="PENDING">待核对</option><option value="SAME" disabled={!both}>一致</option><option value="DIFFERENT" disabled={!both}>不同</option>
              </select>
            </label>
            {comparisons.length > 1 && <Button type="button" variant="outline" onClick={() => setComparisons(rows => rows.filter((_,i) => i !== index))}>移除规格 {index + 1}</Button>}
          </fieldset>;
        })}
        {selectedSku && <>
          <datalist id={candidateId + "-market-attrs"}>{Object.keys(preview.market.attributes).map(key => <option key={key} value={key}/>)}</datalist>
          <datalist id={candidateId + "-sku-attrs"}>{Object.keys(selectedSku.attributes).map(key => <option key={key} value={key}/>)}</datalist>
          <datalist id={candidateId + "-product-attrs"}>{Object.keys(preview.offer.attributes).map(key => <option key={key} value={key}/>)}</datalist>
          <Button type="button" variant="outline" disabled={comparisons.length >= 5} onClick={() => setComparisons(rows => [...rows,initialComparison])}>添加一项规格</Button>
        </>}
        <label className="grid gap-1">人工理由<textarea className="rounded border bg-background p-2" rows={3} maxLength={2048} value={reason} onChange={e => setReason(e.target.value)}/></label>
        <CommercialUnknowns/>
        <p>人工确认仅用于研究。请在理由中说明已核对的差异。</p>
        <div className="flex flex-wrap gap-2">
          <Button type="button" disabled={!canSave} onClick={() => {
            if (!canSave) return;
            void submit({key:crypto.randomUUID(),body:{action:"record_supplier_review",candidateId,
              expectedRevision:preview.revision,linkId:preview.linkId,evidenceDigest:preview.evidenceDigest,
              supplier:{selection,skuId,quantity:Number(quantity),unit:"piece",status:awaitingSpecs?"PENDING_REVIEW":"CONFIRMED",reason:reason.trim(),comparisons}}});
          }}>{awaitingSpecs ? "保存待核对记录" : "保存人工货源确认"}</Button>
          <Button type="button" variant="outline" onClick={clearDraft}>取消核对</Button>
        </div>
      </>}
    </div>}
    {page?.items.length === 0 && <p>还没有保存货源核对。</p>}
    {page?.items.map((row,index) => <article key={index} className="grid gap-1 rounded border p-3">
      {row.detail ? <>
        <p>{row.current ? "当前货源核对" : "历史货源核对"} · {row.detail.status === "CONFIRMED" ? "已人工确认" : "待核对"}</p>
        <p>Amazon {row.detail.marketTitle} · {row.detail.asin} · 原版 {row.detail.marketVersion}</p>
        <p>1688 {row.detail.supplierTitle} · {row.detail.offerId} · 原版 {row.detail.supplierVersion}</p>
        <p>SKU {row.detail.skuLabel} · {row.detail.skuId} · {row.detail.quantity} 件</p>
        <ul>{row.detail.comparisons.map(c => <li key={JSON.stringify([c.marketKey,c.supplierLevel,c.supplierKey])}>
          {c.marketKey}（{c.marketValue ?? "未提供"}）／{c.supplierKey}（{c.supplierLevel === "sku" ? "SKU" : "商品级"}，{c.supplierValue ?? "未提供"}）· {comparisonLabels[c.result]}
        </li>)}</ul>
        <p className="break-words">人工理由：{row.detail.reason}</p>
        <CommercialUnknowns/>
      </> : <p>原资料当前不可查看。</p>}
    </article>)}
    <div className="flex flex-wrap gap-2">
      {pageNumber > 0 && <Button type="button" variant="outline" onClick={() => void readHistory()}>返回最近记录</Button>}
      {page?.nextBefore && <Button type="button" variant="outline" onClick={() => void readHistory(page.nextBefore)}>查看较早记录</Button>}
    </div>
  </section>;
}

function CommercialUnknowns() {
  return <p>采购价：未知 · 该数量适用单价：未知 · MOQ：未知 · 混批：未知</p>;
}
