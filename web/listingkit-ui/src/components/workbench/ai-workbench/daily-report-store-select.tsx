"use client";
import {useInfiniteQuery} from "@tanstack/react-query";
import {Button} from "@/components/ui/button";
import {Select} from "@/components/ui/select";
import styles from "./schedules.module.css";
import {catalogSchema,storeCenterRequest} from "@/lib/api/store-center";
import type {AIScope} from "@/lib/api/ai-workbench";

export function DailyReportStoreSelect({scope,authorizationKey,value,disabled,onChange,initialStore}:{scope:AIScope;authorizationKey:string;value:string;disabled:boolean;onChange:(id:string)=>void;initialStore?:{id:string;name:string}}) {
 const stores=useInfiniteQuery({queryKey:["daily-report-stores",scope.userId,scope.organizationId,authorizationKey],initialPageParam:1,retry:false,gcTime:0,queryFn:({signal,pageParam})=>storeCenterRequest(`catalog?page=${pageParam}&pageSize=100&status=active`,scope,catalogSchema,{signal}),getNextPageParam:(page)=>page.pagination.page*page.pagination.pageSize<page.pagination.total?page.pagination.page+1:undefined});
 const loaded=stores.data?.pages.flatMap(page=>page.items)??[];
 const retained=initialStore?.id===value&&!loaded.some(store=>store.id===value)?initialStore:undefined;
 return <div className={styles.storeField}><label className={styles.field}>店铺<Select value={value} disabled={disabled||stores.isFetching||stores.isError} onChange={e=>onChange(e.target.value)}><option value="">请选择一家店铺</option>{retained?<option value={retained.id}>{retained.name}</option>:null}{loaded.map(store=><option key={store.id} value={store.id}>{store.name} · {store.platform}</option>)}</Select></label>{stores.isPending?<p className={styles.hint} role="status">正在读取可用店铺…</p>:stores.isError?<p className={styles.error} role="alert">店铺读取失败<Button variant="ghost" onClick={()=>void stores.refetch()}>重试</Button></p>:!stores.data.pages[0].items.length?<p className={styles.hint}>当前没有可选择的店铺，请先到店铺中心添加</p>:null}{stores.hasNextPage?<Button variant="ghost" disabled={disabled||stores.isFetchingNextPage} onClick={()=>void stores.fetchNextPage()}>加载更多店铺</Button>:null}</div>;
}
