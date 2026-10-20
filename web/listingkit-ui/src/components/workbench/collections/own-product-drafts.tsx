"use client";
import {useEffect,useState} from "react";
import Link from "next/link";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Table,TableHeader,TableHead,TableBody,TableRow,TableCell} from "@/components/ui/table";
import {listItemCompletionDrafts,type SupplyScope} from "@/lib/api/supply-chain";
import type {SupplyCompletionDraft} from "@/lib/contracts/supply-chain";
import {completionDraftHref} from "@/lib/contracts/completion-draft-navigation";
import {ConsoleState} from "../console/console-page";

type Props={scope:SupplyScope;itemId:string;canManage:boolean};
export function OwnProductDrafts(props:Props){
 return <ScopedDrafts key={`${props.scope.userId}:${props.scope.organizationId}:${props.itemId}`} {...props}/>;
}
function ScopedDrafts({scope,itemId,canManage}:Props){
 const [items,setItems]=useState<SupplyCompletionDraft[]>([]),[total,setTotal]=useState<number>(),[next,setNext]=useState<string>(),[after,setAfter]=useState<string>();
 const [loading,setLoading]=useState(true),[error,setError]=useState(false),[reload,setReload]=useState(0);
 useEffect(()=>{const controller=new AbortController();let live=true;
  void listItemCompletionDrafts(scope,itemId,{limit:20,after},controller.signal).then(page=>{if(live){setItems(page.items);setTotal(page.total);setNext(page.nextCursor)}}).catch(()=>{if(live)setError(true)}).finally(()=>{if(live)setLoading(false)});
  return()=>{live=false;controller.abort()};
 },[scope,itemId,after,reload]);
 function read(cursor?:string){setAfter(cursor);setError(false);setLoading(true);setReload(v=>v+1)}
 return <Card className="overflow-hidden" role="region" aria-label="已有待补全草稿">
  <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border p-6"><div><h2 className="text-lg font-semibold">已有待补全草稿{!loading&&!error&&total!==undefined?`（共 ${total} 份）`:""}</h2><p className="mt-1 text-sm text-muted-foreground">各批次的已保存草稿。继续编辑会读取该草稿最新保存版本，来源商品版本保持固定。</p></div><Button variant="outline" disabled={loading} onClick={()=>read()}>刷新已有草稿</Button></div>
  {loading?<ConsoleState kind="loading" title="正在读取已有草稿"/>:error?<ConsoleState kind="error" title="已有草稿读取失败">请确认当前企业的数据与供应链查看权限后重试。</ConsoleState>:!items.length?<p className="p-6 text-sm text-muted-foreground">{total===0?"此商品还没有已保存的待补全草稿。":"此页没有草稿，请返回首页刷新。"}</p>:<Table className="min-w-[760px]"><TableHeader className="bg-muted/40"><TableRow><TableHead>候选标题</TableHead><TableHead>固定来源版本</TableHead><TableHead>保存版本</TableHead><TableHead>保存时间</TableHead><TableHead>转入批次</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody>{items.map(item=><TableRow key={item.source.id}><TableCell className="max-w-64 whitespace-normal break-words">{item.content.title||"未填写候选标题"}</TableCell><TableCell>V{item.source.source.version}</TableCell><TableCell>R{item.revision}</TableCell><TableCell>{new Date(item.createdAt).toLocaleString("zh-CN")}</TableCell><TableCell><Link className="text-primary underline" title={item.source.preparationId} href={`/workbench/supply/mine/missing?preparation=${item.source.preparationId}`}>{item.source.preparationId.slice(0,8)}…</Link></TableCell><TableCell><Button asChild variant="outline" size="sm"><Link href={completionDraftHref(item.source.id,item.source.preparationId,itemId)}>{canManage?"继续编辑":"查看草稿"}</Link></Button></TableCell></TableRow>)}</TableBody></Table>}
  {!loading&&!error&&(after||next)?<div className="flex justify-end gap-2 border-t border-border p-4">{after?<Button variant="outline" size="sm" onClick={()=>read()}>草稿首页</Button>:null}{next?<Button variant="outline" size="sm" onClick={()=>read(next)}>下一页草稿</Button>:null}</div>:null}
 </Card>;
}
