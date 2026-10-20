"use client";

import Link from "next/link";
import {completionDraftHref} from "@/lib/contracts/completion-draft-navigation";
import Image from "next/image";
import {useEffect,useMemo,useState} from "react";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {Table,TableHeader,TableHead,TableBody,TableRow,TableCell} from "@/components/ui/table";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {CompletionDraftPage} from "./completion-draft-page";
import {listSupplyPreparations,readSupplyPreparation,listSupplyCompletionDrafts,type SupplyScope} from "@/lib/api/supply-chain";
import {type SupplyCompletionDraft,type SupplyPreparation} from "@/lib/contracts/supply-chain";

export function CompletionDraftsPage({initialPreparation,initialSource,fullAvailable=false}:{initialPreparation?:string;initialSource?:string;fullAvailable?:boolean}){
 const context=useWorkbenchContext();const userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.error||context.blockingError||context.selectionRequired)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 if(!context.permissions.includes("workbench.supply.read"))return <ConsoleState kind="unavailable" title="当前身份没有供应链查看权限"/>;
 if(initialSource)return <CompletionDraftPage sourceId={initialSource} preparationId={initialPreparation} sheinAvailable={fullAvailable}/>;
 return <ScopedCompletionDrafts key={`${userId}:${organizationId}:${initialPreparation??""}`} scope={scope} initialPreparation={initialPreparation} fullAvailable={fullAvailable}/>;
}
function ScopedCompletionDrafts({scope,initialPreparation,fullAvailable}:{scope:SupplyScope;initialPreparation?:string;fullAvailable:boolean}){
 const [batches,setBatches]=useState<SupplyPreparation[]>([]),[batchAfter,setBatchAfter]=useState<string>(),[nextBatch,setNextBatch]=useState<string>();
 const [selected,setSelected]=useState(initialPreparation??"");
 const [items,setItems]=useState<SupplyCompletionDraft[]>([]),[total,setTotal]=useState(0),[next,setNext]=useState<string>(),[after,setAfter]=useState<string>();
 const [search,setSearch]=useState(""),[keyword,setKeyword]=useState(""),[reload,setReload]=useState(0),[loading,setLoading]=useState(true),[error,setError]=useState(false);
 useEffect(()=>{const c=new AbortController();let live=true;
  void listSupplyPreparations(scope,{limit:100,after:batchAfter},c.signal).then(async page=>{
   let values=page.items;
   if(!batchAfter&&initialPreparation&&!values.some(v=>v.id===initialPreparation))values=[await readSupplyPreparation(scope,initialPreparation,c.signal),...values];
   if(live){setBatches(values);setNextBatch(page.nextCursor);setSelected(v=>v||values[0]?.id||"");if(!values.length)setLoading(false)}
  }).catch(()=>{if(live){setError(true);setLoading(false)}});return()=>{live=false;c.abort()};
 },[scope,initialPreparation,batchAfter]);
 useEffect(()=>{const c=new AbortController();let live=true;
  if(selected)void listSupplyCompletionDrafts(scope,selected,{limit:20,after,keyword},c.signal).then(page=>{if(live){setItems(page.items);setTotal(page.total);setNext(page.nextCursor)}}).catch(()=>{if(live)setError(true)}).finally(()=>{if(live)setLoading(false)});
  return()=>{live=false;c.abort()};
 },[scope,selected,after,keyword,reload]);
 const waitingQuery=new URLSearchParams({...selected?{preparation:selected}:{},...fullAvailable?{mode:"local"}:{}});
 const waiting=`/workbench/supply/mine/waiting${waitingQuery.size?`?${waitingQuery}`:""}`;
 return <ConsolePage title="待补全" description="补充商品资料，保存后继续编辑；配置店铺后再按平台要求适配。" breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:"我的供应链",href:"/workbench/supply/mine"},{label:"待补全"}]} actions={<><Button asChild variant="outline"><Link href={waiting}>从待适配创建草稿</Link></Button><Button variant="outline" disabled={loading||!selected} onClick={()=>{setLoading(true);setError(false);setReload(v=>v+1)}}>刷新</Button></>}>
  <div className="mb-5 flex items-start gap-3 rounded-lg border border-info-foreground bg-info p-4 text-sm text-info-foreground"><Image src="/supply-completion-document.svg" width={24} height={24} alt=""/><div><p className="font-medium">未绑定店铺，平台规则待配置</p><p className="mt-1">{selected?`当前批次已保存 ${total} 份待补全草稿。`:"选择已转入的商品批次查看草稿。"} 标题和描述独立保存，原始商品资料继续保留。</p></div></div>
  {fullAvailable?<p className="mb-4 text-sm"><Link href={`/workbench/supply/mine/missing?${new URLSearchParams({...selected?{preparation:selected}:{},mode:"platform"})}`} className="text-primary underline">查看已绑定店铺的待补全商品</Link></p>:null}
  <form className="mb-5 flex flex-wrap items-end gap-3 rounded-xl border bg-card p-4" onSubmit={e=>{e.preventDefault();setAfter(undefined);setKeyword(search);setError(false);setLoading(true);setReload(v=>v+1)}}>
   <label className="grid min-w-52 flex-1 gap-2 text-sm">搜索商品<Input value={search} onChange={e=>setSearch(e.target.value)} placeholder="输入候选标题或原商品编号"/></label>
   <label className="grid min-w-44 flex-1 gap-2 text-sm">商品批次<Select value={selected} onChange={e=>{setSelected(e.target.value);setItems([]);setTotal(0);setAfter(undefined);setError(false);setLoading(!!e.target.value)}}><option value="">请选择批次</option>{batches.map(b=><option key={b.id} value={b.id}>{b.name||"未命名批次"}</option>)}</Select></label>
   <label className="grid min-w-36 gap-2 text-sm">目标店铺<Select disabled value="unbound"><option value="unbound">未绑定店铺</option></Select></label>
   <label className="grid min-w-36 gap-2 text-sm">补全状态<Select disabled value="pending"><option value="pending">待补全</option></Select></label>
   <Button type="button" variant="outline" onClick={()=>{setSearch("");setKeyword("");setAfter(undefined);setError(false);setLoading(!!selected);setReload(v=>v+1)}}>重置</Button><Button type="submit" disabled={!selected}>筛选</Button>
  </form>
  {error?<ConsoleState kind="error" title="草稿读取失败">请确认当前企业的数据与供应链权限后重试。</ConsoleState>:loading?<ConsoleState kind="loading" title="正在读取待补全草稿"/>:<Card className="overflow-hidden"><div className="flex items-center justify-between border-b px-5 py-4"><h2 className="font-semibold">商品列表（共 {total} 个）</h2><span className="text-sm text-muted-foreground">保存版本可继续编辑</span></div>
   {!items.length?<div className="space-y-4 p-10 text-center text-sm text-muted-foreground"><p>当前批次还没有保存待补全草稿。</p><Button asChild variant="outline"><Link href={waiting}>选择原始商品创建草稿</Link></Button></div>:<Table className="min-w-[960px]"><TableHeader className="bg-muted/40"><TableRow><TableHead>商品</TableHead><TableHead>来源</TableHead><TableHead>商品批次</TableHead><TableHead>目标店铺</TableHead><TableHead>缺失字段</TableHead><TableHead>补全状态</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody>{items.map(item=><TableRow key={item.draftId}><TableCell><div className="font-medium">{item.content.title||"未填写候选标题"}</div><p className="mt-1 max-w-60 break-all text-xs text-muted-foreground">{item.source.source.productKey}</p></TableCell><TableCell>{({own:"自有商品",acquisition:"采集商品",market:"供应商品",sds_finished:"定制成品"} as Record<string,string>)[item.source.source.kind]??item.source.source.kind}<p className="text-xs text-muted-foreground">原始版本 {item.source.source.version}</p></TableCell><TableCell>{batches.find(b=>b.id===item.source.preparationId)?.name??item.source.preparationId}</TableCell><TableCell>未绑定</TableCell><TableCell><span title="官方规则尚未配置，无法判断必填项">待平台规则确定</span></TableCell><TableCell><span className="rounded-full bg-warning px-3 py-1 text-xs text-warning-foreground">待补全</span><p className="mt-2 text-xs text-muted-foreground">已保存 R{item.revision}</p></TableCell><TableCell><Button asChild variant="ghost" size="sm"><Link href={completionDraftHref(item.source.id,item.source.preparationId)}>查看 / 编辑</Link></Button></TableCell></TableRow>)}</TableBody></Table>}
   <div className="flex justify-end gap-2 border-t p-4">{after?<Button variant="outline" size="sm" onClick={()=>{setAfter(undefined);setLoading(true);setError(false)}}>首页</Button>:null}{next?<Button variant="outline" size="sm" onClick={()=>{setAfter(next);setLoading(true);setError(false)}}>下一页商品</Button>:null}</div>
  </Card>}
  {nextBatch?<Button className="mt-4" variant="outline" onClick={()=>{setBatchAfter(nextBatch);setAfter(undefined);setNext(undefined);setTotal(0);setSelected("");setItems([]);setLoading(true);setError(false)}}>下一页批次</Button>:null}
 </ConsolePage>;
}
