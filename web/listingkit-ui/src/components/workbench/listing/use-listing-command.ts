"use client";
import { useEffect, useRef, useState } from "react";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { requiresDraftPurpose,SupplyAPIError, supplyCommand, readSupplyCommand, type SupplyScope, type SupplyIntent } from "@/lib/api/supply-chain";
import { parseSupplyIntent } from "@/lib/api/supply-intent";

export function useListingCommand(scope:SupplyScope,onSaved:(result:unknown,intent:SupplyIntent)=>void){
 const context=useWorkbenchContext();
 const pending=context.pendingSupplyIntent;
 const [busy,setBusy]=useState(false);const [error,setError]=useState<string|null>(null);
 const alive=useRef(true);const active=useRef(false);const abort=useRef<AbortController|null>(null);
 const saved=useRef(onSaved);
 useEffect(()=>{saved.current=onSaved},[onSaved]);
 const pendingSource=pending?.draftSourceId??(pending?.command as {sourceId?:string;request?:{sourceId?:string}}|undefined)?.sourceId??(pending?.command as {request?:{sourceId?:string}}|undefined)?.request?.sourceId;
 const unresolvedPurpose=!!pending&&requiresDraftPurpose(pending);
 const purposeMismatch=unresolvedPurpose?scope.draftKind!=="local"||!scope.draftSourceId:(pending?.draftKind??"official")!==(scope.draftKind??"official");
 const foreign=!!pending && (pending.userId!==scope.userId||pending.organizationId!==scope.organizationId||purposeMismatch||!!scope.draftSourceId&&pendingSource!==scope.draftSourceId);
 useEffect(()=>{alive.current=true;return()=>{alive.current=false;abort.current?.abort()}},[]);
 useEffect(()=>context.registerOrganizationSwitchGuard(()=>!active.current),[context]);
 async function dispatch(intent:SupplyIntent,verify:boolean){
  if(foreign||!alive.current||active.current||intent.userId!==scope.userId||intent.organizationId!==scope.organizationId)return;
  if(requiresDraftPurpose(intent)&&!verify)return;
  if(!context.setPendingSupplyIntent(intent)){setError("BROWSER_STORAGE_UNAVAILABLE");return}
  setBusy(true);setError(null);active.current=true;
  const controller=new AbortController();abort.current=controller;
  try{
   // Keep the stored envelope/key/input intact. This request-only purpose is
   // selected by the user; an unresolved command is never automatically sent.
   const readIntent=requiresDraftPurpose(intent)?{...intent,draftKind:scope.draftKind}:intent;
   const result=await(verify?readSupplyCommand(readIntent,controller.signal):supplyCommand(intent,controller.signal));
   if(!alive.current)return;
   if(!context.setPendingSupplyIntent(null)){setError("BROWSER_STORAGE_UNAVAILABLE");return}
   saved.current(result,intent);
   return result;
  }catch(failure){
   if(!alive.current)return;
   const code=failure instanceof SupplyAPIError?failure.code:"OUTCOME_UNKNOWN";
   if(!verify && failure instanceof SupplyAPIError && failure.status>=400 && failure.status<500 && code!=="OUTCOME_UNKNOWN"){
    context.setPendingSupplyIntent(null);
   }
   setError(code);
  }finally{active.current=false;if(abort.current===controller)abort.current=null;if(alive.current)setBusy(false)}
 }
 function execute(route:SupplyIntent["route"],command:unknown,preparedKey?:string,lookupOnly?:true){
  if(pending||active.current||!context.supplyIntentReady)return;
  const intent=parseSupplyIntent(JSON.stringify({...scope,key:preparedKey??crypto.randomUUID(),route,command,lookupOnly}));
  if(!intent){setError("INVALID_REQUEST");return}return dispatch(intent,false);
 }
 return {execute,pending,busy,error,foreign,ready:context.supplyIntentReady,verify:()=>{if(pending)void dispatch(pending,true)},retry:()=>{if(pending&&!pending.lookupOnly&&!requiresDraftPurpose(pending))void dispatch(pending,false)}};
}
export type ListingCommandState=ReturnType<typeof useListingCommand>;
