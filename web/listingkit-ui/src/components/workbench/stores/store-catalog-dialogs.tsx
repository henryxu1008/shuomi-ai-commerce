"use client";
import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {z} from "zod";
import {Plus} from "lucide-react";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {ResourceDialog} from "@/components/workbench/resources/resource-dialog";
import {getMembers} from "@/lib/api/members";
import {getWorkbenchStore,type WorkbenchStore} from "@/lib/api/workbench-stores";
import {groupsSchema,metadataCommandSchema,metadataResultSchema,storeCenterID,storeCenterRequest,StoreCenterError,type CatalogItem,type Group,type MetadataCommand,type StoreCenterScope} from "@/lib/api/store-center";
import {StoreForm} from "./store-form";
import {StoreOfficialConnection} from "./store-official-connection";
import {StoreSyncDialog} from "./store-sync-dialog";
import styles from "./store-catalog.module.css";

export type CatalogDialog={kind:"bind"}|{kind:"groups"}|{kind:"profile"|"rename"|"sync";item:CatalogItem;initialKind?:"products"|"orders"}|{kind:"assign";items:CatalogItem[]};
type Props={dialog:CatalogDialog;scope:StoreCenterScope;groups:(Group & {count?:number})[];canManageGroups:boolean;onClose:()=>void;onChanged:()=>Promise<unknown>};
export function StoreCatalogDialogs(props:Props){
 const {dialog}=props;
 if(dialog.kind==="bind")return <BindDialog {...props}/>;
 if(dialog.kind==="sync")return <StoreSyncDialog initialKind={dialog.initialKind} store={dialog.item} scope={props.scope} onClose={props.onClose} onChanged={props.onChanged}/>;
 if(dialog.kind==="rename")return <RenameDialog {...props} item={dialog.item}/>;
 return <MetadataDialog {...props}/>;
}
const intentSchema=z.object({key:storeCenterID,command:metadataCommandSchema}).strict();
function RenameDialog({item,scope,onClose,onChanged}:Props&{item:CatalogItem}){
 const [store,setStore]=useState<WorkbenchStore>(item),[recovery,setRecovery]=useState<"idle"|"loading"|"failed">("idle"),[conflict,setConflict]=useState<{latest:WorkbenchStore;changedFields:("name"|"region")[]}|null>(null),[baseline,setBaseline]=useState<WorkbenchStore>(item);
 const lifetime=useRef(new AbortController());useEffect(()=>{const controller=new AbortController();lifetime.current=controller;return()=>controller.abort()},[]);
 const read=async(base:WorkbenchStore)=>{setBaseline(base);setRecovery("loading");try{const latest=await getWorkbenchStore(item.id,scope.organizationId);if(lifetime.current.signal.aborted)return;if(latest.recordStatus==="deleting"){setStore(latest);setConflict(null)}else{setConflict({latest,changedFields:(["name","region"] as const).filter(field=>base[field]!==latest[field])})}setRecovery("idle")}catch{if(!lifetime.current.signal.aborted)setRecovery("failed")}};
 return <ResourceDialog title="修改店铺名称" className={styles.dialog} onClose={onClose} locked={recovery==="loading"}>{store.recordStatus==="deleting"?<p>店铺正在删除，无法继续修改。</p>:<StoreForm mode="edit" store={store} conflict={conflict} recoveryState={recovery} onConflict={(_draft,base)=>void read(base)} onSaved={()=>{void onChanged();onClose()}}/>}{recovery==="failed"?<p role="alert">最新版本暂时无法确认，草稿已保留。<Button variant="outline" onClick={()=>void read(baseline)}>重试获取最新版本</Button></p>:null}</ResourceDialog>;
}
type Intent=z.infer<typeof intentSchema>;
function useMetadataOperation(scope:StoreCenterScope,onChanged:()=>Promise<unknown>){
 const lifetime=useRef(new AbortController()),key=`store-metadata:${scope.organizationId}:${scope.userId}`;
 const [intent,setIntent]=useState<Intent|null>(null),[ready,setReady]=useState(false),[pending,setPending]=useState(false),[message,setMessage]=useState(""),[rejected,setRejected]=useState(false);
 useEffect(()=>{const controller=new AbortController();lifetime.current=controller;let active=true;void Promise.resolve().then(()=>{if(!active)return;try{const raw=sessionStorage.getItem(key);if(raw){const parsed=intentSchema.safeParse(JSON.parse(raw));if(parsed.success)setIntent(parsed.data);else throw Error("invalid intent")}setReady(true)}catch{setMessage("无法保存或读取操作键，请检查浏览器存储后重试。")}});return()=>{active=false;controller.abort()}},[key]);
 const accept=async()=>{sessionStorage.removeItem(key);if(lifetime.current.signal.aborted)return;setIntent(null);setMessage("设置已保存。后续修改使用新版本。");setRejected(false);await onChanged()};
 const invoke=async(captured:Intent,verify=false)=>{
  if(pending || !ready)return;setPending(true);setMessage(verify?"正在确认原操作结果…":"正在保存…");
  try{await storeCenterRequest(verify?`metadata/${captured.key}`:"metadata",scope,metadataResultSchema,{signal:lifetime.current.signal,...(!verify?{key:captured.key,command:captured.command}:{})});await accept()}
  catch(error){if(lifetime.current.signal.aborted)return;const code=error instanceof StoreCenterError?error.code:"OUTCOME_UNKNOWN";const definitive=["INVALID_REQUEST","VERSION_CONFLICT","STORE_VERSION_CONFLICT","IDEMPOTENCY_KEY_CONFLICT"].includes(code);setRejected(definitive);setMessage(definitive?"操作被拒绝，设置未按此请求更新。请刷新版本后重新确认。":code==="NOT_FOUND"||code==="STORE_NOT_FOUND"||code==="PERMISSION_DENIED"?"当前无法访问原操作结果，请检查店铺权限后用原键确认。":"操作结果待确认。保留原操作键，确认或重试原请求，不创建第二次操作。")}
  finally{if(!lifetime.current.signal.aborted)setPending(false)}
 };
 const execute=async(command:MetadataCommand)=>{
  if(intent || pending || !ready)return;let captured:Intent;
  try{captured={key:crypto.randomUUID(),command:metadataCommandSchema.parse(command)};sessionStorage.setItem(key,JSON.stringify(captured));setIntent(captured)}catch{setMessage("设置格式无效或操作键无法保存，请检查输入和浏览器存储。");return}
  await invoke(captured);
 };
 const discard=async()=>{if(!rejected || pending)return;setPending(true);try{await onChanged();sessionStorage.removeItem(key);if(lifetime.current.signal.aborted)return;setIntent(null);setRejected(false);setMessage("")}finally{if(!lifetime.current.signal.aborted)setPending(false)}};
 return {intent,pending,ready,message,rejected,execute,verify:()=>intent&&invoke(intent,true),retry:()=>intent&&invoke(intent),discard};
}
const emptyCommand:MetadataCommand={action:"create_group",name:"",groupId:null,groupVersion:0,stores:[],responsibleMemberId:null,lowStockThreshold:null};
function MetadataDialog({dialog,scope,groups,canManageGroups,onClose,onChanged}:Props){
 const context=useWorkbenchContext();
 const query=useQuery({queryKey:["store-groups",scope.organizationId,scope.userId],queryFn:({signal})=>storeCenterRequest("groups",scope,groupsSchema,{signal}),retry:false,gcTime:0});
 const [name,setName]=useState(""),[editing,setEditing]=useState<Group|null>(null),[deleting,setDeleting]=useState<Group|null>(null);
 const operation=useMetadataOperation(scope,async()=>{await onChanged();await query.refetch();setEditing(null);setDeleting(null);setName("");if(dialog.kind!=="groups")onClose()});
 const item=dialog.kind==="profile"?dialog.item:null;
 const [groupId,setGroupId]=useState(item?.profile.groupId??(dialog.kind==="assign"&&dialog.items.length===1?dialog.items[0].profile.groupId??"":"")),[responsible,setResponsible]=useState(item?.profile.responsibleMemberId??""),[threshold,setThreshold]=useState(item?.profile.lowStockThreshold?.toString()??"");
 const available=query.isError?[]:query.data?.items??groups,locked=operation.pending||!operation.ready||Boolean(operation.intent);
 const submit=()=>{
  if(dialog.kind==="groups")void operation.execute({...emptyCommand,action:deleting?"delete_group":editing?"rename_group":"create_group",name:deleting?"":name.trim(),groupId:(deleting??editing)?.id??null,groupVersion:(deleting??editing)?.groupVersion??0});
  else if(dialog.kind==="profile")void operation.execute({...emptyCommand,action:"set_profile",groupId:groupId||null,stores:[{storeId:dialog.item.id,profileVersion:dialog.item.profile.profileVersion}],responsibleMemberId:responsible||null,lowStockThreshold:threshold===""?null:Number(threshold)});
  else if(dialog.kind==="assign")void operation.execute({...emptyCommand,action:"assign_group",groupId:groupId||null,stores:dialog.items.map(item=>({storeId:item.id,profileVersion:item.profile.profileVersion}))});
 };
 // Keep the original captured versions until an explicit re-read after a conflict.
 const title=dialog.kind==="groups"?"分组管理":dialog.kind==="assign"?"加入店铺分组":"负责人和库存预警";
 return <ResourceDialog title={title} className={styles.dialog} onClose={onClose} locked={operation.pending}>
  <p>{dialog.kind==="groups"?"统一维护店铺分组，绑定后可按店铺或批量加入分组。":dialog.kind==="assign"?`将 ${dialog.items.length} 家选定店铺加入同一分组，全部成功后保存。`:item?.name}</p>
  {operation.message?<p role="status" className={operation.rejected?styles.error:styles.hint}>{operation.message}</p>:null}
  {operation.intent?<div className={styles.hint}><p>原操作：{operation.intent.command.action} · {operation.intent.key}</p><Button size="sm" disabled={operation.pending} onClick={()=>void operation.verify()}>确认原操作结果</Button><Button size="sm" variant="outline" disabled={operation.pending} onClick={()=>void operation.retry()}>重试原请求</Button>{operation.rejected?<Button size="sm" variant="ghost" onClick={()=>void operation.discard()}>{dialog.kind==="groups"?"刷新后重新编辑":"刷新列表并关闭"}</Button>:null}</div>:null}
  {dialog.kind==="groups"?<>
   {canManageGroups?<form className={styles.groupEditor} onSubmit={event=>{event.preventDefault();submit()}}>{deleting?<p className={styles.error}>确认删除“{deleting.name}”？该组店铺会自动移出分组，授权和服务不会改变。</p>:<label>{editing?"分组名称":"新建分组"}<Input value={name} disabled={locked} maxLength={60} onChange={event=>setName(event.target.value)} placeholder="输入分组名称"/></label>}<footer>{editing||deleting?<Button type="button" variant="outline" disabled={locked} onClick={()=>{setEditing(null);setDeleting(null);setName("")}}>取消</Button>:null}<Button disabled={locked || (!deleting&&!name.trim())} type="submit">{deleting?"确认删除":editing?"保存名称":<><Plus size={14}/>新建分组</>}</Button></footer></form>:<p className={styles.hint}>企业管理员可创建、重命名和删除共享分组。</p>}
   <div className={styles.groupColumns}><span>分组名称</span><span>店铺数量</span><span>操作</span></div>
   <ul className={styles.groupList}>{available.map(group=>{const count=groups.find(value=>value.id===group.id)?.count;return <li key={group.id}><span>{group.name}</span><span>{count===undefined?"数量待确认":`${count} 家店铺`}</span><div>{canManageGroups?<><Button variant="outline" size="sm" disabled={locked} aria-label={`重命名${group.name}`} onClick={()=>{setEditing(group);setDeleting(null);setName(group.name)}}>重命名</Button><Button variant="outline" size="sm" className={styles.groupDelete} disabled={locked} aria-label={`删除${group.name}`} onClick={()=>{setDeleting(group);setEditing(null);setName("")}}>删除</Button></>:<span>仅查看</span>}</div></li>})}</ul>
   {query.isError?<p className={styles.error}>暂时无法读取最新分组，请重试。</p>:null}
   <p>删除分组不会解绑店铺，分组内店铺将自动移出该分组。</p>
  </>:<form onSubmit={event=>{event.preventDefault();submit()}}>
   <label>店铺分组<Select value={groupId} disabled={locked || query.isError} onChange={event=>setGroupId(event.target.value)}><option value="">未分组</option>{available.map(group=><option key={group.id} value={group.id}>{group.name}</option>)}</Select></label>
   {dialog.kind==="profile"?<><ResponsiblePicker scope={scope} value={responsible} onChange={setResponsible} disabled={locked || !context.permissions.includes("workbench.organization_member.read")}/><label>低库存阈值（件）<Input type="number" min={0} max={Number.MAX_SAFE_INTEGER} step={1} value={threshold} disabled={locked} placeholder="未设置" onChange={event=>setThreshold(event.target.value)}/></label><p className={styles.hint}>每个 SKU、每个仓库分别比较。已知库存 ≤ 阈值时预警；留空关闭，缺失库存显示未知。负责人只记录业务归属。</p></>:null}
   <footer><Button type="button" variant="outline" onClick={onClose} disabled={operation.pending}>取消</Button><Button type="submit" disabled={locked || query.isError}>保存设置</Button></footer>
  </form>}
 </ResourceDialog>;
}
function ResponsiblePicker({scope,value,onChange,disabled}:{scope:StoreCenterScope;value:string;onChange:(value:string)=>void;disabled:boolean}){
 const [search,setSearch]=useState(""),[applied,setApplied]=useState(""),[offset,setOffset]=useState(0);
 const query=useQuery({queryKey:["store-responsible-choices",scope.organizationId,scope.userId,applied,offset],queryFn:({signal})=>getMembers({expectedOrganizationId:scope.organizationId,expectedUserId:scope.userId,signal},offset,{q:applied,state:"active"}),enabled:!disabled,retry:false,gcTime:0});
 return <div><label>业务负责人<Input value={search} disabled={disabled} onChange={event=>setSearch(event.target.value)} placeholder="搜索企业成员"/></label><Button type="button" variant="ghost" size="sm" disabled={disabled} onClick={()=>{setApplied(search.trim());setOffset(0)}}>查找成员</Button><div className={styles.memberChoices}><label><input type="radio" checked={!value} onChange={()=>onChange("")} disabled={disabled}/>未设置</label>{value && !query.data?.items.some(member=>member.id===value)?<p>当前负责人 ID：{value}</p>:null}{query.data?.items.map(member=><label key={member.id}><input type="radio" checked={value===member.id} disabled={disabled} onChange={()=>onChange(member.id)}/>{member.displayName||member.loginName}</label>)}</div>{query.isError||disabled?<p>当前无法读取企业成员目录，原负责人保留；请联系有成员读取权限的管理员。</p>:null}<div><Button type="button" size="sm" variant="ghost" disabled={disabled||offset===0} onClick={()=>setOffset(offset-20)}>上一页</Button><Button type="button" size="sm" variant="ghost" disabled={disabled||!query.data||offset+20>=query.data.total} onClick={()=>setOffset(offset+20)}>下一页</Button></div></div>;
}
function BindDialog({scope,onClose,onChanged}:Props){
 const context=useWorkbenchContext(),[store,setStore]=useState<WorkbenchStore|null>(null),[busy,setBusy]=useState(false);
 const refresh=async()=>{if(!store)return;const latest=await getWorkbenchStore(store.id,scope.organizationId);setStore(latest);await onChanged()};
 return <ResourceDialog title="绑定 SHEIN 店铺" locked={busy} onClose={onClose} className={styles.dialog}>
  <p>选择官方应用并在 SHEIN 完成授权。店铺记录创建后，仍需完成官方授权。</p>
  {!store?<StoreForm mode="create" onPendingChange={setBusy} onCreated={saved=>{setBusy(false);setStore(saved);void onChanged()}}/>:<><p className={styles.hint}>店铺记录：{store.name}。继续完成官方 API 授权。</p><StoreOfficialConnection store={store} scope={{expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId}} canWrite={context.permissions.includes("workbench.store.update")} administrator={context.roles.some(role=>["listingkit_admin","platform_admin","admin"].includes(role))} onChanged={refresh}/><footer><Button onClick={onClose}>返回店铺列表</Button></footer></>}
 </ResourceDialog>;
}
