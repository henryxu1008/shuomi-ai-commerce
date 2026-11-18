"use client";

import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {z} from "zod";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {modelRouteRequest,ModelRouteError,type ModelRouteScope} from "@/lib/api/model-route-pricing";
import {currentTextChoice,platformChannelList,selectionEndpoint} from "@/lib/contracts/platform-model-channels";
import {formatCurrencyMicros,routeCurrent,type ModelOption} from "@/lib/contracts/model-route-pricing";
import {useResourcePending} from "../resources/resource-pending";
import {AccountDialog} from "./account-dialog";
import styles from "./model-route-pricing.module.css";

const intentSchema=z.strictObject({key:z.uuid(),receiptPath:z.string().max(260)});
const operationLabels:Record<string,string>={
 aiworkbench_chat_plan:"硕米 Chat",
 aiworkbench_store_route:"Chat 店铺范围判断",
 product_agent_decision:"商品处理",
 "product.content.decision":"商品内容",
 "product.research.decision":"市场调研",
 "product.selection.decision":"商品选品",
 "shop.diagnosis.decision":"店铺诊断",
 "listing.materials.decision":"刊登资料",
};
const explanation:Record<string,string>={
 PLATFORM_ACCESS_WITHDRAWN:"此渠道已停止向本企业开放，请切换。",
 PLATFORM_BINDING_REPLACED:"此渠道的型号配置已更新，请重新选择。",
 PLATFORM_CHANNEL_UNAVAILABLE:"平台渠道当前不可用，请切换。",
 CHANNEL_DISABLED:"当前渠道已停用，请切换。",
 MODEL_BINDING_DISABLED:"当前型号已停用，请切换。",
 OPEN_CONSUMER_MISSING:"此渠道暂不能覆盖企业已启用的文本用途。",
 OFFICIAL_PRICE_MISSING:"缺少适用参考价，请联系维护人员。",
 PRICE_OR_FX_NOT_APPLICABLE:"适用参考价或汇率暂不可用。",
};

function useConfigurationCommand(scope:ModelRouteScope,member:string,onSaved:()=>Promise<unknown>,family:string) {
 const pending=useResourcePending({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId},["channel-configuration",family,member],intentSchema,{storage:"session"});
 const active=useRef<AbortController|null>(null);
 const [busy,setBusy]=useState(false);
 const [message,setMessage]=useState("");
 useEffect(()=>()=>active.current?.abort(),[]);
 async function run(intent:z.infer<typeof intentSchema>,path:string,body?:unknown) {
  const endpoint=selectionEndpoint(path.slice(1).split("/"),body===undefined?"GET":"POST");
  if(!endpoint||active.current)return;
  const controller=new AbortController();
  active.current=controller;
  setBusy(true);
  setMessage("");
  try {
   await modelRouteRequest(scope,path,endpoint.output as z.ZodType<unknown>,body===undefined?{signal:controller.signal}:{
    method:"POST",signal:controller.signal,headers:{"Content-Type":"application/json","Idempotency-Key":intent.key},
    body:JSON.stringify(body),
   });
   if(controller.signal.aborted)return;
   pending.clear(intent);
   setMessage("已保存。");
   try {await onSaved()} catch {setMessage("本次已保存，当前状态暂时无法确认。")}
  } catch(error) {
   if(controller.signal.aborted)return;
   const code=error instanceof ModelRouteError?error.code:"UNAVAILABLE";
   if(body!==undefined&&error instanceof ModelRouteError&&error.status<500&&code!=="OUTCOME_UNKNOWN")pending.clear(intent);
   if(code==="CONFLICT")setMessage("配置或参考价已更新，请重新读取并确认。");
   else if(code==="NEEDS_CONFIGURATION")setMessage("渠道尚未具备发布或选用条件，请核对 Key、型号与适用价格。");
   else if(code==="FORBIDDEN")setMessage("当前账号无权完成这项操作。");
   else if(code==="INVALID")setMessage("请检查填写内容。");
   else setMessage("结果暂未确认，请使用原请求查询结果。");
  } finally {
   if(!controller.signal.aborted)setBusy(false);
   if(active.current===controller)active.current=null;
  }
 }
 function send(path:string,receiptPrefix:string,body:object) {
  if(!pending.ready||pending.error||pending.command||active.current||!member)return;
  const key=crypto.randomUUID(),intent={key,receiptPath:receiptPrefix+key};
  try {
   pending.persist(intent);
   return run(intent,path,{...body,expectedMemberId:member});
  } catch {setMessage("暂无法记录本次操作，请重新打开页面。")}
 }
 const verify=()=>{if(pending.command)void run(pending.command,pending.command.receiptPath)};
 return {blocked:!pending.ready||pending.error||!!pending.command||busy, message, hasPending:!!pending.command, verify, send, busy};
}

function ChoiceDetails({option}:{option:ModelOption}) {
 const price=option.price;
 return <>
  <strong className="block">{option.channelName||option.channel}</strong>
  <span className="block">{option.model}{option.source==="platform"?" · 平台提供":option.source==="enterprise"?" · 企业自有":""}</span>
  {price&&<span className="block">参考价每百万文本 Token · 输入 {formatCurrencyMicros(price.rates.inputText)} {price.currency} · 输出 {formatCurrencyMicros(price.rates.outputText)} {price.currency}</span>}
 </>;
}

export function EnterpriseTextChoice({scope,member,focusId,onSaved}:{scope:ModelRouteScope;member:string;focusId?:string;onSaved:()=>Promise<unknown>}) {
 const [choosing,setChoosing]=useState(false);
 const [selected,setSelected]=useState<ModelOption>();
 const button=useRef<HTMLButtonElement>(null);
 const query=useQuery({queryKey:["model-services",scope,"text-choice"],queryFn:({signal})=>modelRouteRequest(scope,"/enterprise/text-choice",currentTextChoice,{signal}),retry:false,gcTime:0});
 const view=query.data?.view;
 const command=useConfigurationCommand(scope,member,async()=>{
  setChoosing(false);setSelected(undefined);
  const refreshed=await query.refetch();
  await onSaved();
  if(refreshed.error)throw refreshed.error;
 },"enterprise-text");
 useEffect(()=>{
  if(focusId)button.current?.focus();
 },[focusId,view?.revision]);
 if(!view)return <Card className={styles.panel}><h2>企业文本模型</h2><p>{query.isError?"暂时无法读取文本模型。":"正在读取文本模型…"}</p>{query.isError&&<Button onClick={()=>void query.refetch()}>重新读取</Button>}</Card>;
 const purposes=[...new Set((view.purposes??[]).map(p=>operationLabels[p]??"其他已启用的文本用途"))];
 return <Card className={styles.panel}>
  <h2>企业文本模型</h2>
  {purposes.length>0&&<p>用于{purposes.join("、")}</p>}
  {view.current?<div><ChoiceDetails option={view.current}/><p>{view.current.ready?"当前已选":explanation[view.current.reason??""]??"当前选择暂不可用，请切换。"}</p></div>:<p>尚未选择</p>}
  <Button ref={button} disabled={command.blocked} onClick={()=>{setSelected(view.options.find(item=>item.reference.id===focusId));setChoosing(true);}}>切换</Button>
  {command.message&&<p role="status">{command.message}</p>}
  {command.hasPending&&<Button disabled={command.busy} onClick={command.verify}>查询本次结果</Button>}
  {choosing&&<AccountDialog title="切换企业文本模型" onClose={()=>setChoosing(false)} returnFocusRef={button}>
   {view.options.length===0?<p>暂无可选渠道，请联系平台管理员。</p>:<fieldset className={styles.form}>
    <legend>选择渠道</legend>
    {view.options.map(option=><label className={styles.field} key={option.reference.id+option.reference.version}>
     <input type="radio" name="text-channel" checked={selected?.reference.id===option.reference.id&&selected?.reference.version===option.reference.version} onChange={()=>setSelected(option)} />
     <span><ChoiceDetails option={option}/></span>
    </label>)}
   </fieldset>}
   {purposes.length>0&&<p>本次切换影响{purposes.join("、")}。</p>}
   {command.message&&!command.hasPending&&<Button variant="outline" onClick={()=>{setSelected(undefined);void query.refetch();}}>重新读取可选渠道</Button>}
   {command.message&&<p role="status">{command.message}</p>}
   {command.hasPending&&<Button disabled={command.busy} onClick={command.verify}>查询本次结果</Button>}
   <div className={styles.actions}>
    <Button variant="outline" onClick={()=>setChoosing(false)}>取消</Button>
    {selected?.quote&&<Button disabled={command.blocked} onClick={()=>void command.send("/enterprise/text-choice/select","/enterprise/text-choice/receipts/",{reference:selected.reference,quote:selected.quote,expectedRevision:view.revision})}>确认切换</Button>}
   </div>
  </AccountDialog>}
 </Card>;
}

export function PlatformChannelConfiguration({scope,member,organizations=[],onSaved}:{scope:ModelRouteScope;member:string;organizations?:{id:string;name:string}[];onSaved:()=>Promise<unknown>}) {
 const list=useQuery({queryKey:["model-services",scope,"platform-channels"],queryFn:({signal})=>modelRouteRequest(scope,"/platform-channels",platformChannelList,{signal}),retry:false,gcTime:0});
 const price=useQuery({queryKey:["model-services",scope,"platform-channels-prices"],queryFn:({signal})=>modelRouteRequest(scope,"/platform-prices",routeCurrent,{signal}),enabled:!!list.data,retry:false,gcTime:0});
 const [selected,setSelected]=useState<string>();
 const [editor,setEditor]=useState<"create"|"publish"|"rotate"|"disable">();
 const [template,setTemplate]=useState("openai-mini-v1");
 const command=useConfigurationCommand(scope,member,async()=>{setEditor(undefined);const refreshed=await list.refetch();await onSaved();if(refreshed.error)throw refreshed.error;},"platform");
 const row=list.data?.channels.find(item=>item.connection.connectionId===selected);
 if(list.isError) {
  if(list.error instanceof ModelRouteError&&[403,404].includes(list.error.status))return null;
  return <Card className={styles.panel}><h2>平台模型渠道</h2><p>暂时无法读取平台渠道。</p><Button onClick={()=>void list.refetch()}>重新读取</Button></Card>;
 }
 if(!list.data)return null;
 const root="/platform-channels";
 const connection=row?.connection;
 const id=connection?.connectionId;
 const channelPrices=(price.data?.revision?.payload.platformPrices?.prices??[]).filter(item=>item.channel===connection?.channel&&(!item.platformConnectionId||item.platformConnectionId===id)&&item.model==="gpt-5.4-mini-2026-03-17");
 return <Card className={styles.panel}>
  <h2>{connection?.registration?.name??"平台模型渠道"}</h2>
  {command.message&&<p role="status">{command.message}</p>}
  {command.hasPending&&<Button disabled={command.busy} onClick={command.verify}>查询本次结果</Button>}
  {row&&connection&&id?<div className={styles.stack}>
   <p>{connection.channel==="openai-direct"?"OpenAI 官方":"OpenAI-compatible"} · {connection.enabled?"Key 已保存":"已停用"}</p>
   <p>支持型号 gpt-5.4-mini-2026-03-17 · 已向 {row.publication?.organizations.length??0} 家企业开放</p>
   <div className={styles.actions}>
    <Button variant="outline" onClick={()=>setSelected(undefined)}>返回渠道列表</Button>
    {list.data.canWrite&&<>
     <Button disabled={command.blocked||!connection.enabled} onClick={()=>setEditor("publish")}>设置开放企业与价目</Button>
     <Button variant="outline" disabled={command.blocked} onClick={()=>setEditor("rotate")}>{connection.enabled?"轮换 Key":"重新录入 Key"}</Button>
     {connection.enabled&&<Button variant="outline" disabled={command.blocked} onClick={()=>setEditor("disable")}>停用渠道</Button>}
    </>}
   </div>
  </div>:<>
    {list.data.canWrite&&<Button disabled={command.blocked} onClick={()=>{setTemplate("openai-mini-v1");setEditor("create")}}>添加平台渠道</Button>}
    {list.data.channels.length===0&&<p>暂无平台渠道。</p>}
    {list.data.channels.map(({connection,publication})=><div className={styles.history} key={connection.connectionId}>
     <Button variant="outline" onClick={()=>setSelected(connection.connectionId)}>{connection.registration?.name}</Button>
     <p>{!connection.enabled?"已停用":publication?.organizations.length?"已开放":"待发布"}</p>
    </div>)}
  </>}
  {editor&&<AccountDialog title={{create:"添加平台渠道",publish:"设置开放企业与价目",rotate:"保存新 Key",disable:"停用平台渠道"}[editor]} onClose={()=>setEditor(undefined)}>
   <form className={styles.form} onSubmit={event=>{
    event.preventDefault();
    const form=event.currentTarget,data=new FormData(form);
    const keyInput=form.elements.namedItem("apiKey") as HTMLInputElement|null;
    const apiKey=keyInput?.value;
    if(editor==="create")void command.send(root+"/create",root+"/create/receipts/",{name:String(data.get("name")).trim(),template,baseUrl:data.get("baseUrl")||undefined,apiKey});
    else if(id&&connection&&row) {
     if(editor==="publish")void command.send(root+"/"+id+"/publish",root+"/"+id+"/receipts/",{expectedRevision:row.revision,channel:{connectionId:id,priceId:String(data.get("price")??""),organizations:data.getAll("organizations")}});
     else void command.send(root+"/"+id+"/credentials",root+"/"+id+"/credentials/receipts/",{expectedRevision:connection.revision,action:editor==="rotate"?"set-key":"disable",apiKey});
    }
    if(keyInput)keyInput.value="";
   }}>
    {editor==="create"&&<>
     <label className={styles.field}>渠道名称<input name="name" required maxLength={128}/></label>
     <label className={styles.field}>提供方<select value={template} onChange={event=>setTemplate(event.target.value)}>
      <option value="openai-mini-v1">OpenAI 官方</option><option value="openai-compatible-mini-v1">OpenAI-compatible</option>
     </select></label>
     {template==="openai-compatible-mini-v1"?<label className={styles.field}>服务地址<input name="baseUrl" required type="url" placeholder="https://example.com/v1"/></label>:<p>服务地址 https://api.openai.com/v1</p>}
    </>}
    {(editor==="create"||editor==="rotate")&&<label className={styles.field}>API Key<input name="apiKey" required type="password" autoComplete="new-password" maxLength={4096}/></label>}
    {editor==="publish"&&<>
     <p>型号 gpt-5.4-mini-2026-03-17</p>
     <label className={styles.field}>适用参考价<select name="price" defaultValue={row?.publication?.priceId??""}>
      <option value="">请选择参考价</option>
      {channelPrices.map(item=><option value={item.id} key={item.id}>{item.id} · {item.currency} · 输入 {formatCurrencyMicros(item.rates.inputText)}／输出 {formatCurrencyMicros(item.rates.outputText)}</option>)}
     </select></label>
     {channelPrices.length===0&&<p>请先在参考价格与汇率区发布适用价目。</p>}
     <fieldset><legend>开放给指定企业</legend>
      {list.data.organizations.map(org=><label key={org} className={styles.field}><input type="checkbox" name="organizations" value={org} defaultChecked={row?.publication?.organizations.includes(org)}/><span>{organizations.find(item=>item.id===org)?.name??"企业编号 "+org}</span></label>)}
     </fieldset>
     <p>取消勾选并保存会停止向对应企业开放。</p>
    </>}
    {editor==="disable"&&<p>该渠道当前向 {row?.publication?.organizations.length??0} 家企业开放。停用后，已选用的企业需要切换。</p>}
    {command.message&&<p role="status">{command.message}</p>}
    {command.hasPending&&<Button type="button" disabled={command.busy} onClick={command.verify}>查询本次结果</Button>}
    <div className={styles.actions}>
     <Button type="button" variant="outline" onClick={()=>setEditor(undefined)}>取消</Button>
     <Button type="submit" disabled={command.blocked}>{editor==="disable"?"确认停用":editor==="publish"?"保存开放范围":"保存"}</Button>
    </div>
   </form>
  </AccountDialog>}
 </Card>;
}
