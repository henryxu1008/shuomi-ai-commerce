"use client";

import {useEffect,useRef,useState} from "react";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {Select} from "@/components/ui/select";
import {researchRequest} from "@/lib/api/market-research";
import type {ProjectScope} from "@/lib/api/project-center";
import {researchReceipt,supplierReviewPage,type ResearchCandidate,type ResearchCommand} from "@/lib/contracts/market-research";
import {assessmentPage,assessmentView,assessmentPreview,type AssessmentRef,type AssessmentView,type AssessmentSupplierRef} from "@/lib/contracts/research-assessment";
import {costFields,type CostDraft} from "@/lib/workbench/research-cost-preview";
import {ResearchCostPreview,PreviewResult,costFieldLabels} from "./research-cost-preview";

export function ResearchAssessments({scope,projectId,candidate,canWrite,disabled,onSave,onChoose,selectedRef,expanded,onExpandedChange}:{
 scope:ProjectScope;projectId:string;candidate:ResearchCandidate;canWrite:boolean;disabled:boolean;
 onSave:(command:ResearchCommand)=>Promise<z.infer<typeof researchReceipt>|undefined>;
 onChoose:(ref?:AssessmentRef)=>void;
 selectedRef?:AssessmentRef;
 expanded?:boolean;onExpandedChange?:(open:boolean)=>void;
}){
 const [page,setPage]=useState<z.infer<typeof assessmentPage>>({items:[]}),[before,setBefore]=useState<number>(),[refresh,setRefresh]=useState(0);
 const [suppliers,setSuppliers]=useState<z.infer<typeof supplierReviewPage>>({items:[]}),[supplierBefore,setSupplierBefore]=useState<number>();
 const [supplier,setSupplier]=useState<AssessmentSupplierRef>(),[localOpened,setLocalOpened]=useState(false),[dirty,setDirty]=useState(false),[draftKey,setDraftKey]=useState(0);
 const opened=expanded??localOpened;
 function setOpened(value:boolean){setLocalOpened(value);onExpandedChange?.(value);if(!value)setDirty(false)}
 const [selected,setSelected]=useState<AssessmentView>(),[discard,setDiscard]=useState<string>(),[message,setMessage]=useState(""),[busy,setBusy]=useState(false);
 const flight=useRef<AbortController|null>(null);
 const checkedRevision=useRef<number|undefined>(undefined);
 const contextKey=JSON.stringify([scope,projectId,candidate.id,candidate.selectedLinkId,candidate.evidenceDigest]);
 const referenceKey=selectedRef?JSON.stringify([selectedRef.assessmentId,selectedRef.revision,selectedRef.contextHash,selectedRef.inputHash]):"";
 function sameRef(a:AssessmentRef,b:AssessmentRef){return a.assessmentId===b.assessmentId&&a.revision===b.revision&&a.contextHash===b.contextHash&&a.inputHash===b.inputHash}
 useEffect(()=>()=>flight.current?.abort(),[contextKey]);
 useEffect(()=>{
  if(!selectedRef)return;
  if(selected?.detail&&sameRef(selected.detail.ref,selectedRef)&&checkedRevision.current===candidate.revision)return;
  const controller=new AbortController();
  void researchRequest(scope,projectId,assessmentView,controller.signal,{assessment:{candidate:candidate.id,id:selectedRef.assessmentId}}).then(exact=>{
   if(controller.signal.aborted)return;
   if(!exact.detail||exact.applicability!=="CURRENT"||!sameRef(exact.detail.ref,selectedRef)){setSelected(undefined);onChoose(undefined);setMessage("原依据暂不可读或已不适用于当前选择，未将它用于决定。");return}
   checkedRevision.current=candidate.revision;setSelected(exact);setSupplier(exact.detail.supplierReview);onChoose(exact.detail.ref);
  }).catch(()=>{if(!controller.signal.aborted){setSelected(undefined);onChoose(undefined);setMessage("暂时无法核对选定的经营评估，未将它用于决定。");}});
  return()=>controller.abort();
  // Revalidate the exact explicit choice after a technical revision; never choose latest.
  // eslint-disable-next-line react-hooks/exhaustive-deps
 },[contextKey,candidate.revision,referenceKey]);
 useEffect(()=>{
  const controller=new AbortController();
  void researchRequest(scope,projectId,assessmentPage,controller.signal,{assessment:{candidate:candidate.id,before}}).then(result=>{if(!controller.signal.aborted){setPage(result);setMessage("");}}).catch(()=>{if(!controller.signal.aborted){setPage({items:[]});setSelected(undefined);onChoose(undefined);setMessage("暂时无法读取经营评估，请重新读取。");}});
  return()=>controller.abort();
  // onChoose belongs to the parent command owner; changing its closure is not a read trigger.
  // eslint-disable-next-line react-hooks/exhaustive-deps
 },[scope,projectId,candidate.id,candidate.revision,before,refresh]);
 useEffect(()=>{
  if(!candidate.supplierReview||!opened)return;
  const controller=new AbortController();
  void researchRequest(scope,projectId,supplierReviewPage,controller.signal,{supplier:{candidate:candidate.id,before:supplierBefore}}).then(value=>{if(!controller.signal.aborted)setSuppliers(value)}).catch(()=>{if(!controller.signal.aborted){setSuppliers({items:[]});setMessage("货源核对记录暂不可读，不能使用其确认状态。");}});
  return()=>controller.abort();
 },[scope,projectId,candidate.id,candidate.revision,candidate.supplierReview,opened,supplierBefore]);
 function edited(){setDirty(true);setSelected(undefined);onChoose(undefined);setDiscard(undefined)}
 function command(cost:CostDraft):Extract<ResearchCommand,{action:"save_assessment"}>{
  return {action:"save_assessment",candidateId:candidate.id,expectedRevision:candidate.revision,linkId:candidate.selectedLinkId!,evidenceDigest:candidate.evidenceDigest!,assessment:{cost,...(supplier?{supplierReview:supplier}:{})}};
 }
 async function choose(id:string,discardEdits=false){
  if(busy)return;
  if(dirty&&!discardEdits){setDiscard(id);return}
  const controller=new AbortController();flight.current?.abort();flight.current=controller;setBusy(true);setMessage("");setSelected(undefined);onChoose(undefined);
  try{
   const exact=await researchRequest(scope,projectId,assessmentView,controller.signal,{assessment:{candidate:candidate.id,id}});
   if(controller.signal.aborted)return;
   if(!exact.detail){setMessage("原依据暂不可读，已遮蔽条件和结果。");return}
   checkedRevision.current=candidate.revision;setSelected(exact);setSupplier(exact.detail.supplierReview);setDirty(false);setDiscard(undefined);setDraftKey(v=>v+1);setOpened(false);
   if(exact.applicability==="CURRENT"&&canWrite&&!disabled)onChoose(exact.detail.ref);
  }catch{if(!controller.signal.aborted)setMessage("暂时无法核对这份评估，未将它用于决定。")}
  finally{if(flight.current===controller){flight.current=null;setBusy(false)}}
 }
 const writable=canWrite&&!!candidate.observation&&!!candidate.selectedLinkId&&!!candidate.evidenceDigest;
 return <section aria-label="已保存经营评估" className="grid min-w-0 gap-3 border-t border-border pt-4">
  <h5 className="font-medium">经营评估</h5>
  {writable&&<div><Button type="button" variant="secondary" disabled={disabled||busy} aria-expanded={opened} onClick={()=>setOpened(!opened)}>{opened?"收起参考试算":"填写经营假设"}</Button></div>}
  {opened&&writable&&<>
   {candidate.supplierReview&&<div className="grid gap-2"><label>本次明确引用的货源核对
    <Select disabled={disabled||busy} value={supplier?.requestKey??""} onChange={event=>{const row=suppliers.items.find(v=>v.reference?.requestKey===event.target.value);setSupplier(row?.reference);edited();}}>
     <option value="">不引用货源核对（UNKNOWN 手工假设）</option>
     {supplier&&!suppliers.items.some(v=>v.reference?.requestKey===supplier.requestKey)&&<option value={supplier.requestKey}>此评估明确绑定的核对 · {supplier.requestKey}</option>}
     {suppliers.items.filter(v=>v.detail&&v.reference).map(v=><option key={v.reference!.requestKey} value={v.reference!.requestKey}>{v.detail!.skuLabel} · {v.detail!.quantity} 件 · {v.detail!.status} · r{v.reference!.candidateRevision}</option>)}
    </Select></label><p>引用时采购数量必须与原记录一致；规格和费用仍需人工填写，不继承库存或有效报价。</p>
    {suppliers.nextBefore&&<Button type="button" variant="outline" onClick={()=>setSupplierBefore(suppliers.nextBefore)}>更早的货源核对</Button>}
    {supplierBefore&&<Button type="button" variant="outline" onClick={()=>setSupplierBefore(undefined)}>返回最近货源核对</Button>}
   </div>}
   <ResearchCostPreview key={draftKey} disabled={disabled||busy} initialDraft={selected?.detail?.input} onEdit={edited}
    calculate={async(cost,signal)=>(await researchRequest(scope,projectId,assessmentPreview,signal,{assessment:{candidate:candidate.id,preview:command(cost)}})).result}
    onSave={async cost=>{setBusy(true);try{const receipt=await onSave(command(cost));if(receipt){setMessage("已保存评估："+(receipt.assessmentRef?.assessmentId??receipt.key));setDirty(false);setOpened(false);setBefore(undefined);setRefresh(v=>v+1)}}finally{setBusy(false)}}}/>
  </>}
  {message&&<p role="status">{message}</p>}
  {discard&&<div role="alert"><p>有未保存的编辑。另存后再选择，或明确放弃编辑。</p><Button type="button" disabled={disabled||busy} onClick={()=>void choose(discard,true)}>放弃编辑并使用原记录</Button><Button type="button" variant="outline" onClick={()=>setDiscard(undefined)}>继续编辑</Button></div>}
  {selected?.detail&&<section aria-label="选定的经营评估" className="grid gap-2">
   <p>{selected.applicability==="CURRENT"?"使用此记录中的条件":"历史条件，不适用于当前选择"} · {selected.id}</p>
   <p>{selected.detail.input.channel} · {selected.detail.input.fulfillment} · {selected.detail.input.specification} · 采购意向 {selected.detail.input.quantity||"待补"} 件</p>
   <p>原商品 v{selected.detail.market.version} · 货源绑定 {selected.detail.supplierBinding} · {selected.detail.ruleVersion}</p>
   {selected.detail.supplierReview&&<p>原核对 SKU：{selected.detail.supplierSkuId} · {selected.detail.supplierQuantity} 件 · 原货源 v{selected.detail.supplierVersion}。规格说明和费用属于手工假设，不修改这份核对。</p>}
   <details><summary>记录中的完整条件</summary><dl>
    <dt>假设售价 / 退款概率</dt><dd>{selected.detail.input.price||"未知"} USD / {selected.detail.input.refundRate||"未知"}%</dd>
    <dt>销售依据</dt><dd>{selected.detail.input.salesBasis.source||"待补"} · {selected.detail.input.salesBasis.asOf||"待补"}</dd>
    <dt>采购币种 / 人工汇率</dt><dd>{selected.detail.input.purchaseCurrency} / {selected.detail.input.exchangeRate||"未使用"} · {selected.detail.input.exchangeBasis.source} · {selected.detail.input.exchangeBasis.asOf}</dd>
    {(["completed","refunded"] as const).map(name=><div key={name}><dt>{name==="completed"?"正常成交":"全额退款"} · {selected.detail!.input[name].basis.source} · {selected.detail!.input[name].basis.asOf}</dt>{costFields.map(field=>{const value=selected.detail!.input[name].costs[field];return <dd key={field}>{costFieldLabels[field]}：{value.kind==="value"?value.value||"未知":"不适用 · "+value.reason}{value.kind==="value"&&value.basis?" · "+value.basis.source+" · "+value.basis.asOf:""}</dd>})}</div>)}
   </dl></details>
   <PreviewResult result={selected.detail.result}/>
   {canWrite&&<Button type="button" variant="outline" onClick={()=>{setSelected(undefined);onChoose(undefined)}}>本次不引用经营评估</Button>}
  </section>}
  <h6>已保存记录</h6>
  {!page.items.length&&<p>本页暂无经营评估。未引用评估仍可沿原入口作人工决定。</p>}
  {page.items.map(row=><div key={row.id} className="grid gap-1 rounded border border-border p-2">
   <span>{row.id}</span><small>{row.at} · r{row.candidateRevision} · {row.detail?.result.status??"原依据暂不可读，条件和结果已遮蔽"}</small>
   {row.detail&&<Button type="button" variant="outline" disabled={busy} onClick={()=>void choose(row.id)}>查看并使用此记录中的条件</Button>}
  </div>)}
  <div className="flex gap-2">{page.nextBefore&&<Button variant="outline" onClick={()=>setBefore(page.nextBefore)}>更早的评估</Button>}{before&&<Button variant="outline" onClick={()=>setBefore(undefined)}>返回最近评估</Button>}<Button variant="ghost" onClick={()=>{setSelected(undefined);onChoose(undefined);setRefresh(v=>v+1)}}>重新读取评估</Button></div>
 </section>
}
