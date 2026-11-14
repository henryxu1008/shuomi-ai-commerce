"use client";

import { useId, useState, type ReactNode } from "react";
import Image from "next/image";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import type { CatalogEntry } from "@/lib/contracts/agent-configuration";
import { ConsoleState } from "../console/console-page";
import styles from "./agent-market.module.css";

import { agentCategories as categories, agentGroups as groups, groupActivation, type AgentCategory as Category } from "./agent-responsibilities";
import { SelectionOperationLinks } from "./selection-operation-links";
import { ResearchEntries } from "./research-entries";

export function AgentMarket({ entries, research, catalogNotice }: { entries: readonly CatalogEntry[]; research?:{enabled:boolean;permissions:readonly string[]}; catalogNotice?:ReactNode }) {
  const [category, setCategory] = useState<Category>("全部");
  const prefix = useId();
  const cards = groups.flatMap(group => {
    if (research && group.id === "product-selection") return [];
    const operations = group.agentIds.flatMap(agentId => entries.filter(entry => entry.agent.agentId === agentId));
    return operations.length && (category === "全部" || category === group.category) ? [{ ...group, operations }] : [];
  });
  const showResearch=!!research && (category==="全部" || category==="选品");

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
      {catalogNotice}
      {cards.length || showResearch ? <div className={styles.grid}>
        {showResearch && research ? <ResearchEntries enabled={research.enabled} permissions={research.permissions} entries={entries} /> : null}
        {cards.map(card => {
          const enabled = card.operations.filter(entry => entry.agent.activation === "ENABLED").length;
          const status = groupActivation({agentIds:card.agentIds,entries:card.operations}).label;
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
            </div>
            <div className={styles.actions}>
              <span className={styles.activation} data-enabled={enabled > 0}>{status}</span>
              {card.id === "product-selection" ? <SelectionOperationLinks className={styles.operationLinks} configuration={false} agentIds={card.operations.map(entry=>entry.agent.agentId)}/> : <Link href={href} prefetch={false} className={styles.action}>{action}</Link>}
            </div>
          </Card>;
        })}
      </div> : !catalogNotice ? <ConsoleState kind="empty" title={category === "全部" ? "当前没有可展示的智能体" : "该分类暂无智能体"}>
        <p>请选择其他分类查看当前已提供的智能体。</p>
      </ConsoleState> : null}
      <p className={styles.note}>在我的智能体中启用、配置并查看当前可用状态。</p>
    </section>
  );
}
