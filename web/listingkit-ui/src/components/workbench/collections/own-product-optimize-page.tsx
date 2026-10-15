"use client";

import {useEffect,useMemo,useState} from "react";
import Link from "next/link";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {readOwnContentSource,ProductAgentError} from "@/lib/api/product-agent";
import {Button} from "@/components/ui/button";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {ProductAgentPanel} from "../acquisition/product-agent-panel";

export function OwnProductOptimizePage({itemId,enabled,knowledgeAvailable=false}:{itemId:string;enabled:boolean;knowledgeAvailable?:boolean}){
 const context=useWorkbenchContext();
 const userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.selectionRequired||context.error||context.blockingError)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 return <ScopedOptimize key={`${scope.userId}:${scope.organizationId}:${itemId}`} itemId={itemId} scope={scope} enabled={enabled} knowledgeAvailable={knowledgeAvailable}/>;
}
function ScopedOptimize({itemId,scope,enabled,knowledgeAvailable}:{itemId:string;scope:{userId:string;organizationId:string};enabled:boolean;knowledgeAvailable:boolean}){
 const [source,setSource]=useState<Awaited<ReturnType<typeof readOwnContentSource>>|null>(null),[failure,setFailure]=useState(""),[reload,setReload]=useState(0);
 useEffect(()=>{if(!enabled)return;const controller=new AbortController();void readOwnContentSource(itemId,scope,controller.signal).then(value=>{if(!controller.signal.aborted){setSource(value);setFailure("")}}).catch(error=>{if(!controller.signal.aborted)setFailure(error instanceof ProductAgentError?error.code:"DEPENDENCY_UNAVAILABLE")});return()=>controller.abort()},[itemId,scope,enabled,reload]);
 return <ConsolePage title="优化自有商品" description="根据已保存资料优化标题，审核并采用后可重新检查优选申请资格。" breadcrumbs={[{label:"我的数据",href:"/workbench/data/mine?view=own"},{label:"自有商品资料",href:`/workbench/data/mine/own/${itemId}`},{label:"优化商品内容"}]} actions={<div className="flex gap-3"><Button asChild variant="outline"><Link href={`/workbench/data/mine/own/${itemId}`}>返回商品资料</Link></Button><Button asChild variant="outline"><Link href="/workbench/supply/applications/new">检查优选资格</Link></Button></div>}>
 {!enabled?<ConsoleState kind="error" title="当前环境尚未开放内容优化"/>:failure?<ConsoleState kind="error" title={failure==="AGENT_CONFLICT"?"商品来源已变化，请重新读取后确认":"当前商品或内容优化依赖不可用"}><Button variant="outline" onClick={()=>setReload(value=>value+1)}>重新读取商品来源</Button></ConsoleState>:!source?<ConsoleState kind="loading" title="正在读取自有商品来源"/>:<div className="space-y-5"><div><h2 className="font-semibold">{source.title}</h2><p className="text-sm text-muted-foreground">已保存版本 {source.originalVersion} · 本次可优化版本 {source.catalogVersion}。生成建议后需在审核页采用，才能改变商品内容。</p></div><ProductAgentPanel enabled knowledgeAvailable={knowledgeAvailable} operationId={itemId} productKey={source.productKey} catalogVersion={source.catalogVersion} ownSource={{catalogVersion:source.catalogVersion,publicationId:source.publicationId}}/></div>}
 </ConsolePage>;
}
