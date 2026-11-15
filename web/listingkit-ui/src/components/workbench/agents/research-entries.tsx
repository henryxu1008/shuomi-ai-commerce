import Link from "next/link";
import { useId } from "react";
import { Card } from "@/components/ui/card";
import type { CatalogEntry } from "@/lib/contracts/agent-configuration";
import { groupActivation } from "./agent-responsibilities";
import styles from "./agent-market.module.css";

export function ResearchEntries({enabled, permissions, entries=[]}:{enabled:boolean; permissions:readonly string[]; entries?:readonly CatalogEntry[]}) {
  const prefix=useId();
  const projects=permissions.includes("workbench.project.read");
  const evidence=permissions.includes("workbench.collection.read");
  const entriesForDisplay=[
    {label:"市场调研",agentId:"product.research.agent",href:"/workbench/data/research",description:"分析研究项目中已选的商品资料，查看有依据的建议。",allowed:projects,reason:"需要项目查看权限"},
    {label:"智能选品",agentId:"product.selection.agent",href:"/workbench/supply/selection",description:"比较已保存的商品，查看差异和缺少的信息。",allowed:projects && evidence,reason:projects?"需要商品资料查看权限":"需要项目查看权限"},
  ];
  return <>{entriesForDisplay.map((entry,index)=>{
    const agent=entries.find(item=>item.agent.agentId===entry.agentId);
    const status=agent ? groupActivation({agentIds:[agent.agent.agentId],entries:[agent]}).label : "状态待核实";
    return <Card key={entry.href} role="article" aria-labelledby={`${prefix}-${index}`} className={styles.card}>
      <div className={styles.copy}><h3 id={`${prefix}-${index}`}>{entry.label}智能体</h3><p className={styles.description}>{entry.description}</p></div>
      <div className={styles.actions}><span className={styles.activation} data-enabled={agent?.agent.activation==="ENABLED"}>{status}</span>{agent ? <Link className={styles.configLink} href={`/workbench/agents/mine/${agent.agent.agentId}`}>{agent.canConfigure?"智能体配置":"查看智能体"}</Link> : null}{enabled && entry.allowed ? <Link className={styles.action} href={entry.href}>{entry.label}</Link> : <span className={styles.activation}>{enabled?entry.reason:"暂未开放"}</span>}</div>
    </Card>;
  })}</>;
}
