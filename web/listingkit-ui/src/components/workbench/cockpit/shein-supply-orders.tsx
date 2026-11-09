"use client";

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { observationRequest, observationCommandSchema, observationSyncSchema, observationBeginSchema, observationID, type ObservationScope } from "@/lib/api/store-observations";
import { supplyObservationSchema } from "@/lib/contracts/shein-supply-orders";
import styles from "./shein-supply-orders.module.css";
import {SheinSupplyTimes} from "./shein-supply-times";
import {SheinSupplyDates} from "./shein-supply-dates";

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

// Date filters are source calendar labels and never become provider queries.
export function SheinSupplyOrders({scope,storeId,startDate,endDate,onApplicable}:{scope:ObservationScope;storeId:string;startDate:string;endDate:string;onApplicable:(id:string)=>void}) {
  const storageKey=JSON.stringify(["shein-supply-read",scope.organizationId,scope.userId,storeId]);
  const [restored]=useState(()=>restoreIntent(storageKey,storeId));
  const [text,setText]=useState(restored.intent?.input.supplyOrderIds.join("\n")??"");
  const [intent,setIntent]=useState<Intent|undefined>(restored.intent);
  const [busy,setBusy]=useState(false);
  const [error,setError]=useState(restored.error??"");
  const query=useQuery({queryKey:["operations-cockpit",scope.userId,scope.organizationId,"shein-supply-orders",storeId,startDate,endDate],
    queryFn:({signal})=>observationRequest(`supply-orders/observation?${new URLSearchParams({storeId,startDate,endDate})}`,scope,supplyObservationSchema,signal),
    retry:false,gcTime:0,refetchInterval:q=>q.state.data?.status==="running"?2000:false});
  const applicable=query.data?.status!==undefined&&query.data.status!=="not_applicable";
  useEffect(()=>{if(applicable)onApplicable(storeId);},[applicable,onApplicable,storeId]);
  async function submit(){
    setError("");setBusy(true);
    let original=intent;
    try {
      if(!original){
        const raw=text.trim().split(/[\s,，]+/u).filter(Boolean);
        // Validate the original bounded request before canonicalizing duplicate IDs.
        const parsed=observationBeginSchema.parse({kind:"supply-orders",stores:[storeId],supplyOrderIds:raw});
        const next:Intent={key:crypto.randomUUID(),input:{kind:"supply-orders",stores:[storeId],supplyOrderIds:[...new Set(parsed.supplyOrderIds!)].sort()}};
        sessionStorage.setItem(storageKey,JSON.stringify(next));
        original=next;
        setIntent(original);
      }
      await observationRequest("supply-orders/syncs",scope,observationCommandSchema,undefined,original);
      sessionStorage.removeItem(storageKey);setIntent(undefined);
      await query.refetch();
    }catch {setError(original?"读取未确认，请重试原读取；不会重复创建另一批。":"无法准备读取，请检查单号、减少批量并确认本页会话存储可用。");}
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
  if(!query.error&&!query.isFetching&&query.data?.status==="not_applicable")return null;
  const data=query.error||query.isFetching||busy?undefined:query.data;
  const source=intent?undefined:data?.source,summary=intent?undefined:data?.summary;
  return <Card className={styles.panel}>
    <header className={styles.header}><div className={styles.heading}><h2>SHEIN 供货概览</h2><span className={styles.mode}>全托</span></div><span className={styles.note}>最新保存批次</span></header>
    {query.isPending||query.isFetching||busy?<p role="status">正在确认供货单读取…</p>:query.error?<p role="alert">供货单观察暂不可用。<Button variant="outline" onClick={()=>void query.refetch()}>重新读取</Button></p>:null}
    {data&&<>
      <div className={styles.coverage}><span role="status" className={styles.status} data-state={intent?"incomplete":data.status}>{intent?"原读取结果尚未确认，请重试原读取":data.status==="not_synced"?"尚无已保存批次":data.status==="running"?"本批正在读取，完成后自动更新":data.status==="saved"?"本批单号均已保存":"覆盖不足：仅展示本批已保存部分"}</span>
      {source&&<p>本批 {source.orderIds.length} 个单号 · 已保存 {data.savedOrderIds.length} 个 · 未返回 {source.missingIds.length} 个</p>}</div>
      {data.status==="running"&&data.canSync&&source&&<Button variant="outline" disabled={busy} onClick={()=>void resume()}>继续原读取</Button>}
      {source&&data.originalDates&&<SheinSupplyDates data={data.originalDates}/>}
      <p className={styles.note}>仅当前已保存批次，不代表店铺全量。时区未确认，按原文日期筛选，不作转换。</p>
      {source&&<details className={styles.disclosure}><summary>批次详情</summary><div className={styles.detailBody}>
      {summary&&<>
        <h3>本批全部数量（不按日期筛选）</h3>
        <dl className={styles.counts}><div><dt>非母单原单</dt><dd>{summary.orderCount}</dd></div><div><dt>JIT 母单（单列）</dt><dd>{summary.motherCount}</dd></div><div><dt>母子类型待核实</dt><dd>{summary.unknownTypeCount}</dd></div></dl>
        <p className={styles.note}>以下仅合计已保存的非母单；母单与类型待核实单均不叠加。</p>
        <dl className={styles.counts}>{(Object.keys(titles) as (keyof typeof titles)[]).map(k=><div key={k}><dt>{titles[k]}</dt><dd>{summary.quantities[k].value??"待核实"}</dd>{summary.quantities[k].missingLines>0&&<small>{summary.quantities[k].missingLines} 行缺失或有歧义</small>}</div>)}</dl>
      </>}
      <section className={styles.ids} aria-label="本批单号与覆盖范围"><h3>单号与覆盖范围</h3><p>请求：{source.orderIds.join("、")}</p><p>已保存：{data.savedOrderIds.join("、")||"无"}</p>{source.missingIds.length>0&&<p>待核实：{source.missingIds.join("、")}</p>}{source.notes.length>0&&<p>回执覆盖未完全确认，请以本批已保存部分为准。</p>}</section>
      {summary&&data.savedOrderIds.length>0&&<SheinSupplyTimes key={JSON.stringify([scope.organizationId,scope.userId,storeId,source.syncId])} scope={scope} storeId={storeId} syncId={source.syncId} ids={data.savedOrderIds}/>}
      <details className={styles.matched}><summary>统计口径与字段说明</summary><p>急采取下单时间 addTime，备货取下发时间 allocateTime；缺失或无法确认的时间不归入任何一天，不使用同步时间补齐。JIT 母单及母子、采购类型待核实单不计入两组。</p><p>原文含偏移也不转换时区；1970 等格式合法值不自动视为缺失，不据此确认事件发生。订购数量是已保存观察值，不是当日发货或收货量，暂不统计金额。</p></details>
      </div></details>}
      {data.canSync&&<details className={styles.disclosure} open={!source||!!intent||!!restored.error}><summary>读取采购单</summary><form className={styles.form} onSubmit={e=>{e.preventDefault();void submit();}}>
        <label htmlFor="supply-order-ids">读取并保存采购单（每批最多 200 个单号，一行一个）</label>
        <Textarea id="supply-order-ids" value={text} onChange={e=>setText(e.target.value)} disabled={busy||!!intent||data.status==="running"} maxLength={26000}/>
        <Button type="submit" disabled={!!restored.error||busy||(!intent&&data.status==="running")||(!intent&&!text.trim())}>{busy?"正在确认读取…":intent?"重试原读取":"读取并保存本批"}</Button>
      </form></details>}
    </>}
    {data&&error&&<p role="alert" className={styles.alert}>{error}</p>}
  </Card>;
}
