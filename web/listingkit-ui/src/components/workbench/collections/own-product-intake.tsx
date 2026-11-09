"use client";

import {useEffect,useRef,useState} from "react";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Checkbox} from "@/components/ui/checkbox";
import {Select} from "@/components/ui/select";
import {Textarea} from "@/components/ui/textarea";
import {podRequest} from "@/lib/api/pod";
import {MarketAPIError} from "@/lib/api/supply-market";
import {intakePrepared,intakeResult,intakeValidationMessage,type IntakeBody,type IntakeCandidate,type IntakePrepared} from "@/lib/contracts/own-product-intake";
import {collectionID} from "@/lib/contracts/product-collection";
import type {CollectionScope} from "@/lib/api/product-collection";

const namespace="own-product-intake";
type Prepared={key:string;body:IntakeBody;signature:string;prepared:IntakePrepared};
type Receipt={key:string;digest:string};
type Pending={signature:string;notes:string;candidate:IntakeCandidate;confirmedPrices:ReadonlySet<number>};

async function digest(text:string){
 const hash=await crypto.subtle.digest("SHA-256",new TextEncoder().encode(text));
 return Array.from(new Uint8Array(hash),v=>v.toString(16).padStart(2,"0")).join("");
}

function originalReceipt(key:string):Receipt|null{
 try{
  const result=JSON.parse(sessionStorage.getItem(key)??"null");
  return collectionID.safeParse(result?.key).success&&/^[a-f0-9]{64}$/.test(result?.digest)?result:null;
 }catch{return null}
}

export function OwnProductIntake({scope,member,images,uploaded,disabled,onAdopt}:{scope:CollectionScope;member:string;images:string[];uploaded:ReadonlySet<string>;disabled:boolean;onAdopt:(candidate:IntakeCandidate,acceptImages:boolean,gallery:string[],confirmedPrices:ReadonlySet<number>)=>void}){
 const storageKey=`own-intake:${JSON.stringify([scope.userId,scope.organizationId,member])}`;
 const [open,setOpen]=useState(false),[notes,setNotes]=useState(""),[image,setImage]=useState("");
 const [busy,setBusy]=useState(false),[message,setMessage]=useState(""),[error,setError]=useState("");
 const [prepared,setPrepared]=useState<Prepared|null>(null),[result,setResult]=useState<Pending|null>(null);
 const [original,setOriginal]=useState<Receipt|null>(()=>originalReceipt(storageKey)),[confirmImages,setConfirmImages]=useState(false);
 const [missingRunKey,setMissingRunKey]=useState<string|null>(null);
 const current=useRef(""),flight=useRef<AbortController|null>(null);
 const signature=JSON.stringify({notes:notes.trim(),images,image});
 const [lastSignature,setLastSignature]=useState(signature);
 // Invalidate on every input/gallery change, including change-then-restore.
 // Merely hiding the candidate would let an old confirmation become active again.
 if(lastSignature!==signature){
  setLastSignature(signature);
  if(result?.confirmedPrices.size)setResult({...result,confirmedPrices:new Set()});
 }
 useEffect(()=>{current.current=signature},[signature]);
 useEffect(()=>()=>flight.current?.abort(),[]);
 const selected=images.indexOf(image)+1;
 const changed=!!prepared&&prepared.signature!==signature;
 const recovering=!!original&&original.key===missingRunKey;
 const canPrepare=!!notes.trim()&&new TextEncoder().encode(notes.trim()).byteLength<=4096&&(!image||selected>0);
 const candidate=result?.signature===signature?result.candidate:null;
 const hasImages=candidate?.rows?.some(row=>row.image!==undefined)??false;

 function storeReceipt(value:Receipt|null){
  if(value)sessionStorage.setItem(storageKey,JSON.stringify(value));
  else sessionStorage.removeItem(storageKey);
  setOriginal(value);
 }
 async function action(task:(signal:AbortSignal)=>Promise<void>){
  if(flight.current||disabled)return;
  const controller=new AbortController();
  flight.current=controller;setBusy(true);setMessage("");setError("");
  try{await task(controller.signal)}
  catch(e){
   if(!controller.signal.aborted){
    const code=e instanceof MarketAPIError?e.code:"";
    setError(code==="AGENT_NOT_ENABLED"?"当前企业尚未启用自有商品资料整理智能体。":code==="OUTCOME_UNKNOWN"?"请求结果待核实。已确认生成的请求可以查看原结果。":code==="NOT_FOUND"?"暂未查到原生成结果，请稍后再核实。":code==="FORBIDDEN"?"当前成员没有使用权限。":"整理服务暂不可用，或资料和图片未通过校验。");
   }
  }finally{flight.current=null;if(!controller.signal.aborted)setBusy(false)}
 }

 function prepare(recovery?:Receipt){
  void action(async signal=>{
   if(recovery&&await digest(signature)!==recovery.digest){
    setError("请恢复原说明和图库顺序，并选择原图片后再继续。");
    return;
   }
   const key=recovery?.key??crypto.randomUUID();
   const body:IntakeBody={notes:notes.trim(),gallery:await Promise.all(images.map(digest)),...(selected>0?{imageNumber:selected,imageUrl:image}:{})};
   if(signal.aborted||current.current!==signature)return;
   const response=await podRequest(scope,"prepare",intakePrepared,signal,{key,body},namespace);
   if(!signal.aborted&&current.current===signature){setPrepared({key,body,signature,prepared:response});setResult(null);setConfirmImages(false)}
  });
 }

 function receive(response:ReturnType<typeof intakeResult.parse>,key:string,expected:string,sourceNotes:string){
  if(response.requestKey!==key)throw new Error("请求标识已变化");
  if(response.phase==="running"||["model_outcome_unknown","execution_outcome_unknown","usage_unknown"].includes(response.stopReason??"")){
   setMessage("生成结果待核实。可用原请求键再次查看。");
   return;
  }
  storeReceipt(null);
  setMissingRunKey(null);
  setPrepared(null);
  if(!response.candidate){setError(response.validationError?intakeValidationMessage(response.validationError):"本次生成没有通过校验，请核对资料后重新整理。");return}
  setResult({signature:expected,notes:sourceNotes,candidate:response.candidate,confirmedPrices:new Set()});
  setMessage("候选已生成。核对后可采纳到表单。");
 }

 function confirm(){
  const intent=prepared;
  if(!intent||changed||original&&(!recovering||original.key!==intent.key))return;
  void action(async signal=>{
   const signatureDigest=original?.digest??await digest(intent.signature);
   if(signal.aborted||current.current!==intent.signature)return;
   if(!original)storeReceipt({key:intent.key,digest:signatureDigest});
   setMissingRunKey(null);
   const {configurationSnapshotRef,quote}=intent.prepared;
   try{
    const response=await podRequest(scope,intent.key+"/confirm",intakeResult,signal,{key:intent.key,body:{...intent.body,configurationSnapshotRef,quoteDigest:quote.InputHash}},namespace);
    if(!signal.aborted)receive(response,intent.key,intent.signature,intent.body.notes);
   }catch(e){
    if(!original&&e instanceof MarketAPIError&&e.status>=400&&e.status<500&&e.code!=="OUTCOME_UNKNOWN")storeReceipt(null);
    throw e;
   }
  });
 }

 function readOriginal(){
  if(!original)return;
  void action(async signal=>{
   setMissingRunKey(null);
   setPrepared(null);
   const matching=await digest(signature)===original.digest;
   let response:ReturnType<typeof intakeResult.parse>;
   try{
    response=await podRequest(scope,original.key,intakeResult,signal,undefined,namespace);
   }catch(e){
    if(!signal.aborted&&e instanceof MarketAPIError&&e.status===404&&e.code==="NOT_FOUND"){
     setMissingRunKey(original.key);
     setMessage("暂未查到原生成记录，可以用原资料重新检查用量。");
     return;
    }
    throw e;
   }
   if(signal.aborted)return;
   if(!matching&&response.candidate){setMessage("原结果已找到。恢复原说明和图库顺序后可采纳。");return}
   receive(response,original.key,signature,notes.trim());
  });
 }

 return <Card className="space-y-4 p-6">
  <div className="flex flex-wrap items-center justify-between gap-2">
   <div><h2 className="text-lg font-semibold">AI 整理商品资料</h2><p className="mt-1 text-sm text-muted-foreground">把说明和已上传图片整理成可核对的商品与 SKU 候选。</p></div>
   <Button type="button" variant="outline" aria-expanded={open} onClick={()=>setOpen(value=>!value)}>{open?"收起整理资料":"开始整理资料"}</Button>
  </div>
  {open?<div className="space-y-4">
   <label className="grid gap-2 text-sm">给 AI 的商品资料<Textarea rows={5} value={notes} disabled={busy} onChange={e=>setNotes(e.target.value)} placeholder="逐行写已知 SKU、规格、币种、价格和库存。可用「图片 2」指定图片。"/></label>
   <label className="grid gap-2 text-sm">让 AI 查看一张图片<Select value={selected>0?image:""} disabled={busy} onChange={e=>setImage(e.target.value)}><option value="">仅使用文字资料</option>{images.map((url,i)=>uploaded.has(url)?<option key={url} value={url}>图片 {i+1}</option>:null)}</Select></label>
   <div className="flex flex-wrap gap-2">
    <Button type="button" variant="outline" disabled={busy||disabled||!!original||!canPrepare} onClick={()=>prepare()}>检查生成用量</Button>
    {original?<Button type="button" variant="outline" disabled={busy||disabled} onClick={readOriginal}>核实原生成结果</Button>:null}
    {original&&recovering?<Button type="button" variant="outline" disabled={busy||disabled||!canPrepare} onClick={()=>prepare(original)}>按原请求重新检查用量</Button>:null}
   </div>
   {prepared&&!changed&&(!original||recovering&&prepared.key===original.key)?<div className="flex flex-wrap items-center gap-3"><p className="text-sm">最多 {prepared.prepared.quote.MaximumTokens} tokens · 模型成本上限 {prepared.prepared.quote.Currency} {(prepared.prepared.quote.MaximumCostMicros/1_000_000).toFixed(6)}</p><Button type="button" disabled={busy||disabled} onClick={confirm}>确认生成候选</Button></div>:null}
   {busy?<p role="status" className="text-sm">正在处理整理请求…</p>:null}
   {message?<p role="status" className="text-sm">{message}</p>:null}
   {error?<p role="alert" className="text-sm text-destructive">{error}</p>:null}
   {candidate?<div className="space-y-3 rounded-lg border p-4" aria-label="AI 生成候选" role="region">
    <h3 className="font-medium">候选商品资料</h3>
    {candidate.title?<p className="break-words">{candidate.title}</p>:null}
    {candidate.description?<p className="whitespace-pre-wrap break-words text-sm">{candidate.description}</p>:null}
    {candidate.brand?<p className="text-sm">品牌 · {candidate.brand.value}</p>:null}
    {Object.entries(candidate.attributes??{}).map(([key,fact])=><p key={key} className="break-words text-sm">{key} · {fact.value}</p>)}
    <ul className="space-y-2 text-sm">
     {candidate.rows?.map((row,i)=><li key={i} className="space-y-1 break-words"><strong>{Object.values(row.attributes??{}).join(" / ")||"默认规格"}</strong> · {row.sku??"未填写 SKU"} · 库存 {row.stock??"未填写"}{row.image?` · 图片 ${row.image}`:""}
      <p className="text-xs text-muted-foreground">{row.evidence}</p>
      {candidate.stockReviewRows?.includes(i)?<p className="text-sm text-muted-foreground">库存需核对，未填入。请在商品表单手工确认填写。</p>:null}
      {row.priceSuggestion?<div className="space-y-1">
       <p>待确认价格建议：{row.priceSuggestion.currency??"币种未填写"} {row.priceSuggestion.amount}</p>
       <p className="text-xs text-muted-foreground">价格来源：{row.priceSuggestion.evidence}</p>
       <label className="flex items-center gap-2"><Checkbox disabled={busy||disabled||!row.priceSuggestion.currency} checked={result?.confirmedPrices.has(i)??false} onChange={e=>{
        const checked=e.target.checked;
        setResult(prior=>{if(!prior||prior.candidate!==candidate||prior.signature!==signature)return prior;const confirmedPrices=new Set(prior.confirmedPrices);if(checked)confirmedPrices.add(i);else confirmedPrices.delete(i);return {...prior,confirmedPrices}});
       }}/>确认第 {i+1} 个 SKU 的商品单价和币种</label>
       {!row.priceSuggestion.currency?<p className="text-xs text-muted-foreground">币种未知，请在商品表单手工填写金额和币种。</p>:null}
      </div>:<p className="text-muted-foreground">未填写价格</p>}
     </li>)}
    </ul>
    {candidate.rows?.some(row=>row.priceSuggestion)?<div className="space-y-2">
     <p className="text-sm text-muted-foreground">AI 价格只是建议。请核对范围、否定、费用和规格归属；未勾选的价格不会填入，其他资料仍可采纳。已有人工金额或币种均保留，请在表单手动修改。</p>
     <details><summary className="cursor-pointer text-sm">查看本次完整原说明</summary><p className="whitespace-pre-wrap break-words text-sm">{result?.notes}</p></details>
    </div>:null}
    {candidate.questions?.map((question,i)=><p key={i} className="text-sm text-muted-foreground">{question}</p>)}
    {hasImages?<label className="flex items-center gap-2 text-sm"><Checkbox checked={confirmImages} onChange={e=>setConfirmImages(e.target.checked)}/>确认候选中的 SKU 图片归属</label>:null}
    <Button type="button" disabled={disabled||busy} onClick={()=>{try{onAdopt(candidate,confirmImages,images,result?.confirmedPrices??new Set());setResult(null);setMessage("已填入表单。请检查后保存商品。");setError("")}catch(e){setError((e as Error).message)}}}>采纳到商品表单</Button>
   </div>:null}
  </div>:null}
 </Card>;
}
