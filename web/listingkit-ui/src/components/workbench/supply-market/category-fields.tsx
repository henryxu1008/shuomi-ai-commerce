"use client";

import {useInfiniteQuery} from "@tanstack/react-query";
import {Button} from "@/components/ui/button";
import {Select} from "@/components/ui/select";
import {listBrowseCategories,listManagedCategories,type MarketScope} from "@/lib/api/supply-market";
import type {MarketCategory,MarketCategoryOption,MarketCategorySelection} from "@/lib/contracts/supply-market";
import {marketError} from "./shared";

export type CategoryValue = {parentId?:string; childId?:string; assignment?:MarketCategorySelection};
type Channel = "official"|"selected";

export function useManagedCategoryLevel(scope:MarketScope,parentId?:string,enabled=true) {
 return useInfiniteQuery({
  queryKey:["supply-market",scope.userId,scope.organizationId,"category-directory",parentId],
  queryFn:({signal,pageParam})=>listManagedCategories(scope,parentId,pageParam,signal),
  initialPageParam:undefined as string|undefined,
  getNextPageParam:page=>page.nextCursor,
  enabled,
  retry:false,
 });
}

function useBrowseCategoryLevel(scope:MarketScope,channel?:Channel,parentId?:string,enabled=true) {
 return useInfiniteQuery({
  queryKey:["supply-market",scope.userId,scope.organizationId,"category-options",channel,parentId],
  queryFn:({signal,pageParam})=>listBrowseCategories(scope,channel!,parentId,pageParam,signal),
  initialPageParam:undefined as string|undefined,
  getNextPageParam:page=>page.nextCursor,
  enabled:!!channel&&enabled,
  retry:false,
 });
}

export function CategoryFields({scope,channel,value,onChange,disabled=false}:{
 scope:MarketScope; channel?:Channel; value:CategoryValue; onChange:(value:CategoryValue)=>void; disabled?:boolean;
}) {
 const assigning=!channel;
 const adminRoots=useManagedCategoryLevel(scope,undefined,assigning);
 const adminChildren=useManagedCategoryLevel(scope,value.parentId,assigning&&!!value.parentId);
 const browseRoots=useBrowseCategoryLevel(scope,channel);
 const browseChildren=useBrowseCategoryLevel(scope,channel,value.parentId,!!value.parentId);
 const roots=assigning?adminRoots:browseRoots;
 const children=assigning?adminChildren:browseChildren;
 const options=(pages:readonly {items:MarketCategoryOption[]}[]|undefined)=>{
  return pages?.flatMap(page=>page.items).filter(item=>!assigning||(item as MarketCategory).enabled)??[];
 };
 const parentOptions=options(roots.data?.pages),childOptions=value.parentId?options(children.data?.pages):[];
 const error=roots.error||(value.parentId?children.error:null);
 function selectChild(childId:string) {
  const parent=adminRoots.data?.pages.flatMap(p=>p.items).find(i=>i.id===value.parentId);
  const child=adminChildren.data?.pages.flatMap(p=>p.items).find(i=>i.id===childId);
  const assignment=assigning&&parent?.enabled&&child?.enabled
   ?{id:child.id,revision:child.revision,parentRevision:parent.revision}:undefined;
  onChange({parentId:value.parentId,childId:childId||undefined,assignment});
 }
 return <fieldset className="min-w-0 space-y-2">
  <legend className="text-sm font-medium">商品类目</legend>
  <div className="grid gap-2 sm:grid-cols-2">
   <Select aria-label="一级类目" value={value.parentId??""} disabled={disabled||roots.isPending} onChange={event=>onChange({parentId:event.target.value||undefined})}>
    <option value="">{assigning?"选择一级类目":"全部类目"}</option>
    {value.parentId&&!parentOptions.some(item=>item.id===value.parentId)?<option value={value.parentId} disabled>原选择，需重新确认</option>:null}
    {parentOptions.map(item=><option value={item.id} key={item.id}>{item.name}</option>)}
   </Select>
   <Select aria-label="二级类目" value={value.childId??""} disabled={disabled||!value.parentId||children.isPending} onChange={event=>selectChild(event.target.value)}>
    <option value="">{assigning?"选择二级类目":"全部二级类目"}</option>
    {value.childId&&!childOptions.some(item=>item.id===value.childId)?<option value={value.childId} disabled>原选择，需重新确认</option>:null}
    {childOptions.map(item=><option value={item.id} key={item.id}>{item.name}</option>)}
   </Select>
  </div>
  {roots.hasNextPage||value.parentId&&children.hasNextPage?<div className="flex flex-wrap gap-2">
   {roots.hasNextPage?<Button type="button" size="sm" variant="ghost" disabled={disabled||roots.isFetching} onClick={()=>void roots.fetchNextPage()}>加载更多一级类目</Button>:null}
   {value.parentId&&children.hasNextPage?<Button type="button" size="sm" variant="ghost" disabled={disabled||children.isFetching} onClick={()=>void children.fetchNextPage()}>加载更多二级类目</Button>:null}
  </div>:null}
  {error?<div role="alert" className="text-sm">{marketError(error)} <Button type="button" size="sm" variant="ghost" onClick={()=>void (roots.error?roots.refetch():children.refetch())}>重新读取类目</Button></div>:null}
  {assigning&&!error&&((roots.isSuccess&&!roots.hasNextPage&&!parentOptions.length)||(value.parentId&&children.isSuccess&&!children.hasNextPage&&!childOptions.length))?
   <p className="text-sm text-muted-foreground">没有可用类目。请在「商品类目」中新增或恢复类目。</p>:null}
 </fieldset>;
}
