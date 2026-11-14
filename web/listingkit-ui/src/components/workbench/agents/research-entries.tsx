import Link from "next/link";
import { Card } from "@/components/ui/card";
import styles from "./agent-market.module.css";

export function ResearchEntries({enabled, permissions}:{enabled:boolean; permissions:readonly string[]}) {
  const projects=permissions.includes("workbench.project.read");
  const evidence=permissions.includes("workbench.collection.read");
  const entries=[
    {label:"市场调研",href:"/workbench/data/research",description:"整理调研项目和商品资料，比较重点商品。",allowed:projects,reason:"需要项目查看权限"},
    {label:"智能选品",href:"/workbench/supply/selection",description:"比较重点商品，记录评估与选品决定。",allowed:projects && evidence,reason:projects?"需要商品资料查看权限":"需要项目查看权限"},
  ];
  return <section className={styles.market} aria-labelledby="research-entries-title">
    <div className={styles.intro}><h2 id="research-entries-title">调研与选品</h2><p>按需进入已有业务流程；智能体可在需要时单独配置。</p></div>
    <div className={styles.grid}>{entries.map(entry=><Card key={entry.href} className={styles.card}>
      <div className={styles.copy}><h3>{entry.label}</h3><p className={styles.description}>{entry.description}</p></div>
      <div className={styles.actions}>{enabled && entry.allowed ? <Link className={styles.action} href={entry.href}>{entry.label}</Link> : <span className={styles.activation}>{enabled?entry.reason:"暂未开放"}</span>}</div>
    </Card>)}</div>
  </section>;
}
