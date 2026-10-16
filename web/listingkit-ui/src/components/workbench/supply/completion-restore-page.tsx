"use client";
import {restoreDescription} from "@/lib/contracts/product-description";
import {useEffect,useMemo,useState} from "react";
import Link from "next/link";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Badge} from "@/components/ui/badge";
import {Card} from "@/components/ui/card";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {readCompletionRevision,readSupplyCompletionDraft,readSupplySource,type SupplyScope} from "@/lib/api/supply-chain";
import {completionInputSchema,completionReceiptSchema,type SupplyCompletionDraft} from "@/lib/contracts/supply-chain";
import {completionDraftHref,completionHistoryHref,completionRevision} from "@/lib/contracts/completion-draft-navigation";
import {CompletionTextComparison} from "./completion-history-page";
import {useSupplyCommand} from "./use-supply-command";
import {SupplyCommandFeedback} from "./command-feedback";
import {completionContentEqual} from "@/lib/contracts/completion-content";

type Props={sourceId:string;revision:number;preparationId?:string;itemId?:string};
type Preview={old:SupplyCompletionDraft;head:SupplyCompletionDraft};
type CompletedSave={record:SupplyCompletionDraft;input:ReturnType<typeof completionInputSchema.parse>};
const sameContent=(left:SupplyCompletionDraft["content"],right:SupplyCompletionDraft["content"])=>left.title===right.title&&left.description===right.description&&left.descriptionFormat===right.descriptionFormat;
const sameSource=(left:SupplyCompletionDraft["source"],right:SupplyCompletionDraft["source"])=>left.id===right.id&&left.preparationId===right.preparationId&&left.collectionItemId===right.collectionItemId&&left.collectionRevision===right.collectionRevision
 &&left.source.kind===right.source.kind&&left.source.productKey===right.source.productKey&&left.source.publicationId===right.source.publicationId&&left.source.version===right.source.version&&left.source.operationId===right.source.operationId;

export function CompletionRestorePage(props:Props){
 const context=useWorkbenchContext(),userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.error||context.blockingError||context.selectionRequired)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 if(!context.permissions.includes("workbench.supply.read")||!context.permissions.includes("workbench.supply.manage"))return <ConsoleState kind="unavailable" title="当前身份没有草稿恢复权限"/>;
 if(completionRevision(String(props.revision))===null)return <ConsoleState kind="error" title="恢复版本参数无效"/>;
 return <ScopedRestore key={`${userId}:${organizationId}:${props.sourceId}:${props.revision}:${props.preparationId??""}:${props.itemId??""}`} {...props} scope={scope}/>;
}

function ScopedRestore({scope,sourceId,revision,preparationId,itemId}:Props&{scope:SupplyScope}){
 const [preview,setPreview]=useState<Preview>(),[completed,setCompleted]=useState<CompletedSave>(),[loading,setLoading]=useState(true),[readError,setReadError]=useState(false);
 const [reload,setReload]=useState(0),[conflictReviewed,setConflictReviewed]=useState(false);
 const reloadPreview=()=>{setLoading(true);setReload(value=>value+1)};
 const command=useSupplyCommand(scope,(result,intent)=>{
  if(intent.route==="save-completion-draft"){
   const input=completionInputSchema.safeParse(intent.command);
   if(input.success&&input.data.sourceId===sourceId){
    const receipt=completionReceiptSchema.safeParse(result);
    if(!receipt.success||receipt.data.record.source.id!==sourceId||receipt.data.record.revision!==input.data.expectedRevision+1||!completionContentEqual(receipt.data.record.content,input.data.content)){
     setReadError(true);return;
    }
    setCompleted({record:receipt.data.record,input:input.data});
   }
  }
  reloadPreview();
 });
 useEffect(()=>{
  const controller=new AbortController();let active=true;
  void Promise.all([readCompletionRevision(scope,sourceId,revision,controller.signal),readSupplyCompletionDraft(scope,sourceId,controller.signal),readSupplySource(scope,sourceId,controller.signal)]).then(([old,head,detail])=>{
   const source=detail.source;
   if(source.id!==sourceId||old.revision!==revision||old.draftId!==head.draftId||!sameSource(old.source,source)||!sameSource(head.source,source)||old.revision>head.revision||preparationId&&source.preparationId!==preparationId||itemId&&(source.collectionItemId!==itemId||source.source.kind!=="own"))throw new Error("Saved content, current head or navigation differ");
   if(active){setPreview({old,head});setReadError(false)}
  }).catch(()=>{if(active)setReadError(true)}).finally(()=>{if(active)setLoading(false)});
  return()=>{active=false;controller.abort()};
 },[scope,sourceId,revision,preparationId,itemId,reload]);
 const loaded=!loading&&!readError?preview:undefined;
 const restored=loaded&&completed&&sameContent(completed.input.content,loaded.old.content)?completed.record:undefined;
 const receiptError=!!restored&&!!loaded&&(!sameSource(restored.source,loaded.old.source)||restored.draftId!==loaded.old.draftId);
 const visible=receiptError?undefined:loaded,source=visible?.old.source,saved=receiptError?undefined:restored;
 const current=visible?.head,old=visible?.old;
 const conflict=command.error==="REVISION_CONFLICT"&&!conflictReviewed;
 const disabled=loading||readError||receiptError||!current||!old||old.revision>=current.revision||sameContent(old.content,current.content)||current.revision>=Number.MAX_SAFE_INTEGER||!!saved||conflict||!!command.pending||command.busy||!command.ready;
 const refresh=()=>{setCompleted(undefined);setConflictReviewed(true);reloadPreview()};
 return <ConsolePage title="恢复草稿历史内容" description="预览旧版标题和描述，确认后保存为新版本。" breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:"我的供应链",href:"/workbench/supply/mine"},{label:"保存历史",href:source?completionHistoryHref(sourceId,source.preparationId,itemId):undefined},{label:`恢复 R${revision} 内容`}]} actions={source?<div className="flex flex-wrap gap-3">
  <Button asChild variant="outline"><Link href={completionHistoryHref(sourceId,source.preparationId,itemId,{revision})}>返回旧版详情</Link></Button>
  <Button asChild variant="outline"><Link href={completionDraftHref(sourceId,source.preparationId,itemId)}>返回草稿</Link></Button>
  {itemId?<Button asChild variant="outline"><Link href={`/workbench/data/mine/own/${itemId}`}>返回商品</Link></Button>:null}
  <Button asChild variant="outline"><Link href={`/workbench/supply/mine/missing?preparation=${source.preparationId}`}>返回待补全列表</Link></Button>
 </div>:undefined}>
  <SupplyCommandFeedback state={conflictReviewed&&command.error==="REVISION_CONFLICT"?{...command,error:null}:command}/>
  {loading?<ConsoleState kind="loading" title="正在读取旧版与当前保存版本"/>:readError||receiptError?<ConsoleState kind="error" title="无法读取恢复内容">请确认当前权限、来源和版本。<Button variant="outline" disabled={command.busy||!!command.pending} onClick={refresh}>重新读取当前版本</Button></ConsoleState>:visible&&source?<div className="space-y-5">
   {saved?<Card role="status" className="space-y-3 p-5"><h2 className="font-semibold">已保存为 R{saved.revision}</h2><p className="text-sm text-muted-foreground">原操作的标题和描述已保存，全部历史记录及固定来源保持不变。当前版本以下方重新读取结果为准。</p><Button asChild variant="outline"><Link href={completionHistoryHref(sourceId,source.preparationId,itemId,{revision:saved.revision})}>查看已保存 R{saved.revision}</Link></Button></Card>:null}
   <Card className="flex flex-wrap items-center gap-3 p-5"><Badge variant="secondary">固定来源 V{source.source.version}</Badge><p className="break-all text-xs text-muted-foreground">来源 {sourceId} · 转入批次 {source.preparationId}</p></Card>
   <CompletionTextComparison records={[visible.head,visible.old]} title="恢复内容预览" labels={[`当前已保存 R${visible.head.revision}`,`将采用旧版 R${revision} 内容`]}/>
   <Card className="space-y-4 p-5">
    <p className="text-sm">确认后将旧版 R{revision} 的标题和描述保存为新的版本，当前候选属性、价格、库存和全部历史记录保持不变。</p>
    {visible.old.revision===visible.head.revision?<p className="text-sm text-muted-foreground">所选版本已是当前版本，无需恢复。</p>:sameContent(visible.old.content,visible.head.content)?<p className="text-sm text-muted-foreground">标题和描述与当前版本相同，无需恢复。</p>:null}
    {visible.head.revision>=Number.MAX_SAFE_INTEGER?<p role="alert">保存版本已达上限，无法追加新版本。</p>:null}
    {conflict?<p role="alert" className="text-sm text-destructive">当前版本已变化，请重新读取并确认差异后再恢复。</p>:null}
    <div className="flex flex-wrap gap-3"><Button disabled={disabled} onClick={()=>{if(!disabled&&visible){setConflictReviewed(false);void command.execute("save-completion-draft",{sourceId,expectedRevision:visible.head.revision,content:restoreDescription(visible.head.content,visible.old.content)})}}}>恢复为新版本</Button><Button variant="outline" disabled={command.busy||!!command.pending} onClick={refresh}>重新读取当前版本</Button><Button asChild variant="outline"><Link href={completionHistoryHref(sourceId,source.preparationId,itemId)}>返回保存历史</Link></Button></div>
   </Card>
  </div>:null}
 </ConsolePage>;
}
