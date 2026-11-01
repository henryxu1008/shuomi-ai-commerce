"use client";
import {useEffect,useRef,useState} from "react";
import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Select} from "@/components/ui/select";
import {Card} from "@/components/ui/card";
import {DiagnosticReport} from "./diagnostic-report";
import {offlineRequest,OfflineRecordError,readOfflineIntent,writeOfflineIntent} from "@/lib/api/offline-shein-records";
import {offlineSaved,offlineReceipt,type OfflineSaved,type OfflineIntent} from "@/lib/contracts/offline-shein-records";
import {catalogSchema,storeCenterRequest} from "@/lib/api/store-center";
import type {CollectionItem} from "@/lib/contracts/product-collection";
import type {ReportScope} from "@/lib/contracts/report-center";
const message=(e:unknown)=>e instanceof OfflineRecordError?({OUTCOME_UNKNOWN:"结果未知，请查询原请求；不要重新生成。",NOT_FOUND:"暂未找到原记录；不能据此认定原请求未提交，请继续查询原键。",NOT_READY:"该准确商品版本尚无已批准素材，暂不能生成。",FORBIDDEN:"当前来源或权限已失效。",INTENT_STORAGE_UNAVAILABLE:"无法保留原请求，已停止提交。"}[e.code]??"离线诊断暂不可用，请稍后重试。"):"离线诊断暂不可用，请稍后重试。";
export function OfflineDiagnosticPanel({item,recordId}:{item?:CollectionItem;recordId?:string}){
 const c=useWorkbenchContext(),scope=c.user&&c.effectiveOrganization?{userId:c.user.id,organizationId:c.effectiveOrganization.id}:null;
 if(!scope||c.isLoading||c.isSwitching||c.error||c.blockingError||c.selectionRequired)return null;
 const read=c.permissions.includes("listingkit.admin.read"),write=read&&["listingkit.admin.write","workbench.collection.read","workbench.store.read"].every(p=>c.permissions.includes(p));
 if(!read)return <p>当前用户没有离线诊断读取权限。</p>;
 return <ScopedOfflinePanel key={JSON.stringify([scope,c.roles,c.permissions,item?.id,item?.source.version,recordId])} scope={scope} item={item} recordId={recordId} write={write} report={c.permissions.includes("workbench.report.read")&&c.permissions.includes("workbench.report.manage")}/>;
}
function ScopedOfflinePanel({scope,item,recordId,write,report}:{scope:ReportScope;item?:CollectionItem;recordId?:string;write:boolean;report:boolean}){
 const [stores,setStores]=useState<{id:string;name:string}[]>([]),[page,setPage]=useState(1),[hasMore,setHasMore]=useState(false),[storeId,setStoreId]=useState(""),[purpose,setPurpose]=useState<"save_draft"|"publish">("save_draft");
 const recovery=useQuery({queryKey:["offline-record-intent",scope.userId,scope.organizationId,item?.id,recordId],queryFn:()=>readOfflineIntent(scope),retry:false,gcTime:0,staleTime:Infinity,refetchOnWindowFocus:false,refetchOnReconnect:false});
 const [intentOverride,setIntent]=useState<OfflineIntent|null|undefined>(),[saved,setSaved]=useState<OfflineSaved|null>(null),[busy,setBusy]=useState(false),[error,setError]=useState("");
 const intent=intentOverride===undefined?recovery.data??null:intentOverride,ready=recovery.isSuccess;
 const alive=useRef(false),controller=useRef<AbortController|null>(null);
 useEffect(()=>{alive.current=true;return()=>{alive.current=false;controller.current?.abort()}},[]);
 useEffect(()=>{if(!write||!item)return;const abort=new AbortController();let active=true;void storeCenterRequest(`catalog?page=${page}&pageSize=50&platform=shein&status=active`,scope,catalogSchema,{signal:abort.signal}).then(v=>{if(active){setStores(v.items.map(s=>({id:s.id,name:s.name})));setHasMore(v.pagination.total>page*50)}}).catch(()=>{if(active)setError("无法读取当前可用的 SHEIN 店铺。")});return()=>{active=false;abort.abort()}},[scope,item,write,page]);
 useEffect(()=>{if(!recordId)return;const abort=new AbortController();let active=true;void offlineRequest(scope,`/${recordId}`,offlineSaved,abort.signal).then(v=>{if(active)setSaved(v)}).catch(e=>{if(active)setError(message(e))});return()=>{active=false;abort.abort()}},[scope,recordId]);
 async function generate(){
  if(!item||!ready||intent||busy||!write||!storeId)return;
  const next:OfflineIntent={key:crypto.randomUUID(),collectionItemId:item.id,input:{product_key:item.source.productKey,snapshot_version:item.source.version,store_id:storeId,country:"US",language:"en",action:purpose}};
  try{writeOfflineIntent(scope,next);setIntent(next)}catch(e){setError(message(e));return}
  const abort=new AbortController();controller.current=abort;setBusy(true);setError("");setSaved(null);
  let created=false;
  try{
   const receipt=await offlineRequest(scope,"",offlineReceipt,abort.signal,next);
   created=true;
   const value=await offlineRequest(scope,`/commands/${next.key}`,offlineSaved,abort.signal);
   if(value.recordId!==receipt.record_id||value.sourceVersion!==receipt.input_hash||JSON.stringify(value.input)!==JSON.stringify(next.input))throw new OfflineRecordError("OUTCOME_UNKNOWN");
   if(alive.current){writeOfflineIntent(scope,null);setIntent(null);setSaved(value)}
  }catch(e){if(alive.current){setError(message(e));
   if(!created&&e instanceof OfflineRecordError&&["NOT_READY","FORBIDDEN","NOT_FOUND","INVALID_REQUEST","COMMAND_CONFLICT","TOO_LARGE"].includes(e.code)){
    try{writeOfflineIntent(scope,null);setIntent(null)}catch(storage){setError(message(storage))}
   }}}
  finally{if(alive.current)setBusy(false)}
 }
 async function recover(){
  if(!intent||busy)return;const original=intent,abort=new AbortController();controller.current=abort;setBusy(true);setError("");
  try{const value=await offlineRequest(scope,`/commands/${original.key}`,offlineSaved,abort.signal);if(JSON.stringify(value.input)!==JSON.stringify(original.input))throw new OfflineRecordError("OUTCOME_UNKNOWN");if(alive.current){writeOfflineIntent(scope,null);setIntent(null);setSaved(value)}}catch(e){if(alive.current)setError(message(e))}finally{if(alive.current)setBusy(false)}
 }
 return <Card className="my-5 space-y-4 p-5"><h2>SHEIN 离线诊断</h2><p>按原 US / en 范围，针对准确商品版本和已批准素材生成本地结果；不包含后来修改的平台资料，也不代表发布许可。</p>
 {item&&write?<div className="flex flex-wrap items-end gap-3"><label>店铺<Select aria-label="离线诊断店铺" value={storeId} onChange={e=>setStoreId(e.target.value)} disabled={!ready||busy||!!intent}><option value="">选择 SHEIN 店铺</option>{stores.map(s=><option key={s.id} value={s.id}>{s.name}</option>)}</Select></label><Button variant="ghost" disabled={page===1||busy} onClick={()=>{setPage(p=>p-1);setStoreId("")}}>上一页店铺</Button><Button variant="ghost" disabled={!hasMore||busy} onClick={()=>{setPage(p=>p+1);setStoreId("")}}>下一页店铺</Button><label>检查目的<Select aria-label="离线诊断目的" value={purpose} onChange={e=>setPurpose(e.target.value as typeof purpose)} disabled={busy||!!intent}><option value="save_draft">本地草稿规则</option><option value="publish">发布规则（仅离线检查）</option></Select></label><Button disabled={!ready||busy||!!intent||!storeId} onClick={()=>void generate()}>生成离线诊断</Button></div>:item?<p>生成需要当前商品来源、店铺读取及离线诊断写入权限。</p>:null}
 {error||recovery.isError?<p role="alert">{error||message(recovery.error)}</p>:null}{intent?<div role="status"><p>有待确认的原生成请求，查询完成前不能创建新请求。</p><Button disabled={busy} variant="outline" onClick={()=>void recover()}>查询原请求</Button></div>:null}
 {saved?<><p>以下为保存时的诊断结果。</p><p>商品版本 {saved.input.snapshot_version} · 店铺 {saved.input.store_id}</p><DiagnosticReport report={saved.diagnostic} recordId={saved.recordId}/>{report?<Button asChild><Link prefetch={false} href={`/workbench/ai/reports?sourceKind=SHEIN_RECORD&sourceId=${saved.recordId}`}>前往我的报告保存此结果</Link></Button>:<p>保存报告还需要当前报告读取和管理权限。</p>}</>:null}
 </Card>;
}
