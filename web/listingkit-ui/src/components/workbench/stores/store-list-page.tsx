"use client";
import {useState} from "react";
import {useQuery,useQueryClient} from "@tanstack/react-query";
import {usePathname,useRouter,useSearchParams} from "next/navigation";
import Link from "next/link";
import {AlertTriangle,ArrowUpRight,CheckCircle2,ChevronLeft,ChevronRight,FolderPlus,MoreHorizontal,Plus,RefreshCw,Search,SlidersHorizontal,Users} from "lucide-react";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {ConsolePage,ConsoleState} from "@/components/workbench/console/console-page";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {useStoreCatalog} from "@/lib/query/use-store-catalog";
import {catalogFiltersSchema,catalogSearch,type CatalogFilters,type CatalogItem,type StoreCenterScope} from "@/lib/api/store-center";
import {getMember} from "@/lib/api/members";
import {canCreateWorkbenchStore,canUpdateWorkbenchStore} from "@/lib/workbench/permissions";
import {findConsoleRoute} from "@/lib/workbench/console-navigation";
import {StoreLifecycleActions} from "./store-lifecycle-actions";
import {StoreCatalogDialogs,type CatalogDialog} from "./store-catalog-dialogs";
import styles from "./store-catalog.module.css";

export function StoreListPage(){
 const context=useWorkbenchContext(),org=context.effectiveOrganization?.id??"",user=context.user?.id??"";
 if(!org || !user || context.isSwitching)return <ConsoleState kind="loading" title="正在加载企业权限…"/>;
 return <ScopedStoreList key={JSON.stringify([org,user,context.permissions,context.roles])} scope={{organizationId:org,userId:user}}/>;
}
function ScopedStoreList({scope}:{scope:StoreCenterScope}){
 const router=useRouter(),pathname=usePathname()||"/workbench/stores",params=useSearchParams(),context=useWorkbenchContext(),queryClient=useQueryClient();
 const raw:Record<string,unknown>={page:1,pageSize:20};
 for(const key of ["page","pageSize","q","platform","region","applicationType","groupId","attention","status"]){const value=params?.getAll(key).length===1?params.get(key):null;if(params && params.getAll(key).length>1)raw.invalid=true;if(value!==null)raw[key]=key==="page" || key==="pageSize"?Number(value):value}
 if(params)for(const key of params.keys()){if(!["page","pageSize","q","platform","region","applicationType","groupId","attention","status"].includes(key))raw.invalid=true;}
 const parsed=catalogFiltersSchema.safeParse(raw),filters:CatalogFilters=parsed.success?parsed.data:{page:1,pageSize:20};
 const query=useStoreCatalog(filters,parsed.success),data=parsed.success&&!query.isError&&!query.isPending?query.data:undefined,filterIdentity=catalogSearch(filters);
 const [draft,setDraft]=useState<CatalogFilters>(filters),[draftKey,setDraftKey]=useState(filterIdentity),[selected,setSelected]=useState<CatalogItem[]>([]),[dialog,setDialog]=useState<CatalogDialog|null>(null);
 if(draftKey!==filterIdentity){setDraft(filters);setDraftKey(filterIdentity);setSelected([])}
 const scopeFailed=query.isError && ["PERMISSION_DENIED","STORE_NOT_FOUND","ORGANIZATION_CONTEXT_CHANGED","ORGANIZATION_ACCESS_REVOKED","ORGANIZATION_ACCESS_DENIED","IDENTITY_CONTEXT_CHANGED","AUTHENTICATION_REQUIRED","ORGANIZATION_SUSPENDED"].includes((query.error as {code?:string}|null)?.code??"");
 const missing=(id:string)=>Boolean(data&&!data.items.some(item=>item.id===id));
 if(selected.length&&(scopeFailed||selected.some(item=>missing(item.id))))setSelected([]);
 if(dialog && (scopeFailed || "item" in dialog && missing(dialog.item.id) || "items" in dialog && dialog.items.some(item=>missing(item.id))))setDialog(null);
 const canWrite=canUpdateWorkbenchStore(context.permissions),canCreate=canCreateWorkbenchStore(context.permissions);
 const update=(next:CatalogFilters)=>router.push(`${pathname}?${catalogSearch(next)}`);
 const reset=()=>{const next={page:1,pageSize:filters.pageSize};setDraft(next);update(next)};
 const retry=async()=>{if(["ORGANIZATION_CONTEXT_CHANGED","ORGANIZATION_ACCESS_REVOKED","ORGANIZATION_ACCESS_DENIED","IDENTITY_CONTEXT_CHANGED"].includes((query.error as {code?:string}|null)?.code??"")){await context.retry();return;}await query.refetch()};
 const changed=async()=>{await queryClient.invalidateQueries({queryKey:["store-catalog",scope.organizationId,scope.userId]});await queryClient.invalidateQueries({queryKey:["workbench",scope.organizationId,"stores"]})};
 const check=(item:CatalogItem,on:boolean)=>setSelected(current=>on?[...current.filter(value=>value.id!==item.id),item].slice(0,100):current.filter(value=>value.id!==item.id));
 const allSelected=Boolean(data?.items.length && data.items.every(item=>selected.some(value=>value.id===item.id))),errorCode=(query.error as {code?:string}|null)?.code,metrics=data?.metrics;
 const metricRows=[{label:"店铺总数",value:metrics?.total,hint:"当前授权范围"},{label:"正常店铺",value:metrics?.normal,hint:"授权及数据正常"},{label:"需要处理",value:metrics?.attention,hint:"授权、同步或库存异常"},{label:"数据同步正常",value:metrics?.syncHealthy,hint:"商品/订单遍历完成"}];
 return <ConsolePage className={styles.page} title="我的店铺" breadcrumbs={findConsoleRoute("/workbench/stores")?.trail} description="统一管理店铺授权、分组与数据同步，让运营更清晰。" actions={canCreate?<Button className={styles.bind} onClick={()=>setDialog({kind:"bind"})}><Plus size={16}/>绑定新店铺</Button>:undefined}>
  <div className={styles.metrics}>{metricRows.map(({label,value,hint})=><div className={styles.metric} key={label}><div><span>{label}</span><strong>{value??"—"}</strong></div><span className={styles.metricNote}>{hint}</span></div>)}</div>
  {metrics && !metrics.complete?<div className={styles.notice} role="status"><AlertTriangle size={16}/><span>{metrics.unknown} 家店铺状态尚未完整确认。授权、同步和库存缺项会显示未知。</span><Button variant="ghost" size="sm" onClick={()=>void query.refetch()}>刷新状态</Button></div>:metrics?.attention?<div className={styles.notice}><AlertTriangle size={16}/><span>有 {metrics.attention} 家店铺需要处理，请查看具体授权或同步原因。</span><Button variant="ghost" size="sm" onClick={()=>update({...filters,page:1,attention:"attention"})}>立即处理</Button></div>:<div className={styles.coverage}><CheckCircle2 size={14}/>{metrics?"当前筛选范围内暂无已确认异常":"正在读取当前店铺状态"}</div>}
  <form className={styles.filters} aria-label="店铺筛选" onSubmit={event=>{event.preventDefault();update({...draft,page:1,q:draft.q?.trim()||undefined})}}>
   <label className={styles.search}><Search size={15}/><Input type="search" aria-label="搜索店铺" placeholder="搜索店铺名称 / ID" value={draft.q??""} onChange={event=>setDraft({...draft,q:event.target.value})}/></label>
   <Select aria-label="平台" value={draft.platform??""} onChange={event=>setDraft({...draft,platform:event.target.value?"shein":undefined})}><option value="">全部平台</option><option value="shein">SHEIN</option></Select>
   <Select aria-label="站点" value={draft.region??""} onChange={event=>setDraft({...draft,region:event.target.value||undefined})}><option value="">全部站点</option>{data?.facets.regions.map(region=><option key={region} value={region}>{region}</option>)}</Select>
   <Select aria-label="店铺类型" value={draft.applicationType??""} onChange={event=>setDraft({...draft,applicationType:(event.target.value||undefined) as CatalogFilters["applicationType"]})}><option value="">全部类型</option><option value="self_operated">自运营</option><option value="semi_managed">半托管</option><option value="fully_managed">全托管</option></Select>
   <Select aria-label="店铺状态" value={draft.attention??""} onChange={event=>setDraft({...draft,attention:(event.target.value||undefined) as CatalogFilters["attention"]})}><option value="">全部状态</option><option value="normal">正常</option><option value="attention">需要处理</option><option value="unknown">待确认</option></Select>
   <Select aria-label="分组" value={draft.groupId??""} onChange={event=>setDraft({...draft,groupId:event.target.value||undefined})}><option value="">全部分组</option><option value="ungrouped">未分组</option>{data?.facets.groups.map(group=><option key={group.id} value={group.id}>{group.name}</option>)}</Select>
   <Button type="button" variant="outline" onClick={reset}>重置</Button><Button type="submit"><SlidersHorizontal size={14}/>筛选</Button>
  </form>
  <div className={styles.listHeading}><div><h2>店铺列表</h2><span>共 {data?.pagination.total??"—"} 家店铺</span></div><div>{canWrite && data?.items.length?<label><input type="checkbox" checked={allSelected} onChange={event=>setSelected(event.target.checked?data.items:[])}/>选择本页</label>:null}<Button variant="outline" size="sm" disabled={!data} onClick={()=>setDialog({kind:"groups"})}><FolderPlus size={14}/>分组管理</Button></div></div>
  {selected.length?<div className={styles.bulk}><span>已选择 {selected.length} 家店铺</span><Button size="sm" variant="outline" onClick={()=>setDialog({kind:"assign",items:selected})}>加入分组</Button><button onClick={()=>setSelected([])}>取消选择</button></div>:null}
  {!parsed.success?<ConsoleState kind="error" title="筛选参数无效"><Button onClick={reset} variant="outline">重置筛选</Button></ConsoleState>:query.isPending?<ConsoleState kind="loading" title="正在加载店铺…"/>:query.isError?<ConsoleState kind="error" title={errorCode==="CATALOG_SUMMARY_UNAVAILABLE"?"当前摘要不足，无法完整应用此状态或类型筛选":errorCode==="PERMISSION_DENIED"?"没有查看当前企业店铺的权限":errorCode?.startsWith("ORGANIZATION_")?"当前企业访问已不可用":"店铺服务暂时不可用"}><p>请刷新重试，或重置筛选查看基础店铺资料。</p><Button onClick={()=>void retry()} variant="outline">重试</Button><Button onClick={reset} variant="ghost">重置筛选</Button></ConsoleState>:!data?.items.length?<ConsoleState kind="empty" title={Object.keys(filters).length>2?"没有符合筛选条件的店铺":"还没有店铺"}><p>绑定官方店铺后，在这里管理授权、分组与同步。</p></ConsoleState>:<ul className={styles.list} aria-label="我的店铺列表">{data.items.map(item=><li key={item.id} className={styles.card}>
   <div className={styles.cardHeader}><div className={styles.identity}>{canWrite?<input aria-label={`选择${item.name}`} type="checkbox" checked={selected.some(value=>value.id===item.id)} onChange={event=>check(item,event.target.checked)}/>:null}<span className={styles.avatar} aria-hidden="true">S</span><div><h3>{item.name}</h3><p>店铺 ID：{item.externalStoreId||item.id}</p></div></div><div className={styles.tags}><span>{typeLabel(item.signals.applicationType)}</span><span>SHEIN · {item.region}</span><span>{data.facets.groups.find(group=>group.id===item.profile.groupId)?.name??"未分组"}</span></div><div className={styles.states}><span data-tone={item.connectionStatus==="connected"?"good":"unknown"}>{connectionLabel(item.connectionStatus)}</span><span data-tone={item.signals.syncHealthy?"good":item.attention==="attention"?"warning":"unknown"}>{item.signals.syncHealthy?"同步正常":syncLabel(item.signals.productSync,item.signals.orderSync)}</span>{canWrite?<details className={styles.more}><summary aria-label={`${item.name}更多操作`}><MoreHorizontal size={18}/></summary><div><button onClick={()=>setDialog({kind:"rename",item})}>修改名称</button><button onClick={()=>setDialog({kind:"profile",item})}>调整负责人 / 设置预警</button><button onClick={()=>setDialog({kind:"assign",items:[item]})}>修改分组</button><Link href={`/workbench/stores/${item.id}#authorization`}>重新授权</Link><StoreLifecycleActions store={item} onDeleted={()=>void changed()}/></div></details>:null}</div></div>
   <div className={styles.cardBody}><div className={styles.facts}><Fact label="在线商品（SKC）" value={item.signals.products.value} reason={item.signals.products.reason}/><Fact label="待处理订单" value={item.signals.orders.value} reason={item.signals.orders.reason}/><Link href={`/workbench/store-products?storeId=${item.id}`}><Fact label="库存预警（SKU/仓库）" value={item.signals.inventory.value} reason={item.signals.inventory.reason} unknown={item.signals.inventory.unknown} warning={Boolean(item.signals.inventory.value)}/></Link><div><span>负责人</span><strong><Users size={12}/><ResponsibleName id={item.profile.responsibleMemberId} scope={scope}/></strong></div><div><span>服务状态</span><strong>{item.serviceStatus==="active"?"服务有效":item.serviceStatus==="expired"?"服务已到期":item.serviceStatus==="suspended"?"服务暂停":item.serviceStatus==="pending_activation"?"待激活":"—"}</strong></div></div>
   <div className={styles.cardFooter}><div><time dateTime={item.signals.observedAt??undefined}>最近同步：{item.signals.observedAt?new Date(item.signals.observedAt).toLocaleString("zh-CN"):"尚未取得"}</time>{item.profile.lowStockThreshold!==null?<span>库存 ≤ {item.profile.lowStockThreshold} 件时预警</span>:null}{item.recordStatus!=="active"?<span>{item.recordStatus==="disabled"?"记录已停用":"记录删除中"}</span>:null}</div><div><Link className={styles.outlineLink} href={`/workbench/stores/${item.id}`}>管理</Link><Button size="sm" variant="outline" onClick={()=>setDialog({kind:"sync",item})}><RefreshCw size={13}/>立即同步</Button><Link className={styles.enter} href={`/workbench/store-products?storeId=${item.id}`}>进入店铺<ArrowUpRight size={13}/></Link></div></div></div>
  </li>)}</ul>}
  {data?<div className={styles.pagination}><span>第 {data.pagination.page} 页，共 {data.pagination.total} 家店铺</span><div><Select aria-label="每页店铺数" value={filters.pageSize} onChange={event=>update({...filters,page:1,pageSize:Number(event.target.value)})}>{[10,20,50,100].map(size=><option key={size} value={size}>{size} 条 / 页</option>)}</Select><Button size="sm" variant="outline" aria-label="上一页" disabled={data.pagination.page<=1} onClick={()=>update({...filters,page:filters.page-1})}><ChevronLeft size={14}/></Button><span>{filters.page}</span><Button size="sm" variant="outline" aria-label="下一页" disabled={data.pagination.page*data.pagination.pageSize>=data.pagination.total} onClick={()=>update({...filters,page:filters.page+1})}><ChevronRight size={14}/></Button></div></div>:null}
  {dialog?<StoreCatalogDialogs key={JSON.stringify([dialog.kind,"item"in dialog?dialog.item.id:""])} dialog={dialog} scope={scope} groups={data?.facets.groups??[]} canManageGroups={data?.facets.canManageGroups??false} onClose={()=>setDialog(null)} onChanged={changed}/>:null}
 </ConsolePage>;
}
function Fact({label,value,reason,unknown=0,warning=false}:{label:string;value:number|null;reason:string;unknown?:number;warning?:boolean}){return <div><span>{label}</span><strong data-warning={warning||undefined}>{value??(reason==="threshold_not_set"?"未设置":"未知")}</strong>{reason && reason!=="threshold_not_set"?<small>{unknown>0?`另 ${unknown} 项库存未知`:reason==="inventory_unknown"?"部分库存缺失":reason==="sync_incomplete"?"同步范围未完整":"暂无完整观察"}</small>:null}</div>}
function ResponsibleName({id,scope}:{id:string|null;scope:StoreCenterScope}){
 const context=useWorkbenchContext();const query=useQuery({queryKey:["store-responsible",scope.organizationId,scope.userId,id,context.permissions],queryFn:({signal})=>getMember({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId,signal},id!),enabled:Boolean(id && context.permissions.includes("workbench.organization_member.read")),retry:false,gcTime:0});
 if(!id)return <>未设置</>;const member=query.data?.items[0];if(member)return <>{member.displayName||id}{member.state!=="active"?"（已不可用）":""}</>;
 return <span title={id}>{(query.error as {status?:number}|null)?.status===404?"成员已不可用":id}</span>;
}
function typeLabel(type:string){return type==="semi_managed"?"半托管":type==="fully_managed"?"全托管":type==="self_operated"?"自运营":"类型未确认"}
function connectionLabel(status:string){return status==="connected"?"授权正常":status==="expired"?"授权已过期":status==="disconnected"?"未授权":"授权待确认"}
function syncLabel(products:string,orders:string){const statuses=[products,orders];return statuses.includes("failed")?"同步失败":statuses.includes("partial")?"部分同步":statuses.includes("suspended")?"同步暂停":statuses.includes("running")||statuses.includes("pending")?"同步中":"同步待确认"}
