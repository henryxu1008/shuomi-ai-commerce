"use client";
import {ImageReferenceEstimate} from "./image-reference-estimate";
import {useEffect,useRef,useState} from "react";
import Link from "next/link";
import {useQuery,useQueryClient} from "@tanstack/react-query";
import {z} from "zod";
import {automaticFXStatus} from "@/lib/contracts/model-route-pricing";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {useResourcePending} from "../resources/resource-pending";
import {ModelRouteError,modelRouteRequest,type ModelRouteScope} from "@/lib/api/model-route-pricing";
import {publishedModelManufacturers,canonicalPriceFacts,withoutPublicationVersion,officialRates,routeInput,priceInput,pricePreview,routeOptions,routeCurrent,routeHistory,routePreview,routeReceipt,modelLabels,formatCurrencyMicros,parseCurrencyMicros,type RouteInput,type RouteRevision,type ModelOption,type PriceInput} from "@/lib/contracts/model-route-pricing";
import {modelServicesPath,type ModelServiceCategory,type ModelServicesNavigation} from "@/lib/contracts/model-services-navigation";
import styles from "./model-route-pricing.module.css";
import {ChannelCredentialsManager} from "./channel-credentials";
import { MaintenanceEntryCard } from "./maintenance-entry-card";
import { ModelServicesOverview } from "./model-services-overview";

const defaults:RouteInput["selection"]={schemaVersion:1,unitVersion:"ai-micropoint-v1",defaultChannel:"openai-direct",text:{id:"openai-mini",version:"approved-default-v1"},image:{id:"openai-flare",version:"approved-default-v1"},multimodal:{id:"openai-mini",version:"approved-default-v1"}};
const categories=["text","image-generation","multimodal"] as const;
const keyFor={text:"text","image-generation":"image",multimodal:"multimodal"} as const;
function sameReference(first:ModelOption["reference"], second:ModelOption["reference"]){
 return first.id===second.id&&first.version===second.version;
}
const reasonLabels: Record<string, string> = {
 POINT_BUDGET_EXCEEDED: "费用估算超出本次分析的点数预算",
 POINT_BUDGET_MISSING: "尚未配置本次分析的点数预算",
 METERING_MAPPING_NOT_ADMITTED: "暂时无法按该模型的用量计算点数",
 BINDING_NOT_ADMITTED: "当前企业暂不能使用此选项",
 PROTOCOL_AND_COST_BOUND_MISSING: "图片生成服务尚未配置",
 VISION_CONSUMER_NOT_IMPLEMENTED: "此选项暂不能用于图片分析",
 OFFICIAL_PRICE_MISSING: "缺少适用参考价",
 PRICE_OR_FX_NOT_APPLICABLE: "参考价或汇率缺失、过期或不适用",
 CREDENTIAL_NOT_ADMITTED: "渠道暂不可用，请核对渠道配置",
 CREDENTIAL_READER_MISSING: "暂时无法读取渠道状态，请联系系统维护人员",
 COMPLETE_BILLING_USAGE_MISSING: "暂时无法确认该模型的计费用量",
 COST_BOUND_MISSING: "暂时无法估算该模型所需点数",
 PRICE_APPLICABILITY_MISSING: "参考价格的地区、服务类型或图片参数不匹配",
 OPEN_CONSUMER_MISSING: "部分已开放用途暂不能使用这个型号",
 CONSUMER_NOT_QUALIFIED: "该用途暂不能使用这个型号",
 CONSUMER_BINDING_MISMATCH: "所选模型与当前用途不匹配",
 DEPENDENCY_MISSING: "当前环境暂不能使用该模型，请联系系统维护人员",
};
const pendingSchema=z.strictObject({key:z.uuid(),input:z.union([routeInput,priceInput])});
function message(error:unknown){const code=error instanceof ModelRouteError?error.code:"UNAVAILABLE";return ({INVALID:"请检查填写内容，参考价与汇率最多保留六位小数。",CONFLICT:"当前版本已有更新，请重新读取后校验。",NEEDS_CONFIGURATION:"三类模型尚未全部满足准入，请查看具体缺口。",FORBIDDEN:"无此操作权限，或企业访问已撤销。",PERMISSION_DENIED:"无此操作权限，请确认权限后重试。",AUTHENTICATION_REQUIRED:"登录已失效，请重新登录。",ORGANIZATION_ACCESS_DENIED:"无当前企业访问权限。",ORGANIZATION_ACCESS_REVOKED:"当前企业访问已撤销。",ORGANIZATION_SUSPENDED:"当前企业暂不可用。",ORGANIZATION_SELECTION_REQUIRED:"请先选择当前企业。",OUTCOME_UNKNOWN:"发布结果待核实，请使用原请求键查询回执。",ORGANIZATION_CONTEXT_CHANGED:"当前企业已变化，请重新打开页面。",IDENTITY_CONTEXT_CHANGED:"登录身份已变化，请重新打开页面。"} as Record<string,string>)[code]??"配置服务暂不可用，编辑和原回执保留。"}
export function ModelRoutePricingEntry({scope}:{scope:ModelRouteScope}){
 const context=useWorkbenchContext();
 const {configure,credentials,platform}=settingsPermissions(context.permissions);
 if (context.isLoading || context.isSwitching || context.error || context.blockingError || context.selectionRequired ||
     context.user?.id!==scope.userId || context.effectiveOrganization?.id!==scope.organizationId ||
     !(configure||credentials||platform)) return null;
 return <MaintenanceEntryCard title="AI 配置" description={configure?"查看企业的 Chat 规划状态及商品图片模型配置。":credentials?"录入、更新或停用本企业共用的渠道 Key。":"查看平台共用的模型参考价格与汇率。"} href={modelServicesPath} action="模型与服务" />;
}

function settingsPermissions(permissions: string[]) {
 return {
  configure: permissions.includes("workbench.agent.configure"),
  credentials: permissions.includes("workbench.ai.credentials.manage"),
  platform: permissions.includes("listingkit.platform_admin"),
 };
}

export function ModelRoutePricingPage({expectedUserId,navigation}:{expectedUserId:string;navigation?:ModelServicesNavigation}){
 const context=useWorkbenchContext();
 const org=context.effectiveOrganization;
 const permissions=settingsPermissions(context.permissions);
 const allowed=permissions.configure||permissions.credentials||permissions.platform;
 const transientError=context.error&&(context.error.status===0||context.error.status>=500);
 const validScope=allowed&&org&&context.user?.id===expectedUserId&&!context.blockingError&&!context.selectionRequired&&(!context.error||transientError);
 const destination=navigation?.organizationId===org?.id?navigation:undefined;
 return <ConsolePage title="模型与服务" description="查看和管理企业使用的 AI 模型、渠道 Key 和参考价格。" breadcrumbs={[{label:"我的账户",href:"/workbench/account"},{label:"企业空间",href:"/workbench/account/organization"},{label:"模型与服务"}]}>
 {context.isLoading||context.isSwitching?<ConsoleState kind="loading" title="正在确认当前企业"/>:!validScope?<ConsoleState kind="error" title="无模型配置权限或当前企业不可用"/>:<ModelSettings key={JSON.stringify([expectedUserId,org.id,context.effectiveMemberId,context.permissions,context.roles,destination])} scope={{userId:expectedUserId,organizationId:org.id}} name={org.name} {...permissions} canMaintain={context.permissions.includes("listingkit.admin.write")} navigation={destination}/>}
 </ConsolePage>;
}

type ModelSettingsArea = "enterprise" | "platform-prices" | "credentials";
type ModelDestination = { area: ModelSettingsArea; category?: ModelServiceCategory; model?: ModelOption };
const settingsKey = (scope: ModelRouteScope) => ["model-services",scope] as const;

function usePublishedModels(scope:ModelRouteScope,enabled=true) {
 return useQuery({queryKey:[...settingsKey(scope),"current"],queryFn:({signal})=>modelRouteRequest(scope,"/enterprise",routeCurrent,{signal}),enabled,retry:false,gcTime:0,refetchOnWindowFocus:false});
}

function ModelSettings({ scope, name, configure, platform, credentials, canMaintain, navigation }: { scope: ModelRouteScope; name: string; configure: boolean; platform: boolean; credentials: boolean; canMaintain: boolean; navigation?:ModelServicesNavigation }) {
 const context=useWorkbenchContext();
 const client=useQueryClient();
 const [destination,setDestination]=useState<ModelDestination>(()=>({
  area:configure?"enterprise":credentials?"credentials":"platform-prices",
  category:configure?navigation?.category:undefined,
 }));
 const [draft,setDraft]=useState<RouteInput|null>(null);
 const [refreshState,setRefreshState]=useState<"idle"|"checking"|"failed">("idle");
 const pendingRefresh=useRef<Promise<void>|null>(null);
 const active=useRef(true);
 useEffect(()=>{
  active.current=true;
  return()=>{active.current=false};
 },[]);
 const areaHeading=useRef<HTMLDivElement>(null);
 const overview=useRef<HTMLDivElement>(null);
 const overviewCategory=useRef<ModelServiceCategory>(null);
 const published=usePublishedModels(scope,configure && !context.error);
 const {area,model}=destination;
 const checking=refreshState==="checking";
 const blocked=!!context.error||checking;

 function refresh(afterSave=false) {
  if(pendingRefresh.current&&!afterSave)return pendingRefresh.current;
  setRefreshState("checking");
  // A refresh already started before the receipt may have read the old version.
  const run=(pendingRefresh.current??Promise.resolve()).then(async()=>{
   if(!active.current)return;
   const results=await Promise.allSettled([
    client.invalidateQueries({queryKey:settingsKey(scope),predicate:query=>query.queryKey[2]!=="history"}, {throwOnError:true}),
    client.invalidateQueries({queryKey:[...settingsKey(scope),"history"]}),
    context.retry(),
   ]);
   if(pendingRefresh.current!==run)return;
   pendingRefresh.current=null;
   const failure=results.some(result=>result.status==="rejected" || result.value===null);
   if(active.current)setRefreshState(afterSave&&failure?"failed":"idle");
  });
  pendingRefresh.current=run;
  return run;
 }

 function openSettings(next:ModelDestination, fromOverview?:ModelServiceCategory) {
  if(fromOverview)overviewCategory.current=fromOverview;
  setDestination(next);
  if(!next.category&&!next.model)areaHeading.current?.focus();
 }

 function returnToOverview() {
  const category=overviewCategory.current??navigation?.category;
  const heading=category&&overview.current?.querySelector<HTMLElement>(`[data-model-service="${category}"]`);
  (heading||overview.current)?.focus();
 }

 const areas = [
   { id: "enterprise", label: "企业模型", available: configure },
   { id: "credentials", label: "提供方与 Key", available: credentials },
   { id: "platform-prices", label: "参考价格与汇率", available: platform },
 ] as const;
 const availableAreas = areas.filter(item => item.available);
 return <div className={styles.stack}>
   <div className={styles.workspaceHeader}>
     <p className={styles.scope}>当前企业 <strong>{name}</strong></p>
     {navigation?.returnTo&&!context.error&&<Button asChild variant="outline"><Link href={navigation.returnTo} prefetch={false}>返回原业务页面</Link></Button>}
   </div>
   {refreshState==="failed"&&<p role="status">本次已保存，当前状态暂时无法确认。</p>}
   {context.error&&<p role="alert">当前企业状态读取失败，请重新读取后继续。</p>}
   {configure&&<div ref={overview} tabIndex={-1} aria-label="使用概览">
     <ModelServicesOverview
       current={!context.error&&!published.isError&&!published.isFetching?published.data:undefined}
       checking={checking||published.isFetching}
       chat={checking?"checking":context.error?"unknown":!context.aiWorkbenchAvailable?"closed":context.aiWorkbenchPlanningReadiness}
       credentials={credentials} platform={platform} disabled={blocked}
       onSelectModel={selected=>openSettings({area:"enterprise",category:selected},selected)}
       onCredentials={selected=>openSettings({area:"credentials",model:selected},selected.category)}
       onPrices={selected=>openSettings({area:"platform-prices"},selected)}
       onRefresh={()=>void refresh()}
     />
   </div>}
   <div className={styles.workspaceHeader} ref={areaHeading} tabIndex={-1} aria-label="配置区域">
     {availableAreas.length > 1 && <div className={styles.tabs} role="group" aria-label="配置区域">
       {availableAreas.map(item => <Button key={item.id} variant="ghost" aria-pressed={area === item.id} disabled={blocked} onClick={() => setDestination({area:item.id})}>{item.label}</Button>)}
     </div>}
     {area === "platform-prices" && <p className={styles.scope}>适用范围：平台共用</p>}
     {configure&&<Button variant="ghost" onClick={returnToOverview}>返回使用概览</Button>}
     {context.error&&!configure&&<Button variant="outline" disabled={checking} onClick={()=>void refresh()}>重新读取状态</Button>}
   </div>
   {area === "enterprise" ? <EnterpriseModelEditor scope={scope} draftState={[draft,setDraft]} focusTarget={destination} disabled={blocked} unconfirmed={!!context.error} onSaved={()=>refresh(true)} onCredentials={credentials?selected=>openSettings({area:"credentials",model:selected},selected.category):undefined} />
     : area === "credentials" ? <ChannelCredentialsManager scope={scope} testMemberId={context.permissions.includes("workbench.ai.connection.test")?context.effectiveMemberId??undefined:undefined} focusModel={model} disabled={blocked} unconfirmed={!!context.error} onSaved={()=>refresh(true)} />
       : <OfficialPriceEditor scope={scope} canPublish={canMaintain&&!blocked} disabled={blocked} unconfirmed={!!context.error} onSaved={()=>refresh(true)} />}
   {area !== "credentials" && <details className={styles.help}>
     <summary>AI 点数如何结算</summary>
     <p>1 AI 点等于 ¥0.01，精确到百万分之一点。发起调用前预留点数，完成后按已记录用量和当时的参考价与汇率结算，释放多余点数。缺少可靠用量时保留待确认状态。估算与厂商最终账单可能有差异。</p>
   </details>}
 </div>;
}
function usePublish(scope:ModelRouteScope,family:"enterprise"|"platform-prices",refresh:()=>Promise<unknown>){
 const pending=useResourcePending({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId},["model-route-pricing",family,"unified-v1"],pendingSchema,{storage:"session",maxLength:16384});
 const [busy,setBusy]=useState(false),[failure,setFailure]=useState<unknown>(),[receipt,setReceipt]=useState<RouteRevision|null>(null),active=useRef(true),running=useRef(false),controller=useRef<AbortController|null>(null);
 useEffect(()=>{active.current=true;return()=>{active.current=false;controller.current?.abort()}},[]);
 async function run(work:(signal:AbortSignal)=>Promise<void>){if(running.current)return;running.current=true;setBusy(true);setFailure(undefined);const abort=new AbortController();controller.current=abort;try{await work(abort.signal)}catch(error){if(active.current)setFailure(error)}finally{running.current=false;if(active.current)setBusy(false)}}
 async function publish(input:RouteInput|PriceInput){if(!pending.ready||pending.command||pending.error)return;await run(async signal=>{const intent={key:crypto.randomUUID(),input};pending.persist(intent);try{const result=await modelRouteRequest(scope,"/"+family+"/publish",routeReceipt,{method:"POST",headers:{"Content-Type":"application/json","Idempotency-Key":intent.key},body:JSON.stringify(input),signal});if(active.current){pending.clear(intent);setReceipt(result.receipt);await refresh()}}catch(error){if(active.current&&error instanceof ModelRouteError&&error.status<500&&error.code!=="OUTCOME_UNKNOWN")pending.clear(intent);throw error}})}
 function verifyReceipt(receipt:RouteRevision,intent:{key:string;input:RouteInput|PriceInput}){
  const platform=family==="platform-prices";
  if(receipt.key!==intent.key||receipt.actorId!==scope.userId||receipt.parent!==intent.input.expectedRevision||receipt.scope.organizationId!==(platform?"platform-ai-models":scope.organizationId)||receipt.scope.capability!==(platform?"ai.models.official-prices":"ai.models.enterprise"))throw new ModelRouteError("OUTCOME_UNKNOWN");
  const matches=platform?!!receipt.payload.platformPrices&&canonicalPriceFacts(receipt.payload.platformPrices)===canonicalPriceFacts(priceInput.parse(intent.input)):JSON.stringify(receipt.payload.sharedModels)===JSON.stringify(routeInput.parse(intent.input).selection);
  if(!matches)throw new ModelRouteError("OUTCOME_UNKNOWN");
 }
 async function verify(){const intent=pending.command;if(!intent)return;await run(async signal=>{const result=await modelRouteRequest(scope,"/"+family+"/receipts/"+encodeURIComponent(intent.key),routeReceipt,{signal});verifyReceipt(result.receipt,intent);if(active.current){pending.clear(intent);setReceipt(result.receipt);await refresh()}})}
 return {busy,failure,receipt,run,publish,verify,pending,active,locked:busy||!pending.ready||!!pending.command||pending.error};
}
function PublishStatus({state,disabled=false}:{state:ReturnType<typeof usePublish>;disabled?:boolean}){return <>{state.failure?<p role="alert" className={styles.error}>{message(state.failure)}</p>:null}{state.pending.command||state.pending.error?<Card className={styles.panel}><h2>发布结果待核实</h2><p>请求编号：{state.pending.command?.key??"记录读取失败"}。请查询本次结果，确认后再提交其他修改。</p><Button disabled={disabled||state.busy||!state.pending.command} onClick={()=>void state.verify()}>查询本次发布结果</Button></Card>:null}{state.receipt?<Card className={styles.panel} role="status"><h2>发布回执</h2><p>{state.receipt.id} · {state.receipt.publishedAt}</p><p>这是本次发布结果，实际当前版本以重新读取为准。</p></Card>:null}</>}
type ModelEditorProps = {
 scope:ModelRouteScope;
 draftState?:[RouteInput|null,(draft:RouteInput|null)=>void];
 focusTarget?:ModelDestination;
 disabled?:boolean;
 unconfirmed?:boolean;
 onSaved?:()=>Promise<void>;
 onCredentials?:(model:ModelOption)=>void;
};

export function EnterpriseModelEditor({scope,draftState,focusTarget,disabled=false,unconfirmed=false,onSaved,onCredentials}:ModelEditorProps){
 const published=usePublishedModels(scope,!unconfirmed);
 const options=useQuery({queryKey:[...settingsKey(scope),"options"],queryFn:({signal})=>modelRouteRequest(scope,"/enterprise/options",routeOptions,{signal}),enabled:!unconfirmed,retry:false,gcTime:0,refetchOnWindowFocus:false});
 const history=useQuery({queryKey:[...settingsKey(scope),"history","enterprise"],queryFn:({signal})=>modelRouteRequest(scope,"/enterprise/history",routeHistory,{signal}),enabled:!unconfirmed,retry:false,gcTime:0,refetchOnWindowFocus:false});
 const localDraft=useState<RouteInput|null>(null);
 const [draft,setDraft]=draftState??localDraft;
 const [preview,setPreview]=useState<{input:RouteInput;value:z.infer<typeof routePreview>;readAt:number;optionsAt:number}|null>(null);
 const state=usePublish(scope,"enterprise",async()=>{
  setDraft(null);
  setPreview(null);
  await (onSaved ? onSaved() : Promise.all([published.refetch(),options.refetch(),history.refetch()]));
 });
 const current=published.data?.revision,id=current?.id??"NONE",choice=draft?.selection??current?.payload.sharedModels??defaults;
 const loading=published.isPending||options.isPending;
 const readFailed=published.isError||options.isError;
 const draftStale=!!draft&&draft.expectedRevision!==id;
 const editingBlocked=disabled||state.locked||draftStale||published.isFetching||options.isFetching;
 const optionsData=options.data;
 const unavailableCategories=categories.filter(category=>{
  const selected=choice[keyFor[category]];
  return !optionsData?.models.some(model=>model.category===category&&model.reference.id===selected.id&&model.reference.version===selected.version);
 });
 const validSelection=unavailableCategories.length===0&&!!optionsData?.channels.includes(choice.defaultChannel);
 const currentPreview=preview&&preview.readAt===published.dataUpdatedAt&&preview.optionsAt===options.dataUpdatedAt&&preview.input.expectedRevision===id?preview:null;
 function choose(selection:RouteInput["selection"]|null) {
  setPreview(null);
  setDraft(selection?{expectedRevision:id,selection}:null);
 }
 function select(key:keyof typeof keyFor,value:string){
  const model=optionsData?.models.find(m=>m.category===key&&m.reference.id+"/"+m.reference.version===value);
  if(model)choose({...choice,[keyFor[key]]:model.reference});
 }
 async function validate(){
  if(loading||readFailed||editingBlocked||!validSelection||!published.data)return;
  const input=routeInput.parse({expectedRevision:id,selection:choice});
  const readAt=published.dataUpdatedAt,optionsAt=options.dataUpdatedAt;
  await state.run(async signal=>{
   const value=await modelRouteRequest(scope,"/enterprise/validate",routePreview,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(input),signal});
   if(state.active.current)setPreview({input,value,readAt,optionsAt});
  });
 }
 function reload() {
  setPreview(null);
  void Promise.all([published.refetch(),options.refetch()]);
 }
 const previewHeading = useRef<HTMLHeadingElement>(null);
 const sectionHeading=useRef<HTMLHeadingElement>(null);
 const fields=useRef<Partial<Record<ModelServiceCategory,HTMLSelectElement|null>>>({});
 const lastFocusedTarget=useRef<ModelDestination>(null);
 useEffect(() => { if (currentPreview) previewHeading.current?.focus(); }, [currentPreview]);
 useEffect(()=>{
  if(!focusTarget?.category||focusTarget===lastFocusedTarget.current||loading||disabled)return;
  lastFocusedTarget.current=focusTarget;
  (fields.current[focusTarget.category]??sectionHeading.current)?.focus();
 },[focusTarget,loading,disabled]);
 return <>
 {!unconfirmed&&<>
   <Card className={styles.panel}>
     <div className={styles.sectionHeader}>
       <div>
         <h2 ref={sectionHeading} tabIndex={-1}>企业共用模型</h2>
         <p>为企业成员选择文本、图片生成和图片理解模型。</p>
       </div>
       <Button variant="outline" disabled={disabled||state.busy} onClick={reload}>重新读取当前配置</Button>
     </div>
     {loading ? <p role="status">正在读取模型配置…</p>
       : readFailed||!optionsData||!published.data ? <p role="alert">{published.isError?"当前配置暂时无法确认。请重试，仍失败时联系系统维护人员。":"可选模型暂时无法读取，请重试。"}</p>
         : <>
           <p>当前版本：{current?.id ?? "暂无"} · {published.data.status === "READY" ? "已发布" : published.data.status === "UNCONFIGURED" ? "尚未发布" : "已发布，部分模型待处理"}</p>
           {draftStale&&<div role="alert"><p>企业模型已更新，请核对当前版本后重新选择。</p><Button variant="outline" disabled={disabled||state.locked} onClick={()=>choose(null)}>放弃草稿并载入当前版本</Button></div>}
           <fieldset disabled={editingBlocked} className={styles.form}>
             <label className={`${styles.field} ${styles.channelField}`}>
               <strong>默认渠道</strong>
               <select value={choice.defaultChannel} onChange={event => choose({...choice, defaultChannel: event.target.value})}>
                 {!optionsData.channels.includes(choice.defaultChannel)&&<option value={choice.defaultChannel}>{choice.defaultChannel}（当前不可用）</option>}
                 {optionsData.channels.map(channel => <option key={channel}>{channel}</option>)}
               </select>
             </label>
             <div className={styles.modelGrid}>
               {categories.map(category => {
                 const candidates = optionsData.models.filter(model => model.category === category);
                 const key = keyFor[category];
                 const ref = choice[key];
                 const selected = candidates.find(model => sameReference(model.reference,ref));
                 const publishedRef = current?.payload.sharedModels?.[key];
                 return <div className={styles.modelCard} key={category}>
                   <label className={styles.field}>
                     <strong>{modelLabels[category]}</strong>
                     <select ref={element=>{fields.current[category]=element}} value={ref.id + "/" + ref.version} onChange={event => select(category, event.target.value)}>
                       {!selected && <option value={ref.id + "/" + ref.version}>{ref.id}（原版本当前不可用）</option>}
                       {candidates.map(model => {
                         const value = model.reference.id + "/" + model.reference.version;
                         const sameName = candidates.some(other => other !== model && other.model === model.model && other.channel === model.channel);
                         const currentLabel = publishedRef && sameReference(publishedRef, model.reference) ? "当前已发布 · " : "";
                         const readiness = model.ready ? "模型配置符合条件" : reasonLabels[model.reason??""] ?? "尚未就绪";
                         const suffix = sameName ? `（${currentLabel}${readiness}）` : model.ready ? "" : "（尚未就绪）";
                         return <option key={value} value={value}>{model.model} · {model.channel}{suffix}</option>;
                       })}
                     </select>
                   </label>
                   {candidates.length===0&&<p>暂无可选模型，请联系系统维护人员。</p>}
                   {selected && <ModelReadiness model={selected} showModel={false} />}
                   {selected?.reason==="CREDENTIAL_NOT_ADMITTED"&&(onCredentials
                    ? <Button variant="outline" size="sm" onClick={()=>onCredentials(selected)}>核对候选模型的渠道</Button>
                    : <p>如需管理渠道，请联系本企业渠道管理员。</p>)}
                 </div>;
               })}
             </div>
             {!validSelection&&<p>请选择当前可用的默认渠道和三类候选型号，再检查并发布。</p>}
             <div className={styles.actions}><Button disabled={!validSelection} onClick={() => void validate()}>检查并预览</Button></div>
           </fieldset>
         </>}
   </Card>
   {currentPreview && <Card className={styles.panel}>
     <h2 ref={previewHeading} tabIndex={-1}>确认企业模型配置</h2>
     <div className={styles.modelGrid}>
       {currentPreview.value.preview.models.map(model => <div className={styles.modelCard} key={model.category}><ModelReadiness model={model} /></div>)}
     </div>
     <p>三类模型选择一并保存，使用时分别检查所选模型。发布配置不会发起模型调用或启用尚未就绪的功能。已有回执和待确认费用会保留。</p>
     {!currentPreview.value.preview.ready && <p>部分模型还需配置，请在对应业务页核对使用条件。</p>}
     <div className={styles.actions}>
       <Button disabled={editingBlocked||readFailed||!validSelection} onClick={() => void state.publish(currentPreview.input)}>确认发布三类模型</Button>
     </div>
   </Card>}
   <History rows={history.isError?[]:history.data?.revisions??[]} current={id} select={row=>{if(row.payload.sharedModels)choose(row.payload.sharedModels)}} disabled={editingBlocked||readFailed||loading} loading={history.isPending} failed={history.isError} retry={()=>void history.refetch()}/>
 </>}
 <PublishStatus state={state} disabled={disabled}/>
 </>;
}

function ModelReadiness({ model, showModel = true }: { model: ModelOption; showModel?: boolean }) {
 return <div className={styles.summary}>
   {showModel && <h3>{modelLabels[model.category]}：{model.model}</h3>}
   <p data-ready={model.ready} className={styles.readiness}>{model.ready ? "模型配置符合条件" : reasonLabels[model.reason ?? ""] ?? "模型配置待处理"}</p>
   {model.price ? <details className={styles.rateDetails}>
     <summary>参考价与汇率</summary>
     <RateSummary rates={model.price.rates} currency={model.price.currency} />
     <p>价目版本 {model.price.priceVersion}</p>
     <p>1 {model.price.currency} = {formatCurrencyMicros(model.price.cnyMicrosPerCurrencyUnit)} 元人民币 · 汇率版本 {model.price.fxVersion}</p>
   </details> : <p>暂无适用参考价或汇率。</p>}
 </div>;
}

function History({ rows, current, select, disabled, loading, failed, retry }: { rows: RouteRevision[]; current: string; select: (row: RouteRevision) => void; disabled: boolean; loading?:boolean;failed?:boolean;retry?:()=>void }) {
 return <Card className={styles.panel}>
   <h2>最近版本</h2>
   {loading ? <p role="status">正在读取最近版本…</p> : failed ? <div><p>最近版本暂时无法读取。</p><Button variant="outline" disabled={disabled} onClick={retry}>重新读取最近版本</Button></div> : rows.length ? <>
     <p>采用历史版本会载入编辑区，并按当前条件重新检查。</p>
     <ul className={styles.historyList}>
       {rows.map(row => <li className={styles.historyRow} key={row.id}>
         <div>
           <strong>{row.id}{row.id === current ? " · 当前版本" : ""}</strong>
           <p><time dateTime={row.publishedAt}>{row.publishedAt}</time> · {row.actorId}</p>
         </div>
         <Button variant="outline" disabled={disabled} onClick={() => select(row)}>用此版本重新编辑</Button>
       </li>)}
     </ul>
   </> : <p>暂无发布版本。</p>}
 </Card>;
}
const rateLabels={inputText:"文字输入 / 百万 Token",cachedInputText:"缓存文字输入 / 百万 Token",inputImage:"图片输入 / 百万 Token",cachedInputImage:"缓存图片输入 / 百万 Token",outputText:"文字输出 / 百万 Token",outputImage:"图片输出 / 百万 Token",outputImagePerItem:"图片输出 / 张"};
function RateSummary({rates,currency}:{rates:z.infer<typeof officialRates>;currency:string}){return <dl>{Object.entries(rates).filter(([,value])=>value>0).map(([key,value])=><div key={key}><dt>{rateLabels[key as keyof typeof rateLabels]}</dt><dd>{formatCurrencyMicros(value)} {currency}</dd></div>)}</dl>}
export function OfficialPriceEditor({scope,canPublish,disabled=false,unconfirmed=false,onSaved}:{scope:ModelRouteScope;canPublish:boolean;disabled?:boolean;unconfirmed?:boolean;onSaved?:()=>Promise<void>}){
 const automatic=useQuery({queryKey:["automatic-fx-status",scope],queryFn:({signal})=>modelRouteRequest(scope,"/platform-prices/automatic-fx",automaticFXStatus,{signal}),retry:false,gcTime:0,refetchOnWindowFocus:false});
 const [fxStartState,setFxStartState]=useState<"ready"|"pending"|"unknown"|"started">("ready"),fxStartLock=useRef(false);
 async function startFX(){
  if(fxStartLock.current||!canPublish||fxStartState!=="ready"||automatic.data?.state!=="PREPARED")return;
  fxStartLock.current=true;setFxStartState("pending");
  try{const current=await modelRouteRequest(scope,"/platform-prices/automatic-fx",automaticFXStatus);if(current.state!=="PREPARED"){setFxStartState("unknown");await automatic.refetch();return}await modelRouteRequest(scope,"/platform-prices/automatic-fx/start",z.strictObject({state:z.literal("STARTED")}),{method:"POST",headers:{"Content-Type":"application/json"},body:"{}"});setFxStartState("started");await automatic.refetch()}catch{setFxStartState("unknown")}
 }
 const query=useQuery({queryKey:[...settingsKey(scope),"prices"],queryFn:({signal})=>modelRouteRequest(scope,"/platform-prices",routeCurrent,{signal}),enabled:!unconfirmed,retry:false,gcTime:0,refetchOnWindowFocus:false,refetchInterval:60000});
 const history=useQuery({queryKey:[...settingsKey(scope),"history","prices"],queryFn:({signal})=>modelRouteRequest(scope,"/platform-prices/history",routeHistory,{signal}),enabled:!unconfirmed,retry:false,gcTime:0,refetchOnWindowFocus:false});
 const [draft,setDraft]=useState<PriceInput|null>(null),[preview,setPreview]=useState<{input:PriceInput;value:z.infer<typeof pricePreview>}|null>(null),[fieldError,setFieldError]=useState("");
 const state=usePublish(scope,"platform-prices",async()=>{
  setDraft(null);
  setPreview(null);
  await (onSaved ? onSaved() : Promise.all([query.refetch(),history.refetch()]));
 }),current=query.data?.revision,id=current?.id??"NONE",snapshot=current?.payload.platformPrices;
 const input=draft??{expectedRevision:id,prices:(snapshot?.prices??[]).map(withoutPublicationVersion),fx:(snapshot?.fx??[]).map(withoutPublicationVersion)};
 const draftStale=draft!==null&&draft.expectedRevision!==id;
 const edit=(next:PriceInput)=>{setPreview(null);setDraft(next);setFieldError("")};
 async function validate(){if(state.locked||!canPublish||query.isError||!query.data||draftStale)return;const parsed=priceInput.safeParse(input);if(!parsed.success){setFieldError("请补齐官网来源、核验及生效时间、币种、准确型号与价格。");return}await state.run(async signal=>{const value=await modelRouteRequest(scope,"/platform-prices/validate",pricePreview,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(parsed.data),signal});if(state.active.current)setPreview({input:parsed.data,value})})}
 const interval={sourceUrl:"",verifiedAt:"",effectiveFrom:"",effectiveUntil:""},zeroRates={inputText:0,cachedInputText:0,inputImage:0,cachedInputImage:0,outputText:0,outputImage:0};
 const previewHeading = useRef<HTMLHeadingElement>(null);
 useEffect(() => { if (preview) previewHeading.current?.focus(); }, [preview]);
 return <>
 {!unconfirmed&&<>
 <Card className={styles.panel}>
   <div className={styles.sectionHeader}>
     <div><h2>模型参考价与人民币汇率</h2><p>参考价与汇率用于换算企业 AI 点数。已到期的参考配置需要更新。</p></div>
     <Button variant="outline" disabled={state.busy} onClick={()=>{void query.refetch();void automatic.refetch()}}>重新读取参考价与汇率</Button>
   </div>
   {query.isPending ? <p role="status">正在读取当前版本…</p> : query.isError ? <p role="alert">当前参考价格读取失败，暂不可发布。</p> : <p>当前版本：{current?.id ?? "暂无"}</p>}
   <section className={styles.notice} aria-label="美元汇率自动更新">
   <div className={styles.sectionHeader}>
     <div><h3>美元汇率自动更新</h3><p>美元自动汇率每周更新一次，周内使用同一参考价。</p></div>
   {canPublish&&automatic.data?.state==="PREPARED"&&fxStartState==="ready"?<Button onClick={()=>void startFX()}>启用每周自动更新</Button>:null}
   </div>
   {fxStartState==="pending"?<p role="status">正在启用自动更新。</p>:null}{fxStartState==="unknown"?<p role="alert">启用结果待核实，请勿重复启用；请先核对自动更新状态。</p>:null}{fxStartState==="started"?<p role="status">已启动自动更新，正在读取状态。</p>:null}<p>自动更新：{automatic.isPending?"正在读取":automatic.isError?"未启用或状态未知":({PREPARED:"准备就绪，等待启用",WAITING:"等待下一次周更新",CHECKING:"正在检查",SOURCE_UNAVAILABLE:"数据源不可用",NOT_READY:"暂不能更新，请检查汇率来源及配置",USD_CONFLICT:"人工美元汇率发生冲突，已停止自动发布并需核对",RETRYING:"版本冲突，正在重新检查",PUBLISHING:"正在发布",CHECKED:"上次检查完成",PUBLISHED:"上次已发布",UNKNOWN:"发布结果待核实",DRAINING:"正在停用",QUIESCED:"已停止自动发布"} as Record<string,string>)[automatic.data.state]??"尚无检查结果"}{automatic.data?.checkedAt&&!automatic.data.checkedAt.startsWith("0001-")?` · ${automatic.data.checkedAt}`:""}</p>{snapshot?.fx.filter(row=>row.currency==="USD").map(row=><p key={row.version??row.effectiveFrom}>美元公告 {row.rateDate}：1 美元 = {formatCurrencyMicros(row.cnyMicrosPerCurrencyUnit)} 元人民币，有效至 {row.effectiveUntil}{Date.parse(row.effectiveUntil)<=query.dataUpdatedAt?"（读取时已过期）":""}</p>)}
   </section>
   {!canPublish && <p>当前可以查看参考配置，发布需平台维护权限。</p>}
   {draftStale?<div role="alert"><p>草稿基于旧版本，当前价格或汇率已有变化。请先核对并载入当前版本。</p><Button variant="outline" disabled={state.locked} onClick={()=>{setDraft(null);setPreview(null);setFieldError("")}}>放弃草稿并载入当前版本</Button></div>:null}<fieldset disabled={state.locked||!canPublish||query.isError||query.isPending||draftStale} className={styles.form}>
 <section className={styles.editorGroup} aria-label="模型参考价">
   <h3>模型参考价</h3>
   {input.prices.length === 0 && <p>暂无模型参考价。</p>}
   {input.prices.map((price,index)=><section className={styles.editorRecord} key={index}><h4>{price.model || ("参考价目 " + (index + 1))}</h4><div className={styles.grid}>{(["id","region","serviceType","channel","model","currency","usageVersion","sourceUrl","verifiedAt","effectiveFrom","effectiveUntil"] as const).map(field=><label className={styles.field} key={field}><span>{{id:"价目标识",region:"适用地区",serviceType:"服务类型（standard-sync / image-sync）",channel:"准确渠道",model:"准确模型版本",currency:"官网币种",usageVersion:"已准入用量映射",sourceUrl:"参考价格来源",verifiedAt:"核验时间（含时区）",effectiveFrom:"生效时间（含时区）",effectiveUntil:"失效时间（含时区）"}[field]}</span><input value={price[field]} onChange={event=>edit({...input,prices:input.prices.map((p,i)=>i===index?{...p,[field]:event.target.value}:p)})}/></label>)}<label className={styles.field}>模型厂商<select value={price.provider} onChange={event=>edit({...input,prices:input.prices.map((p,i)=>i===index?{...p,provider:event.target.value as typeof p.provider}:p)})}>{publishedModelManufacturers.map(provider=><option key={provider}>{provider}</option>)}</select></label></div><label className={styles.checkbox}><span><input type="checkbox" checked={!!price.image} onChange={event=>edit({...input,prices:input.prices.map((p,i)=>i===index?{...p,image:event.target.checked?{width:1024,height:1024,quality:"auto"}:undefined}:p)})}/>限定图片价格适用参数</span></label>{price.image?<div className={styles.grid}>{(["width","height","quality"] as const).map(field=><label className={styles.field} key={field}><span>{{width:"宽度（像素）",height:"高度（像素）",quality:"质量档"}[field]}</span><input value={price.image![field]} onChange={event=>edit({...input,prices:input.prices.map((p,i)=>i===index?{...p,image:{...p.image!,[field]:field==="quality"?event.target.value:/^[1-9][0-9]{0,3}$/.test(event.target.value)?Number(event.target.value):0}}:p)})}/></label>)}</div>:null}{price.image&&price.usageVersion==="reference-images-totals-v1"?<ImageReferenceEstimate value={price.image.referenceEstimate} onChange={referenceEstimate=>edit({...input,prices:input.prices.map((p,i)=>i===index?{...p,image:{...p.image!,referenceEstimate}}:p)})}/>:null}<div className={styles.two}>{Object.entries(rateLabels).map(([field,label])=><CurrencyField key={field} label={label+"（参考原币）"} value={price.rates[field as keyof typeof price.rates]??0} change={value=>edit({...input,prices:input.prices.map((p,i)=>i===index?{...p,rates:{...p.rates,[field]:value}}:p)})}/>)}</div><Button variant="ghost" onClick={()=>edit({...input,prices:input.prices.filter((_,i)=>i!==index)})}>删除此价目</Button></section>)}
 <Button variant="outline" disabled={input.prices.length>=32} onClick={()=>edit({...input,prices:[...input.prices,{id:"",region:"",serviceType:"standard-sync",provider:"openai",channel:"openai-direct",model:"",currency:"USD",usageVersion:"",rates:zeroRates,...interval}]})}>添加参考价目</Button>
 </section>
 <section className={styles.editorGroup} aria-label="人民币汇率">
   <h3>人民币汇率</h3>
   {input.fx.length === 0 && <p>暂无人民币汇率。</p>}
   {input.fx.map((fx,index)=><section className={styles.editorRecord} key={index}><h4>{fx.currency || ("汇率 " + (index + 1))}</h4><div className={styles.grid}>{(["currency","rateDate","sourceUrl","verifiedAt","effectiveFrom","effectiveUntil"] as const).map(field=><label className={styles.field} key={field}><span>{{currency:"原币种（人民币价目无需汇率）",rateDate:"中间价日期 YYYY-MM-DD",sourceUrl:"外汇交易中心来源",verifiedAt:"核验时间（含时区）",effectiveFrom:"生效时间（含时区）",effectiveUntil:"失效时间（含时区）"}[field]}</span><input value={fx[field]} onChange={event=>edit({...input,fx:input.fx.map((f,i)=>i===index?{...f,[field]:event.target.value}:f)})}/></label>)}<CurrencyField label="1 单位原币兑人民币" value={fx.cnyMicrosPerCurrencyUnit} change={value=>edit({...input,fx:input.fx.map((f,i)=>i===index?{...f,cnyMicrosPerCurrencyUnit:value}:f)})}/></div><Button variant="ghost" onClick={()=>edit({...input,fx:input.fx.filter((_,i)=>i!==index)})}>删除此汇率</Button></section>)}
 <Button variant="outline" disabled={input.fx.length>=8} onClick={()=>edit({...input,fx:[...input.fx,{currency:"USD",rateDate:"",cnyMicrosPerCurrencyUnit:0,...interval}]})}>添加汇率</Button>
 </section>
 <div className={styles.actions}><Button onClick={()=>void validate()}>校验参考价与汇率</Button></div>
 </fieldset>{fieldError?<p role="alert">{fieldError}</p>:null}</Card>
 {preview?<Card className={styles.panel}><h2 ref={previewHeading} tabIndex={-1}>确认发布参考价与汇率</h2>{preview.value.snapshot.prices.map((price,index)=><div key={index}><strong>{price.model} · {price.channel}</strong><p>{price.sourceUrl} · {price.effectiveFrom} 至 {price.effectiveUntil}</p><RateSummary rates={price.rates} currency={price.currency}/></div>)}<p>新任务按适用参考价估算。已发送任务和待确认费用仍按提交时的价格处理。</p><Button disabled={state.locked||!canPublish||query.isError||preview.input.expectedRevision!==id} onClick={()=>void state.publish(preview.input)}>确认发布参考价与汇率</Button></Card>:null}
 <History rows={history.isError?[]:history.data?.revisions??[]} current={id} disabled={state.locked||!canPublish||query.isError} loading={history.isPending} failed={history.isError} retry={()=>void history.refetch()} select={row=>{const value=row.payload.platformPrices;if(value)edit({expectedRevision:id,prices:value.prices.map(withoutPublicationVersion),fx:value.fx.map(withoutPublicationVersion)})}}/>
 </>}
 <PublishStatus state={state} disabled={disabled}/>
 </>;
}
function CurrencyField({label,value,change}:{label:string;value:number;change:(value:number)=>void}){
 const [draft,setDraft]=useState<{text:string;value:number;invalid:boolean}|null>(null);
 // A history selection or a deleted sibling can replace the parent value while
 // React reuses this field. Never keep showing text for a different price fact.
 const matching=draft?.value===value,invalid=matching&&draft.invalid;
 return <label className={styles.field}><span>{label}</span><input inputMode="decimal" value={matching?draft.text:formatCurrencyMicros(value)} aria-invalid={invalid} onChange={event=>{const text=event.target.value;try{const next=parseCurrencyMicros(text);setDraft({text,value:next,invalid:false});change(next)}catch{setDraft({text,value:-1,invalid:true});change(-1)}}}/>{invalid?<small>须为非负金额，最多六位小数，且不超过准入上界。</small>:null}</label>
}
