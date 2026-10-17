"use client";
import {useEffect,useRef,useState} from "react";
import {useQueryClient} from "@tanstack/react-query";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {useResourcePending} from "@/components/workbench/resources/resource-pending";
import {MarketAPIError,resolveMarket,writeMarket,type MarketScope,type MarketIntent} from "@/lib/api/supply-market";
import type {MarketReceipt} from "@/lib/contracts/supply-market";
import {freezeBatch,isSelectedBatch,marketPendingLimit,marketPendingSchema,selectedReceipt,type SelectedBatch,type SelectedCommand} from "./selected-batch-intent";

export type ApplicationResult={title:string;key:string;intent:MarketIntent;state:"waiting"|"processing"|"submitted"|"failed"|"unknown";receipt?:MarketReceipt;error?:unknown};
const definiteFailure=(e:unknown)=>e instanceof MarketAPIError&&((e.status===400&&e.code==="INVALID_REQUEST")||(e.status===409&&e.code==="REVISION_CONFLICT")||(e.status===404&&e.code==="NOT_FOUND"));
const missing=(e:unknown)=>e instanceof MarketAPIError&&e.code==="NOT_FOUND"&&e.status===404;

export function useSelectedBatch(scope:MarketScope){
  const context=useWorkbenchContext(),client=useQueryClient(),live=useRef(context),active=useRef(false),alive=useRef(true),controller=useRef<AbortController|null>(null);
  useEffect(()=>{live.current=context},[context]);
  useEffect(()=>{alive.current=true;return()=>{alive.current=false;controller.current?.abort()}},[]);
  const pending=useResourcePending({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId},["supply-market","member"],marketPendingSchema,{storage:"local",maxLength:marketPendingLimit});
  const [busy,setBusy]=useState(false),[error,setError]=useState<unknown>(null),[results,setResults]=useState<ApplicationResult[]>([]);
  // Memory only: owner receipts already observed by this page remain evidence
  // of success. A subsequent missing read must never turn them into new POSTs.
  const observed=useRef(new Map<string,{payload:string;receipt:MarketReceipt}>());
  const register=context.registerOrganizationSwitchGuard;
  useEffect(()=>register(()=>!active.current&&!pending.command&&!pending.error),[register,pending.command,pending.error]);
  const journal=pending.command&&isSelectedBatch(pending.command)?pending.command:null;
  function current(){const c=live.current;if(c.user?.id!==scope.userId)throw new MarketAPIError("IDENTITY_CONTEXT_CHANGED",409);if(c.isLoading||c.isSwitching||c.error||c.blockingError||c.effectiveOrganization?.id!==scope.organizationId)throw new MarketAPIError("ORGANIZATION_CONTEXT_CHANGED",409)}
  async function run(mode:"start"|"verify"|"continue",batch:SelectedBatch){
    if(active.current||!pending.ready||pending.error)return;
    if(!navigator.locks){setError(new MarketAPIError("DEPENDENCY_UNAVAILABLE",503));return}
    try{await navigator.locks.request(pending.storageKey,{ifAvailable:true},async lock=>{
      if(!lock)throw new MarketAPIError("OUTCOME_UNKNOWN",409);
      current();const original=pending.read();
      if(mode==="start"){if(original)throw new MarketAPIError("OUTCOME_UNKNOWN",409);pending.persist(batch);observed.current.clear()}
      else if(!original||JSON.stringify(original)!==JSON.stringify(batch))throw new MarketAPIError("OUTCOME_UNKNOWN",409);
      const abort=new AbortController();controller.current=abort;active.current=true;setBusy(true);setError(null);
      const rows:ApplicationResult[]=batch.items.map(item=>({title:item.title,key:item.intent.key,intent:item.intent,state:"waiting",receipt:observed.current.get(item.intent.key)?.receipt}));
      const update=()=>{if(alive.current)setResults([...rows])};
      const check=()=>{abort.signal.throwIfAborted();current();if(JSON.stringify(pending.read())!==JSON.stringify(batch))throw new MarketAPIError("OUTCOME_UNKNOWN",409)};
      update();
      try{
        for(let index=0;index<batch.items.length;index++){
          const item=batch.items[index]!;
          try{
            check();rows[index]={...rows[index]!,state:"processing"};update();let receipt:MarketReceipt|undefined;
            const known=observed.current.get(item.intent.key);
            if(known&&known.payload!==JSON.stringify(item.intent))throw new MarketAPIError("OUTCOME_UNKNOWN",409);
            if(mode!=="start"){
              try{receipt=selectedReceipt(await resolveMarket(scope,item.intent,abort.signal));check();if(known&&(receipt.operationId!==known.receipt.operationId||receipt.recordId!==known.receipt.recordId||receipt.revision!==known.receipt.revision))throw new MarketAPIError("OUTCOME_UNKNOWN",502)}
              catch(e){check();if(!missing(e))throw e;if(known)throw new MarketAPIError("OUTCOME_UNKNOWN",502);if(mode==="verify"){rows[index]={...rows[index]!,state:"unknown",error:e};update();continue}}
            }
            if(!receipt){
              check();
              try{receipt=selectedReceipt(await writeMarket(scope,item.intent,abort.signal));check()}
              catch(e){check();if(definiteFailure(e)){rows[index]={...rows[index]!,state:"failed",error:e};update();continue}throw e}
            }
            if(rows.some((row,i)=>i!==index&&row.receipt&&(row.receipt.recordId===receipt.recordId||row.receipt.operationId===receipt.operationId)))throw new MarketAPIError("OUTCOME_UNKNOWN",502);
            observed.current.set(item.intent.key,{payload:JSON.stringify(item.intent),receipt});
            rows[index]={...rows[index]!,state:"submitted",receipt};update();
          }catch(e){rows[index]={...rows[index]!,state:"unknown",error:e};update();throw e}
        }
        check();if(rows.every(row=>row.state==="submitted"||row.state==="failed"))pending.clear(batch);
      }finally{
        active.current=false;if(alive.current)setBusy(false);if(controller.current===abort)controller.current=null;
        void client.invalidateQueries({queryKey:["supply-market",scope.userId,scope.organizationId]});
      }
    })}catch(e){if(alive.current)setError(e)}
  }
  function start(items:{title:string;command:SelectedCommand}[]){
    if(pending.command||active.current)return Promise.resolve();
    try{return run("start",freezeBatch(items))}catch(e){setError(e);return Promise.resolve()}
  }
  return {start,verify:()=>journal?run("verify",journal):Promise.resolve(),resume:()=>journal?run("continue",journal):Promise.resolve(),reset:()=>{if(!pending.command&&!active.current){observed.current.clear();setResults([]);setError(null)}},journal,results,busy,error,pendingError:pending.error,locked:!pending.ready||!!pending.command||pending.error||busy};
}
