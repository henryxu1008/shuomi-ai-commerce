"use client";

import {useEffect,useRef,useState} from "react";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {CollectionAPIError,mutateCollection,readCollectionOperation,type CollectionIntent,type CollectionScope} from "@/lib/api/product-collection";
import {collectionCommandSchema,collectionReceiptSchema,type CollectionCommand} from "@/lib/contracts/product-collection";
import type {z} from "zod";

export const collectionFailureText:Record<string,string>={OUTCOME_UNKNOWN:"结果待核实",INTENT_STORAGE_UNAVAILABLE:"无法保存原请求，请恢复浏览器存储后重试。",REVISION_CONFLICT:"资料已发生变化，请刷新后重试。",PERMISSION_DENIED:"当前权限不足。",NOT_FOUND:"未找到当前身份下的记录。",ORGANIZATION_CONTEXT_CHANGED:"企业上下文已变化。",IDENTITY_CONTEXT_CHANGED:"登录身份已变化。",INVALID_REQUEST:"请检查填写内容。",STORAGE_LIMIT_EXCEEDED:"企业存储空间不足，请扩容后重试。"};
export function useCollectionCommand(scope:CollectionScope,onSaved:(receipt:z.infer<typeof collectionReceiptSchema>,intent:CollectionIntent)=>void,blocked=false){
 const context=useWorkbenchContext(),pending=context.pendingCollectionIntent;
 const [busy,setBusy]=useState(false),[error,setError]=useState<string|null>(null);
 const alive=useRef(true),flight=useRef(false),abort=useRef<AbortController|null>(null),saved=useRef(onSaved);
 useEffect(()=>{saved.current=onSaved},[onSaved]);
 useEffect(()=>{alive.current=true;return()=>{alive.current=false;abort.current?.abort()}},[]);
 useEffect(()=>context.registerOrganizationSwitchGuard(()=>!flight.current),[context]);
 const foreign=!!pending&&(pending.userId!==scope.userId||pending.organizationId!==scope.organizationId);
 const locked=busy||blocked||!!pending||!context.collectionIntentReady||!context.permissions.includes("workbench.collection.manage");
 async function run(input:CollectionCommand,original?:CollectionIntent,verify=false){
  if(!alive.current||flight.current||!verify&&blocked||!context.collectionIntentReady||foreign||pending&&!original||!verify&&!context.permissions.includes("workbench.collection.manage"))return;
  if(original&&(original.userId!==scope.userId||original.organizationId!==scope.organizationId||original!==pending))return;
  const parsed=collectionCommandSchema.safeParse(input);if(!parsed.success){setError("INVALID_REQUEST");return}
  const intent=original??{...scope,key:crypto.randomUUID(),command:parsed.data};
  if(!context.setPendingCollectionIntent(intent)){setError("INTENT_STORAGE_UNAVAILABLE");return}
  const controller=new AbortController();abort.current=controller;flight.current=true;setBusy(true);setError(null);
  try{
   const receipt=await(verify?readCollectionOperation(intent,controller.signal):mutateCollection(intent,controller.signal));
   if(!alive.current)return;
   if(!context.setPendingCollectionIntent(null)){setError("INTENT_STORAGE_UNAVAILABLE");return}
   saved.current(receipt,intent);return receipt;
  }catch(failure){
   if(!alive.current)return;
   const code=failure instanceof CollectionAPIError?failure.code:"DEPENDENCY_UNAVAILABLE";
   if(!verify&&failure instanceof CollectionAPIError&&failure.status>=400&&failure.status<500&&code!=="OUTCOME_UNKNOWN")setError(context.setPendingCollectionIntent(null)?code:"INTENT_STORAGE_UNAVAILABLE");
   else setError(verify?code:"OUTCOME_UNKNOWN");
  }finally{flight.current=false;if(abort.current===controller)abort.current=null;if(alive.current)setBusy(false)}
 }
 return {run,pending,busy,error,foreign,locked};
}
