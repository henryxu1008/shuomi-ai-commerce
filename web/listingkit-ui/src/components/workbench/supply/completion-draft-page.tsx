"use client";
import {useEffect,useMemo,useRef,useState} from "react";
import Link from "next/link";
import Image from "next/image";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {DescriptionEditor} from "../description-editor";
import {SourceImageUploader} from "../collections/source-image-uploader";
import {descriptionReferences} from "@/lib/contracts/product-description";
import {DescriptionView} from "../description-view";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {OwnProductContents} from "../collections/own-product-detail-page";
import {SupplyCommandFeedback} from "./command-feedback";
import {useSupplyCommand} from "./use-supply-command";
import {SupplyAPIError,readSupplyCompletionDraft,readSupplySource,type SupplyScope} from "@/lib/api/supply-chain";
import {completionInputSchema,type SupplyCompletionDraft,type SupplySourceDetail} from "@/lib/contracts/supply-chain";
import {completionHistoryHref,completionStructuredHref} from "@/lib/contracts/completion-draft-navigation";
import {sheinImportHref} from "@/lib/contracts/shein-import-navigation";

export function CompletionDraftPage({sourceId,preparationId,itemId,sheinAvailable=false}:{sourceId:string;preparationId?:string;itemId?:string;sheinAvailable?:boolean}){
 const context=useWorkbenchContext(),userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const [uploadBlocked,setUploadBlocked]=useState(false);
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.error||context.blockingError||context.selectionRequired)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 if(!context.permissions.includes("workbench.supply.read"))return <ConsoleState kind="unavailable" title="当前身份没有供应链查看权限"/>;
 const list="/workbench/supply/mine/missing"+(preparationId?"?preparation="+preparationId:"");
 return <div onClickCapture={event=>{if(uploadBlocked&&event.target instanceof Element&&event.target.closest("a"))event.preventDefault()}}><ConsolePage title="待补全草稿" description="候选标题和描述独立保存，原始商品版本保持固定。" breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:"我的供应链",href:"/workbench/supply/mine"},{label:"待补全",href:list},{label:"编辑草稿"}]} actions={<div className="flex flex-wrap gap-3">{itemId?<Button asChild variant="outline"><Link href={`/workbench/data/mine/own/${itemId}`}>返回商品</Link></Button>:null}<Button asChild variant="outline"><Link href={list}>返回待补全列表</Link></Button></div>}>
  <CompletionDraftEditor key={`${userId}:${organizationId}:${sourceId}:${preparationId??""}:${itemId??""}`} scope={scope} sourceId={sourceId} preparationId={preparationId} itemId={itemId} canManage={context.permissions.includes("workbench.supply.manage")} onUploadBlocked={setUploadBlocked} sheinAvailable={sheinAvailable} onSaved={()=>{}}/>
 </ConsolePage></div>;
}
function CompletionDraftEditor({scope,sourceId,preparationId,itemId,canManage,onSaved,sheinAvailable,onUploadBlocked}:{scope:SupplyScope;sourceId:string;preparationId?:string;itemId?:string;canManage:boolean;onSaved:()=>void;sheinAvailable:boolean;onUploadBlocked:(blocked:boolean)=>void}){
 const context=useWorkbenchContext();
 const [descriptionImages,setDescriptionImages]=useState<string[]>([]),[mediaBlocked,setMediaBlocked]=useState(false);
 const imagesRef=useRef<string[]>([]);
 useEffect(()=>{onUploadBlocked(mediaBlocked);return()=>onUploadBlocked(false)},[mediaBlocked,onUploadBlocked]);
 const [descriptionFormat,setDescriptionFormat]=useState<"html-v1"|undefined>();
 const [source,setSource]=useState<SupplySourceDetail>(),[head,setHead]=useState<SupplyCompletionDraft>(),[title,setTitle]=useState(""),[description,setDescription]=useState("");
 const [reload,setReload]=useState(0),[loading,setLoading]=useState(true),[error,setError]=useState(false),[conflictReviewed,setConflictReviewed]=useState(false);
 const refresh=()=>{setConflictReviewed(true);setLoading(true);setReload(v=>v+1)};
 const command=useSupplyCommand(scope,(_,intent)=>{if(intent.route==="save-completion-draft"){setLoading(true);setReload(v=>v+1);onSaved()}});
 useEffect(()=>{const c=new AbortController();let live=true;
  void (async()=>{const detail=await readSupplySource(scope,sourceId,c.signal);if(preparationId&&detail.source.preparationId!==preparationId)throw new Error("source differs from selected batch");if(itemId&&(detail.source.collectionItemId!==itemId||detail.source.source.kind!=="own"))throw new Error("source differs from selected own item");
   let saved:SupplyCompletionDraft|undefined;try{saved=await readSupplyCompletionDraft(scope,sourceId,c.signal)}catch(e){if(!(e instanceof SupplyAPIError)||e.code!=="NOT_FOUND")throw e}
   if(saved&&JSON.stringify(saved.source)!==JSON.stringify(detail.source))throw new Error("Draft and fixed source differ");
   if(live){setSource(detail);setHead(saved);setTitle(saved?.content.title??detail.product.title??"");setDescription(saved?.content.description??detail.product.description??"");setDescriptionFormat(saved?saved.content.descriptionFormat:detail.product.description_format);setDescriptionImages(saved?.content.descriptionImages??[]);imagesRef.current=saved?.content.descriptionImages??[];setError(false)}
  })().catch(()=>{if(live)setError(true)}).finally(()=>{if(live)setLoading(false)});return()=>{live=false;c.abort()};
 },[scope,sourceId,preparationId,itemId,reload]);
 const conflict=command.error==="REVISION_CONFLICT"&&!conflictReviewed;
 const blocked=loading||error||!canManage||conflict||!!command.pending||command.busy||!command.ready;
 const input={sourceId,expectedRevision:head?.revision??0,content:{...head?.content,title,description,descriptionFormat,descriptionImages:descriptionImages.length?descriptionImages:undefined}};
 return <div className="space-y-5"><SupplyCommandFeedback state={conflictReviewed&&command.error==="REVISION_CONFLICT"?{...command,error:null}:command}/>{loading?<ConsoleState kind="loading" title="正在读取原资料与保存版本"/>:error?<ConsoleState kind="error" title="草稿或原资料读取失败">未读取准确版本前无法保存。<Button variant="outline" onClick={refresh}>重新读取</Button></ConsoleState>:source?<>
  <div className="rounded-lg border border-border bg-muted p-3 text-sm text-muted-foreground">未绑定店铺 · {head?`已保存 R${head.revision}`:"尚未保存"} · 平台必填项待配置后确定</div>
  {head&&!command.busy&&!command.pending?<Button asChild variant="outline"><Link href={completionHistoryHref(sourceId,source.source.preparationId,itemId)}>保存历史</Link></Button>:null}
  {sheinAvailable&&head&&canManage&&!command.busy&&!command.pending?<Button asChild variant="outline"><Link href={sheinImportHref(sourceId,source.source.preparationId,source.source.source.kind==="own"?source.source.collectionItemId:undefined,head.revision)} prefetch={false}>SHEIN 带入预览（已保存 R{head.revision}）</Link></Button>:null}
  {!command.busy&&!command.pending?<Button asChild variant="outline"><Link href={completionStructuredHref(sourceId,source.source.preparationId,source.source.source.kind==="own"?source.source.collectionItemId:undefined)}>编辑商品属性和 SKU</Link></Button>:null}
  <div className="grid gap-5 md:grid-cols-2"><section className="space-y-3 rounded-xl border p-4" aria-label="原始商品资料"><h3 className="font-semibold">原始商品资料</h3><p className="font-medium">{source.product.title||"原商品未填写标题"}</p><DescriptionView value={source.product.description} format={source.product.description_format} images={source.images.map(v=>v.url)}/><p className="break-all text-xs text-muted-foreground">{source.source.source.productKey} · 原始版本 {source.source.source.version}</p><div className="flex flex-wrap gap-2">{source.images.map((img,index)=><Image key={img.id} unoptimized src={`/api/image-proxy?url=${encodeURIComponent(img.url)}`} width={96} height={96} alt={`原始商品图片 ${index+1}`} className="rounded-md object-contain"/>)}</div><details className="text-sm"><summary>查看固定原始商品资料</summary>{source.source.source.kind==="own"?<div className="mt-4"><OwnProductContents detail={{product:source.product,item:{source:source.source.source}}}/></div>:<pre className="mt-3 max-h-80 overflow-auto whitespace-pre-wrap break-all rounded-md bg-muted p-3 text-xs">{JSON.stringify(source.product,null,2)}</pre>}</details><Link className="text-sm text-primary underline" href={source.source.source.kind==="own"?`/workbench/data/mine/own/${source.source.collectionItemId}`:`/workbench/data/mine?item=${source.source.collectionItemId}`}>查看我的数据</Link></section>
  <form className="space-y-4 rounded-xl border p-4" onSubmit={e=>{e.preventDefault();if(!blocked&&!mediaBlocked&&completionInputSchema.safeParse(input).success){setConflictReviewed(false);void command.execute("save-completion-draft",input)}}}><h3 className="font-semibold">候选商品资料</h3><fieldset disabled={blocked} className="space-y-4"><label className="grid gap-2 text-sm">候选标题<Input aria-label="候选标题" value={title} onChange={e=>setTitle(e.target.value)}/><span className="text-xs text-muted-foreground">{new TextEncoder().encode(title).length} / 1024 字节</span></label><DescriptionEditor label="候选描述" value={description} format={descriptionFormat} images={[...new Set([...source.images.map(v=>v.url),...descriptionImages])]} disabled={blocked} maxBytes={32768} onChange={(value,format)=>{setDescription(value);setDescriptionFormat(format);const refs=descriptionReferences(value,format);const next=imagesRef.current.filter(url=>refs.includes(url));imagesRef.current=next;setDescriptionImages(next)}} uploadBlocked={mediaBlocked} renderUpload={insert=><div>{!context.permissions.includes("workbench.collection.manage")?<p className="text-sm text-muted-foreground">当前没有图片上传权限，仍可编辑文字和使用已保存的图片。</p>:null}<SourceImageUploader scope={scope} disabled={blocked||!context.permissions.includes("workbench.collection.manage")} allowNewUpload={descriptionImages.length<40} onBlocked={setMediaBlocked} onImage={url=>{if(blocked)return;const next=[...new Set([...imagesRef.current,url])];if(next.length>40)return;imagesRef.current=next;setDescriptionImages(next);insert(url)}}/></div>}/></fieldset><p className="text-xs text-muted-foreground">允许留空。新增描述图片只随候选版本保存，不改变固定来源或SKU图片。平台带入使用纯文字，图片仍需另行批准。</p>{!completionInputSchema.safeParse(input).success?<p role="alert" className="text-sm text-red-700">内容超出长度上限或包含无效字符。</p>:null}<Button type="submit" disabled={blocked||mediaBlocked||!completionInputSchema.safeParse(input).success}>保存待补全草稿</Button><Button className="ml-2" variant="outline" disabled={command.busy||!!command.pending||loading||mediaBlocked} onClick={refresh}>重新读取保存版本</Button></form></div>
 </>:null}</div>;
}
