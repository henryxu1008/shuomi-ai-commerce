"use client";

import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import type { ConfigurationScope } from "@/lib/api/agent-configuration";
import { catalogPageSchema, type CatalogEntry } from "@/lib/contracts/agent-configuration";
import { ConsoleState } from "../console/console-page";
import { AgentMarket } from "./agent-market";
import { AgentCapabilityDialog } from "./agent-capability-dialog";
import { configurationError, useConfigurationRead } from "./agent-configuration-hooks";
import { groupActivation, groupAgentEntries } from "./agent-responsibilities";
import sharedStyles from "./agents.module.css";
import styles from "./agent-mine.module.css";

const configurationHref = (entry: CatalogEntry) => `/workbench/agents/mine/${entry.agent.agentId}`;

function usageHref(entry: CatalogEntry) {
  switch (entry.agent.agentId) {
    case "product.title.agent": return "/workbench/supply/acquisition?agent_kind=title";
    case "product.content.agent": return "/workbench/supply/acquisition?agent_kind=content";
    case "pod.design.agent":
    case "product.selection.agent":
    case "shop.operations-diagnosis.agent": return configurationHref(entry);
    default: return "/workbench/supply/acquisition";
  }
}

function readiness(entries: readonly CatalogEntry[]) {
  const active = entries.filter(entry => entry.agent.activation === "ENABLED");
  const usable = active.filter(entry => entry.canUse);
  if (usable.length) return usable.length < active.length ? "部分操作可用，其余操作请查看能力配置。" : "可进入使用，按对应业务流程确认本次操作。";
  const required = entries.flatMap(entry => entry.capabilities.filter(cap => cap.support === "REQUIRED"));
  if (required.some(cap => cap.readiness === "REQUIRES_AUTHORIZATION")) return "需要授权";
  if (required.some(cap => cap.readiness === "NEEDS_CONFIGURATION")) return "待配置 · 请在能力配置中完善所需配置。";
  if (entries.every(entry => entry.agent.activation === "DISABLED")) return "已停用，可在能力配置中重新启用。";
  if (!active.length) return "尚未启用，可在能力配置中确认并启用。";
  return "当前不能执行，请在能力配置中查看原因。";
}

export function AgentMine({ entries, scope, pageScoped, pagination }: {
  entries: readonly CatalogEntry[];
  scope: ConfigurationScope;
  pageScoped: boolean;
  pagination?: ReactNode;
}) {
  const [filter, setFilter] = useState("ALL");
  const [picker, setPicker] = useState(false);
  const [capability, setCapability] = useState<CatalogEntry["agent"]["agentId"] | null>(null);
  const capabilityTrigger = useRef<HTMLButtonElement | null>(null);
  useEffect(()=>{
    if (!capability && capabilityTrigger.current?.isConnected) {
      capabilityTrigger.current.focus();
      capabilityTrigger.current = null;
    }
  },[capability]);
  const { permissions } = useWorkbenchContext();
  const groups = groupAgentEntries(entries);
  const enabled = groups.filter(group => groupActivation(group).active).length;
  const disabled = groups.filter(group => groupActivation(group).disabled).length;
  const visible = groups.filter(group => filter === "ALL" || (filter === "ENABLED" ? groupActivation(group).active : groupActivation(group).disabled));
  const canConfigure = permissions.includes("workbench.agent.configure");

  return <section className={styles.mine} aria-label="企业智能体管理">
    <div className={styles.toolbar}>
      <p className={styles.count}>{pageScoped ? "当前页" : "当前企业"}已启用 <strong>{enabled}</strong></p>
      <div className={styles.filters} role="group" aria-label="企业智能体状态">
        {[["ALL","全部",groups.length],["ENABLED","已启用",enabled],["DISABLED","已停用",disabled]].map(([value,label,count]) => <Button key={value} variant="outline" size="sm" className={styles.filter} aria-pressed={filter===value} onClick={()=>setFilter(String(value))}>{label} {count}</Button>)}
      </div>
      <Button className={styles.enable} size="sm" variant="outline" disabled={!canConfigure} title={canConfigure ? undefined : "需要当前企业智能体配置权限"} onClick={()=>setPicker(true)}>启用智能体</Button>
    </div>
    <div className={styles.layout}>
      <div className={styles.cards}>
        {visible.map(group => {
          const activation = groupActivation(group);
          const configuration = group.entries.find(entry=>entry.agent.activation==="ENABLED") ?? group.entries[0];
          const usable = group.entries.find(entry=>entry.agent.activation==="ENABLED" && entry.canUse);
          const inspect = group.entries.find(entry=>entry.capabilities.some(cap=>["product.selection.inspect","shop.operations-diagnosis.inspect"].includes(cap.id) && cap.readiness==="AVAILABLE"));
          const titleId = `mine-${group.id}`;
          return <Card key={group.id} role="article" aria-labelledby={titleId} className={styles.card}>
            <div className={styles.copy}>
              <h2 id={titleId}>{group.name}</h2>
              <p className={styles.description}>{group.description}</p>
              <p className={styles.summary}>{group.summary}</p>
            </div>
            <p className={styles.readiness}>{readiness(group.entries)}</p>
            <div className={styles.footer}>
              <span className={styles.status} data-active={activation.active}>{activation.label}</span>
              <div className={styles.actions}>
                <Button variant="outline" size="sm" className={styles.configuration} aria-haspopup="dialog" onClick={event=>{capabilityTrigger.current=event.currentTarget;setCapability(configuration.agent.agentId);}}>能力配置</Button>
                {usable ? <Button asChild size="sm" className={styles.use}><Link href={usageHref(usable)} prefetch={false}>进入使用</Link></Button> : inspect ? <Button asChild size="sm" className={styles.use}><Link href={configurationHref(inspect)} prefetch={false}>{inspect.agent.agentId==="product.selection.agent"?"查看商品事实 →":"查看经营事实 →"}</Link></Button> : <Button size="sm" className={styles.use} disabled>暂不可用</Button>}
              </div>
            </div>
          </Card>;
        })}
        {!visible.length ? <ConsoleState kind="empty" title={groups.length ? "当前企业尚无符合条件的智能体" : "当前企业尚未启用智能体"}><p>{groups.length ? "切换状态筛选查看其他智能体。" : "点击上方启用智能体，选择当前企业需要的能力。"}</p></ConsoleState> : null}
        {pagination}
      </div>
      <aside aria-label="使用流程">
        <Card className={styles.guide}>
          <h2>使用流程</h2>
          <ol>
            <li><strong>选择智能体</strong><p>在当前企业启用智能体，确认所需能力和配置。</p></li>
            <li><strong>准备业务资料</strong><p>按智能体职责选择已保存商品、店铺或设计主题。</p></li>
            <li><strong>进入使用</strong><p>核对本次权限与参数，按对应业务流程继续。</p></li>
          </ol>
          <p className={styles.note}>启用和配置不会创建执行任务，实际使用仍需确认权限与预算。</p>
        </Card>
      </aside>
    </div>
    {picker && <ActivationPicker scope={scope} onClose={()=>setPicker(false)}/>}
    {capability && <AgentCapabilityDialog key={capability} scope={scope} agentId={capability} onClose={()=>setCapability(null)}/>}
  </section>;
}

function ActivationPicker({scope,onClose}:{scope:ConfigurationScope;onClose:()=>void}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const id = useId();
  const [cursor,setCursor] = useState("");
  const [nonce,setNonce] = useState(0);
  const result = useConfigurationRead(scope,`market?pageSize=100${cursor ? `&cursor=${cursor}` : ""}`,catalogPageSchema,nonce);
  useEffect(()=>{
    const element=dialog.current, previous=document.activeElement;
    element?.showModal();
    return ()=>{element?.close();if(previous instanceof HTMLElement && previous.isConnected)previous.focus();};
  },[]);
  const entries = result.data?.items.filter(entry=>entry.agent.activation!=="ENABLED") ?? [];
  return <dialog ref={dialog} className={`${sharedStyles.dialog} ${styles.picker}`} aria-labelledby={`${id}-title`} onCancel={event=>{event.preventDefault();onClose();}}>
    <div className={styles.pickerHeader}><h2 id={`${id}-title`}>启用当前企业智能体</h2><Button variant="outline" size="sm" onClick={onClose}>关闭</Button></div>
    <p>选择智能体后进入本企业的配置页，确认后启用。已启用的操作继续保留原配置。</p>
    {result.error ? <ConsoleState kind="error" title="可启用智能体暂时不可用"><p>{configurationError(result.error)}</p><Button onClick={()=>setNonce(value=>value+1)}>重新读取目录</Button></ConsoleState> : !result.data ? <ConsoleState kind="loading" title="正在读取可启用智能体"/> : entries.length ? <AgentMarket entries={entries}/> : <ConsoleState kind="empty" title={cursor||result.data.nextCursor ? "本页没有待启用的智能体" : "当前没有待启用的智能体"}/>}
    {cursor || result.data?.nextCursor ? <nav aria-label="可启用智能体分页" className={styles.pickerPages}>{cursor ? <Button variant="outline" onClick={()=>setCursor("")}>返回目录首页</Button> : null}{result.data?.nextCursor ? <Button variant="outline" onClick={()=>setCursor(result.data!.nextCursor)}>下一页目录</Button> : null}</nav> : null}
  </dialog>;
}
