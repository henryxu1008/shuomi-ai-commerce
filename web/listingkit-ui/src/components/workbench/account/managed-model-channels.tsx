"use client";

import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {ModelRouteError,modelRouteRequest,type ModelRouteScope} from "@/lib/api/model-route-pricing";
import {channelTemplate,openAIQueryModel,type ManagedConnection} from "@/lib/contracts/channel-credentials";
import {createChannel,createChannelReceipt,editBinding,editBindingReceipt,managedChannelPath,managedChannelPending,managedChannels,renameChannel,renameChannelReceipt,type ManagedChannelPending,type ManagedChannelReceipt} from "@/lib/contracts/managed-model-channels";
import {useResourcePending} from "../resources/resource-pending";
import {AccountDialog} from "./account-dialog";
import {ChannelModelQuery} from "./channel-model-query";
import styles from "./model-route-pricing.module.css";

const definitiveFailures=new Set(["INVALID","FORBIDDEN","CONFLICT","NEEDS_CONFIGURATION","CHANNEL_LIMIT","NOT_FOUND","AUTHENTICATION_REQUIRED","PERMISSION_DENIED","ORGANIZATION_ACCESS_REVOKED","ORGANIZATION_ACCESS_DENIED","ORGANIZATION_SUSPENDED","ORGANIZATION_SELECTION_REQUIRED","ORGANIZATION_CONTEXT_CHANGED","IDENTITY_CONTEXT_CHANGED"]);
const names={"openai-mini-v1":"OpenAI 官方","openai-compatible-mini-v1":"OpenAI-compatible"};
type Props={
 scope:ModelRouteScope;
 memberId:string;
 testMemberId?:string;
 focusModelId?:string;
 blocked:boolean;
 onSaved:()=>Promise<unknown>;
 onCredentials:(channel:ManagedConnection,action:"set-key"|"disable")=>void;
 onSelect?:(channel:ManagedConnection,reference:{id:string;version:string})=>void;
};

export function ManagedModelChannels({scope,memberId,testMemberId,focusModelId,blocked,onSaved,onCredentials,onSelect}:Props){
 const query=useQuery({
  queryKey:["model-services",scope,"managed-channels"],
  queryFn:({signal})=>modelRouteRequest(scope,"/managed-channels",managedChannels,{signal}),
  enabled:!blocked,retry:false,gcTime:0,refetchOnWindowFocus:false,
 });
 const pending=useResourcePending({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId},
  ["model-route-pricing","managed-channels",memberId],managedChannelPending,{storage:"session",maxLength:2048});
 const [dialog,setDialog]=useState<{action:"create"|"name"|"replace";channel?:ManagedConnection}|null>(null);
 const [confirmDetach,setConfirmDetach]=useState(false);
 const [selected,setSelected]=useState<string>();
 const [busy,setBusy]=useState(false);
 const [message,setMessage]=useState("");
 const [failure,setFailure]=useState("");
 const [formError,setFormError]=useState("");
 const [template,setTemplate]=useState<keyof typeof names>("openai-mini-v1");
 const adding=useRef<HTMLButtonElement>(null);
 const detail=useRef<HTMLHeadingElement>(null);
 const controller=useRef<AbortController|null>(null);
 const active=useRef(true);
 useEffect(()=>{active.current=true;return()=>{active.current=false;controller.current?.abort()}},[]);
 const keyInput=useRef<HTMLInputElement>(null);
 const intent=pending.command;
 const canReconcile=!!intent&&intent.memberId===memberId&&intent.organizationId===scope.organizationId&&intent.userId===scope.userId;
 const locked=blocked||busy||!pending.ready||!!intent||pending.error||query.isFetching||query.isError;
 const channels=query.data?.channels??[];
 const focused=channels.find(channel=>channel.models.some(model=>model.id===focusModelId));
 const selectedChannel=channels.find(channel=>channel.connectionId===(selected??focused?.connectionId));
 useEffect(()=>{if(selectedChannel)detail.current?.focus()},[selectedChannel]);

 async function command(original:ManagedChannelPending,body?:string){
  if(controller.current)return;
  setBusy(true);setFailure("");setMessage("");
  const request=new AbortController();
  controller.current=request;
  try{
   if(body)pending.persist(original);
   const path=managedChannelPath(original)+(body?"":"/receipts/"+encodeURIComponent(original.key));
   const schema=original.operation==="create"?createChannelReceipt:original.operation==="name"?renameChannelReceipt:editBindingReceipt;
   const init=body?{method:"POST",headers:{"Content-Type":"application/json","Idempotency-Key":original.key},body,signal:request.signal}:{signal:request.signal};
   const receipt=await modelRouteRequest<ManagedChannelReceipt>(scope,path,schema,init);
   const connectionId="connection" in receipt.result?receipt.result.connection.connectionId:receipt.result.connectionId;
   if(receipt.organizationId!==original.organizationId||receipt.actorId!==original.userId||receipt.memberId!==original.memberId||
      receipt.requestKey!==original.key||original.connectionId&&connectionId!==original.connectionId)throw new ModelRouteError("OUTCOME_UNKNOWN");
   if(active.current){
    pending.clear(original);setSelected(connectionId);setDialog(null);setConfirmDetach(false);
    setMessage(original.operation==="create"?"渠道与 Key 已保存。":original.operation==="name"?"渠道名称已保存。":"模型配置已保存。");
    await onSaved();
   }
  }catch(error){
   if(!active.current)return;
   const code=error instanceof ModelRouteError?error.code:"UNAVAILABLE";
   if(body&&error instanceof ModelRouteError&&error.status<500&&definitiveFailures.has(code))pending.clear(original);
   setFailure(code==="CONFLICT"?"渠道配置已有变化，请重新读取后再试。":
    code==="INVALID"?"请检查渠道名称、基础地址和 Key 后再试。":
    code==="FORBIDDEN"||code==="PERMISSION_DENIED"?"当前账号没有此项操作权限。":
    code==="CHANNEL_LIMIT"?"当前企业已保存 128 个渠道，暂时不能再添加。":
    code==="NEEDS_CONFIGURATION"?"此型号暂不能用于当前企业，请查看已有模型配置。":
    "暂时无法确认保存结果，请查询原请求。");
  }finally{
   controller.current=null;
   if(active.current)setBusy(false);
  }
 }
 function save(operation:ManagedChannelPending["operation"],input:unknown,connectionId?:string){
  if(locked)return;
  const schema=operation==="create"?createChannel:operation==="name"?renameChannel:editBinding;
  const parsed=schema.safeParse({expectedMemberId:memberId,input});
  if(!parsed.success){setFormError("请检查填写内容后再试。");return;}
  setFormError("");
  if(keyInput.current)keyInput.current.value="";
  void command({key:crypto.randomUUID(),userId:scope.userId,organizationId:scope.organizationId,memberId,operation,connectionId},JSON.stringify(parsed.data));
 }
 function open(action:"create"|"name"|"replace",channel?:ManagedConnection){
  setFailure("");setFormError("");
  setTemplate(action==="replace"?"openai-compatible-mini-v1":channel?.registration?.template??"openai-mini-v1");
  setDialog({action,channel});
 }
 const activeBinding=query.data?.bindings.find(binding=>binding.enabled&&binding.connectionId===selectedChannel?.connectionId);
 const queryTarget=query.data?.modelQueryTargets?.find(target=>target.connectionId===selectedChannel?.connectionId);
 return <>
  <Card className={styles.panel}>
   {!selectedChannel?<><div className={styles.sectionHeader}>
    <h2>企业自助渠道</h2>
    {query.data?.canSave&&<Button ref={adding} disabled={locked} onClick={()=>open("create")}>添加渠道</Button>}
   </div>
   {query.isPending?<p role="status">正在读取企业渠道…</p>:query.isError?<p role="alert">暂时无法读取企业渠道。</p>:<>
    {!channels.length&&<p>暂无企业自助渠道。</p>}
    {!query.data.canSave&&<p>当前无法保存渠道，请核对权限或联系维护人员。</p>}
    {channels.map(channel=><div className={styles.history} key={channel.connectionId}>
     <Button variant="outline" onClick={()=>setSelected(channel.connectionId)}>{channel.registration?.name}</Button>
     <p>{channel.baseUrl} · {channel.enabled?"Key 已保存":"已停用"} · {channel.models.length?"已关联文本模型":"尚未关联模型"}</p>
    </div>)}
   </>}</>:<>
    <h2 ref={detail} tabIndex={-1}>{selectedChannel.registration?.name}</h2>
    <p>{names[selectedChannel.registration!.template]} · {selectedChannel.baseUrl}</p>
    <p>{selectedChannel.enabled?"Key 已保存":"已停用"}{selectedChannel.account!=="default"?" · "+selectedChannel.account:""}</p>
    <div className={styles.actions}>
     <Button variant="outline" onClick={()=>{setSelected("");requestAnimationFrame(()=>adding.current?.focus())}}>返回渠道列表</Button>
     {query.data?.canRename&&<Button variant="outline" disabled={locked} onClick={()=>open("name",selectedChannel)}>更改名称</Button>}
     {query.data?.canSave&&<>
      <Button variant="outline" disabled={locked} onClick={()=>onCredentials(selectedChannel,"set-key")}>{selectedChannel.enabled?"轮换 Key":"重新录入 Key"}</Button>
      {selectedChannel.enabled&&<Button variant="outline" disabled={locked} onClick={()=>onCredentials(selectedChannel,"disable")}>停用渠道</Button>}
      <Button variant="outline" disabled={locked} onClick={()=>open("replace",selectedChannel)}>更换服务地址</Button>
     </>}
    </div>
    {activeBinding?<div className={styles.history}>
     <h3>已关联文本模型</h3><p>{openAIQueryModel}</p>
     {query.data?.bindingReadiness&&((query.data.bindingReadiness.find(r=>r.bindingId===activeBinding.bindingId)?.ready&&selectedChannel.enabled)
      ?(!onSelect&&<p>已加入企业可选渠道，请联系企业配置人员选用。</p>)
      :<p>型号已关联，暂不可选用。{({
        CHANNEL_DISABLED:"此渠道已停用。", OFFICIAL_PRICE_MISSING:"缺少适用参考价。",
        PRICE_APPLICABILITY_MISSING:"当前参考价不适用。", PRICE_OR_FX_NOT_APPLICABLE:"参考价或汇率暂不可用。",
        OPEN_CONSUMER_MISSING:"部分已启用用途暂不能使用该型号。",
      } as Record<string,string>)[query.data.bindingReadiness.find(r=>r.bindingId===activeBinding.bindingId)?.reason??""]??"请检查当前渠道及模型配置。"}</p>)}
     <div className={styles.actions}>
      {onSelect&&(!query.data?.bindingReadiness||query.data.bindingReadiness.some(r=>r.bindingId===activeBinding.bindingId&&r.ready))&&<Button disabled={locked||!selectedChannel.enabled} onClick={()=>onSelect(selectedChannel,{id:activeBinding.bindingId,version:"v1"})}>去选用模型</Button>}
      {query.data?.canAttach&&<Button variant="outline" disabled={locked} onClick={()=>setConfirmDetach(true)}>取消关联</Button>}
     </div>
    </div>:<div className={styles.history}>
     <h3>尚未关联模型</h3>
     {query.data?.canAttach&&selectedChannel.enabled&&(query.data.bindingTemplates.includes(selectedChannel.registration!.template)?<>
      <p>可关联文本型号 {openAIQueryModel}。符合选用条件后，可在企业模型中切换。</p>
      <Button disabled={locked} onClick={()=>save("models",{action:"attach",expectedRevision:selectedChannel.revision},selectedChannel.connectionId)}>关联文本模型</Button>
     </>:<p>这个渠道暂时无法覆盖企业已开启的文本用途。你可以保留渠道与 Key。</p>)}
    </div>}
    {selectedChannel.registration?.template==="openai-compatible-mini-v1"?<p>此渠道暂不支持资料查询。</p>:
     queryTarget&&testMemberId?<ChannelModelQuery key={JSON.stringify([selectedChannel.connectionId,selectedChannel.revision,queryTarget.reference])}
      scope={scope} connectionId={selectedChannel.connectionId} disabled={locked||!selectedChannel.enabled}
      input={{expectedMemberId:testMemberId,expectedRevision:selectedChannel.revision,reference:queryTarget.reference}}/>:<p>尚未检查模型资料。</p>}
   </>}
   {failure&&<p role="alert">{failure}</p>}
   {message&&<p role="status">{message}</p>}
   {intent&&<div><p>保存结果待核实。原请求编号 {intent.key}</p>
    <Button variant="outline" disabled={busy||blocked||!canReconcile} onClick={()=>void command(intent)}>查询本次结果</Button></div>}
  </Card>
  {confirmDetach&&activeBinding&&selectedChannel&&<AccountDialog title="取消模型关联" onClose={()=>setConfirmDetach(false)}>
   <p>取消关联后，企业模型中已选用这个型号的配置需要更换并重新发布。</p>
   <div className={styles.actions}><Button variant="outline" onClick={()=>setConfirmDetach(false)}>返回</Button>
    <Button disabled={locked} onClick={()=>save("models",{action:"disable",bindingId:activeBinding.bindingId,expectedRevision:selectedChannel.revision},selectedChannel.connectionId)}>确认取消关联</Button></div>
  </AccountDialog>}
  {dialog&&<AccountDialog title={dialog.action==="name"?"更改渠道名称":dialog.action==="replace"?"更换服务地址":"添加渠道"} onClose={()=>setDialog(null)}>
   <form className={styles.form} autoComplete="off" onSubmit={event=>{
    event.preventDefault();
    const data=new FormData(event.currentTarget),name=String(data.get("name")??"").trim();
    if(dialog.action==="name"){save("name",{name,expectedRevision:dialog.channel?.registration?.revision},dialog.channel?.connectionId);return;}
    save("create",{name,template,baseUrl:template==="openai-compatible-mini-v1"?String(data.get("baseUrl")??"").trim():undefined,
     account:String(data.get("account")??"").trim()||undefined,apiKey:keyInput.current?.value??""});
   }}>
    <label className={styles.field}>渠道名称<input name="name" required maxLength={128} defaultValue={dialog.channel?.registration?.name}/></label>
    {dialog.action!=="name"&&<>
     {dialog.action==="replace"&&<p>这会新建渠道。保存后，关联模型并发布新选择即可切换地址。</p>}
     <label className={styles.field}>提供方<select value={template} onChange={e=>setTemplate(e.target.value as keyof typeof names)}>
      {channelTemplate.options.map(value=><option key={value} value={value}>{names[value]}</option>)}</select></label>
     {template==="openai-compatible-mini-v1"?<label className={styles.field}>服务基础地址<input name="baseUrl" type="url" required maxLength={2048} placeholder="https://models.example.com/v1"/></label>:<p>服务地址 https://api.openai.com/v1</p>}
     <label className={styles.field}>账号备注（选填）<input name="account" maxLength={128}/></label>
     <label className={styles.field}>API Key<input ref={keyInput} type="password" autoComplete="new-password" spellCheck={false} required maxLength={4096}/></label>
     <p>使用此渠道时，Key 和请求内容会发送到上面的服务地址。保存时不会调用模型。</p>
    </>}
    {formError&&<p role="alert">{formError}</p>}
    <div className={styles.actions}><Button type="button" variant="outline" onClick={()=>setDialog(null)}>取消</Button><Button type="submit" disabled={locked}>{busy?"正在保存…":"保存"}</Button></div>
   </form>
  </AccountDialog>}
 </>;
}
