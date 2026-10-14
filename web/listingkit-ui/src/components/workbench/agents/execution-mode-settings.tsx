"use client";
import { useEffect, useRef, useState } from "react";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { configurationRequest, ConfigurationError, type ConfigurationScope } from "@/lib/api/agent-configuration";
import { executionModeInputSchema, executionModeViewSchema, type ExecutionMode, type ModeSubject } from "@/lib/contracts/execution-mode";
import { knowledgeId } from "@/lib/api/knowledge";
import { useResourcePending } from "../resources/resource-pending";
import { ResourceDialog } from "../resources/resource-dialog";
import { configurationError, useConfigurationRead } from "./agent-configuration-hooks";
import styles from "./execution-mode-settings.module.css";

export const executionModeLabels: Record<ExecutionMode,string> = {default:"默认权限",auto_approve:"自动审批",read_only:"只读",full_access:"完全访问"};
export const executionModeAdoptionText:Record<ExecutionMode,string>={default:"生成建议，采用仍需人工确认。",auto_approve:"通过程序校验的本地标题自动采用；需人工判断的内容保留审核。",read_only:"保存分析与建议，不采用到商品或草稿。",full_access:"已授权的本地内容可按策略采用，未经逐项人工审阅。"};
const descriptions: Record<ExecutionMode,string> = {
 default:"生成建议，本地采用仍由本人确认。",
 auto_approve:"支持的本地动作通过程序校验后采用；图片保真仍需人工确认。",
 read_only:"读取、分析并保存建议；不能修改商品、草稿或生成图片资产。",
 full_access:"已授权企业业务工具可自动执行和采用，本地内容不逐项人工审阅。",
};
const intentSchema = z.strictObject({key:knowledgeId,payload:executionModeInputSchema});
type Intent = z.infer<typeof intentSchema>;

export function ExecutionModeSettings({scope,subject,onSaved}:{scope:ConfigurationScope;subject:ModeSubject;onSaved?:()=>void}) {
 return <BoundExecutionModeSettings key={JSON.stringify([scope.userId,scope.organizationId,subject.kind,subject.id])} scope={scope} subject={subject} onSaved={onSaved}/>;
}

function BoundExecutionModeSettings({scope,subject,onSaved}:{scope:ConfigurationScope;subject:ModeSubject;onSaved?:()=>void}) {
 const path=`execution-modes/${subject.kind}/${subject.id}`;
 const pending=useResourcePending({expectedUserId:scope.userId,expectedOrganizationId:scope.organizationId},["agent-execution-mode",subject.kind,subject.id],intentSchema,{storage:"local",maxLength:2048});
 const context=useWorkbenchContext();
 const [nonce,setNonce]=useState(0),[open,setOpen]=useState(false),[selected,setSelected]=useState<ExecutionMode>("default"),[busy,setBusy]=useState(false),[failure,setFailure]=useState<unknown>(),[message,setMessage]=useState("");
 const active=useRef(true),flight=useRef(false),abort=useRef<AbortController|null>(null);
 const read=useConfigurationRead(scope,path,executionModeViewSchema,nonce),view=read.data;
 const locked=busy||!!pending.command||!pending.ready||!!pending.error;
 useEffect(()=>{active.current=true;return()=>{active.current=false;abort.current?.abort()}},[]);
 const register=context.registerOrganizationSwitchGuard;
 useEffect(()=>register(()=>!open&&!flight.current&&!pending.command&&!pending.error),[register,pending.command,pending.error,open]);
 async function send(intent:Intent) {
  if(flight.current||!active.current||!pending.ready)return;
	const verifying=!!pending.command;
  flight.current=true;setBusy(true);setFailure(undefined);setMessage("");
  const controller=new AbortController();abort.current=controller;
  try {
   pending.persist(intent);
   const saved=await configurationRequest(scope,path,executionModeViewSchema,{method:"PUT",headers:{"Content-Type":"application/json","Idempotency-Key":intent.key},body:JSON.stringify(intent.payload),signal:controller.signal});
   if(saved.subject.kind!==subject.kind||saved.subject.id!==subject.id||saved.receipt?.requestKey!==intent.key||saved.mode!==intent.payload.mode||saved.receipt.beforeRevision!==intent.payload.expectedRevision)throw new ConfigurationError("OUTCOME_UNKNOWN");
   if(!active.current)return;
   pending.clear(intent);setOpen(false);setNonce(n=>n+1);onSaved?.();setMessage("权限模式已保存，下一次新运行将使用当前模式。已存在的运行不会升级权限。");
  } catch(error) {
   if(!active.current)return;
   setFailure(error);
   if(error instanceof ConfigurationError&&error.code!=="OUTCOME_UNKNOWN"&&error.status<500&&!verifying)pending.clear(intent);
   if(verifying)setMessage("暂时无法核实原保存结果，请保留原请求编号，恢复授权后核实。");
  } finally { flight.current=false;if(active.current)setBusy(false) }
 }
 const operationStatus=<>
  {failure?<p role="alert" className={styles.error}>{configurationError(failure)}</p>:null}{pending.error?<p role="alert" className={styles.error}>原操作记录不可用，请保留原请求编号并核实。</p>:null}
  {pending.command?<Button disabled={busy||!pending.ready} onClick={()=>void send(pending.command!)}>核实原保存操作</Button>:null}
  {message?<p role="status">{message}</p>:null}
 </>;
 return <Card className={styles.panel}>
  <div className={styles.header}><h2>执行权限</h2>{view?.canConfigure?<Button variant="outline" size="sm" disabled={locked} onClick={()=>{setSelected(view.mode);setOpen(true)}}>配置权限</Button>:null}</div>
  {view?<div className={styles.summary}><p className={styles.modeLabel}>{executionModeLabels[view.mode]} · 配置版本 {view.revision}</p><p>{descriptions[view.mode]}</p></div>:read.error?<p role="alert">{configurationError(read.error)}</p>:<p role="status">正在读取权限模式…</p>}
  <div className={styles.boundary}><strong>权限范围</strong><p>范围受当前企业、成员及资源权限限制。发布、改价、下架、删除和增加付费额度需另行授权；生成前仍须确认计划和预算。</p></div>
  {!open?operationStatus:null}
  <div className={styles.refresh}><Button variant="ghost" size="sm" disabled={busy} onClick={()=>setNonce(n=>n+1)}>刷新权限状态</Button></div>
  {open&&view?<ResourceDialog title="智能体能力配置" onClose={()=>setOpen(false)} locked={locked} className={styles.dialog}>
   <p className={styles.intro}>选择本地结果的处理方式。模式不会增加当前企业、成员或资源的访问权限。</p>
   <fieldset disabled={locked} className={styles.options}><legend>执行权限模式</legend>{(Object.keys(executionModeLabels) as ExecutionMode[]).map(mode=><label key={mode} className={styles.option} data-selected={selected===mode}><input type="radio" name={`execution-mode-${subject.id}`} value={mode} checked={selected===mode} onChange={()=>setSelected(mode)}/><span className={styles.optionContent}><span className={styles.optionHeading}><strong>{executionModeLabels[mode]}</strong>{selected===mode?<span className={styles.selected} aria-hidden="true">已选</span>:null}</span><span className={styles.description}>{descriptions[mode]}</span></span></label>)}</fieldset>
   <div className={styles.boundary}><p>保存后只影响新的执行准入；已运行操作变更策略后需要新运行。</p></div>
   {operationStatus}
   <div className={styles.actions}><Button variant="outline" disabled={locked} onClick={()=>setOpen(false)}>关闭</Button><Button disabled={locked} onClick={()=>void send({key:crypto.randomUUID(),payload:{mode:selected,expectedRevision:view.revision}})}>保存配置</Button></div>
  </ResourceDialog>:null}
 </Card>;
}
