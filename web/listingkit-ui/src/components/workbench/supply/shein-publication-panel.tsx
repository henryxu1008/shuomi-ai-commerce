"use client";
import {useEffect,useRef,useState} from "react";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {readSheinPublicationStatus,prepareSheinShelf,type SupplyScope} from "@/lib/api/supply-chain";
import {sheinShelfActionSchema,type SheinPublicationStatus,type SheinShelfConfirmation} from "@/lib/contracts/shein-publication";
import {CollectionDialog} from "../collections/collection-page";
import type {SupplyCommandState} from "./use-supply-command";

const modes={self_operated:"自运营",semi_managed:"半托管",fully_managed:"全托管"};
const date=(value?:string)=>value?new Date(value).toLocaleString("zh-CN",{hour12:false}):"未查询";
export function SheinPublicationPanel({scope,sourceId,storeId,receiptId,storeName,command,canSubmit}:{scope:SupplyScope;sourceId:string;storeId:string;receiptId:string;storeName:string;command:SupplyCommandState;canSubmit:boolean}){
 const [value,setValue]=useState<SheinPublicationStatus>(),[confirmation,setConfirmation]=useState<SheinShelfConfirmation>(),[busy,setBusy]=useState(false),[error,setError]=useState(""),[notice,setNotice]=useState("");
 const request=useRef<AbortController|null>(null),alive=useRef(true),active=useRef(false);
 const [response,setResponse]=useState<NonNullable<ReturnType<typeof sheinShelfActionSchema.parse>["response"]>>();
 useEffect(()=>{alive.current=true;return()=>{alive.current=false;request.current?.abort()}},[]);
 const blocked=busy||command.busy||!!command.pending||!command.ready;
 async function query(offset=0){
  if(active.current)return;active.current=true;setBusy(true);setError("");setConfirmation(undefined);
  const c=new AbortController();request.current=c;
  try{const next=await readSheinPublicationStatus(scope,{sourceId,storeId,receiptId,stockOffset:offset},c.signal);if(alive.current)setValue(next)}
  catch{if(alive.current)setError("本次平台查询失败。下方如有结果，仍为上次查询，不能据此发起上架。")}
  finally{active.current=false;if(alive.current)setBusy(false)}
 }
 async function prepare(skc:string){
  if(blocked||active.current||!canSubmit)return;active.current=true;setBusy(true);setError("");const c=new AbortController();request.current=c;
  try{const next=await prepareSheinShelf(scope,{sourceId,storeId,receiptId,skc},c.signal);if(alive.current)setConfirmation(next)}
  catch{if(alive.current)setError("无法准备上架确认。请重新查询审核、计划及当前状态，并确认店铺权限。")}
  finally{active.current=false;if(alive.current)setBusy(false)}
 }
 async function confirm(){
  if(!confirmation||blocked||!canSubmit)return;
  if(Date.now()>=Date.parse(confirmation.expiresAt)){setConfirmation(undefined);setError("确认已过期，请重新查询后准备上架。");return}
  const prepared=confirmation;setConfirmation(undefined);
  const result=await command.execute("shein-shelf-confirm",{sourceId,storeId,receiptId,skc:prepared.skc,token:prepared.token},prepared.key);
  if(!alive.current)return;
  const parsed=sheinShelfActionSchema.safeParse(result);
  if(parsed.success){setNotice(parsed.data.message);setResponse(parsed.data.response);if(parsed.data.observation)setValue(parsed.data.observation);else setError("请查询平台结果，核实当前状态。原请求不会自动重发。");}
  else setError("操作结果尚未确认，请使用原操作结果查询，避免重复提交。");
 }
 return <section className="space-y-4" aria-label="SHEIN 平台状态">
  <div className="flex flex-wrap items-center justify-between gap-3"><div><h3 className="text-lg font-semibold">平台状态</h3><p className="text-sm text-muted-foreground">{storeName} · 美国站{value?" · "+modes[value.mode]:""}</p></div><Button variant="outline" disabled={busy||command.busy} onClick={()=>void query()}>查询平台结果</Button></div>
  <p className="text-sm text-muted-foreground">按原上传回执读取审核；当前上架、SKU 在售与库存分别核实。查询不会重新上传或上架。</p>
  {busy?<p role="status">正在读取平台结果…</p>:null}{error?<p role="alert" className="text-sm text-warning-foreground">{error}</p>:null}{notice?<p role="status" className="text-sm">{notice}</p>:null}
  {response?<Card className="space-y-2 p-4 text-sm"><h4 className="font-semibold">本次平台响应 · 结果仍待核实</h4><p>{response.message} · {response.code}</p>{response.failures.map((f,i)=><p key={i} className="break-all">{f.identity} · {f.message} · {f.code}</p>)}<p className="break-all text-xs text-muted-foreground">响应摘要 {response.responseHash}。刷新后以原操作的持久状态为准；此摘要不解除请求保护。</p></Card>:null}
  {value?<>
   <Card className="space-y-2 p-4"><h4 className="font-semibold">原上传回执</h4><p className="break-all text-sm">SPU {value.spu} · 原版本 {value.version||"缺失"}</p><p className="break-all text-xs text-muted-foreground">回执 {receiptId}</p><p className="text-xs text-muted-foreground">本次查询 {date(value.observedAt)}</p></Card>
   {value.skcs.map(row=><Card key={row.skc} className="space-y-4 p-4">
    <h4 className="break-all font-semibold">SKC {row.skc}</h4>
    <div className="grid gap-4 md:grid-cols-2"><div className="space-y-2 rounded-lg border border-border p-4"><h5 className="text-sm font-semibold">原版本审核</h5><p>{row.auditLabel}</p>{row.reasons.map((reason,i)=><p className="break-words text-sm" key={i}>{reason.language?reason.language+" · ":""}{reason.content}</p>)}<p className="text-xs text-muted-foreground">{date(value.auditAt)}</p>{value.auditError?<p className="text-sm text-warning-foreground">{value.auditError}</p>:null}</div>
     <div className="space-y-2 rounded-lg border border-border p-4"><h5 className="text-sm font-semibold">当前美国站状态</h5><p>{row.shelfState===1?"已上架":row.shelfState===0?"未上架":"上架状态未知"}</p>{row.attempt?.status==="outcome_unknown"?<p className="text-sm text-warning-foreground">{row.shelfState===1?"当前已上架，原请求结果待核实":"原上架请求结果待核实"}</p>:null}<p className="text-xs text-muted-foreground">{date(value.liveAt)}</p>{value.liveError?<p className="text-sm text-warning-foreground">{value.liveError}</p>:null}</div></div>
    <div className="rounded-lg bg-muted p-4 text-sm"><p className="font-medium">原提交计划</p><p>{value.mode==="fully_managed"?"全托管由平台安排上架；原 shelf_require："+(row.shelfRequire||"未返回"):value.mode==="semi_managed"?row.shelfWay==="2"?"预约上架（北京时间）："+(row.hopeOnSaleDate||"缺失"):row.shelfWay==="1"?"审核后自动上架":"原计划未知":"自运营首次上架"}</p>{row.blockingReason?<p className="mt-1 text-muted-foreground">{row.blockingReason}</p>:null}</div>
    <div className="space-y-2"><h5 className="text-sm font-semibold">SKU 可售条件 · {value.stockType} 库存</h5>{row.skus.map(sku=><div key={sku.code} className="rounded-md border border-border p-3 text-sm"><p className="break-all">{sku.code} · {sku.mallState===1?"SKU 在售":sku.mallState===2?"SKU 停售":"SKU 状态未知"}</p><p>{sku.inventoryScopeKnown?sku.usable===0?"美国可用库存为 0":"美国可用库存："+sku.usable:"库存未知（美国仓归属或数量未核实）"}</p>{sku.reportedUsable!==null&&!sku.inventoryScopeKnown?<p className="text-xs text-muted-foreground">平台汇总 {sku.reportedUsable}，尚不能证明美国可用库存。</p>:null}</div>)}<p className="text-xs text-muted-foreground">库存查询 {date(value.stockAt)} · {value.stockComplete?"本商品库存条件已完整读取":"本次库存仍有未核实或未读取项"}</p></div>
    {canSubmit&&!error&&row.canShelf&&value.mode!=="fully_managed"?<Button disabled={blocked} onClick={()=>void prepare(row.skc)}>上架到美国站</Button>:null}
   </Card>)}
   {value.stockError?<p className="text-sm text-warning-foreground">{value.stockError}</p>:null}
   <div className="flex flex-wrap items-center gap-3 text-sm"><span>本次 SKU {value.stockOffset+1}–{Math.min(value.stockOffset+100,value.stockTotal)} / {value.stockTotal}</span>{value.stockOffset>0?<Button variant="outline" disabled={busy} onClick={()=>void query(0)}>返回首组 SKU</Button>:null}{value.nextStockOffset!==undefined?<Button variant="outline" disabled={busy} onClick={()=>void query(value.nextStockOffset)}>查询下一组 SKU</Button>:null}</div>
   <p className="rounded-lg bg-primary/5 p-4 text-sm text-muted-foreground">审核通过、已上架、SKU 在售、美国仓可用库存需分别满足；当前结果不保证买家可结算。原请求结果待核实时，继续查询不会解除原请求保护。</p>
  </>:<Card className="p-4 text-sm text-muted-foreground">尚未查询平台结果。</Card>}
  {confirmation?<CollectionDialog title="确认首次上架" className="w-[min(640px,calc(100vw-32px))]" onClose={()=>setConfirmation(undefined)}><dl className="grid gap-2 text-sm"><dt>企业</dt><dd className="break-all">{scope.organizationId}</dd><dt>目标店铺 / 模式</dt><dd>{storeName} · {modes[confirmation.mode]} · 美国站</dd><dt>原上传对象</dt><dd className="break-all">SPU {confirmation.spu} · 版本 {confirmation.version} · SKC {confirmation.skc}</dd><dt>原计划</dt><dd>{confirmation.shelfWay==="2"?"预约 "+confirmation.hopeOnSaleDate+"（北京时间）":confirmation.mode==="semi_managed"?"审核后自动上架":"自运营首次上架"}</dd></dl><p className="rounded-lg bg-muted p-4 text-sm">确认有效期至 {date(confirmation.expiresAt)}（5 分钟）。本次只请求此 SKC 上架到美国站。提交后如结果待核实，只查询原结果，不重复发送。</p><div className="flex justify-end gap-3"><Button variant="outline" onClick={()=>setConfirmation(undefined)}>取消</Button><Button disabled={blocked} onClick={()=>void confirm()}>确认首次上架</Button></div></CollectionDialog>:null}
 </section>;
}
