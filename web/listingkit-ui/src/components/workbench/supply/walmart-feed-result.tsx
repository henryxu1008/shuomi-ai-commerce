"use client";
import {useEffect,useRef,useState} from "react";
import {Button} from "@/components/ui/button";
import {readSupplyPublicationStatus,type SupplyScope} from "@/lib/api/supply-chain";
import type {SupplyPublication} from "@/lib/contracts/supply-chain";

export function WalmartFeedResult({value,scope,sourceId,storeId,onChanged}:{value:SupplyPublication;scope:SupplyScope;sourceId:string;storeId:string;onChanged?:()=>void}){
 const [current,setCurrent]=useState<SupplyPublication>();const [busy,setBusy]=useState(false),[error,setError]=useState(false);const active=useRef<AbortController|null>(null);
 useEffect(()=>()=>active.current?.abort(),[]);
 async function query(receiptId?:string){if(active.current)return;const c=new AbortController();active.current=c;setBusy(true);setError(false);try{const next=await readSupplyPublicationStatus(scope,sourceId,storeId,c.signal,receiptId);if(!c.signal.aborted){setCurrent(next);onChanged?.()}}catch{if(!c.signal.aborted)setError(true)}finally{active.current=null;if(!c.signal.aborted)setBusy(false)}}
 const result=current??value;const latestByReceipt=(result.walmartResults??[]).filter((item,index,all)=>all.findIndex(v=>v.receiptId===item.receiptId)===index);
 return <div className="mt-2 space-y-2 break-all text-xs"><p>Walmart 已受理 Feed：{result.walmart?.feedId}</p><p>受理只代表收到资料，逐 SKU 处理、发布和可售状态分别读取。</p><Button size="sm" variant="outline" disabled={busy} onClick={()=>void query()}>{busy?"正在查询":"查询平台结果"}</Button>{error?<p role="alert">本次查询未成功，原受理回执仍保留；不会重发 Feed。</p>:null}{latestByReceipt.map(item=><section key={item.id} className="space-y-1 rounded-lg border p-2"><p>Feed {item.observation.feedId} · {item.observation.feedStatus||"尚未确认"}{item.observation.partial?" · 部分结果，不能据此重发":""}</p><Button size="sm" variant="outline" disabled={busy} onClick={()=>void query(item.receiptId)}>刷新此 Feed 与仍归属的 SKU</Button><p>查询时间：{new Date(item.observation.observedAt).toLocaleString()}</p>{item.observation.items.map(s=><div key={s.sku} className="border-t pt-1"><p>SKU：{s.sku}</p><p>处理：{s.ingestionStatus||"待核实"}</p><p>发布：{s.publishedStatus||"尚未读取"} · 生命周期：{s.lifecycleStatus||"未知"} · 可售性：{s.availability||"平台未返回"}</p>{s.publicationObservedAt?<p>发布信息取得时间：{new Date(s.publicationObservedAt).toLocaleString()}</p>:null}{s.errors?.map((e,i)=><p key={i}>{e.code}{e.field?` · ${e.field}`:""}：{e.description}</p>)}</div>)}</section>)}</div>
}
