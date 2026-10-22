"use client";
import {useEffect,useRef,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {z} from "zod";
import {ResourceDialog} from "@/components/workbench/resources/resource-dialog";
import {Button} from "@/components/ui/button";
import {Select} from "@/components/ui/select";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {StoreReadinessNotice} from "./store-readiness";
import {ObservationError,observationRequest,observationBeginSchema,observationID,observationCapabilitiesSchema,observationCommandSchema,observationSyncSchema,type ObservationKind} from "@/lib/api/store-observations";
import type {CatalogItem,StoreCenterScope} from "@/lib/api/store-center";
import styles from "./store-catalog.module.css";
const intentSchema=z.object({key:observationID,input:observationBeginSchema}).strict();
type Intent=z.infer<typeof intentSchema>;
const statusLabels:Record<string,string>={pending:"等待启动",running:"正在同步",completed:"遍历完成",partial:"部分结果",failed:"同步失败",suspended:"同步暂停"};
const inaccessibleReceipt=(error:unknown)=>error instanceof ObservationError&&((error.status===404&&error.code==="NOT_FOUND")||(error.status===403&&error.code==="PERMISSION_DENIED"));
export function StoreSyncDialog({store,scope,onClose,onChanged,initialKind="products"}:{initialKind?:ObservationKind;store:CatalogItem;scope:StoreCenterScope;onClose:()=>void;onChanged:()=>Promise<unknown>}){
 const context=useWorkbenchContext();
 const [selectedKind,setSelectedKind]=useState<ObservationKind>(initialKind),[all,setAll]=useState(false),[intent,setIntent]=useState<Intent|null>(null),[ready,setReady]=useState(false),[message,setMessage]=useState("");
 const key=`store-dialog-sync:${scope.organizationId}:${scope.userId}`,lifetime=useRef(new AbortController());
 useEffect(()=>{const controller=new AbortController();lifetime.current=controller;let active=true;void Promise.resolve().then(()=>{if(!active)return;try{const raw=sessionStorage.getItem(key);if(raw){const parsed=intentSchema.safeParse(JSON.parse(raw));if(!parsed.success)throw Error("invalid intent");setIntent(parsed.data)}setReady(true)}catch{setMessage("无法读取或保存原同步操作键，请检查浏览器存储。")}});return()=>{active=false;controller.abort()}},[key]);
 const kind=intent?.input.kind??selectedKind;
 const readinessStore=intent?(intent.input.stores.length===1&&intent.input.stores[0]===store.id?store:undefined):all?undefined:store;
 const client=useQueryClient();
 const unsupportedNewKind=!intent&&store.platform!=="shein"&&kind==="orders";
 const caps=useQuery({queryKey:["store-dialog-sync-capability",scope.organizationId,scope.userId,kind,context.permissions,context.storeObservationsAvailable],queryFn:({signal})=>observationRequest(`${kind}/capabilities`,scope,observationCapabilitiesSchema,signal),enabled:context.storeObservationsAvailable!==false&&context.permissions.includes(`workbench.store.${kind}.read`),retry:false,gcTime:0});
 const receipt=useQuery({queryKey:["store-dialog-sync-receipt",scope.organizationId,scope.userId,kind,intent?.key],queryFn:({signal})=>observationRequest(`${kind}/commands/${intent!.key}`,scope,observationCommandSchema,signal),enabled:ready&&Boolean(intent)&&caps.data?.available===true,retry:false,gcTime:0,refetchInterval:query=>query.state.data?.syncs.some(sync=>sync.status==="pending"||sync.status==="running")?3000:false});
 const start=useMutation({mutationFn:(captured:Intent)=>observationRequest(`${captured.input.kind}/syncs`,scope,observationCommandSchema,lifetime.current.signal,{key:captured.key,input:captured.input}),retry:false,onSuccess:()=>{if(!lifetime.current.signal.aborted){void receipt.refetch();void onChanged()}},onError:()=>{if(!lifetime.current.signal.aborted)setMessage("同步结果待确认。保留原命令键，可查看原状态或重试原请求。")}});
 const ensure=useMutation({mutationFn:(id:string)=>observationRequest(`${kind}/syncs/${id}/ensure`,scope,observationSyncSchema,lifetime.current.signal,{}),retry:false,onSuccess:()=>{if(!lifetime.current.signal.aborted)void receipt.refetch()}});
 const begin=()=>{
  if(!ready||start.isPending||!caps.data?.canSync||unsupportedNewKind)return;
  if(intent){start.mutate(intent);return}
  try{const captured:Intent={key:crypto.randomUUID(),input:{kind,stores:all?[]:[store.id]}};sessionStorage.setItem(key,JSON.stringify(captured));setIntent(captured);start.mutate(captured)}catch{setMessage("无法保存原同步操作键，同步尚未开始。")}
 };
 const finished=Boolean(receipt.data?.syncs.length && receipt.data.syncs.every(sync=>sync.status!=="pending"&&sync.status!=="running"));
 const reset=()=>{if(!finished)return;sessionStorage.removeItem(key);setIntent(null);setMessage("");start.reset()};
 const canDiscard=Boolean(intent&&!receipt.data&&!receipt.isFetching&&inaccessibleReceipt(receipt.error)&&!start.isPending&&!ensure.isPending&&(!start.error||inaccessibleReceipt(start.error)));
 const discard=()=>{if(!canDiscard)return;try{sessionStorage.removeItem(key)}catch{setMessage("无法清除此页记录，请检查浏览器存储。");return}client.removeQueries({queryKey:["store-dialog-sync-receipt",scope.organizationId,scope.userId,kind,intent!.key],exact:true});setIntent(null);setSelectedKind("products");setAll(false);setMessage("已停止本地跟踪；旧同步不会被取消，也未被判定为失败。");start.reset();ensure.reset()};
 return <ResourceDialog title="同步店铺数据" onClose={onClose} locked={start.isPending||ensure.isPending} className={styles.dialog}>
  <p>读取平台数据并保存观察结果，经营操作继续在平台处理。TEMU 与 Walmart 当前只开放商品同步；Walmart 库存缺项仍显示未知。</p>
  <label>同步数据<Select value={kind} disabled={Boolean(intent)||start.isPending} onChange={event=>setSelectedKind(event.target.value as ObservationKind)}><option value="products">商品与库存</option><option value="orders" disabled={!intent&&store.platform!=="shein"}>订单</option></Select></label>
  <label>同步范围<Select value={intent?(intent.input.stores.length?"original":"all"):all?"all":"current"} disabled={Boolean(intent)||start.isPending} onChange={event=>setAll(event.target.value==="all")}>{intent?.input.stores.length?<option value="original">原选定店铺：{intent.input.stores.map(id=>id===store.id?store.name:id).join("、")}</option>:null}<option value="current">当前店铺：{store.name}</option><option value="all">当前全部授权店铺</option></Select></label>
  <StoreReadinessNotice caps={{...caps,admitted:context.storeObservationsAvailable}} permissions={context.permissions} kind={kind} store={readinessStore}/>
  {kind==="orders"?<p className={styles.hint}>全托管消费者订单沿平台合同未开放。</p>:null}
  {intent?<p>原命令键：{intent.key}</p>:null}
  {canDiscard?<div><p>原同步回执不存在或当前成员无权访问。清除此页记录只停止本地跟踪，不取消旧同步或证明请求未受理。</p><Button variant="outline" onClick={discard}>清除此页同步记录</Button></div>:null}
  {receipt.data?<ul className={styles.groupList}>{receipt.data.syncs.map(sync=><li key={sync.id}><span>{sync.storeId===store.id?store.name:sync.storeId}<small>{statusLabels[sync.status]??sync.status} · 已读取 {sync.progress.pages} 页{sync.errorCode?` · ${sync.errorCode}`:""}</small></span>{sync.status==="pending"||sync.status==="running"?<Button size="sm" variant="outline" disabled={ensure.isPending||!caps.data?.canSync} onClick={()=>ensure.mutate(sync.id)}>恢复执行</Button>:null}</li>)}</ul>:null}
  {message||receipt.isError||ensure.isError?<p className={styles.error} role="status">{message||"当前无法确认原同步状态，请保留原键后重试。"}</p>:null}
  <footer>{intent?<Button variant="outline" disabled={receipt.isFetching} onClick={()=>void receipt.refetch()}>查看原同步状态</Button>:null}{finished?<Button variant="outline" onClick={reset}>准备下一次同步</Button>:null}<Button disabled={!ready||start.isPending||!caps.data?.canSync||unsupportedNewKind||Boolean(receipt.data)} onClick={begin}>{intent?"重试原同步请求":"开始同步"}</Button></footer>
 </ResourceDialog>;
}
