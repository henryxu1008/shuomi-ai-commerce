"use client";
import {useEffect,useMemo,useState} from "react";
import Link from "next/link";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Badge} from "@/components/ui/badge";
import {Card} from "@/components/ui/card";
import {Table,TableBody,TableCell,TableHead,TableHeader,TableRow} from "@/components/ui/table";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {listCompletionHistory,readCompletionRevision,type SupplyScope} from "@/lib/api/supply-chain";
import type {CompletionHistory,SupplyCompletionDraft} from "@/lib/contracts/supply-chain";
import {completionDraftHref,completionHistoryHref,completionRestoreHref,completionRevision} from "@/lib/contracts/completion-draft-navigation";
import {CompletionStructuredDetails,CompletionStructuredComparison} from "./completion-structured-view";

type Props={sourceId:string;preparationId?:string;itemId?:string;revision?:number;from?:number;to?:number};
type Source=SupplyCompletionDraft["source"];
const savedTime=(value:string)=>new Date(value).toLocaleString("zh-CN");

export function CompletionHistoryPage(props:Props){
 const context=useWorkbenchContext(),userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.error||context.blockingError||context.selectionRequired)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 if(!context.permissions.includes("workbench.supply.read"))return <ConsoleState kind="unavailable" title="当前身份没有供应链查看权限"/>;
 const comparing=props.from!==undefined||props.to!==undefined;
 if(props.revision!==undefined&&(comparing||completionRevision(String(props.revision))===null)||comparing&&(completionRevision(String(props.from))===null||completionRevision(String(props.to))===null||props.from!>=props.to!))return <ConsoleState kind="error" title="比较版本参数无效"/>;
 return <ScopedHistory key={`${userId}:${organizationId}:${props.sourceId}:${props.preparationId??""}:${props.itemId??""}:${props.revision??""}:${props.from??""}:${props.to??""}`} {...props} scope={scope} canManage={context.permissions.includes("workbench.supply.manage")}/>;
}

function ScopedHistory({scope,sourceId,preparationId,itemId,revision,from,to,canManage}:Props&{scope:SupplyScope;canManage:boolean}){
 const [history,setHistory]=useState<CompletionHistory>(),[details,setDetails]=useState<SupplyCompletionDraft[]>(),[error,setError]=useState(false),[loading,setLoading]=useState(true);
 const [before,setBefore]=useState<number>(),[reload,setReload]=useState(0),[selected,setSelected]=useState<number[]>([]);
 const comparing=from!==undefined&&to!==undefined;
 useEffect(()=>{
  const controller=new AbortController();let active=true;
  const check=(source:Source)=>{if(source.id!==sourceId||preparationId&&source.preparationId!==preparationId||itemId&&(source.collectionItemId!==itemId||source.source.kind!=="own"))throw new Error("Saved source and navigation differ")};
  const operation=revision!==undefined||comparing?Promise.all((revision!==undefined?[revision]:[from!,to!]).map(r=>readCompletionRevision(scope,sourceId,r,controller.signal))).then(values=>{
   const expected=revision!==undefined?[revision]:[from!,to!];
   values.forEach((value,index)=>{check(value.source);if(value.revision!==expected[index]||value.draftId!==values[0]!.draftId||JSON.stringify(value.source)!==JSON.stringify(values[0]!.source))throw new Error("Saved revisions belong to different drafts")});
   if(active){setDetails(values);setError(false)}
  }):listCompletionHistory(scope,sourceId,{before,limit:20},controller.signal).then(value=>{check(value.source);if(active){setHistory(value);setError(false)}});
  void operation.catch(()=>{if(active)setError(true)}).finally(()=>{if(active)setLoading(false)});
  return()=>{active=false;controller.abort()};
 },[scope,sourceId,preparationId,itemId,revision,from,to,comparing,before,reload]);
 const source=!loading&&!error?(details?.[0]?.source??history?.source):undefined;
 const batch=source?.preparationId,ownItem=itemId;
 const historyHref=completionHistoryHref(sourceId,batch,ownItem);
 const refresh=()=>{setLoading(true);setReload(value=>value+1)};
 return <ConsolePage title={comparing?"草稿差异对比":revision!==undefined?`草稿保存版本 R${revision}`:"草稿保存历史"} description="查看已保存的候选标题和描述，固定来源资料保持原版本。" breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:"我的供应链",href:"/workbench/supply/mine"},{label:"待补全草稿",href:source?completionDraftHref(sourceId,source.preparationId,ownItem):undefined},{label:"保存历史",href:revision!==undefined||comparing?historyHref:undefined},...(revision!==undefined?[{label:`R${revision}`}]:comparing?[{label:"差异对比"}]:[])]} actions={source?<div className="flex flex-wrap gap-3">
  {revision!==undefined||comparing?<Button asChild variant="outline"><Link href={historyHref}>返回保存历史</Link></Button>:null}
  <Button asChild variant="outline"><Link href={completionDraftHref(sourceId,source.preparationId,ownItem)}>返回草稿</Link></Button>
  {ownItem?<Button asChild variant="outline"><Link href={`/workbench/data/mine/own/${ownItem}`}>返回商品</Link></Button>:null}
  <Button asChild variant="outline"><Link href={`/workbench/supply/mine/missing?preparation=${source.preparationId}`}>返回待补全列表</Link></Button>
 </div>:undefined}>
  {loading?<ConsoleState kind="loading" title="正在读取准确保存记录"/>:error?<ConsoleState kind="error" title="无法读取保存历史">请确认当前权限、来源和版本。<Button variant="outline" onClick={refresh}>重试读取</Button></ConsoleState>:source?<div className="space-y-5">
   <Card className="flex flex-wrap items-center gap-3 p-5"><Badge variant="secondary">固定来源 V{source.source.version}</Badge><p className="break-all text-xs text-muted-foreground">来源 {sourceId} · 转入批次 {source.preparationId}</p></Card>
   {details?comparing?<><CompletionTextComparison records={details}/><CompletionStructuredComparison records={details}/></>:<><SavedDetail record={details[0]!}/><CompletionStructuredDetails value={details[0]!.content.structured}/>{canManage?<Button asChild><Link href={completionRestoreHref(sourceId,details[0]!.revision,batch,ownItem)}>将此版内容恢复为新版本</Link></Button>:null}</>:history?<HistoryList page={history} selected={selected} setSelected={setSelected} href={view=>completionHistoryHref(sourceId,batch,ownItem,view)} earlier={()=>{setLoading(true);setBefore(history.nextBefore)}} refresh={()=>{setBefore(undefined);setSelected([]);refresh()}}/>:null}
  </div>:null}
 </ConsolePage>;
}

function HistoryList({page,selected,setSelected,href,earlier,refresh}:{page:CompletionHistory;selected:number[];setSelected:(value:number[])=>void;href:(view:{revision?:number;from?:number;to?:number})=>string;earlier:()=>void;refresh:()=>void}){
 const chosen=[...selected].sort((a,b)=>a-b);
 return <Card className="overflow-hidden">
  <div className="flex flex-wrap items-center justify-between gap-4 border-b border-border p-5"><div><h2 className="font-semibold">保存记录</h2><p className="mt-1 text-sm text-muted-foreground">{page.currentRevision?`当前已保存 R${page.currentRevision}`:"尚未保存"} · 已选 {chosen.length}/2{chosen.length?`（${chosen.map(r=>`R${r}`).join("、")}）`:""}</p></div>{chosen.length===2?<Button asChild><Link href={href({from:chosen[0],to:chosen[1]})}>对比所选版本</Link></Button>:<Button disabled>选择两个保存版本</Button>}</div>
  {page.items.length?<Table className="min-w-[640px]"><TableHeader><TableRow><TableHead>选择</TableHead><TableHead>保存版本</TableHead><TableHead>候选标题</TableHead><TableHead>保存时间</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody>{page.items.map(record=><TableRow key={record.id}><TableCell><input type="checkbox" className="h-4 w-4 accent-primary" aria-label={`选择 R${record.revision}`} checked={selected.includes(record.revision)} disabled={selected.length===2&&!selected.includes(record.revision)} onChange={event=>setSelected(event.target.checked?[...selected,record.revision]:selected.filter(value=>value!==record.revision))}/></TableCell><TableCell>R{record.revision}{record.revision===page.currentRevision?<Badge className="ml-2" variant="secondary">当前</Badge>:null}</TableCell><TableCell className="max-w-80 whitespace-normal break-words">{record.content.title||"未填写标题"}</TableCell><TableCell>{savedTime(record.createdAt)}</TableCell><TableCell><Button asChild size="sm" variant="outline"><Link href={href({revision:record.revision})}>查看 R{record.revision}</Link></Button></TableCell></TableRow>)}</TableBody></Table>:<ConsoleState kind="empty" title={page.currentRevision?"此页没有更早保存":"尚无保存历史"}>返回草稿保存后，这里会显示真实记录。</ConsoleState>}
  <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border p-5"><Button variant="outline" onClick={refresh}>刷新最新历史</Button>{page.nextBefore?<Button variant="outline" onClick={earlier}>查看更早保存</Button>:<span className="text-xs text-muted-foreground">已到最早保存记录</span>}</div>
 </Card>;
}

function SavedDetail({record}:{record:SupplyCompletionDraft}){
 return <Card className="space-y-5 p-5"><div className="flex flex-wrap items-center justify-between gap-3"><Badge variant="outline">保存版本 R{record.revision} · 只读</Badge><p className="text-xs text-muted-foreground">保存于 {savedTime(record.createdAt)}</p></div><section className="space-y-2"><h2 className="font-semibold">候选标题</h2><TextValue value={record.content.title}/></section><section className="space-y-2"><h2 className="font-semibold">候选描述</h2><TextValue value={record.content.description}/></section></Card>;
}
export function CompletionTextComparison({records,title,labels}:{records:SupplyCompletionDraft[];title?:string;labels?:[string,string]}){
 const [left,right]=records;
 const rows=[{key:"title" as const,label:"候选标题"},{key:"description" as const,label:"候选描述"}];
 return <Card className="overflow-hidden"><div className="space-y-2 border-b border-border p-5"><h2 className="font-semibold">{title??`R${left!.revision} → R${right!.revision}`}</h2><p className="text-xs text-muted-foreground">R{left!.revision}：{savedTime(left!.createdAt)} · R{right!.revision}：{savedTime(right!.createdAt)}</p><p className="text-sm text-muted-foreground">比较已保存标题和描述，保留空值、空白和换行。</p></div><Table aria-label="标题和描述差异" className="min-w-[640px]"><TableHeader><TableRow><TableHead>字段</TableHead><TableHead>变化</TableHead><TableHead>{labels?.[0]??`R${left!.revision}`}</TableHead><TableHead>{labels?.[1]??`R${right!.revision}`}</TableHead></TableRow></TableHeader><TableBody>{rows.map(({key,label})=><TableRow key={key}><TableCell className="font-medium">{label}</TableCell><TableCell><Badge variant="secondary">{left!.content[key]===right!.content[key]?"未变化":"已变化"}</Badge></TableCell><TableCell className="w-[36%]"><TextValue value={left!.content[key]}/></TableCell><TableCell className="w-[36%]"><TextValue value={right!.content[key]}/></TableCell></TableRow>)}</TableBody></Table>{rows.every(({key})=>left!.content[key]===right!.content[key])?<p className="p-5 text-sm text-muted-foreground">标题和描述没有变化。</p>:null}</Card>;
}
function TextValue({value}:{value:string}){
 return <div className="max-h-96 overflow-auto"><p className="whitespace-pre-wrap break-words text-sm">{value===""?"未填写":value}</p>{value!==""&&value.trim()===""?<p className="text-xs text-muted-foreground">仅含空白文本 · {value.length} 个字符</p>:null}</div>;
}
