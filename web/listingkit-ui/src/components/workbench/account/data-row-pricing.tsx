"use client";
import Link from "next/link";
import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {z} from "zod";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {useResourcePending} from "../resources/resource-pending";
import {dataRowPriceInput,dataRowPriceOffer,dataRowPriceReceipt,dataRowPriceHistory,dataRowPriceRequest,type PriceScope} from "@/lib/api/data-row-pricing";
const path="/workbench/account/organization/data-row-pricing";
const intentSchema=z.object({key:z.uuid(),input:dataRowPriceInput}).strict();
const money=(v:string)=>{const n=BigInt(v);return `¥${n/BigInt(100)}.${(n%BigInt(100)).toString().padStart(2,"0")}`};
export function DataRowPricingEntry(){const context=useWorkbenchContext();return context.permissions.includes("listingkit.platform_admin")?<Link href={path} prefetch={false}>数据条数价格</Link>:null}
export function DataRowPricingPage({expectedUserId}:{expectedUserId:string}){
 const c=useWorkbenchContext(),org=c.effectiveOrganization;
 const allowed=c.permissions.includes("listingkit.platform_admin");
 return <ConsolePage title="数据条数价格" description="平台维护当前部署的 DATA_ROW 售价，企业按报价取得条数" breadcrumbs={[{label:"我的账户",href:"/workbench/account"},{label:"企业空间",href:"/workbench/account/organization"},{label:"数据条数价格"}]}>
 {c.isLoading||c.isSwitching?<ConsoleState kind="loading" title="正在确认平台权限"/>:!allowed||!org||c.user?.id!==expectedUserId||c.error||c.blockingError||c.selectionRequired?<ConsoleState kind="error" title="无平台定价权限或当前企业不可用"/>:<DataRowPriceEditor key={JSON.stringify([expectedUserId,org.id,c.permissions,c.roles])} scope={{userId:expectedUserId,organizationId:org.id}} canWrite={c.permissions.includes("listingkit.admin.write")}/>}
 </ConsolePage>
}
export function DataRowPriceEditor({scope,canWrite}:{scope:PriceScope;canWrite:boolean}){
 const current=useQuery({queryKey:["data-row-pricing",scope],queryFn:({signal})=>dataRowPriceRequest(scope,"",dataRowPriceOffer,{signal}),retry:false,gcTime:0,staleTime:0});
 const history=useQuery({queryKey:["data-row-price-history",scope],queryFn:({signal})=>dataRowPriceRequest(scope,"/history?limit=20",dataRowPriceHistory,{signal}),retry:false,gcTime:0});
 const pending=useResourcePending({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId},["data-row-price"],intentSchema);
 const [amount,setAmount]=useState(""),[reason,setReason]=useState(""),[preview,setPreview]=useState<z.infer<typeof dataRowPriceInput>|null>(null),[message,setMessage]=useState(""),[busy,setBusy]=useState(false);
 const active=useRef<AbortController|null>(null),locked=useRef(false);
 useEffect(()=>()=>active.current?.abort(),[]);
 function prepare(){const v=dataRowPriceInput.safeParse({expected_version:current.data?.pricing_version,unit_price_minor:amount,reason});if(v.success){setPreview(v.data);setMessage("")}else setMessage("请输入有效的非负整数分和调整原因。")}
 async function run(action:(signal:AbortSignal)=>Promise<void>){if(locked.current)return;locked.current=true;setBusy(true);setMessage("");const c=new AbortController();active.current=c;try{await action(c.signal)}catch(error){if(!c.signal.aborted)setMessage(error&&typeof error==="object"&&"code"in error&&error.code==="CONFLICT"?"价格版本或原请求发生冲突，请核对当前版本后重新预览。":"发布结果尚未确认，请按原请求核对；不要重新发布。")}finally{locked.current=false;if(!c.signal.aborted)setBusy(false)}}
 async function finish(receipt:z.infer<typeof dataRowPriceReceipt>,intent:z.infer<typeof intentSchema>,signal:AbortSignal){signal.throwIfAborted();if(receipt.actor_id!==scope.userId||receipt.organization_id!==scope.organizationId||receipt.previous_version!==intent.input.expected_version||receipt.offer.unit_price_minor!==intent.input.unit_price_minor||receipt.reason!==intent.input.reason)throw Error("receipt mismatch");pending.clear(intent);setPreview(null);setMessage(`已发布价格版本 ${receipt.offer.pricing_version}；发布回执 ${receipt.id}`);await Promise.all([current.refetch(),history.refetch()])}
 async function publish(intent?:z.infer<typeof intentSchema>){if(!canWrite||!pending.ready||!preview&&!intent)return;const command=intent??{key:crypto.randomUUID(),input:preview!};await run(async signal=>{pending.persist(command);try{const receipt=await dataRowPriceRequest(scope,"/publish",dataRowPriceReceipt,{body:command.input,idempotencyKey:command.key,signal});await finish(receipt,command,signal)}catch(error){if(error&&typeof error==="object"&&"code"in error&&["CONFLICT","INVALID_REQUEST"].includes(String(error.code))){pending.clear(command);setPreview(null);await current.refetch()}throw error}})}
 async function recover(){const command=pending.command;if(!command)return;await run(async signal=>{const receipt=await dataRowPriceRequest(scope,`/receipts/${encodeURIComponent(command.key)}`,dataRowPriceReceipt,{signal});await finish(receipt,command,signal)})}
 return <div className="space-y-4">
 <Card className="p-5 space-y-3"><h2>当前部署售价</h2>{current.data?<><p>{money(current.data.unit_price_minor)} / 条</p><p>版本：{current.data.pricing_version}</p><p>每单数量：{current.data.min_quantity}–{current.data.max_quantity} 条</p></>:<p role="status">{current.isError?"当前价格不可用；不会按免费处理。":"正在读取当前价格…"}</p>}<p>影响本部署所有企业的新报价；已有有效报价与订单保留原价格。不自动发放资源或开始采集。</p></Card>
 {pending.error?<p role="alert">原发布信息不可读，已暂停新发布。</p>:pending.command?<Card className="p-5 space-y-3"><h2>原发布待确认</h2><p>请求编号：{pending.command.key}</p><Button disabled={busy} onClick={()=>void recover()}>核对原发布回执</Button><Button variant="outline" disabled={busy||!canWrite} onClick={()=>void publish(pending.command!)}>按原请求重试</Button></Card>:<Card className="p-5 space-y-3"><h2>调整后续报价</h2><label className="block">单价（人民币分）<Input aria-label="单价（人民币分）" inputMode="numeric" maxLength={19} value={amount} disabled={!canWrite||busy} onChange={e=>{setAmount(e.target.value);setPreview(null)}}/></label><label className="block">调整原因<Input aria-label="调整原因" maxLength={500} value={reason} disabled={!canWrite||busy} onChange={e=>{setReason(e.target.value);setPreview(null)}}/></label><Button disabled={!canWrite||busy||!pending.ready||!current.data} onClick={prepare}>预览价格调整</Button>
 {preview?<section aria-label="价格调整预览"><p>从当前版本 {preview.expected_version} 调整为 {money(preview.unit_price_minor)} / 条</p>{preview.unit_price_minor==="0"?<p>零价发布后，企业可按数量免费领取，仍记录订单和条数。</p>:null}<Button disabled={busy||current.data?.pricing_version!==preview.expected_version} onClick={()=>void publish()}>确认发布价格</Button><Button variant="outline" disabled={busy} onClick={()=>setPreview(null)}>取消</Button></section>:null}</Card>}
 {message?<p role="status">{message}</p>:null}
 <Card className="p-5 space-y-3"><h2>价格发布记录</h2>{history.data?.items.map(r=><div key={r.id}><p>{money(r.offer.unit_price_minor)} / 条 · {r.offer.pricing_version}</p><p>{r.actor_id} · {r.created_at} · {r.reason}</p></div>)}{history.data?.items.length===0?<p>暂无发布记录</p>:null}{history.isError?<p>发布记录暂不可读</p>:null}<p>显示最近20条发布。发布结果以原回执和当前价格分别核对。</p></Card>
 </div>
}
