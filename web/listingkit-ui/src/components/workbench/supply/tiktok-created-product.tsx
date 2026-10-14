"use client";
import {useEffect,useRef,useState} from "react";
import type {TikTokEffect} from "@/lib/contracts/tiktokshop-model";
import type {SupplyPublication} from "@/lib/contracts/supply-chain";
import {readSupplyPublicationStatus,type SupplyScope} from "@/lib/api/supply-chain";
const labels:Record<string,string>={DRAFT:"平台草稿",PENDING:"审核中",FAILED:"审核未通过",ACTIVATE:"平台当前在售",SELLER_DEACTIVATED:"卖家已下架",PLATFORM_DEACTIVATED:"平台已下架",FREEZE:"已冻结",DELETED:"已删除",SCHEDULED:"已安排发布"};
const auditLabels:Record<string,string>={NONE:"未提交审核",AUDITING:"审核中",FAILED:"审核未通过",PRE_APPROVED:"预审通过，仍有刊登前置条件",APPROVED:"审核通过"};
const reasonLabels:Record<string,string>={KYC_PENDING:"商家身份审核待完成",RESTRICTED_CATEGORY_PENDING:"限制类目资质待审批",PAYMENT_ACCOUNT_NOT_LINKED:"收款账户未关联",SHOP_INACTIVE:"店铺未激活",W8_NOT_CONFIGURED:"W-8 税务信息待配置"};
export function TikTokCreatedProduct({value,scope,sourceId,storeId}:{value:TikTokEffect;scope?:SupplyScope;sourceId?:string;storeId?:string}){
 const [observation,setObservation]=useState<SupplyPublication["tiktokStatus"]>();
 const [busy,setBusy]=useState(false),[error,setError]=useState(false);const active=useRef<AbortController|null>(null);
 useEffect(()=>()=>{active.current?.abort()},[]);
 async function queryStatus(){if(!scope||!sourceId||!storeId||active.current)return;const c=new AbortController();active.current=c;setBusy(true);setError(false);try{const result=await readSupplyPublicationStatus(scope,sourceId,storeId,c.signal);if(!c.signal.aborted)setObservation(result.tiktokStatus)}catch{if(!c.signal.aborted)setError(true)}finally{active.current=null;if(!c.signal.aborted)setBusy(false)}}
 const result=value.publication;
 const status=result?.state==="submitted_for_review"?"已提交美国站审核":result?.state==="remote_draft"?"已创建美国站草稿，平台校验未通过":result?.state==="publication_failed"?"美国同步未成功，全球商品已保留":value.localSaveMode==="AS_DRAFT"?"已创建平台草稿":value.local?"已创建商品，审核状态待平台确认":value.global?"已创建全球商品，尚未确认美国刊登":"已保留图片上传回执";
 return <div className="mt-2 space-y-1 break-all text-xs text-slate-600"><p>{status}</p>{value.global?<p>全球商品：{value.global.id}</p>:null}{value.local?<><p>商品：{value.local.id}</p>{value.local.skus.map(s=><p key={s.id}>{s.seller_sku} → {s.id}</p>)}</>:null}{result?.productId?<><p>美国商品：{result.productId}</p>{result.skus.map(s=><p key={s.localSkuId}>全球 SKU {s.globalSkuId} → 美国 SKU {s.localSkuId}</p>)}</>:null}<p>已保存原创建结果。平台当前状态需单独查询。</p>{scope&&sourceId&&storeId&&(value.local?.id||result?.productId)?<button type="button" disabled={busy} onClick={()=>void queryStatus()} className="text-blue-600 disabled:opacity-50">{busy?"正在查询平台状态":"查询平台当前状态"}</button>:null}{observation?<div role="status"><p>{labels[observation.status]}；{auditLabels[observation.auditStatus]}</p>{observation.reasons.map(reason=><p key={reason}>{reasonLabels[reason]}</p>)}<p>查询时间：{new Date(observation.observedAt).toLocaleString()}</p></div>:null}{error?<p role="alert">平台状态查询未成功，请稍后重试；原创建结果已保留。</p>:null}</div>;
}
