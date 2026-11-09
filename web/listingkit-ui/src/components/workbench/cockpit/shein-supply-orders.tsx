"use client";

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { observationRequest, observationCommandSchema, observationSyncSchema, observationBeginSchema, observationID, type ObservationScope } from "@/lib/api/store-observations";
import { supplyObservationSchema } from "@/lib/contracts/shein-supply-orders";
import styles from "./cockpit-overview.module.css";

type Intent = {key:string;input: {kind:"supply-orders";stores:string[];supplyOrderIds:string[]}};
const titles = {ordered:"订购数量",delivered:"已发货数量",received:"已收货数量",stored:"已入库数量",defective:"次品数量"};
function restoreIntent(storageKey:string,storeId:string):{intent?:Intent;error?:string}{
  if(typeof window==="undefined")return {};
  try {
    const raw=sessionStorage.getItem(storageKey);
    if(!raw)return {};
    const saved=JSON.parse(raw) as Intent;
    const parsed=observationBeginSchema.safeParse(saved.input);
    if(!observationID.safeParse(saved.key).success||!parsed.success||parsed.data.kind!=="supply-orders"||parsed.data.stores[0]!==storeId)throw new Error();
    return {intent:saved};
  }catch {return {error:"原读取记录暂不可用，请恢复本页会话存储后重试。"};}
}

// This panel deliberately has no date input: the provider's natural-day contract is unconfirmed.
export function SheinSupplyOrders({scope,storeId,onApplicable}:{scope:ObservationScope;storeId:string;onApplicable:(id:string)=>void}) {
  const storageKey=JSON.stringify(["shein-supply-read",scope.organizationId,scope.userId,storeId]);
  const [restored]=useState(()=>restoreIntent(storageKey,storeId));
  const [text,setText]=useState(restored.intent?.input.supplyOrderIds.join("\n")??"");
  const [intent,setIntent]=useState<Intent|undefined>(restored.intent);
  const [busy,setBusy]=useState(false);
  const [error,setError]=useState(restored.error??"");
  const query=useQuery({queryKey:["shein-supply-orders",scope.organizationId,scope.userId,storeId],
    queryFn:({signal})=>observationRequest(`supply-orders/observation?storeId=${encodeURIComponent(storeId)}`,scope,supplyObservationSchema,signal),
    retry:false,gcTime:0,refetchInterval:q=>q.state.data?.status==="running"?2000:false});
  const applicable=query.data?.status!==undefined&&query.data.status!=="not_applicable";
  useEffect(()=>{if(applicable)onApplicable(storeId);},[applicable,onApplicable,storeId]);
  async function submit(){
    setError("");setBusy(true);
    try {
      let original=intent;
      if(!original){
        const raw=text.trim().split(/[\s,，]+/u).filter(Boolean);
        // Validate the original bounded request before canonicalizing duplicate IDs.
        const parsed=observationBeginSchema.parse({kind:"supply-orders",stores:[storeId],supplyOrderIds:raw});
        original={key:crypto.randomUUID(),input:{kind:"supply-orders",stores:[storeId],supplyOrderIds:[...new Set(parsed.supplyOrderIds!)].sort()}};
        sessionStorage.setItem(storageKey,JSON.stringify(original));
        setIntent(original);
      }
      await observationRequest("supply-orders/syncs",scope,observationCommandSchema,undefined,original);
      sessionStorage.removeItem(storageKey);setIntent(undefined);
      await query.refetch();
    }catch {setError("读取未确认。请重试原读取；不会重复创建另一批。每批最多 200 个采购单号。");}
    finally {setBusy(false);}
  }
  async function resume(){
    const id=query.data?.source?.syncId;
    if(!id)return;
    setBusy(true);setError("");
    try {await observationRequest(`supply-orders/syncs/${id}/ensure`,scope,observationSyncSchema,undefined,{});await query.refetch();}
    catch {setError("继续读取尚未确认，请重试继续原读取。");}
    finally {setBusy(false);}
  }
  if(query.data?.status==="not_applicable")return null;
  const data=query.data,source=data?.source,summary=data?.summary;
  return <Card className={styles.panel}>
    <header className={styles.panelHeader}><h2>SHEIN 全托供货单观察</h2><small>按采购单号批次</small></header>
    <p className={styles.panelNote}>仅观察本批采购单号，不代表店铺全量。日期口径待确认，不随首页日期变化；暂不统计金额。</p>
    {query.isPending?<p role="status">正在读取供货单观察…</p>:query.error?<p role="alert">供货单观察暂不可用。<Button variant="outline" onClick={()=>void query.refetch()}>重新读取</Button></p>:null}
    {data&&<>
      <p role="status">{data.status==="not_synced"?"尚无已保存批次":data.status==="running"?"本批正在读取，完成后自动更新":data.status==="saved"?"本批单号均已保存":"覆盖不足：仅展示本批已保存部分"}</p>
      {data.status==="running"&&data.canSync&&source&&<Button variant="outline" disabled={busy} onClick={()=>void resume()}>继续原读取</Button>}
      {source&&<p className={styles.panelNote}>本批 {source.orderIds.length} 个单号 · 已保存 {data.savedOrderIds.length} 个 · 未返回 {source.missingIds.length} 个</p>}
      {summary&&<>
        <dl className={styles.publicationCounts}><div><dt>非母单原单</dt><dd>{summary.orderCount}</dd></div><div><dt>JIT 母单（单列）</dt><dd>{summary.motherCount}</dd></div><div><dt>母子类型待核实</dt><dd>{summary.unknownTypeCount}</dd></div></dl>
        <p className={styles.panelNote}>以下仅合计已保存的非母单；母单与类型待核实单均不叠加。</p>
        <dl className={styles.publicationCounts}>{(Object.keys(titles) as (keyof typeof titles)[]).map(k=><div key={k}><dt>{titles[k]}</dt><dd>{summary.quantities[k].value??"待核实"}</dd>{summary.quantities[k].missingLines>0&&<small>{summary.quantities[k].missingLines} 行缺失或有歧义</small>}</div>)}</dl>
      </>}
      {source&&<details className={styles.panelNote}><summary>查看本批单号与覆盖范围</summary><p className="break-all">请求：{source.orderIds.join("、")}</p><p className="break-all">已保存：{data.savedOrderIds.join("、")||"无"}</p>{source.missingIds.length>0&&<p className="break-all">待核实：{source.missingIds.join("、")}</p>}{source.notes.length>0&&<p>回执覆盖未完全确认，请以本批已保存部分为准。</p>}</details>}
      {data.canSync&&<form className="grid gap-3 p-3" onSubmit={e=>{e.preventDefault();void submit();}}>
        <label htmlFor="supply-order-ids">读取并保存采购单（每批最多 200 个单号，一行一个）</label>
        <Textarea id="supply-order-ids" value={text} onChange={e=>setText(e.target.value)} disabled={busy||!!intent||data.status==="running"} maxLength={26000}/>
        <Button type="submit" disabled={!!restored.error||busy||data.status==="running"||(!intent&&!text.trim())}>{busy?"正在确认读取…":intent?"重试原读取":"读取并保存本批"}</Button>
      </form>}
    </>}
    {data&&error&&<p role="alert">{error}</p>}
  </Card>;
}
