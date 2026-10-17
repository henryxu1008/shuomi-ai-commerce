"use client";

import { useId, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import type { CatalogEntry } from "@/lib/contracts/agent-configuration";
import { ConsoleState } from "../console/console-page";
import styles from "./agent-market.module.css";

const categories = ["全部", "商品", "选品", "运营", "广告", "客服", "数据"] as const;
type Category = (typeof categories)[number];
type AgentId = CatalogEntry["agent"]["agentId"];
const groups: readonly {
  id: string;
  agentIds: readonly AgentId[];
  category: Category;
  name?: string;
  description?: string;
  summary: string;
}[] = [
  {
    id: "product-content", agentIds: ["product.content.agent", "product.title.agent"], category: "商品",
    name: "商品内容与本地化", description: "完善商品标题、描述、卖点与关键词，按目标语言处理内容。",
    summary: "标题优化 · 内容完善与本地化",
  },
  { id: "listing-materials", agentIds: ["listing.materials.agent"], category: "商品", summary: "平台上架资料 · 可检查草稿" },
  { id: "product-image", agentIds: ["product.image.agent"], category: "商品", summary: "主图与详情图 · 人工选择批准" },
  { id: "pod-design", agentIds: ["pod.design.agent"], category: "商品", summary: "原创图案 · 保存后布局与效果图" },
  { id: "product-selection", agentIds: ["product.selection.agent"], category: "选品", summary: "商品事实对比 · 有引用建议" },
  { id: "shop-diagnosis", agentIds: ["shop.operations-diagnosis.agent"], category: "运营", summary: "经营事实诊断 · 有引用建议" },
];

export function AgentMarket({ entries }: { entries: readonly CatalogEntry[] }) {
  const [category, setCategory] = useState<Category>("全部");
  const prefix = useId();
  const cards = groups.flatMap(group => {
    const operations = group.agentIds.flatMap(agentId => entries.filter(entry => entry.agent.agentId === agentId));
    return operations.length && (category === "全部" || category === group.category) ? [{ ...group, operations }] : [];
  });

  return (
    <section className={styles.market} aria-label="智能体列表">
      <div className={styles.intro}><h2>智能体列表</h2><p>按业务分类查找适合的智能体。</p></div>
      <div className={styles.categories} role="group" aria-label="智能体业务分类">
        {categories.map(value => (
          <Button key={value} variant="outline" size="sm" aria-pressed={category === value}
            onClick={() => setCategory(value)} className={styles.category}>
            {category === value ? <Image src="/sumi/agent-market-selected.svg" width={6} height={6} alt="" aria-hidden="true" /> : null}
            {value}
          </Button>
        ))}
      </div>
      {cards.length ? <div className={styles.grid}>
        {cards.map(card => {
          const enabled = card.operations.filter(entry => entry.agent.activation === "ENABLED").length;
          const status = enabled === card.agentIds.length ? "已启用" : enabled ? "部分已启用"
            : card.operations.every(entry => entry.agent.activation === "DISABLED") ? "已停用" : "尚未启用";
          const target = card.operations.find(entry => entry.canConfigure) ?? card.operations[0];
          const href = enabled ? "/workbench/agents/mine" : "/workbench/agents/mine/" + target.agent.agentId;
          const action = enabled ? "前往我的智能体" : target.canConfigure ? "前往启用" : "查看智能体";
          const title = card.name ?? card.operations[0].name;
          const description = card.description ?? card.operations[0].description;
          const heading = prefix + "-" + card.id;
          return <Card key={card.id} role="article" aria-labelledby={heading} className={styles.card}>
            <div className={styles.copy}>
              <h3 id={heading}>{title}</h3>
              <p className={styles.description} title={description}>{description}</p>
              <p className={styles.summary}>{card.summary}</p>
            </div>
            <div className={styles.actions}>
              <span className={styles.activation} data-enabled={enabled > 0}>{status}</span>
              <Link href={href} prefetch={false} className={styles.action}>{action}</Link>
            </div>
          </Card>;
        })}
      </div> : <ConsoleState kind="empty" title={category === "全部" ? "当前没有可展示的智能体" : "该分类暂无智能体"}>
        <p>请选择其他分类查看当前已提供的智能体。</p>
      </ConsoleState>}
      <p className={styles.note}>在我的智能体中启用、配置并查看当前可用状态。</p>
    </section>
  );
}
