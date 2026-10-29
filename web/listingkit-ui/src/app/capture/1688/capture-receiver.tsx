"use client";
import { useEffect,useLayoutEffect,useRef,useState } from "react";
import { Button } from "@/components/ui/button";
import { capture1688,readBrowserCapture,readBrowserCaptureByKey } from "@/lib/api/browser-capture";
import { WorkbenchContextError } from "@/lib/api/workbench-context";
import { browserCaptureSchema } from "@/lib/contracts/browser-capture";
import type { AcquisitionResult } from "@/lib/contracts/product-acquisition";
import { useCaptureDraft } from "../use-capture-draft";
import { captureHandoffExpiredMessage,isCaptureHandoffExpired } from "../capture-draft";

type View={attempted:boolean;busy:boolean;scopeKey?:string;result?:AcquisitionResult;message:string;refusal?:string;handoffExpired?:boolean};
const unknown="Outcome is unknown. Local capture data is retained. Check the original operation.";
export function CaptureReceiver(){
 const capture=useCaptureDraft("1688",browserCaptureSchema),{context,scope,ready,entry,draft,client}=capture;
 const [view,setView]=useState<View>({attempted:false,busy:false,message:"Confirm the current account and enterprise before submitting."});
 const active=useRef<AbortController|null>(null),busy=useRef(false),latest=useRef(capture);useLayoutEffect(()=>{latest.current=capture;});
 const scopeKey=JSON.stringify(scope),started=view.attempted||entry?.kind==="recovery"||!!draft&&!draft.canSubmit;
 const handoffExpired=view.scopeKey===scopeKey&&view.handoffExpired;
 const stillCurrent=()=>latest.current.stillCurrent()&&JSON.stringify(latest.current.scope)===scopeKey;
 useEffect(()=>{if(!ready)active.current?.abort();return()=>active.current?.abort();},[ready,scopeKey]);
 async function run(submit:boolean){
  if(!entry||!ready||view.scopeKey&&view.scopeKey!==scopeKey||busy.current||submit&&(started||!draft||!client||handoffExpired))return;
  busy.current=true;const controller=new AbortController();active.current=controller;
  const original=view.result?.operationId;
  setView(v=>({...v,busy:true,result:undefined,refusal:undefined,scopeKey}));
  let sendIntent=false,dispatched=false;
  const consent={userId:scope.userId,organizationId:scope.organizationId};
  const valid=()=>stillCurrent()&&!controller.signal.aborted;
  try{
   const refreshed=await context.retry();
   if(!refreshed||refreshed.selectionRequired||refreshed.user.id!==scope.userId||refreshed.effectiveOrganizationId!==scope.organizationId||refreshed.effectiveMemberId!==scope.memberId||!valid())throw Error("IDENTITY_CONTEXT_CHANGED");
   let body:string|undefined;
   if(submit){
    body=draft!.intent?.body??JSON.stringify(draft!.payload);
    await client!.bind(body);if(!valid())throw Error("IDENTITY_CONTEXT_CHANGED");
    try{history.replaceState(null,"",`/capture/1688#operationKey=${entry.key}`);}catch{throw Error("OPERATION_KEY_UNRETAINED");}
    setView(v=>({...v,attempted:true,message:"Local data is protected. Submitting the frozen capture..."}));
    sendIntent=true;await client!.send();
   }
   if(!valid())throw Error("IDENTITY_CONTEXT_CHANGED");
   dispatched=true;
   const result=submit?await capture1688(Object.freeze({...consent,key:entry.key,body:body!}),controller.signal):original?await readBrowserCapture(consent,original,controller.signal):await readBrowserCaptureByKey(consent,entry.key,controller.signal);
   if(!valid())return;
   if(client)await client.report(result.outcome==="published"?"saved_reported":result.outcome==="failed"?"rejected":"unknown",result.operationId).catch(()=>{});
   setView(v=>({...v,result,scopeKey,message:result.outcome==="published"?"Published version "+result.catalogVersion:result.outcome==="failed"?"The backend recorded this operation as failed.":unknown}));
  }catch(failure){
   if(!stillCurrent())return;
   if(!dispatched&&isCaptureHandoffExpired(failure)){
    setView(v=>({...v,handoffExpired:true,refusal:"NOT_DISPATCHED",message:captureHandoffExpiredMessage}));return;
   }
   const code=failure instanceof WorkbenchContextError?failure.code:failure instanceof Error&&/^(DRAFT_|IDENTITY_|OPERATION_KEY_)/.test(failure.message)?failure.message:"OUTCOME_UNKNOWN";
   const preAdmission=["ACQUISITION_CAPACITY","DEADLINE_EXCEEDED","INVALID_ACQUISITION","SOURCE_TOO_LARGE"].includes(code);
   if(client&&sendIntent)await client.report(preAdmission?"rejected":"unknown",undefined,preAdmission?code:undefined).catch(()=>{});
   const retryable=preAdmission&&["ACQUISITION_CAPACITY","DEADLINE_EXCEEDED"].includes(code);
   const message=code==="ACQUISITION_CAPACITY"?"Capture was not admitted because active capacity is full. Local data is retained.":code==="DEADLINE_EXCEEDED"?"Capture was not submitted before the deadline. Local data is retained.":preAdmission?"Capture was rejected before admission: "+code+". Local data is retained.":code==="ACQUISITION_NOT_FOUND"?"No operation is visible for this key. This does not prove a prior request failed. No new submission will be made.":!dispatched?"Capture was not submitted: "+code+". Restore the original context or recover the local draft.":unknown;
   setView(v=>({...v,attempted:retryable?false:v.attempted,refusal:preAdmission?code:!dispatched?"NOT_DISPATCHED":undefined,message}));
  }finally{busy.current=false;if(active.current===controller)active.current=null;setView(v=>({...v,busy:false}));}
 }
 const payload=ready?draft?.payload:undefined;
 const receipt=ready&&view.scopeKey===scopeKey?view.result:undefined;
 const terminal=receipt&&["published","failed","outcome_unknown"].includes(receipt.outcome)?receipt:undefined;
 return <section className="mx-auto w-full max-w-3xl space-y-6 py-6">
  <header><p className="text-sm text-muted-foreground">1688 / Browser capture</p><h1 className="mt-2 text-2xl font-semibold">Confirm a browser capture</h1><p className="mt-2 text-sm text-muted-foreground">Protected data stays in the original extension’s local draft. Reloading never submits again.</p></header>
  <div className="grid gap-4 rounded-xl border border-border bg-card p-5 sm:grid-cols-2"><div><p className="text-xs text-muted-foreground">Current verified account</p><p className="break-all" data-batch-scope-actor={ready?"":undefined}>{ready?scope.userId:"Unavailable"}</p></div><div><p className="text-xs text-muted-foreground">Effective enterprise</p><p>{ready?context.effectiveOrganization?.name:"Unavailable"}</p><p className="break-all text-xs text-muted-foreground" data-batch-scope-organization={ready?"":undefined}>{ready?scope.organizationId:null}</p></div></div>
  {payload&&!started?<section className="space-y-3 rounded-xl border border-border bg-card p-5"><h2 className="break-words text-lg font-medium">{payload.evidence.title??"Title not observed"}</h2><p className="break-all text-sm text-muted-foreground">{payload.evidence.sourceURL}</p><p className="text-sm">{payload.evidence.warnings.length} warnings; {payload.evidence.missingFacts.length} missing facts</p><ul className="space-y-1 text-sm text-muted-foreground">{payload.evidence.missingFacts.map((f,i)=><li key={i}>{f.field}: {f.reason}</li>)}</ul></section>:null}
  <p role="status" className="break-words rounded-xl border border-border bg-muted/40 p-4 text-sm">{!ready?"Context changed or access is unavailable. Return to the original account, enterprise and member.":capture.error??view.message}</p>
  <div hidden data-batch-submit-result={terminal?terminal.outcome:""} data-batch-operation-id={terminal?.operationId}></div>
  <div hidden data-batch-submit-refusal>{view.refusal??(capture.error?"HANDOFF_UNREADABLE":"")}</div>
  {receipt?.outcome==="published"?<dl className="space-y-2 rounded-xl border border-border bg-card p-5 text-sm"><dt>Product</dt><dd>{receipt.productKey}</dd><dt>Original publication</dt><dd>{receipt.publicationId}</dd><dt>Warnings / missing facts</dt><dd>{receipt.warnings.length} / {receipt.missingFacts.length}</dd></dl>:null}
  <div className="flex flex-wrap gap-3">{!started&&payload&&!handoffExpired?<Button data-batch-confirm-submit="" disabled={!ready||view.busy} onClick={()=>void run(true)}>Confirm and submit</Button>:null}{started&&entry?<Button disabled={!ready||view.busy} variant="outline" onClick={()=>void run(false)}>Check original operation</Button>:null}{view.busy?<Button variant="outline" onClick={()=>{active.current?.abort();setView(v=>({...v,message:unknown}));}}>Stop waiting</Button>:null}</div>
  <p className="text-xs text-muted-foreground">Use the original extension to restore after expiry. Do not share this page’s private address. Removing extension storage deletes local drafts.</p>
 </section>;
}
