"use client";
import {useEffect,useRef,useState,useSyncExternalStore} from "react";
import {useQuery} from "@tanstack/react-query";
import Link from "next/link";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {getCommercialWallet} from "@/lib/api/commercial-billing";
import {createStorageOrder,createStorageQuote,getStorageOffers,getStorageSummary,readStorageOrder,type StorageQuote} from "@/lib/api/storage-capacity";
import {pendingSchema,readPending} from "../commercial/resource-purchase-panel";
import styles from "./enterprise-storage.module.css";
const subscribe=(f:()=>void)=>{window.addEventListener("storage-purchase-pending",f);window.addEventListener("storage",f);return()=>{window.removeEventListener("storage-purchase-pending",f);window.removeEventListener("storage",f)}};
const gib=(v:string)=>{const n=BigInt(v),scaled=n*BigInt(100)/BigInt(1073741824);if(n>BigInt(0)&&scaled===BigInt(0))return "< 0.01 GiB";return `${scaled/BigInt(100)}.${(scaled%BigInt(100)).toString().padStart(2,"0")} GiB`};
const money=(v:string)=>{const n=BigInt(v);return `¥${n/BigInt(100)}.${(n%BigInt(100)).toString().padStart(2,"0")}`};
type Props={userId:string;organizationId:string;scope:string;sequence:number;canPurchase:boolean;readOnly?:boolean};
export function EnterpriseStorage(props:Props){return <ScopedStorage key={JSON.stringify([props.userId,props.organizationId,props.scope])} {...props}/>}
function ScopedStorage({userId,organizationId,scope,sequence,canPurchase,readOnly=false}:Props){
  const summary=useQuery({queryKey:["workbench",userId,organizationId,scope,"storage",sequence],queryFn:({signal})=>getStorageSummary(userId,organizationId,signal),staleTime:0,gcTime:0,retry:false});
  const offers=useQuery({queryKey:["workbench",userId,organizationId,scope,"storage-offers",sequence],queryFn:({signal})=>getStorageOffers(userId,organizationId,signal),staleTime:0,gcTime:0,retry:false,enabled:!readOnly&&!!summary.data&&!summary.isError});
  const key=`storage-purchase.pending:${JSON.stringify([userId,organizationId])}`;
  const raw=useSyncExternalStore(subscribe,()=>{if(readOnly)return "";try{return sessionStorage.getItem(key)??""}catch{return "__unavailable__"}},()=>"__loading__");
  const pending=readPending(raw);const ready=raw!=="__loading__"&&!pending.invalid;
  const [quantity,setQuantity]=useState("1"),[dialogOpen,setDialogOpen]=useState(false);
  const [quote,setQuote]=useState<StorageQuote|null>(null),[wallet,setWallet]=useState<string|null>(null),[message,setMessage]=useState(""),[busy,setBusy]=useState(false);
  const active=useRef<AbortController|null>(null),mounted=useRef(true),locked=useRef(false),dialog=useRef<HTMLDialogElement|null>(null);
  useEffect(()=>{mounted.current=true;return()=>{mounted.current=false;active.current?.abort()}},[]);
  useEffect(()=>{if(dialogOpen&&!dialog.current?.open)dialog.current?.showModal();if(!dialogOpen&&dialog.current?.open)dialog.current.close()},[dialogOpen]);
  function persist(value:ReturnType<typeof pendingSchema.parse>|null){try{if(value)sessionStorage.setItem(key,JSON.stringify(value));else sessionStorage.removeItem(key);window.dispatchEvent(new Event("storage-purchase-pending"));return true}catch{setMessage("无法保存原订单恢复信息，已停止提交购买。");return false}}
  async function run(work:(signal:AbortSignal)=>Promise<void>){
    if(locked.current||readOnly||!canPurchase||!ready)return;
    locked.current=true;setBusy(true);setMessage("");
    const controller=new AbortController();active.current=controller;
    try{await work(controller.signal)}catch(error){
      if(mounted.current){
        const code=error&&typeof error==="object"&&"code"in error?String(error.code):"DEPENDENCY_UNAVAILABLE";
        let current:ReturnType<typeof readPending>={value:null,invalid:true};
        try{current=readPending(sessionStorage.getItem(key)??"")}catch{}
        // The original order lookup runs before quote expiry on the server.
        // This response is a confirmed rejection before creating an order.
        if(code==="QUOTE_EXPIRED"&&current.value&&!current.value.orderId)persist(null);
        setMessage(code==="INSUFFICIENT_FUNDS"?"企业钱包余额不足，请充值后重新确认。":code==="QUOTE_EXPIRED"?"报价已过期，请重新获取报价。":current.value||current.invalid?"本次结果未确认，请恢复原订单；不要重复购买。":"报价暂不可用，请刷新后重试。")
      }
    }finally{locked.current=false;if(mounted.current)setBusy(false)}
  }
  const offer=offers.data?.items[0];
  function prepare(count=quantity){if(locked.current||!canPurchase||readOnly||!ready||!offer||pending.value||!/^[1-9]$|^10$/.test(count))return;setQuantity(count);setQuote(null);setWallet(null);setDialogOpen(true);void run(async signal=>{const [q,w]=await Promise.all([createStorageQuote(userId,organizationId,offer.offer_id,count,signal),getCommercialWallet(userId,organizationId,signal)]);if(mounted.current){setQuote(q);setWallet(w.available_minor)}})}
  const insufficient=!!quote&&wallet!==null&&BigInt(wallet)<BigInt(quote.total_minor);
  function buy(){if(locked.current||readOnly||!canPurchase||!ready||!quote||quote.resource_quantity!==quantity||wallet===null||insufficient||pending.value||Date.parse(quote.expires_at)<=Date.now())return;const intent={offerId:quote.offer_id,quoteId:quote.quote_id,key:crypto.randomUUID()};if(!persist(intent))return;setQuote(null);setDialogOpen(false);void submit(intent)}
  function submit(intent:ReturnType<typeof pendingSchema.parse>){return run(async signal=>{
    const order=intent.orderId?await readStorageOrder(userId,organizationId,intent.orderId,signal):await createStorageOrder(userId,organizationId,intent.quoteId,intent.key,signal);
    if(!mounted.current)return;
    if(order.status==="FULFILLED"){persist(null);setMessage("扩容已生效。每个包从生效时起有效 30 天。");await summary.refetch();return}
    if(order.status==="CANCELLED"){persist(null);setMessage(order.failure_code==="INSUFFICIENT_FUNDS"?"企业钱包余额不足，订单已取消。":"原订单已取消。");return}
    persist({...intent,orderId:order.order_id});setMessage("正在核对原订单，请查看原订单结果；请勿重新购买。");
  })}
  const value=summary.data?.summary;
  return <Card className={styles.card} role="region" aria-label="企业存储容量">
    <div className={styles.heading}><div><h2>企业存储容量</h2><p>企业共享容量，商品图片、AI 图片和知识库原文件统一计量。</p></div>{!readOnly&&canPurchase&&<Button onClick={()=>prepare()} disabled={!ready||busy||!!pending.value||!value||!offer}>购买扩容包</Button>}</div>
    {summary.isPending||summary.isFetching?<p role="status">正在读取企业存储容量…</p>:summary.isError||!value?<p role="alert">存储容量服务暂不可用，本次未取得用量。请刷新重试。</p>:<>
      <dl className={styles.metrics}>{[["总容量",value.quota_bytes],["已使用",value.occupied_bytes],["预占中",value.reserved_bytes],["可用容量",value.remaining_bytes]].map(([label,bytes])=><div key={label}><dt>{label}</dt><dd title={`${bytes} 字节`}>{gib(bytes)}</dd></div>)}</dl>
      <p>商品媒体 {gib(value.categories.product_media??"0")} · AI 图片 {gib(value.categories.ai_image??"0")} · 知识库 {gib(value.categories.knowledge_document??"0")}{value.categories.business_attachment&&<> · 业务附件 {gib(value.categories.business_attachment)}</>}</p>
      {value.overage_bytes!=="0"&&<p role="alert">已超过当前容量 {gib(value.overage_bytes)}。新增文件已暂停；已有文件可以继续访问。</p>}
      {value.overage_bytes==="0"&&value.remaining_bytes==="0"&&<p role="alert">已达到当前容量，暂无剩余空间。新增文件已暂停；已有文件可以继续访问。</p>}
      {value.overage_bytes==="0"&&value.remaining_bytes!=="0"&&(BigInt(value.occupied_bytes)+BigInt(value.reserved_bytes))*BigInt(100)>=BigInt(value.quota_bytes)*BigInt(80)&&<p role="status">已使用与预占空间合计已达到 80%，请关注剩余容量。</p>}
      {value.active_lots.map(l=><p key={l.order_id}>{gib(l.bytes)} 扩容包 · 到期时间 {new Date(l.expires_at).toLocaleString("zh-CN")}</p>)}
      {value.more_lots&&<p>此处显示最先到期的 50 笔扩容包，全部容量已计入总容量。更多购买记录请在<Link href="/workbench/plans/orders" prefetch={false}>账单与订单</Link>查看。</p>}
    </>}
    {!readOnly&&<div className={styles.offer}>{offer?<span>{gib(offer.storage_unit_bytes)} 扩容包 · {money(offer.unit_price_minor)} / 30 天</span>:<span>扩容价格本次未取得</span>}</div>}
    <p>每企业默认 5 GiB。扩容包到期或容量不足会停止新增占用，不会自动删除已有数据。历史版本和未使用的已保存文件也计量。</p>
    {value&&<p className={styles.observed}>数据更新时间：{new Date(value.observed_at).toLocaleString("zh-CN")}</p>}
    {!readOnly&&!canPurchase&&<p>当前身份没有扩容购买权限。</p>}
    {!ready&&raw!=="__loading__"&&<p role="alert">原订单恢复信息暂不可读，购买已暂停，请联系支持核对。</p>}
    {pending.value&&<div><p>有一笔待确认的原订单。恢复会使用原报价与原请求，不会创建第二笔购买。</p><Button variant="outline" disabled={busy||!canPurchase} onClick={()=>void submit(pending.value!)}>恢复原订单</Button>{pending.value.orderId&&<span> 订单：{pending.value.orderId}</span>}</div>}
    {message&&<p role="status">{message}</p>}
    {!readOnly&&<dialog ref={dialog} className={styles.dialog} onCancel={e=>{if(busy)e.preventDefault();else setDialogOpen(false)}}><h2>确认购买存储扩容包</h2><label className={styles.quantity}>扩容包数量<select aria-label="扩容包数量" value={quantity} disabled={busy||!canPurchase||!!pending.value} onChange={e=>prepare(e.target.value)}>{Array.from({length:10},(_,i)=><option key={i+1} value={String(i+1)}>{i+1} 包</option>)}</select></label>{quote?<><p>{quote.resource_quantity} 个 × 50 GiB，共增加 {gib((BigInt(quote.resource_quantity)*BigInt(quote.storage_unit_bytes)).toString())}</p><p>应付 {money(quote.total_minor)} · 企业钱包余额 {wallet===null?"未取得":money(wallet)}</p><p>确认扣款后立即生效，有效 30 天。多次购买分别到期，不自动续费。</p><p>报价有效至 {new Date(quote.expires_at).toLocaleString("zh-CN")}</p>{insufficient&&<p role="alert">企业钱包余额不足，请充值后重新获取报价。</p>}</>:<p role="status">{busy?"正在获取冻结报价与钱包余额…":"本次未取得报价，请重试。"}</p>}<div className={styles.actions}><Button variant="outline" disabled={busy} onClick={()=>setDialogOpen(false)}>取消</Button>{quote?<Button disabled={busy||!canPurchase||wallet===null||insufficient||!!pending.value||quote.resource_quantity!==quantity||Date.parse(quote.expires_at)<=Date.now()} onClick={buy}>确认支付 {money(quote.total_minor)}</Button>:<Button disabled={busy||!canPurchase} onClick={()=>prepare()}>重新获取报价</Button>}</div><Link href="/workbench/plans/top-up" prefetch={false}>钱包充值</Link></dialog>}
  </Card>
}
