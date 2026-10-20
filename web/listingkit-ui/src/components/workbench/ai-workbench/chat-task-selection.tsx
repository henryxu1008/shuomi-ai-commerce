"use client";
import {useEffect,useRef,useState} from "react";
import {CollectionPicker} from "../pod/collection-picker";
import {ContentSelectionFields} from "../agents/content-selection-fields";
import {requestAIWorkbench,type AIScope} from "@/lib/api/ai-workbench";
import type {AITaskSubject} from "@/lib/contracts/ai-workbench";
import type {CollectionItem} from "@/lib/contracts/product-collection";
import type {ContentSelection} from "@/lib/contracts/content-selection";

function contentSubject(item:CollectionItem){
 if(item.source.kind==="own")return {kind:"own_product" as const,id:item.id};
 if(item.source.kind==="acquisition"&&item.source.operationId)return {kind:"acquisition" as const,id:item.source.operationId};
 return null;
}

export function ChatTaskSelection({scope,subject,parameters,onSubject,onParameters,disabled,authorizationKey}:{scope:AIScope;subject:AITaskSubject|null;parameters:ContentSelection;onSubject:(subject:AITaskSubject|null)=>void;onParameters:(parameters:ContentSelection)=>void;disabled:boolean;authorizationKey:string}) {
 const abort=useRef<AbortController|null>(null);
 const [loading,setLoading]=useState(false),[error,setError]=useState("");
 useEffect(()=>()=>abort.current?.abort(),[]);
 async function select(item:CollectionItem) {
  abort.current?.abort();setLoading(false);onSubject(null);setError("");
  const selected=contentSubject(item);
  if(!selected){setError("该对象尚未接入内容任务，请选择自有商品或已保存的源商品。");return}
  const {kind,id}=selected;
  const controller=new AbortController();abort.current=controller;setLoading(true);
  try {
   const result=await requestAIWorkbench({route:"subject-read",method:"GET",path:`chat/subjects/${kind}/${id}`,scope,signal:controller.signal});
   if(controller.signal.aborted)return;
   if(result.subject.kind!==kind||result.subject.id!==id)throw new Error("source changed");
   onSubject(result.subject);
  } catch {if(!controller.signal.aborted)setError("无法确认当前商品版本，请重新选择。")} finally {if(!controller.signal.aborted)setLoading(false)}
 }
 return <fieldset disabled={disabled} className="space-y-4"><legend>任务对象</legend>
  <CollectionPicker authorizationKey={authorizationKey} scope={scope} kind="product" isEligible={item=>contentSubject(item)!==null} emptyMessage="当前页没有可用于内容任务的自有商品或已保存源商品。" disabled={disabled||loading} onSelect={item=>void select(item)}/>
  {loading?<p role="status">正在确认商品当前版本…</p>:null}
  {error?<p role="alert">{error}</p>:null}
  {subject?<p>已选择{subject.kind==="own_product"?"自有商品":"源商品"} · {subject.id} · 版本 {subject.catalogVersion}</p>:<p>从我的数据选择商品，来源信息由原商品记录保留。</p>}
  <ContentSelectionFields value={parameters} onChange={onParameters} disabled={disabled||loading}/>
 </fieldset>;
}
