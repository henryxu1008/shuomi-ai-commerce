"use client";
import {useEffect,useRef,useState} from "react";
import {Button} from "@/components/ui/button";
import {readSupplyPublicationStatus,type SupplyScope} from "@/lib/api/supply-chain";
import type {SupplyPublication} from "@/lib/contracts/supply-chain";

type WalmartFeedResultProps={value:SupplyPublication;scope:SupplyScope;sourceId:string;storeId:string;onChanged?:()=>void};
export function WalmartFeedResult(props:WalmartFeedResultProps){
 const {scope,sourceId,storeId,value}=props;
 const inputKey=JSON.stringify([scope.userId,scope.organizationId,sourceId,storeId,value.receiptId,value.walmart?.feedId]);
 return <BoundWalmartFeedResult key={inputKey} {...props}/>;
}
function BoundWalmartFeedResult({value,scope,sourceId,storeId,onChanged}:WalmartFeedResultProps){
 const [current,setCurrent]=useState<SupplyPublication>();const [busy,setBusy]=useState(false),[error,setError]=useState(false);const active=useRef<AbortController|null>(null);
 useEffect(()=>()=>{active.current?.abort();active.current=null},[]);
 async function query(receiptId?:string){if(active.current)return;const c=new AbortController();active.current=c;setBusy(true);setError(false);const valid=()=>!c.signal.aborted&&active.current===c;try{const next=await readSupplyPublicationStatus(scope,sourceId,storeId,c.signal,receiptId);if(valid()){if(next.sourceId!==sourceId||next.storeId!==storeId||next.receiptId!==value.receiptId){setError(true);onChanged?.();return}setCurrent(next);onChanged?.()}}catch{if(valid())setError(true)}finally{if(active.current===c){active.current=null;setBusy(false)}}}
 const result=current??value;const latestByReceipt=(result.walmartResults??[]).filter((item,index,all)=>all.findIndex(v=>v.receiptId===item.receiptId)===index);
 return <div className="mt-2 space-y-2 break-all text-xs">
  <p>Walmart 已受理 Feed：{result.walmart?.feedId}</p>
  <p>受理只代表收到资料，逐 SKU 处理、发布和可售状态分别读取。</p>
  <Button size="sm" variant="outline" disabled={busy} onClick={()=>void query()}>{busy?"正在查询":"查询平台结果"}</Button>
  {error?<p role="alert">本次查询未成功，原受理回执仍保留；不会重发 Feed。</p>:null}
  {latestByReceipt.map(item=><section key={item.id} className="space-y-1 rounded-lg border p-2">
   <p>Feed {item.observation.feedId} · {item.observation.feedStatus||"尚未确认"}{item.observation.partial?" · 部分结果，不能据此重发":""}</p>
   <Button size="sm" variant="outline" disabled={busy} onClick={()=>void query(item.receiptId)}>刷新此 Feed 与仍归属的 SKU</Button>
   {item.readProgress||item.readOffset!==undefined?<p>本轮已读取 {item.readProgress?.nextOffset??item.readOffset} 条明细；再次查询将继续读取或刷新。逐页观察不代表全部处理完成。</p>:null}
   {item.limited?<p>部分明细因大小限制未展示，已保存进度仍保留；当前覆盖受限，不能据此重发。</p>:null}
   <p>查询时间：{new Date(item.observation.observedAt).toLocaleString()}</p>
   <WalmartItemResults items={item.observation.items}/>
   {item.observedPages?.length?<div className="space-y-2 border-t pt-2">
    <p>此前窗口页面仅供核对，不能拼成完整的新结果。</p>
    {item.observedPages.map((page,index)=><div key={`${page.offset}:${index}`} className="rounded border p-2">
     <p>此前读取的第 {page.offset+1} 条起明细 · {page.observation.feedStatus||"尚未确认"}</p>
     <p>页面取得时间：{new Date(page.observation.observedAt).toLocaleString()}</p>
     <WalmartItemResults items={page.observation.items}/>
    </div>)}
   </div>:null}
  </section>)}
 </div>
}

type WalmartItems=NonNullable<SupplyPublication["walmartResults"]>[number]["observation"]["items"];
function WalmartItemResults({items}:{items:WalmartItems}){
 return items.map((s,index)=><div key={`${s.sku}:${index}`} className="border-t pt-1">
  <p>SKU：{s.sku}</p><p>处理：{s.ingestionStatus||"待核实"}</p>
  <p>发布：{s.publishedStatus||"尚未读取"} · 生命周期：{s.lifecycleStatus||"未知"} · 可售性：{s.availability||"平台未返回"}</p>
  {s.publicationObservedAt?<p>发布信息取得时间：{new Date(s.publicationObservedAt).toLocaleString()}</p>:null}
  {s.errors?.map((e,i)=><p key={i}>{e.code}{e.field?` · ${e.field}`:""}：{e.description}</p>)}
 </div>)
}
