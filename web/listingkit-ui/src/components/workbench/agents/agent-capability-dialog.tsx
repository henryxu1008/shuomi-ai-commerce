"use client";

import Link from "next/link";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import type { ConfigurationScope } from "@/lib/api/agent-configuration";
import { catalogEntrySchema, type CatalogEntry } from "@/lib/contracts/agent-configuration";
import { ResourceDialog } from "../resources/resource-dialog";
import { AgentCapabilityPanel } from "./agent-capability-panel";
import { configurationError, useConfigurationRead } from "./agent-configuration-hooks";
import styles from "./agent-capabilities.module.css";

export function AgentCapabilityDialog({ scope, agentId, onClose }: {
  scope: ConfigurationScope;
  agentId: CatalogEntry["agent"]["agentId"];
  onClose: () => void;
}) {
  const [nonce, setNonce] = useState(0);
  const read = useConfigurationRead(scope, agentId, catalogEntrySchema, nonce);
  const entry = read.data?.agent.agentId === agentId ? read.data : undefined;
  const failed = !!read.error || (!!read.data && !entry);
  return <ResourceDialog title="智能体能力配置" onClose={onClose} className={styles.dialog}>
    <p className={styles.intro}>确认当前企业可用能力；已有设置在原配置页管理。</p>
    {entry ? <>
      <p className={styles.identity}>{entry.name} <span>{entry.definitionVersion}</span></p>
      {["product.title.agent", "product.content.agent"].includes(agentId) ? <nav aria-label="商品内容操作配置" className={styles.operationLinks}>
        <Link href="/workbench/agents/mine/product.title.agent" prefetch={false}>标题优化配置与历史</Link>
        <Link href="/workbench/agents/mine/product.content.agent" prefetch={false}>内容本地化配置与历史</Link>
      </nav> : null}
      <AgentCapabilityPanel entry={entry}/>
    </> : failed ? <div role="alert" className={styles.error}>
      <p>{read.error ? configurationError(read.error) : "能力信息与当前智能体不一致，请重新读取。"}</p>
      <Button variant="outline" size="sm" onClick={() => setNonce(value => value + 1)}>重新读取能力</Button>
    </div> : <p role="status" className={styles.loading}>正在读取当前能力…</p>}
    <div className={styles.footer}>
      {entry ? <Button variant="ghost" size="sm" className={styles.refresh} onClick={() => setNonce(value => value + 1)}>刷新能力状态</Button> : null}
      <Button variant="outline" aria-label="关闭能力配置" onClick={onClose}>关闭</Button>
      {entry ? <Button asChild className={styles.primary}>
        <Link href={`/workbench/agents/mine/${agentId}`} prefetch={false}>{entry.canConfigure ? "管理权限与模板" : "查看权限与模板"}</Link>
      </Button> : null}
    </div>
  </ResourceDialog>;
}
