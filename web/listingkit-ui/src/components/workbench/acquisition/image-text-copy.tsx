"use client";

import {useEffect,useState} from "react";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {ProductAgentPanel,type StartIntent} from "./product-agent-panel";
import {readOwnContentSource} from "@/lib/api/product-agent";
import {contentAgentStartRequestSchema,ownContentStartRequestSchema,type OwnSourceSelection} from "@/lib/contracts/product-agent";
import type {ImageTextSelection,ImageSetRun} from "@/lib/contracts/product-image-set";
import type {ImageSetScope} from "@/lib/api/product-image-set";
import {fetchProductTitleProposal,decideProductTitleProposal,ProductTitleReviewError} from "@/lib/api/product-title-review-client";
import {productTitleDecisionSchema,type ProductTitleDecisionInput,type ProductTitleProposal,type ContentValue} from "@/lib/api/product-title-review";
import {TitleReviewControls} from "../task-center/title-review-detail";
import {ContentComparison} from "../task-center/content-review-presentation";
import {isAcquisitionUUID} from "@/lib/contracts/product-acquisition";

export const imageTextLanguageNames:Record<string,string>={en:"英语",zh:"中文",es:"西班牙语",fr:"法语",de:"德语",ja:"日语"};
type Props={scope:ImageSetScope;source:ImageSetRun["plan"]["Source"];language:string;templateKey:string;taskIds:string[];ownItemId?:string;disabled:boolean;onChange:(value:ImageTextSelection|null)=>void};
type Original={key:string;start:StartIntent};
type ReviewIntent={key:string;proposalId:string;body:ProductTitleDecisionInput};

export function confirmedImageTextLines(p:ProductTitleProposal):string[]{
 if(p.policy!=="product-content-review-v1")return [];
 const title=p.content.after.find(v=>v.field==="title")?.value;
 const points=p.content.after.find(v=>v.field==="selling_points")?.value;
 let list:unknown=[];try{if(points)list=JSON.parse(points)}catch{return []}
 return [...(title?[title]:[]),...(Array.isArray(list)&&typeof list[0]==="string"?[list[0]]:[])];
}

export function ImageTextCopy(props:Props){
 const {scope,source,language,templateKey,taskIds,ownItemId,disabled,onChange}=props;
 const context=useWorkbenchContext();
 const storage=`image-text:${scope.userId}:${scope.organizationId}:${scope.kind}:${scope.contextId}:${source.EffectiveVersion}:${templateKey}:${language}`;
 const [mode,setMode]=useState<"none"|"reviewed">("none"),[expanded,setExpanded]=useState(false),[ready,setReady]=useState(false);
 const [original,setOriginal]=useState<Original>(),[ownSource,setOwnSource]=useState<OwnSourceSelection>();
 const [proposalId,setProposalId]=useState(""),[proposal,setProposal]=useState<ProductTitleProposal>();
 const [pending,setPending]=useState<ReviewIntent>(),[busy,setBusy]=useState(false),[error,setError]=useState("");
 const [confirmedTasks,setConfirmed]=useState<string|null>(null);
 const taskKey=JSON.stringify(taskIds),confirmed=confirmedTasks===taskKey;
 useEffect(()=>{
  let active=true;queueMicrotask(()=>{if(!active)return;
  try{
   const raw=localStorage.getItem(storage+":request");
   if(raw){const v=JSON.parse(raw);const parsed=(scope.kind==="supply"?ownContentStartRequestSchema:contentAgentStartRequestSchema).safeParse(v.start);if(!isAcquisitionUUID(v.key)||!parsed.success||!parsed.data.manualReviewOnly||parsed.data.contentSelection.targetLanguage!==language)throw Error("原翻译请求无法核实，已保留原文");setOriginal({key:v.key,start:parsed.data});if(scope.kind==="supply")setOwnSource(ownContentStartRequestSchema.parse(parsed.data).ownSource);setExpanded(true);setMode("reviewed");onChange(null)}
   const id=localStorage.getItem(storage+":proposal");if(id&&isAcquisitionUUID(id))setProposalId(id);
   const command=localStorage.getItem(storage+":review");if(command){const v=JSON.parse(command);if(!isAcquisitionUUID(v.key)||!isAcquisitionUUID(v.proposalId)||!productTitleDecisionSchema.safeParse(v.body).success)throw Error("原审核请求无法核实，已保留原文");setPending(v);setMode("reviewed");onChange(null)}
  }catch(e){setError(e instanceof Error?e.message:"原请求无法读取")}finally{setReady(true)}
  });return()=>{active=false};
 // This scope is immutable; the parent remounts on source/template/language changes.
 // eslint-disable-next-line react-hooks/exhaustive-deps
 },[storage]);
 async function readReview(){
  if(!isAcquisitionUUID(proposalId)||busy)return;setBusy(true);setError("");setConfirmed(null);onChange(null);
  try{const p=await fetchProductTitleProposal({...scope,proposalId});setProposal(p);localStorage.setItem(storage+":proposal",p.proposal_id)}catch{setProposal(undefined);setError("审核记录暂不可读，请保留原编号核实当前身份、企业与权限。")}finally{setBusy(false)}
 }
 async function begin(){
  if(disabled||busy||!ready)return;setError("");
  if(scope.kind==="supply"&&!original){
   if(!ownItemId){setError("此来源暂未提供对应的商品翻译入口；可读取已有审核译文或选择不加文字。");return}
   setBusy(true);try{const s=await readOwnContentSource(ownItemId,scope,new AbortController().signal);if(s.productKey!==source.ProductID||s.catalogVersion!==source.EffectiveVersion||source.OriginalVersion===source.EffectiveVersion&&s.publicationId!==source.OriginalPublicationID)throw Error("source mismatch");setOwnSource({catalogVersion:s.catalogVersion,publicationId:s.publicationId})}catch{setError("商品翻译来源与当前图片来源不一致，未开始翻译。");return}finally{setBusy(false)}
  }
  setExpanded(true);
 }
 function remember(key:string,start:StartIntent){
  const raw=JSON.stringify({key,start});const previous=localStorage.getItem(storage+":request");if(previous&&previous!==raw)throw Error("原翻译请求尚未收束");localStorage.setItem(storage+":request",raw);if(localStorage.getItem(storage+":request")!==raw)throw Error("请求未保留");setOriginal({key,start});
 }
 function resetRejectedRequest(){
  if(pending)throw Error("原审核操作尚未核实");
  const key=storage+":request",raw=localStorage.getItem(key);
  if(raw){const history=storage+":request-history:"+original?.key;localStorage.setItem(history,raw);if(localStorage.getItem(history)!==raw)throw Error("原请求未保留");localStorage.removeItem(key)}
  setOriginal(undefined);
 }
 async function decide(command:ReviewIntent){
  if(busy||disabled)return;const key=storage+":review";const raw=JSON.stringify(command);setBusy(true);setError("");setConfirmed(null);onChange(null);
  try{
   const previous=localStorage.getItem(key);if(previous&&previous!==raw)throw Error("原审核操作尚未核实");localStorage.setItem(key,raw);if(localStorage.getItem(key)!==raw)throw Error("审核请求未完整保留");setPending(command);
   await decideProductTitleProposal({...scope,proposalId:command.proposalId,idempotencyKey:command.key,input:command.body});
   localStorage.setItem(storage+":review-history:"+command.key,raw);if(localStorage.getItem(storage+":review-history:"+command.key)!==raw)throw Error("原审核证据未完整保留");localStorage.removeItem(key);setPending(undefined);
   const current=await fetchProductTitleProposal({...scope,proposalId:command.proposalId});setProposal(current);
  }catch(e){if(e instanceof ProductTitleReviewError&&e.outcome==="rejected"){localStorage.setItem(storage+":review-history:"+command.key,raw);localStorage.removeItem(key);setPending(undefined)}setError(e instanceof ProductTitleReviewError&&e.outcome==="unknown"?"审核结果待核实，已保留原请求。请核实原操作，不要生成新译文。":"审核操作未完成，请核实原编号与当前权限。")}finally{setBusy(false)}
 }
 const lines=proposal?confirmedImageTextLines(proposal):[];
 const matches=proposal?.policy==="product-content-review-v1"&&proposal.input.product_key===source.ProductID&&proposal.input.base_version===source.EffectiveVersion&&proposal.content.selection.targetLanguage===language&&proposal.execution_mode?.mode==="default"&&!proposal.policy_adoption&&!proposal.apply_receipt;
 const usable=matches&&proposal.state==="accepted"&&lines.length>0&&lines.every(line=>[...line].length<=32)&&!pending;
 function confirm(value:boolean){setConfirmed(value?taskKey:null);onChange(value&&usable&&proposal?{mode:"reviewed",review:{proposalId:proposal.proposal_id,revision:proposal.revision},taskIds}:null)}
 return <section aria-label="图片文字语言" className="space-y-4 rounded-xl border border-border p-4">
  <h3 className="font-semibold">目标语言：{imageTextLanguageNames[language]??language}（{language}）</h3>
  <p className="text-sm text-muted-foreground">来自所选模板版本。翻译、人工审核和图片生成分别确认；旧任务保留原文案与语言。</p>
  <label className="block space-y-2 text-sm">图片文字<Select aria-label="图片文字方式" value={mode} disabled={disabled||busy||!!pending} onChange={e=>{const next=e.target.value as "none"|"reviewed";setMode(next);setConfirmed(null);onChange(next==="none"?{mode:"none"}:null)}}><option value="none">不加文字</option><option value="reviewed">自动翻译后人工确认</option></Select></label>
  {mode==="reviewed"&&<>
   <Button variant="outline" disabled={disabled||busy||!ready||!!pending||!context.permissions.includes("local_agent.write")} onClick={()=>void begin()}>翻译商品文案（先确认文本预算）</Button>
   {!context.permissions.includes("local_agent.write")&&<p>当前身份没有文本执行与审核访问权；可按现有权限选择不加文字，不会自动增加权限。</p>}
   {expanded&&(scope.kind==="acquisition"||ownSource)&&<ProductAgentPanel enabled operationId={scope.kind==="supply"?ownItemId!:scope.contextId} productKey={source.ProductID} catalogVersion={source.EffectiveVersion} ownSource={ownSource} imageTranslation={{language,original,persist:remember,reset:resetRejectedRequest,onReview:id=>{setProposalId(id);localStorage.setItem(storage+":proposal",id)}}}/>}
   <div className="flex flex-wrap items-end gap-2"><label className="block text-sm">文案审核编号<Input aria-label="文案审核编号" value={proposalId} disabled={busy||!!pending||disabled} onChange={e=>{setProposalId(e.target.value);setProposal(undefined);setConfirmed(null);onChange(null)}}/></label><Button variant="outline" disabled={busy||!isAcquisitionUUID(proposalId)} onClick={()=>void readReview()}>读取审核译文</Button></div>
   {pending&&<><p role="status">原审核操作尚待核实，其他写操作保持锁定。</p><Button disabled={busy||disabled} variant="outline" onClick={()=>void decide(pending)}>核实原审核操作</Button></>}
   {proposal?.policy==="product-content-review-v1"&&<>
    <ContentComparison proposal={proposal}/>
    {!pending&&matches&&<TitleReviewControls proposal={proposal} roles={context.roles} permissions={context.permissions} userId={scope.userId} pending={busy||disabled} allowApply={false} apply={()=>{}} decide={(action,title,content:ContentValue[]|undefined)=>{const body:ProductTitleDecisionInput=action==="edit"?{action,expected_revision:proposal.revision,content:content??[]}:{action,expected_revision:proposal.revision};void decide({key:crypto.randomUUID(),proposalId:proposal.proposal_id,body})}}/>}
    {!context.permissions.includes("listingkit.admin.write")&&proposal.state!=="accepted"&&<p>译文须由已有审核权限的成员接受后才能用于图片；本次不改变成员权限。</p>}
    {!matches&&<p role="alert">审核来源、目标语言或人工模式与本次模板不一致，不能带入图片。</p>}
    {lines.some(line=>[...line].length>32)&&<p role="alert">图片文案每条最多32个字符，请在审核中编辑并重新接受；不会截断或换回原文。</p>}
    <div className="space-y-2 rounded-lg bg-accent/30 p-4"><h4 className="font-semibold">本次图片文案 · 最多两条</h4>{lines.map((line,i)=><p key={i} className="break-words">{line}</p>)}<p className="text-xs text-muted-foreground">审核 {proposal.proposal_id} · 修订 {proposal.revision}；将用于本次所选 {taskIds.length} 项图片任务，随计划固定。</p></div>
    <label className="flex items-start gap-2 text-sm"><input type="checkbox" aria-label="确认目标语言图片文案" checked={confirmed} disabled={!usable||busy||disabled||taskIds.length===0} onChange={e=>confirm(e.target.checked)}/>已核对目标语言、商品事实与单位，确认使用这些文案</label>
   </>}
  </>}
  {error&&<p role="alert">{error}</p>}
 </section>;
}
