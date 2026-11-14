import Link from "next/link";
import { useId } from "react";
import { Card } from "@/components/ui/card";
import styles from "./agent-market.module.css";

export function ResearchEntries({enabled, permissions}:{enabled:boolean; permissions:readonly string[]}) {
  const prefix=useId();
  const projects=permissions.includes("workbench.project.read");
  const evidence=permissions.includes("workbench.collection.read");
  const entries=[
    {label:"市场调研",href:"/workbench/data/research",description:"整理调研项目和商品资料，比较重点商品。",allowed:projects,reason:"需要项目查看权限"},
    {label:"智能选品",href:"/workbench/supply/selection",description:"比较重点商品，记录评估与选品决定。",allowed:projects && evidence,reason:projects?"需要商品资料查看权限":"需要项目查看权限"},
  ];
  return <>{entries.map((entry,index)=><Card key={entry.href} role="article" aria-labelledby={`${prefix}-${index}`} className={styles.card}>
      <div className={styles.copy}><h3 id={`${prefix}-${index}`}>{entry.label}</h3><p className={styles.description}>{entry.description}</p></div>
      <div className={styles.actions}><span className={styles.activation}>业务流程</span>{enabled && entry.allowed ? <Link className={styles.action} href={entry.href}>{entry.label}</Link> : <span className={styles.activation}>{enabled?entry.reason:"暂未开放"}</span>}</div>
    </Card>)}</>;
}
