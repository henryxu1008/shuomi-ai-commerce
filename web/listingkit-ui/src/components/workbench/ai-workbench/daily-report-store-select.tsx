"use client";
import {useInfiniteQuery} from "@tanstack/react-query";
import {Button} from "@/components/ui/button";
import {catalogSchema,storeCenterRequest} from "@/lib/api/store-center";
import type {AIScope} from "@/lib/api/ai-workbench";

export function DailyReportStoreSelect({scope,authorizationKey,value,disabled,onChange}:{scope:AIScope;authorizationKey:string;value:string;disabled:boolean;onChange:(id:string)=>void}) {
 const stores=useInfiniteQuery({queryKey:["daily-report-stores",scope.userId,scope.organizationId,authorizationKey],initialPageParam:1,retry:false,gcTime:0,queryFn:({signal,pageParam})=>storeCenterRequest(`catalog?page=${pageParam}&pageSize=100&status=active`,scope,catalogSchema,{signal}),getNextPageParam:(page)=>page.pagination.page*page.pagination.pageSize<page.pagination.total?page.pagination.page+1:undefined});
 return <div><label>店铺<select value={value} disabled={disabled||stores.isFetching||stores.isError} onChange={e=>onChange(e.target.value)}><option value="">请选择一家店铺</option>{stores.data?.pages.flatMap(page=>page.items).map(store=><option key={store.id} value={store.id}>{store.name} · {store.platform}</option>)}</select></label>{stores.isPending?<p role="status">正在读取可用店铺…</p>:stores.isError?<p role="alert">店铺读取失败。<Button variant="ghost" onClick={()=>void stores.refetch()}>重试</Button></p>:!stores.data.pages[0].items.length?<p>当前没有可选择的店铺，请先到店铺中心添加。</p>:null}{stores.hasNextPage?<Button variant="ghost" disabled={disabled||stores.isFetchingNextPage} onClick={()=>void stores.fetchNextPage()}>加载更多店铺</Button>:null}</div>;
}
