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
    { title: "硕米Chat", subtitle: "统一的AI业务入口", description: "管理真实会话、历史和收藏。模型、计价和预算配置就绪后，才可发送AI业务需求。", audience: "适用：选品分析、市场调研、店铺诊断和内容生成。", example: "分析Amazon美国站宠物市场，推荐5个适合现有供应链的商品。", href: "/workbench/ai/chat/new", available: context.aiWorkbenchAvailable, tone: "teal" },
    { title: "任务中心", subtitle: "复杂业务的执行管理", description: "追踪业务执行进度，在需要人工确认的节点查看并处理当前已开放任务。", audience: "适用：批量优化、竞品分析和资料处理。", example: "将商品优化拆成关键词分析、内容生成和人工确认。", href: taskHref, available: tasks, tone: "blue" },
    { title: "项目中心", subtitle: "长期业务的协同空间", description: "围绕店铺、品牌或创业计划长期保存资料、知识、任务和成果，让AI持续理解业务背景。", audience: "适用：店铺增长、品牌孵化和新市场拓展。", example: "建立Amazon美国站增长项目，持续跟踪选品、运营和利润目标。", href: "/workbench/ai/projects", available: context.projectCenterAvailable, tone: "purple" },
    { title: "知识库", subtitle: "业务资料的查阅与积累", description: "集中查阅官方知识和自己的资料，按需查找业务说明与参考内容。", audience: "适用：查找平台规则、运营说明和商品资料。", example: "打开我的知识库，查找已保存的商品资料。", href: "/workbench/ai/knowledge", available: props.knowledgeAvailable, tone: "teal" },
    { title: "技能库", subtitle: "可重复使用的工作方法", description: "查看官方技能，复用商品标题、商品图片的处理方法，也可管理个人和团队技能。", audience: "适用：标题整理、本地化和商品图片处理。", example: "选择标题本地化技能，进入已有商品的处理流程。", href: "/workbench/agents/skills/official", available: true, tone: "blue" },
    { title: "我的报告", subtitle: "已保存成果的查阅与复用", description: "查看当前已开放来源的历史报告，收藏重要结果并下载。", audience: "适用：保存结果和回顾原始资料。", example: "打开已保存的商品资料，查看保存时的内容。", href: "/workbench/ai/reports", available: props.reportCenterAvailable, tone: "pink" },
  ];
  return <section className={styles.page} aria-labelledby="ai-workbench-home">
    <p className={styles.eyebrow}>AI工作台</p>
    <header className={styles.header}><h1 id="ai-workbench-home">AI工作台</h1><p>在这里发起对话、跟进任务、管理项目，查阅知识、复用技能并保存工作成果。</p></header>
    <Card className={styles.flow}>提出需求 → 自动执行 → 长期协同 → 沉淀报告<span>各模块可独立使用，执行能力以当前配置与准入状态为准。</span></Card>
    <div className={styles.cards}>{entries.map(entry => {
      const content = <><div className={styles.accent} /><h2>{entry.title}</h2><p className={styles.subtitle}>{entry.subtitle}</p><p className={styles.description}>{entry.description}</p><p className={styles.audience}>{entry.audience}</p><div className={styles.example}><p>使用示例</p><span>{entry.example}</span></div><span className={styles.action}>{entry.available ? `进入${entry.title} →` : "暂未启用"}</span></>;
      return entry.available ? <Link key={entry.href} className={styles.card} data-tone={entry.tone} href={entry.href} prefetch={false}>{content}</Link> : <Card key={entry.href} className={styles.card} data-tone={entry.tone} data-state="unavailable">{content}</Card>;
    })}</div>
  </section>;
}
