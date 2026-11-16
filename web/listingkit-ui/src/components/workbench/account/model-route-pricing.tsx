"use client";
import {ImageReferenceEstimate} from "./image-reference-estimate";
import Link from "next/link";
import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {z} from "zod";
import {automaticFXStatus} from "@/lib/contracts/model-route-pricing";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {useResourcePending} from "../resources/resource-pending";
import {ModelRouteError,modelRouteRequest,type ModelRouteScope} from "@/lib/api/model-route-pricing";
import {publishedModelManufacturers,canonicalPriceFacts,withoutPublicationVersion,officialRates,routeInput,priceInput,pricePreview,routeOptions,routeCurrent,routeHistory,routePreview,routeReceipt,routeCapabilities,modelLabels,formatCurrencyMicros,parseCurrencyMicros,type RouteInput,type RouteRevision,type ModelOption,type PriceInput} from "@/lib/contracts/model-route-pricing";
import styles from "./model-route-pricing.module.css";
import {credentialConnections} from "@/lib/contracts/channel-credentials";
import {ChannelCredentialsManager} from "./channel-credentials";

const pagePath="/workbench/account/organization/model-route-pricing";
const defaults:RouteInput["selection"]={schemaVersion:1,unitVersion:"ai-micropoint-v1",defaultChannel:"openai-direct",text:{id:"openai-mini",version:"approved-default-v1"},image:{id:"openai-flare",version:"approved-default-v1"},multimodal:{id:"openai-mini",version:"approved-default-v1"}};
const categories=["text","image-generation","multimodal"] as const;
const keyFor={text:"text","image-generation":"image",multimodal:"multimodal"} as const;
const reasonLabels:Record<string,string>={POINT_BUDGET_EXCEEDED:"费用估算超出本次分析的点数预算",POINT_BUDGET_MISSING:"尚未配置本次分析的点数预算",METERING_MAPPING_NOT_ADMITTED:"该协议的完整官网计费用量映射尚未准入",BINDING_NOT_ADMITTED:"组织模型绑定尚未准入",PROTOCOL_AND_COST_BOUND_MISSING:"图片调用尚未接通",VISION_CONSUMER_NOT_IMPLEMENTED:"图片理解消费链路尚未完成",OFFICIAL_PRICE_MISSING:"缺已核验的参考价目",PRICE_OR_FX_NOT_APPLICABLE:"参考价或汇率缺失、过期或不适用",CREDENTIAL_NOT_ADMITTED:"组织凭据版本尚未准入",CREDENTIAL_READER_MISSING:"凭据元数据读取尚未接线",COMPLETE_BILLING_USAGE_MISSING:"响应缺少完整计费用量",COST_BOUND_MISSING:"缺保守费用上界",PRICE_APPLICABILITY_MISSING:"参考价格的地区、服务类型或图片参数不匹配",OPEN_CONSUMER_MISSING:"部分已开放用途尚未完成准入",CONSUMER_NOT_QUALIFIED:"消费链路尚未满足准入",CONSUMER_BINDING_MISMATCH:"用途与所选模型的绑定不一致",DEPENDENCY_MISSING:"缺少必要依赖"};
const pendingSchema=z.strictObject({key:z.uuid(),input:z.union([routeInput,priceInput])});
function message(error:unknown){const code=error instanceof ModelRouteError?error.code:"UNAVAILABLE";return ({INVALID:"请检查填写内容，参考价与汇率最多保留六位小数。",CONFLICT:"当前版本已有更新，请重新读取后校验。",NEEDS_CONFIGURATION:"三类模型尚未全部满足准入，请查看具体缺口。",FORBIDDEN:"无此操作权限，或企业访问已撤销。",PERMISSION_DENIED:"无此操作权限，请确认权限后重试。",AUTHENTICATION_REQUIRED:"登录已失效，请重新登录。",ORGANIZATION_ACCESS_DENIED:"无当前企业访问权限。",ORGANIZATION_ACCESS_REVOKED:"当前企业访问已撤销。",ORGANIZATION_SUSPENDED:"当前企业暂不可用。",ORGANIZATION_SELECTION_REQUIRED:"请先选择当前企业。",OUTCOME_UNKNOWN:"发布结果待核实，请使用原请求键查询回执。",ORGANIZATION_CONTEXT_CHANGED:"当前企业已变化，请重新打开页面。",IDENTITY_CONTEXT_CHANGED:"登录身份已变化，请重新打开页面。"} as Record<string,string>)[code]??"配置服务暂不可用，编辑和原回执保留。"}
export function ModelRoutePricingEntry({scope}:{scope:ModelRouteScope}){
 const context=useWorkbenchContext(),configure=context.permissions.includes("workbench.agent.configure"),platform=context.permissions.includes("listingkit.platform_admin"),credentials=context.permissions.includes("workbench.ai.credentials.manage"),allowed=configure||platform||credentials;
 const query=useQuery({queryKey:["model-routes-entry",scope,configure,platform,credentials],queryFn:async({signal})=>configure?(await modelRouteRequest(scope,"",routeCapabilities,{signal})).available:credentials?!!(await modelRouteRequest(scope,"/connections",credentialConnections,{signal})):!!(await modelRouteRequest(scope,"/platform-prices",routeCurrent,{signal})),enabled:allowed,retry:false,gcTime:0});
 if(!allowed||!query.data)return null;
 return <Card className={styles.panel}><h2>{configure?"企业 AI 模型":credentials?"渠道凭据":"参考价格与汇率"}</h2><p>{configure?"配置全体成员共用的文本、图片生成和多模态理解模型。":credentials?"统一录入、轮换和停用本企业多模型共用的渠道 Key。":"维护平台共用的模型参考价目与人民币汇率。"}</p><Button asChild variant="outline"><Link href={pagePath} prefetch={false}>模型与费率</Link></Button></Card>;
}
export function ModelRoutePricingPage({expectedUserId}:{expectedUserId:string}){
 const context=useWorkbenchContext(),org=context.effectiveOrganization,configure=context.permissions.includes("workbench.agent.configure"),platform=context.permissions.includes("listingkit.platform_admin"),credentials=context.permissions.includes("workbench.ai.credentials.manage"),allowed=configure||platform||credentials;
 return <ConsolePage title="模型与费率" description="管理企业使用的 AI 模型、渠道 Key 和参考费率。" breadcrumbs={[{label:"我的账户",href:"/workbench/account"},{label:"企业空间",href:"/workbench/account/organization"},{label:"模型与费率"}]}>
 {context.isLoading||context.isSwitching?<ConsoleState kind="loading" title="正在确认当前企业"/>:!allowed||!org||context.user?.id!==expectedUserId||context.error||context.blockingError||context.selectionRequired?<ConsoleState kind="error" title="无模型配置权限或当前企业不可用"/>:<ModelSettings key={JSON.stringify([expectedUserId,org.id,context.permissions,context.roles])} scope={{userId:expectedUserId,organizationId:org.id}} name={org.name} configure={configure} platform={platform} credentials={credentials} canMaintain={context.permissions.includes("listingkit.admin.write")}/>}
 </ConsolePage>;
}
function ModelSettings({ scope, name, configure, platform, credentials, canMaintain }: { scope: ModelRouteScope; name: string; configure: boolean; platform: boolean; credentials: boolean; canMaintain: boolean }) {
 const [area, setArea] = useState<"enterprise" | "platform-prices" | "credentials">(configure ? "enterprise" : credentials ? "credentials" : "platform-prices");
 const areas = [
   { id: "enterprise", label: "企业模型", available: configure },
   { id: "credentials", label: "渠道凭据", available: credentials },
   { id: "platform-prices", label: "参考价格与汇率", available: platform },
 ] as const;
 const availableAreas = areas.filter(item => item.available);
 return <div className={styles.stack}>
   <div className={styles.workspaceHeader}>
     {availableAreas.length > 1 && <div className={styles.tabs} role="group" aria-label="配置区域">
       {availableAreas.map(item => <Button key={item.id} variant="ghost" aria-pressed={area === item.id} onClick={() => setArea(item.id)}>{item.label}</Button>)}
     </div>}
     <p className={styles.scope}>{area === "platform-prices" ? "适用范围：平台共用" : <>当前企业 <strong>{name}</strong></>}</p>
   </div>
   {area === "enterprise" ? <EnterpriseModelEditor scope={scope} />
     : area === "credentials" ? <ChannelCredentialsManager scope={scope} />
       : <OfficialPriceEditor scope={scope} canPublish={canMaintain} />}
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
function PublishStatus({state}:{state:ReturnType<typeof usePublish>}){return <>{state.failure?<p role="alert" className={styles.error}>{message(state.failure)}</p>:null}{state.pending.command||state.pending.error?<Card className={styles.panel}><h2>发布结果待核实</h2><p>原请求键：{state.pending.command?.key??"记录读取失败"}。查询失败或未找到回执不代表发布失败，不能换键重复发布。</p><Button disabled={state.busy||!state.pending.command} onClick={()=>void state.verify()}>查询本次发布结果</Button></Card>:null}{state.receipt?<Card className={styles.panel} role="status"><h2>发布回执</h2><p>{state.receipt.id} · {state.receipt.publishedAt}</p><p>这是本次发布结果，实际当前版本以重新读取为准。</p></Card>:null}</>}
export function EnterpriseModelEditor({scope}:{scope:ModelRouteScope}){
 const query=useQuery({queryKey:["shared-model-editor",scope],queryFn:async({signal})=>{const[current,options,history]=await Promise.all([modelRouteRequest(scope,"/enterprise",routeCurrent,{signal}),modelRouteRequest(scope,"/enterprise/options",routeOptions,{signal}),modelRouteRequest(scope,"/enterprise/history",routeHistory,{signal})]);return {current,options,history}},retry:false,gcTime:0,refetchOnWindowFocus:false});
 const [selection,setSelection]=useState<RouteInput["selection"]|null>(null),[preview,setPreview]=useState<{input:RouteInput;value:z.infer<typeof routePreview>}|null>(null);
 const state=usePublish(scope,"enterprise",()=>query.refetch());
 const current=query.data?.current.revision,id=current?.id??"NONE",choice=selection??current?.payload.sharedModels??defaults;
 const select=(key:keyof typeof keyFor,value:string)=>{setPreview(null);const model=query.data?.options.models.find(m=>m.category===key&&m.reference.id+"/"+m.reference.version===value);if(model)setSelection({...choice,[keyFor[key]]:model.reference})};
 async function validate(next=choice){if(!query.data||query.isError||state.locked)return;const input=routeInput.parse({expectedRevision:id,selection:next});await state.run(async signal=>{const value=await modelRouteRequest(scope,"/enterprise/validate",routePreview,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(input),signal});if(state.active.current)setPreview({input,value})})}
 const previewHeading = useRef<HTMLHeadingElement>(null);
 useEffect(() => { if (preview) previewHeading.current?.focus(); }, [preview]);
 return <>
   <Card className={styles.panel}>
     <div className={styles.sectionHeader}>
       <div>
         <h2>企业共用模型</h2>
         <p>为企业成员选择文本、图片生成和图片理解模型。</p>
       </div>
       <Button variant="outline" disabled={state.busy} onClick={() => void query.refetch()}>重新读取当前配置</Button>
     </div>
     {query.isPending ? <p role="status">正在读取模型配置…</p>
       : query.isError ? <p role="alert">当前配置读取失败，暂不可发布。</p>
         : <>
           <p>当前版本：{current?.id ?? "暂无"} · {query.data.current.status === "READY" ? "已发布" : query.data.current.status === "UNCONFIGURED" ? "尚未发布" : "选择已保存，部分功能尚未就绪"}</p>
           <fieldset disabled={state.locked} className={styles.form}>
             <label className={`${styles.field} ${styles.channelField}`}>
               <strong>默认渠道</strong>
               <select value={choice.defaultChannel} onChange={event => { setPreview(null); setSelection({...choice, defaultChannel: event.target.value}); }}>
                 {query.data.options.channels.map(channel => <option key={channel}>{channel}</option>)}
               </select>
             </label>
             <div className={styles.modelGrid}>
               {categories.map(category => {
                 const options = query.data.options.models.filter(model => model.category === category);
                 const ref = choice[keyFor[category]];
                 const selected = options.find(model => model.reference.id === ref.id && model.reference.version === ref.version);
                 return <div className={styles.modelCard} key={category}>
                   <label className={styles.field}>
                     <strong>{modelLabels[category]}</strong>
                     <select value={ref.id + "/" + ref.version} onChange={event => select(category, event.target.value)}>
                       {!selected && <option value={ref.id + "/" + ref.version}>{ref.id}（原版本当前不可用）</option>}
                       {options.map(model => <option key={model.reference.id + "/" + model.reference.version} value={model.reference.id + "/" + model.reference.version}>
                         {model.model} · {model.channel}{model.ready ? "" : "（尚未就绪）"}
                       </option>)}
                     </select>
                   </label>
                   {selected && <ModelReadiness model={selected} showModel={false} />}
                 </div>;
               })}
             </div>
             <div className={styles.actions}><Button onClick={() => void validate()}>检查并预览</Button></div>
           </fieldset>
         </>}
   </Card>
   {preview && <Card className={styles.panel}>
     <h2 ref={previewHeading} tabIndex={-1}>确认企业模型配置</h2>
     <div className={styles.modelGrid}>
       {preview.value.preview.models.map(model => <div className={styles.modelCard} key={model.category}><ModelReadiness model={model} /></div>)}
     </div>
     <p>三类模型选择一并保存，使用时分别检查所选模型。发布配置不会发起模型调用或启用尚未就绪的功能。已有回执和待确认费用会保留。</p>
     {!preview.value.preview.ready && <p>部分功能尚未就绪，已就绪的文字分析可独立使用。</p>}
     <div className={styles.actions}>
       <Button disabled={state.locked || query.isError || preview.input.expectedRevision !== id} onClick={() => void state.publish(preview.input)}>确认发布三类模型</Button>
     </div>
   </Card>}
   <PublishStatus state={state}/>
   <History rows={query.data?.history.revisions??[]} current={id} select={row=>{if(row.payload.sharedModels){setSelection(row.payload.sharedModels);setPreview(null)}}} disabled={state.locked||query.isError}/>
 </>;
}

function ModelReadiness({ model, showModel = true }: { model: ModelOption; showModel?: boolean }) {
 return <div className={styles.summary}>
   {showModel && <h3>{modelLabels[model.category]}：{model.model}</h3>}
   <p data-ready={model.ready} className={styles.readiness}>{model.ready ? "模型已就绪" : reasonLabels[model.reason ?? ""] ?? "模型尚未就绪"}</p>
   {model.price ? <details className={styles.rateDetails}>
     <summary>参考价与汇率</summary>
     <RateSummary rates={model.price.rates} currency={model.price.currency} />
     <p>价目版本 {model.price.priceVersion}</p>
     <p>1 {model.price.currency} = {formatCurrencyMicros(model.price.cnyMicrosPerCurrencyUnit)} 元人民币 · 汇率版本 {model.price.fxVersion}</p>
   </details> : <p>暂无适用参考价或汇率。</p>}
 </div>;
}

function History({ rows, current, select, disabled }: { rows: RouteRevision[]; current: string; select: (row: RouteRevision) => void; disabled: boolean }) {
 return <Card className={styles.panel}>
   <h2>最近版本</h2>
   {rows.length ? <>
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
export function OfficialPriceEditor({scope,canPublish}:{scope:ModelRouteScope;canPublish:boolean}){
 const automatic=useQuery({queryKey:["automatic-fx-status",scope],queryFn:({signal})=>modelRouteRequest(scope,"/platform-prices/automatic-fx",automaticFXStatus,{signal}),retry:false,gcTime:0,refetchOnWindowFocus:false});
 const [fxStartState,setFxStartState]=useState<"ready"|"pending"|"unknown"|"started">("ready"),fxStartLock=useRef(false);
 async function startFX(){
  if(fxStartLock.current||!canPublish||fxStartState!=="ready"||automatic.data?.state!=="PREPARED")return;
  fxStartLock.current=true;setFxStartState("pending");
  try{const current=await modelRouteRequest(scope,"/platform-prices/automatic-fx",automaticFXStatus);if(current.state!=="PREPARED"){setFxStartState("unknown");await automatic.refetch();return}await modelRouteRequest(scope,"/platform-prices/automatic-fx/start",z.strictObject({state:z.literal("STARTED")}),{method:"POST",headers:{"Content-Type":"application/json"},body:"{}"});setFxStartState("started");await automatic.refetch()}catch{setFxStartState("unknown")}
 }
 const query=useQuery({queryKey:["official-model-prices",scope],queryFn:async({signal})=>{const[current,history]=await Promise.all([modelRouteRequest(scope,"/platform-prices",routeCurrent,{signal}),modelRouteRequest(scope,"/platform-prices/history",routeHistory,{signal})]);return{current,history}},retry:false,gcTime:0,refetchOnWindowFocus:false,refetchInterval:60000});
 const [draft,setDraft]=useState<PriceInput|null>(null),[preview,setPreview]=useState<{input:PriceInput;value:z.infer<typeof pricePreview>}|null>(null),[fieldError,setFieldError]=useState("");
 const state=usePublish(scope,"platform-prices",()=>query.refetch()),current=query.data?.current.revision,id=current?.id??"NONE",snapshot=current?.payload.platformPrices;
 const input=draft??{expectedRevision:id,prices:(snapshot?.prices??[]).map(withoutPublicationVersion),fx:(snapshot?.fx??[]).map(withoutPublicationVersion)};
 const draftStale=draft!==null&&draft.expectedRevision!==id;
 const edit=(next:PriceInput)=>{setPreview(null);setDraft(next);setFieldError("")};
 async function validate(){if(state.locked||!canPublish||query.isError||!query.data||draftStale)return;const parsed=priceInput.safeParse(input);if(!parsed.success){setFieldError("请补齐官网来源、核验及生效时间、币种、准确型号与价格。");return}await state.run(async signal=>{const value=await modelRouteRequest(scope,"/platform-prices/validate",pricePreview,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(parsed.data),signal});if(state.active.current)setPreview({input:parsed.data,value})})}
 const interval={sourceUrl:"",verifiedAt:"",effectiveFrom:"",effectiveUntil:""},zeroRates={inputText:0,cachedInputText:0,inputImage:0,cachedInputImage:0,outputText:0,outputImage:0};
 const previewHeading = useRef<HTMLHeadingElement>(null);
 useEffect(() => { if (preview) previewHeading.current?.focus(); }, [preview]);
 return <>
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
 {preview?<Card className={styles.panel}><h2 ref={previewHeading} tabIndex={-1}>确认发布参考价与汇率</h2>{preview.value.snapshot.prices.map((price,index)=><div key={index}><strong>{price.model} · {price.channel}</strong><p>{price.sourceUrl} · {price.effectiveFrom} 至 {price.effectiveUntil}</p><RateSummary rates={price.rates} currency={price.currency}/></div>)}<p>发布后新报价读取适用价格；已发送请求按冻结价格处理，UNKNOWN 与预留保留。不会改写历史费用。</p><Button disabled={state.locked||!canPublish||query.isError||preview.input.expectedRevision!==id} onClick={()=>void state.publish(preview.input)}>确认发布参考价与汇率</Button></Card>:null}<PublishStatus state={state}/><History rows={query.data?.history.revisions??[]} current={id} disabled={state.locked||!canPublish||query.isError} select={row=>{const value=row.payload.platformPrices;if(value)edit({expectedRevision:id,prices:value.prices.map(withoutPublicationVersion),fx:value.fx.map(withoutPublicationVersion)})}}/>
 </>;
}
function CurrencyField({label,value,change}:{label:string;value:number;change:(value:number)=>void}){
 const [draft,setDraft]=useState<{text:string;value:number;invalid:boolean}|null>(null);
 // A history selection or a deleted sibling can replace the parent value while
 // React reuses this field. Never keep showing text for a different price fact.
 const matching=draft?.value===value,invalid=matching&&draft.invalid;
 return <label className={styles.field}><span>{label}</span><input inputMode="decimal" value={matching?draft.text:formatCurrencyMicros(value)} aria-invalid={invalid} onChange={event=>{const text=event.target.value;try{const next=parseCurrencyMicros(text);setDraft({text,value:next,invalid:false});change(next)}catch{setDraft({text,value:-1,invalid:true});change(-1)}}}/>{invalid?<small>须为非负金额，最多六位小数，且不超过准入上界。</small>:null}</label>
}
