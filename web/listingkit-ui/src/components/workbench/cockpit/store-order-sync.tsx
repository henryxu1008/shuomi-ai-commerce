"use client";

import {useEffect,useState} from "react";
import {useQuery,useQueryClient} from "@tanstack/react-query";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {observationRequest,observationBeginSchema,observationID,observationCommandSchema,observationSyncSchema,observationCapabilitiesSchema,ObservationError,type ObservationScope} from "@/lib/api/store-observations";
import styles from "./store-order-sync.module.css";

const intentSchema=z.object({key:observationID,input:observationBeginSchema}).strict().refine(v=>v.input.kind==="orders"&&v.input.stores.length===1&&!!v.input.start&&!!v.input.end);
type Intent=z.infer<typeof intentSchema>;
const active=(status:string)=>status==="pending"||status==="running";
const labels:Record<string,string>={pending:"等待启动",running:"正在读取订单",completed:"遍历完成，已更新所选范围统计",partial:"覆盖不足：仅保存部分订单，请查看金额区提示",failed:"同步失败，未确认完整覆盖",suspended:"同步暂停，请检查店铺授权和订单权限"};
function requestInput(storeId:string,startDate:string,endDate:string){
 const next=new Date(`${endDate}T00:00:00Z`);next.setUTCDate(next.getUTCDate()+1);
 return observationBeginSchema.parse({kind:"orders",stores:[storeId],start:`${startDate}T00:00:00+08:00`,end:`${next.toISOString().slice(0,10)}T00:00:00+08:00`});
}

export function StoreOrderSync({platform,scope,storeId,startDate,endDate,permissions}:{platform:"walmart"|"tiktokshop";scope:ObservationScope;storeId:string;startDate:string;endDate:string;permissions:readonly string[]}){
 const client=useQueryClient();
 const label=platform==="walmart"?"Walmart US":"TikTok Shop US";
 const storageKey=JSON.stringify([`${platform}-order-sync`,scope.organizationId,scope.userId,storeId]);
 const [intent,setIntent]=useState<Intent>();
 const [resolvedKey,setResolvedKey]=useState("");
 const [hydrated,setHydrated]=useState(false);
 const [busy,setBusy]=useState(false);
 const [error,setError]=useState("");
 const [storageError,setStorageError]=useState(false);
 useEffect(()=>{
  let mounted=true;
  void Promise.resolve().then(()=>{if(!mounted)return;
  try{const raw=sessionStorage.getItem(storageKey);if(raw){const saved=intentSchema.parse(JSON.parse(raw));if(saved.input.stores[0]!==storeId)throw new Error();setIntent(saved);}}
  catch{setStorageError(true);setError("原同步记录不可用，请恢复本页会话存储后重试。");}
  setHydrated(true);
  });
  return ()=>{mounted=false;};
 },[storageKey,storeId]);
 const prefix=[`${platform}-order-sync`,scope.organizationId,scope.userId,storeId];
 const caps=useQuery({queryKey:[...prefix,"capabilities",permissions],queryFn:({signal})=>observationRequest("orders/capabilities",scope,observationCapabilitiesSchema,signal),retry:false});
 const receipt=useQuery({queryKey:[...prefix,"command",intent?.key],enabled:hydrated&&!!intent&&intent.key!==resolvedKey,
  queryFn:async({signal})=>{
   const result=await observationRequest(`orders/commands/${intent!.key}`,scope,observationCommandSchema,signal);
   if(result.input.kind!=="orders"||result.input.stores.length!==1||result.input.stores[0]!==storeId||result.syncs.length!==1||result.syncs[0].storeId!==storeId||new Date(result.input.start!).getTime()!==new Date(intent!.input.start!).getTime()||new Date(result.input.end!).getTime()!==new Date(intent!.input.end!).getTime())throw new Error("original order scope changed");
   return result;
  },retry:false,refetchInterval:q=>q.state.data?.syncs.some(s=>active(s.status))?2000:false,gcTime:0});
 const sync=receipt.data?.syncs[0];
 const terminal=!!sync&&!active(sync.status)||!!intent&&intent.key===resolvedKey;
 useEffect(()=>{
  if(!terminal||!intent)return;
  let mounted=true,storageFailed=false;
  try{sessionStorage.removeItem(storageKey);}catch{storageFailed=true;}
  void Promise.resolve().then(()=>{if(!mounted)return;setResolvedKey(intent.key);if(storageFailed){setStorageError(true);setError("同步已完成，但无法清理会话记录。请恢复本页会话存储后重试。");}});
  void client.invalidateQueries({queryKey:["operations-cockpit",scope.userId,scope.organizationId]});
  return ()=>{mounted=false;};
 },[terminal,intent,storageKey,client,scope.userId,scope.organizationId]);
 const canSync=!!caps.data?.canSync&&permissions.includes("workbench.store.orders.sync");
 async function submit(){
  setBusy(true);setError("");
  try{
   let original=intent;
   if(!original||terminal){
    original={key:crypto.randomUUID(),input:requestInput(storeId,startDate,endDate)};
    sessionStorage.setItem(storageKey,JSON.stringify(original));setIntent(original);
   }else{
    // A lost POST is first reconciled by the original identity; only a definite
    // not-found permits replaying that exact key and original payload.
    const existing=await receipt.refetch();
    if(existing.data)return;
    if(!(existing.error instanceof ObservationError&&existing.error.status===404&&existing.error.code==="NOT_FOUND"))throw existing.error;
   }
   await observationRequest("orders/syncs",scope,observationCommandSchema,undefined,original);
   await client.invalidateQueries({queryKey:[...prefix,"command",original.key]});
  }catch{setError("同步结果尚未确认，请确认原同步结果；不会另建重复任务。");}
  finally{setBusy(false);}
 }
 async function ensure(){if(!sync)return;setBusy(true);setError("");try{await observationRequest(`orders/syncs/${sync.id}/ensure`,scope,observationSyncSchema,undefined,{});await receipt.refetch();}catch{setError("继续同步结果尚未确认，请继续原任务。");}finally{setBusy(false);}}
 return <section className={styles.bar} aria-label={`${label} 订单同步`}>
  <div><strong>{label} 订单</strong><p>{platform==="tiktokshop"?"仅当前支持的 US 跨境店":"仅手动读取所选店铺"} · {startDate} — {endDate}（UTC+8）</p>{intent&&<p>原任务日期：{intent.input.start?.slice(0,10)} — {new Date(new Date(intent.input.end!).getTime()-1).toLocaleDateString("sv-SE",{timeZone:"Asia/Shanghai"})}</p>}</div>
  <div className={styles.actions}>
   <Button variant="outline" disabled={!hydrated||storageError||busy||!canSync||!!sync&&active(sync.status)} onClick={()=>void submit()}>{busy?"正在确认同步…":intent&&!terminal?"确认原同步结果":"同步所选日期订单"}</Button>
   {sync&&active(sync.status)&&canSync&&<Button variant="outline" disabled={busy} onClick={()=>void ensure()}>继续原同步</Button>}
   <span role="status">{sync?labels[sync.status]:caps.isPending?"正在确认同步能力…":!canSync?"同步尚未配置或无同步权限":""}</span>
   {error&&(!terminal||storageError)&&<p role="alert">{error}</p>}
  </div>
 </section>;
}
