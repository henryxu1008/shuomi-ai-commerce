"use client";

import Link from "next/link";
import {useEffect,useMemo,useRef,useState} from "react";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {readCollectionBatch,readCollectionItem,type CollectionScope} from "@/lib/api/product-collection";
import {verifyResearchHandoff} from "@/lib/api/research-handoff";
import type {CollectionBatch} from "@/lib/contracts/product-collection";
import {selectionOf} from "@/lib/contracts/market-research";
import {transferReceiptSchema} from "@/lib/contracts/supply-chain";
import {claimResearchTransfer,handoffToken,researchHandoffAllowed,type ResearchTransferInput} from "@/lib/workbench/research-handoff";
import {ListingCommandFeedback} from "../listing/command-feedback";
import {useListingCommand} from "../listing/use-listing-command";
import {researchHandoffScope,useResearchHandoff} from "../project-center/research-handoff-provider";
import {ConsolePage} from "../console/console-page";
import {TransferSelection} from "./transfer-selection";

export function ResearchCollectionTransfer({scope,token,supplyAvailable}:{
  scope:CollectionScope; token:string; supplyAvailable:boolean;
}) {
  const context=useWorkbenchContext();
  const {store,draft}=useResearchHandoff();
  const scopeKey=researchHandoffScope(context);
  const active=draft?.scope===scopeKey && handoffToken(draft)===token && draft.confirmed && draft.transfer?draft:null;
  const allowed=supplyAvailable && !!scopeKey && researchHandoffAllowed(context.permissions,true);
  const [batch,setBatch]=useState<CollectionBatch|null>(null);
  const [error,setError]=useState("");
  const [receipt,setReceipt]=useState<{id:string;count:number;scope:string;lookup:boolean}|null>(null);
  const operation=useRef<{key:string;scope:string;lookup:boolean}|null>(null);
  const flight=useRef<AbortController|null>(null);
  useEffect(()=>()=>flight.current?.abort(),[token,scopeKey]);
  const supply=useListingCommand(scope,(value,intent)=>{
    const original=operation.current;
    if (!allowed || !original || original.key!==intent.key || original.scope!==scopeKey ||
        intent.route!=="transfer" || !intent.lookupOnly ||
        intent.userId!==scope.userId || intent.organizationId!==scope.organizationId) return;
    const {preparation}=transferReceiptSchema.parse(value);
    setReceipt({id:preparation.id,count:preparation.count,scope:original.scope,lookup:original.lookup});
  });
  const original=useMemo(()=>{
    const choice=active?.confirmed?.choices.find(candidate=>candidate.candidateId===active.confirmed?.representative);
    return choice?[{
      itemId:choice.selection.itemId,revision:Number(choice.selection.expectedRevision),
      publicationId:choice.selection.originalPublicationId,version:choice.selection.originalVersion,
    }]:undefined;
  },[active]);
  useEffect(()=>{
    if (!active?.job.batchId || active.transfer?.dispatched || !allowed || !store) return;
    const controller=new AbortController();
    const signal=AbortSignal.any([controller.signal,AbortSignal.timeout(10_000)]);
    readCollectionBatch(scope,active.job.batchId,signal).then(current=>{
      if (controller.signal.aborted || store.getSnapshot()!==active) return;
      if (current.id!==active.job.batchId || current.archivedAt || !current.supplyTransferSupported ||
          current.revision!==active.transfer?.revision) throw new Error("Collection changed");
      setBatch(current);
    }).catch(()=>{
      if (!controller.signal.aborted && store.getSnapshot()===active)
        store.start(active.scope,active.projectId,active.job);
    });
    return ()=>controller.abort();
  },[active,allowed,scope,store]);

  const blocked=!allowed||!active||!!active.transfer?.dispatched||!supply.ready||supply.busy||!!supply.pending||
    !context.collectionIntentReady||!!context.pendingCollectionIntent;
  async function submit(input:ResearchTransferInput) {
    if (blocked || !store || !active || !original || flight.current || store.getSnapshot()!==active) return;
    const controller=new AbortController();
    flight.current=controller;
    setError("");
    try {
      const signal=AbortSignal.any([controller.signal,AbortSignal.timeout(10_000)]);
      const [current,detail]=await Promise.all([
        readCollectionBatch(scope,input.batchId,signal),
        readCollectionItem(scope,original[0].itemId,signal),
      ]);
      signal.throwIfAborted();
      if (current.id!==active.job.batchId || current.archivedAt || !current.supplyTransferSupported ||
          current.revision!==input.expectedRevision || detail.item.archivedAt || detail.item.batchId!==current.id ||
          JSON.stringify(selectionOf(detail.item))!==JSON.stringify(original[0])) throw new Error("Collection changed");
      await verifyResearchHandoff(scope,active,signal,true);
      signal.throwIfAborted();
      const key=claimResearchTransfer(store,active,input);
      if (key) {
        operation.current={key,scope:scopeKey,lookup:false};
        void supply.execute("transfer",input,key,true);
      }
    } catch {
      if (!controller.signal.aborted && store.getSnapshot()===active) {
        store.start(active.scope,active.projectId,active.job);
        setError("项目、批准或原版资料已变化，本次未提交。");
      }
    } finally {
      if (flight.current===controller) flight.current=null;
    }
  }
  function verify() {
    if (!allowed || !supply.pending || supply.foreign || supply.busy) return;
    operation.current={key:supply.pending.key,scope:scopeKey,lookup:true};
    supply.verify();
  }
  const visibleReceipt=allowed && receipt?.scope===scopeKey && (active || receipt.lookup)?receipt:null;
  return <ConsolePage title="核对研究商品" description="确认将本次批准的一件商品转入待上传。"
    breadcrumbs={[{label:"我的数据",href:"/workbench/data/mine"},{label:"核对研究商品"}]}>
    {allowed&&!supply.foreign?<ListingCommandFeedback state={{...supply,verify}}/>:null}
    {visibleReceipt?<p role="status">已转入 {visibleReceipt.count} 件。<Link href={"/workbench/listing/waiting?preparation="+visibleReceipt.id}>查看待上传商品</Link></p>:
      !active || !allowed?<p role="alert">{error||"研究交接已失效。"} <Link prefetch={false} href={draft?.scope===scopeKey?
        "/workbench/data/research/"+draft.projectId:"/workbench/data/research"}>回市场调研重新确认</Link></p>:
      active.transfer?.dispatched?<p role="status">本次确认已使用。请按原操作核实结果。</p>:
      batch && original?<TransferSelection key={token} scope={scope} batch={batch} initialSelections={original}
        blocked={blocked} fixedResearch submit={submit}/>:<p role="status">正在读取本次转入批次…</p>}
  </ConsolePage>;
}
