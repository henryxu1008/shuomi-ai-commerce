"use client";
import {DescriptionView} from "../description-view";
import {useEffect,useMemo,useState} from "react";
import Link from "next/link";
import Image from "next/image";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {CollectionAPIError,listOwnHistory,readOwnVersion,type CollectionScope} from "@/lib/api/product-collection";
import type {OwnHistory,OwnVersionDetail} from "@/lib/contracts/product-collection";
import {Button} from "@/components/ui/button";
import {Badge} from "@/components/ui/badge";
import {Card} from "@/components/ui/card";
import {Table,TableBody,TableCell,TableHead,TableHeader,TableRow} from "@/components/ui/table";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {OwnProductContents} from "./own-product-detail-page";
import {compareOwnProducts,type OwnDiffRow} from "./own-product-diff";
import {collectionFailureText} from "./use-collection-command";

type Props={itemId:string;version?:string;from?:string;to?:string};
const savedTime=(value:string)=>new Date(value).toLocaleString("zh-CN");
export function OwnProductHistoryPage(props:Props){
 const context=useWorkbenchContext(),userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.selectionRequired||context.error||context.blockingError)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 if(!context.permissions.includes("workbench.collection.read"))return <ConsoleState kind="error" title="你没有读取商品资料的权限"/>;
 return <ScopedHistory key={`${scope.userId}:${scope.organizationId}:${props.itemId}:${props.version??""}:${props.from??""}:${props.to??""}`} {...props} scope={scope}/>;
}
function ScopedHistory({scope,itemId,version,from,to}:Props&{scope:CollectionScope}){
 const base=`/workbench/data/mine/own/${itemId}`,comparing=!!from&&!!to;
 const [history,setHistory]=useState<OwnHistory|null>(null),[details,setDetails]=useState<OwnVersionDetail[]|null>(null),[error,setError]=useState(""),[reload,setReload]=useState(0),[before,setBefore]=useState<string|undefined>(),[selected,setSelected]=useState<string[]>([]);
 useEffect(()=>{
  const controller=new AbortController();let active=true;
  const operation=version||comparing?Promise.all((version?[version]:[from!,to!]).map(v=>readOwnVersion(scope,itemId,v,controller.signal))).then(values=>{
   if(values.some((value,index)=>value.item.id!==itemId||value.item.source.kind!=="own"||value.version.version!==(version?[version]:[from!,to!])[index])||values.length===2&&values[0]!.item.source.productKey!==values[1]!.item.source.productKey)throw new CollectionAPIError("DEPENDENCY_UNAVAILABLE",502);
   if(active){setDetails(values);setError("")}
  }):listOwnHistory(scope,itemId,before,controller.signal).then(value=>{if(value.item.id!==itemId||value.item.source.kind!=="own")throw new CollectionAPIError("DEPENDENCY_UNAVAILABLE",502);if(active){setHistory(value);setError("")}});
  void operation.catch(failure=>{if(active)setError(failure instanceof CollectionAPIError?collectionFailureText[failure.code]??"暂时无法读取历史资料":"暂时无法读取历史资料")});
  return()=>{active=false;controller.abort()};
 },[scope,itemId,version,comparing,from,to,before,reload]);
 const chosen=[...selected].sort((a,b)=>BigInt(a)<BigInt(b)?-1:1);
 const title=comparing?`版本 ${from} → ${to} 差异对比`:version?`历史版本 ${version} · ${details?.[0]?.product.title??"商品资料"}`:"版本历史";
 function refresh(){setHistory(null);setDetails(null);setError("");setReload(value=>value+1)}
 return <ConsolePage title={title} description={comparing?"比较两个手工保存版本的商品资料变化。":version?"查看此版本保存时的完整资料。历史资料只读。":"查看商品每次手工保存的真实版本，选择两版查看差异。"} breadcrumbs={[{label:"我的数据",href:"/workbench/data/mine?view=own"},{label:"商品详情",href:base},{label:"版本历史",href:version||comparing?`${base}/history`:undefined},...(version?[{label:`版本 ${version}`}]:comparing?[{label:"差异对比"}]:[])]} actions={<div className="flex flex-wrap gap-3">{version||comparing?<Button asChild variant="outline"><Link href={`${base}/history`}>返回版本历史</Link></Button>:null}<Button asChild variant="outline"><Link href={base}>查看当前资料</Link></Button></div>}>
  {error?<ConsoleState kind="error" title={error}><Button variant="outline" onClick={refresh}>重试读取</Button></ConsoleState>:version||comparing?!details?<ConsoleState kind="loading" title="正在读取准确历史版本"/>:comparing?<Comparison details={details}/>:<div className="space-y-6"><Card className="flex flex-wrap items-center justify-between gap-3 p-5"><Badge variant="secondary">历史版本 {details[0]!.version.version} · 只读</Badge><p className="text-sm text-muted-foreground">保存于 {savedTime(details[0]!.version.savedAt)}</p></Card><OwnProductContents savedAt={details[0]!.version.savedAt} detail={{product:details[0]!.product,item:{...details[0]!.item,source:{...details[0]!.item.source,version:details[0]!.version.version,publicationId:details[0]!.version.publicationId}}}}/></div>:!history?<ConsoleState kind="loading" title="正在读取保存历史"/>:<Card className="overflow-hidden">
   <div className="flex flex-wrap items-center justify-between gap-4 border-b border-border p-6"><div><h2 className="text-lg font-semibold">{history.item.title||"商品保存历史"}</h2><p className="mt-1 text-sm text-muted-foreground">当前资料版本 {history.item.source.version} · 已选 {selected.length}/2</p></div>{chosen.length===2?<Button asChild><Link href={`${base}/history/compare?from=${chosen[0]}&to=${chosen[1]}`}>对比所选版本</Link></Button>:<Button disabled>选择两个版本进行对比</Button>}</div>
   {history.versions.length?<Table className="min-w-[680px]"><TableHeader><TableRow><TableHead>选择</TableHead><TableHead>保存版本</TableHead><TableHead>商品名称</TableHead><TableHead>保存时间</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody>{history.versions.map(entry=><TableRow key={entry.version}><TableCell><input className="h-4 w-4 accent-primary" type="checkbox" aria-label={`选择版本 ${entry.version}`} checked={selected.includes(entry.version)} disabled={selected.length===2&&!selected.includes(entry.version)} onChange={event=>setSelected(values=>event.target.checked?[...values,entry.version]:values.filter(value=>value!==entry.version))}/></TableCell><TableCell><span className="font-medium">版本 {entry.version}</span>{entry.version===history.item.source.version?<Badge className="ml-2" variant="secondary">当前</Badge>:null}</TableCell><TableCell className="max-w-80 whitespace-normal break-words">{entry.title}</TableCell><TableCell>{savedTime(entry.savedAt)}</TableCell><TableCell><Button asChild size="sm" variant="outline"><Link href={`${base}/history/${entry.version}`}>查看版本 {entry.version}</Link></Button></TableCell></TableRow>)}</TableBody></Table>:<p className="p-6 text-sm text-muted-foreground">此页没有更早的手工保存版本。</p>}
   <div className="flex flex-wrap justify-between gap-3 border-t border-border p-5"><Button variant="outline" onClick={()=>{setBefore(undefined);setSelected([]);refresh()}}>刷新最新历史</Button>{history.nextBefore?<Button variant="outline" onClick={()=>{setBefore(history.nextBefore);setHistory(null)}}>查看更早版本</Button>:<span className="self-center text-xs text-muted-foreground">已到最早保存版本</span>}</div>
  </Card>}
 </ConsolePage>;
}
function Comparison({details}:{details:OwnVersionDetail[]}){
 const [left,right]=details;let rows:OwnDiffRow[];
 try{rows=compareOwnProducts(left!.product,right!.product)}catch(error){return <ConsoleState kind="error" title={error instanceof Error?error.message:"历史资料无法准确比较"}/>}
 return <div className="space-y-6"><div className="grid gap-4 sm:grid-cols-2">{details.map((detail,index)=><Card key={index} className="space-y-2 p-5"><Badge variant="outline">{index===0?"比较起点":"比较终点"} · 版本 {detail.version.version}</Badge><h2 className="break-words font-semibold">{detail.product.title}</h2><p className="text-xs text-muted-foreground">保存于 {savedTime(detail.version.savedAt)}</p></Card>)}</div><Card className="overflow-hidden"><div className="border-b border-border p-6"><h2 className="text-lg font-semibold">资料变化 · {rows.length} 项</h2><p className="mt-1 text-sm text-muted-foreground">按稳定变体身份匹配 SKU，显示新增、删除和修改的资料。</p></div>{rows.length?<Table className="min-w-[720px]"><TableHeader><TableRow><TableHead>字段</TableHead><TableHead>变化</TableHead><TableHead>版本 {left!.version.version}</TableHead><TableHead>版本 {right!.version.version}</TableHead></TableRow></TableHeader><TableBody>{rows.map(row=><TableRow key={row.key}><TableCell className="w-52 whitespace-normal break-words font-medium">{row.label}</TableCell><TableCell><Badge variant={row.status==="removed"?"destructive":"secondary"}>{row.status==="added"?"新增":row.status==="removed"?"删除":"修改"}</Badge></TableCell><TableCell className="w-[35%]">{row.key===JSON.stringify(["description"])?<DescriptionView value={row.before} format={left!.product.description_format} source/>:<DiffValue value={row.before} images={row.images}/>}</TableCell><TableCell className="w-[35%]">{row.key===JSON.stringify(["description"])?<DescriptionView value={row.after} format={right!.product.description_format} source/>:<DiffValue value={row.after} images={row.images}/>}</TableCell></TableRow>)}</TableBody></Table>:<p className="p-6 text-sm text-muted-foreground">这两个版本的商品资料没有变化。</p>}</Card></div>;
}
function DiffValue({value,images}:{value:string;images?:boolean}){
 if(images&&value&&value!=="未提供")return <div className="flex flex-wrap gap-2">{value.split("\n").map((url,index)=><div key={`${url}:${index}`} className="space-y-1"><Image unoptimized src={`/api/image-proxy?url=${encodeURIComponent(url)}`} width={72} height={72} className="h-18 w-18 rounded-md border border-border object-contain" alt={`图片 ${index+1}`}/><p className="text-center text-xs text-muted-foreground">{index+1}</p></div>)}</div>;
 return <p className="max-h-64 overflow-auto whitespace-pre-wrap break-words text-sm">{value||"未填写"}</p>;
}
