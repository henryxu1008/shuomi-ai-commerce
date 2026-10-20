"use client";
import {useState} from "react";
import {useInfiniteQuery,useQueries} from "@tanstack/react-query";
import {configurationRequest,type ConfigurationScope} from "@/lib/api/agent-configuration";
import {catalogPageSchema} from "@/lib/contracts/agent-configuration";
import {readNativeRunPage} from "./native-run-records";

const policy={retry:false,staleTime:0,gcTime:0,refetchOnWindowFocus:false,refetchOnReconnect:false} as const;
export function useNativeRuns(scope:ConfigurationScope,authorizationKey:string,enabled:boolean,generation:number){
 const identity=JSON.stringify([scope,authorizationKey,generation]);
 const [paging,setPaging]=useState({identity,cursors:{} as Record<string,string[]>});
 if(paging.identity!==identity)setPaging({identity,cursors:{}});
 const positions=paging.identity===identity?paging.cursors:{};
 const catalog=useInfiniteQuery({...policy,queryKey:["task-center-native-catalog",scope.userId,scope.organizationId,authorizationKey,generation],enabled,initialPageParam:"",
  queryFn:({signal,pageParam})=>configurationRequest(scope,`market?pageSize=100${pageParam?`&cursor=${encodeURIComponent(pageParam)}`:""}`,catalogPageSchema,{signal}),
  getNextPageParam:(page,_pages,cursor)=>page.nextCursor&&page.nextCursor!==cursor?page.nextCursor:undefined,
 });
 // A failed/re-authorizing directory cannot keep advertising cached sources.
 const entries=enabled&&catalog.isSuccess&&!catalog.isFetching&&!catalog.isError?Array.from(new Map(catalog.data.pages.flatMap(p=>p.items).filter(e=>e.canReadRuns).map(e=>[e.agent.agentId,e])).values()):[];
 const requests=entries.flatMap(entry=>(positions[entry.agent.agentId]??[""]).map(cursor=>({entry,cursor})));
 const queries=useQueries({queries:requests.map(({entry,cursor})=>({...policy,queryKey:["task-center-native-runs",scope.userId,scope.organizationId,authorizationKey,generation,entry.agent.agentId,cursor],queryFn:({signal}:{signal:AbortSignal})=>readNativeRunPage(scope,entry,cursor,signal)}))});
 const sources=entries.map(entry=>{
  const pages=queries.filter((_q,index)=>requests[index].entry.agent.agentId===entry.agent.agentId);
  const error=pages.find(p=>p.isError)?.error;
  const loading=pages.some(p=>p.isPending||p.isFetching);
  const next=pages.at(-1)?.data?.nextCursor??"";
  const cursors=positions[entry.agent.agentId]??[""];
  const repeated=!!next&&cursors.includes(next);
  return {entry,error,loading,repeated,next:repeated?"":next,items:error||loading?[]:Array.from(new Map(pages.flatMap(p=>p.data?.items??[]).map(r=>[r.key,r])).values()),
   // After an access denial, all retained pages must pass the owner again.
   retry:()=>Promise.all(pages.map(p=>p.refetch())),
   more:()=>{if(next&&!repeated&&!loading&&!error)setPaging(old=>({identity,cursors:{...(old.identity===identity?old.cursors:{}),[entry.agent.agentId]:[...cursors,next]}}));},
  };
 });
 return {catalog,sources,items:sources.flatMap(s=>s.items),ready:!enabled||catalog.isSuccess&&!catalog.isFetching&&!catalog.isError&&!catalog.hasNextPage&&sources.every(s=>!s.error&&!s.loading&&!s.repeated),enabled};
}
export type NativeRuns=ReturnType<typeof useNativeRuns>;
