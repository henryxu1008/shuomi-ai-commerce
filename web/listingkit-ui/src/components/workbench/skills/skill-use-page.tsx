"use client";
import {useEffect,useMemo,useState} from "react";
import Link from "next/link";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {listOwnProducts,type CollectionScope} from "@/lib/api/product-collection";
import type {CollectionItem} from "@/lib/contracts/product-collection";
import {skillLaunchHref,skillLaunchMatches,type SkillLaunch,type SkillLaunchInput} from "@/lib/contracts/skill-launch";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {SkillLaunchError,SkillLaunchSummary,useLaunchVersion} from "./skill-launch";

export function SkillUsePage({launch,titleAvailable,imageAvailable}:{launch:SkillLaunchInput;titleAvailable:boolean;imageAvailable:boolean}){
 const context=useWorkbenchContext(),userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.error||context.blockingError||context.selectionRequired||!skillLaunchMatches(launch,scope))return <SkillLaunchError/>;
 const image=launch.target==="product-image",required=image?["workbench.collection.read","workbench.supply.read","workbench.supply.manage","listingkit.image_agent.read","listingkit.image_agent.write"]:["workbench.collection.read","local_agent.write","workbench.agent.use"];
 if(!(image?imageAvailable:titleAvailable))return <ConsoleState kind="unavailable" title="当前环境尚未开放此商品处理入口"/>;
 if(!required.every(p=>context.permissions.includes(p)))return <ConsoleState kind="unavailable" title="当前身份没有此技能所需的商品处理权限"/>;
 return <ScopedUse key={JSON.stringify([scope,launch,context.permissions])} launch={launch} scope={scope}/>;
}
function ScopedUse({launch,scope}:{launch:SkillLaunch;scope:CollectionScope}){
 const version=useLaunchVersion(launch,scope),[items,setItems]=useState<CollectionItem[]>([]),[next,setNext]=useState<string>(),[search,setSearch]=useState("");
 const [query,setQuery]=useState<{keyword:string;after?:string;nonce:number}>({keyword:"",nonce:0}),[read,setRead]=useState<{key:string;error?:boolean}>();
 const key=JSON.stringify(query),loading=read?.key!==key;
 useEffect(()=>{if(!version.ready)return;const controller=new AbortController();void listOwnProducts(scope,{keyword:query.keyword,after:query.after,limit:20},controller.signal).then(page=>{if(!controller.signal.aborted){setItems(page.items.filter(item=>item.source.kind==="own"));setNext(page.nextCursor);setRead({key})}}).catch(()=>{if(!controller.signal.aborted)setRead({key,error:true})});return()=>controller.abort()},[version.ready,scope,key,query.keyword,query.after]);
 return <ConsolePage title="选择已有商品" description="选择当前企业已保存的自有商品，继续使用所选准确技能版本。" breadcrumbs={[{label:"AI工作台",href:"/workbench/ai"},{label:"技能库",href:"/workbench/agents/skills"},{label:"官方模板",href:"/workbench/agents/skills/official"},{label:"选择已有商品"}]} actions={<Button asChild variant="outline"><Link href="/workbench/agents/skills/official">返回官方模板</Link></Button>}>
  <div className="space-y-5"><SkillLaunchSummary launch={launch} version={version.version}/>
  {version.error?<ConsoleState kind="unavailable" title={version.error}/>:!version.ready?<ConsoleState kind="loading" title="正在核实准确技能版本"/>:<Card className="space-y-4 p-5">
   <form className="flex flex-wrap items-end gap-3" onSubmit={event=>{event.preventDefault();setQuery({keyword:search.trim(),nonce:query.nonce+1})}}><label className="grid flex-1 gap-2 text-sm">商品名称<Input value={search} maxLength={200} onChange={event=>setSearch(event.target.value)} placeholder="搜索已有商品"/></label><Button type="submit" variant="outline">搜索</Button></form>
   <p className="text-sm text-muted-foreground">{launch.target==="product-image"?"选择后明确确认单件转入，再直接勾选已有原图，无需重新采集或填写草稿。":"选择后进入原有标题优化页，确认时已带入本次准确技能版本。"}</p>
   {loading?<p role="status">正在读取已有商品…</p>:read?.error?<div role="alert"><p>暂时无法读取已有商品，请重读。</p><Button variant="outline" onClick={()=>setQuery({...query,nonce:query.nonce+1})}>重新读取商品</Button></div>:items.length?<ul className="divide-y divide-border">{items.map(item=><li key={item.id} className="flex flex-wrap items-center justify-between gap-4 py-4"><div><p className="font-medium">{item.title||"未命名商品"}</p><p className="text-xs text-muted-foreground">原始资料版本 {item.source.version}</p></div><Button asChild variant="outline"><Link prefetch={false} href={skillLaunchHref(`/workbench/data/mine/own/${item.id}/${launch.target==="product-image"?"prepare":"optimize"}`,launch)}>选择此商品</Link></Button></li>)}</ul>:<p className="text-sm text-muted-foreground">当前没有符合条件的已有商品。</p>}
   {!loading&&!read?.error&&next?<Button variant="outline" onClick={()=>setQuery({...query,after:next})}>下一页</Button>:null}
  </Card>}</div>
 </ConsolePage>;
}
