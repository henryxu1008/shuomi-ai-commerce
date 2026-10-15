"use client";
import {useEffect,useMemo,useState} from "react";
import Link from "next/link";
import Image from "next/image";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Textarea} from "@/components/ui/textarea";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {OwnProductContents} from "../collections/own-product-detail-page";
import {SupplyCommandFeedback} from "./command-feedback";
import {useSupplyCommand} from "./use-supply-command";
import {SupplyAPIError,readSupplyCompletionDraft,readSupplySource,type SupplyScope} from "@/lib/api/supply-chain";
import {completionInputSchema,type SupplyCompletionDraft,type SupplySourceDetail} from "@/lib/contracts/supply-chain";
import {completionHistoryHref,completionStructuredHref} from "@/lib/contracts/completion-draft-navigation";

export function CompletionDraftPage({sourceId,preparationId,itemId}:{sourceId:string;preparationId?:string;itemId?:string}){
 const context=useWorkbenchContext(),userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.error||context.blockingError||context.selectionRequired)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 if(!context.permissions.includes("workbench.supply.read"))return <ConsoleState kind="unavailable" title="当前身份没有供应链查看权限"/>;
 const list="/workbench/supply/mine/missing"+(preparationId?"?preparation="+preparationId:"");
 return <ConsolePage title="待补全草稿" description="候选标题和描述独立保存，原始商品版本保持固定。" breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:"我的供应链",href:"/workbench/supply/mine"},{label:"待补全",href:list},{label:"编辑草稿"}]} actions={<div className="flex flex-wrap gap-3">{itemId?<Button asChild variant="outline"><Link href={`/workbench/data/mine/own/${itemId}`}>返回商品</Link></Button>:null}<Button asChild variant="outline"><Link href={list}>返回待补全列表</Link></Button></div>}>
  <CompletionDraftEditor key={`${userId}:${organizationId}:${sourceId}:${preparationId??""}:${itemId??""}`} scope={scope} sourceId={sourceId} preparationId={preparationId} itemId={itemId} canManage={context.permissions.includes("workbench.supply.manage")} onSaved={()=>{}}/>
 </ConsolePage>;
}
function CompletionDraftEditor({scope,sourceId,preparationId,itemId,canManage,onSaved}:{scope:SupplyScope;sourceId:string;preparationId?:string;itemId?:string;canManage:boolean;onSaved:()=>void}){
 const [source,setSource]=useState<SupplySourceDetail>(),[head,setHead]=useState<SupplyCompletionDraft>(),[title,setTitle]=useState(""),[description,setDescription]=useState("");
 const [reload,setReload]=useState(0),[loading,setLoading]=useState(true),[error,setError]=useState(false),[conflictReviewed,setConflictReviewed]=useState(false);
 const refresh=()=>{setConflictReviewed(true);setLoading(true);setReload(v=>v+1)};
 const command=useSupplyCommand(scope,(_,intent)=>{if(intent.route==="save-completion-draft"){setLoading(true);setReload(v=>v+1);onSaved()}});
 useEffect(()=>{const c=new AbortController();let live=true;
  void (async()=>{const detail=await readSupplySource(scope,sourceId,c.signal);if(preparationId&&detail.source.preparationId!==preparationId)throw new Error("source differs from selected batch");if(itemId&&(detail.source.collectionItemId!==itemId||detail.source.source.kind!=="own"))throw new Error("source differs from selected own item");
   let saved:SupplyCompletionDraft|undefined;try{saved=await readSupplyCompletionDraft(scope,sourceId,c.signal)}catch(e){if(!(e instanceof SupplyAPIError)||e.code!=="NOT_FOUND")throw e}
   if(saved&&JSON.stringify(saved.source)!==JSON.stringify(detail.source))throw new Error("Draft and fixed source differ");
   if(live){setSource(detail);setHead(saved);setTitle(saved?.content.title??detail.product.title??"");setDescription(saved?.content.description??detail.product.description??"");setError(false)}
  })().catch(()=>{if(live)setError(true)}).finally(()=>{if(live)setLoading(false)});return()=>{live=false;c.abort()};
 },[scope,sourceId,preparationId,itemId,reload]);
 const conflict=command.error==="REVISION_CONFLICT"&&!conflictReviewed;
 const blocked=loading||error||!canManage||conflict||!!command.pending||command.busy||!command.ready;
 const input={sourceId,expectedRevision:head?.revision??0,content:{...head?.content,title,description}};
 return <div className="space-y-5"><SupplyCommandFeedback state={conflictReviewed&&command.error==="REVISION_CONFLICT"?{...command,error:null}:command}/>{loading?<ConsoleState kind="loading" title="正在读取原资料与保存版本"/>:error?<ConsoleState kind="error" title="草稿或原资料读取失败">未读取准确版本前无法保存。<Button variant="outline" onClick={refresh}>重新读取</Button></ConsoleState>:source?<>
  <div className="rounded-lg border border-border bg-muted p-3 text-sm text-muted-foreground">未绑定店铺 · {head?`已保存 R${head.revision}`:"尚未保存"} · 平台必填项待配置后确定</div>
  {head&&!command.busy&&!command.pending?<Button asChild variant="outline"><Link href={completionHistoryHref(sourceId,source.source.preparationId,itemId)}>保存历史</Link></Button>:null}
  {!command.busy&&!command.pending?<Button asChild variant="outline"><Link href={completionStructuredHref(sourceId,source.source.preparationId,source.source.source.kind==="own"?source.source.collectionItemId:undefined)}>编辑商品属性和 SKU</Link></Button>:null}
  <div className="grid gap-5 md:grid-cols-2"><section className="space-y-3 rounded-xl border p-4" aria-label="原始商品资料"><h3 className="font-semibold">原始商品资料</h3><p className="font-medium">{source.product.title||"原商品未填写标题"}</p><p className="whitespace-pre-wrap text-sm">{source.product.description||"原商品未填写描述"}</p><p className="break-all text-xs text-muted-foreground">{source.source.source.productKey} · 原始版本 {source.source.source.version}</p><div className="flex flex-wrap gap-2">{source.images.map((img,index)=><Image key={img.id} unoptimized src={`/api/image-proxy?url=${encodeURIComponent(img.url)}`} width={96} height={96} alt={`原始商品图片 ${index+1}`} className="rounded-md object-contain"/>)}</div><details className="text-sm"><summary>查看固定原始商品资料</summary>{source.source.source.kind==="own"?<div className="mt-4"><OwnProductContents detail={{product:source.product,item:{source:source.source.source}}}/></div>:<pre className="mt-3 max-h-80 overflow-auto whitespace-pre-wrap break-all rounded-md bg-muted p-3 text-xs">{JSON.stringify(source.product,null,2)}</pre>}</details><Link className="text-sm text-primary underline" href={source.source.source.kind==="own"?`/workbench/data/mine/own/${source.source.collectionItemId}`:`/workbench/data/mine?item=${source.source.collectionItemId}`}>查看我的数据</Link></section>
  <form className="space-y-4 rounded-xl border p-4" onSubmit={e=>{e.preventDefault();if(!blocked&&completionInputSchema.safeParse(input).success){setConflictReviewed(false);void command.execute("save-completion-draft",input)}}}><h3 className="font-semibold">候选商品资料</h3><fieldset disabled={blocked} className="space-y-4"><label className="grid gap-2 text-sm">候选标题<Input aria-label="候选标题" value={title} onChange={e=>setTitle(e.target.value)}/><span className="text-xs text-muted-foreground">{new TextEncoder().encode(title).length} / 1024 字节</span></label><label className="grid gap-2 text-sm">候选描述<Textarea aria-label="候选描述" rows={8} value={description} onChange={e=>setDescription(e.target.value)}/><span className="text-xs text-muted-foreground">{new TextEncoder().encode(description).length} / 32768 字节</span></label></fieldset><p className="text-xs text-muted-foreground">允许留空。保存为独立待补全版本，配置店铺后再带入平台适配。</p>{!completionInputSchema.safeParse(input).success?<p role="alert" className="text-sm text-red-700">内容超出长度上限或包含无效字符。</p>:null}<Button type="submit" disabled={blocked||!completionInputSchema.safeParse(input).success}>保存待补全草稿</Button><Button className="ml-2" variant="outline" disabled={command.busy||!!command.pending||loading} onClick={refresh}>重新读取保存版本</Button></form></div>
 </>:null}</div>;
}
