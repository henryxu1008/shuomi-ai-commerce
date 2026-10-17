"use client";
import {useEffect,useMemo,useState} from "react";
import Link from "next/link";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Card} from "@/components/ui/card";
import {Badge} from "@/components/ui/badge";
import {Table,TableBody,TableCell,TableHead,TableHeader,TableRow} from "@/components/ui/table";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {SupplyCommandFeedback} from "./command-feedback";
import {useSupplyCommand} from "./use-supply-command";
import {SupplyAPIError,readSupplySource,readSupplyCompletionDraft,type SupplyScope} from "@/lib/api/supply-chain";
import {COMPLETION_MAX_BYTES,completionInputSchema,type SupplyCompletionDraft,type SupplySourceDetail} from "@/lib/contracts/supply-chain";
import {completionDraftHref,completionHistoryHref} from "@/lib/contracts/completion-draft-navigation";
import {completionPayloadBytes,type CompletionStructured} from "@/lib/contracts/completion-content";

type Props={sourceId:string;preparationId?:string;itemId?:string};
type VariantFields={currency:string;amount:string;stock:string};
function fields(detail:SupplySourceDetail,saved?:SupplyCompletionDraft):{attributes:CompletionStructured["attributes"];variants:VariantFields[]}{
 const stored=saved?.content.structured;
 if(stored)return {attributes:stored.attributes.map(a=>({...a})),variants:stored.variants.map(v=>({currency:v.price?.currency??"",amount:v.price?String(v.price.amount):"",stock:v.stock===null?"":String(v.stock)}))};
 return {attributes:(detail.product.attributes??[]).map(a=>({name:a.name??"",value:a.value??""})),variants:(detail.product.variants??[]).map(v=>({currency:v.price?.currency??"",amount:v.price?.amount===undefined?"":String(v.price.amount),stock:v.stock===undefined?(detail.source.source.kind==="own"?"0":""):String(v.stock)}))};
}

export function CompletionStructuredPage(props:Props){
 const context=useWorkbenchContext(),userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.error||context.blockingError||context.selectionRequired)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 if(!context.permissions.includes("workbench.supply.read"))return <ConsoleState kind="unavailable" title="当前身份没有供应链查看权限"/>;
 return <StructuredEditor key={`${userId}:${organizationId}:${props.sourceId}:${props.preparationId??""}:${props.itemId??""}`} {...props} scope={scope} canManage={context.permissions.includes("workbench.supply.manage")}/>;
}
function StructuredEditor({scope,sourceId,preparationId,itemId,canManage}:Props&{scope:SupplyScope;canManage:boolean}){
 const [detail,setDetail]=useState<SupplySourceDetail>(),[head,setHead]=useState<SupplyCompletionDraft>(),[form,setForm]=useState<ReturnType<typeof fields>>({attributes:[],variants:[]});
 const [loading,setLoading]=useState(true),[error,setError]=useState(false),[reload,setReload]=useState(0),[conflictReviewed,setConflictReviewed]=useState(false);
 const refresh=()=>{setConflictReviewed(true);setLoading(true);setReload(n=>n+1)};
 const command=useSupplyCommand(scope,(_,intent)=>{if(intent.route==="save-completion-draft"&&completionInputSchema.parse(intent.command).sourceId===sourceId)refresh()});
 useEffect(()=>{const controller=new AbortController();let active=true;
  void(async()=>{
   const source=await readSupplySource(scope,sourceId,controller.signal);
   if(preparationId&&source.source.preparationId!==preparationId||itemId&&(source.source.collectionItemId!==itemId||source.source.source.kind!=="own"))throw new Error("Fixed source and navigation differ");
   let saved:SupplyCompletionDraft|undefined;try{saved=await readSupplyCompletionDraft(scope,sourceId,controller.signal)}catch(e){if(!(e instanceof SupplyAPIError)||e.code!=="NOT_FOUND")throw e}
   if(saved&&(JSON.stringify(saved.source)!==JSON.stringify(source.source)||saved.content.structured&&saved.content.structured.variants.length!==(source.product.variants?.length??0)))throw new Error("Saved candidate and fixed source differ");
   if(active){setDetail(source);setHead(saved);setForm(fields(source,saved));setError(false)}
  })().catch(()=>{if(active)setError(true)}).finally(()=>{if(active)setLoading(false)});
  return()=>{active=false;controller.abort()};
 },[scope,sourceId,preparationId,itemId,reload]);
 const source=!loading&&!error?detail?.source:undefined,ownItem=source?.source.kind==="own"?source.collectionItemId:undefined;
 const conflict=command.error==="REVISION_CONFLICT"&&!conflictReviewed;
 const overLimit=(detail?.product.variants?.length??0)>256;
 const input={sourceId,expectedRevision:head?.revision??0,content:{...(head?.content.descriptionImages?.length?{descriptionImages:head.content.descriptionImages}:{}),title:head?.content.title??detail?.product.title??"",description:head?.content.description??detail?.product.description??"",...((head?head.content.descriptionFormat:detail?.product.description_format)?{descriptionFormat:head?head.content.descriptionFormat:detail?.product.description_format}:{}),structured:{attributes:form.attributes,variants:form.variants.map((v,sourceIndex)=>({sourceIndex,price:v.amount.trim()===""?null:{currency:v.currency,amount:Number(v.amount)},stock:v.stock.trim()===""?null:Number(v.stock)}))}}};
 const valid=completionInputSchema.safeParse(input).success;
 const blocked=loading||error||overLimit||!canManage||conflict||command.busy||!!command.pending||!command.ready||(head?.revision??0)>=Number.MAX_SAFE_INTEGER;
 const changeVariant=(index:number,field:keyof VariantFields,value:string)=>setForm(old=>({...old,variants:old.variants.map((row,i)=>i===index?{...row,[field]:value}:row)}));
 return <ConsolePage title="编辑商品属性和 SKU" description="在固定来源版本上整理候选资料，保存为新的草稿版本。" breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:"我的供应链",href:"/workbench/supply/mine"},{label:"待补全草稿",href:source?completionDraftHref(sourceId,source.preparationId,ownItem):undefined},{label:"商品属性和 SKU"}]} actions={source?<div className="flex flex-wrap gap-3"><Button asChild variant="outline"><Link href={completionDraftHref(sourceId,source.preparationId,ownItem)}>返回草稿</Link></Button>{ownItem?<Button asChild variant="outline"><Link href={`/workbench/data/mine/own/${ownItem}`}>返回商品</Link></Button>:null}<Button asChild variant="outline"><Link href={`/workbench/supply/mine/missing?preparation=${source.preparationId}`}>返回待补全列表</Link></Button></div>:undefined}>
  <SupplyCommandFeedback state={conflictReviewed&&command.error==="REVISION_CONFLICT"?{...command,error:null}:command}/>
  {loading?<ConsoleState kind="loading" title="正在读取固定来源与保存版本"/>:error?<ConsoleState kind="error" title="结构化草稿读取失败"><Button onClick={refresh} variant="outline" disabled={command.busy||!!command.pending}>重新读取保存版本</Button></ConsoleState>:source&&detail?<div className="space-y-5">
   <Card className="space-y-2 p-5"><div className="flex flex-wrap items-center gap-3"><Badge variant="secondary">固定来源 V{source.source.version}</Badge><Badge variant="outline">{head?`已保存 R${head.revision}`:"尚未保存"}</Badge><span>{detail.product.title||"未填写来源标题"}</span></div><p className="break-all text-xs text-muted-foreground">转入批次 {source.preparationId}</p><p className="text-sm text-muted-foreground">{head?.content.structured?"正在编辑已保存的结构化候选资料。":"来源预填，尚未保存结构化候选资料。"} 标题和描述保留当前保存内容。</p>{head?<Link className="text-sm text-primary underline" href={completionHistoryHref(sourceId,source.preparationId,ownItem)}>保存历史</Link>:null}</Card>
   {overLimit?<ConsoleState kind="unavailable" title="来源 SKU 超过 256 条编辑上限">可返回草稿继续编辑文字。</ConsoleState>:<form className="space-y-5" onSubmit={e=>{e.preventDefault();if(blocked||!valid)return;setConflictReviewed(false);void command.execute("save-completion-draft",input)}}>
    <fieldset disabled={blocked} className="min-w-0 space-y-5">
     <Card className="space-y-4 p-5"><div className="flex flex-wrap items-center justify-between gap-3"><div><h2 className="font-semibold">候选商品属性</h2><p className="text-sm text-muted-foreground">商品通用信息，与来源 SKU 销售规格分别保存。</p></div><Button type="button" variant="outline" disabled={blocked||form.attributes.length>=256} onClick={()=>setForm(old=>({...old,attributes:[...old.attributes,{name:"",value:""}]}))}>添加属性</Button></div>{!form.attributes.length?<p className="text-sm text-muted-foreground">未添加候选属性。</p>:form.attributes.map((a,index)=><div key={index} className="grid gap-3 sm:grid-cols-[1fr_2fr_auto]"><label className="grid gap-2 text-sm">属性名称 {index+1}<Input aria-label={`属性名称 ${index+1}`} value={a.name} onChange={e=>setForm(old=>({...old,attributes:old.attributes.map((row,i)=>i===index?{...row,name:e.target.value}:row)}))}/></label><label className="grid gap-2 text-sm">属性值 {index+1}<Input aria-label={`属性值 ${index+1}`} value={a.value} onChange={e=>setForm(old=>({...old,attributes:old.attributes.map((row,i)=>i===index?{...row,value:e.target.value}:row)}))}/></label><Button type="button" className="self-end" variant="outline" aria-label={`删除属性 ${index+1}`} onClick={()=>setForm(old=>({...old,attributes:old.attributes.filter((_,i)=>i!==index)}))}>删除</Button></div>)}</Card>
     <Card className="overflow-hidden"><div className="space-y-2 p-5"><h2 className="font-semibold">现有 SKU 的候选价格和库存</h2><p className="text-sm text-muted-foreground">SKU 和销售规格保持固定。留空表示未填写，0 表示明确的零。</p></div>{form.variants.length?<Table className="min-w-[850px]"><TableHeader><TableRow><TableHead>固定来源 SKU</TableHead><TableHead>币种</TableHead><TableHead>候选价格</TableHead><TableHead>候选库存</TableHead></TableRow></TableHeader><TableBody>{form.variants.map((v,index)=>{const original=detail.product.variants?.[index];return <TableRow key={index}><TableCell className="max-w-80 whitespace-normal"><p className="font-medium">SKU {index+1} · {original?.sku||original?.title||"未填写编码"}</p><p className="text-xs text-muted-foreground">{(original?.attributes??[]).map(a=>`${a.name??""}：${a.value??""}`).join(" · ")}</p></TableCell><TableCell><Input aria-label={`候选币种 SKU ${index+1}`} value={v.currency} maxLength={3} className="w-24" onChange={e=>changeVariant(index,"currency",e.target.value.toUpperCase())}/></TableCell><TableCell><Input aria-label={`候选价格 SKU ${index+1}`} type="number" min={0} max={Number.MAX_SAFE_INTEGER} step="any" value={v.amount} onChange={e=>changeVariant(index,"amount",e.target.value)}/></TableCell><TableCell><Input aria-label={`候选库存 SKU ${index+1}`} type="number" min={0} max={Number.MAX_SAFE_INTEGER} step={1} value={v.stock} onChange={e=>changeVariant(index,"stock",e.target.value)}/></TableCell></TableRow>})}</TableBody></Table>:<p className="p-5 text-sm text-muted-foreground">固定来源中没有 SKU。</p>}</Card>
    </fieldset>
    <Card className="space-y-3 p-5"><p className="text-sm text-muted-foreground">未绑定店铺的候选资料 · {completionPayloadBytes(input)} / {COMPLETION_MAX_BYTES} 字节</p>{!valid?<p role="alert" className="text-sm text-destructive">请检查属性名称、重复项、金额、库存和内容长度。标题或描述超限时，请返回草稿调整。</p>:null}{conflict?<p role="alert" className="text-sm text-destructive">当前版本已变化，请重新读取后再确认保存。</p>:null}<div className="flex flex-wrap gap-3"><Button type="submit" disabled={blocked||!valid}>保存商品属性和 SKU</Button><Button type="button" variant="outline" disabled={loading||command.busy||!!command.pending} onClick={refresh}>重新读取保存版本</Button></div></Card>
   </form>}
  </div>:null}
 </ConsolePage>;
}
