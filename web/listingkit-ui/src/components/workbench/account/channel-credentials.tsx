"use client";

import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {ModelRouteError,modelRouteRequest,type ModelRouteScope} from "@/lib/api/model-route-pricing";
import {credentialConnections,credentialInput,credentialPending,credentialReceipt,matchesCredentialReceipt,openAIQueryModel,type CredentialInput,type CredentialPending,type CredentialReceipt,type ManagedConnection} from "@/lib/contracts/channel-credentials";
import {useResourcePending} from "../resources/resource-pending";
import type { ModelOption } from "@/lib/contracts/model-route-pricing";
import {AccountDialog} from "./account-dialog";
import {ChannelModelQuery} from "./channel-model-query";
import {ManagedModelChannels} from "./managed-model-channels";
import styles from "./model-route-pricing.module.css";

const definiteCodes=new Set(["INVALID","FORBIDDEN","NOT_FOUND","CONFLICT","AUTHENTICATION_REQUIRED","PERMISSION_DENIED","ORGANIZATION_ACCESS_REVOKED","ORGANIZATION_ACCESS_DENIED","ORGANIZATION_SUSPENDED","ORGANIZATION_SELECTION_REQUIRED","ORGANIZATION_CONTEXT_CHANGED","IDENTITY_CONTEXT_CHANGED"]);
function failureText(error:unknown){
 const code=error instanceof ModelRouteError?error.code:"UNAVAILABLE";
 return ({INVALID:"请检查输入格式并重新输入 Key。",CONFLICT:"连接版本或账号备注已有变化，请重新读取后确认。",FORBIDDEN:"没有渠道凭据管理权限，或企业访问已撤销。",PERMISSION_DENIED:"当前渠道凭据管理权限不可用。",NOT_FOUND:"未找到本次回执；这不能证明保存失败，原请求仍保留。",OUTCOME_UNKNOWN:"保存结果待核实，请查询本次结果。"} as Record<string,string>)[code]??"暂时无法确认结果，请保留原请求并查询。";
}

type ChannelSettingsProps = {
 scope:ModelRouteScope;
 testMemberId?:string;
 focusModel?:ModelOption;
 disabled?:boolean;
 unconfirmed?:boolean;
 onSaved?:()=>Promise<void>;
 manageChannelsMemberId?:string;
 canManageKeys?:boolean;
 onSelectModel?:(channel:ManagedConnection,reference:{id:string;version:string})=>void;
};

export function ChannelCredentialsManager(props:ChannelSettingsProps){
 return <ChannelCredentialsContent key={JSON.stringify([props.scope,props.testMemberId,props.focusModel?.reference])} {...props}/>;
}

function ChannelCredentialsContent({scope,testMemberId,focusModel,disabled=false,unconfirmed=false,onSaved,manageChannelsMemberId,canManageKeys=true,onSelectModel}:ChannelSettingsProps){
 const query=useQuery({queryKey:["model-services",scope,"connections"],queryFn:async({signal})=>{const value=await modelRouteRequest(scope,"/connections",credentialConnections,{signal});if(value.organizationId!==scope.organizationId)throw new ModelRouteError("ORGANIZATION_CONTEXT_CHANGED");return value},enabled:!unconfirmed&&canManageKeys,retry:false,gcTime:0,refetchOnWindowFocus:false});
 const pending=useResourcePending({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId},["model-route-pricing","credentials","v1"],credentialPending,{storage:"session",maxLength:2048});
 const [dialog,setDialog]=useState<{connection:ManagedConnection;action:"set-key"|"disable"}|null>(null);
 const [busy,setBusy]=useState(false),[failure,setFailure]=useState(""),[receipt,setReceipt]=useState<CredentialReceipt|null>(null);
 const active=useRef(true),running=useRef(false),controller=useRef<AbortController|null>(null);
 useEffect(()=>{active.current=true;return()=>{active.current=false;controller.current?.abort()}},[]);
 const intent=pending.command,scopeMatches=!intent||intent.userId===scope.userId&&intent.organizationId===scope.organizationId;
 const locked=disabled||busy||!pending.ready||!!intent||pending.error||!scopeMatches||query.isFetching;
 const heading=useRef<HTMLHeadingElement>(null);
 const target=useRef<HTMLHeadingElement>(null);
 const matches=focusModel&&!query.isError?query.data?.connections.filter(connection=>connection.models.some(model=>
  model.category===focusModel.category&&model.id===focusModel.reference.id&&model.version===focusModel.reference.version
 )):undefined;
 const targetId=matches?.length===1?matches[0].connectionId:undefined;
 useEffect(()=>{
  if(focusModel&&!query.isFetching)(target.current??heading.current)?.focus();
 },[focusModel,query.isFetching,targetId]);
 async function run(work:(signal:AbortSignal)=>Promise<void>){
  if(running.current)return;
  running.current=true;setBusy(true);setFailure("");const abort=new AbortController();controller.current=abort;
  try{await work(abort.signal)}catch(error){if(active.current)setFailure(failureText(error))}finally{running.current=false;if(active.current)setBusy(false)}
 }
 function check(result:CredentialReceipt,original:CredentialPending){if(!matchesCredentialReceipt(result,original))throw new ModelRouteError("OUTCOME_UNKNOWN")}
 async function save(connection:ManagedConnection,input:CredentialInput){
  if(locked||query.isError||!query.data?.canManage)return;
  const original:CredentialPending={key:crypto.randomUUID(),userId:scope.userId,organizationId:scope.organizationId,connectionId:connection.connectionId,expectedRevision:input.expectedRevision,action:input.action};
  await run(async signal=>{
   // Persist only the strict, secret-free reference before a possible dispatch.
   // The Key stays in this one request body, outside Query/mutation persistence.
   pending.persist(original);
   try{
    const body=connection.registration?{...input,expectedMemberId:manageChannelsMemberId}:input;
    const result=await modelRouteRequest(scope,"/connections/"+encodeURIComponent(original.connectionId)+"/commands",credentialReceipt,{method:"POST",headers:{"Content-Type":"application/json","Idempotency-Key":original.key},body:JSON.stringify(body),signal});
    check(result.receipt,original);
    if(active.current){pending.clear(original);setReceipt(result.receipt);setDialog(null);await (onSaved?onSaved():query.refetch())}
   }catch(error){
    if(active.current&&error instanceof ModelRouteError&&error.status<500&&definiteCodes.has(error.code))pending.clear(original);
    throw error;
   }
  });
 }
 async function verify(){
  if(disabled||!intent||!scopeMatches)return;
  const original=intent;
  await run(async signal=>{
   const result=await modelRouteRequest(scope,"/connections/"+encodeURIComponent(original.connectionId)+"/receipts/"+encodeURIComponent(original.key),credentialReceipt,{signal});
   check(result.receipt,original);
   if(active.current){pending.clear(original);setReceipt(result.receipt);setDialog(null);await (onSaved?onSaved():query.refetch())}
  });
 }
 return <>
  {manageChannelsMemberId&&!unconfirmed&&<ManagedModelChannels scope={scope} memberId={manageChannelsMemberId} testMemberId={testMemberId}
   focusModelId={focusModel?.reference.id} blocked={locked||!!dialog} onSelect={onSelectModel} onSaved={()=>onSaved?onSaved():query.refetch()}
   onCredentials={(connection,action)=>{if(canManageKeys&&query.data?.canManage)setDialog({connection,action})}} />}
  {!unconfirmed&&canManageKeys&&<Card className={styles.panel}><h2 ref={heading} tabIndex={-1}>渠道凭据</h2><p>一份渠道 Key 可供下面列出的多个模型共用。保存时不会调用模型或扣点。</p>
   {query.isPending?<p>正在读取渠道连接…</p>:query.isError?<p role="alert">渠道连接读取失败，暂不可保存。</p>:<>
    {!query.data.canManage?<p>当前仅可查看连接状态。如需录入或更新 Key，请联系系统维护人员。</p>:null}
    {focusModel&&query.data.connections.length>0&&!targetId&&<p>暂时无法定位到单个渠道，请从以下连接中核对。</p>}
    {!query.data.connections.length?<p>{manageChannelsMemberId?"暂无由维护人员登记的渠道。":"当前企业暂无已登记渠道，请联系系统维护人员。"}</p>:query.data.connections.map(connection=>{
     const queryTarget=query.data.modelQueryTargets?.find(t=>t.connectionId===connection.connectionId);
     const canEditConnection=query.data.canManage&&!connection.readOnly;
     const showModelQuery=testMemberId&&connection.provider==="openai"&&connection.channel==="openai-direct"&&connection.models.some(m=>m.category==="text"&&m.model===openAIQueryModel);
     return <section className={styles.history} key={connection.connectionId}>
     <h3 ref={targetId===connection.connectionId?target:undefined} tabIndex={targetId===connection.connectionId?-1:undefined}>{connection.provider} · {connection.channel}</h3><p>连接：{connection.connectionId} · 地域：{connection.region}</p>
     <p>状态：{!connection.keyPresent?"未配置":connection.enabled?"已保存":"已停用"} · 版本：{connection.revision}{connection.account?" · 账号备注："+connection.account:""}</p>
     <p>共用此凭据的模型：{connection.models.map(m=>m.model+"（"+({text:"文本",multimodal:"多模态","image-generation":"生图"} as const)[m.category]+"，"+m.version+"）").join("、")||"暂无已准入模型"}</p>
     {connection.readOnly&&query.data.canManage&&<p>此渠道暂不能修改 Key，请联系系统维护人员。</p>}
     <div className={styles.actions}><Button variant="outline" disabled={locked||!canEditConnection} onClick={()=>{setFailure("");setDialog({connection,action:"set-key"})}}>{!connection.keyPresent?"录入 Key":connection.enabled?"轮换 Key":"重新录入 Key"}</Button><Button variant="outline" disabled={locked||!canEditConnection||!connection.enabled} onClick={()=>{setFailure("");setDialog({connection,action:"disable"})}}>停用</Button></div>
     {showModelQuery&&(queryTarget&&testMemberId?
      <ChannelModelQuery key={JSON.stringify([connection.revision,queryTarget.reference,connection.enabled])}
       scope={scope} connectionId={connection.connectionId} disabled={locked||!connection.enabled||!connection.keyPresent}
       input={{expectedMemberId:testMemberId,expectedRevision:connection.revision,reference:queryTarget.reference}}/>
      :<p>当前无法检查模型查询权限。</p>)}
    </section>})}
   </>}
   <Button variant="outline" disabled={disabled||busy} onClick={()=>void query.refetch()}>重新读取渠道状态</Button>
  </Card>}
  {failure?<p role="alert" className={styles.error}>{failure}</p>:null}
  {intent||pending.error||!scopeMatches?<Card className={styles.panel}><h2>保存结果待核实</h2><p>连接：{intent?.connectionId??"待核实"} · 原请求编号：{intent?.key??"本地记录读取失败"}</p><p>请用本次请求编号查询结果，确认后再录入 Key。</p><Button disabled={disabled||busy||!intent||!scopeMatches} onClick={()=>void verify()}>查询本次结果</Button></Card>:null}
  {receipt?<Card className={styles.panel} role="status"><h2>本次保存回执</h2><p>连接：{receipt.connectionId} · 本次版本：{receipt.connection.revision} · {receipt.action==="disable"?"已停用":"已保存"}</p><p>这是原请求的本地保存结果，当前状态以重新读取为准；不代表厂商账号或模型访问已验证。</p></Card>:null}
  {dialog&&!unconfirmed?<CredentialDialog key={dialog.connection.connectionId+dialog.action} connection={dialog.connection} action={dialog.action} busy={busy} locked={locked||query.isError||!query.data?.canManage} onClose={()=>setDialog(null)} onSave={input=>save(dialog.connection,input)}/>:null}
 </>;
}

function CredentialDialog({connection,action,busy,locked,onClose,onSave}:{connection:ManagedConnection;action:"set-key"|"disable";busy:boolean;locked:boolean;onClose:()=>void;onSave:(input:CredentialInput)=>Promise<void>}){
 const keyInput=useRef<HTMLInputElement>(null),accountInput=useRef<HTMLInputElement>(null);
 const [error,setError]=useState("");
 useEffect(()=>{const input=keyInput.current;return()=>{if(input)input.value=""}},[]);
 const title=action==="disable"?"停用渠道凭据":connection.keyPresent?"轮换渠道 Key":"录入渠道 Key";
 return <AccountDialog title={title} onClose={onClose}><form className={styles.form} autoComplete="off" onSubmit={event=>{
  event.preventDefault();if(locked)return;
  const value=action==="disable"?{action,expectedRevision:connection.revision}:{action,expectedRevision:connection.revision,apiKey:keyInput.current?.value??"",...(connection.revision==="0"?{account:accountInput.current?.value??""}:{})};
  const parsed=credentialInput.safeParse(value);if(!parsed.success){setError("请检查账号备注和 Key，Key 不可为空或超过 4096 字节。");return}
  setError("");if(keyInput.current)keyInput.current.value="";
  void onSave(parsed.data);
 }}>
  <p>渠道：{connection.registration?.name??connection.connectionId}。此操作影响该渠道下全部模型。</p>
  {action==="set-key"?<>{connection.revision==="0"?<label className={styles.field}>账号备注<input ref={accountInput} maxLength={128} required autoComplete="off" disabled={locked}/></label>:<p>账号备注：{connection.account}</p>}<label className={styles.field}>新的渠道 Key<input ref={keyInput} type="password" required maxLength={4096} autoComplete="new-password" spellCheck={false} disabled={locked}/></label><p>Key 只输入，不回显、不导出。保存不会自动选模型、启用 AI、测试连接或扣点。</p><p>轮换只更新本系统保存的 Key，不会替你撤销厂商旧 Key；已通过最后发送门禁的原一次在途调用可能完成。</p></>:<p>停用会阻止后续准入；原记录、在途调用、待确认结果和费用预留均保留，不删除或自动重发。</p>}
  {error?<p role="alert" className={styles.error}>{error}</p>:null}
  <div className={styles.actions}><Button type="button" variant="outline" onClick={onClose}>{busy?"关闭":"取消"}</Button><Button type="submit" disabled={locked}>{busy?"正在保存…":action==="disable"?"确认停用":"确认保存 Key"}</Button></div>
 </form></AccountDialog>;
}
