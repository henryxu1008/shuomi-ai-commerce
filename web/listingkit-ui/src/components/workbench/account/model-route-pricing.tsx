"use client";
import Link from "next/link";
import {useEffect,useRef,useState,type ReactNode} from "react";
import {useQuery} from "@tanstack/react-query";
import {z} from "zod";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {useResourcePending} from "../resources/resource-pending";
import {ModelRouteError,modelRouteRequest,type ModelRouteScope} from "@/lib/api/model-route-pricing";
import {routeInput,routeCurrent,routeBindings,routeHistory,routePreview,routeReceipt,routeCapabilities,routeCapabilityLabels,routeRateError,type RouteInput,type RoutePayload,type RouteRevision,type RouteBinding} from "@/lib/contracts/model-route-pricing";
import styles from "./model-route-pricing.module.css";

const pagePath="/workbench/account/organization/model-route-pricing";
const impact="未发送的旧确认将阻断并要求重新准备；多步任务下次调用遇配置变化会停止。已通过最终发送门禁的请求按原版本完成与结算；UNKNOWN、预留、已有结果与 Review 保留。";
const pendingSchema=z.strictObject({key:z.uuid(),input:routeInput});
function message(e:unknown){const code=e instanceof ModelRouteError?e.code:"UNAVAILABLE";return ({INVALID:"费率或绑定不符合当前限制，请检查字段后重新校验。",CONFLICT:"预览已过期。保留编辑，读取当前后重新校验。",NEEDS_CONFIGURATION:"凭据版本已变化或绑定不可用，请刷新合格绑定后重新校验。",FORBIDDEN:"无平台维护权限或企业访问已撤销。",IDENTITY_CONTEXT_CHANGED:"登录身份已变化，请重新打开页面。",ORGANIZATION_CONTEXT_CHANGED:"当前企业已变化，旧表单不可提交。",OUTCOME_UNKNOWN:"发布结果待核实。保留原请求键，查询本次发布回执；未确认前不能再次发布。"} as Record<string,string>)[code]??"配置服务暂不可用。编辑内容保留，请稍后重试。"}
export function ModelRoutePricingEntry({scope}:{scope:ModelRouteScope}){
 const c=useWorkbenchContext(),allowed=c.permissions.includes("listingkit.platform_admin");
 const q=useQuery({queryKey:["model-routes-entry",scope],queryFn:({signal})=>modelRouteRequest(scope,"",routeCapabilities,{signal}),enabled:allowed,retry:false,gcTime:0});
 if(!allowed||!q.data?.capabilities.length)return null;
 return <Card className={styles.panel}><h2>平台维护</h2><p>维护当前企业已接线用途的模型绑定与费率。</p><Button asChild variant="outline"><Link href={pagePath} prefetch={false}>模型与费率</Link></Button></Card>;
}
export function ModelRoutePricingPage({expectedUserId}:{expectedUserId:string}){
 const c=useWorkbenchContext(),org=c.effectiveOrganization;
 const allowed=c.permissions.includes("listingkit.platform_admin");
 return <ConsolePage title="模型与费率" description="维护已准入模型的当前路由和消费费率" breadcrumbs={[{label:"我的账户",href:"/workbench/account"},{label:"企业空间",href:"/workbench/account/organization"},{label:"平台维护"},{label:"模型与费率"}]}>
 {c.isLoading||c.isSwitching?<ConsoleState kind="loading" title="正在确认当前上下文"/>:!allowed||!org||c.user?.id!==expectedUserId||c.error||c.blockingError||c.selectionRequired?<ConsoleState kind="error" title="无平台维护权限或当前企业不可用"/>:<ScopedRoutes key={JSON.stringify([expectedUserId,org.id,c.permissions,c.roles])} scope={{userId:expectedUserId,organizationId:org.id}} name={org.name} canPublish={c.permissions.includes("listingkit.admin.write")}/>}
 </ConsolePage>;
}
function ScopedRoutes({scope,name,canPublish}:{scope:ModelRouteScope;name:string;canPublish:boolean}){
 const q=useQuery({queryKey:["model-route-capabilities",scope],queryFn:({signal})=>modelRouteRequest(scope,"",routeCapabilities,{signal}),retry:false,gcTime:0}),[selected,setSelected]=useState("");
 if(q.isPending)return <ConsoleState kind="loading" title="正在读取已接线用途"/>;
 if(q.isError)return <ConsoleState kind="error" title="配置服务未接线或暂不可用">{message(q.error)}</ConsoleState>;
 if(!q.data.capabilities.length)return <ConsoleState kind="unavailable" title="当前企业暂无已接线用途">请通过原运维入口准备合格绑定与运行依赖。</ConsoleState>;
 const units=q.data.capabilities,unit=units.find(u=>`${u.family}/${u.capability}`===selected)??units[0];
 return <><Card className={styles.panel}><strong>当前企业：{name} · {scope.organizationId}</strong><p>维护身份：{scope.userId} · 平台维护者</p><p>各用途分别发布；配置校验不会调用模型。供应商成本、产品 AI 点数与充值售价分别管理。</p></Card><div className={styles.tabs} aria-label="配置用途">{units.map(u=><Button key={u.capability} variant={u===unit?"secondary":"outline"} onClick={()=>setSelected(`${u.family}/${u.capability}`)}>{routeCapabilityLabels[u.capability]}</Button>)}</div><RoutePricingEditor key={unit.capability} scope={scope} family={unit.family} capability={unit.capability} canPublish={canPublish}/></>;
}
export function RoutePricingEditor({scope,family,capability,canPublish=true}:{scope:ModelRouteScope;family:"text"|"image";capability:string;canPublish?:boolean}){
 const path=`/${family}/${capability}`,q=useQuery({queryKey:["model-route-editor",scope,path],queryFn:async({signal})=>{const [current,bindings,history]=await Promise.all([modelRouteRequest(scope,path,routeCurrent,{signal}),modelRouteRequest(scope,path+"/bindings",routeBindings,{signal}),modelRouteRequest(scope,path+"/history",routeHistory,{signal})]);return {current,bindings:bindings.bindings,history}},retry:false,gcTime:0,refetchOnWindowFocus:false});
 if(q.isPending)return <ConsoleState kind="loading" title="正在读取当前配置"/>;
 if(q.isError&&!q.data)return <UnavailableEditor scope={scope} path={path} family={family} capability={capability} refresh={()=>q.refetch().then(()=>{})}/>;
 return <LoadedEditor scope={scope} family={family} capability={capability} path={path} data={q.data!} readFailed={q.isError} refresh={()=>q.refetch().then(()=>{})} canPublish={canPublish}/>;
}
type EditorData={current:z.infer<typeof routeCurrent>;bindings:RouteBinding[];history:z.infer<typeof routeHistory>};
function LoadedEditor({scope,family,capability,path,data,readFailed,refresh,canPublish}:{scope:ModelRouteScope;family:"text"|"image";capability:string;path:string;data:EditorData;readFailed:boolean;refresh:()=>Promise<void>;canPublish:boolean}){
 const initial=data.current.revision?.payload;
 const [fieldErrors,setFieldErrors]=useState<Record<string,string>>({});
 const [bindingId,setBindingId]=useState(initial?.bindingId??""),[rates,setRates]=useState<RouteInput["rates"]>(initial?.rates??(family==="image"?{pointsPerImage:""}:{inputCost:"",outputCost:"",inputPoints:"",outputPoints:""})),[rollback,setRollback]=useState<RouteRevision|null>(null),[preview,setPreview]=useState<{input:RouteInput;payload:RoutePayload}|null>(null),[receipt,setReceipt]=useState<RouteRevision|null>(null),[failure,setFailure]=useState<unknown>(),[busy,setBusy]=useState(false),[history,setHistory]=useState<RouteRevision[]|null>(null),[cursor,setCursor]=useState<string|null>(null);
 const pending=useResourcePending({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId},["model-route-pricing",family,capability],pendingSchema,{storage:"session",maxLength:16384});
 const c=useWorkbenchContext(),active=useRef(true),flight=useRef(false),controller=useRef<AbortController|null>(null);
 useEffect(()=>{active.current=true;return()=>{active.current=false;controller.current?.abort()}},[]);
 const guard=c.registerOrganizationSwitchGuard;
 useEffect(()=>guard(()=>!flight.current&&!pending.command&&!pending.error),[guard,pending.command,pending.error]);
 const locked=busy||!!pending.command||!pending.ready||!!pending.error,binding=data.bindings.find(b=>b.id===bindingId),currentId=readFailed?"无法读取":data.current.revision?.id??"NONE";
 const rows=history??data.history.revisions,next=cursor??data.history.nextCursor;
 const edit=()=>{setPreview(null);setRollback(null);setFailure(undefined)};
 async function run(operation:(signal:AbortSignal)=>Promise<void>){if(flight.current)return;flight.current=true;setBusy(true);setFailure(undefined);const a=new AbortController();controller.current=a;try{await operation(a.signal)}catch(e){if(active.current)setFailure(e)}finally{flight.current=false;if(active.current)setBusy(false)}}
 async function validate(old?:RouteRevision){
 if(readFailed){setFailure(new ModelRouteError("UNAVAILABLE"));return}
  const selected=old?data.bindings.find(b=>b.id===old.payload.bindingId&&b.credentialVersion===old.payload.credentialVersion):binding;
  const input:RouteInput={expectedRevision:currentId,bindingId:old?.payload.bindingId??bindingId,credentialVersion:old?.payload.credentialVersion??selected?.credentialVersion??"",rates:old?.payload.rates??rates,...(old?{rollbackFrom:old.id}:rollback?{rollbackFrom:rollback.id}:{})};
  const errors=Object.fromEntries(Object.entries(input.rates).map(([k,v])=>[k,routeRateError(k,v)]).filter(([,v])=>!!v)) as Record<string,string>;setFieldErrors(errors);
 const parsed=routeInput.safeParse(input);if(!parsed.success){setFailure(new ModelRouteError("INVALID",400));return}
  await run(async signal=>{const result=await modelRouteRequest(scope,path+"/validate",routePreview,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(parsed.data),signal});if(active.current){setPreview({input:{...parsed.data,rates:result.payload.rates},payload:result.payload});if(old){setBindingId(old.payload.bindingId);setRates(old.payload.rates);setRollback(old)}}});
 }
 async function publish(){if(!preview||locked||readFailed||preview.input.expectedRevision!==currentId)return;const intent={key:crypto.randomUUID(),input:preview.input};await run(async signal=>{
  pending.persist(intent);
  try{const r=await modelRouteRequest(scope,path+"/publish",routeReceipt,{method:"POST",headers:{"Content-Type":"application/json","Idempotency-Key":intent.key},body:JSON.stringify(intent.input),signal});if(active.current){pending.clear(intent);setReceipt(r.receipt);setPreview(null);await refresh()}}
  catch(e){if(e instanceof ModelRouteError&&e.status<500&&e.code!=="OUTCOME_UNKNOWN")pending.clear(intent);throw e}
 })}
 async function verify(){if(!pending.command)return;const intent=pending.command;await run(async signal=>{const r=await modelRouteRequest(scope,path+"/receipts/"+intent.key,routeReceipt,{signal});if(active.current){pending.clear(intent);setReceipt(r.receipt);setPreview(null);await refresh()}})}
 const changeRate=(field:string,value:string)=>{edit();setFieldErrors(e=>({...e,[field]:""}));setRates(r=>({...r,[field]:value}))};
 return <div className={styles.stack}>
 <Card className={styles.panel}><h2>{routeCapabilityLabels[capability]}</h2><p>{family==="image"?"单一共享图片配置：影响既有商品图片与 artwork 用途，不按单个任务切换。":"模型绑定和费率按业务用途分别发布。"}</p><p>当前版本：{currentId} · {readFailed?"读取失败，当前值待核实":data.current.status==="READY"?"已发布":data.current.status==="UNCONFIGURED"?"尚未发布，暂不可执行":"凭据或绑定不可用，暂不可执行"}</p>
 {!readFailed&&initial?<div><h3>当前已发布配置</h3><PayloadSummary payload={initial}/></div>:null}<h3>未发布编辑</h3>
 <fieldset disabled={locked} className={styles.form}><legend className="sr-only">模型绑定与费率</legend><div className={styles.grid}><Field label="业务用途"><input readOnly value={routeCapabilityLabels[capability]}/></Field><Field label={family==="image"?"已准入图片绑定":"已准入模型绑定"} hint="仅显示当前企业已准入且凭据有效的绑定"><select value={bindingId} onChange={e=>{edit();setBindingId(e.target.value)}}><option value="">请选择绑定</option>{!binding&&bindingId?<option value={bindingId}>{bindingId}（不可用）</option>:null}{data.bindings.map(b=><option key={b.id} value={b.id}>{b.id} · {b.model}</option>)}</select></Field><Field label="模型与协议" hint={binding?.credentialVersion??"先由原运维入口准备；此处不录入 Key"}><input readOnly value={binding?`${binding.provider} / ${binding.model} / ${binding.protocol}`:"无合格绑定"}/></Field></div>
 {family==="image"?<><Field label="每张图片 AI 点数" error={fieldErrors.pointsPerImage} hint={`正整数；当前上界 ${binding?.maximumPointsPerImage??"未提供"} 点 / 张`}><input aria-invalid={!!fieldErrors.pointsPerImage} inputMode="numeric" value={"pointsPerImage" in rates?rates.pointsPerImage:""} onChange={e=>changeRate("pointsPerImage",e.target.value)}/></Field><Note title="供应商成本：尚未接入">这里只维护每张图片的产品 AI 点数；不推造供应商金额。原计划、预留和已完成图片继续保留。</Note></>:<><h3>供应商估算成本 · {binding?.currency??"原币种"} / 百万 Token</h3><div className={styles.two}>{(["inputCost","outputCost"] as const).map((field,i)=><Field key={field} error={fieldErrors[field]} label={i?"输出估算成本":"输入估算成本"} hint="大于 0，最多 6 位小数"><input aria-invalid={!!fieldErrors[field]} inputMode="decimal" value={field in rates?(rates as Record<string,string>)[field]:""} onChange={e=>changeRate(field,e.target.value)}/></Field>)}</div><h3>产品 AI 点数 / 百万 Token</h3><div className={styles.two}>{(["inputPoints","outputPoints"] as const).map((field,i)=><Field key={field} error={fieldErrors[field]} label={i?"输出 AI 点数":"输入 AI 点数"} hint="大于 0 的整数"><input aria-invalid={!!fieldErrors[field]} inputMode="numeric" value={field in rates?(rates as Record<string,string>)[field]:""} onChange={e=>changeRate(field,e.target.value)}/></Field>)}</div><p>只读限制：输入 {binding?.maximumInputBytes??"—"} bytes / 输出 {binding?.maximumOutputBytes??"—"} bytes；Token 上界 {binding?.maximumPromptTokens??"—"} / {binding?.maximumCompletionTokens??"—"}；期限 {binding?.deadlineSeconds??"—"} 秒；单次成本上界 {binding?.maximumCostMicros??"—"} micros。</p></>}
 <div className={styles.actions}><Button variant="outline" disabled={!binding||readFailed} onClick={()=>void validate()}>校验配置并预览</Button><Button variant="ghost" onClick={()=>{edit();setBindingId(initial?.bindingId??"");setRates(initial?.rates??(family==="image"?{pointsPerImage:""}:{inputCost:"",outputCost:"",inputPoints:"",outputPoints:""}))}}>放弃编辑</Button></div></fieldset>
 {!data.bindings.length?<Note title="暂不可发布">没有合格绑定。请先由原运维入口准备凭据和已准入描述。</Note>:null}
 </Card>
 {readFailed?<p role="alert" className={styles.error}>当前配置读取失败。编辑与原回执保留，请重新读取后校验。</p>:null}
 {failure?<p role="alert" className={styles.error}>{message(failure)}</p>:null}
 {pending.command||pending.error?<Card className={styles.panel}><h2>发布结果待核实</h2><p>保留原请求键 {pending.command?.key??"（记录读取失败）"}。未确认前不能换键发布；查询失败或未找到回执仍保持待核实。</p><Button disabled={busy||!pending.command} onClick={()=>void verify()}>查询本次发布结果</Button></Card>:null}
 {preview?<Card className={styles.panel}><h2>确认发布这个用途的新配置</h2><p>当前企业：{scope.organizationId} · 用途：{routeCapabilityLabels[capability]} · 基于 {preview.input.expectedRevision}</p><div className={styles.two}><div><h3>当前 {currentId}</h3>{initial?<PayloadSummary payload={initial}/>:<p>尚未发布</p>}</div><div><h3>待发布的新版本{preview.input.rollbackFrom?"（历史恢复）":""}</h3><PayloadSummary payload={preview.payload}/></div></div><Note title="对已有任务的影响">{impact}</Note><p>发布时再次核验权限、企业、凭据版本和当前版本。不会静默替换旧任务。</p>{preview.input.expectedRevision!==currentId?<p role="alert">当前版本已有更新，请重新校验。</p>:null}<div className={styles.actions}><Button disabled={locked||readFailed||!canPublish||preview.input.expectedRevision!==currentId} onClick={()=>void publish()}>确认发布</Button><Button variant="outline" disabled={locked} onClick={()=>setPreview(null)}>返回编辑</Button></div></Card>:null}
 {receipt?<Card className={styles.panel} role="status"><h2>发布回执</h2><p>本次发布已记录：{receipt.id} · {receipt.actorId} · {receipt.publishedAt}</p><PayloadSummary payload={receipt.payload}/><p>发布版本：{receipt.id} / 实际当前版本：{currentId}</p>{!readFailed&&receipt.id!==currentId?<p>本次已发布，当前已有更新。旧回执不代表当前配置。</p>:null}</Card>:null}
 <Card className={styles.panel}><h2>版本历史</h2><Button variant="outline" disabled={busy} onClick={()=>void run(async()=>{setHistory(null);setCursor(null);await refresh()})}>读取当前与历史</Button>{!rows.length?<p>暂无已发布版本。</p>:rows.map(r=><article key={r.id} className={styles.history}><strong>{r.id}{r.id===currentId?" · 当前指针":""}</strong><p>{r.actorId} · {r.publishedAt}</p><PayloadSummary payload={r.payload}/><Button variant="outline" disabled={locked||readFailed} onClick={()=>void validate(r)}>以历史配置创建新预览</Button></article>)}{next?<Button disabled={busy} onClick={()=>void run(async signal=>{const h=await modelRouteRequest(scope,path+"/history?cursor="+encodeURIComponent(next),routeHistory,{signal});if(active.current){setHistory([...rows,...h.revisions]);setCursor(h.nextCursor)}})}>更多版本</Button>:null}<p>历史恢复也生成新版本，并重新验证当前凭据与限制；旧报价不会复活。</p></Card>
 </div>;
}
function Field({label,hint,error,children}:{label:string;hint?:string;error?:string;children:ReactNode}){return <label className={styles.field}><strong>{label}</strong>{children}{hint?<small>{hint}</small>:null}{error?<small className={styles.error}>{error}</small>:null}</label>}
function Note({title,children}:{title:string;children:ReactNode}){return <aside className={styles.note}><strong>{title}</strong><p>{children}</p></aside>}
function PayloadSummary({payload}:{payload:RoutePayload}){return <div className={styles.summary}><p>{payload.bindingId} / {payload.credentialVersion}</p>{"image" in payload?<><p>产品 AI 点数：{payload.rates.pointsPerImage} / 张</p><p>供应商成本：尚未接入</p></>:<><p>{payload.text.ProviderID} / {payload.text.ModelID} / {payload.text.AdapterKind}</p><p>估算成本：输入 {payload.rates.inputCost} / 输出 {payload.rates.outputCost} {payload.text.Currency} / 百万 Token</p><p>AI 点数：输入 {payload.rates.inputPoints} / 输出 {payload.rates.outputPoints} / 百万 Token</p></>}</div>}

function UnavailableEditor({scope,path,family,capability,refresh}:{scope:ModelRouteScope;path:string;family:string;capability:string;refresh:()=>Promise<void>}){
 const pending=useResourcePending({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId},["model-route-pricing",family,capability],pendingSchema,{storage:"session",maxLength:16384});
 const[busy,setBusy]=useState(false),[failure,setFailure]=useState<unknown>(),[receipt,setReceipt]=useState<RouteRevision|null>(null);
 return <ConsoleState kind="error" title="当前配置读取失败"><p>本次未取得当前配置，暂不可发布。</p>{pending.command||pending.error?<><p>发布结果待核实。保留原请求键 {pending.command?.key??"（记录读取失败）"}，读取失败不代表发布失败。</p><Button disabled={busy||!pending.command} onClick={async()=>{const intent=pending.command;if(!intent)return;setBusy(true);try{const r=await modelRouteRequest(scope,path+"/receipts/"+intent.key,routeReceipt);pending.clear(intent);setReceipt(r.receipt)}catch(e){setFailure(e)}finally{setBusy(false)}}}>查询本次发布结果</Button></>:null}{receipt?<p>本次发布已记录：{receipt.id}。实际当前配置仍无法读取。</p>:null}{failure?<p>{message(failure)}</p>:null}<Button disabled={busy} onClick={()=>void refresh()}>重新读取</Button></ConsoleState>
}
