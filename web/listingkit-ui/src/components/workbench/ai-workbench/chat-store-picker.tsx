"use client";

import {useEffect,useRef,useState} from "react";
import {useInfiniteQuery} from "@tanstack/react-query";
import Link from "next/link";
import {Button} from "@/components/ui/button";
import {AccountDialog} from "@/components/workbench/account/account-dialog";
import styles from "./chat-store-picker.module.css";
import {getStoreConnection} from "@/lib/api/store-connection";
import {requestAIWorkbench,type AIScope} from "@/lib/api/ai-workbench";
import {aiStoreChoice,type WorkScopeOption} from "@/lib/contracts/ai-workbench";
import type {z} from "zod";

export type ChatStoreChoice = z.infer<typeof aiStoreChoice>;
export function chatStoreChoiceKey(scope:AIScope, context:string) {return `ai-workbench:store-choice:${JSON.stringify([scope.userId,scope.organizationId,context])}`;}
type StoreRow={option:WorkScopeOption;version:number};
export type ChatStoreState={choice:ChatStoreChoice|undefined; ready:boolean};

export function ChatStorePicker({scope,authorizationKey,context,disabled,baseStoreName,onChange}:{scope:AIScope;authorizationKey:string;context:string;disabled:boolean;baseStoreName?:string;onChange:(value:ChatStoreState)=>void}) {
 const [open,setOpen]=useState(false),[value,setValue]=useState<ChatStoreChoice>(),[name,setName]=useState(""),[error,setError]=useState("");
 const key=chatStoreChoiceKey(scope,context), generation=useRef(0),callback=useRef(onChange);
 callback.current=onChange;
 const identity={expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId};
 async function verify(choice:Extract<ChatStoreChoice,{kind:"store"}>,signal:AbortSignal) {
  const query=new URLSearchParams({kind:"store",storeId:choice.storeId});
  const [scopes,connection]=await Promise.all([requestAIWorkbench({route:"scope-options",method:"GET",path:`chat/conversations/scopes?${query}`,scope,signal}),getStoreConnection(identity,choice.storeId,signal)]);
  const item=scopes.options.length===1&&!scopes.incomplete?scopes.options[0]:undefined;
  if(!item||item.scope.kind!=="store"||item.scope.storeId!==choice.storeId||item.confirmation!==choice.scopeConfirmation||connection.connectionStatus!=="connected"||connection.version!==choice.connectionVersion||!connection.observedAt||connection.credentialExpiresAt&&Date.parse(connection.credentialExpiresAt)<=Date.now())throw new Error("unavailable");
  return item.storeName||"已连接店铺";
 }
 useEffect(()=>{
  const current=++generation.current,controller=new AbortController();
  setValue(undefined);setName("");setError("");callback.current({choice:undefined,ready:false});
  try {
   const raw=sessionStorage.getItem(key);
   if(raw===null){callback.current({choice:undefined,ready:true});return ()=>controller.abort();}
   const result=aiStoreChoice.safeParse(JSON.parse(raw));
   if(!result.success)throw new Error("invalid");
   setValue(result.data);
   if(result.data.kind==="none"){callback.current({choice:result.data,ready:true});return ()=>controller.abort();}
   void verify(result.data,controller.signal).then(label=>{if(current!==generation.current||controller.signal.aborted)return;setName(label);callback.current({choice:result.data,ready:true});}).catch(()=>{if(current!==generation.current||controller.signal.aborted)return;setName("");setError("所选店铺暂不可用，请重新选择。");callback.current({choice:undefined,ready:false});});
  } catch {setError("无法读取店铺选择，请检查浏览器存储。");callback.current({choice:undefined,ready:false});}
  return ()=>{generation.current++;controller.abort();};
 // Scope identifiers and authorizationKey form the complete identity boundary.
 // eslint-disable-next-line react-hooks/exhaustive-deps
 },[key,authorizationKey]);
 useEffect(()=>{
  if(value?.kind!=="store")return;
  let controller:AbortController|undefined;
  const refresh=()=>{
   if(document.visibilityState==="hidden")return;
   controller?.abort();controller=new AbortController();const current=++generation.current,signal=controller.signal;
   setName("");callback.current({choice:undefined,ready:false});
   void verify(value,signal).then(label=>{if(current!==generation.current||signal.aborted)return;setName(label);setError("");callback.current({choice:value,ready:true});}).catch(()=>{if(current!==generation.current||signal.aborted)return;setError("所选店铺暂不可用，请重新选择。");callback.current({choice:undefined,ready:false});});
  };
  window.addEventListener("focus",refresh);document.addEventListener("visibilitychange",refresh);
  return ()=>{controller?.abort();window.removeEventListener("focus",refresh);document.removeEventListener("visibilitychange",refresh);};
 // Recheck when returning to the page without querying a platform.
 // eslint-disable-next-line react-hooks/exhaustive-deps
 },[key,authorizationKey,value]);
 const rows=useInfiniteQuery({queryKey:["chat-store-choices",scope.userId,scope.organizationId,authorizationKey],enabled:open,initialPageParam:"",retry:false,gcTime:0,
  queryFn:async({signal,pageParam})=>{
   const params=new URLSearchParams({kind:"store",...(pageParam?{after:pageParam}:{})});
   const page=await requestAIWorkbench({route:"scope-options",method:"GET",path:`chat/conversations/scopes?${params}`,scope,signal});
   const checks=await Promise.allSettled(page.options.map(async option=>{
    if(option.scope.kind!=="store"||!option.scope.storeId)throw new Error("invalid");
    const c=await getStoreConnection(identity,option.scope.storeId,signal);
    return c.connectionStatus==="connected"&&c.version>0&&!!c.observedAt&&(!c.credentialExpiresAt||Date.parse(c.credentialExpiresAt)>Date.now())?{option,version:c.version}:null;
   }));
   return {items:checks.flatMap(result=>result.status==="fulfilled"&&result.value?[result.value]:[]),incomplete:page.incomplete||checks.some(result=>result.status==="rejected"),next:page.next};
  },getNextPageParam:page=>page.next||undefined});
 function save(choice:ChatStoreChoice,label:string) {
  try {sessionStorage.setItem(key,JSON.stringify(choice));generation.current++;setValue(choice);setName(label);setError("");callback.current({choice,ready:true});setOpen(false);}
  catch {setError("无法保存店铺选择，请检查浏览器存储。");callback.current({choice:undefined,ready:false});}
 }
 const visible=!rows.isFetching&&!rows.isError?rows.data?.pages.flatMap(page=>page.items):undefined;
 return <div>
  <Button type="button" variant="outline" disabled={disabled} onClick={()=>setOpen(true)} aria-haspopup="dialog" aria-expanded={open}>{value?.kind==="store"?(name||"店铺待确认"):!value&&baseStoreName?baseStoreName:"选择店铺"}</Button>
  {value?.kind==="store"||!value&&baseStoreName?<Button type="button" variant="ghost" disabled={disabled} onClick={()=>save({kind:"none"},"")}>取消选择</Button>:null}
  {error?<p role="status">{error}</p>:null}
  {open?<AccountDialog title="选择店铺" drawer className={styles.drawer} onClose={()=>setOpen(false)}><div className={styles.content}><p className={styles.intro}>选择已连接的店铺，为接下来的对话提供背景。</p>
   {rows.isPending||rows.isFetching?<p role="status">正在读取店铺…</p>:rows.isError?<p role="status">无法读取店铺，请重试。</p>:visible?.length===0?<p>{rows.data?.pages.some(page=>page.incomplete)?"暂未找到可选择的店铺。":"当前没有可选择的已连接店铺。"}</p>:null}
   {rows.data?.pages.some(page=>page.incomplete)?<p role="status">部分店铺暂无法确认连接状态。</p>:null}
   {visible?.map(({option,version}:StoreRow)=><Button className={styles.store} key={option.scope.storeId} type="button" variant="ghost" disabled={disabled} onClick={()=>save({kind:"store",storeId:option.scope.storeId!,scopeConfirmation:option.confirmation,connectionVersion:version},option.storeName||"已连接店铺")}>{option.storeName||"已连接店铺"}</Button>)}
   {rows.isError?<Button type="button" variant="outline" onClick={()=>void rows.refetch()}>重新读取</Button>:null}
   {rows.hasNextPage?<Button type="button" variant="outline" disabled={rows.isFetching} onClick={()=>void rows.fetchNextPage()}>加载更多</Button>:null}
   <div className={styles.connect}><Button asChild variant="outline"><Link href="/workbench/stores">连接新店铺</Link></Button></div>
  </div></AccountDialog>:null}
 </div>;
}
