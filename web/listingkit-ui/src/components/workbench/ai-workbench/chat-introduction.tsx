import Link from "next/link";
import type { ReactNode } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import styles from "./chat-page.module.css";

const collaboration = [
  ["明确目标", "说明要处理的商品内容、目标语言与期望结果。"],
  ["按需引用", "选择已保存商品，可补充模板与知识库。"],
  ["核对方案", "检查对象、执行权限与额度上限，再明确确认。"],
  ["继续跟进", "在任务中心查看结果，收藏会话以便回顾。"],
];

export function ChatIntroduction({ createAction, contentTasks }: { createAction: ReactNode; contentTasks: boolean }) {
  const entries = [
    { title: "新建会话", subtitle: "开始一项新需求", description: "从已保存商品出发，描述内容目标，查看并确认执行方案。", tags: ["目标描述", "方案确认"], example: contentTasks ? "将这个商品的描述和卖点整理成英文。" : "优化这个商品的标题，突出已有资料支持的卖点。", action: createAction },
    { title: "最近会话", subtitle: "继续之前的工作", description: "保留对话背景与已有方案，从上次讨论的位置继续。", tags: ["继续交流", "任务跟进"], example: "打开之前的讨论，核对方案与已确认任务。", action: <Button asChild variant="outline"><Link href="/workbench/ai/chat/recent" prefetch={false}>查看最近会话 →</Link></Button> },
    { title: "收藏会话", subtitle: "沉淀重要内容", description: "集中查找已收藏的会话，回顾值得保留的需求与结论。", tags: ["重点收藏", "随时回顾"], example: "回到已收藏的内容讨论，继续查看原有消息。", action: <Button asChild variant="outline"><Link href="/workbench/ai/chat/saved" prefetch={false}>进入收藏会话 →</Link></Button> },
  ];
  return <div className={styles.introduction}>
    <section className={styles.landingGrid} aria-label="会话入口">{entries.map(entry => <Card key={entry.title} className={styles.landingCard}>
      <div><h2>{entry.title}</h2><p className={styles.subtitle}>{entry.subtitle}</p></div>
      <p>{entry.description}</p>
      <div className={styles.entryActions}><div className={styles.tags}>{entry.tags.map(tag => <Badge key={tag} variant="neutral">{tag}</Badge>)}</div>{entry.action}</div>
      <div className={styles.example}><strong>使用示例</strong><p>{entry.example}</p></div>
    </Card>)}</section>
    <section className={styles.introSection} aria-labelledby="chat-collaboration"><h2 id="chat-collaboration">会话协作方式</h2>
      <ol className={styles.steps}>{collaboration.map(([title, description], index) => <li key={title}><Card className={styles.step}>
        <div className={styles.stepHeading}><Badge variant="neutral">{String(index + 1).padStart(2, "0")}</Badge><h3>{title}</h3></div><p>{description}</p>
      </Card></li>)}</ol>
    </section>
    <section className={styles.introSection} aria-labelledby="chat-scenarios"><h2 id="chat-scenarios">常见业务场景</h2>
      <div className={styles.scenarios}>{[
        [contentTasks ? "整理商品内容" : "优化商品标题", contentTasks ? "围绕已保存资料选择语言和待处理字段，提出内容整理目标。" : "围绕已保存商品与目标平台，提出有资料依据的标题目标。"],
        ["核对执行方案", "查看商品、模板、知识与额度限制，确认后创建业务任务。"],
        ["回顾重要讨论", "从最近或收藏会话找回原有消息，继续跟进已确认任务。"],
      ].map(([title, description]) => <Card className={styles.step} key={title}><h3>{title}</h3><p>{description}</p></Card>)}</div>
    </section>
  </div>;
}

export function NewConversationIntroduction({ createAction }: { createAction: ReactNode }) {
  return <div className={styles.newWorkspace}>
    <section className={styles.newMain} aria-labelledby="chat-new-goal">
      <header className={styles.newHeading}><h2 id="chat-new-goal">开始商品任务会话</h2><p>创建会话后，选择已有商品，描述要处理的内容。</p></header>
      <Card className={styles.createPanel}><h3>创建会话</h3><p>会话会保存到当前账号与企业。任务方案在你核对并确认后才会执行。</p><div className={styles.createAction}>{createAction}</div></Card>
      <div className={styles.newGuide}><h3>开始前准备</h3><ol><li>选好要处理的已保存商品。</li><li>说明需要整理的内容与目标。</li><li>核对方案和额度上限后再确认执行。</li></ol></div>
    </section>
  </div>;
}
