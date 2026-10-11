"use client";
import {useEffect,useRef,useState} from "react";
import {useMutation,useQuery} from "@tanstack/react-query";
import {z} from "zod";
import {ResourceDialog} from "@/components/workbench/resources/resource-dialog";
import {Button} from "@/components/ui/button";
import {Select} from "@/components/ui/select";
import {observationRequest,observationBeginSchema,observationID,observationCapabilitiesSchema,observationCommandSchema,observationSyncSchema,type ObservationKind} from "@/lib/api/store-observations";
import type {CatalogItem,StoreCenterScope} from "@/lib/api/store-center";
import styles from "./store-catalog.module.css";
const intentSchema=z.object({key:observationID,input:observationBeginSchema}).strict();
type Intent=z.infer<typeof intentSchema>;
const statusLabels:Record<string,string>={pending:"等待启动",running:"正在同步",completed:"遍历完成",partial:"部分结果",failed:"同步失败",suspended:"同步暂停"};
export function StoreSyncDialog({store,scope,onClose,onChanged,initialKind="products"}:{initialKind?:ObservationKind;store:CatalogItem;scope:StoreCenterScope;onClose:()=>void;onChanged:()=>Promise<unknown>}){
 const [selectedKind,setSelectedKind]=useState<ObservationKind>(initialKind),[all,setAll]=useState(false),[intent,setIntent]=useState<Intent|null>(null),[ready,setReady]=useState(false),[message,setMessage]=useState("");
 const key=`store-dialog-sync:${scope.organizationId}:${scope.userId}`,lifetime=useRef(new AbortController());
 useEffect(()=>{const controller=new AbortController();lifetime.current=controller;let active=true;void Promise.resolve().then(()=>{if(!active)return;try{const raw=sessionStorage.getItem(key);if(raw){const parsed=intentSchema.safeParse(JSON.parse(raw));if(!parsed.success)throw Error("invalid intent");setIntent(parsed.data)}setReady(true)}catch{setMessage("无法读取或保存原同步操作键，请检查浏览器存储。")}});return()=>{active=false;controller.abort()}},[key]);
 const kind=intent?.input.kind??selectedKind;
 const caps=useQuery({queryKey:["store-dialog-sync-capability",scope.organizationId,scope.userId,kind],queryFn:({signal})=>observationRequest(`${kind}/capabilities`,scope,observationCapabilitiesSchema,signal),retry:false,gcTime:0});
 const receipt=useQuery({queryKey:["store-dialog-sync-receipt",scope.organizationId,scope.userId,kind,intent?.key],queryFn:({signal})=>observationRequest(`${kind}/commands/${intent!.key}`,scope,observationCommandSchema,signal),enabled:ready&&Boolean(intent)&&caps.data?.available===true,retry:false,gcTime:0,refetchInterval:query=>query.state.data?.syncs.some(sync=>sync.status==="pending"||sync.status==="running")?3000:false});
 const start=useMutation({mutationFn:(captured:Intent)=>observationRequest(`${captured.input.kind}/syncs`,scope,observationCommandSchema,lifetime.current.signal,{key:captured.key,input:captured.input}),retry:false,onSuccess:()=>{if(!lifetime.current.signal.aborted){void receipt.refetch();void onChanged()}},onError:()=>{if(!lifetime.current.signal.aborted)setMessage("同步结果待确认。保留原命令键，可查看原状态或重试原请求。")}});
 const ensure=useMutation({mutationFn:(id:string)=>observationRequest(`${kind}/syncs/${id}/ensure`,scope,observationSyncSchema,lifetime.current.signal,{}),retry:false,onSuccess:()=>{if(!lifetime.current.signal.aborted)void receipt.refetch()}});
 const begin=()=>{
  if(!ready||start.isPending||!caps.data?.canSync)return;
  if(intent){start.mutate(intent);return}
  try{const captured:Intent={key:crypto.randomUUID(),input:{kind,stores:all?[]:[store.id]}};sessionStorage.setItem(key,JSON.stringify(captured));setIntent(captured);start.mutate(captured)}catch{setMessage("无法保存原同步操作键，同步尚未开始。")}
 };
 const finished=Boolean(receipt.data?.syncs.length && receipt.data.syncs.every(sync=>sync.status!=="pending"&&sync.status!=="running"));
 const reset=()=>{if(!finished)return;sessionStorage.removeItem(key);setIntent(null);setMessage("");start.reset()};
 return <ResourceDialog title="同步店铺数据" onClose={onClose} locked={start.isPending||ensure.isPending} className={styles.dialog}>
  <p>读取 SHEIN 商品或订单并保存观察结果，经营操作继续在平台处理。</p>
  <label>同步数据<Select value={kind} disabled={Boolean(intent)||start.isPending} onChange={event=>setSelectedKind(event.target.value as ObservationKind)}><option value="products">商品与库存</option><option value="orders">订单</option></Select></label>
  <label>同步范围<Select value={intent?(intent.input.stores.length?"original":"all"):all?"all":"current"} disabled={Boolean(intent)||start.isPending} onChange={event=>setAll(event.target.value==="all")}>{intent?.input.stores.length?<option value="original">原选定店铺：{intent.input.stores.map(id=>id===store.id?store.name:id).join("、")}</option>:null}<option value="current">当前店铺：{store.name}</option><option value="all">当前全部授权店铺</option></Select></label>
  <p className={styles.hint}>{caps.data?.available?`当前开放站点：${caps.data.site}。同步进度以平台观察的真实检查点为准。`:caps.isPending?"正在检查同步能力…":"当前平台观察能力未配置或不可用。"}{kind==="orders"?"全托管消费者订单沿平台合同未开放。":""}</p>
  {intent?<p>原命令键：{intent.key}</p>:null}
  {receipt.data?<ul className={styles.groupList}>{receipt.data.syncs.map(sync=><li key={sync.id}><span>{sync.storeId===store.id?store.name:sync.storeId}<small>{statusLabels[sync.status]??sync.status} · 已读取 {sync.progress.pages} 页{sync.errorCode?` · ${sync.errorCode}`:""}</small></span>{sync.status==="pending"||sync.status==="running"?<Button size="sm" variant="outline" disabled={ensure.isPending||!caps.data?.canSync} onClick={()=>ensure.mutate(sync.id)}>恢复执行</Button>:null}</li>)}</ul>:null}
  {message||receipt.isError||ensure.isError?<p className={styles.error} role="status">{message||"当前无法确认原同步状态，请保留原键后重试。"}</p>:null}
  <footer>{intent?<Button variant="outline" disabled={receipt.isFetching} onClick={()=>void receipt.refetch()}>查看原同步状态</Button>:null}{finished?<Button variant="outline" onClick={reset}>准备下一次同步</Button>:null}<Button disabled={!ready||start.isPending||!caps.data?.canSync||Boolean(receipt.data)} onClick={begin}>{intent?"重试原同步请求":"开始同步"}</Button></footer>
 </ResourceDialog>;
}
