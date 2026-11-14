"use client";

import { useInfiniteQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { configurationRequest } from "@/lib/api/agent-configuration";
import { basesSchema, knowledgeRequest } from "@/lib/api/knowledge";
import { templatesPageSchema } from "@/lib/contracts/agent-configuration";

type Props={scope:{userId:string;organizationId:string};authorizationKey:string;agentId:"product.content.agent"|"product.title.agent";target:string;enabled:boolean;disabled:boolean;templateId:string;templateRevision:string;knowledgeBaseId:string;onTemplate:(id:string,revision:string)=>void;onKnowledge:(id:string)=>void;onReady?:(ready:boolean)=>void};

// Reuses the existing tenant-scoped configuration and knowledge owners. A
// choice keeps its exact version; a changed or inaccessible choice never
// silently falls back to another template or knowledge base.
export function ChatContextPicker(p:Props) {
 const {userId,organizationId}=p.scope;
 const templates=useInfiniteQuery({queryKey:["chat-context-templates",userId,organizationId,p.authorizationKey,p.agentId],enabled:p.enabled,initialPageParam:"",retry:false,staleTime:0,gcTime:0,
  queryFn:({signal,pageParam})=>configurationRequest({userId,organizationId},`${p.agentId}/templates?lifecycle=ACTIVE&pageSize=100${pageParam?`&cursor=${encodeURIComponent(pageParam)}`:""}`,templatesPageSchema,{signal}),
  getNextPageParam:last=>last.nextCursor||undefined});
 const knowledge=useInfiniteQuery({queryKey:["chat-context-knowledge",userId,organizationId,p.authorizationKey],enabled:p.enabled,initialPageParam:1,retry:false,staleTime:0,gcTime:0,
  queryFn:({signal,pageParam})=>knowledgeRequest({userId,organizationId},`knowledge-bases?pageSize=100&page=${pageParam}`,basesSchema,{signal}),
  getNextPageParam:last=>last.pagination.page*last.pagination.pageSize<last.pagination.total?last.pagination.page+1:undefined});
 const items=(templates.isError||templates.isFetching?[]:templates.data?.pages.flatMap(x=>x.items)??[]).filter(t=>t.agentId===p.agentId&&t.lifecycle==="ACTIVE"&&t.targetPlatform===p.target);
 const bases=(knowledge.isError||knowledge.isFetching?[]:knowledge.data?.pages.flatMap(x=>x.items)??[]).filter(b=>b.state==="ACTIVE");
 const templateFound=!p.templateId||items.some(t=>t.templateId===p.templateId&&t.version===p.templateRevision);
 const knowledgeFound=!p.knowledgeBaseId||bases.some(b=>b.id===p.knowledgeBaseId);
 const ready=(!p.templateId||!templates.isError&&templateFound)&&(!p.knowledgeBaseId||!knowledge.isError&&knowledgeFound);
 useEffect(()=>{p.onReady?.(ready)},[p.onReady,ready]);
 return <div>
  <label>模板<Select aria-label="模板" value={p.templateId?`${p.templateId}@${p.templateRevision}`:""} disabled={p.disabled||(templates.isPending||templates.isError)&&!p.templateId} onChange={e=>{const selected=items.find(t=>`${t.templateId}@${t.version}`===e.target.value);p.onTemplate(selected?.templateId??"",selected?.version??"")}}>
   <option value="">不使用模板</option>{p.templateId&&!templateFound?<option value={`${p.templateId}@${p.templateRevision}`} disabled>此前选择的模板暂不可用</option>:null}{items.map(t=><option key={`${t.templateId}@${t.version}`} value={`${t.templateId}@${t.version}`}>{t.name}</option>)}
  </Select></label>
  {templates.isError?<p role="status">模板暂时无法读取。<Button type="button" variant="outline" disabled={p.disabled} onClick={()=>void templates.refetch()}>重新读取模板</Button></p>:null}
  {templates.hasNextPage?<Button type="button" variant="outline" disabled={p.disabled||templates.isFetchingNextPage} onClick={()=>void templates.fetchNextPage()}>查看更多模板</Button>:null}
  <label>知识库<Select aria-label="知识库" value={p.knowledgeBaseId} disabled={p.disabled||(knowledge.isPending||knowledge.isError)&&!p.knowledgeBaseId} onChange={e=>p.onKnowledge(e.target.value)}>
   <option value="">不使用知识库</option>{p.knowledgeBaseId&&!knowledgeFound?<option value={p.knowledgeBaseId} disabled>此前选择的知识库暂不可用</option>:null}{bases.map(b=><option key={b.id} value={b.id}>{b.name}</option>)}
  </Select></label>
  {knowledge.isError?<p role="status">知识库暂时无法读取。<Button type="button" variant="outline" disabled={p.disabled} onClick={()=>void knowledge.refetch()}>重新读取知识库</Button></p>:null}
  {knowledge.hasNextPage?<Button type="button" variant="outline" disabled={p.disabled||knowledge.isFetchingNextPage} onClick={()=>void knowledge.fetchNextPage()}>查看更多知识库</Button>:null}
  {!ready?<p role="status">此前的选择暂不可用，请重新选择后发送。</p>:null}
 </div>;
}
