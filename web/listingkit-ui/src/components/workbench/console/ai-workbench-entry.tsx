"use client";

import Link from "next/link";
import { Card } from "@/components/ui/card";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {type AIWorkbenchEntryCapabilities} from "@/lib/workbench/ai-workbench-entry";
import styles from "./ai-workbench-home.module.css";
import {ConsoleState} from "./console-page";
import {canDiscoverNativeRuns} from "../ai-workbench/native-run-records";

export function AIWorkbenchEntry(props: Omit<AIWorkbenchEntryCapabilities, "aiWorkbenchAvailable" | "projectCenterAvailable">) {
  const context = useWorkbenchContext();
  if (context.isLoading || context.isSwitching || context.selectionRequired || !context.effectiveOrganization || context.error || context.blockingError) {
    return <ConsoleState kind="loading" title="正在读取工作台..." />;
  }
  const tasks = context.aiWorkbenchAvailable || props.productReviewAvailable || props.sheinRecordsAvailable || props.nativeRunsAvailable && canDiscoverNativeRuns(context.permissions);
  const taskHref = "/workbench/ai/tasks";
  const entries = [
    { title: "硕米Chat", subtitle: "统一的AI业务入口", description: "管理真实会话、历史和收藏。模型、计价和预算配置就绪后，才可发送AI业务需求。", audience: "适用：选品分析、市场调研、店铺诊断和内容生成。", example: "分析Amazon美国站宠物市场，推荐5个适合现有供应链的商品。", href: "/workbench/ai/chat", available: context.aiWorkbenchAvailable, tone: "teal" },
    { title: "任务中心", subtitle: "复杂业务的执行管理", description: "追踪业务执行进度，在需要人工确认的节点查看并处理当前已开放任务。", audience: "适用：批量优化、竞品分析和资料处理。", example: "将商品优化拆成关键词分析、内容生成和人工确认。", href: taskHref, available: tasks, tone: "blue" },
    { title: "项目中心", subtitle: "长期业务的协同空间", description: "围绕店铺、品牌或创业计划长期保存资料、知识、任务和成果，让AI持续理解业务背景。", audience: "适用：店铺增长、品牌孵化和新市场拓展。", example: "建立Amazon美国站增长项目，持续跟踪选品、运营和利润目标。", href: "/workbench/ai/projects", available: context.projectCenterAvailable, tone: "purple" },
    { title: "我的报告", subtitle: "AI成果的沉淀与复用", description: "集中保存市场分析、商品报告、店铺诊断和运营方案，并支持查阅、收藏与复用。", audience: "适用：成果归档、经营复盘和团队共享。", example: "打开已保存的宠物用品市场报告，复盘机会商品。", href: "/workbench/ai/reports", available: props.reportCenterAvailable, tone: "pink" },
  ];
  return <section className={styles.page} aria-labelledby="ai-workbench-home">
    <p className={styles.eyebrow}>AI工作台</p>
    <header className={styles.header}><h1 id="ai-workbench-home">AI工作台</h1><p>从提出业务需求，到持续执行、长期管理和成果沉淀，四个核心模块共同组成完整的AI业务工作流。</p></header>
    <Card className={styles.flow}>提出需求 → 自动执行 → 长期协同 → 沉淀报告<span>各模块可独立使用，执行能力以当前配置与准入状态为准。</span></Card>
    <div className={styles.cards}>{entries.map(entry => {
      const content = <><div className={styles.accent} /><h2>{entry.title}</h2><p className={styles.subtitle}>{entry.subtitle}</p><p className={styles.description}>{entry.description}</p><p className={styles.audience}>{entry.audience}</p><div className={styles.example}><p>使用示例</p><span>{entry.example}</span></div><span className={styles.action}>{entry.available ? `进入${entry.title} →` : "暂未启用"}</span></>;
      return entry.available ? <Link key={entry.href} className={styles.card} data-tone={entry.tone} href={entry.href} prefetch={false}>{content}</Link> : <Card key={entry.href} className={styles.card} data-tone={entry.tone} data-state="unavailable">{content}</Card>;
    })}</div>
  </section>;
}
